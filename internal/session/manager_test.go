package session

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/srnoob2570/better-rpt/internal/config"
	"github.com/srnoob2570/better-rpt/internal/domain"
	"github.com/srnoob2570/better-rpt/internal/helper"
	"github.com/srnoob2570/better-rpt/internal/steam"
)

func managerSource(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "better-rpt-helper")
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := os.Chmod(path, 0o755); err != nil {
		t.Fatalf("chmod source: %v", err)
	}
	return path
}

func testManager(t *testing.T) *Manager {
	t.Helper()
	paths := config.Paths{DataDir: t.TempDir()}
	spec := testSpec(900001)
	specB := testSpec(900002)
	donorSpec := testSpec(480)
	manager := &Manager{
		Paths: paths,
		Store: NewStore(paths.DataDir),
		Catalog: steam.Catalog{
			Installation: steam.Installation{Root: t.TempDir()},
			Games: []domain.Game{
				{AppID: 480, Name: "Spacewar", Status: domain.StatusUnknown, Launch: &donorSpec},
				{AppID: 900001, Name: "Game", Status: domain.StatusReady, Launch: &spec},
				{AppID: 900002, Name: "Game B", Status: domain.StatusReady, Launch: &specB},
			},
		},
		Lifecycle: &steam.Lifecycle{
			ProcRoot:     t.TempDir(),
			RegistryPath: filepath.Join(t.TempDir(), "missing-registry.vdf"),
			SteamBinary:  "/bin/true",
			StartCommand: func(string, ...string) error {
				t.Fatal("manager must not start Steam directly in tests")
				return nil
			},
		},
		Installer: &helper.Installer{
			DataDir:    paths.DataDir,
			SystemPath: managerSource(t),
		},
		SetupDetect: func(config.Paths, string) (config.SetupSnapshot, error) {
			return config.SetupSnapshot{Status: config.SetupInstalled}, nil
		},
		SetupInstall: func(config.InstallSetupOptions) (config.Setup, error) {
			return config.Setup{Status: config.SetupInstalled}, nil
		},
		SetupGuide: func(config.GuideSetupOptions) (config.Setup, error) {
			return config.Setup{Status: config.SetupGuide}, nil
		},
		SetupRestore: func(config.RestoreSetupOptions) (config.Restored, error) {
			return config.Restored{}, nil
		},
		RestoreFn: func(config.RestoreOptions) (config.Restored, error) {
			return config.Restored{}, nil
		},
		Now:            time.Now,
		PollInterval:   time.Millisecond,
		StartupTimeout: 100 * time.Millisecond,
	}
	manager.LaunchSession = activeLauncher(t, manager)
	return manager
}

func activeLauncher(t *testing.T, manager *Manager) LaunchSessionFunc {
	t.Helper()
	start := helperStart(t)
	return func(_ context.Context, appID uint32, token string) error {
		if appID != domain.SpaceWarAppID {
			t.Fatalf("launched app = %d, want 480", appID)
		}
		plan, err := manager.Store.ReadPlan(token, time.Now())
		if err != nil {
			return err
		}
		state := NewState(plan, domain.PhaseActive, time.Now())
		state.HelperPID = os.Getpid()
		state.HelperStartTime = start
		state.TargetPID = 12345
		state.StartedAt = time.Now()
		return manager.Store.WriteState(state)
	}
}

func writeManagerActive(t *testing.T, manager *Manager, helperPID int) State {
	t.Helper()
	plan := NewPlan(testToken, 900001, testSpec(900001), time.Now())
	if err := manager.Store.WritePlan(plan); err != nil {
		t.Fatalf("write plan: %v", err)
	}
	state := NewState(plan, domain.PhaseActive, time.Now())
	state.HelperPID = helperPID
	state.HelperStartTime = helperStart(t)
	state.TargetPID = 12345
	state.StartedAt = time.Now()
	if err := manager.Store.WriteState(state); err != nil {
		t.Fatalf("write state: %v", err)
	}
	return state
}

func TestManagerStartReportsPhasesAndWritesLastLaunch(t *testing.T) {
	manager := testManager(t)
	var phases []domain.Phase
	active, err := manager.Start(context.Background(), StartOptions{
		TargetAppID: 900001,
		OnState: func(state State) {
			phases = append(phases, state.Phase)
		},
	})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if len(phases) < 2 || phases[0] != domain.PhasePreparing || phases[len(phases)-1] != domain.PhaseActive {
		t.Fatalf("phases = %v", phases)
	}
	if active.Token == "" || active.Plan.Token != active.Token {
		t.Fatalf("active token = %q", active.Token)
	}
	if active.State.Phase != domain.PhaseActive || active.State.TargetPID != 12345 {
		t.Fatalf("state = %+v", active.State)
	}
	last, err := manager.Store.ReadLastLaunch()
	if err != nil {
		t.Fatalf("read last: %v", err)
	}
	if last.AppID != 900001 || last.Token != active.Token || last.Outcome != LaunchOK || last.Source != SourceBetterRPT {
		t.Fatalf("last = %+v", last)
	}
	if _, err := manager.Store.ReadIntent(time.Now()); !errors.Is(err, ErrIntentMissing) {
		t.Fatalf("intent after active = %v, want removed", err)
	}
}

func TestManagerStartNeverShutsDownSteam(t *testing.T) {
	manager := testManager(t)
	// The fake lifecycle panics on StartCommand. The launch itself is
	// an HTTP-like URL request injected through LaunchSession.
	manager.LaunchSession = func(_ context.Context, _ uint32, token string) error {
		return activeLauncher(t, manager)(context.Background(), domain.SpaceWarAppID, token)
	}
	if _, err := manager.Start(context.Background(), StartOptions{TargetAppID: 900001}); err != nil {
		t.Fatalf("start: %v", err)
	}
}

func TestManagerStartRequiresInstalledSetup(t *testing.T) {
	manager := testManager(t)
	manager.SetupDetect = func(config.Paths, string) (config.SetupSnapshot, error) {
		return config.SetupSnapshot{Status: config.SetupGuide}, nil
	}
	launched := false
	manager.LaunchSession = func(context.Context, uint32, string) error {
		launched = true
		return nil
	}
	if _, err := manager.Start(context.Background(), StartOptions{TargetAppID: 900001}); !errors.Is(err, config.ErrSetupMissing) {
		t.Fatalf("error = %v, want ErrSetupMissing", err)
	}
	if launched {
		t.Fatal("donor launched without an installed dispatcher")
	}
}

func TestManagerStartRefusesSetupConflict(t *testing.T) {
	manager := testManager(t)
	manager.SetupDetect = func(config.Paths, string) (config.SetupSnapshot, error) {
		return config.SetupSnapshot{Status: config.SetupConflict}, nil
	}
	if _, err := manager.Start(context.Background(), StartOptions{TargetAppID: 900001}); !errors.Is(err, config.ErrSetupConflict) {
		t.Fatalf("error = %v, want ErrSetupConflict", err)
	}
}

func TestManagerStartLaunchFailureClearsIntentAndLast(t *testing.T) {
	manager := testManager(t)
	manager.LaunchSession = func(context.Context, uint32, string) error {
		return errors.New("launch failed")
	}
	active, err := manager.Start(context.Background(), StartOptions{TargetAppID: 900001})
	if err == nil {
		t.Fatal("launch failure returned nil")
	}
	if active.Token == "" {
		t.Fatal("failure lost the token")
	}
	if _, err := manager.Store.ReadIntent(time.Now()); !errors.Is(err, ErrIntentMissing) {
		t.Fatalf("intent after failure = %v, want removed", err)
	}
	last, lastErr := manager.Store.ReadLastLaunch()
	if lastErr != nil {
		t.Fatalf("read last: %v", lastErr)
	}
	if last.Outcome != LaunchError || last.Error == "" || !last.EndedAt.After(last.StartedAt) {
		t.Fatalf("last = %+v, want terminal error", last)
	}
	state, stateErr := manager.Store.ReadState(active.Token)
	if stateErr != nil {
		t.Fatalf("read state: %v", stateErr)
	}
	if state.Phase != domain.PhaseError || state.ErrorCode != "launch_failed" {
		t.Fatalf("state = %+v", state)
	}
}

func TestManagerStartTimeoutClearsIntentAndLast(t *testing.T) {
	manager := testManager(t)
	manager.StartupTimeout = 5 * time.Millisecond
	manager.LaunchSession = func(context.Context, uint32, string) error { return nil }
	active, err := manager.Start(context.Background(), StartOptions{TargetAppID: 900001})
	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("error = %v, want ErrTimeout", err)
	}
	if _, err := manager.Store.ReadIntent(time.Now()); !errors.Is(err, ErrIntentMissing) {
		t.Fatalf("intent after timeout = %v, want removed", err)
	}
	last, lastErr := manager.Store.ReadLastLaunch()
	if lastErr != nil {
		t.Fatalf("read last: %v", lastErr)
	}
	if last.Outcome != LaunchTimeout || last.Token != active.Token {
		t.Fatalf("last = %+v, want timeout for %s", last, active.Token)
	}
	state, stateErr := manager.Store.ReadState(active.Token)
	if stateErr != nil || state.Phase != domain.PhaseError || state.ErrorCode != "startup_timeout" {
		t.Fatalf("state = %+v, err=%v", state, stateErr)
	}
}

func TestManagerStartTerminalState(t *testing.T) {
	manager := testManager(t)
	manager.LaunchSession = func(_ context.Context, _ uint32, token string) error {
		plan, err := manager.Store.ReadPlan(token, time.Now())
		if err != nil {
			return err
		}
		state := NewState(plan, domain.PhaseError, time.Now())
		state.Outcome = "error"
		state.ErrorCode = "target_exited_early"
		state.Error = "target exited early"
		state.TargetPID = 12345
		state.EndedAt = time.Now()
		return manager.Store.WriteState(state)
	}
	active, err := manager.Start(context.Background(), StartOptions{TargetAppID: 900001})
	if !errors.Is(err, ErrTargetExited) {
		t.Fatalf("error = %v, want ErrTargetExited", err)
	}
	if active.State.ErrorCode != "target_exited_early" {
		t.Fatalf("active = %+v", active)
	}
	last, lastErr := manager.Store.ReadLastLaunch()
	if lastErr != nil || last.Outcome != LaunchError {
		t.Fatalf("last = %+v, err=%v", last, lastErr)
	}
}

func TestManagerStartCancelClearsIntent(t *testing.T) {
	manager := testManager(t)
	ctx, cancel := context.WithCancel(context.Background())
	manager.LaunchSession = func(context.Context, uint32, string) error {
		cancel()
		return nil
	}
	active, err := manager.Start(ctx, StartOptions{TargetAppID: 900001})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	if _, err := manager.Store.ReadIntent(time.Now()); !errors.Is(err, ErrIntentMissing) {
		t.Fatalf("intent after cancel = %v, want removed", err)
	}
	if active.Token == "" {
		t.Fatal("cancel lost token")
	}
}

func TestManagerStartRefusesActiveSession(t *testing.T) {
	manager := testManager(t)
	writeManagerActive(t, manager, os.Getpid())
	launched := false
	manager.LaunchSession = func(context.Context, uint32, string) error {
		launched = true
		return nil
	}
	if _, err := manager.Start(context.Background(), StartOptions{TargetAppID: 900001}); !errors.Is(err, ErrSessionBusy) {
		t.Fatalf("error = %v, want ErrSessionBusy", err)
	}
	if launched {
		t.Fatal("donor launched while session was active")
	}
}

func TestManagerReplayUsesLastApp(t *testing.T) {
	manager := testManager(t)
	if _, err := manager.Start(context.Background(), StartOptions{TargetAppID: 900001}); err != nil {
		t.Fatalf("first start: %v", err)
	}
	last, err := manager.Store.ReadLastLaunch()
	if err != nil {
		t.Fatalf("read last: %v", err)
	}
	state, err := manager.Store.ReadState(last.Token)
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	state.Phase = domain.PhaseEnded
	state.Outcome = "ok"
	state.EndedAt = time.Now()
	if err := manager.Store.WriteState(state); err != nil {
		t.Fatalf("end state: %v", err)
	}
	active, err := manager.Replay(context.Background())
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if active.Game.AppID != 900001 || active.Token == "" {
		t.Fatalf("replay = %+v", active)
	}
}

func TestManagerReplayRejectsChangedParameters(t *testing.T) {
	manager := testManager(t)
	if _, err := manager.Start(context.Background(), StartOptions{TargetAppID: 900001}); err != nil {
		t.Fatalf("start: %v", err)
	}
	last, err := manager.Store.ReadLastLaunch()
	if err != nil {
		t.Fatalf("read last: %v", err)
	}
	state, err := manager.Store.ReadState(last.Token)
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	state.Phase = domain.PhaseEnded
	state.Outcome = "ok"
	state.EndedAt = time.Now()
	if err := manager.Store.WriteState(state); err != nil {
		t.Fatalf("end state: %v", err)
	}
	last.Parameters = append(last.Parameters, "--different")
	if err := manager.Store.WriteLastLaunch(*last); err != nil {
		t.Fatalf("write last: %v", err)
	}
	if _, err := manager.Replay(context.Background()); !errors.Is(err, domain.ErrInvalidLaunchSpec) {
		t.Fatalf("error = %v, want ErrInvalidLaunchSpec", err)
	}
}

func TestManagerRestoreRefusesLiveHelper(t *testing.T) {
	manager := testManager(t)
	writeManagerActive(t, manager, os.Getpid())
	if err := manager.Restore(context.Background(), testToken); !errors.Is(err, ErrSessionBusy) {
		t.Fatalf("error = %v, want ErrSessionBusy", err)
	}
}

func TestManagerRestoreDeadHelperDoesNotShutdownSteam(t *testing.T) {
	manager := testManager(t)
	writeManagerActive(t, manager, 999999)
	restored := 0
	manager.RestoreFn = func(config.RestoreOptions) (config.Restored, error) {
		restored++
		return config.Restored{}, nil
	}
	if err := manager.Restore(context.Background(), testToken); err != nil {
		t.Fatalf("restore: %v", err)
	}
	if restored != 1 {
		t.Fatalf("restore calls = %d, want 1", restored)
	}
}

func TestManagerRestoreSteamRunningIsReportedWithoutShutdown(t *testing.T) {
	manager := testManager(t)
	writeManagerActive(t, manager, 999999)
	manager.RestoreFn = func(config.RestoreOptions) (config.Restored, error) {
		return config.Restored{}, domain.ErrSteamRunning
	}
	if err := manager.Restore(context.Background(), testToken); !errors.Is(err, domain.ErrSteamRunning) {
		t.Fatalf("error = %v, want ErrSteamRunning", err)
	}
}

func TestManagerRestoreFailureMarksState(t *testing.T) {
	manager := testManager(t)
	writeManagerActive(t, manager, 999999)
	manager.RestoreFn = func(config.RestoreOptions) (config.Restored, error) {
		return config.Restored{}, errors.New("restore boom")
	}
	if err := manager.Restore(context.Background(), testToken); err == nil {
		t.Fatal("restore failure returned nil")
	}
	state, err := manager.Store.ReadState(testToken)
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	if state.Phase != domain.PhaseError || state.ErrorCode != "restore_failed" {
		t.Fatalf("state = %+v, want terminal restore_failed", state)
	}
}

func TestManagerRecoverRemovesExpiredPlan(t *testing.T) {
	manager := testManager(t)
	manager.Catalog.Installation.Root = ""
	plan := NewPlan(testToken, 900001, testSpec(900001), time.Now())
	plan.ExpiresAt = time.Now().Add(-time.Minute)
	data, err := json.Marshal(plan)
	if err != nil {
		t.Fatalf("marshal plan: %v", err)
	}
	if err := manager.Store.Ensure(); err != nil {
		t.Fatalf("ensure store: %v", err)
	}
	if err := os.WriteFile(manager.Store.PlanPath(testToken), data, 0o600); err != nil {
		t.Fatalf("write expired plan: %v", err)
	}
	recovery, err := manager.Recover(context.Background())
	if err != nil {
		t.Fatalf("recover: %v", err)
	}
	if recovery == nil || len(recovery.ExpiredPlans) != 1 || recovery.ExpiredPlans[0] != testToken {
		t.Fatalf("recovery = %+v, want expired plan %s", recovery, testToken)
	}
	if _, err := manager.Store.ReadPlan(testToken, time.Now()); !errors.Is(err, ErrPlanMissing) {
		t.Fatalf("expired plan still present: %v", err)
	}
}

func TestManagerRecoverLiveAndDead(t *testing.T) {
	manager := testManager(t)
	live := writeManagerActive(t, manager, os.Getpid())
	recovery, err := manager.Recover(context.Background())
	if err != nil {
		t.Fatalf("recover live: %v", err)
	}
	if recovery == nil || !recovery.HelperAlive || recovery.State == nil || recovery.State.Token != live.Token {
		t.Fatalf("live recovery = %+v", recovery)
	}

	dead := writeManagerActive(t, manager, 999999)
	recovery, err = manager.Recover(context.Background())
	if err != nil {
		t.Fatalf("recover dead: %v", err)
	}
	if recovery == nil || recovery.HelperAlive || recovery.State == nil {
		t.Fatalf("dead recovery = %+v", recovery)
	}
	if recovery.State.Phase != domain.PhaseError || recovery.State.ErrorCode != "helper_dead" {
		t.Fatalf("dead state = %+v", recovery.State)
	}
	loaded, err := manager.Store.ReadState(testToken)
	if err != nil || loaded.Phase != domain.PhaseError || loaded.ErrorCode != "helper_dead" {
		t.Fatalf("persisted dead state = %+v, err=%v", loaded, err)
	}
	_ = dead
}

func TestManagerSituationKeepsNewestLaunchSeparateFromOldSession(t *testing.T) {
	manager := testManager(t)
	manager.Catalog.Installation.Root = ""
	// A terminal session for A.
	writeManagerActive(t, manager, 999999)
	state, err := manager.Store.ReadState(testToken)
	if err != nil {
		t.Fatalf("read A state: %v", err)
	}
	state.Phase = domain.PhaseEnded
	state.Outcome = "ok"
	if err := manager.Store.WriteState(state); err != nil {
		t.Fatalf("write A state: %v", err)
	}
	// B is the newest launch.
	lastB := LastLaunch{
		Version:   LastLaunchVersion,
		AppID:     900002,
		Name:      "B",
		Token:     "fedcba9876543210fedcba9876543210",
		Source:    SourceBetterRPT,
		StartedAt: time.Now().UTC(),
		Outcome:   LaunchOK,
	}
	if err := manager.Store.WriteLastLaunch(lastB); err != nil {
		t.Fatalf("write B: %v", err)
	}
	situation, err := manager.Situation()
	if err != nil {
		t.Fatalf("situation: %v", err)
	}
	if situation.Last == nil || situation.Last.AppID != 900002 {
		t.Fatalf("last = %+v, want B", situation.Last)
	}
	if situation.Kind == SituationRestorePending {
		t.Fatalf("old session A leaked as pending: %+v", situation)
	}
}

func TestManagerSituationReconcilesNewerSteamLaunch(t *testing.T) {
	manager := testManager(t)
	now := time.Now().UTC()
	lastA := LastLaunch{
		Version:   LastLaunchVersion,
		AppID:     900001,
		Name:      "Game",
		Token:     testToken,
		Source:    SourceBetterRPT,
		StartedAt: now.Add(-time.Hour),
		Outcome:   LaunchOK,
	}
	if err := manager.Store.WriteLastLaunch(lastA); err != nil {
		t.Fatalf("write A: %v", err)
	}
	manager.LastPlayed = func(string) (*config.LastPlayed, error) {
		return &config.LastPlayed{AppID: 900002, At: now}, nil
	}
	situation, err := manager.Situation()
	if err != nil {
		t.Fatalf("situation: %v", err)
	}
	if situation.Last == nil || situation.Last.AppID != 900002 || situation.Last.Source != SourceSteam {
		t.Fatalf("last = %+v, want newest Steam game B", situation.Last)
	}
	stored, err := manager.Store.ReadLastLaunch()
	if err != nil {
		t.Fatalf("read stored last: %v", err)
	}
	if stored.AppID != 900002 || stored.Source != SourceSteam {
		t.Fatalf("stored = %+v, want B from Steam", stored)
	}
	if situation.Kind == SituationRestorePending {
		t.Fatalf("Steam launch leaked as restore pending: %+v", situation)
	}
}

func TestManagerSituationKeepsBetterRPTRecordForSameApp(t *testing.T) {
	manager := testManager(t)
	now := time.Now().UTC()
	lastB := LastLaunch{
		Version:    LastLaunchVersion,
		AppID:      900002,
		Name:       "Game B",
		Token:      testToken,
		Parameters: []string{"--windowed"},
		Source:     SourceBetterRPT,
		StartedAt:  now.Add(-time.Minute),
		Outcome:    LaunchOK,
	}
	if err := manager.Store.WriteLastLaunch(lastB); err != nil {
		t.Fatalf("write B: %v", err)
	}
	manager.LastPlayed = func(string) (*config.LastPlayed, error) {
		return &config.LastPlayed{AppID: 900002, At: now}, nil
	}
	situation, err := manager.Situation()
	if err != nil {
		t.Fatalf("situation: %v", err)
	}
	if situation.Last == nil || situation.Last.Token != testToken || len(situation.Last.Parameters) != 1 {
		t.Fatalf("last = %+v, want richer Better RPT record", situation.Last)
	}
}

func TestManagerSetupHelpersDoNotShutdownSteam(t *testing.T) {
	manager := testManager(t)
	called := false
	manager.SetupInstall = func(config.InstallSetupOptions) (config.Setup, error) {
		called = true
		return config.Setup{Status: config.SetupInstalled}, nil
	}
	if _, err := manager.InstallSetup(context.Background()); err != nil {
		t.Fatalf("install setup: %v", err)
	}
	if !called {
		t.Fatal("setup install was not delegated")
	}
}
