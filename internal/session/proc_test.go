package session

import (
	"context"
	"errors"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/srnoob2570/better-rpt/internal/domain"
)

func TestProcessStartTimeAndAlive(t *testing.T) {
	pid := os.Getpid()
	start, err := ProcessStartTime(pid)
	if err != nil || start == 0 {
		t.Fatalf("start = %d, err=%v", start, err)
	}
	if !ProcessAlive(pid, start) {
		t.Fatal("current process is not alive")
	}
	if ProcessAlive(pid, start+1) {
		t.Fatal("wrong start time reported alive")
	}
	if ProcessAlive(999999, 1) {
		t.Fatal("invalid pid reported alive")
	}
	if _, err := ProcessStartTime(999999); err == nil {
		t.Fatal("invalid pid returned a start time")
	}
}

func TestDonorEnvironment(t *testing.T) {
	good := []string{"SteamAppId=480", "SteamGameId=480", "DISPLAY=:0"}
	if err := DonorEnvironment(good, 480); err != nil {
		t.Fatalf("good env: %v", err)
	}
	if err := DonorEnvironment([]string{"SteamAppId=480"}, 480); !errors.Is(err, ErrEnvironment) {
		t.Fatalf("missing game id error = %v", err)
	}
	if err := DonorEnvironment([]string{"SteamAppId=480", "SteamGameId=900001"}, 480); !errors.Is(err, ErrEnvironment) {
		t.Fatalf("wrong game id error = %v", err)
	}
	if value, ok := EnvironmentValue(good, "SteamAppId"); !ok || value != "480" {
		t.Fatalf("EnvironmentValue = %q, %v", value, ok)
	}
	if _, ok := EnvironmentValue(good, "MISSING"); ok {
		t.Fatal("missing key found")
	}
}

func TestSelfEnvironment(t *testing.T) {
	env, err := SelfEnvironment()
	if err != nil {
		t.Fatalf("self environment: %v", err)
	}
	if len(env) == 0 {
		t.Fatal("empty environment")
	}
}

func TestProcessGroupHelpers(t *testing.T) {
	pgid := syscall.Getpgrp()
	if !ProcessGroupAlive(pgid) {
		t.Fatal("own process group is not alive")
	}
	if ProcessGroupAlive(999999) {
		t.Fatal("invalid process group reported alive")
	}
	if err := SignalProcessGroup(999999, syscall.SIGTERM); err != nil {
		t.Fatalf("signal invalid group: %v", err)
	}
	if err := WaitProcessGroupGone(context.Background(), 999999, 50*time.Millisecond); err != nil {
		t.Fatalf("wait invalid group: %v", err)
	}
}

func TestTargetState(t *testing.T) {
	pid := os.Getpid()
	start, err := ProcessStartTime(pid)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	phase, err := TargetState(pid, start)
	if err != nil {
		t.Fatalf("target state: %v", err)
	}
	if phase != domain.PhaseActive {
		t.Fatalf("phase = %q, want active", phase)
	}
	phase, err = TargetState(pid, start+1)
	if err != nil {
		t.Fatalf("stale target state: %v", err)
	}
	if phase != domain.PhaseEnded {
		t.Fatalf("stale phase = %q, want ended", phase)
	}
}
