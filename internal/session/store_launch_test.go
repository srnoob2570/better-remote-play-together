package session

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/srnoob2570/better-rpt/internal/domain"
)

func testLast(appID uint32, source LaunchSource, outcome LaunchOutcome) LastLaunch {
	now := time.Now().UTC()
	last := LastLaunch{
		Version:   LastLaunchVersion,
		AppID:     appID,
		Name:      "Game",
		Source:    source,
		StartedAt: now,
		Outcome:   outcome,
	}
	if source == SourceBetterRPT {
		last.Token = testToken
		last.Parameters = []string{"--windowed"}
		last.Kind = domain.KindNative.String()
	}
	return last
}

func TestStoreLastLaunchRoundTrip(t *testing.T) {
	store := sessionStore(t)
	first := testLast(900001, SourceBetterRPT, LaunchOK)
	first.EndedAt = first.StartedAt.Add(time.Minute)
	if err := store.WriteLastLaunch(first); err != nil {
		t.Fatalf("write last: %v", err)
	}
	info, err := os.Stat(store.LastLaunchPath())
	if err != nil {
		t.Fatalf("stat last: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o, want 600", info.Mode().Perm())
	}
	reloaded := NewStore(filepath.Dir(store.Dir))
	got, err := reloaded.ReadLastLaunch()
	if err != nil {
		t.Fatalf("read last: %v", err)
	}
	if got.AppID != first.AppID || got.Token != first.Token || got.Outcome != LaunchOK ||
		got.Source != SourceBetterRPT || len(got.Parameters) != 1 || got.EndedAt.IsZero() {
		t.Fatalf("last = %+v, want %+v", got, first)
	}

	second := testLast(900002, SourceSteam, LaunchUnknown)
	if err := store.WriteLastLaunch(second); err != nil {
		t.Fatalf("write second: %v", err)
	}
	got, err = store.ReadLastLaunch()
	if err != nil {
		t.Fatalf("read second: %v", err)
	}
	if got.AppID != 900002 || got.Source != SourceSteam || got.Token != "" {
		t.Fatalf("second = %+v", got)
	}
}

func TestStoreLastLaunchMissingCorruptAndSymlink(t *testing.T) {
	store := sessionStore(t)
	if _, err := store.ReadLastLaunch(); !errors.Is(err, ErrLastLaunchMissing) {
		t.Fatalf("missing error = %v, want ErrLastLaunchMissing", err)
	}
	if err := os.WriteFile(store.LastLaunchPath(), []byte("{"), 0o600); err != nil {
		t.Fatalf("write corrupt: %v", err)
	}
	if _, err := store.ReadLastLaunch(); !errors.Is(err, ErrLastLaunchInvalid) {
		t.Fatalf("corrupt error = %v, want ErrLastLaunchInvalid", err)
	}
	if err := os.Remove(store.LastLaunchPath()); err != nil {
		t.Fatalf("remove corrupt: %v", err)
	}
	real := filepath.Join(t.TempDir(), "real.json")
	if err := os.WriteFile(real, []byte("{}"), 0o600); err != nil {
		t.Fatalf("write real: %v", err)
	}
	if err := os.Symlink(real, store.LastLaunchPath()); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	if _, err := store.ReadLastLaunch(); !errors.Is(err, ErrStore) {
		t.Fatalf("symlink error = %v, want ErrStore", err)
	}
}

func TestStoreIntentRoundTripAndExpiry(t *testing.T) {
	store := sessionStore(t)
	now := time.Now().UTC()
	intent := NewIntent(testToken, 900001, now)
	if err := store.WriteIntent(intent); err != nil {
		t.Fatalf("write intent: %v", err)
	}
	info, err := os.Stat(store.IntentPath())
	if err != nil {
		t.Fatalf("stat intent: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o, want 600", info.Mode().Perm())
	}
	got, err := store.ReadIntent(now)
	if err != nil {
		t.Fatalf("read intent: %v", err)
	}
	if got.Token != testToken || got.TargetAppID != 900001 || got.State != IntentArmed {
		t.Fatalf("intent = %+v", got)
	}
	if _, err := store.ReadIntent(intent.ExpiresAt); !errors.Is(err, ErrIntentExpired) {
		t.Fatalf("expired error = %v, want ErrIntentExpired", err)
	}
	if err := store.ClearIntent(); err != nil {
		t.Fatalf("clear intent: %v", err)
	}
	if _, err := store.ReadIntent(now); !errors.Is(err, ErrIntentMissing) {
		t.Fatalf("missing error = %v, want ErrIntentMissing", err)
	}
}

func TestStoreIntentRejectsCorruptAndSymlink(t *testing.T) {
	store := sessionStore(t)
	if err := os.WriteFile(store.IntentPath(), []byte("{"), 0o600); err != nil {
		t.Fatalf("write corrupt: %v", err)
	}
	if _, err := store.ReadIntent(time.Now()); !errors.Is(err, ErrIntentInvalid) {
		t.Fatalf("corrupt error = %v, want ErrIntentInvalid", err)
	}
	if err := os.Remove(store.IntentPath()); err != nil {
		t.Fatalf("remove corrupt: %v", err)
	}
	real := filepath.Join(t.TempDir(), "real.json")
	if err := os.WriteFile(real, []byte("{}"), 0o600); err != nil {
		t.Fatalf("write real: %v", err)
	}
	if err := os.Symlink(real, store.IntentPath()); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	if _, err := store.ReadIntent(time.Now()); !errors.Is(err, ErrStore) {
		t.Fatalf("symlink error = %v, want ErrStore", err)
	}
}
