package diagnostics

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/srnoob2570/better-rpt/internal/config"
	"github.com/srnoob2570/better-rpt/internal/helper"
	"github.com/srnoob2570/better-rpt/internal/session"
	"github.com/srnoob2570/better-rpt/internal/steam"
)

func TestFedoraAppImageGolden(t *testing.T) {
	report := Run(context.Background(), Options{
		Version:  "0.2.0",
		Language: "es",
		Deps:     fedoraDeps(),
		Now:      time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC),
	})
	if report.Summary.Counts.Fail == 0 {
		t.Fatalf("expected Fedora AppImage failures, got %+v", report.Summary)
	}
	data, err := RenderJSON(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateJSON(data); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, data)
	}
	if strings.Contains(string(data), "fedora") || strings.Contains(string(data), "0123456789abcdef0123456789abcdef") {
		t.Fatalf("report leaked identity: %s", data)
	}
	golden := filepath.Join("testdata", "fedora-appimage.json")
	if os.Getenv("UPDATE_DIAGNOSTICS_GOLDEN") == "1" {
		if err := os.MkdirAll(filepath.Dir(golden), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(golden, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("missing golden (run UPDATE_DIAGNOSTICS_GOLDEN=1): %v", err)
	}
	if !bytes.Equal(data, want) {
		t.Fatalf("golden mismatch\n got: %s\nwant: %s", data, want)
	}
	markdown := RenderMarkdown(report)
	if !strings.Contains(markdown, "libgtk-3.so.0") || !strings.Contains(markdown, "fuse3") {
		t.Fatalf("markdown is missing actionable evidence:\n%s", markdown)
	}
}

func TestRunWithoutDisplayReportsSessionFail(t *testing.T) {
	deps := fedoraDeps()
	base := deps.Getenv
	deps.Getenv = func(key string) string {
		switch key {
		case "DISPLAY", "WAYLAND_DISPLAY", "XDG_SESSION_TYPE":
			return ""
		default:
			return base(key)
		}
	}
	report := Run(context.Background(), Options{Version: "0.2.0", Language: "en", Deps: deps,
		Now: time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)})
	for _, check := range report.Checks {
		if check.ID == CheckSessionEnv {
			if check.Status != StatusFail {
				t.Fatalf("session check = %s", check.Status)
			}
			return
		}
	}
	t.Fatal("session check missing")
}

func TestRunAsRootReportsFail(t *testing.T) {
	deps := fedoraDeps()
	deps.EUID = 0
	report := Run(context.Background(), Options{Version: "0.2.0", Language: "en", Deps: deps,
		Now: time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)})
	found := false
	for _, check := range report.Checks {
		if check.ID == CheckRoot {
			found = true
			if check.Status != StatusFail {
				t.Fatalf("root check = %s", check.Status)
			}
		}
	}
	if !found {
		t.Fatal("root check missing")
	}
}

func TestCLIExitCodesAndOutput(t *testing.T) {
	okay := minimalReport(StatusOK, 0)
	failing := minimalReport(StatusFail, 1)
	cases := []struct {
		name string
		args []string
		code int
	}{
		{"ok", []string{"--diagnose", "--json"}, 0},
		{"fail", []string{"--diagnose", "--json"}, 1},
		{"extra", []string{"--diagnose", "extra"}, 2},
		{"missing diagnose", []string{"--json"}, 2},
		{"bad flag", []string{"--diagnose", "--nope"}, 2},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			report := okay
			if test.name == "fail" {
				report = failing
			}
			code := runCLI(test.args, &stdout, &stderr, "0.2.0", func(context.Context, Options) (Report, error) {
				return report, nil
			})
			if code != test.code {
				t.Fatalf("code=%d want=%d stderr=%s", code, test.code, stderr.String())
			}
			if test.name == "ok" && !json.Valid(stdout.Bytes()) {
				t.Fatalf("stdout is not JSON: %s", stdout.String())
			}
		})
	}

	out := filepath.Join(t.TempDir(), "report.md")
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"--diagnose", "--out", out}, &stdout, &stderr, "0.2.0",
		func(context.Context, Options) (Report, error) { return okay, nil })
	if code != 0 {
		t.Fatalf("out code=%d stderr=%s", code, stderr.String())
	}
	if info, err := os.Stat(out); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("report file: %v %#v", err, info)
	}
}

func minimalReport(status Status, fails int) Report {
	return Report{
		SchemaVersion: SchemaVersion,
		GeneratedAt:   "2026-10-20T12:00:00Z",
		Version:       "0.2.0",
		Format:        "appimage",
		Language:      "es",
		Summary: Summary{
			Verdict: status,
			Counts:  Counts{Fail: fails, OK: 1 - fails},
			Total:   1,
		},
		Checks: []Check{{
			ID:       CheckVersion,
			Status:   status,
			Title:    "Versión",
			Message:  "Mensaje",
			Evidence: []string{},
		}},
	}
}

func fedoraDeps() *Deps {
	home := "/home/fedora"
	paths := config.Paths{
		ConfigDir:  home + "/.config/better-rpt",
		DataDir:    home + "/.local/share/better-rpt",
		RuntimeDir: "/run/user/1000/better-rpt",
		ConfigFile: home + "/.config/better-rpt/config.json",
	}
	files := map[string]string{
		"/etc/os-release":            "NAME=\"Fedora Linux\"\nID=fedora\nVERSION_ID=44\nPRETTY_NAME=\"Fedora Linux 44 (Workstation Edition)\"\n",
		"/proc/sys/kernel/osrelease": "6.15.0-200.fc44.x86_64\n",
		"/proc/self/mountinfo":       "36 35 0:32 / " + home + "/Apps rw,nosuid,nodev - fuse.better-rpt none rw\n",
	}
	dirs := map[string]bool{
		paths.ConfigDir:  true,
		paths.DataDir:    true,
		paths.RuntimeDir: true,
		home + "/Apps":   true,
	}
	env := map[string]string{
		"APPIMAGE":            home + "/Apps/BetterRPT-0.2.0-x86_64.AppImage",
		"XDG_SESSION_TYPE":    "wayland",
		"WAYLAND_DISPLAY":     "wayland-0",
		"XDG_CURRENT_DESKTOP": "GNOME",
		"LANG":                "en_US.UTF-8",
	}
	return &Deps{
		Now:    func() time.Time { return time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC) },
		Getenv: func(key string) string { return env[key] },
		ReadFile: func(path string) ([]byte, error) {
			if value, ok := files[path]; ok {
				return []byte(value), nil
			}
			return nil, fs.ErrNotExist
		},
		Stat: func(path string) (os.FileInfo, error) {
			if dirs[path] {
				return fakeInfo{name: filepath.Base(path), mode: os.ModeDir | 0o755, dir: true}, nil
			}
			if path == home+"/Apps/BetterRPT-0.2.0-x86_64.AppImage" {
				return fakeInfo{name: filepath.Base(path), mode: 0o755}, nil
			}
			return nil, fs.ErrNotExist
		},
		Lstat:    func(path string) (os.FileInfo, error) { return nil, fs.ErrNotExist },
		Access:   func(string, uint32) error { return nil },
		CanWrite: func(string, bool) error { return nil },
		LookPath: func(string) (string, error) { return "", fs.ErrNotExist },
		Run: func(_ context.Context, name string, args ...string) ([]byte, error) {
			if name == "ldd" && len(args) == 1 && args[0] == "--version" {
				return []byte("ldd (GNU libc) 2.39\n"), nil
			}
			return nil, fs.ErrNotExist
		},
		Executable: func() (string, error) { return home + "/Apps/BetterRPT-0.2.0-x86_64.AppImage", nil },
		EUID:       1000,
		Home:       home,
		Username:   "fedora",
		GOOS:       "linux",
		GOARCH:     "amd64",
		ProcRoot:   "/proc",
		MountInfo:  "/proc/self/mountinfo",
		Paths:      paths,
		LoadConfig: func(config.Paths) (config.Config, error) { return config.Default(), nil },
		DetectSteam: func(context.Context) (steam.Installation, error) {
			return steam.Installation{}, &steam.UnsupportedError{
				Variant: steam.VariantFlatpak,
				Root:    home + "/.var/app/com.valvesoftware.Steam/.local/share/Steam",
			}
		},
		LoadCatalog: func(context.Context, steam.Installation) (steam.Catalog, error) {
			return steam.Catalog{}, errors.New("unexpected catalog load")
		},
		ScanSteam: func(string) ([]steam.Process, error) { return nil, nil },
		SelectAccount: func(string) (config.SteamAccount, error) {
			return config.SteamAccount{}, errors.New("unexpected account load")
		},
		DetectSetup: func(config.Paths, string) (config.SetupSnapshot, error) {
			return config.SetupSnapshot{}, errors.New("unexpected setup load")
		},
		ListBackups: func(config.Paths) ([]config.Backup, error) { return nil, nil },
		InspectOverlay: func(string) (config.OverlayInspection, error) {
			return config.OverlayInspection{}, errors.New("unexpected overlay load")
		},
		InspectControls: func(string) (config.ControllerLayoutInspection, error) {
			return config.ControllerLayoutInspection{}, errors.New("unexpected controller load")
		},
		InspectHelper: func(paths config.Paths) (helper.Info, error) {
			return helper.Info{Path: paths.HelperPath()}, nil
		},
		ReadLastLaunch: func(config.Paths) (*session.LastLaunch, error) {
			return nil, session.ErrLastLaunchMissing
		},
		ActiveSession: func(config.Paths) (*session.State, error) { return nil, nil },
		ProcessAlive:  func(int, uint64) bool { return false },
		PIDAlive:      func(int) bool { return false },
		ReadLogTail:   func(config.Paths, int) ([]string, error) { return nil, fs.ErrNotExist },
		LogPath:       func(config.Paths) string { return paths.DataDir + "/logs/better-rpt.log" },
		ELFNeeded: func(string) ([]string, error) {
			return []string{"libgtk-3.so.0", "libwebkit2gtk-4.1.so.0"}, nil
		},
		LDD: func(context.Context, string) ([]byte, error) {
			return []byte("libgtk-3.so.0 => not found\nlibwebkit2gtk-4.1.so.0 => not found\n"), errors.New("ldd exited nonzero")
		},
	}
}

type fakeInfo struct {
	name string
	mode os.FileMode
	dir  bool
}

func (f fakeInfo) Name() string       { return f.name }
func (f fakeInfo) Size() int64        { return 0 }
func (f fakeInfo) Mode() os.FileMode  { return f.mode }
func (f fakeInfo) ModTime() time.Time { return time.Time{} }
func (f fakeInfo) IsDir() bool        { return f.dir }
func (f fakeInfo) Sys() any           { return nil }
