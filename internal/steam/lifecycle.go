package steam

import (
	"bytes"
	"context"
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

	"github.com/srnoob2570/better-rpt/internal/domain"
	"github.com/srnoob2570/better-rpt/internal/steam/vdf"
)

// Process is one live Steam-related process.
type Process struct {
	PID  int
	Name string
}

// Lifecycle owns Steam process discovery and the non-blocking launch requests
// used by Better RPT. It never closes or restarts Steam.
type Lifecycle struct {
	Root         string
	SteamBinary  string
	ProcRoot     string
	RegistryPath string

	StartCommand func(name string, args ...string) error
}

// NewLifecycle builds a lifecycle for a detected native installation.
func NewLifecycle(root string) (*Lifecycle, error) {
	if root == "" {
		return nil, fmt.Errorf("%w: empty Steam root", domain.ErrSteamNotFound)
	}
	binary, err := FindSteamBinary(root)
	if err != nil {
		return nil, err
	}
	return &Lifecycle{
		Root:         root,
		SteamBinary:  binary,
		ProcRoot:     "/proc",
		RegistryPath: "",
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

// AppLaunchSession asks Steam to launch the donor with an unguessable session
// argument. Steam shows its "Allow game launch?" confirmation for this URL
// form; the dispatcher helper consumes the argument.
func (l *Lifecycle) AppLaunchSession(ctx context.Context, appID uint32, token string) error {
	if !domain.ValidSessionToken(token) {
		return fmt.Errorf("%w: invalid session token", domain.ErrInvalidSession)
	}
	return l.AppLaunchURL(ctx, appID, "--better-rpt-session="+token)
}

// AppLaunchURL asks Steam to launch an AppID with extra arguments through the
// steam:// URL handler. The request is fire-and-forget for the same reason as
// AppLaunch: the launcher may become the Steam client.
func (l *Lifecycle) AppLaunchURL(ctx context.Context, appID uint32, args string) error {
	if appID == 0 {
		return fmt.Errorf("%w: empty app id", domain.ErrInvalidSession)
	}
	if l.SteamBinary == "" {
		return domain.ErrSteamNotFound
	}
	for i := 0; i < len(args); i++ {
		if args[i] < 0x20 || args[i] == 0x7f {
			return fmt.Errorf("%w: launch arguments contain control characters", domain.ErrInvalidSession)
		}
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	url := fmt.Sprintf("steam://run/%d//%s", appID, args)
	if err := l.start(l.SteamBinary, url); err != nil {
		return fmt.Errorf("launch app %d with arguments: %w", appID, err)
	}
	return nil
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

func (l *Lifecycle) start(name string, args ...string) error {
	if l.StartCommand != nil {
		return l.StartCommand(name, args...)
	}
	command := exec.Command(name, args...)
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
