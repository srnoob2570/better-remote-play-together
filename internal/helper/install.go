package helper

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/srnoob2570/better-rpt/internal/config"
	"github.com/srnoob2570/better-rpt/internal/fsutil"
)

// Errors returned by the stable helper installer.
var (
	ErrSourceMissing = errors.New("stable helper source not found")
	ErrSourceInvalid = errors.New("stable helper source is invalid")
	ErrInstall       = errors.New("stable helper installation failed")
	ErrActiveSession = errors.New("cannot replace the helper during an active session")
)

// Installed describes one installer result.
type Installed struct {
	Source      string
	Destination string
	SHA256      string
	Changed     bool
}

// Installer copies the packaged helper to the stable XDG path.
type Installer struct {
	DataDir    string
	AppDir     string
	SystemPath string
}

// NewInstaller uses the packaged helper locations from a .deb or AppImage.
func NewInstaller(paths config.Paths) *Installer {
	return &Installer{
		DataDir:    paths.DataDir,
		AppDir:     os.Getenv("APPDIR"),
		SystemPath: "/usr/lib/better-rpt/better-rpt-helper",
	}
}

// Destination returns the stable helper path.
func (i *Installer) Destination() string {
	return filepath.Join(i.DataDir, "bin", "better-rpt-helper")
}

// Sources returns candidate packaged helpers in preference order. An AppImage
// uses its own bundle; a .deb uses /usr/lib.
func (i *Installer) Sources() []string {
	sources := make([]string, 0, 2)
	if i.AppDir != "" {
		sources = append(sources, filepath.Join(i.AppDir, "usr", "lib", "better-rpt", "better-rpt-helper"))
	}
	if i.SystemPath != "" {
		sources = append(sources, i.SystemPath)
	}
	return sources
}

// Install copies the first valid packaged helper. It refuses to replace a
// different helper while a session is active.
func (i *Installer) Install(activeSession bool) (Installed, error) {
	source, data, sum, err := i.readSource()
	if err != nil {
		return Installed{}, err
	}
	destination := i.Destination()
	if err := fsutil.EnsurePrivateDir(filepath.Dir(destination)); err != nil {
		return Installed{}, fmt.Errorf("%w: %v", ErrInstall, err)
	}
	if existing, err := hashFile(destination); err == nil {
		if existing == sum {
			if err := ensureMode(destination, 0o755); err != nil {
				return Installed{}, err
			}
			return Installed{Source: source, Destination: destination, SHA256: sum}, nil
		}
		if activeSession {
			return Installed{}, fmt.Errorf("%w: destination %s differs", ErrActiveSession, destination)
		}
	}
	if err := fsutil.AtomicWriteFile(destination, data, 0o755); err != nil {
		return Installed{}, fmt.Errorf("%w: %v", ErrInstall, err)
	}
	return Installed{Source: source, Destination: destination, SHA256: sum, Changed: true}, nil
}

func (i *Installer) readSource() (string, []byte, string, error) {
	for _, candidate := range i.Sources() {
		info, err := os.Lstat(candidate)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return "", nil, "", fmt.Errorf("%w: inspect %s: %v", ErrSourceInvalid, candidate, err)
		}
		if !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 || info.Mode().Perm()&0o022 != 0 {
			return "", nil, "", fmt.Errorf("%w: %s", ErrSourceInvalid, candidate)
		}
		data, err := os.ReadFile(candidate)
		if err != nil {
			return "", nil, "", fmt.Errorf("%w: read %s: %v", ErrSourceInvalid, candidate, err)
		}
		sum := sha256.Sum256(data)
		return candidate, data, hex.EncodeToString(sum[:]), nil
	}
	return "", nil, "", ErrSourceMissing
}

func hashFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func ensureMode(path string, mode os.FileMode) error {
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("%w: stat %s: %v", ErrInstall, path, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%w: %s is not a regular file", ErrInstall, path)
	}
	if info.Mode().Perm() != mode {
		if err := os.Chmod(path, mode); err != nil {
			return fmt.Errorf("%w: chmod %s: %v", ErrInstall, path, err)
		}
	}
	return nil
}
