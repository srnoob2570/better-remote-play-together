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
)

// DispatcherOptions configures the persistent SpaceWar wrapper.
type DispatcherOptions struct {
	HelperOptions
	// Fallback is the original game command after `--`. It runs when there is
	// no armed Better RPT intent, so SpaceWar keeps working outside sessions.
	Fallback []string
	// RunFallback is injectable for tests.
	RunFallback func(context.Context, []string, []string) error
}

// Dispatcher runs as SpaceWar's launch option. It consumes an armed intent and
// launches the target, or executes the original game command as a fallback.
type Dispatcher struct {
	options        HelperOptions
	now            func() time.Time
	fallback       []string
	fallbackRunner func(context.Context, []string, []string) error
}

// NewDispatcher validates the dispatcher wrapper arguments.
func NewDispatcher(options DispatcherOptions) (*Dispatcher, error) {
	if err := normalizeHelperOptions(&options.HelperOptions); err != nil {
		return nil, err
	}
	return &Dispatcher{
		options:        options.HelperOptions,
		now:            options.Now,
		fallback:       append([]string(nil), options.Fallback...),
		fallbackRunner: options.RunFallback,
	}, nil
}

// Run reads the durable intent. A valid armed intent is consumed and its plan
// is executed; anything else runs the original game command.
func (d *Dispatcher) Run(ctx context.Context) (State, error) {
	env, err := d.options.Environ()
	if err != nil {
		return State{}, err
	}
	if err := DonorEnvironment(env, d.options.DonorAppID); err != nil {
		return State{}, err
	}
	now := d.now().UTC()
	intent, err := d.options.Store.ReadIntent(now)
	if errors.Is(err, ErrIntentExpired) {
		_ = d.options.Store.ClearIntent()
		return d.runFallback(ctx, env)
	}
	if errors.Is(err, ErrIntentMissing) {
		return d.runFallback(ctx, env)
	}
	if err != nil {
		// A corrupt or unreadable intent must not prevent SpaceWar from
		// starting. Clear it and fall back to the original game command.
		_ = d.options.Store.ClearIntent()
		state, fallbackErr := d.runFallback(ctx, env)
		if fallbackErr != nil && errors.Is(fallbackErr, ErrIntentMissing) {
			return State{}, err
		}
		return state, fallbackErr
	}
	if intent.State != IntentArmed {
		return d.runFallback(ctx, env)
	}
	plan, err := d.options.Store.ReadPlan(intent.Token, now)
	if err != nil {
		_ = d.options.Store.ClearIntent()
		return State{}, err
	}
	if plan.DonorAppID != d.options.DonorAppID || plan.TargetAppID != intent.TargetAppID ||
		plan.Token != intent.Token {
		_ = d.options.Store.ClearIntent()
		return State{}, fmt.Errorf("%w: intent and plan do not match", ErrPlanInvalid)
	}
	intent.State = IntentConsumed
	if err := d.options.Store.WriteIntent(*intent); err != nil {
		return State{}, err
	}
	helper := &Helper{
		options: d.options,
		now:     d.now,
	}
	helper.options.Token = intent.Token
	helper.options.TargetAppID = intent.TargetAppID
	return helper.Run(ctx)
}

func (d *Dispatcher) runFallback(ctx context.Context, env []string) (State, error) {
	if len(d.fallback) == 0 {
		return State{}, fmt.Errorf("%w: no armed intent and no game command", ErrIntentMissing)
	}
	if d.fallbackRunner != nil {
		if err := d.fallbackRunner(ctx, d.fallback, env); err != nil {
			return State{}, err
		}
		return State{}, nil
	}
	if err := runFallbackCommand(ctx, d.fallback, env); err != nil {
		return State{}, err
	}
	return State{}, nil
}

func runFallbackCommand(ctx context.Context, args, env []string) error {
	if len(args) == 0 || args[0] == "" {
		return fmt.Errorf("%w: empty game command", domain.ErrInvalidLaunchSpec)
	}
	command := exec.CommandContext(ctx, args[0], args[1:]...)
	command.Env = env
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := command.Start(); err != nil {
		return fmt.Errorf("start game command: %w", err)
	}
	err := command.Wait()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return fmt.Errorf("game command exited with %d", exitErr.ExitCode())
		}
		return fmt.Errorf("wait for game command: %w", err)
	}
	return nil
}
