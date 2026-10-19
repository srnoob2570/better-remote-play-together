package helper

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func helperSource(t *testing.T, mode os.FileMode) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "better-rpt-helper")
	if err := os.WriteFile(path, []byte("#!/bin/sh\nprintf helper\n"), mode); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatalf("chmod source: %v", err)
	}
	return path
}

func TestInstallerInstallAndReinstall(t *testing.T) {
	source := helperSource(t, 0o755)
	installer := &Installer{DataDir: t.TempDir(), SystemPath: source}
	first, err := installer.Install(false)
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if !first.Changed || first.Source != source || first.Destination != installer.Destination() {
		t.Fatalf("installed = %+v", first)
	}
	info, err := os.Stat(first.Destination)
	if err != nil {
		t.Fatalf("stat destination: %v", err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Fatalf("destination mode = %o, want 755", info.Mode().Perm())
	}
	binInfo, err := os.Stat(filepath.Dir(first.Destination))
	if err != nil {
		t.Fatalf("stat bin: %v", err)
	}
	if binInfo.Mode().Perm() != 0o700 {
		t.Fatalf("bin mode = %o, want 700", binInfo.Mode().Perm())
	}

	second, err := installer.Install(false)
	if err != nil {
		t.Fatalf("reinstall: %v", err)
	}
	if second.Changed {
		t.Fatal("unchanged helper reported as changed")
	}
}

func TestInstallerRefusesReplacementDuringActiveSession(t *testing.T) {
	source := helperSource(t, 0o755)
	installer := &Installer{DataDir: t.TempDir(), SystemPath: source}
	if _, err := installer.Install(false); err != nil {
		t.Fatalf("initial install: %v", err)
	}
	if err := os.WriteFile(source, []byte("#!/bin/sh\nprintf changed\n"), 0o755); err != nil {
		t.Fatalf("change source: %v", err)
	}
	if _, err := installer.Install(true); !errors.Is(err, ErrActiveSession) {
		t.Fatalf("error = %v, want ErrActiveSession", err)
	}
	replaced, err := installer.Install(false)
	if err != nil {
		t.Fatalf("replace: %v", err)
	}
	if !replaced.Changed {
		t.Fatal("changed helper not installed")
	}
}

func TestInstallerMissingAndInvalidSource(t *testing.T) {
	installer := &Installer{DataDir: t.TempDir(), SystemPath: filepath.Join(t.TempDir(), "missing")}
	if _, err := installer.Install(false); !errors.Is(err, ErrSourceMissing) {
		t.Fatalf("missing error = %v, want ErrSourceMissing", err)
	}

	invalid := helperSource(t, 0o644)
	installer = &Installer{DataDir: t.TempDir(), SystemPath: invalid}
	if _, err := installer.Install(false); !errors.Is(err, ErrSourceInvalid) {
		t.Fatalf("non-executable error = %v, want ErrSourceInvalid", err)
	}

	writable := helperSource(t, 0o775)
	installer = &Installer{DataDir: t.TempDir(), SystemPath: writable}
	if _, err := installer.Install(false); !errors.Is(err, ErrSourceInvalid) {
		t.Fatalf("group-writable error = %v, want ErrSourceInvalid", err)
	}
}
