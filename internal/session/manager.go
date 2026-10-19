package session

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/srnoob2570/better-rpt/internal/config"
	"github.com/srnoob2570/better-rpt/internal/domain"
	"github.com/srnoob2570/better-rpt/internal/helper"
	"github.com/srnoob2570/better-rpt/internal/steam"
)

// PrepareFunc is the localconfig transaction; it is injectable for tests.
type PrepareFunc func(config.PrepareOptions) (config.Prepared, error)

// RestoreFunc is the inverse transaction; it is injectable for tests.
type RestoreFunc func(config.RestoreOptions) (config.Restored, error)

// LaunchFunc starts the donor AppID through Steam; it is injectable for tests.
type LaunchFunc func(ctx context.Context, appID uint32) error

// Manager orchestrates one donor/target session around the durable store. It
// does not contain UI confirmation; CloseSteam is the caller's decision.
type Manager struct {
	Paths          config.Paths
	Store          Store
	Catalog        steam.Catalog
	Lifecycle      *steam.Lifecycle
	Installer      *helper.Installer
	PrepareFn      PrepareFunc
	RestoreFn      RestoreFunc
	Launch         LaunchFunc
	Now            func() time.Time
	PollInterval   time.Duration
	StartupTimeout time.Duration
}

// StartOptions controls one session start.
type StartOptions struct {
	TargetAppID uint32
	CloseSteam  bool
	// OnState reports durable phases while Start waits. It is called from the
	// manager goroutine and must not touch UI directly.
	OnState func(State)
}

// Active is the result of a successful start, or the useful context after a
// failed start.
type Active struct {
	Token    string
	Plan     Plan
	Prepared config.Prepared
	Game     domain.Game
	State    State
}

// Recovery describes startup recovery.
type Recovery struct {
	State        *State
	HelperAlive  bool
	Stale        *config.Stale
	ExpiredPlans []string
}

// NewManager builds a manager for the detected catalog.
func NewManager(paths config.Paths, catalog steam.Catalog) (*Manager, error) {
	root := catalog.Installation.Root
	if root == "" {
		return nil, domain.ErrSteamNotFound
	}
	account, err := config.SelectAccount(root)
	if err != nil {
		return nil, err
	}
	lifecycle, err := steam.NewLifecycle(root, account.LocalConfigPath(root))
	if err != nil {
		return nil, err
	}
	return &Manager{
		Paths:          paths,
		Store:          NewStore(paths.DataDir),
		Catalog:        catalog,
		Lifecycle:      lifecycle,
		Installer:      helper.NewInstaller(paths),
		PrepareFn:      config.Prepare,
		RestoreFn:      config.Restore,
		Launch:         lifecycle.AppLaunch,
		PollInterval:   time.Second,
		StartupTimeout: StartupTimeout,
		Now:            time.Now,
	}, nil
}

// Start validates, prepares, launches the donor and waits for the helper to
// reach active (or fail). It restores the launch option if the donor launch or
// the helper fails terminally.
func (m *Manager) Start(ctx context.Context, options StartOptions) (Active, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if options.TargetAppID == 0 || options.TargetAppID == domain.SpaceWarAppID {
		return Active{}, fmt.Errorf("%w: invalid target %d", domain.ErrInvalidSession, options.TargetAppID)
	}
	game, ok := m.game(options.TargetAppID)
	if !ok || game.Launch == nil {
		return Active{}, fmt.Errorf("%w: target %d is not installed", domain.ErrInvalidLaunchSpec, options.TargetAppID)
	}
	switch game.Status {
	case domain.StatusIncompatible, domain.StatusMissing, domain.StatusError, domain.StatusActive:
		return Active{}, fmt.Errorf("%w: target %d is not launchable: %v", domain.ErrInvalidLaunchSpec, game.AppID, game.Problem)
	}
	donor, ok := m.game(domain.SpaceWarAppID)
	if !ok {
		return Active{}, domain.ErrSpaceWarMissing
	}
	switch donor.Status {
	case domain.StatusIncompatible, domain.StatusMissing, domain.StatusError:
		return Active{}, fmt.Errorf("%w: %v", domain.ErrSpaceWarMissing, donor.Problem)
	}
	if m.Store.Dir == "" {
		m.Store = NewStore(m.Paths.DataDir)
	}
	if err := m.Store.Ensure(); err != nil {
		return Active{}, err
	}
	if active, err := m.Store.ActiveState(); err != nil {
		return Active{}, err
	} else if active != nil {
		return Active{}, fmt.Errorf("%w: %s", ErrSessionBusy, active.Token)
	}
	if m.Installer == nil {
		m.Installer = helper.NewInstaller(m.Paths)
	}
	if _, err := m.Installer.Install(false); err != nil {
		return Active{}, err
	}
	if m.Lifecycle == nil {
		return Active{}, domain.ErrSteamNotFound
	}
	if m.Launch == nil {
		return Active{}, fmt.Errorf("%w: no donor launcher", ErrStore)
	}
	running, err := m.Lifecycle.IsRunning()
	if err != nil {
		return Active{}, err
	}
	steamWasRunning := false
	if running {
		if !options.CloseSteam {
			return Active{}, fmt.Errorf("%w: confirmation required", domain.ErrSteamRunning)
		}
		if err := m.Lifecycle.Shutdown(ctx); err != nil {
			return Active{}, err
		}
		steamWasRunning = true
	}

	token, err := NewToken()
	if err != nil {
		return Active{}, err
	}
	now := m.now()
	plan := NewPlan(token, options.TargetAppID, *game.Launch, now)
	if err := m.Store.WritePlan(plan); err != nil {
		return Active{}, err
	}
	preparing := NewState(plan, domain.PhasePreparing, now)
	if err := m.Store.WriteState(preparing); err != nil {
		_ = m.Store.RemovePlan(token)
		relaunchErr := m.relaunchIfNeeded(ctx, steamWasRunning)
		if relaunchErr != nil {
			return Active{Token: token, Plan: plan, Game: game}, fmt.Errorf("%w; Steam relaunch failed: %v", err, relaunchErr)
		}
		return Active{Token: token, Plan: plan, Game: game}, err
	}
	if options.OnState != nil {
		options.OnState(preparing)
	}
	if m.PrepareFn == nil {
		m.PrepareFn = config.Prepare
	}
	prepared, err := m.PrepareFn(config.PrepareOptions{
		Paths:        m.Paths,
		SteamRoot:    m.Catalog.Installation.Root,
		HelperPath:   m.Paths.HelperPath(),
		TargetAppID:  options.TargetAppID,
		SessionToken: token,
		Now:          now,
		SteamRunning: m.steamGuard,
	})
	if err != nil {
		_ = m.Store.RemovePlan(token)
		_ = m.Store.RemoveState(token)
		relaunchErr := m.relaunchIfNeeded(ctx, steamWasRunning)
		if relaunchErr != nil {
			return Active{Token: token, Plan: plan, Game: game}, fmt.Errorf("%w; Steam relaunch failed: %v", err, relaunchErr)
		}
		return Active{Token: token, Plan: plan, Game: game}, err
	}
	result := Active{Token: token, Plan: plan, Prepared: prepared, Game: game}
	if err := ctx.Err(); err != nil {
		cleanupErr := m.restore(ctx, token, false, steamWasRunning)
		if cleanupErr == nil {
			_ = m.Store.RemovePlan(token)
			_ = m.Store.RemoveState(token)
		}
		if cleanupErr != nil {
			return result, fmt.Errorf("%w; restore failed: %v", err, cleanupErr)
		}
		return result, err
	}
	if err := m.Launch(ctx, domain.SpaceWarAppID); err != nil {
		cleanupErr := m.restore(ctx, token, false, steamWasRunning)
		if cleanupErr == nil {
			_ = m.Store.RemovePlan(token)
			_ = m.Store.RemoveState(token)
		}
		if cleanupErr != nil {
			return result, fmt.Errorf("%w; restore failed: %v", err, cleanupErr)
		}
		return result, err
	}

	deadline := now.Add(m.startupTimeout())
	for {
		state, readErr := m.Store.ReadState(token)
		if readErr == nil {
			result.State = state
			if options.OnState != nil {
				options.OnState(state)
			}
			switch state.Phase {
			case domain.PhaseActive:
				return result, nil
			case domain.PhaseError, domain.PhaseEnded:
				restoreErr := m.restore(ctx, token, false, steamWasRunning)
				if restoreErr == nil {
					_ = m.Store.RemovePlan(token)
				}
				if restoreErr != nil {
					return result, fmt.Errorf("%w; restore failed: %v", m.stateError(state), restoreErr)
				}
				return result, m.stateError(state)
			}
		} else if !errors.Is(readErr, ErrStateMissing) {
			_ = m.markStateError(token, "state_unreadable", readErr)
			restoreErr := m.restore(ctx, token, false, steamWasRunning)
			if restoreErr == nil {
				_ = m.Store.RemovePlan(token)
				if _, err := m.Store.ReadState(token); err != nil {
					_ = m.Store.RemoveState(token)
				}
			}
			if restoreErr != nil {
				return result, fmt.Errorf("%w; restore failed: %v", readErr, restoreErr)
			}
			return result, readErr
		}
		if !m.now().Before(deadline) {
			err := fmt.Errorf("%w: helper did not reach active", ErrTimeout)
			_ = m.markStateError(token, "startup_timeout", err)
			restoreErr := m.restore(ctx, token, false, steamWasRunning)
			if restoreErr == nil {
				_ = m.Store.RemovePlan(token)
			}
			if restoreErr != nil {
				return result, fmt.Errorf("%w; restore failed: %v", err, restoreErr)
			}
			return result, err
		}
		if err := sleepContext(ctx, m.pollInterval()); err != nil {
			_ = m.markStateError(token, "canceled", err)
			restoreErr := m.restore(ctx, token, false, steamWasRunning)
			if restoreErr == nil {
				_ = m.Store.RemovePlan(token)
			}
			if restoreErr != nil {
				return result, fmt.Errorf("%w; restore failed: %v", err, restoreErr)
			}
			return result, err
		}
	}
}

// Restore closes a session's launch-option transaction. It refuses while the
// helper is still alive in an active phase.
func (m *Manager) Restore(ctx context.Context, token string) error {
	state, err := m.Store.ReadState(token)
	if err != nil && !errors.Is(err, ErrStateMissing) {
		return err
	}
	active := false
	if err == nil && state.HelperPID > 0 && !state.Terminal() {
		active = ProcessAlive(state.HelperPID, state.HelperStartTime)
	}
	if active {
		return ErrSessionBusy
	}
	m.markRestoring(token)
	if err := m.restore(ctx, token, active, false); err != nil {
		return err
	}
	if err := m.Store.RemovePlan(token); err != nil {
		return err
	}
	return m.Store.RemoveState(token)
}

// Recover inspects startup state: a live helper is active; a dead active
// helper is terminalized; otherwise a stale launch option is reported.
func (m *Manager) Recover(ctx context.Context) (*Recovery, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	active, err := m.Store.ActiveState()
	if err != nil {
		return nil, err
	}
	if active != nil {
		alive := active.HelperPID > 0 && ProcessAlive(active.HelperPID, active.HelperStartTime)
		if alive {
			return &Recovery{State: active, HelperAlive: true}, nil
		}
		active.Phase = domain.PhaseError
		active.Outcome = "error"
		active.ErrorCode = "helper_dead"
		active.Error = "the session helper is no longer running"
		active.EndedAt = m.now()
		active.UpdatedAt = active.EndedAt
		if err := m.Store.WriteState(*active); err != nil {
			return nil, err
		}
		return &Recovery{State: active}, nil
	}
	expired, err := m.Store.RemoveExpiredPlans(m.now())
	if err != nil {
		return nil, err
	}
	root := m.Catalog.Installation.Root
	if root != "" {
		stale, err := config.DetectStale(m.Paths, root, false)
		if err != nil {
			return nil, err
		}
		if stale != nil {
			return &Recovery{Stale: stale, ExpiredPlans: expired}, nil
		}
	}
	if len(expired) > 0 {
		return &Recovery{ExpiredPlans: expired}, nil
	}
	return nil, nil
}

func (m *Manager) restore(ctx context.Context, token string, active, relaunch bool) (err error) {
	defer func() {
		if err != nil {
			_ = m.markStateError(token, "restore_failed", err)
		}
	}()
	if m.RestoreFn == nil {
		m.RestoreFn = config.Restore
	}
	cleanupCtx, cancel := m.cleanupContext(ctx)
	defer cancel()

	wasRunning := false
	if m.Lifecycle != nil {
		running, err := m.Lifecycle.IsRunning()
		if err != nil {
			return err
		}
		if running {
			wasRunning = true
			relaunch = true
			if err := m.Lifecycle.Shutdown(cleanupCtx); err != nil {
				_ = m.Lifecycle.Relaunch(cleanupCtx)
				return err
			}
		}
	}
	_, restoreErr := m.RestoreFn(config.RestoreOptions{
		Paths:         m.Paths,
		SteamRoot:     m.Catalog.Installation.Root,
		SessionToken:  token,
		ActiveSession: active,
		SteamRunning:  m.steamGuard,
	})

	var relaunchErr error
	if relaunch || wasRunning {
		if m.Lifecycle == nil {
			relaunchErr = domain.ErrSteamNotFound
		} else {
			relaunchErr = m.Lifecycle.Relaunch(cleanupCtx)
		}
	}
	switch {
	case restoreErr != nil && relaunchErr != nil:
		return fmt.Errorf("%w; Steam relaunch failed: %v", restoreErr, relaunchErr)
	case restoreErr != nil:
		return restoreErr
	case relaunchErr != nil:
		return fmt.Errorf("launch option restored but Steam did not relaunch: %w", relaunchErr)
	default:
		return nil
	}
}

func (m *Manager) markRestoring(token string) {
	state, err := m.Store.ReadState(token)
	if err != nil {
		return
	}
	state.Phase = domain.PhaseRestoring
	state.UpdatedAt = m.now()
	_ = m.Store.WriteState(state)
}

func (m *Manager) markStateError(token, code string, cause error) error {
	state, err := m.Store.ReadState(token)
	if err != nil {
		return err
	}
	now := m.now()
	state.Phase = domain.PhaseError
	state.Outcome = "error"
	state.ErrorCode = code
	if cause != nil {
		state.Error = cause.Error()
	}
	state.EndedAt = now
	state.UpdatedAt = now
	return m.Store.WriteState(state)
}

func (m *Manager) relaunchIfNeeded(ctx context.Context, needed bool) error {
	if !needed {
		return nil
	}
	if m.Lifecycle == nil {
		return domain.ErrSteamNotFound
	}
	cleanupCtx, cancel := m.cleanupContext(ctx)
	defer cancel()
	return m.Lifecycle.Relaunch(cleanupCtx)
}

func (m *Manager) cleanupContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithTimeout(context.WithoutCancel(ctx), 2*time.Minute)
}

func (m *Manager) steamGuard() (bool, error) {
	if m.Lifecycle == nil {
		return false, nil
	}
	return m.Lifecycle.IsRunning()
}

func (m *Manager) game(appID uint32) (domain.Game, bool) {
	for _, game := range m.Catalog.Games {
		if game.AppID == appID {
			return game, true
		}
	}
	return domain.Game{}, false
}

func (m *Manager) now() time.Time {
	if m.Now != nil {
		return m.Now()
	}
	return time.Now()
}

func (m *Manager) pollInterval() time.Duration {
	if m.PollInterval > 0 {
		return m.PollInterval
	}
	return time.Second
}

func (m *Manager) startupTimeout() time.Duration {
	if m.StartupTimeout > 0 {
		return m.StartupTimeout
	}
	return StartupTimeout
}

func sleepContext(ctx context.Context, duration time.Duration) error {
	if duration <= 0 {
		duration = time.Millisecond
	}
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (m *Manager) stateError(state State) error {
	if state.ErrorCode == "target_exited_early" {
		return fmt.Errorf("%w: %s", ErrTargetExited, state.Error)
	}
	if state.Error != "" {
		return fmt.Errorf("%w: %s", ErrTargetExited, state.Error)
	}
	return fmt.Errorf("%w: target ended", ErrTargetExited)
}
