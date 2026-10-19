package app

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/srnoob2570/better-rpt/internal/config"
	"github.com/srnoob2570/better-rpt/internal/domain"
)

func testPaths(t *testing.T) config.Paths {
	t.Helper()
	base := t.TempDir()
	return config.Paths{
		ConfigDir:  filepath.Join(base, "config"),
		DataDir:    filepath.Join(base, "data"),
		RuntimeDir: filepath.Join(base, "runtime"),
		ConfigFile: filepath.Join(base, "config", "config.json"),
	}
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestInstanceLock(t *testing.T) {
	paths := testPaths(t)
	first, err := NewWithPaths(paths, testLogger())
	if err != nil {
		t.Fatalf("first service: %v", err)
	}
	defer first.Close()

	if _, err := NewWithPaths(paths, testLogger()); !errors.Is(err, domain.ErrInstanceLocked) {
		t.Fatalf("second service error = %v, want ErrInstanceLocked", err)
	}

	if err := first.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	third, err := NewWithPaths(paths, testLogger())
	if err != nil {
		t.Fatalf("service after close: %v", err)
	}
	if err := third.Close(); err != nil {
		t.Fatalf("close third: %v", err)
	}
}

func TestSetLanguagePersists(t *testing.T) {
	paths := testPaths(t)
	svc, err := NewWithPaths(paths, testLogger())
	if err != nil {
		t.Fatalf("service: %v", err)
	}
	if err := svc.SetLanguage("en"); err != nil {
		t.Fatalf("set language: %v", err)
	}
	if got := svc.Language(); got != "en" {
		t.Fatalf("language = %q, want en", got)
	}
	if got := svc.Config().Language; got != "en" {
		t.Fatalf("config language = %q, want en", got)
	}
	if err := svc.SetLanguage("fr"); !errors.Is(err, domain.ErrUnsupported) {
		t.Fatalf("unsupported error = %v", err)
	}
	if err := svc.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	reopened, err := NewWithPaths(paths, testLogger())
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer reopened.Close()
	if got := reopened.Language(); got != "en" {
		t.Fatalf("reopened language = %q, want en", got)
	}
}

func TestCorruptConfig(t *testing.T) {
	paths := testPaths(t)
	if err := paths.Ensure(); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	if err := os.WriteFile(paths.ConfigFile, []byte("{"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := NewWithPaths(paths, testLogger()); !errors.Is(err, domain.ErrConfig) {
		t.Fatalf("error = %v, want ErrConfig", err)
	}
}

func TestRefreshCatalog(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", "")
	steamRoot := filepath.Join(home, ".local", "share", "Steam")
	apps := filepath.Join(steamRoot, "steamapps")
	if err := os.MkdirAll(filepath.Join(apps, "common", "Native Game"), 0o700); err != nil {
		t.Fatalf("create steam tree: %v", err)
	}
	if err := os.WriteFile(filepath.Join(apps, "common", "Native Game", "game.x86_64"), []byte("fixture\n"), 0o755); err != nil {
		t.Fatalf("write executable: %v", err)
	}
	appInfo, err := os.ReadFile(filepath.Join("..", "..", "testdata", "steam", "fixtures", "primary", "appcache", "appinfo.vdf"))
	if err != nil {
		t.Fatalf("read appinfo fixture: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(steamRoot, "appcache"), 0o700); err != nil {
		t.Fatalf("create appcache: %v", err)
	}
	if err := os.WriteFile(filepath.Join(steamRoot, "appcache", "appinfo.vdf"), appInfo, 0o600); err != nil {
		t.Fatalf("write appinfo: %v", err)
	}
	manifest := `"AppState" { "appid" "900001" "name" "Native Game" "installdir" "Native Game" "StateFlags" "4" "SizeOnDisk" "42" }`
	if err := os.WriteFile(filepath.Join(apps, "appmanifest_900001.acf"), []byte(manifest), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	svc, err := NewWithPaths(testPaths(t), testLogger())
	if err != nil {
		t.Fatalf("service: %v", err)
	}
	defer svc.Close()

	if _, ok := svc.Catalog(); ok {
		t.Fatal("catalog available before refresh")
	}
	catalog, err := svc.RefreshCatalog(context.Background())
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if len(catalog.Games) != 1 || catalog.Games[0].Name != "Native Game" || catalog.Games[0].SizeBytes != 42 || catalog.Games[0].Status != domain.StatusReady {
		t.Fatalf("catalog = %+v", catalog)
	}
	cached, ok := svc.Catalog()
	if !ok || len(cached.Games) != 1 {
		t.Fatalf("cached catalog = %+v, ok=%v", cached, ok)
	}
}

func TestCloseIsIdempotent(t *testing.T) {
	svc, err := NewWithPaths(testPaths(t), testLogger())
	if err != nil {
		t.Fatalf("service: %v", err)
	}
	if err := svc.Close(); err != nil {
		t.Fatalf("first close: %v", err)
	}
	if err := svc.Close(); err != nil {
		t.Fatalf("second close: %v", err)
	}
}
