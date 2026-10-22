package update

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

func (m *Manager) installAppImage(
	ctx context.Context,
	checksumURL string,
	asset Asset,
	version string,
	target string,
	progress func(Progress),
) (InstallResult, error) {
	dir := filepath.Dir(target)
	temp, err := os.CreateTemp(dir, ".better-rpt-download-*")
	saved := false
	if err != nil {
		saved = true
		if m.downloadsDir == "" {
			return InstallResult{}, fmt.Errorf("%w: %v", ErrDownload, err)
		}
		if err := os.MkdirAll(m.downloadsDir, 0o755); err != nil {
			return InstallResult{}, fmt.Errorf("%w: create downloads directory: %v", ErrDownload, err)
		}
		temp, err = os.CreateTemp(m.downloadsDir, ".better-rpt-download-*")
		if err != nil {
			return InstallResult{}, fmt.Errorf("%w: %v", ErrDownload, err)
		}
	}
	tempName := temp.Name()
	defer func() { _ = os.Remove(tempName) }()
	if err := temp.Chmod(0o600); err != nil {
		_ = temp.Close()
		return InstallResult{}, fmt.Errorf("%w: set temporary permissions: %v", ErrDownload, err)
	}
	if err := m.downloadVerified(ctx, checksumURL, asset, temp, progress); err != nil {
		_ = temp.Close()
		return InstallResult{}, err
	}
	m.report(StageInstalling, 0, 0, progress)
	if err := os.Chmod(tempName, 0o755); err != nil {
		return InstallResult{}, fmt.Errorf("%w: mark AppImage executable: %v", ErrInstall, err)
	}
	final := target
	status := StatusReplaced
	if saved {
		final = filepath.Join(m.downloadsDir, "BetterRPT-"+version+"-x86_64.AppImage")
		status = StatusSaved
	}
	if err := os.Rename(tempName, final); err != nil {
		return InstallResult{}, fmt.Errorf("%w: replace %s: %v", ErrInstall, final, err)
	}
	if err := syncDir(filepath.Dir(final)); err != nil {
		return InstallResult{}, fmt.Errorf("%w: sync %s: %v", ErrInstall, filepath.Dir(final), err)
	}
	return InstallResult{
		Status:      status,
		InstallKind: KindAppImage,
		Path:        final,
		Restart:     status == StatusReplaced,
	}, nil
}

func syncDir(dir string) error {
	file, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	if err := file.Sync(); err != nil && !errors.Is(err, syscall.EINVAL) {
		return err
	}
	return nil
}
