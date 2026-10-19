package session

import (
	"fmt"
	"time"

	"github.com/srnoob2570/better-rpt/internal/domain"
)

// StateVersion is the durable state schema version.
const StateVersion = 1

// State is the helper-owned durable snapshot polled by the app.
type State struct {
	Version         int          `json:"version"`
	Token           string       `json:"token"`
	DonorAppID      uint32       `json:"donor_app_id"`
	TargetAppID     uint32       `json:"target_app_id"`
	Phase           domain.Phase `json:"phase"`
	HelperPID       int          `json:"helper_pid,omitempty"`
	HelperStartTime uint64       `json:"helper_start_time,omitempty"`
	TargetPID       int          `json:"target_pid,omitempty"`
	TargetStartTime uint64       `json:"target_start_time,omitempty"`
	CreatedAt       time.Time    `json:"created_at"`
	UpdatedAt       time.Time    `json:"updated_at"`
	StartedAt       time.Time    `json:"started_at,omitempty"`
	EndedAt         time.Time    `json:"ended_at,omitempty"`
	Outcome         string       `json:"outcome,omitempty"`
	ErrorCode       string       `json:"error_code,omitempty"`
	Error           string       `json:"error,omitempty"`
	ExitCode        *int         `json:"exit_code,omitempty"`
}

// NewState creates a state for a plan in one phase.
func NewState(plan Plan, phase domain.Phase, now time.Time) State {
	now = now.UTC()
	return State{
		Version:     StateVersion,
		Token:       plan.Token,
		DonorAppID:  plan.DonorAppID,
		TargetAppID: plan.TargetAppID,
		Phase:       phase,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// Validate checks identity and phase consistency.
func (s State) Validate() error {
	if s.Version != StateVersion {
		return fmt.Errorf("%w: version %d", ErrStateInvalid, s.Version)
	}
	if !domain.ValidSessionToken(s.Token) {
		return fmt.Errorf("%w: token", ErrStateInvalid)
	}
	if s.DonorAppID != domain.SpaceWarAppID {
		return fmt.Errorf("%w: donor %d", ErrStateInvalid, s.DonorAppID)
	}
	if s.TargetAppID == 0 || s.TargetAppID == domain.SpaceWarAppID {
		return fmt.Errorf("%w: target %d", ErrStateInvalid, s.TargetAppID)
	}
	if !s.Phase.Valid() {
		return fmt.Errorf("%w: phase %q", ErrStateInvalid, s.Phase)
	}
	if s.HelperPID < 0 || s.TargetPID < 0 {
		return fmt.Errorf("%w: negative pid", ErrStateInvalid)
	}
	if s.Phase == domain.PhaseActive && s.TargetPID == 0 {
		return fmt.Errorf("%w: active without target pid", ErrStateInvalid)
	}
	return nil
}

// Terminal reports whether the session cannot make further progress.
func (s State) Terminal() bool {
	return s.Phase == domain.PhaseEnded || s.Phase == domain.PhaseError
}
