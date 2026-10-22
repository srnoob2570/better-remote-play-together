package session

import (
	"errors"
	"os"
	"testing"
)

func TestStoreLockContention(t *testing.T) {
	store := sessionStore(t)
	first, err := store.TryLock()
	if err != nil {
		t.Fatalf("first lock: %v", err)
	}
	locked, err := store.Locked()
	if err != nil || !locked {
		t.Fatalf("locked = %v, %v", locked, err)
	}
	if _, err := store.TryLock(); !errors.Is(err, ErrSessionBusy) {
		t.Fatalf("second lock error = %v, want ErrSessionBusy", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("close first: %v", err)
	}
	locked, err = store.Locked()
	if err != nil || locked {
		t.Fatalf("after close locked = %v, %v", locked, err)
	}
	second, err := store.TryLock()
	if err != nil {
		t.Fatalf("second lock after close: %v", err)
	}
	if err := second.Close(); err != nil {
		t.Fatalf("close second: %v", err)
	}
	info, err := os.Stat(store.LockPath())
	if err != nil {
		t.Fatalf("stat lock: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("lock mode = %o, want 600", info.Mode().Perm())
	}
}
