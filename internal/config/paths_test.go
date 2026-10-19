package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePathsXDG(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "home")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(base, "cfg"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(base, "data"))
	t.Setenv("XDG_RUNTIME_DIR", filepath.Join(base, "run"))

	paths, err := ResolvePaths()
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if want := filepath.Join(base, "cfg", "better-rpt"); paths.ConfigDir != want {
		t.Fatalf("ConfigDir = %q, want %q", paths.ConfigDir, want)
	}
	if want := filepath.Join(base, "data", "better-rpt"); paths.DataDir != want {
		t.Fatalf("DataDir = %q, want %q", paths.DataDir, want)
	}
	if want := filepath.Join(base, "run", "better-rpt"); paths.RuntimeDir != want {
		t.Fatalf("RuntimeDir = %q, want %q", paths.RuntimeDir, want)
	}
	if want := filepath.Join(paths.ConfigDir, "config.json"); paths.ConfigFile != want {
		t.Fatalf("ConfigFile = %q, want %q", paths.ConfigFile, want)
	}
}

func TestResolvePathsFallback(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "home")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("XDG_RUNTIME_DIR", "")

	paths, err := ResolvePaths()
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if want := filepath.Join(home, ".config", "better-rpt"); paths.ConfigDir != want {
		t.Fatalf("ConfigDir = %q, want %q", paths.ConfigDir, want)
	}
	if want := filepath.Join(home, ".local", "share", "better-rpt"); paths.DataDir != want {
		t.Fatalf("DataDir = %q, want %q", paths.DataDir, want)
	}
	wantRun := filepath.Join(os.TempDir(), fmt.Sprintf("better-rpt-%d", os.Getuid()), "better-rpt")
	if paths.RuntimeDir != wantRun {
		t.Fatalf("RuntimeDir = %q, want %q", paths.RuntimeDir, wantRun)
	}
}

func TestResolvePathsIgnoresRelativeXDG(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "home")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "relative/config")
	t.Setenv("XDG_DATA_HOME", "relative/data")
	t.Setenv("XDG_RUNTIME_DIR", "relative/run")

	paths, err := ResolvePaths()
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if want := filepath.Join(home, ".config", "better-rpt"); paths.ConfigDir != want {
		t.Fatalf("ConfigDir = %q, want %q", paths.ConfigDir, want)
	}
	if want := filepath.Join(home, ".local", "share", "better-rpt"); paths.DataDir != want {
		t.Fatalf("DataDir = %q, want %q", paths.DataDir, want)
	}
}

func TestPathsEnsure(t *testing.T) {
	base := t.TempDir()
	paths := Paths{
		ConfigDir:  filepath.Join(base, "config"),
		DataDir:    filepath.Join(base, "data"),
		RuntimeDir: filepath.Join(base, "runtime"),
		ConfigFile: filepath.Join(base, "config", "config.json"),
	}
	if err := paths.Ensure(); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	for _, dir := range []string{paths.ConfigDir, paths.DataDir, paths.RuntimeDir} {
		info, err := os.Stat(dir)
		if err != nil {
			t.Fatalf("stat %s: %v", dir, err)
		}
		if got := info.Mode().Perm(); got != 0o700 {
			t.Fatalf("%s mode = %o, want 700", dir, got)
		}
	}
}
