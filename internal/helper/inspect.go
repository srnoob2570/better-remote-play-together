package helper

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"syscall"

	"github.com/srnoob2570/better-rpt/internal/config"
)

// Info is the read-only state of the stable helper installation.
type Info struct {
	Path           string
	Exists         bool
	SHA256         string
	Mode           os.FileMode
	UID            uint32
	ExpectedSHA256 string
	ExpectedSource string
	Matches        bool
	SourceError    string
}

// Inspect reads the stable helper and its packaged source without installing,
// copying or changing permissions.
func Inspect(paths config.Paths) (Info, error) {
	installer := NewInstaller(paths)
	info := Info{Path: installer.Destination()}
	stat, err := os.Lstat(info.Path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return inspectSource(installer, info)
		}
		return info, fmt.Errorf("inspect helper: %w", err)
	}
	if stat.Mode()&os.ModeSymlink != 0 || !stat.Mode().IsRegular() {
		return info, fmt.Errorf("helper %s is not a regular file", info.Path)
	}
	info.Exists = true
	info.Mode = stat.Mode().Perm()
	if raw, ok := stat.Sys().(*syscall.Stat_t); ok {
		info.UID = raw.Uid
	}
	sum, err := hashFile(info.Path)
	if err != nil {
		return info, fmt.Errorf("hash helper: %w", err)
	}
	info.SHA256 = sum
	return inspectSource(installer, info)
}

func inspectSource(installer *Installer, info Info) (Info, error) {
	source, _, sum, err := installer.readSource()
	switch {
	case err == nil:
		info.ExpectedSource = source
		info.ExpectedSHA256 = sum
		info.Matches = info.Exists && info.SHA256 == sum
	case errors.Is(err, ErrSourceMissing):
	default:
		info.SourceError = err.Error()
	}
	return info, nil
}
