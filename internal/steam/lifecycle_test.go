package steam

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
	"time"
)

func fakeProc(t *testing.T) string {
	t.Helper()
	return t.TempDir()
}

func addProc(t *testing.T, root string, pid int, name string) {
	t.Helper()
	dir := filepath.Join(root, strconv.Itoa(pid))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("mkdir proc %d: %v", pid, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "comm"), []byte(name+"\n"), 0o600); err != nil {
		t.Fatalf("write comm %d: %v", pid, err)
	}
}

func removeProc(root string, pid int) {
	_ = os.RemoveAll(filepath.Join(root, strconv.Itoa(pid)))
}

func registryWithPID(t *testing.T, pid int) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "registry.vdf")
	content := `"Registry" { "HKCU" { "Software" { "Valve" { "Steam" { "SteamPID" "` + strconv.Itoa(pid) + `" } } } } }`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write registry: %v", err)
	}
	return path
}

func TestScanProcesses(t *testing.T) {
	root := fakeProc(t)
	addProc(t, root, 10, "steam")
	addProc(t, root, 11, "steamwebhelper")
	addProc(t, root, 12, "bash")
	addProc(t, root, 13, "steam-runtime")
	if err := os.MkdirAll(filepath.Join(root, "not-a-pid"), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	processes, err := ScanProcesses(root)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	want := []Process{{PID: 10, Name: "steam"}, {PID: 11, Name: "steamwebhelper"}}
	if !reflect.DeepEqual(processes, want) {
		t.Fatalf("processes = %+v, want %+v", processes, want)
	}
}

func TestIsRunningProcessAndRegistry(t *testing.T) {
	root := fakeProc(t)
	lifecycle := &Lifecycle{
		ProcRoot:     root,
		RegistryPath: filepath.Join(t.TempDir(), "missing.vdf"),
	}
	addProc(t, root, 20, "steam")
	running, err := lifecycle.IsRunning()
	if err != nil || !running {
		t.Fatalf("running = %v, err=%v", running, err)
	}
	removeProc(root, 20)

	addProc(t, root, 21, "steamwebhelper")
	running, err = lifecycle.IsRunning()
	if err != nil || !running {
		t.Fatalf("webhelper running = %v, err=%v", running, err)
	}
	removeProc(root, 21)

	lifecycle.RegistryPath = registryWithPID(t, 30)
	addProc(t, root, 30, "steam")
	running, err = lifecycle.IsRunning()
	if err != nil || !running {
		t.Fatalf("registry pid running = %v, err=%v", running, err)
	}
	removeProc(root, 30)
	if err := os.WriteFile(filepath.Join(root, "30", "comm"), nil, 0o600); err == nil {
		t.Fatal("removed process still present")
	}
	running, err = lifecycle.IsRunning()
	if err != nil || running {
		t.Fatalf("stale registry pid running = %v, err=%v", running, err)
	}
}

func TestShutdownAndRelaunchWithFakes(t *testing.T) {
	root := fakeProc(t)
	localConfig := filepath.Join(t.TempDir(), "localconfig.vdf")
	if err := os.WriteFile(localConfig, []byte("stable\n"), 0o600); err != nil {
		t.Fatalf("write localconfig: %v", err)
	}
	addProc(t, root, 40, "steam")
	var calls [][]string
	lifecycle := &Lifecycle{
		Root:            t.TempDir(),
		LocalConfigPath: localConfig,
		SteamBinary:     "/usr/bin/steam",
		ProcRoot:        root,
		RegistryPath:    filepath.Join(t.TempDir(), "missing.vdf"),
		RunCommand: func(ctx context.Context, name string, args ...string) error {
			calls = append(calls, append([]string{name}, args...))
			removeProc(root, 40)
			return nil
		},
		StartCommand: func(name string) error {
			addProc(t, root, 41, "steamwebhelper")
			return nil
		},
		ShutdownTimeout: 100 * time.Millisecond,
		StopTimeout:     100 * time.Millisecond,
		StableTimeout:   100 * time.Millisecond,
		StartTimeout:    100 * time.Millisecond,
		PollInterval:    time.Millisecond,
		StableInterval:  time.Millisecond,
	}
	if err := lifecycle.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
	want := [][]string{{"/usr/bin/steam", "-shutdown"}}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls = %+v, want %+v", calls, want)
	}
	running, err := lifecycle.IsRunning()
	if err != nil || running {
		t.Fatalf("still running = %v, err=%v", running, err)
	}
	if err := lifecycle.Relaunch(context.Background()); err != nil {
		t.Fatalf("relaunch: %v", err)
	}
	running, err = lifecycle.IsRunning()
	if err != nil || !running {
		t.Fatalf("not relaunched = %v, err=%v", running, err)
	}
}

func TestShutdownTimeoutLeavesProcess(t *testing.T) {
	root := fakeProc(t)
	addProc(t, root, 50, "steam")
	lifecycle := &Lifecycle{
		Root:            t.TempDir(),
		LocalConfigPath: filepath.Join(t.TempDir(), "localconfig.vdf"),
		SteamBinary:     "steam",
		ProcRoot:        root,
		RegistryPath:    filepath.Join(t.TempDir(), "missing.vdf"),
		RunCommand:      func(context.Context, string, ...string) error { return nil },
		ShutdownTimeout: 10 * time.Millisecond,
		StopTimeout:     5 * time.Millisecond,
		StableTimeout:   10 * time.Millisecond,
		PollInterval:    time.Millisecond,
		StableInterval:  time.Millisecond,
	}
	if err := lifecycle.Shutdown(context.Background()); err == nil {
		t.Fatal("shutdown timed out but returned nil")
	}
	running, err := lifecycle.IsRunning()
	if err != nil || !running {
		t.Fatalf("process disappeared after timeout: %v, %v", running, err)
	}
}

func TestWaitStable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "localconfig.vdf")
	if err := os.WriteFile(path, []byte("stable\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	lifecycle := &Lifecycle{
		LocalConfigPath: path,
		StableTimeout:   100 * time.Millisecond,
		StableInterval:  time.Millisecond,
	}
	if err := lifecycle.WaitStable(context.Background()); err != nil {
		t.Fatalf("wait stable: %v", err)
	}
}

func TestWaitStableMissing(t *testing.T) {
	lifecycle := &Lifecycle{
		LocalConfigPath: filepath.Join(t.TempDir(), "missing.vdf"),
		StableTimeout:   10 * time.Millisecond,
		StableInterval:  time.Millisecond,
	}
	if err := lifecycle.WaitStable(context.Background()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("error = %v, want not-exist", err)
	}
}
