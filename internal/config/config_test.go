package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/srnoob2570/better-rpt/internal/domain"
)

func testPaths(t *testing.T) Paths {
	t.Helper()
	base := t.TempDir()
	return Paths{
		ConfigDir:  filepath.Join(base, "config"),
		DataDir:    filepath.Join(base, "data"),
		RuntimeDir: filepath.Join(base, "runtime"),
		ConfigFile: filepath.Join(base, "config", "config.json"),
	}
}

func TestLoadMissing(t *testing.T) {
	cfg, err := Load(testPaths(t))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg != (Config{}) {
		t.Fatalf("config = %+v, want defaults", cfg)
	}
}

func TestSaveLoad(t *testing.T) {
	paths := testPaths(t)
	if err := Save(paths, Config{Language: "en"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	info, err := os.Stat(paths.ConfigFile)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("mode = %o, want 600", got)
	}
	cfg, err := Load(paths)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.Language != "en" {
		t.Fatalf("language = %q, want en", cfg.Language)
	}
}

func TestLoadMalformed(t *testing.T) {
	paths := testPaths(t)
	if err := paths.Ensure(); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	if err := os.WriteFile(paths.ConfigFile, []byte("{"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := Load(paths); err == nil {
		t.Fatal("malformed config accepted")
	}
}

func TestLoadUnsupportedLanguageFallsBackToAuto(t *testing.T) {
	paths := testPaths(t)
	if err := paths.Ensure(); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	if err := os.WriteFile(paths.ConfigFile, []byte(`{"language":"fr"}`), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	cfg, err := Load(paths)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.Language != "" {
		t.Fatalf("language = %q, want auto", cfg.Language)
	}
}

func TestSaveUnsupportedLanguage(t *testing.T) {
	err := Save(testPaths(t), Config{Language: "fr"})
	if !errors.Is(err, domain.ErrUnsupported) {
		t.Fatalf("error = %v, want ErrUnsupported", err)
	}
}
