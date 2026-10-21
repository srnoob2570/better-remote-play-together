package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/srnoob2570/better-rpt/internal/domain"
	"github.com/srnoob2570/better-rpt/internal/fsutil"
)

type instanceLock struct {
	file *os.File
}

const (
	// Set by the updater when it launches the replacement binary. The new
	// process briefly waits for this one to release the instance lock.
	waitForInstanceLockEnv = "BETTER_RPT_WAIT_FOR_INSTANCE_LOCK"
	waitForInstanceLockTTL = 5 * time.Second
)

func acquireInstanceLock(dir string) (*instanceLock, error) {
	if err := fsutil.EnsurePrivateDir(dir); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrPaths, err)
	}
	path := filepath.Join(dir, "app.lock")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("%w: open lock: %v", domain.ErrPaths, err)
	}
	deadline := time.Now().Add(waitForInstanceLockTTL)
	for {
		err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			break
		}
		if !errors.Is(err, syscall.EWOULDBLOCK) {
			file.Close()
			return nil, fmt.Errorf("%w: lock instance: %v", domain.ErrPaths, err)
		}
		if os.Getenv(waitForInstanceLockEnv) != "1" || time.Now().After(deadline) {
			file.Close()
			return nil, domain.ErrInstanceLocked
		}
		time.Sleep(50 * time.Millisecond)
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
