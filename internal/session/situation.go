package session

import "github.com/srnoob2570/better-rpt/internal/domain"

// SituationKind is the explicit, user-visible lifecycle state of the launch
// integration. It is derived from durable evidence and never from a launch
// option alone.
type SituationKind string

const (
	SituationIdle           SituationKind = "idle"
	SituationPrepared       SituationKind = "prepared"
	SituationActive         SituationKind = "active"
	SituationFinished       SituationKind = "finished"
	SituationRestorePending SituationKind = "restore_pending"
	SituationRestored       SituationKind = "restored"
	SituationOrphan         SituationKind = "orphan"
	SituationConflict       SituationKind = "conflict"
)

// SetupState describes Better RPT's persistent SpaceWar dispatcher record.
type SetupState string

const (
	SetupMissing   SetupState = "missing"
	SetupGuide     SetupState = "guide"
	SetupInstalled SetupState = "installed"
	SetupRestored  SetupState = "restored"
	SetupConflict  SetupState = "conflict"
)

// MarkerKind classifies the current SpaceWar launch option without trusting its
// contents as session evidence.
type MarkerKind string

const (
	MarkerNone       MarkerKind = "none"
	MarkerDispatcher MarkerKind = "dispatcher"
	MarkerLegacy     MarkerKind = "legacy"
	MarkerForeign    MarkerKind = "foreign"
)

// RestoreState explains whether, and why, the restore action is available.
type RestoreState string

const (
	RestoreAvailable     RestoreState = "available"
	RestoreNotNeeded     RestoreState = "not_needed"
	RestoreSteamRunning  RestoreState = "steam_running"
	RestoreSessionActive RestoreState = "session_active"
	RestoreConflict      RestoreState = "conflict"
	RestoreNoBackup      RestoreState = "no_backup"
)

// LegacyEvidence is the durable backup evidence for a pre-dispatcher session.
type LegacyEvidence struct {
	BackupToken     string
	BackupTarget    uint32
	BackupAvailable bool
}

// Evidence is the complete input to the lifecycle classifier. All fields are
// already validated by the callers; Classify performs no I/O.
type Evidence struct {
	Setup        SetupState
	SetupBackup  bool
	Marker       MarkerKind
	Legacy       *LegacyEvidence
	Intent       *Intent
	State        *State
	HelperAlive  bool
	SteamRunning bool
	Last         *LastLaunch
}

// Situation is the UI-facing result of classification.
type Situation struct {
	Kind         SituationKind
	Setup        SetupState
	Restore      RestoreState
	Token        string
	TargetAppID  uint32
	Legacy       *LegacyEvidence
	Intent       *Intent
	State        *State
	Last         *LastLaunch
	SteamRunning bool
}

// Classify derives the lifecycle state from durable evidence. It deliberately
// ignores foreign launch options: a non-default SpaceWar LaunchOptions value is
// not, by itself, a pending Better RPT session.
func Classify(e Evidence) Situation {
	s := Situation{
		Setup:        e.Setup,
		Intent:       e.Intent,
		State:        e.State,
		Last:         e.Last,
		SteamRunning: e.SteamRunning,
	}
	effectiveSetup := e.Setup
	if e.Setup == SetupGuide && e.Marker == MarkerDispatcher {
		effectiveSetup = SetupInstalled
	}
	s.Setup = effectiveSetup

	// A setup record and the current marker must agree. A user edit is a
	// conflict, never something Better RPT silently overwrites.
	if e.Setup == SetupConflict ||
		(e.Setup == SetupInstalled && e.Marker != MarkerDispatcher) ||
		(e.Setup == SetupRestored && e.Marker != MarkerNone) {
		s.Kind = SituationConflict
		s.Restore = RestoreConflict
		return s
	}

	// Legacy transaction evidence. The token must be coherent across marker,
	// backup and state; a lone launch option is only an orphan.
	if e.Marker == MarkerLegacy && e.Legacy != nil {
		s.Legacy = e.Legacy
		s.Token = e.Legacy.BackupToken
		s.TargetAppID = e.Legacy.BackupTarget
		if e.State != nil {
			if e.State.Token != e.Legacy.BackupToken ||
				(e.Legacy.BackupTarget != 0 && e.State.TargetAppID != e.Legacy.BackupTarget) {
				s.Kind = SituationConflict
				s.Restore = RestoreConflict
				return s
			}
			s.Token = e.State.Token
			s.TargetAppID = e.State.TargetAppID
		}
		if e.Legacy.BackupAvailable && e.State != nil {
			s.Kind = SituationRestorePending
			if effectiveSetup == SetupInstalled && e.HelperAlive && e.State.Phase == domain.PhaseActive {
				s.Kind = SituationActive
			}
			s.Restore = restoreState(e, true)
			return s
		}
		// Backup exists but no session state, or no backup at all: orphan.
		s.Kind = SituationOrphan
		s.Restore = restoreState(e, e.Legacy.BackupAvailable)
		return s
	}

	// Dispatcher markers are setup evidence, not session evidence.
	if e.Marker == MarkerDispatcher && effectiveSetup != SetupInstalled {
		s.Kind = SituationOrphan
		s.Restore = RestoreNoBackup
		return s
	}

	if e.Setup == SetupRestored && e.Marker == MarkerNone {
		s.Kind = SituationRestored
		s.Restore = RestoreNotNeeded
		return s
	}

	if e.State != nil {
		s.Token = e.State.Token
		s.TargetAppID = e.State.TargetAppID
		if e.State.Phase == domain.PhaseActive && e.HelperAlive {
			s.Kind = SituationActive
			s.Restore = RestoreSessionActive
			return s
		}
		if e.State.Phase == domain.PhaseActive && !e.HelperAlive {
			s.Kind = SituationFinished
			s.Restore = restoreState(e, effectiveSetup == SetupInstalled || e.SetupBackup)
			return s
		}
		if e.State.Terminal() {
			s.Kind = SituationFinished
			s.Restore = restoreState(e, effectiveSetup == SetupInstalled || e.SetupBackup)
			return s
		}
	}

	if e.Intent != nil && e.Intent.State == IntentArmed && effectiveSetup == SetupInstalled {
		s.Kind = SituationPrepared
		s.Token = e.Intent.Token
		s.TargetAppID = e.Intent.TargetAppID
		s.Restore = restoreState(e, true)
		return s
	}

	// A non-terminal state with no live helper is a crashed or interrupted
	// launch, not an active session. It must be recoverable, not misreported as
	// pending from its old launch option.
	if e.State != nil && !e.State.Terminal() && !e.HelperAlive {
		s.Kind = SituationFinished
		s.Restore = restoreState(e, effectiveSetup == SetupInstalled || e.SetupBackup)
		return s
	}

	// A legacy marker with no matching backup is orphaned configuration; it is
	// not the last session and is not restorable without a backup.
	if e.Marker == MarkerLegacy {
		s.Kind = SituationOrphan
		s.Restore = RestoreNoBackup
		return s
	}

	s.Kind = SituationIdle
	s.Restore = restoreState(e, effectiveSetup == SetupInstalled && e.SetupBackup)
	return s
}

func restoreState(e Evidence, restorable bool) RestoreState {
	if !restorable {
		return RestoreNotNeeded
	}
	if e.HelperAlive {
		return RestoreSessionActive
	}
	if e.SteamRunning {
		return RestoreSteamRunning
	}
	return RestoreAvailable
}
