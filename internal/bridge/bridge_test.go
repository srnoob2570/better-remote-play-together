package bridge

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/srnoob2570/better-rpt/internal/config"
	"github.com/srnoob2570/better-rpt/internal/domain"
	"github.com/srnoob2570/better-rpt/internal/i18n"
	"github.com/srnoob2570/better-rpt/internal/launch"
	"github.com/srnoob2570/better-rpt/internal/session"
	"github.com/srnoob2570/better-rpt/internal/steam"
)

type recordedEvent struct {
	name string
	data any
}

type eventRecorder struct {
	mu     sync.Mutex
	events []recordedEvent
}

func (r *eventRecorder) Emit(name string, data any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, recordedEvent{name: name, data: data})
}

func (r *eventRecorder) snapshot() []recordedEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]recordedEvent(nil), r.events...)
}

type fakeApp struct {
	paths      config.Paths
	language   i18n.Language
	catalog    steam.Catalog
	hasCatalog bool

	refreshFn      func(context.Context) (steam.Catalog, error)
	ensureHelperFn func() error
	startFn        func(context.Context, session.StartOptions) (session.Active, error)
	replayFn       func(context.Context) (session.Active, error)
	restoreFn      func(context.Context, string) error
	recoverFn      func(context.Context) (*session.Recovery, error)
	stateFn        func(string) (session.State, error)
	situationFn    func() (session.Situation, error)
	setupFn        func() (config.SetupSnapshot, error)
	installFn      func(context.Context) (config.Setup, error)
	recordFn       func(context.Context) (config.Setup, error)
	restoreSetupFn func(context.Context) (config.Restored, error)
	dispatcher     string
	dispatcherErr  error
	setLanguageFn  func(string) error
}

func (f *fakeApp) Paths() config.Paths            { return f.paths }
func (f *fakeApp) Language() i18n.Language        { return f.language }
func (f *fakeApp) Catalog() (steam.Catalog, bool) { return f.catalog, f.hasCatalog }

func (f *fakeApp) RefreshCatalog(ctx context.Context) (steam.Catalog, error) {
	if f.refreshFn != nil {
		return f.refreshFn(ctx)
	}
	return f.catalog, nil
}

func (f *fakeApp) EnsureHelper() error {
	if f.ensureHelperFn != nil {
		return f.ensureHelperFn()
	}
	return nil
}

func (f *fakeApp) StartSession(ctx context.Context, options session.StartOptions) (session.Active, error) {
	if f.startFn != nil {
		return f.startFn(ctx, options)
	}
	return session.Active{}, nil
}

func (f *fakeApp) ReplaySession(ctx context.Context) (session.Active, error) {
	if f.replayFn != nil {
		return f.replayFn(ctx)
	}
	return session.Active{}, nil
}

func (f *fakeApp) RestoreSession(ctx context.Context, token string) error {
	if f.restoreFn != nil {
		return f.restoreFn(ctx, token)
	}
	return nil
}

func (f *fakeApp) RecoverSession(ctx context.Context) (*session.Recovery, error) {
	if f.recoverFn != nil {
		return f.recoverFn(ctx)
	}
	return nil, nil
}

func (f *fakeApp) SessionState(token string) (session.State, error) {
	if f.stateFn != nil {
		return f.stateFn(token)
	}
	return session.State{}, nil
}

func (f *fakeApp) SessionSituation() (session.Situation, error) {
	if f.situationFn != nil {
		return f.situationFn()
	}
	return session.Situation{}, nil
}

func (f *fakeApp) SetupSnapshot() (config.SetupSnapshot, error) {
	if f.setupFn != nil {
		return f.setupFn()
	}
	return config.SetupSnapshot{}, nil
}

func (f *fakeApp) InstallSetup(ctx context.Context) (config.Setup, error) {
	if f.installFn != nil {
		return f.installFn(ctx)
	}
	return config.Setup{}, nil
}

func (f *fakeApp) RecordGuideSetup(ctx context.Context) (config.Setup, error) {
	if f.recordFn != nil {
		return f.recordFn(ctx)
	}
	return config.Setup{}, nil
}

func (f *fakeApp) RestoreSetup(ctx context.Context) (config.Restored, error) {
	if f.restoreSetupFn != nil {
		return f.restoreSetupFn(ctx)
	}
	return config.Restored{}, nil
}

func (f *fakeApp) DispatcherCommand() (string, error) {
	return f.dispatcher, f.dispatcherErr
}

func (f *fakeApp) SetLanguage(lang string) error {
	if f.setLanguageFn != nil {
		return f.setLanguageFn(lang)
	}
	return nil
}

func testToken() string {
	return "0123456789abcdef0123456789abcdef"
}

func testGame() domain.Game {
	return domain.Game{
		AppID:       1599600,
		Name:        "PlateUp!",
		InstallDir:  "/steam/steamapps/common/PlateUp",
		LibraryPath: "/steam",
		SizeBytes:   1024,
		Kind:        domain.KindProton,
		Status:      domain.StatusUnknown,
		Problem:     fmt.Errorf("%w: no launch entry", launch.ErrNoLaunchForOS),
	}
}

func testCatalog() steam.Catalog {
	return steam.Catalog{
		Games: []domain.Game{testGame()},
		Warnings: []steam.Warning{{
			Path: "/steam/steamapps",
			Err:  fmt.Errorf("%w: unreadable", domain.ErrLibraryUnreadable),
		}},
	}
}

func testState(phase domain.Phase) session.State {
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	return session.State{
		Version:     session.StateVersion,
		Token:       testToken(),
		DonorAppID:  domain.SpaceWarAppID,
		TargetAppID: 1599600,
		Phase:       phase,
		CreatedAt:   now,
		UpdatedAt:   now,
		StartedAt:   now,
	}
}

func TestSnapshot(t *testing.T) {
	app := &fakeApp{
		language:   i18n.English,
		catalog:    testCatalog(),
		hasCatalog: true,
	}
	got, err := New(app, "1.2.3", nil).Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.SchemaVersion != SchemaVersion || got.Version != "1.2.3" || got.Language != "en" {
		t.Fatalf("snapshot header = %+v", got)
	}
	if got.Config.Language != "en" {
		t.Fatalf("config language = %q", got.Config.Language)
	}
	if got.Catalog == nil || len(got.Catalog.Games) != 1 || len(got.Catalog.Warnings) != 1 {
		t.Fatalf("catalog = %+v", got.Catalog)
	}
	game := got.Catalog.Games[0]
	if game.AppID != 1599600 || game.Kind != "proton" || game.Status != "unknown" {
		t.Fatalf("game = %+v", game)
	}
	if game.Problem == nil || game.Problem.Code != "no_launch_for_os" {
		t.Fatalf("game problem = %+v", game.Problem)
	}
	if got.Catalog.Warnings[0].Problem.Code != "library_unreadable" {
		t.Fatalf("warning = %+v", got.Catalog.Warnings[0])
	}
}

func TestSnapshotWithoutCatalog(t *testing.T) {
	app := &fakeApp{language: i18n.Spanish}
	got, err := New(app, "dev", nil).Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Catalog != nil {
		t.Fatalf("catalog = %+v, want nil", got.Catalog)
	}
}

func TestRefreshCatalogReturnsCatalog(t *testing.T) {
	recorder := &eventRecorder{}
	app := &fakeApp{refreshFn: func(context.Context) (steam.Catalog, error) {
		return testCatalog(), nil
	}}
	got, err := New(app, "dev", recorder).RefreshCatalog(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Games) != 1 {
		t.Fatalf("catalog = %+v", got)
	}
	if events := recorder.snapshot(); len(events) != 0 {
		t.Fatalf("catalog was delivered twice: events = %+v", events)
	}
}

func TestRefreshCatalogError(t *testing.T) {
	app := &fakeApp{refreshFn: func(context.Context) (steam.Catalog, error) {
		return steam.Catalog{}, domain.ErrSteamNotFound
	}}
	_, err := New(app, "dev", nil).RefreshCatalog(context.Background())
	if !errors.Is(err, domain.ErrSteamNotFound) {
		t.Fatalf("err = %v", err)
	}
}

func TestStartSessionEmitsPhasesAndFinalState(t *testing.T) {
	recorder := &eventRecorder{}
	preparing := testState(domain.PhasePreparing)
	donor := testState(domain.PhaseDonorStarting)
	active := testState(domain.PhaseActive)
	app := &fakeApp{startFn: func(_ context.Context, options session.StartOptions) (session.Active, error) {
		options.OnState(preparing)
		options.OnState(donor)
		options.OnState(active)
		return session.Active{Token: testToken(), Game: testGame(), State: active}, nil
	}}
	got, err := New(app, "dev", recorder).StartSession(context.Background(), StartSessionInput{TargetAppID: 1599600})
	if err != nil {
		t.Fatal(err)
	}
	if got.Token != testToken() || got.State.Phase != "active" {
		t.Fatalf("active = %+v", got)
	}
	events := recorder.snapshot()
	if len(events) != 4 {
		t.Fatalf("events = %d, want 4: %+v", len(events), events)
	}
	wantPhases := []string{"preparing", "donor_starting", "active", "active"}
	for index, event := range events {
		if event.name != EventSessionUpdated {
			t.Fatalf("event %d name = %q", index, event.name)
		}
		state, ok := event.data.(SessionState)
		if !ok {
			t.Fatalf("event %d payload = %T", index, event.data)
		}
		if state.Phase != wantPhases[index] {
			t.Fatalf("event %d phase = %q, want %q", index, state.Phase, wantPhases[index])
		}
	}
}

func TestStartSessionIgnoresCallContext(t *testing.T) {
	app := &fakeApp{startFn: func(ctx context.Context, _ session.StartOptions) (session.Active, error) {
		if err := ctx.Err(); err != nil {
			return session.Active{}, fmt.Errorf("bridge used the call context: %w", err)
		}
		return session.Active{Token: testToken(), Game: testGame(), State: testState(domain.PhaseActive)}, nil
	}}
	callCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := New(app, "dev", nil).StartSession(callCtx, StartSessionInput{TargetAppID: 1599600}); err != nil {
		t.Fatal(err)
	}
}

func TestCancelSessionOnlyWhilePreparing(t *testing.T) {
	recorder := &eventRecorder{}
	release := make(chan struct{})
	app := &fakeApp{startFn: func(ctx context.Context, options session.StartOptions) (session.Active, error) {
		options.OnState(testState(domain.PhasePreparing))
		close(release)
		<-ctx.Done()
		state := testState(domain.PhaseError)
		state.Outcome = "canceled"
		state.ErrorCode = "canceled"
		state.Error = "canceled"
		return session.Active{Token: testToken(), Game: testGame(), State: state}, ctx.Err()
	}}
	b := New(app, "dev", recorder)
	result := make(chan error, 1)
	go func() {
		_, err := b.StartSession(context.Background(), StartSessionInput{TargetAppID: 1599600})
		result <- err
	}()
	<-release
	if !b.CancelSession() {
		t.Fatal("CancelSession() = false, want true while preparing")
	}
	err := <-result
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if code := ErrorCode(err); code != "session_canceled" {
		t.Fatalf("code = %q", code)
	}
	events := recorder.snapshot()
	final := events[len(events)-1].data.(SessionState)
	if final.Phase != "error" || final.Outcome != "canceled" {
		t.Fatalf("final event = %+v", final)
	}
}

func TestCancelSessionRefusesAfterPreparing(t *testing.T) {
	release := make(chan struct{})
	app := &fakeApp{startFn: func(_ context.Context, options session.StartOptions) (session.Active, error) {
		options.OnState(testState(domain.PhaseDonorStarting))
		close(release)
		return session.Active{Token: testToken(), Game: testGame(), State: testState(domain.PhaseActive)}, nil
	}}
	b := New(app, "dev", nil)
	result := make(chan error, 1)
	go func() {
		_, err := b.StartSession(context.Background(), StartSessionInput{TargetAppID: 1599600})
		result <- err
	}()
	<-release
	if b.CancelSession() {
		t.Fatal("CancelSession() = true after donor started")
	}
	if err := <-result; err != nil {
		t.Fatal(err)
	}
}

func TestSecondStartIsBusy(t *testing.T) {
	release := make(chan struct{})
	app := &fakeApp{startFn: func(ctx context.Context, options session.StartOptions) (session.Active, error) {
		options.OnState(testState(domain.PhasePreparing))
		<-release
		return session.Active{Token: testToken(), Game: testGame(), State: testState(domain.PhaseActive)}, nil
	}}
	b := New(app, "dev", nil)
	result := make(chan error, 1)
	go func() {
		_, err := b.StartSession(context.Background(), StartSessionInput{TargetAppID: 1599600})
		result <- err
	}()
	<-waitForPhase(t, b, domain.PhasePreparing)
	_, err := b.StartSession(context.Background(), StartSessionInput{TargetAppID: 1599600})
	if code := ErrorCode(err); code != "session_busy" {
		t.Fatalf("second start code = %q, err = %v", code, err)
	}
	close(release)
	if err := <-result; err != nil {
		t.Fatal(err)
	}
}

func TestReplayUsesBridgeContextAndEmitsFinalState(t *testing.T) {
	recorder := &eventRecorder{}
	app := &fakeApp{replayFn: func(ctx context.Context) (session.Active, error) {
		if err := ctx.Err(); err != nil {
			return session.Active{}, err
		}
		return session.Active{Token: testToken(), Game: testGame(), State: testState(domain.PhaseActive)}, nil
	}}
	got, err := New(app, "dev", recorder).ReplaySession(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Token != testToken() {
		t.Fatalf("active = %+v", got)
	}
	events := recorder.snapshot()
	if len(events) != 1 || events[0].name != EventSessionUpdated {
		t.Fatalf("events = %+v", events)
	}
}

func TestShutdownCancelsAndWaits(t *testing.T) {
	started := make(chan struct{})
	app := &fakeApp{startFn: func(ctx context.Context, _ session.StartOptions) (session.Active, error) {
		close(started)
		<-ctx.Done()
		return session.Active{}, ctx.Err()
	}}
	b := New(app, "dev", nil)
	done := make(chan struct{})
	go func() {
		_, _ = b.StartSession(context.Background(), StartSessionInput{TargetAppID: 1599600})
		close(done)
	}()
	<-started
	b.Shutdown()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Shutdown did not wait for StartSession")
	}
}

func TestRestoreSessionPassesToken(t *testing.T) {
	var got string
	app := &fakeApp{restoreFn: func(_ context.Context, token string) error {
		got = token
		return nil
	}}
	if err := New(app, "dev", nil).RestoreSession(context.Background(), testToken()); err != nil {
		t.Fatal(err)
	}
	if got != testToken() {
		t.Fatalf("token = %q", got)
	}
}

func TestRecoverSessionMapsRecovery(t *testing.T) {
	last := &session.LastLaunch{
		AppID:     1599600,
		Name:      "PlateUp!",
		Token:     testToken(),
		Source:    session.SourceBetterRPT,
		StartedAt: time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC),
		Outcome:   session.LaunchOK,
	}
	state := testState(domain.PhaseEnded)
	app := &fakeApp{recoverFn: func(context.Context) (*session.Recovery, error) {
		return &session.Recovery{
			State:       &state,
			HelperAlive: false,
			Situation: session.Situation{
				Kind:         session.SituationRestorePending,
				Setup:        session.SetupInstalled,
				Restore:      session.RestoreAvailable,
				Token:        testToken(),
				TargetAppID:  1599600,
				SteamRunning: false,
				State:        &state,
				Last:         last,
			},
			Setup: config.SetupSnapshot{
				Status: config.SetupInstalled,
				Setup: &config.Setup{
					Status: config.SetupInstalled,
					Backup: config.Backup{Path: "/backup", Meta: config.BackupMeta{OriginalValue: "orig"}},
				},
			},
			Last:         last,
			ExpiredPlans: []string{"expired"},
		}, nil
	}}
	got, err := New(app, "dev", nil).RecoverSession(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.State == nil || got.State.Phase != "ended" || got.HelperAlive {
		t.Fatalf("recovery state = %+v", got)
	}
	if got.Situation == nil || got.Situation.Kind != "restore_pending" || got.Situation.Setup != "installed" {
		t.Fatalf("situation = %+v", got.Situation)
	}
	if got.Stale == nil || got.Stale.Token != testToken() || got.Stale.Active {
		t.Fatalf("stale = %+v", got.Stale)
	}
	if got.Setup == nil || got.Setup.Status != "installed" || got.Setup.OriginalValue != "orig" || !got.Setup.HasBackup {
		t.Fatalf("setup = %+v", got.Setup)
	}
	if got.Last == nil || got.Last.AppID != 1599600 || got.Last.Outcome != "ok" {
		t.Fatalf("last = %+v", got.Last)
	}
	if len(got.ExpiredPlans) != 1 || got.ExpiredPlans[0] != "expired" {
		t.Fatalf("expired = %+v", got.ExpiredPlans)
	}
}

func TestRecoverSessionNilHasEmptySlices(t *testing.T) {
	got, err := New(&fakeApp{}, "dev", nil).RecoverSession(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.ExpiredPlans == nil || len(got.ExpiredPlans) != 0 {
		t.Fatalf("expired = %#v", got.ExpiredPlans)
	}
}

func TestSituationAndSetup(t *testing.T) {
	app := &fakeApp{
		situationFn: func() (session.Situation, error) {
			return session.Situation{
				Kind:    session.SituationIdle,
				Setup:   session.SetupMissing,
				Restore: session.RestoreNotNeeded,
				SetupSnapshot: config.SetupSnapshot{
					Status:  config.SetupGuide,
					Current: config.LaunchOption{Value: "current"},
				},
			}, nil
		},
		setupFn: func() (config.SetupSnapshot, error) {
			return config.SetupSnapshot{Status: config.SetupGuide, Current: config.LaunchOption{Value: "current"}}, nil
		},
	}
	b := New(app, "dev", nil)
	situation, err := b.Situation()
	if err != nil {
		t.Fatal(err)
	}
	if situation.Kind != "idle" || situation.Setup != "missing" {
		t.Fatalf("situation = %+v", situation)
	}
	if situation.SetupSnapshot == nil || situation.SetupSnapshot.Status != "guide" ||
		situation.SetupSnapshot.CurrentValue != "current" {
		t.Fatalf("situation setup snapshot = %+v", situation.SetupSnapshot)
	}
	setup, err := b.SetupSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if setup.Status != "guide" || setup.CurrentValue != "current" || setup.HasBackup {
		t.Fatalf("setup = %+v", setup)
	}
}

func TestEnsureHelperDelegates(t *testing.T) {
	want := errors.New("helper source missing")
	app := &fakeApp{ensureHelperFn: func() error { return want }}
	if err := New(app, "dev", nil).EnsureHelper(); !errors.Is(err, want) {
		t.Fatalf("ensure helper error = %v", err)
	}
}

func TestSetLanguage(t *testing.T) {
	var got string
	app := &fakeApp{setLanguageFn: func(lang string) error { got = lang; return nil }}
	if err := New(app, "dev", nil).SetLanguage("en"); err != nil {
		t.Fatal(err)
	}
	if got != "en" {
		t.Fatalf("language = %q", got)
	}
}

func TestOpenURLsAreFixed(t *testing.T) {
	var opened []string
	app := &fakeApp{}
	b := New(app, "dev", nil)
	b.openURL = func(target string) error {
		opened = append(opened, target)
		return nil
	}
	if err := b.OpenSpaceWarInstall(); err != nil {
		t.Fatal(err)
	}
	if err := b.OpenSteam(); err != nil {
		t.Fatal(err)
	}
	want := []string{"steam://install/480", "steam://open/main"}
	if len(opened) != len(want) {
		t.Fatalf("opened = %+v", opened)
	}
	for index := range want {
		if opened[index] != want[index] {
			t.Fatalf("opened[%d] = %q, want %q", index, opened[index], want[index])
		}
	}
}

func TestInstallRecordAndRestoreSetup(t *testing.T) {
	app := &fakeApp{
		installFn: func(context.Context) (config.Setup, error) {
			return config.Setup{Status: config.SetupInstalled, Backup: config.Backup{Path: "/b", Meta: config.BackupMeta{OriginalValue: "orig"}}}, nil
		},
		recordFn: func(context.Context) (config.Setup, error) {
			return config.Setup{Status: config.SetupGuide, Backup: config.Backup{Path: "/b", Meta: config.BackupMeta{OriginalValue: "orig"}}}, nil
		},
		restoreSetupFn: func(context.Context) (config.Restored, error) {
			return config.Restored{Backup: config.Backup{Path: "/b", Meta: config.BackupMeta{OriginalValue: "orig"}}}, nil
		},
	}
	b := New(app, "dev", nil)
	installed, err := b.InstallSetup(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if installed.Status != "installed" || !installed.HasBackup || installed.OriginalValue != "orig" {
		t.Fatalf("installed = %+v", installed)
	}
	guide, err := b.RecordGuideSetup(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if guide.Status != "guide" {
		t.Fatalf("guide = %+v", guide)
	}
	restored, err := b.RestoreSetup(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if restored.Status != "restored" || restored.OriginalValue != "orig" {
		t.Fatalf("restored = %+v", restored)
	}
}

func TestVersion(t *testing.T) {
	if got := New(&fakeApp{}, "9.9.9", nil).Version(); got != "9.9.9" {
		t.Fatalf("Version() = %q", got)
	}
}

func TestSessionStateMapsProblemAndExitCode(t *testing.T) {
	exitCode := 7
	state := testState(domain.PhaseError)
	state.ErrorCode = "launch_failed"
	state.Error = "launch failed"
	state.ExitCode = &exitCode
	app := &fakeApp{stateFn: func(token string) (session.State, error) {
		if token != testToken() {
			t.Fatalf("token = %q", token)
		}
		return state, nil
	}}
	got, err := New(app, "dev", nil).SessionState(testToken())
	if err != nil {
		t.Fatal(err)
	}
	if got.Phase != "error" || got.Error == nil || got.Error.Code != "launch_failed" {
		t.Fatalf("state = %+v", got)
	}
	if got.ExitCode == nil || *got.ExitCode != 7 {
		t.Fatalf("exit code = %+v", got.ExitCode)
	}
	if got.CreatedAt == "" || got.StartedAt == "" {
		t.Fatalf("timestamps = %+v", got)
	}
}

func TestSaveDiagnosticsWritesPrivateReport(t *testing.T) {
	paths := config.Paths{DataDir: t.TempDir()}
	b := New(&fakeApp{paths: paths}, "0.2.0", nil)

	path, err := b.SaveDiagnostics("# Better RPT diagnostics\n")
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "# Better RPT diagnostics\n" {
		t.Fatalf("report = %q", data)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("report mode: %v %#v", err, info)
	}
	dir, err := os.Stat(filepath.Dir(path))
	if err != nil || dir.Mode().Perm() != 0o700 {
		t.Fatalf("diagnostics dir mode: %v %#v", err, dir)
	}
	if _, err := b.SaveDiagnostics(""); err == nil {
		t.Fatal("expected empty report error")
	}
}

func TestDispatcherCommand(t *testing.T) {
	app := &fakeApp{dispatcher: "'/helper' --donor -- %command%"}
	got, err := New(app, "dev", nil).DispatcherCommand()
	if err != nil {
		t.Fatal(err)
	}
	if got != "'/helper' --donor -- %command%" {
		t.Fatalf("command = %q", got)
	}
}

func waitForPhase(t *testing.T, b *Bridge, phase domain.Phase) <-chan struct{} {
	t.Helper()
	ready := make(chan struct{})
	go func() {
		defer close(ready)
		for {
			b.mu.Lock()
			current := b.run
			matches := current != nil && current.phase == phase
			b.mu.Unlock()
			if matches {
				return
			}
			time.Sleep(time.Millisecond)
		}
	}()
	return ready
}
