package steam

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/srnoob2570/better-rpt/internal/domain"
	"github.com/srnoob2570/better-rpt/internal/steam/vdf"
)

// Process is one live Steam-related process.
type Process struct {
	PID  int
	Name string
}

// Lifecycle owns the Steam close/wait/relaunch sequence around a localconfig
// transaction. Root is the native Steam installation, not a library.
type Lifecycle struct {
	Root            string
	LocalConfigPath string
	SteamBinary     string
	ProcRoot        string
	RegistryPath    string

	RunCommand   func(ctx context.Context, name string, args ...string) error
	StartCommand func(name string) error

	ShutdownTimeout time.Duration
	StopTimeout     time.Duration
	StableTimeout   time.Duration
	StartTimeout    time.Duration
	PollInterval    time.Duration
	StableInterval  time.Duration
}

// NewLifecycle builds a lifecycle for a detected native installation.
func NewLifecycle(root, localConfigPath string) (*Lifecycle, error) {
	if root == "" {
		return nil, fmt.Errorf("%w: empty Steam root", domain.ErrSteamNotFound)
	}
	binary, err := FindSteamBinary(root)
	if err != nil {
		return nil, err
	}
	return &Lifecycle{
		Root:            root,
		LocalConfigPath: localConfigPath,
		SteamBinary:     binary,
		ProcRoot:        "/proc",
		ShutdownTimeout: 10 * time.Second,
		StopTimeout:     30 * time.Second,
		StableTimeout:   5 * time.Second,
		StartTimeout:    60 * time.Second,
		PollInterval:    200 * time.Millisecond,
		StableInterval:  200 * time.Millisecond,
	}, nil
}

// FindSteamBinary locates Steam's launcher without starting it.
func FindSteamBinary(steamRoot string) (string, error) {
	if path, err := exec.LookPath("steam"); err == nil {
		return path, nil
	}
	for _, candidate := range []string{
		filepath.Join(steamRoot, "steam.sh"),
		"/usr/bin/steam",
		"/usr/games/steam",
	} {
		info, err := os.Stat(candidate)
		if err == nil && info.Mode().IsRegular() && info.Mode().Perm()&0o111 != 0 {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("%w: no steam launcher found", domain.ErrSteamNotFound)
}

// IsRunning reports whether Steam or steamwebhelper is alive, or whether the
// registry's SteamPID points at a live steam process. steam.pid is deliberately
// ignored because it is often stale.
func (l *Lifecycle) IsRunning() (bool, error) {
	processes, err := ScanProcesses(l.procRoot())
	if err != nil {
		return false, err
	}
	if len(processes) > 0 {
		return true, nil
	}
	pid, err := l.registryPID()
	if err != nil {
		return false, err
	}
	if pid <= 0 {
		return false, nil
	}
	name, err := processName(l.procRoot(), pid)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return isSteamProcess(name), nil
}

// ScanProcesses reads /proc/*/comm and returns steam/steamwebhelper processes.
func ScanProcesses(procRoot string) ([]Process, error) {
	entries, err := os.ReadDir(procRoot)
	if err != nil {
		return nil, fmt.Errorf("scan Steam processes: %w", err)
	}
	processes := make([]Process, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}
		name, err := processName(procRoot, pid)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			continue
		}
		if isSteamProcess(name) {
			processes = append(processes, Process{PID: pid, Name: name})
		}
	}
	sort.Slice(processes, func(i, j int) bool { return processes[i].PID < processes[j].PID })
	return processes, nil
}

func processName(procRoot string, pid int) (string, error) {
	data, err := os.ReadFile(filepath.Join(procRoot, strconv.Itoa(pid), "comm"))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func isSteamProcess(name string) bool {
	name = strings.TrimSpace(name)
	return name == "steam" || strings.HasPrefix(name, "steamwebhelper")
}

// Shutdown asks Steam to close, waits for the processes to disappear and for
// localconfig to stabilize. It never writes localconfig itself.
func (l *Lifecycle) Shutdown(ctx context.Context) error {
	running, err := l.IsRunning()
	if err != nil {
		return err
	}
	if !running {
		return nil
	}
	if l.SteamBinary == "" {
		return domain.ErrSteamNotFound
	}
	shutdownCtx, cancel := context.WithTimeout(ctx, l.shutdownTimeout())
	defer cancel()
	if err := l.run(shutdownCtx, l.SteamBinary, "-shutdown"); err != nil {
		return fmt.Errorf("steam -shutdown: %w", err)
	}
	if err := l.WaitStopped(ctx); err != nil {
		return err
	}
	return l.WaitStable(ctx)
}

// AppLaunch asks Steam to launch an AppID. The game's own lifetime is not
// waited on; this only covers the short Steam client command.
func (l *Lifecycle) AppLaunch(ctx context.Context, appID uint32) error {
	if appID == 0 {
		return fmt.Errorf("%w: empty app id", domain.ErrInvalidSession)
	}
	if l.SteamBinary == "" {
		return domain.ErrSteamNotFound
	}
	launchCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return l.run(launchCtx, l.SteamBinary, "-applaunch", strconv.FormatUint(uint64(appID), 10))
}

// WaitStopped waits until no Steam process remains.
func (l *Lifecycle) WaitStopped(ctx context.Context) error {
	deadline := time.Now().Add(l.stopTimeout())
	for {
		running, err := l.IsRunning()
		if err != nil {
			return err
		}
		if !running {
			return nil
		}
		if !time.Now().Before(deadline) {
			return fmt.Errorf("Steam did not exit within %s", l.stopTimeout())
		}
		if err := sleepContext(ctx, l.pollInterval()); err != nil {
			return err
		}
	}
}

// WaitStable waits for two consecutive identical localconfig hashes.
func (l *Lifecycle) WaitStable(ctx context.Context) error {
	if l.LocalConfigPath == "" {
		return fmt.Errorf("%w: empty localconfig path", domain.ErrNoLaunchOption)
	}
	previous, err := hashFile(l.LocalConfigPath)
	if err != nil {
		return fmt.Errorf("read localconfig: %w", err)
	}
	deadline := time.Now().Add(l.stableTimeout())
	for {
		if err := sleepContext(ctx, l.stableInterval()); err != nil {
			return err
		}
		current, err := hashFile(l.LocalConfigPath)
		if err != nil {
			return fmt.Errorf("read localconfig: %w", err)
		}
		if bytes.Equal(previous, current) {
			return nil
		}
		previous = current
		if !time.Now().Before(deadline) {
			return fmt.Errorf("localconfig did not stabilize within %s", l.stableTimeout())
		}
	}
}

// Relaunch starts Steam in its own session and waits until it is visible. The
// call is non-blocking with respect to the Steam client itself.
func (l *Lifecycle) Relaunch(ctx context.Context) error {
	running, err := l.IsRunning()
	if err != nil {
		return err
	}
	if running {
		return nil
	}
	if l.SteamBinary == "" {
		return domain.ErrSteamNotFound
	}
	if err := l.start(l.SteamBinary); err != nil {
		return fmt.Errorf("start Steam: %w", err)
	}
	return l.WaitStarted(ctx)
}

// WaitStarted waits until a Steam process is visible.
func (l *Lifecycle) WaitStarted(ctx context.Context) error {
	deadline := time.Now().Add(l.startTimeout())
	for {
		running, err := l.IsRunning()
		if err != nil {
			return err
		}
		if running {
			return nil
		}
		if !time.Now().Before(deadline) {
			return fmt.Errorf("Steam did not start within %s", l.startTimeout())
		}
		if err := sleepContext(ctx, l.pollInterval()); err != nil {
			return err
		}
	}
}

func (l *Lifecycle) registryPID() (int, error) {
	path := l.RegistryPath
	if path == "" {
		candidates := []string{filepath.Join(l.Root, "registry.vdf")}
		if home, err := os.UserHomeDir(); err == nil {
			candidates = append(candidates, filepath.Join(home, ".steam", "registry.vdf"))
		}
		for _, candidate := range candidates {
			if _, err := os.Stat(candidate); err == nil {
				path = candidate
				break
			}
		}
	}
	if path == "" {
		return 0, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return 0, nil
		}
		return 0, fmt.Errorf("read Steam registry: %w", err)
	}
	document, err := vdf.Parse(bytes.NewReader(data))
	if err != nil {
		return 0, fmt.Errorf("parse Steam registry: %w", err)
	}
	steam, ok := nestedMap(document, []string{"Registry", "HKCU", "Software", "Valve", "Steam"})
	if !ok {
		return 0, nil
	}
	value, ok := steam["SteamPID"].(string)
	if !ok {
		return 0, nil
	}
	pid, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || pid < 0 {
		return 0, nil
	}
	return pid, nil
}

func nestedMap(root map[string]interface{}, path []string) (map[string]interface{}, bool) {
	current := root
	for _, key := range path {
		raw, ok := current[key]
		if !ok {
			return nil, false
		}
		next, ok := raw.(map[string]interface{})
		if !ok {
			return nil, false
		}
		current = next
	}
	return current, true
}

func (l *Lifecycle) run(ctx context.Context, name string, args ...string) error {
	if l.RunCommand != nil {
		return l.RunCommand(ctx, name, args...)
	}
	command := exec.CommandContext(ctx, name, args...)
	command.Stdin = nil
	command.Stdout = nil
	command.Stderr = nil
	return command.Run()
}

func (l *Lifecycle) start(name string) error {
	if l.StartCommand != nil {
		return l.StartCommand(name)
	}
	command := exec.Command(name)
	command.Stdin = nil
	command.Stdout = nil
	command.Stderr = nil
	command.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := command.Start(); err != nil {
		return err
	}
	return command.Process.Release()
}

func (l *Lifecycle) procRoot() string {
	if l.ProcRoot != "" {
		return l.ProcRoot
	}
	return "/proc"
}

func (l *Lifecycle) shutdownTimeout() time.Duration {
	if l.ShutdownTimeout > 0 {
		return l.ShutdownTimeout
	}
	return 10 * time.Second
}

func (l *Lifecycle) stopTimeout() time.Duration {
	if l.StopTimeout > 0 {
		return l.StopTimeout
	}
	return 30 * time.Second
}

func (l *Lifecycle) stableTimeout() time.Duration {
	if l.StableTimeout > 0 {
		return l.StableTimeout
	}
	return 5 * time.Second
}

func (l *Lifecycle) startTimeout() time.Duration {
	if l.StartTimeout > 0 {
		return l.StartTimeout
	}
	return 60 * time.Second
}

func (l *Lifecycle) pollInterval() time.Duration {
	if l.PollInterval > 0 {
		return l.PollInterval
	}
	return 200 * time.Millisecond
}

func (l *Lifecycle) stableInterval() time.Duration {
	if l.StableInterval > 0 {
		return l.StableInterval
	}
	return 200 * time.Millisecond
}

func hashFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(data)
	return []byte(hex.EncodeToString(sum[:])), nil
}

func sleepContext(ctx context.Context, duration time.Duration) error {
	if duration <= 0 {
		duration = time.Millisecond
	}
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
