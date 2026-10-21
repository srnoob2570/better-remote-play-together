package session

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/srnoob2570/better-rpt/internal/domain"
	"github.com/srnoob2570/better-rpt/internal/launch"
)

// HelperOptions are injectable for tests; zero values use the real process and
// command implementations.
type HelperOptions struct {
	Store        *Store
	Token        string
	DonorAppID   uint32
	TargetAppID  uint32
	HelperPID    int
	HelperStart  uint64
	Now          func() time.Time
	Environ      func() ([]string, error)
	BuildCommand func(context.Context, domain.LaunchSpec, []string) (*exec.Cmd, error)
	SignalGroup  func(int, syscall.Signal) error
	EarlyExit    time.Duration
	GroupGrace   time.Duration
}

// Helper executes one plan as the SpaceWar donor process.
type Helper struct {
	options HelperOptions
	now     func() time.Time
}

// NewHelper validates the launch arguments and returns an executor.
func NewHelper(options HelperOptions) (*Helper, error) {
	if err := normalizeHelperOptions(&options); err != nil {
		return nil, err
	}
	if options.TargetAppID == 0 || options.TargetAppID == domain.SpaceWarAppID {
		return nil, fmt.Errorf("%w: invalid target %d", ErrPlanInvalid, options.TargetAppID)
	}
	if !domain.ValidSessionToken(options.Token) {
		return nil, fmt.Errorf("%w: invalid session token", ErrPlanInvalid)
	}
	return &Helper{options: options, now: options.Now}, nil
}

func normalizeHelperOptions(options *HelperOptions) error {
	if options.Store == nil {
		return fmt.Errorf("%w: nil session store", ErrStore)
	}
	if options.DonorAppID == 0 {
		options.DonorAppID = domain.SpaceWarAppID
	}
	if options.DonorAppID != domain.SpaceWarAppID {
		return fmt.Errorf("%w: donor must be 480", ErrEnvironment)
	}
	if options.HelperPID == 0 {
		options.HelperPID = os.Getpid()
	}
	if options.HelperStart == 0 {
		start, err := ProcessStartTime(options.HelperPID)
		if err != nil {
			return fmt.Errorf("helper start time: %w", err)
		}
		options.HelperStart = start
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	if options.Environ == nil {
		options.Environ = SelfEnvironment
	}
	if options.BuildCommand == nil {
		options.BuildCommand = launch.BuildCommand
	}
	if options.SignalGroup == nil {
		options.SignalGroup = SignalProcessGroup
	}
	if options.EarlyExit <= 0 {
		options.EarlyExit = EarlyExit
	}
	if options.GroupGrace <= 0 {
		options.GroupGrace = GroupGrace
	}
	return nil
}

// Run validates the inherited Steam environment, takes the session lock, reads
// the plan, launches the target and records every state transition.
func (h *Helper) Run(ctx context.Context) (State, error) {
	env, err := h.options.Environ()
	if err != nil {
		return State{}, err
	}
	if err := DonorEnvironment(env, h.options.DonorAppID); err != nil {
		return State{}, err
	}
	lock, err := h.options.Store.TryLock()
	if err != nil {
		return State{}, err
	}
	defer lock.Close()

	now := h.now().UTC()
	plan, err := h.options.Store.ReadPlan(h.options.Token, now)
	if err != nil {
		state := h.newState(nil, domain.PhaseError, now)
		return h.finishError(state, "plan_unavailable", err)
	}
	if plan.DonorAppID != h.options.DonorAppID || plan.TargetAppID != h.options.TargetAppID {
		state := h.newState(&plan, domain.PhaseError, now)
		return h.finishError(state, "plan_mismatch", fmt.Errorf("%w: token/plan does not match arguments", ErrPlanInvalid))
	}
	return h.runPlan(ctx, plan, env)
}

func (h *Helper) runPlan(ctx context.Context, plan Plan, env []string) (State, error) {
	now := h.now().UTC()
	state := h.newState(&plan, domain.PhaseDonorStarting, now)
	if err := h.options.Store.WriteState(state); err != nil {
		return state, err
	}

	command, err := h.options.BuildCommand(ctx, plan.Launch, env)
	if err != nil {
		return h.finishError(state, "command_invalid", err)
	}
	state.Phase = domain.PhaseTargetStarting
	state.UpdatedAt = h.now().UTC()
	if err := h.options.Store.WriteState(state); err != nil {
		return state, err
	}
	if err := command.Start(); err != nil {
		return h.finishError(state, "target_start_failed", fmt.Errorf("start target: %w", err))
	}
	targetPID := command.Process.Pid
	startTime, err := waitStartTime(targetPID, h.now)
	if err != nil {
		_ = command.Process.Kill()
		return h.finishError(state, "target_starttime_failed", err)
	}
	startedAt := h.now().UTC()
	state.Phase = domain.PhaseActive
	state.TargetPID = targetPID
	state.TargetStartTime = startTime
	state.StartedAt = startedAt
	state.UpdatedAt = startedAt
	if err := h.options.Store.WriteState(state); err != nil {
		_ = command.Process.Kill()
		return state, err
	}

	waitCh := make(chan error, 1)
	go func() { waitCh <- command.Wait() }()
	var waitErr error
	interrupted := false
	select {
	case waitErr = <-waitCh:
	case <-ctx.Done():
		interrupted = true
		_ = h.options.SignalGroup(targetPID, syscall.SIGTERM)
		select {
		case waitErr = <-waitCh:
		case <-time.After(h.options.GroupGrace):
			_ = h.options.SignalGroup(targetPID, syscall.SIGKILL)
			select {
			case waitErr = <-waitCh:
			case <-time.After(h.options.GroupGrace):
				return h.finishError(state, "target_wait_failed", fmt.Errorf("%w: target did not exit", ErrTimeout))
			}
		}
	}

	elapsed := h.now().Sub(startedAt)
	_ = WaitProcessGroupGone(context.Background(), targetPID, h.options.GroupGrace)
	exitCode := exitCode(waitErr)
	state.ExitCode = exitCode
	state.UpdatedAt = h.now().UTC()
	state.EndedAt = state.UpdatedAt
	if interrupted {
		state.Phase = domain.PhaseEnded
		state.Outcome = "interrupted"
		state.ErrorCode = "helper_stopped"
		state.Error = "helper stopped by signal"
		if err := h.options.Store.WriteState(state); err != nil {
			return state, err
		}
		return state, nil
	}
	if elapsed < h.options.EarlyExit {
		state.Phase = domain.PhaseError
		state.Outcome = "error"
		state.ErrorCode = "target_exited_early"
		state.Error = fmt.Sprintf("target exited after %s", elapsed.Round(time.Millisecond))
		if err := h.options.Store.WriteState(state); err != nil {
			return state, err
		}
		return state, fmt.Errorf("%w: after %s", ErrTargetExited, elapsed.Round(time.Millisecond))
	}
	if waitErr != nil {
		state.Phase = domain.PhaseError
		state.Outcome = "error"
		state.ErrorCode = "target_exited"
		state.Error = waitErr.Error()
		if err := h.options.Store.WriteState(state); err != nil {
			return state, err
		}
		return state, fmt.Errorf("%w: %v", ErrTargetExited, waitErr)
	}
	state.Phase = domain.PhaseEnded
	state.Outcome = "ok"
	if err := h.options.Store.WriteState(state); err != nil {
		return state, err
	}
	return state, nil
}

func (h *Helper) newState(plan *Plan, phase domain.Phase, now time.Time) State {
	state := State{
		Version:         StateVersion,
		Token:           h.options.Token,
		DonorAppID:      h.options.DonorAppID,
		TargetAppID:     h.options.TargetAppID,
		Phase:           phase,
		HelperPID:       h.options.HelperPID,
		HelperStartTime: h.options.HelperStart,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if plan != nil {
		state.Token = plan.Token
		state.DonorAppID = plan.DonorAppID
		state.TargetAppID = plan.TargetAppID
	}
	return state
}

func (h *Helper) finishError(state State, code string, err error) (State, error) {
	state.Phase = domain.PhaseError
	state.Outcome = "error"
	state.ErrorCode = code
	state.Error = err.Error()
	state.UpdatedAt = h.now().UTC()
	state.EndedAt = state.UpdatedAt
	if writeErr := h.options.Store.WriteState(state); writeErr != nil {
		return state, writeErr
	}
	return state, err
}

func waitStartTime(pid int, now func() time.Time) (uint64, error) {
	deadline := now().Add(time.Second)
	for {
		start, err := ProcessStartTime(pid)
		if err == nil {
			return start, nil
		}
		if !now().Before(deadline) {
			return 0, fmt.Errorf("target start time: %w", err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func exitCode(err error) *int {
	if err == nil {
		code := 0
		return &code
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		code := exitErr.ExitCode()
		return &code
	}
	return nil
}
