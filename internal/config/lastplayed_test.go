package config

import (
	"testing"
	"time"
)

func TestReadLastPlayedReturnsNewestNonDonor(t *testing.T) {
	root := transactionRoot(t, "localconfig_lastplayed.vdf")
	got, err := ReadLastPlayed(root)
	if err != nil {
		t.Fatalf("read last played: %v", err)
	}
	if got == nil {
		t.Fatal("last played is nil")
	}
	if got.AppID != 900002 {
		t.Fatalf("app = %d, want 900002", got.AppID)
	}
	want := time.Unix(1700000200, 0).UTC()
	if !got.At.Equal(want) {
		t.Fatalf("time = %s, want %s", got.At, want)
	}
}

func TestReadLastPlayedReturnsNilWithoutEntries(t *testing.T) {
	root := transactionRoot(t, "localconfig.vdf")
	got, err := ReadLastPlayed(root)
	if err != nil {
		t.Fatalf("read last played: %v", err)
	}
	if got != nil {
		t.Fatalf("last played = %+v, want nil", got)
	}
}
