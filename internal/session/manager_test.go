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
	donorSpec := testSpec(480)
	manager := &Manager{
		Paths: paths,
		Store: NewStore(paths.DataDir),
		Catalog: steam.Catalog{
			Installation: steam.Installation{Root: t.TempDir()},
			Games: []domain.Game{
				{AppID: 480, Name: "Spacewar", Status: domain.StatusUnknown, Launch: &donorSpec},
				{AppID: 900001, Name: "Game", Status: domain.StatusReady, Launch: &spec},
			},
		},
		Lifecycle: &steam.Lifecycle{
			ProcRoot:     t.TempDir(),
			RegistryPath: filepath.Join(t.TempDir(), "missing-registry.vdf"),
			SteamBinary:  "/bin/true",
		},
		Installer: &helper.Installer{
			DataDir:    paths.DataDir,
			SystemPath: managerSource(t),
		},
		PrepareFn:      func(config.PrepareOptions) (config.Prepared, error) { return config.Prepared{}, nil },
		RestoreFn:      func(config.RestoreOptions) (config.Restored, error) { return config.Restored{}, nil },
		Launch:         func(context.Context, uint32) error { return nil },
		Now:            time.Now,
		PollInterval:   time.Millisecond,
		StartupTimeout: 100 * time.Millisecond,
	}
	return manager
}

func activePreparer(t *testing.T, manager *Manager) PrepareFunc {
	t.Helper()
	start := helperStart(t)
	return func(options config.PrepareOptions) (config.Prepared, error) {
		plan, err := manager.Store.ReadPlan(options.SessionToken, time.Now())
		if err != nil {
			return config.Prepared{}, err
		}
		state := NewState(plan, domain.PhaseActive, time.Now())
		state.HelperPID = os.Getpid()
		state.HelperStartTime = start
		state.TargetPID = 12345
		state.StartedAt = time.Now()
		if err := manager.Store.WriteState(state); err != nil {
			return config.Prepared{}, err
		}
		return config.Prepared{}, nil
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

func TestManagerStartReportsPhases(t *testing.T) {
	manager := testManager(t)
	manager.PrepareFn = activePreparer(t, manager)
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
	_ = active
	if len(phases) < 2 || phases[0] != domain.PhasePreparing || phases[len(phases)-1] != domain.PhaseActive {
		t.Fatalf("phases = %v", phases)
	}
}

func TestManagerStartSuccess(t *testing.T) {
	manager := testManager(t)
	manager.PrepareFn = activePreparer(t, manager)
	var launched uint32
	manager.Launch = func(_ context.Context, appID uint32) error {
		launched = appID
		return nil
	}
	active, err := manager.Start(context.Background(), StartOptions{TargetAppID: 900001})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if active.Token == "" || active.Plan.Token != active.Token {
		t.Fatalf("active token = %q", active.Token)
	}
	if active.State.Phase != domain.PhaseActive || active.State.TargetPID != 12345 {
		t.Fatalf("state = %+v", active.State)
	}
	if launched != domain.SpaceWarAppID {
		t.Fatalf("launched = %d, want 480", launched)
	}
	if _, err := os.Stat(manager.Installer.Destination()); err != nil {
		t.Fatalf("helper not installed: %v", err)
	}
}

func TestManagerStartPrepareFailure(t *testing.T) {
	manager := testManager(t)
	manager.PrepareFn = func(config.PrepareOptions) (config.Prepared, error) {
		return config.Prepared{}, errors.New("prepare failed")
	}
	launched := false
	manager.Launch = func(context.Context, uint32) error {
		launched = true
		return nil
	}
	if _, err := manager.Start(context.Background(), StartOptions{TargetAppID: 900001}); err == nil {
		t.Fatal("prepare failure returned nil")
	}
	if launched {
		t.Fatal("donor launched after prepare failure")
	}
	if active, err := manager.Store.ActiveState(); err != nil || active != nil {
		t.Fatalf("active state after failure = %+v, err=%v", active, err)
	}
}

func TestManagerStartLaunchFailureRestores(t *testing.T) {
	manager := testManager(t)
	manager.PrepareFn = func(config.PrepareOptions) (config.Prepared, error) {
		return config.Prepared{}, nil
	}
	manager.Launch = func(context.Context, uint32) error { return errors.New("launch failed") }
	restored := 0
	manager.RestoreFn = func(config.RestoreOptions) (config.Restored, error) {
		restored++
		return config.Restored{}, nil
	}
	if _, err := manager.Start(context.Background(), StartOptions{TargetAppID: 900001}); err == nil {
		t.Fatal("launch failure returned nil")
	}
	if restored != 1 {
		t.Fatalf("restore calls = %d, want 1", restored)
	}
	if active, err := manager.Store.ActiveState(); err != nil || active != nil {
		t.Fatalf("active state after launch failure = %+v, err=%v", active, err)
	}
}

func TestManagerStartTimeoutRestores(t *testing.T) {
	manager := testManager(t)
	manager.StartupTimeout = 5 * time.Millisecond
	manager.PollInterval = time.Millisecond
	manager.PrepareFn = func(config.PrepareOptions) (config.Prepared, error) {
		return config.Prepared{}, nil
	}
	restored := 0
	manager.RestoreFn = func(config.RestoreOptions) (config.Restored, error) {
		restored++
		return config.Restored{}, nil
	}
	active, err := manager.Start(context.Background(), StartOptions{TargetAppID: 900001})
	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("error = %v, want ErrTimeout", err)
	}
	if restored != 1 {
		t.Fatalf("restore calls = %d, want 1", restored)
	}
	state, stateErr := manager.Store.ReadState(active.Token)
	if stateErr != nil {
		t.Fatalf("read timeout state: %v", stateErr)
	}
	if state.Phase != domain.PhaseError || state.ErrorCode != "startup_timeout" {
		t.Fatalf("timeout state = %+v, want terminal startup_timeout", state)
	}
}

func TestManagerStartTerminalStateRestores(t *testing.T) {
	manager := testManager(t)
	manager.PrepareFn = func(options config.PrepareOptions) (config.Prepared, error) {
		plan, err := manager.Store.ReadPlan(options.SessionToken, time.Now())
		if err != nil {
			return config.Prepared{}, err
		}
		state := NewState(plan, domain.PhaseError, time.Now())
		state.Outcome = "error"
		state.ErrorCode = "target_exited_early"
		state.Error = "target exited early"
		state.TargetPID = 12345
		state.EndedAt = time.Now()
		if err := manager.Store.WriteState(state); err != nil {
			return config.Prepared{}, err
		}
		return config.Prepared{}, nil
	}
	restored := 0
	manager.RestoreFn = func(config.RestoreOptions) (config.Restored, error) {
		restored++
		return config.Restored{}, nil
	}
	_, err := manager.Start(context.Background(), StartOptions{TargetAppID: 900001})
	if !errors.Is(err, ErrTargetExited) {
		t.Fatalf("error = %v, want ErrTargetExited", err)
	}
	if restored != 1 {
		t.Fatalf("restore calls = %d, want 1", restored)
	}
}

func TestManagerStartRefusesActiveSession(t *testing.T) {
	manager := testManager(t)
	writeManagerActive(t, manager, os.Getpid())
	launched := false
	manager.Launch = func(context.Context, uint32) error {
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

func TestManagerRestoreRefusesLiveHelper(t *testing.T) {
	manager := testManager(t)
	writeManagerActive(t, manager, os.Getpid())
	if err := manager.Restore(context.Background(), testToken); !errors.Is(err, ErrSessionBusy) {
		t.Fatalf("error = %v, want ErrSessionBusy", err)
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

func TestManagerRestoreDeadHelper(t *testing.T) {
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
	if _, err := manager.Store.ReadPlan(testToken, time.Now()); !errors.Is(err, ErrPlanMissing) {
		t.Fatalf("plan error = %v, want ErrPlanMissing", err)
	}
	if _, err := manager.Store.ReadState(testToken); !errors.Is(err, ErrStateMissing) {
		t.Fatalf("state error = %v, want ErrStateMissing", err)
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
