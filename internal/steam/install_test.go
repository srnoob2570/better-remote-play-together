package steam

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/srnoob2570/better-rpt/internal/domain"
)

func isolateHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", "")
	return home
}

func makeSteamRoot(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(path, "steamapps"), 0o700); err != nil {
		t.Fatalf("create steam root %s: %v", path, err)
	}
}

func TestDetectNative(t *testing.T) {
	home := isolateHome(t)
	root := filepath.Join(home, ".local", "share", "Steam")
	makeSteamRoot(t, root)

	inst, err := Detect(context.Background())
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	want, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatalf("resolve fixture: %v", err)
	}
	if inst.Variant != VariantNative || inst.Root != want {
		t.Fatalf("installation = %+v, want native %s", inst, want)
	}
}

func TestDetectSteamSymlinkRoot(t *testing.T) {
	home := isolateHome(t)
	realRoot := filepath.Join(t.TempDir(), "Steam")
	makeSteamRoot(t, realRoot)
	link := filepath.Join(home, ".steam", "steam")
	if err := os.MkdirAll(filepath.Dir(link), 0o700); err != nil {
		t.Fatalf("create link parent: %v", err)
	}
	if err := os.Symlink(realRoot, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	inst, err := Detect(context.Background())
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if inst.Root != realRoot {
		t.Fatalf("root = %q, want %q", inst.Root, realRoot)
	}
}

func TestDetectXDGDataHome(t *testing.T) {
	isolateHome(t)
	dataHome := filepath.Join(t.TempDir(), "data")
	t.Setenv("XDG_DATA_HOME", dataHome)
	root := filepath.Join(dataHome, "Steam")
	makeSteamRoot(t, root)

	inst, err := Detect(context.Background())
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if inst.Root != root {
		t.Fatalf("root = %q, want %q", inst.Root, root)
	}
}

func TestDetectFlatpak(t *testing.T) {
	home := isolateHome(t)
	root := filepath.Join(home, ".var", "app", "com.valvesoftware.Steam", ".local", "share", "Steam")
	makeSteamRoot(t, root)

	_, err := Detect(context.Background())
	if !errors.Is(err, domain.ErrSteamUnsupported) {
		t.Fatalf("error = %v, want ErrSteamUnsupported", err)
	}
	var unsupported *UnsupportedError
	if !errors.As(err, &unsupported) || unsupported.Variant != VariantFlatpak {
		t.Fatalf("error = %#v, want flatpak UnsupportedError", err)
	}
}

func TestDetectSnap(t *testing.T) {
	home := isolateHome(t)
	root := filepath.Join(home, "snap", "steam", "common", ".local", "share", "Steam")
	makeSteamRoot(t, root)

	_, err := Detect(context.Background())
	if !errors.Is(err, domain.ErrSteamUnsupported) {
		t.Fatalf("error = %v, want ErrSteamUnsupported", err)
	}
	var unsupported *UnsupportedError
	if !errors.As(err, &unsupported) || unsupported.Variant != VariantSnap {
		t.Fatalf("error = %#v, want snap UnsupportedError", err)
	}
}

func TestDetectNativeWinsOverFlatpak(t *testing.T) {
	home := isolateHome(t)
	makeSteamRoot(t, filepath.Join(home, ".local", "share", "Steam"))
	makeSteamRoot(t, filepath.Join(home, ".var", "app", "com.valvesoftware.Steam", ".local", "share", "Steam"))

	inst, err := Detect(context.Background())
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if inst.Variant != VariantNative {
		t.Fatalf("variant = %s, want native", inst.Variant)
	}
}

func TestDetectNotFound(t *testing.T) {
	isolateHome(t)
	_, err := Detect(context.Background())
	if !errors.Is(err, domain.ErrSteamNotFound) {
		t.Fatalf("error = %v, want ErrSteamNotFound", err)
	}
}

func TestDetectCanceled(t *testing.T) {
	isolateHome(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := Detect(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}
