// Package bridge is the only surface the React frontend binds to. It owns the
// translation from domain values to serializable DTOs, the session context and
// the event emission. It deliberately does not import Wails.
package bridge

import (
	"context"
	"os/exec"
	"sync"

	"github.com/srnoob2570/better-rpt/internal/config"
	"github.com/srnoob2570/better-rpt/internal/domain"
	"github.com/srnoob2570/better-rpt/internal/i18n"
	"github.com/srnoob2570/better-rpt/internal/session"
	"github.com/srnoob2570/better-rpt/internal/steam"
)

const (
	// EventCatalogUpdated carries a Catalog after a successful refresh.
	EventCatalogUpdated = "catalog.updated"
	// EventSessionUpdated carries a SessionState on every observed phase.
	EventSessionUpdated = "session.updated"
)

// Emitter is the Wails event adapter. The bridge stays free of Wails imports
// so it can be tested without a display server.
type Emitter interface {
	Emit(name string, data any)
}

// App is the part of app.Service the bridge needs. *app.Service satisfies it;
// tests provide a fake.
type App interface {
	Paths() config.Paths
	Config() config.Config
	Language() i18n.Language
	Catalog() (steam.Catalog, bool)
	RefreshCatalog(ctx context.Context) (steam.Catalog, error)
	EnsureHelper() error
	SetLanguage(lang string) error
	StartSession(ctx context.Context, options session.StartOptions) (session.Active, error)
	ReplaySession(ctx context.Context) (session.Active, error)
	RestoreSession(ctx context.Context, token string) error
	RecoverSession(ctx context.Context) (*session.Recovery, error)
	SessionState(token string) (session.State, error)
	SessionSituation() (session.Situation, error)
	SetupSnapshot() (config.SetupSnapshot, error)
	InstallSetup(ctx context.Context) (config.Setup, error)
	RecordGuideSetup(ctx context.Context) (config.Setup, error)
	RestoreSetup(ctx context.Context) (config.Restored, error)
	DispatcherCommand() (string, error)
}

// Bridge is the Wails-bound facade.
type Bridge struct {
	app     App
	version string
	emitter Emitter
	openURL func(string) error
	mu      sync.Mutex
	run     *runState
}

type runState struct {
	cancel context.CancelFunc
	done   chan struct{}
	phase  domain.Phase
}

// New builds a bridge. A nil emitter is allowed for tests and headless runs.
func New(app App, version string, emitter Emitter) *Bridge {
	return &Bridge{
		app:     app,
		version: version,
		emitter: emitter,
		openURL: defaultOpenURL,
	}
}

// Snapshot returns the cached startup state. It performs no Steam I/O.
func (b *Bridge) Snapshot(_ context.Context) (Snapshot, error) {
	language := string(b.app.Language())
	snapshot := Snapshot{
		SchemaVersion: SchemaVersion,
		Version:       b.version,
		Language:      language,
		Config:        Config{Language: language},
	}
	if catalog, ok := b.app.Catalog(); ok {
		mapped := mapCatalog(catalog)
		snapshot.Catalog = &mapped
	}
	return snapshot, nil
}

// RefreshCatalog rebuilds the library and emits catalog.updated on success.
func (b *Bridge) RefreshCatalog(ctx context.Context) (Catalog, error) {
	catalog, err := b.app.RefreshCatalog(contextOrBackground(ctx))
	if err != nil {
		return Catalog{}, err
	}
	mapped := mapCatalog(catalog)
	b.emit(EventCatalogUpdated, mapped)
	return mapped, nil
}

// StartSession starts a launch using a context owned by the bridge. The Wails
// call context is intentionally ignored: resolving the promise must not cancel
// the preparation.
func (b *Bridge) StartSession(_ context.Context, input StartSessionInput) (Active, error) {
	return b.withRun(false, func(ctx context.Context, run *runState) (session.Active, error) {
		return b.app.StartSession(ctx, session.StartOptions{
			TargetAppID: input.TargetAppID,
			OnState: func(state session.State) {
				b.observe(run, state)
			},
		})
	})
}

// ReplaySession repeats the last launch. It uses the same bridge-owned context
// and emits the final state; the manager does not report intermediate phases.
func (b *Bridge) ReplaySession(_ context.Context) (Active, error) {
	return b.withRun(true, func(ctx context.Context, _ *runState) (session.Active, error) {
		return b.app.ReplaySession(ctx)
	})
}

// CancelSession cancels only while the observed phase is preparing. Once the
// donor or target is starting, cancellation would orphan a Steam process.
func (b *Bridge) CancelSession() bool {
	b.mu.Lock()
	run := b.run
	if run == nil || run.phase != domain.PhasePreparing {
		b.mu.Unlock()
		return false
	}
	b.mu.Unlock()
	run.cancel()
	return true
}

// Shutdown cancels any in-flight preparation and waits for the start call to
// finish. The Wails shell calls it before closing the service.
//
//wails:ignore
func (b *Bridge) Shutdown() {
	b.mu.Lock()
	run := b.run
	b.mu.Unlock()
	if run == nil {
		return
	}
	run.cancel()
	<-run.done
}

// RestoreSession restores one legacy or persistent configuration.
func (b *Bridge) RestoreSession(ctx context.Context, token string) error {
	return b.app.RestoreSession(contextOrBackground(ctx), token)
}

// RecoverSession reads startup recovery state. It does not emit events; the
// frontend calls Snapshot and RecoverSession again after a reload.
func (b *Bridge) RecoverSession(ctx context.Context) (Recovery, error) {
	recovery, err := b.app.RecoverSession(contextOrBackground(ctx))
	if err != nil {
		return Recovery{ExpiredPlans: []string{}}, err
	}
	return mapRecovery(recovery), nil
}

// SessionState reads the durable state for polling.
func (b *Bridge) SessionState(token string) (SessionState, error) {
	state, err := b.app.SessionState(token)
	if err != nil {
		return SessionState{}, err
	}
	return mapSessionState(state), nil
}

// Situation classifies the current lifecycle state after an action.
func (b *Bridge) Situation() (Situation, error) {
	situation, err := b.app.SessionSituation()
	if err != nil {
		return Situation{}, err
	}
	return mapSituation(situation), nil
}

// EnsureHelper installs or refreshes the stable helper. It is safe to call
// repeatedly and leaves an in-use helper alone during an active session.
func (b *Bridge) EnsureHelper() error {
	return b.app.EnsureHelper()
}

// SetupSnapshot reads the dispatcher setup and the current SpaceWar option.
func (b *Bridge) SetupSnapshot() (SetupSnapshot, error) {
	snapshot, err := b.app.SetupSnapshot()
	if err != nil {
		return SetupSnapshot{}, err
	}
	return mapSetupSnapshot(snapshot), nil
}

// InstallSetup installs the persistent dispatcher; Steam must be closed.
func (b *Bridge) InstallSetup(ctx context.Context) (Setup, error) {
	setup, err := b.app.InstallSetup(contextOrBackground(ctx))
	if err != nil {
		return Setup{}, err
	}
	return mapSetup(setup), nil
}

// RecordGuideSetup records a manual dispatcher installation.
func (b *Bridge) RecordGuideSetup(ctx context.Context) (Setup, error) {
	setup, err := b.app.RecordGuideSetup(contextOrBackground(ctx))
	if err != nil {
		return Setup{}, err
	}
	return mapSetup(setup), nil
}

// RestoreSetup restores the persistent dispatcher; Steam must be closed.
func (b *Bridge) RestoreSetup(ctx context.Context) (Setup, error) {
	restored, err := b.app.RestoreSetup(contextOrBackground(ctx))
	if err != nil {
		return Setup{}, err
	}
	return Setup{
		Status:        string(config.SetupRestored),
		OriginalValue: restored.Backup.Meta.OriginalValue,
		HasBackup:     restored.Backup.Path != "",
	}, nil
}

// DispatcherCommand returns the exact SpaceWar launch option.
func (b *Bridge) DispatcherCommand() (string, error) {
	return b.app.DispatcherCommand()
}

// SetLanguage validates and persists the UI language. It emits no event; the
// caller already knows the new value.
func (b *Bridge) SetLanguage(lang string) error {
	return b.app.SetLanguage(lang)
}

// OpenSpaceWarInstall opens Steam's fixed install URL for the donor.
func (b *Bridge) OpenSpaceWarInstall() error {
	return b.open("steam://install/480")
}

// OpenSteam opens Steam's main window through a fixed URL.
func (b *Bridge) OpenSteam() error {
	return b.open("steam://open/main")
}

// Version returns the build version displayed in the header.
func (b *Bridge) Version() string {
	return b.version
}

func (b *Bridge) withRun(preparing bool, start func(context.Context, *runState) (session.Active, error)) (Active, error) {
	b.mu.Lock()
	if b.run != nil {
		b.mu.Unlock()
		return Active{}, newProblemError("session_busy", "Another session is already active.", "")
	}
	ctx, cancel := context.WithCancel(context.Background())
	run := &runState{cancel: cancel, done: make(chan struct{})}
	if preparing {
		run.phase = domain.PhasePreparing
	}
	b.run = run
	b.mu.Unlock()

	active, err := start(ctx, run)
	b.finish(run, active)
	if err != nil {
		return Active{}, err
	}
	return mapActive(active), nil
}

func (b *Bridge) observe(run *runState, state session.State) {
	b.mu.Lock()
	if b.run != run {
		b.mu.Unlock()
		return
	}
	run.phase = state.Phase
	b.mu.Unlock()
	b.emit(EventSessionUpdated, mapSessionState(state))
}

func (b *Bridge) finish(run *runState, active session.Active) {
	if active.State.Token != "" {
		b.observe(run, active.State)
	}
	b.mu.Lock()
	if b.run == run {
		b.run = nil
	}
	run.cancel()
	close(run.done)
	b.mu.Unlock()
}

func (b *Bridge) emit(name string, data any) {
	if b.emitter != nil {
		b.emitter.Emit(name, data)
	}
}

func (b *Bridge) open(target string) error {
	if b.openURL == nil {
		return defaultOpenURL(target)
	}
	return b.openURL(target)
}

func defaultOpenURL(target string) error {
	return exec.Command("xdg-open", target).Start()
}

func contextOrBackground(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}
