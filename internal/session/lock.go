package session

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

// Lock is the global helper lock. The helper holds it for the whole target
// lifetime; the app probes it before starting a new session.
type Lock struct {
	file *os.File
}

// TryLock acquires the global session lock without waiting. It returns
// ErrSessionBusy when another helper already holds it.
func (s Store) TryLock() (*Lock, error) {
	if err := s.Ensure(); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(s.LockPath(), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("%w: open session lock: %v", ErrStore, err)
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = file.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, ErrSessionBusy
		}
		return nil, fmt.Errorf("%w: lock session: %v", ErrStore, err)
	}
	return &Lock{file: file}, nil
}

// Locked reports whether another helper holds the global session lock.
func (s Store) Locked() (bool, error) {
	lock, err := s.TryLock()
	if err == nil {
		return false, lock.Close()
	}
	if errors.Is(err, ErrSessionBusy) {
		return true, nil
	}
	return false, err
}

// Close releases the session lock.
func (l *Lock) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	_ = syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)
	err := l.file.Close()
	l.file = nil
	return err
}
