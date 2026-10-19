package session

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/srnoob2570/better-rpt/internal/domain"
)

const (
	// PlanVersion is the durable plan schema version.
	PlanVersion = 1
	// PlanTTL is the maximum time between writing a plan and starting its helper.
	PlanTTL = 15 * time.Minute
	// StartupTimeout is how long the assistant waits for a helper to become active.
	StartupTimeout = 120 * time.Second
	// GroupGrace is the grace period for target process-group children after exit.
	GroupGrace = 5 * time.Second
	// EarlyExit is the window in which a target exit means a launch failure.
	EarlyExit = 2 * time.Second
)

// Plan is written by the app before the donor launch option is prepared. It
// contains everything the helper needs to launch the target without trusting
// command-line data beyond the target id and token.
type Plan struct {
	Version     int               `json:"version"`
	Token       string            `json:"token"`
	DonorAppID  uint32            `json:"donor_app_id"`
	TargetAppID uint32            `json:"target_app_id"`
	CreatedAt   time.Time         `json:"created_at"`
	ExpiresAt   time.Time         `json:"expires_at"`
	Launch      domain.LaunchSpec `json:"launch"`
}

// NewPlan creates a plan with the standard TTL.
func NewPlan(token string, targetAppID uint32, spec domain.LaunchSpec, now time.Time) Plan {
	now = now.UTC()
	return Plan{
		Version:     PlanVersion,
		Token:       token,
		DonorAppID:  domain.SpaceWarAppID,
		TargetAppID: targetAppID,
		CreatedAt:   now,
		ExpiresAt:   now.Add(PlanTTL),
		Launch:      spec,
	}
}

// Validate checks identity, embedded launch context and expiry.
func (p Plan) Validate(now time.Time) error {
	if p.Version != PlanVersion {
		return fmt.Errorf("%w: version %d", ErrPlanInvalid, p.Version)
	}
	if !domain.ValidSessionToken(p.Token) {
		return fmt.Errorf("%w: token", ErrPlanInvalid)
	}
	if p.DonorAppID != domain.SpaceWarAppID {
		return fmt.Errorf("%w: donor %d", ErrPlanInvalid, p.DonorAppID)
	}
	if p.TargetAppID == 0 || p.TargetAppID == domain.SpaceWarAppID {
		return fmt.Errorf("%w: target %d", ErrPlanInvalid, p.TargetAppID)
	}
	if p.CreatedAt.IsZero() || p.ExpiresAt.IsZero() {
		return fmt.Errorf("%w: missing timestamps", ErrPlanInvalid)
	}
	if !now.Before(p.ExpiresAt) {
		return fmt.Errorf("%w: %s", ErrPlanExpired, p.ExpiresAt.Format(time.RFC3339))
	}
	if p.Launch.AppID != p.TargetAppID {
		return fmt.Errorf("%w: launch app %d != target %d", ErrPlanInvalid, p.Launch.AppID, p.TargetAppID)
	}
	if err := p.Launch.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrPlanInvalid, err)
	}
	return nil
}

// NewToken returns a 32-character lowercase hexadecimal session token.
func NewToken() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("generate session token: %w", err)
	}
	return hex.EncodeToString(raw[:]), nil
}
