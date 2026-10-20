package session

import (
	"fmt"
	"time"

	"github.com/srnoob2570/better-rpt/internal/domain"
)

// LastLaunchVersion is the durable last-launch schema version.
const LastLaunchVersion = 1

// LaunchOutcome is the durable result of the most recent launch attempt.
type LaunchOutcome string

const (
	LaunchPending  LaunchOutcome = "pending"
	LaunchOK       LaunchOutcome = "ok"
	LaunchError    LaunchOutcome = "error"
	LaunchTimeout  LaunchOutcome = "timeout"
	LaunchCanceled LaunchOutcome = "canceled"
	LaunchUnknown  LaunchOutcome = "unknown"
)

// LaunchSource identifies who initiated the most recent launch.
type LaunchSource string

const (
	SourceBetterRPT LaunchSource = "better-rpt"
	SourceSteam     LaunchSource = "steam"
)

// LastLaunch is the durable record shown as "Last session". It is independent
// from restore state and survives app restarts.
type LastLaunch struct {
	Version    int           `json:"version"`
	AppID      uint32        `json:"app_id"`
	Name       string        `json:"name,omitempty"`
	Token      string        `json:"token,omitempty"`
	Parameters []string      `json:"parameters,omitempty"`
	Kind       string        `json:"kind,omitempty"`
	Source     LaunchSource  `json:"source"`
	StartedAt  time.Time     `json:"started_at"`
	EndedAt    time.Time     `json:"ended_at,omitempty"`
	Outcome    LaunchOutcome `json:"outcome"`
	Error      string        `json:"error,omitempty"`
}

// Validate checks the durable shape of a last-launch record.
func (l LastLaunch) Validate() error {
	if l.Version != LastLaunchVersion {
		return fmt.Errorf("%w: version %d", ErrLastLaunchInvalid, l.Version)
	}
	if l.AppID == 0 || l.AppID == domain.SpaceWarAppID {
		return fmt.Errorf("%w: app %d", ErrLastLaunchInvalid, l.AppID)
	}
	if l.StartedAt.IsZero() {
		return fmt.Errorf("%w: missing started_at", ErrLastLaunchInvalid)
	}
	switch l.Source {
	case SourceBetterRPT, SourceSteam:
	default:
		return fmt.Errorf("%w: source %q", ErrLastLaunchInvalid, l.Source)
	}
	if l.Source == SourceBetterRPT && !domain.ValidSessionToken(l.Token) {
		return fmt.Errorf("%w: token", ErrLastLaunchInvalid)
	}
	if l.Source == SourceSteam && l.Token != "" {
		return fmt.Errorf("%w: steam launch with token", ErrLastLaunchInvalid)
	}
	switch l.Outcome {
	case LaunchPending, LaunchOK, LaunchError, LaunchTimeout, LaunchCanceled, LaunchUnknown:
	default:
		return fmt.Errorf("%w: outcome %q", ErrLastLaunchInvalid, l.Outcome)
	}
	return nil
}

// Started reports whether the target actually started.
func (l LastLaunch) Started() bool {
	return !l.StartedAt.IsZero()
}
