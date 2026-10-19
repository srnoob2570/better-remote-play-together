package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/srnoob2570/better-rpt/internal/config"
	"github.com/srnoob2570/better-rpt/internal/domain"
	"github.com/srnoob2570/better-rpt/internal/session"
)

type fakeSessionRunner struct {
	startOptions session.StartOptions
	startActive  session.Active
	startErr     error

	restoreToken string
	restoreErr   error

	recovery   *session.Recovery
	recoverErr error
}

func (f *fakeSessionRunner) Start(_ context.Context, options session.StartOptions) (session.Active, error) {
	f.startOptions = options
	return f.startActive, f.startErr
}

func (f *fakeSessionRunner) Restore(_ context.Context, token string) error {
	f.restoreToken = token
	return f.restoreErr
}

func (f *fakeSessionRunner) Recover(_ context.Context) (*session.Recovery, error) {
	return f.recovery, f.recoverErr
}

func TestSessionMethodsDelegate(t *testing.T) {
	fake := &fakeSessionRunner{
		startActive: session.Active{Token: "started"},
		recovery:    &session.Recovery{HelperAlive: true},
	}
	service := &Service{sessions: fake}

	active, err := service.StartSession(nil, session.StartOptions{TargetAppID: 900001, CloseSteam: true})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if active.Token != "started" || fake.startOptions.TargetAppID != 900001 || !fake.startOptions.CloseSteam {
		t.Fatalf("start active=%+v options=%+v", active, fake.startOptions)
	}

	if err := service.RestoreSession(context.Background(), "token-1"); err != nil {
		t.Fatalf("restore: %v", err)
	}
	if fake.restoreToken != "token-1" {
		t.Fatalf("restore token = %q", fake.restoreToken)
	}

	recovery, err := service.RecoverSession(context.Background())
	if err != nil {
		t.Fatalf("recover: %v", err)
	}
	if recovery == nil || !recovery.HelperAlive {
		t.Fatalf("recovery = %+v", recovery)
	}
}

func TestSessionMethodsReturnRunnerErrors(t *testing.T) {
	want := errors.New("session runner failed")
	fake := &fakeSessionRunner{startErr: want, restoreErr: want, recoverErr: want}
	service := &Service{sessions: fake}

	if _, err := service.StartSession(context.Background(), session.StartOptions{}); !errors.Is(err, want) {
		t.Fatalf("start error = %v", err)
	}
	if err := service.RestoreSession(context.Background(), "token"); !errors.Is(err, want) {
		t.Fatalf("restore error = %v", err)
	}
	if _, err := service.RecoverSession(context.Background()); !errors.Is(err, want) {
		t.Fatalf("recover error = %v", err)
	}
}

func TestSessionStateReadsStore(t *testing.T) {
	paths := config.Paths{DataDir: t.TempDir()}
	store := session.NewStore(paths.DataDir)
	if err := store.Ensure(); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	now := time.Now().UTC()
	plan := session.NewPlan("0123456789abcdef0123456789abcdef", 900001, domain.LaunchSpec{
		AppID: 900001, Kind: domain.KindNative, Executable: "/bin/true", WorkingDir: "/tmp",
	}, now)
	state := session.NewState(plan, domain.PhaseActive, now)
	state.HelperPID = 100
	state.TargetPID = 200
	if err := store.WriteState(state); err != nil {
		t.Fatalf("write state: %v", err)
	}

	service := &Service{paths: paths}
	got, err := service.SessionState(plan.Token)
	if err != nil {
		t.Fatalf("session state: %v", err)
	}
	if got.Phase != domain.PhaseActive || got.TargetPID != 200 {
		t.Fatalf("state = %+v", got)
	}
}
