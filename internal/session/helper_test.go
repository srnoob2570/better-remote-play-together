package session

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"

	"github.com/srnoob2570/better-rpt/internal/domain"
)

func helperOptions(store *Store, target uint32) HelperOptions {
	return HelperOptions{
		Store:       store,
		Token:       testToken,
		DonorAppID:  domain.SpaceWarAppID,
		TargetAppID: target,
		HelperPID:   os.Getpid(),
		HelperStart: helperStartForTest(),
		Now:         time.Now,
		Environ:     func() ([]string, error) { return []string{"SteamAppId=480", "SteamGameId=480"}, nil },
		SignalGroup: func(int, syscall.Signal) error { return nil },
		EarlyExit:   10 * time.Millisecond,
		GroupGrace:  30 * time.Millisecond,
	}
}

func helperStartForTest() uint64 {
	start, _ := ProcessStartTime(os.Getpid())
	return start
}

func commandBuilder(command *exec.Cmd) func(context.Context, domain.LaunchSpec, []string) (*exec.Cmd, error) {
	return func(context.Context, domain.LaunchSpec, []string) (*exec.Cmd, error) {
		return command, nil
	}
}

func TestHelperRunsTargetToCompletion(t *testing.T) {
	store := sessionStore(t)
	now := time.Now().UTC()
	writeTestPlan(t, store, 900001, now)
	options := helperOptions(&store, 900001)
	options.BuildCommand = commandBuilder(exec.Command("sleep", "0.08"))
	helper, err := NewHelper(options)
	if err != nil {
		t.Fatalf("new helper: %v", err)
	}
	state, err := helper.Run(context.Background())
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if state.Phase != domain.PhaseEnded || state.Outcome != "ok" {
		t.Fatalf("state = %+v", state)
	}
	if state.TargetPID == 0 || state.TargetStartTime == 0 {
		t.Fatalf("target identity missing: %+v", state)
	}
	if state.ExitCode == nil || *state.ExitCode != 0 {
		t.Fatalf("exit code = %v", state.ExitCode)
	}
	read, err := store.ReadState(testToken)
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	if read.Phase != domain.PhaseEnded {
		t.Fatalf("persisted state = %+v", read)
	}
}

func TestHelperPersistsTerminalStateBeforeGroupCleanup(t *testing.T) {
	store := sessionStore(t)
	now := time.Now().UTC()
	writeTestPlan(t, store, 900001, now)
	options := helperOptions(&store, 900001)
	options.EarlyExit = time.Millisecond
	options.BuildCommand = commandBuilder(exec.Command("sleep", "0.05"))
	cleanupStarted := make(chan struct{})
	releaseCleanup := make(chan struct{})
	options.WaitGroup = func(ctx context.Context, _ int, _ time.Duration) error {
		close(cleanupStarted)
		select {
		case <-releaseCleanup:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	helper, err := NewHelper(options)
	if err != nil {
		t.Fatalf("new helper: %v", err)
	}

	type result struct {
		state State
		err   error
	}
	done := make(chan result, 1)
	go func() {
		state, runErr := helper.Run(context.Background())
		done <- result{state: state, err: runErr}
	}()

	select {
	case <-cleanupStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("helper did not start process-group cleanup")
	}
	persisted, err := store.ReadState(testToken)
	if err != nil {
		t.Fatalf("read persisted state: %v", err)
	}
	if persisted.Phase != domain.PhaseEnded || persisted.Outcome != "ok" {
		t.Fatalf("state before cleanup = %+v", persisted)
	}
	locked, err := store.Locked()
	if err != nil {
		t.Fatalf("check lock: %v", err)
	}
	if !locked {
		t.Fatal("session lock released before process-group cleanup")
	}

	close(releaseCleanup)
	select {
	case got := <-done:
		if got.err != nil {
			t.Fatalf("run: %v", got.err)
		}
		if got.state.Phase != domain.PhaseEnded {
			t.Fatalf("final state = %+v", got.state)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("helper did not finish")
	}
}

func TestHelperEarlyExitIsError(t *testing.T) {
	store := sessionStore(t)
	now := time.Now().UTC()
	writeTestPlan(t, store, 900001, now)
	options := helperOptions(&store, 900001)
	options.EarlyExit = 200 * time.Millisecond
	options.BuildCommand = commandBuilder(exec.Command("sleep", "0.01"))
	helper, err := NewHelper(options)
	if err != nil {
		t.Fatalf("new helper: %v", err)
	}
	state, err := helper.Run(context.Background())
	if !errors.Is(err, ErrTargetExited) {
		t.Fatalf("error = %v, want ErrTargetExited", err)
	}
	if state.Phase != domain.PhaseError || state.ErrorCode != "target_exited_early" {
		t.Fatalf("state = %+v", state)
	}
}

func TestHelperMissingPlanWritesError(t *testing.T) {
	store := sessionStore(t)
	options := helperOptions(&store, 900001)
	options.BuildCommand = commandBuilder(exec.Command("true"))
	helper, err := NewHelper(options)
	if err != nil {
		t.Fatalf("new helper: %v", err)
	}
	state, err := helper.Run(context.Background())
	if !errors.Is(err, ErrPlanMissing) {
		t.Fatalf("error = %v, want ErrPlanMissing", err)
	}
	if state.Phase != domain.PhaseError || state.ErrorCode != "plan_unavailable" {
		t.Fatalf("state = %+v", state)
	}
	read, readErr := store.ReadState(testToken)
	if readErr != nil || read.Phase != domain.PhaseError {
		t.Fatalf("persisted state = %+v, err=%v", read, readErr)
	}
}

func TestHelperPlanMismatch(t *testing.T) {
	store := sessionStore(t)
	now := time.Now().UTC()
	plan := NewPlan(testToken, 900002, testSpec(900002), now)
	if err := store.WritePlan(plan); err != nil {
		t.Fatalf("write plan: %v", err)
	}
	options := helperOptions(&store, 900001)
	options.BuildCommand = commandBuilder(exec.Command("true"))
	helper, err := NewHelper(options)
	if err != nil {
		t.Fatalf("new helper: %v", err)
	}
	state, err := helper.Run(context.Background())
	if !errors.Is(err, ErrPlanInvalid) {
		t.Fatalf("error = %v, want ErrPlanInvalid", err)
	}
	if state.Phase != domain.PhaseError || state.ErrorCode != "plan_mismatch" {
		t.Fatalf("state = %+v", state)
	}
}

func TestHelperRefusesBusyLock(t *testing.T) {
	store := sessionStore(t)
	lock, err := store.TryLock()
	if err != nil {
		t.Fatalf("lock: %v", err)
	}
	defer func() { _ = lock.Close() }()
	options := helperOptions(&store, 900001)
	options.BuildCommand = commandBuilder(exec.Command("true"))
	helper, err := NewHelper(options)
	if err != nil {
		t.Fatalf("new helper: %v", err)
	}
	if _, err := helper.Run(context.Background()); !errors.Is(err, ErrSessionBusy) {
		t.Fatalf("error = %v, want ErrSessionBusy", err)
	}
}

func TestHelperRejectsInvalidEnvironment(t *testing.T) {
	store := sessionStore(t)
	options := helperOptions(&store, 900001)
	options.Environ = func() ([]string, error) { return []string{"SteamAppId=480"}, nil }
	helper, err := NewHelper(options)
	if err != nil {
		t.Fatalf("new helper: %v", err)
	}
	if _, err := helper.Run(context.Background()); !errors.Is(err, ErrEnvironment) {
		t.Fatalf("error = %v, want ErrEnvironment", err)
	}
}

func TestHelperInterruptStopsTarget(t *testing.T) {
	store := sessionStore(t)
	now := time.Now().UTC()
	writeTestPlan(t, store, 900001, now)
	command := exec.Command("sleep", "2")
	options := helperOptions(&store, 900001)
	options.BuildCommand = commandBuilder(command)
	options.SignalGroup = func(_ int, sig syscall.Signal) error {
		if command.Process == nil {
			return nil
		}
		return command.Process.Signal(sig)
	}
	options.GroupGrace = 100 * time.Millisecond
	helper, err := NewHelper(options)
	if err != nil {
		t.Fatalf("new helper: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()
	state, err := helper.Run(ctx)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if state.Phase != domain.PhaseEnded || state.Outcome != "interrupted" || state.ErrorCode != "helper_stopped" {
		t.Fatalf("state = %+v", state)
	}
}
