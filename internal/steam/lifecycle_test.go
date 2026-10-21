package steam

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
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

func TestAppLaunchURLUsesSteamHandlerWithSessionArgument(t *testing.T) {
	var started [][]string
	lifecycle := &Lifecycle{
		SteamBinary: "/usr/bin/steam",
		StartCommand: func(name string, args ...string) error {
			started = append(started, append([]string{name}, args...))
			return nil
		},
	}
	token := "0123456789abcdef0123456789abcdef"
	if err := lifecycle.AppLaunchSession(context.Background(), 480, token); err != nil {
		t.Fatalf("applaunch session: %v", err)
	}
	want := [][]string{
		{"/usr/bin/steam", "steam://run/480//--better-rpt-session=" + token},
		{"/usr/bin/steam", "steam://open/main"},
	}
	if !reflect.DeepEqual(started, want) {
		t.Fatalf("started = %+v, want %+v", started, want)
	}
	if err := lifecycle.AppLaunchSession(context.Background(), 480, "short"); err == nil {
		t.Fatal("invalid token accepted")
	}
	if err := lifecycle.AppLaunchURL(context.Background(), 480, "bad\narg"); err == nil {
		t.Fatal("control character accepted")
	}
}

func TestAppLaunchSessionKeepsLaunchWhenFocusFails(t *testing.T) {
	var started [][]string
	lifecycle := &Lifecycle{
		SteamBinary: "/usr/bin/steam",
		StartCommand: func(name string, args ...string) error {
			started = append(started, append([]string{name}, args...))
			if len(args) > 0 && args[0] == "steam://open/main" {
				return errors.New("window manager denied activation")
			}
			return nil
		},
	}
	token := "0123456789abcdef0123456789abcdef"
	if err := lifecycle.AppLaunchSession(context.Background(), 480, token); err != nil {
		t.Fatalf("applaunch session with failed focus: %v", err)
	}
	want := [][]string{
		{"/usr/bin/steam", "steam://run/480//--better-rpt-session=" + token},
		{"/usr/bin/steam", "steam://open/main"},
	}
	if !reflect.DeepEqual(started, want) {
		t.Fatalf("started = %+v, want %+v", started, want)
	}
}
