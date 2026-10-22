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

// RestoreFunc is the legacy localconfig transaction; it is injectable for
// tests that exercise migration and manual restore paths.
type RestoreFunc func(config.RestoreOptions) (config.Restored, error)

// SetupDetectFunc reads the persistent dispatcher state.
type SetupDetectFunc func(config.Paths, config.LocalState) (config.SetupSnapshot, error)

// SetupInstallFunc installs the persistent dispatcher.
type SetupInstallFunc func(config.InstallSetupOptions) (config.Setup, error)

// SetupGuideFunc records a manual dispatcher installation.
type SetupGuideFunc func(config.GuideSetupOptions) (config.Setup, error)

// SetupRestoreFunc restores the persistent dispatcher.
type SetupRestoreFunc func(config.RestoreSetupOptions) (config.Restored, error)

// LaunchSessionFunc asks Steam to start the donor with a session token.
type LaunchSessionFunc func(ctx context.Context, appID uint32, token string) error

// Manager orchestrates launches around the durable store. It never closes or
// relaunches Steam: the dispatcher is installed once and each launch is a
// Steam URL request while Steam may remain open.
type Manager struct {
	Paths          config.Paths
	Store          Store
	Catalog        steam.Catalog
	Lifecycle      *steam.Lifecycle
	Installer      *helper.Installer
	RestoreFn      RestoreFunc
	SetupDetect    SetupDetectFunc
	LocalState     func(string) (config.LocalState, error)
	SetupInstall   SetupInstallFunc
	SetupGuide     SetupGuideFunc
	SetupRestore   SetupRestoreFunc
	LaunchSession  LaunchSessionFunc
	Now            func() time.Time
	PollInterval   time.Duration
	StartupTimeout time.Duration
}

// StartOptions controls one session start.
type StartOptions struct {
	TargetAppID uint32
	// OnState reports durable phases while Start waits. It is called from the
	// manager goroutine and must not touch UI directly.
	OnState func(State)
}

// Active is the result of a successful start or a failed launch attempt.
type Active struct {
	Token string
	Plan  Plan
	Game  domain.Game
	State State
	Last  *LastLaunch
}

// Recovery describes startup recovery.
type Recovery struct {
	State        *State
	HelperAlive  bool
	Situation    Situation
	Setup        config.SetupSnapshot
	ExpiredPlans []string
	Last         *LastLaunch
}

// NewManager builds a manager for the detected catalog.
func NewManager(paths config.Paths, catalog steam.Catalog) (*Manager, error) {
	root := catalog.Installation.Root
	if root == "" {
		return nil, domain.ErrSteamNotFound
	}
	lifecycle, err := steam.NewLifecycle(root)
	if err != nil {
		return nil, err
	}
	return &Manager{
		Paths:          paths,
		Store:          NewStore(paths.DataDir),
		Catalog:        catalog,
		Lifecycle:      lifecycle,
		Installer:      helper.NewInstaller(paths),
		RestoreFn:      config.Restore,
		SetupDetect:    config.DetectSetupFromState,
		SetupInstall:   config.InstallSetup,
		SetupGuide:     config.RecordGuideSetup,
		SetupRestore:   config.RestoreSetup,
		LaunchSession:  lifecycle.AppLaunchSession,
		LocalState:     config.ReadLocalState,
		PollInterval:   time.Second,
		StartupTimeout: StartupTimeout,
		Now:            time.Now,
	}, nil
}

// Start validates the target, arms a durable intent and asks Steam to launch
// the donor. It never writes SpaceWar's launch option and never calls
// Steam's shutdown or relaunch paths.
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
	if locked, err := m.Store.Locked(); err != nil {
		return Active{}, err
	} else if locked {
		return Active{}, ErrSessionBusy
	}
	if active, err := m.Store.ActiveState(); err != nil {
		return Active{}, err
	} else if active != nil {
		return Active{}, fmt.Errorf("%w: %s", ErrSessionBusy, active.Token)
	}
	_, _ = m.Store.PruneStates(maxRetainedStates)
	if m.Installer == nil {
		m.Installer = helper.NewInstaller(m.Paths)
	}
	if _, err := m.Installer.Install(false); err != nil {
		return Active{}, err
	}
	if m.Lifecycle == nil {
		return Active{}, domain.ErrSteamNotFound
	}
	if m.LaunchSession == nil {
		m.LaunchSession = m.Lifecycle.AppLaunchSession
	}

	local, err := m.localState()
	if err != nil {
		return Active{}, err
	}
	snapshot, err := m.setupSnapshot(local)
	if err != nil {
		return Active{}, err
	}
	if snapshot.Status != config.SetupInstalled {
		if snapshot.Status == config.SetupConflict {
			return Active{}, fmt.Errorf("%w: current SpaceWar configuration differs from the setup record", config.ErrSetupConflict)
		}
		return Active{}, fmt.Errorf("%w: install or paste the dispatcher from the guide first", config.ErrSetupMissing)
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
		return Active{}, err
	}
	if options.OnState != nil {
		options.OnState(preparing)
	}
	intent := NewIntent(token, options.TargetAppID, now)
	if err := m.Store.WriteIntent(intent); err != nil {
		_ = m.Store.RemovePlan(token)
		_ = m.Store.RemoveState(token)
		return Active{}, err
	}
	last := LastLaunch{
		Version:    LastLaunchVersion,
		AppID:      game.AppID,
		Name:       game.Name,
		Token:      token,
		Parameters: append([]string(nil), game.Launch.Arguments...),
		Kind:       game.Launch.Kind.String(),
		Source:     SourceBetterRPT,
		StartedAt:  now.UTC(),
		Outcome:    LaunchPending,
	}
	if err := m.Store.WriteLastLaunch(last); err != nil {
		_ = m.Store.ClearIntent()
		_ = m.Store.RemovePlan(token)
		_ = m.Store.RemoveState(token)
		return Active{}, err
	}
	result := Active{Token: token, Plan: plan, Game: game, Last: &last}
	if err := ctx.Err(); err != nil {
		_ = m.Store.ClearIntent()
		_ = m.markStateError(token, "canceled", err)
		_ = m.updateLast(token, LaunchCanceled, err, true)
		return result, err
	}
	// The Steam request happens before the helper can report donor_starting.
	// Record the launch request so the UI can stop offering cancel and show
	// the confirmation prompt at the right step.
	launching := preparing
	launching.Phase = domain.PhaseLaunching
	launching.UpdatedAt = m.now().UTC()
	if err := m.Store.WriteState(launching); err != nil {
		_ = m.Store.ClearIntent()
		_ = m.Store.RemovePlan(token)
		_ = m.Store.RemoveState(token)
		_ = m.updateLast(token, LaunchError, err, true)
		return result, err
	}
	if options.OnState != nil {
		options.OnState(launching)
	}
	if err := m.LaunchSession(ctx, domain.SpaceWarAppID, token); err != nil {
		_ = m.Store.ClearIntent()
		_ = m.markStateError(token, "launch_failed", err)
		_ = m.updateLast(token, LaunchError, err, true)
		if state, readErr := m.Store.ReadState(token); readErr == nil {
			result.State = state
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
				_ = m.Store.ClearIntent()
				_ = m.updateLast(token, LaunchOK, nil, false)
				if updated, err := m.Store.ReadLastLaunch(); err == nil {
					result.Last = updated
				}
				return result, nil
			case domain.PhaseError, domain.PhaseEnded:
				_ = m.Store.ClearIntent()
				_ = m.Store.RemovePlan(token)
				outcome := LaunchOK
				if state.Phase == domain.PhaseError {
					outcome = LaunchError
				}
				_ = m.updateLast(token, outcome, nil, true)
				return result, m.stateError(state)
			}
		} else if !errors.Is(readErr, ErrStateMissing) {
			_ = m.Store.ClearIntent()
			if markErr := m.markStateError(token, "state_unreadable", readErr); markErr != nil {
				_ = m.Store.RemoveState(token)
			}
			_ = m.Store.RemovePlan(token)
			_ = m.updateLast(token, LaunchError, readErr, true)
			if state, stateErr := m.Store.ReadState(token); stateErr == nil {
				result.State = state
			}
			return result, readErr
		}
		if !m.now().Before(deadline) {
			err := fmt.Errorf("%w: helper did not reach active", ErrTimeout)
			_ = m.Store.ClearIntent()
			_ = m.markStateError(token, "startup_timeout", err)
			_ = m.Store.RemovePlan(token)
			_ = m.updateLast(token, LaunchTimeout, err, true)
			if state, readErr := m.Store.ReadState(token); readErr == nil {
				result.State = state
			}
			return result, err
		}
		if err := sleepContext(ctx, m.pollInterval()); err != nil {
			_ = m.Store.ClearIntent()
			_ = m.markStateError(token, "canceled", err)
			_ = m.updateLast(token, LaunchCanceled, err, true)
			if state, readErr := m.Store.ReadState(token); readErr == nil {
				result.State = state
			}
			return result, err
		}
	}
}

// Replay starts the durable last launch again with the same target and
// parameters. It refuses to reproduce a command whose appinfo launch
// parameters changed since the record was written.
func (m *Manager) Replay(ctx context.Context) (Active, error) {
	last, err := m.Store.ReadLastLaunch()
	if err != nil {
		return Active{}, err
	}
	game, ok := m.game(last.AppID)
	if !ok || game.Launch == nil {
		return Active{}, fmt.Errorf("%w: last app %d is no longer installed", domain.ErrInvalidLaunchSpec, last.AppID)
	}
	if len(last.Parameters) > 0 {
		if !sameStrings(last.Parameters, game.Launch.Arguments) {
			return Active{}, fmt.Errorf("%w: last launch parameters changed for app %d", domain.ErrInvalidLaunchSpec, last.AppID)
		}
	}
	return m.Start(ctx, StartOptions{TargetAppID: last.AppID})
}

// Restore closes one legacy session's launch-option transaction. It refuses
// while the helper is alive and never closes Steam; with Steam open the caller
// must show the manual guide.
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
	if err := m.restore(ctx, token, active); err != nil {
		return err
	}
	if err := m.Store.RemovePlan(token); err != nil {
		return err
	}
	return m.Store.RemoveState(token)
}

// InstallSetup installs the persistent dispatcher. It requires Steam to be
// closed and never closes Steam itself.
func (m *Manager) InstallSetup(ctx context.Context) (config.Setup, error) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return config.Setup{}, err
		}
	}
	if m.Installer == nil {
		m.Installer = helper.NewInstaller(m.Paths)
	}
	if _, err := m.Installer.Install(false); err != nil {
		return config.Setup{}, err
	}
	if m.SetupInstall == nil {
		m.SetupInstall = config.InstallSetup
	}
	return m.SetupInstall(config.InstallSetupOptions{
		Paths:        m.Paths,
		SteamRoot:    m.Catalog.Installation.Root,
		HelperPath:   m.Paths.HelperPath(),
		Now:          m.now(),
		SteamRunning: m.steamGuard,
	})
}

// RecordGuideSetup records the original value for a manual paste. It never
// writes localconfig.
func (m *Manager) RecordGuideSetup(ctx context.Context) (config.Setup, error) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return config.Setup{}, err
		}
	}
	if m.SetupGuide == nil {
		m.SetupGuide = config.RecordGuideSetup
	}
	return m.SetupGuide(config.GuideSetupOptions{
		Paths:      m.Paths,
		SteamRoot:  m.Catalog.Installation.Root,
		HelperPath: m.Paths.HelperPath(),
		Now:        m.now(),
	})
}

// RestoreSetup removes the persistent dispatcher. It requires Steam to be
// closed; the caller shows the manual guide while Steam is open.
func (m *Manager) RestoreSetup(ctx context.Context) (config.Restored, error) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return config.Restored{}, err
		}
	}
	if m.SetupRestore == nil {
		m.SetupRestore = config.RestoreSetup
	}
	return m.SetupRestore(config.RestoreSetupOptions{
		Paths:        m.Paths,
		SteamRoot:    m.Catalog.Installation.Root,
		SteamRunning: m.steamGuard,
	})
}

// SetupSnapshot reads the current setup without writing.
func (m *Manager) SetupSnapshot() (config.SetupSnapshot, error) {
	local, err := m.localState()
	if err != nil {
		return config.SetupSnapshot{}, err
	}
	return m.setupSnapshot(local)
}

// Situation classifies the current durable and Steam configuration state.
func (m *Manager) Situation() (Situation, error) {
	situation, _, err := m.situation()
	return situation, err
}

func (m *Manager) situation() (Situation, config.SetupSnapshot, error) {
	now := m.now()
	last, err := m.Store.ReadLastLaunch()
	if err != nil && !errors.Is(err, ErrLastLaunchMissing) {
		return Situation{}, config.SetupSnapshot{}, err
	}
	if last != nil {
		last = m.reconcileLast(last)
	}

	var snapshot config.SetupSnapshot
	marker := MarkerNone
	var legacy *LegacyEvidence
	var stale *config.Stale
	root := m.Catalog.Installation.Root
	if root != "" {
		local, err := m.localState()
		if err != nil {
			return Situation{}, config.SetupSnapshot{}, err
		}
		last = m.reconcileSteamLast(last, local)
		snapshot, err = m.setupSnapshot(local)
		if err != nil {
			return Situation{}, config.SetupSnapshot{}, err
		}
		if snapshot.Current.Exists && snapshot.Current.Value != "" {
			switch {
			case config.IsDispatcherMarker(snapshot.Current.Value):
				marker = MarkerDispatcher
			case config.IsSessionMarker(snapshot.Current.Value):
				marker = MarkerLegacy
			default:
				marker = MarkerForeign
			}
		}
		if marker == MarkerLegacy {
			stale, err = config.DetectStaleFromState(m.Paths, local, false)
			if err != nil {
				return Situation{}, config.SetupSnapshot{}, err
			}
			if stale != nil {
				legacy = &LegacyEvidence{}
				if stale.Backup.Path != "" {
					legacy.BackupAvailable = true
					legacy.BackupToken = stale.Backup.Meta.SessionToken
					legacy.BackupTarget = stale.Backup.Meta.TargetAppID
				}
			}
		}
	}

	var state *State
	stateToken := ""
	if legacy != nil && legacy.BackupToken != "" {
		stateToken = legacy.BackupToken
	} else if last != nil && last.Token != "" {
		stateToken = last.Token
	}
	if stateToken != "" {
		loaded, readErr := m.Store.ReadState(stateToken)
		if readErr != nil && !errors.Is(readErr, ErrStateMissing) {
			return Situation{}, config.SetupSnapshot{}, readErr
		}
		if readErr == nil {
			state = &loaded
		}
	}
	if state == nil {
		active, activeErr := m.Store.ActiveState()
		if activeErr != nil {
			return Situation{}, config.SetupSnapshot{}, activeErr
		}
		if active != nil {
			state = active
		} else if states, listErr := m.Store.ListStates(); listErr == nil && len(states) > 0 {
			state = &states[0]
		}
	}
	helperAlive := false
	if state != nil && state.HelperPID > 0 && !state.Terminal() {
		helperAlive = ProcessAlive(state.HelperPID, state.HelperStartTime)
	}
	intent, intentErr := m.Store.ReadIntent(now)
	if intentErr != nil && !errors.Is(intentErr, ErrIntentMissing) &&
		!errors.Is(intentErr, ErrIntentExpired) {
		return Situation{}, config.SetupSnapshot{}, intentErr
	}
	steamRunning := false
	if m.Lifecycle != nil {
		steamRunning, err = m.Lifecycle.IsRunning()
		if err != nil {
			return Situation{}, config.SetupSnapshot{}, err
		}
	}
	situation := Classify(Evidence{
		Setup:        setupState(snapshot.Status),
		SetupBackup:  snapshot.Setup != nil && snapshot.Setup.Backup.Path != "",
		Marker:       marker,
		Legacy:       legacy,
		Intent:       intent,
		State:        state,
		HelperAlive:  helperAlive,
		SteamRunning: steamRunning,
		Last:         last,
	})
	situation.SetupSnapshot = snapshot
	return situation, snapshot, nil
}

// Recover inspects startup state and returns the classified situation.
func (m *Manager) Recover(ctx context.Context) (*Recovery, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	_, _ = m.Store.PruneStates(maxRetainedStates)
	active, err := m.Store.ActiveState()
	if err != nil {
		return nil, err
	}
	recovery := &Recovery{}
	if active != nil {
		alive := active.HelperPID > 0 && ProcessAlive(active.HelperPID, active.HelperStartTime)
		recovery.State = active
		recovery.HelperAlive = alive
		if !alive {
			active.Phase = domain.PhaseError
			active.Outcome = "error"
			active.ErrorCode = "helper_dead"
			active.Error = "the session helper is no longer running"
			active.EndedAt = m.now()
			active.UpdatedAt = active.EndedAt
			if err := m.Store.WriteState(*active); err != nil {
				return nil, err
			}
			_ = m.updateLast(active.Token, LaunchError, errors.New(active.Error), true)
		}
	} else {
		expired, err := m.Store.RemoveExpiredPlans(m.now())
		if err != nil {
			return nil, err
		}
		recovery.ExpiredPlans = expired
	}
	if intent, err := m.Store.ReadIntent(m.now()); errors.Is(err, ErrIntentExpired) {
		_ = intent
		_ = m.Store.ClearIntent()
	}
	situation, setup, err := m.situation()
	if err != nil {
		return nil, err
	}
	recovery.Situation = situation
	recovery.Last = situation.Last
	recovery.Setup = setup
	if recovery.State == nil && recovery.Last == nil && situation.Kind == SituationIdle &&
		situation.Setup == SetupMissing && len(recovery.ExpiredPlans) == 0 {
		return nil, nil
	}
	return recovery, nil
}

func (m *Manager) restore(ctx context.Context, token string, active bool) (err error) {
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
	if err := cleanupCtx.Err(); err != nil {
		return err
	}
	_, err = m.RestoreFn(config.RestoreOptions{
		Paths:         m.Paths,
		SteamRoot:     m.Catalog.Installation.Root,
		SessionToken:  token,
		ActiveSession: active,
		SteamRunning:  m.steamGuard,
	})
	return err
}

func (m *Manager) localState() (config.LocalState, error) {
	if m.LocalState == nil {
		m.LocalState = config.ReadLocalState
	}
	return m.LocalState(m.Catalog.Installation.Root)
}

func (m *Manager) setupSnapshot(local config.LocalState) (config.SetupSnapshot, error) {
	if m.SetupDetect == nil {
		m.SetupDetect = config.DetectSetupFromState
	}
	return m.SetupDetect(m.Paths, local)
}

func (m *Manager) reconcileSteamLast(last *LastLaunch, local config.LocalState) *LastLaunch {
	played := local.LastPlayed
	if played == nil {
		return last
	}
	if last != nil {
		// A Better RPT record for the same app carries the token and exact
		// parameters needed by Replay. Steam's LastPlayed must not erase them.
		if last.AppID == played.AppID {
			return last
		}
		if !played.At.After(last.StartedAt) {
			return last
		}
		if last.Token != "" {
			if state, readErr := m.Store.ReadState(last.Token); readErr == nil && !state.Terminal() {
				return last
			}
		}
	}
	name := ""
	if game, ok := m.game(played.AppID); ok {
		name = game.Name
	}
	next := LastLaunch{
		Version:   LastLaunchVersion,
		AppID:     played.AppID,
		Name:      name,
		Source:    SourceSteam,
		StartedAt: played.At.UTC(),
		Outcome:   LaunchUnknown,
	}
	if err := m.Store.WriteLastLaunch(next); err != nil {
		return last
	}
	return &next
}

func (m *Manager) reconcileLast(last *LastLaunch) *LastLaunch {
	if last == nil || last.Token == "" {
		return last
	}
	state, err := m.Store.ReadState(last.Token)
	if err != nil {
		return last
	}
	changed := false
	switch state.Phase {
	case domain.PhaseActive:
		if last.Outcome != LaunchOK {
			last.Outcome = LaunchOK
			changed = true
		}
	case domain.PhaseEnded:
		last.Outcome = LaunchOK
		changed = true
	case domain.PhaseError:
		last.Outcome = LaunchError
		changed = true
	}
	if !state.EndedAt.IsZero() && !state.EndedAt.Equal(last.EndedAt) {
		last.EndedAt = state.EndedAt
		changed = true
	}
	if changed {
		_ = m.Store.WriteLastLaunch(*last)
	}
	return last
}

func (m *Manager) updateLast(token string, outcome LaunchOutcome, cause error, ended bool) error {
	last, err := m.Store.ReadLastLaunch()
	if err != nil {
		if errors.Is(err, ErrLastLaunchMissing) {
			return nil
		}
		return err
	}
	if last.Token != token {
		return nil
	}
	last.Outcome = outcome
	if cause != nil {
		last.Error = cause.Error()
	}
	if ended {
		last.EndedAt = m.now().UTC()
	}
	return m.Store.WriteLastLaunch(*last)
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

func setupState(status config.SetupStatus) SetupState {
	switch status {
	case config.SetupGuide:
		return SetupGuide
	case config.SetupInstalled:
		return SetupInstalled
	case config.SetupRestored:
		return SetupRestored
	case config.SetupConflict:
		return SetupConflict
	default:
		return SetupMissing
	}
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
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
