package session

import (
	"fmt"
	"time"

	"github.com/srnoob2570/better-rpt/internal/domain"
)

// IntentVersion is the durable launch-intent schema version.
const IntentVersion = 1

// IntentState is the lifecycle of the app-owned launch intent consumed by the
// dispatcher helper.
type IntentState string

const (
	IntentArmed    IntentState = "armed"
	IntentConsumed IntentState = "consumed"
	IntentExpired  IntentState = "expired"
	IntentCanceled IntentState = "canceled"
)

// Intent tells the dispatcher which plan it is allowed to launch. It is written
// by the app immediately before asking Steam to start SpaceWar and is never
// derived from Steam's LaunchOptions.
type Intent struct {
	Version     int         `json:"version"`
	Token       string      `json:"token"`
	TargetAppID uint32      `json:"target_app_id"`
	State       IntentState `json:"state"`
	CreatedAt   time.Time   `json:"created_at"`
	ExpiresAt   time.Time   `json:"expires_at"`
}

// NewIntent creates an armed intent with the plan's standard startup window.
func NewIntent(token string, targetAppID uint32, now time.Time) Intent {
	now = now.UTC()
	return Intent{
		Version:     IntentVersion,
		Token:       token,
		TargetAppID: targetAppID,
		State:       IntentArmed,
		CreatedAt:   now,
		ExpiresAt:   now.Add(PlanTTL),
	}
}

// Validate checks the intent shape and its expiry.
func (i Intent) Validate(now time.Time) error {
	if i.Version != IntentVersion {
		return fmt.Errorf("%w: version %d", ErrIntentInvalid, i.Version)
	}
	if !domain.ValidSessionToken(i.Token) {
		return fmt.Errorf("%w: token", ErrIntentInvalid)
	}
	if i.TargetAppID == 0 || i.TargetAppID == domain.SpaceWarAppID {
		return fmt.Errorf("%w: target %d", ErrIntentInvalid, i.TargetAppID)
	}
	switch i.State {
	case IntentArmed, IntentConsumed, IntentExpired, IntentCanceled:
	default:
		return fmt.Errorf("%w: state %q", ErrIntentInvalid, i.State)
	}
	if i.CreatedAt.IsZero() || i.ExpiresAt.IsZero() {
		return fmt.Errorf("%w: missing timestamps", ErrIntentInvalid)
	}
	if !now.Before(i.ExpiresAt) {
		return fmt.Errorf("%w: %s", ErrIntentExpired, i.ExpiresAt.Format(time.RFC3339))
	}
	return nil
}
