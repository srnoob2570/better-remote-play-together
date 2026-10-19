package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/srnoob2570/better-rpt/internal/domain"
	"github.com/srnoob2570/better-rpt/internal/fsutil"
)

type instanceLock struct {
	file *os.File
}

func acquireInstanceLock(dir string) (*instanceLock, error) {
	if err := fsutil.EnsurePrivateDir(dir); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrPaths, err)
	}
	path := filepath.Join(dir, "app.lock")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("%w: open lock: %v", domain.ErrPaths, err)
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		file.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, domain.ErrInstanceLocked
		}
		return nil, fmt.Errorf("%w: lock instance: %v", domain.ErrPaths, err)
	}
	if err := file.Truncate(0); err != nil {
		unlockAndClose(file)
		return nil, fmt.Errorf("%w: truncate lock: %v", domain.ErrPaths, err)
	}
	if _, err := file.Seek(0, 0); err != nil {
		unlockAndClose(file)
		return nil, fmt.Errorf("%w: seek lock: %v", domain.ErrPaths, err)
	}
	if _, err := file.WriteString(strconv.Itoa(os.Getpid()) + "\n"); err != nil {
		unlockAndClose(file)
		return nil, fmt.Errorf("%w: write lock: %v", domain.ErrPaths, err)
	}
	if err := file.Sync(); err != nil {
		unlockAndClose(file)
		return nil, fmt.Errorf("%w: sync lock: %v", domain.ErrPaths, err)
	}
	return &instanceLock{file: file}, nil
}

func (l *instanceLock) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	unlockAndClose(l.file)
	l.file = nil
	return nil
}

func unlockAndClose(file *os.File) {
	_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
	_ = file.Close()
}
