package app

import (
	"context"

	"github.com/srnoob2570/better-rpt/internal/config"
	"github.com/srnoob2570/better-rpt/internal/session"
)

// sessionRunner isolates the long-running session orchestration from the UI
// service. Tests inject a fake; production builds a Manager from the cached
// catalog.
type sessionRunner interface {
	Start(context.Context, session.StartOptions) (session.Active, error)
	Replay(context.Context) (session.Active, error)
	Restore(context.Context, string) error
	Recover(context.Context) (*session.Recovery, error)
	Situation() (session.Situation, error)
	SetupSnapshot() (config.SetupSnapshot, error)
	InstallSetup(context.Context) (config.Setup, error)
	RecordGuideSetup(context.Context) (config.Setup, error)
	RestoreSetup(context.Context) (config.Restored, error)
}

// StartSession validates the target, arms the dispatcher intent and asks Steam
// to launch SpaceWar. Steam may remain open; Better RPT never closes it.
func (s *Service) StartSession(ctx context.Context, options session.StartOptions) (session.Active, error) {
	runner, err := s.sessionRunner(ctx)
	if err != nil {
		return session.Active{}, err
	}
	return runner.Start(ctx, options)
}

// ReplaySession repeats the durable last launch with the same parameters.
func (s *Service) ReplaySession(ctx context.Context) (session.Active, error) {
	runner, err := s.sessionRunner(ctx)
	if err != nil {
		return session.Active{}, err
	}
	return runner.Replay(ctx)
}

// RestoreSession restores the legacy per-session launch option.
func (s *Service) RestoreSession(ctx context.Context, token string) error {
	runner, err := s.sessionRunner(ctx)
	if err != nil {
		return err
	}
	return runner.Restore(ctx, token)
}

// RecoverSession inspects durable state at startup. It never starts a game.
func (s *Service) RecoverSession(ctx context.Context) (*session.Recovery, error) {
	runner, err := s.sessionRunner(ctx)
	if err != nil {
		return nil, err
	}
	return runner.Recover(ctx)
}

// SessionSituation classifies the current launch and restore state.
func (s *Service) SessionSituation() (session.Situation, error) {
	runner, err := s.sessionRunner(context.Background())
	if err != nil {
		return session.Situation{}, err
	}
	return runner.Situation()
}

// SetupSnapshot reads the persistent dispatcher setup without writing.
func (s *Service) SetupSnapshot() (config.SetupSnapshot, error) {
	runner, err := s.sessionRunner(context.Background())
	if err != nil {
		return config.SetupSnapshot{}, err
	}
	return runner.SetupSnapshot()
}

// InstallSetup installs the dispatcher; Steam must be closed.
func (s *Service) InstallSetup(ctx context.Context) (config.Setup, error) {
	runner, err := s.sessionRunner(ctx)
	if err != nil {
		return config.Setup{}, err
	}
	return runner.InstallSetup(ctx)
}

// RecordGuideSetup records a manual dispatcher installation.
func (s *Service) RecordGuideSetup(ctx context.Context) (config.Setup, error) {
	runner, err := s.sessionRunner(ctx)
	if err != nil {
		return config.Setup{}, err
	}
	return runner.RecordGuideSetup(ctx)
}

// RestoreSetup removes the dispatcher; Steam must be closed.
func (s *Service) RestoreSetup(ctx context.Context) (config.Restored, error) {
	runner, err := s.sessionRunner(ctx)
	if err != nil {
		return config.Restored{}, err
	}
	return runner.RestoreSetup(ctx)
}

// SessionState reads the durable state for an active or completed session.
func (s *Service) SessionState(token string) (session.State, error) {
	return session.NewStore(s.paths.DataDir).ReadState(token)
}

// DispatcherCommand returns the exact SpaceWar launch option for this install.
func (s *Service) DispatcherCommand() (string, error) {
	return config.BuildDispatcherLaunchOption(s.paths.HelperPath())
}

func (s *Service) sessionRunner(ctx context.Context) (sessionRunner, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if s.sessions != nil {
		return s.sessions, nil
	}
	catalog, ok := s.Catalog()
	if !ok {
		refreshed, err := s.RefreshCatalog(ctx)
		if err != nil {
			return nil, err
		}
		catalog = refreshed
	}
	return session.NewManager(s.paths, catalog)
}
