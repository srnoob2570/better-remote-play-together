package update

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func (m *Manager) installDeb(
	ctx context.Context,
	checksumURL string,
	asset Asset,
	progress func(Progress),
) (InstallResult, error) {
	if filepath.Base(asset.Name) != asset.Name {
		return InstallResult{}, fmt.Errorf("%w: unsafe package name %q", ErrInvalidResponse, asset.Name)
	}
	dir, err := os.MkdirTemp("", "better-rpt-update-")
	if err != nil {
		return InstallResult{}, fmt.Errorf("%w: create temporary directory: %v", ErrDownload, err)
	}
	defer func() { _ = os.RemoveAll(dir) }()
	if err := os.Chmod(dir, 0o700); err != nil {
		return InstallResult{}, fmt.Errorf("%w: secure temporary directory: %v", ErrDownload, err)
	}
	packagePath := filepath.Join(dir, asset.Name)
	file, err := os.OpenFile(packagePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return InstallResult{}, fmt.Errorf("%w: create package: %v", ErrDownload, err)
	}
	if err := m.downloadVerified(ctx, checksumURL, asset, file, progress); err != nil {
		_ = file.Close()
		return InstallResult{}, err
	}
	m.report(StageInstalling, 0, 0, progress)

	pkexec := m.pkexecPath
	if pkexec == "" {
		pkexec = "/usr/bin/pkexec"
	}
	stat := m.env.stat
	if stat == nil {
		stat = os.Stat
	}
	info, err := stat(pkexec)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
		return InstallResult{}, fmt.Errorf("%w: %s", ErrNoPkexec, pkexec)
	}
	// Pass an absolute, packaged dpkg path. Resolving "dpkg" here would let a
	// PATH entry choose the program pkexec runs as root.
	command := exec.Command(pkexec, "/usr/bin/dpkg", "-i", packagePath)
	command.Stdin = nil
	var output bytes.Buffer
	command.Stdout = &output
	command.Stderr = &output
	if err := command.Run(); err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) && (exitError.ExitCode() == 126 || exitError.ExitCode() == 127) {
			return InstallResult{}, fmt.Errorf("%w: %s", ErrCanceled, outputText(output.String()))
		}
		return InstallResult{}, fmt.Errorf("%w: %v %s", ErrInstall, err, outputText(output.String()))
	}
	return InstallResult{
		Status:      StatusInstalled,
		InstallKind: KindDeb,
		Path:        "/usr/bin/better-rpt",
		Restart:     true,
	}, nil
}

func outputText(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 4096 {
		value = value[len(value)-4096:]
	}
	return value
}
