package app

import (
	"context"

	"github.com/srnoob2570/better-rpt/internal/session"
)

// sessionRunner isolates the long-running session orchestration from the UI
// service. Tests inject a fake; production builds a Manager from the cached
// catalog.
type sessionRunner interface {
	Start(context.Context, session.StartOptions) (session.Active, error)
	Restore(context.Context, string) error
	Recover(context.Context) (*session.Recovery, error)
}

// StartSession validates the target, prepares SpaceWar and launches the donor
// context. The caller owns the confirmation behind CloseSteam.
func (s *Service) StartSession(ctx context.Context, options session.StartOptions) (session.Active, error) {
	runner, err := s.sessionRunner(ctx)
	if err != nil {
		return session.Active{}, err
	}
	return runner.Start(ctx, options)
}

// RestoreSession restores the launch option for a completed or failed session.
func (s *Service) RestoreSession(ctx context.Context, token string) error {
	runner, err := s.sessionRunner(ctx)
	if err != nil {
		return err
	}
	return runner.Restore(ctx, token)
}

// RecoverSession inspects durable session state at startup. It never starts a
// game; it only reports live helpers, terminalizes dead ones or reports a
// stale launch option.
func (s *Service) RecoverSession(ctx context.Context) (*session.Recovery, error) {
	runner, err := s.sessionRunner(ctx)
	if err != nil {
		return nil, err
	}
	return runner.Recover(ctx)
}

// SessionState reads the durable state for an active or completed session.
func (s *Service) SessionState(token string) (session.State, error) {
	return session.NewStore(s.paths.DataDir).ReadState(token)
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
