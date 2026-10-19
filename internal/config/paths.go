package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/srnoob2570/better-rpt/internal/fsutil"
)

// Paths are the XDG locations used by Better RPT. They are user-scoped; the
// package never writes to the Steam installation.
type Paths struct {
	ConfigDir  string
	DataDir    string
	RuntimeDir string
	ConfigFile string
}

// ResolvePaths resolves the XDG config, data and runtime directories. Relative
// XDG values are ignored, as required by the XDG Base Directory specification.
func ResolvePaths() (Paths, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Paths{}, fmt.Errorf("resolve home directory: %w", err)
	}
	configHome := xdgDir("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	dataHome := xdgDir("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
	runtimeHome := os.Getenv("XDG_RUNTIME_DIR")
	if !filepath.IsAbs(runtimeHome) {
		runtimeHome = filepath.Join(os.TempDir(), fmt.Sprintf("better-rpt-%d", os.Getuid()))
	}

	configDir := filepath.Join(configHome, "better-rpt")
	return Paths{
		ConfigDir:  configDir,
		DataDir:    filepath.Join(dataHome, "better-rpt"),
		RuntimeDir: filepath.Join(runtimeHome, "better-rpt"),
		ConfigFile: filepath.Join(configDir, "config.json"),
	}, nil
}

// BackupsDir is the private directory for localconfig backups.
func (p Paths) BackupsDir() string {
	return filepath.Join(p.DataDir, "backups")
}

// HelperPath is the stable helper location shared by .deb and AppImage.
func (p Paths) HelperPath() string {
	return filepath.Join(p.DataDir, "bin", "better-rpt-helper")
}

// Ensure creates the private user directories.
func (p Paths) Ensure() error {
	for _, dir := range []string{p.ConfigDir, p.DataDir, p.RuntimeDir} {
		if err := fsutil.EnsurePrivateDir(dir); err != nil {
			return fmt.Errorf("prepare %s: %w", dir, err)
		}
	}
	return nil
}

func xdgDir(key, fallback string) string {
	if value := os.Getenv(key); filepath.IsAbs(value) {
		return value
	}
	return fallback
}
