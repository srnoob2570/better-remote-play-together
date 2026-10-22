// Package domain contains the pure vocabulary of Better RPT: game states,
// launch descriptions, session state and the errors shared by the layers.
//
// It must not import UI packages, read files or execute processes.
package domain

import (
	"errors"
	"fmt"
	"strings"
)

const (
	// SpaceWarAppID is the only supported Remote Play Together donor.
	SpaceWarAppID uint32 = 480
	// SessionTokenLength is the length of a lowercase hexadecimal session token.
	SessionTokenLength = 32
)

// Kind describes how a target is launched.
type Kind uint8

const (
	KindUnknown Kind = iota
	KindNative
	KindProton
)

func (k Kind) String() string {
	switch k {
	case KindNative:
		return "native"
	case KindProton:
		return "proton"
	default:
		return "unknown"
	}
}

// Valid reports whether the kind can be launched.
func (k Kind) Valid() bool {
	return k == KindNative || k == KindProton
}

// Status is the compatibility state shown by the dashboard.
type Status uint8

const (
	StatusUnknown Status = iota
	StatusReady
	StatusIncompatible
	StatusMissing
	StatusError
	StatusActive
)

func (s Status) String() string {
	switch s {
	case StatusReady:
		return "ready"
	case StatusIncompatible:
		return "incompatible"
	case StatusMissing:
		return "missing"
	case StatusError:
		return "error"
	case StatusActive:
		return "active"
	default:
		return "unknown"
	}
}

// Game is one installed Steam application.
type Game struct {
	AppID       uint32
	Name        string
	InstallDir  string // absolute path to steamapps/common/<installdir>
	LibraryPath string // library root, not the steamapps directory
	SizeBytes   uint64
	Kind        Kind
	Status      Status
	Launch      *LaunchSpec
	Problem     error // why the game is not ready, when known
}

// LaunchSpec is a pure description of how to start a target. It does not hold
// an exec.Cmd; the launch layer turns it into a process.
type LaunchSpec struct {
	AppID          uint32
	Kind           Kind
	Executable     string
	Arguments      []string
	WorkingDir     string
	InstallDir     string
	SteamRoot      string
	LibraryPaths   []string
	CompatDataPath string
	ProtonPath     string
	ProtonName     string
}

// Validate checks the invariants that are independent of the filesystem.
func (s LaunchSpec) Validate() error {
	if s.AppID == 0 {
		return fmt.Errorf("%w: empty app id", ErrInvalidLaunchSpec)
	}
	if !s.Kind.Valid() {
		return fmt.Errorf("%w: unsupported kind %q", ErrInvalidLaunchSpec, s.Kind)
	}
	if strings.TrimSpace(s.Executable) == "" {
		return fmt.Errorf("%w: empty executable", ErrInvalidLaunchSpec)
	}
	if s.Kind == KindProton && strings.TrimSpace(s.ProtonPath) == "" {
		return fmt.Errorf("%w: proton path is required", ErrInvalidLaunchSpec)
	}
	return nil
}

// Phase is the lifecycle state of a donor session.
type Phase string

const (
	PhaseIdle           Phase = "idle"
	PhasePreparing      Phase = "preparing"
	PhaseLaunching      Phase = "launching"
	PhaseDonorStarting  Phase = "donor_starting"
	PhaseTargetStarting Phase = "target_starting"
	PhaseActive         Phase = "active"
	PhaseEnded          Phase = "ended"
	PhaseError          Phase = "error"
	PhaseRestoring      Phase = "restoring"
)

// Valid reports whether p is a known phase.
func (p Phase) Valid() bool {
	switch p {
	case PhaseIdle, PhasePreparing, PhaseLaunching, PhaseDonorStarting, PhaseTargetStarting,
		PhaseActive, PhaseEnded, PhaseError, PhaseRestoring:
		return true
	default:
		return false
	}
}

// ValidSessionToken reports whether token is exactly 32 lowercase hex digits.
func ValidSessionToken(token string) bool {
	if len(token) != SessionTokenLength {
		return false
	}
	for _, c := range token {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

// RefuseRoot enforces the no-root rule. It is pure so commands and tests can
// check the policy without changing the process uid.
func RefuseRoot(euid int) error {
	if euid == 0 {
		return ErrRoot
	}
	return nil
}

var (
	ErrRoot              = errors.New("better-rpt must not run as root")
	ErrSteamNotFound     = errors.New("steam installation not found")
	ErrSteamUnsupported  = errors.New("steam installation variant is unsupported")
	ErrLibraryUnreadable = errors.New("steam library is unreadable")
	ErrVDFMalformed      = errors.New("steam VDF is malformed")
	ErrManifestCorrupt   = errors.New("steam app manifest is corrupt")
	ErrPathEscape        = errors.New("path escapes its allowed root")
	ErrSpaceWarMissing   = errors.New("spacewar is not installed")
	ErrSteamRunning      = errors.New("steam is running")
	ErrInstanceLocked    = errors.New("another better-rpt instance is running")
	ErrConfig            = errors.New("user configuration is invalid")
	ErrPaths             = errors.New("user data directories are unavailable")
	ErrUnsupported       = errors.New("unsupported value")
	ErrInvalidSession    = errors.New("invalid session")
	ErrInvalidLaunchSpec = errors.New("invalid launch spec")
	ErrHelperMissing     = errors.New("stable helper is missing")
	ErrNoLaunchOption    = errors.New("launch option is missing")
)
