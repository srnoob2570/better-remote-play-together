package session

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// SelfEnvironment reads the helper's inherited environment from /proc/self/environ.
func SelfEnvironment() ([]string, error) {
	data, err := os.ReadFile("/proc/self/environ")
	if err != nil {
		return nil, fmt.Errorf("%w: read self environ: %v", ErrEnvironment, err)
	}
	raw := bytes.Split(data, []byte{0})
	env := make([]string, 0, len(raw))
	for _, entry := range raw {
		if len(entry) == 0 {
			continue
		}
		env = append(env, string(entry))
	}
	return env, nil
}

// DonorEnvironment verifies that Steam launched the helper as AppID 480.
func DonorEnvironment(env []string, donorAppID uint32) error {
	want := strconv.FormatUint(uint64(donorAppID), 10)
	for _, key := range []string{"SteamAppId", "SteamGameId"} {
		value, ok := EnvironmentValue(env, key)
		if !ok || value != want {
			return fmt.Errorf("%w: %s=%q, want %s", ErrEnvironment, key, value, want)
		}
	}
	return nil
}

// EnvironmentValue returns a variable from a KEY=VALUE slice.
func EnvironmentValue(env []string, key string) (string, bool) {
	prefix := key + "="
	for i := len(env) - 1; i >= 0; i-- {
		if strings.HasPrefix(env[i], prefix) {
			return strings.TrimPrefix(env[i], prefix), true
		}
	}
	return "", false
}

// ProcessStartTime returns the kernel starttime (clock ticks) for pid.
func ProcessStartTime(pid int) (uint64, error) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return 0, fmt.Errorf("read /proc/%d/stat: %w", pid, err)
	}
	closeParen := bytes.LastIndexByte(data, ')')
	if closeParen < 0 || closeParen+2 >= len(data) {
		return 0, fmt.Errorf("malformed /proc/%d/stat", pid)
	}
	fields := strings.Fields(string(data[closeParen+2:]))
	// fields[0] is state (field 3); starttime is field 22.
	if len(fields) < 20 {
		return 0, fmt.Errorf("short /proc/%d/stat", pid)
	}
	value, err := strconv.ParseUint(fields[19], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse /proc/%d/stat starttime: %w", pid, err)
	}
	return value, nil
}

// ProcessAlive reports whether pid still exists with the same starttime and is
// not a zombie.
func ProcessAlive(pid int, startTime uint64) bool {
	if pid <= 0 {
		return false
	}
	current, err := ProcessStartTime(pid)
	if err != nil || current != startTime {
		return false
	}
	state, err := processState(pid)
	if err != nil {
		return false
	}
	return state != "Z" && state != "X"
}

// ProcessGroupAlive reports whether any process remains in pgid.
func ProcessGroupAlive(pgid int) bool {
	if pgid <= 0 {
		return false
	}
	err := syscall.Kill(-pgid, 0)
	if err == nil {
		return true
	}
	return errors.Is(err, syscall.EPERM)
}

// SignalProcessGroup sends sig to the target process group.
func SignalProcessGroup(pgid int, sig syscall.Signal) error {
	if pgid <= 0 {
		return fmt.Errorf("%w: invalid process group %d", ErrStateInvalid, pgid)
	}
	if err := syscall.Kill(-pgid, sig); err != nil && !errors.Is(err, syscall.ESRCH) {
		return fmt.Errorf("signal process group %d: %w", pgid, err)
	}
	return nil
}

// WaitProcessGroupGone waits up to timeout for the group to disappear.
func WaitProcessGroupGone(ctx context.Context, pgid int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for ProcessGroupAlive(pgid) {
		if !time.Now().Before(deadline) {
			return fmt.Errorf("%w: process group %d still alive", ErrTimeout, pgid)
		}
		timer := time.NewTimer(100 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	return nil
}

func processState(pid int) (string, error) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return "", err
	}
	closeParen := bytes.LastIndexByte(data, ')')
	if closeParen < 0 || closeParen+2 >= len(data) {
		return "", fmt.Errorf("malformed /proc/%d/stat", pid)
	}
	fields := strings.Fields(string(data[closeParen+2:]))
	if len(fields) == 0 {
		return "", fmt.Errorf("short /proc/%d/stat", pid)
	}
	return fields[0], nil
}
