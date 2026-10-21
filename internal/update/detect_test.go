package update

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestDetectInstallAppImage(t *testing.T) {
	appImage := filepath.Join(t.TempDir(), "BetterRPT-0.2.1-x86_64.AppImage")
	if err := os.WriteFile(appImage, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	mount := t.TempDir()
	env := environment{
		getenv: func(key string) string {
			if key == "APPIMAGE" {
				return appImage
			}
			if key == "APPDIR" {
				return mount
			}
			return ""
		},
		executable:   func() (string, error) { return filepath.Join(mount, "usr/bin/better-rpt"), nil },
		stat:         os.Stat,
		evalSymlinks: filepath.EvalSymlinks,
		lookPath:     func(string) (string, error) { return "", errors.New("not found") },
		output:       func(context.Context, string, ...string) ([]byte, error) { return nil, errors.New("unused") },
	}
	info := detectInstall(env)
	if info.Kind != KindAppImage || info.Path != appImage {
		t.Fatalf("info = %+v", info)
	}
}

func TestDetectInstallDeb(t *testing.T) {
	env := environment{
		getenv:       func(string) string { return "" },
		executable:   func() (string, error) { return "/usr/bin/better-rpt", nil },
		stat:         os.Stat,
		evalSymlinks: filepath.EvalSymlinks,
		lookPath:     func(string) (string, error) { return "/usr/bin/dpkg", nil },
		output: func(context.Context, string, ...string) ([]byte, error) {
			return []byte("better-rpt: /usr/bin/better-rpt\n"), nil
		},
	}
	info := detectInstall(env)
	if info.Kind != KindDeb || info.Path != "/usr/bin/better-rpt" {
		t.Fatalf("info = %+v", info)
	}
}

func TestDetectInstallDevelopment(t *testing.T) {
	env := environment{
		getenv:       func(string) string { return "" },
		executable:   func() (string, error) { return "/usr/local/bin/better-rpt", nil },
		stat:         os.Stat,
		evalSymlinks: filepath.EvalSymlinks,
		lookPath:     func(string) (string, error) { return "/usr/bin/dpkg", nil },
		output: func(context.Context, string, ...string) ([]byte, error) {
			return []byte("other-package: /usr/bin/better-rpt\n"), nil
		},
	}
	info := detectInstall(env)
	if info.Kind != KindDevelopment || info.Path != "/usr/local/bin/better-rpt" {
		t.Fatalf("info = %+v", info)
	}
}

func TestAppImageRequiresExecutableFile(t *testing.T) {
	appImage := filepath.Join(t.TempDir(), "BetterRPT.AppImage")
	if err := os.WriteFile(appImage, []byte("binary"), 0o644); err != nil {
		t.Fatal(err)
	}
	env := environment{
		getenv: func(key string) string {
			if key == "APPIMAGE" {
				return appImage
			}
			return ""
		},
		executable:   func() (string, error) { return appImage, nil },
		stat:         os.Stat,
		evalSymlinks: filepath.EvalSymlinks,
		lookPath:     func(string) (string, error) { return "", errors.New("not found") },
		output:       func(context.Context, string, ...string) ([]byte, error) { return nil, errors.New("unused") },
	}
	if info := detectInstall(env); info.Kind != KindDevelopment {
		t.Fatalf("info = %+v, want development", info)
	}
}
