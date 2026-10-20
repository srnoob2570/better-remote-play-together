package session

import (
	"testing"
	"time"

	"github.com/srnoob2570/better-rpt/internal/domain"
)

func situationState(token string, phase domain.Phase, target uint32) *State {
	return &State{
		Version:     StateVersion,
		Token:       token,
		DonorAppID:  domain.SpaceWarAppID,
		TargetAppID: target,
		Phase:       phase,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
}

func armedIntent(token string, target uint32) *Intent {
	return &Intent{
		Version:     IntentVersion,
		Token:       token,
		TargetAppID: target,
		State:       IntentArmed,
		CreatedAt:   time.Now().UTC(),
		ExpiresAt:   time.Now().UTC().Add(time.Minute),
	}
}

func situationLast(appID uint32) *LastLaunch {
	return &LastLaunch{
		Version:   LastLaunchVersion,
		AppID:     appID,
		Name:      "Game",
		Token:     testToken,
		Source:    SourceBetterRPT,
		Outcome:   LaunchOK,
		StartedAt: time.Now().UTC(),
	}
}

func TestClassifySituation(t *testing.T) {
	tests := []struct {
		name       string
		evidence   Evidence
		kind       SituationKind
		restore    RestoreState
		token      string
		target     uint32
		lastAppID  uint32
		legacySeen bool
	}{
		{
			name:     "empty installation is idle",
			evidence: Evidence{Setup: SetupMissing, Marker: MarkerNone},
			kind:     SituationIdle,
			restore:  RestoreNotNeeded,
		},
		{
			name: "foreign launch option is not pending",
			evidence: Evidence{
				Setup:  SetupMissing,
				Marker: MarkerForeign,
			},
			kind:    SituationIdle,
			restore: RestoreNotNeeded,
		},
		{
			name: "installed dispatcher with no session is idle",
			evidence: Evidence{
				Setup:       SetupInstalled,
				SetupBackup: true,
				Marker:      MarkerDispatcher,
			},
			kind:    SituationIdle,
			restore: RestoreAvailable,
		},
		{
			name: "armed intent is prepared",
			evidence: Evidence{
				Setup:       SetupInstalled,
				SetupBackup: true,
				Marker:      MarkerDispatcher,
				Intent:      armedIntent(testToken, 900001),
				State:       situationState(testToken, domain.PhasePreparing, 900001),
			},
			kind:    SituationPrepared,
			restore: RestoreAvailable,
			token:   testToken,
			target:  900001,
		},
		{
			name: "live helper is active",
			evidence: Evidence{
				Setup:       SetupInstalled,
				SetupBackup: true,
				Marker:      MarkerDispatcher,
				State:       situationState(testToken, domain.PhaseActive, 900001),
				HelperAlive: true,
			},
			kind:    SituationActive,
			restore: RestoreSessionActive,
			token:   testToken,
			target:  900001,
		},
		{
			name: "active state with dead helper is finished",
			evidence: Evidence{
				Setup:       SetupInstalled,
				SetupBackup: true,
				Marker:      MarkerDispatcher,
				State:       situationState(testToken, domain.PhaseActive, 900001),
				HelperAlive: false,
			},
			kind:    SituationFinished,
			restore: RestoreAvailable,
			token:   testToken,
			target:  900001,
		},
		{
			name: "interrupted preparing state is finished",
			evidence: Evidence{
				Setup:       SetupInstalled,
				SetupBackup: true,
				Marker:      MarkerDispatcher,
				State:       situationState(testToken, domain.PhasePreparing, 900001),
				HelperAlive: false,
			},
			kind:    SituationFinished,
			restore: RestoreAvailable,
			token:   testToken,
			target:  900001,
		},
		{
			name: "terminal session is finished",
			evidence: Evidence{
				Setup:       SetupInstalled,
				SetupBackup: true,
				Marker:      MarkerDispatcher,
				State:       situationState(testToken, domain.PhaseEnded, 900001),
			},
			kind:    SituationFinished,
			restore: RestoreAvailable,
			token:   testToken,
			target:  900001,
		},
		{
			name: "coherent legacy marker is restore pending",
			evidence: Evidence{
				Setup:  SetupMissing,
				Marker: MarkerLegacy,
				Legacy: &LegacyEvidence{
					BackupToken:     testToken,
					BackupTarget:    900001,
					BackupAvailable: true,
				},
				State: situationState(testToken, domain.PhaseEnded, 900001),
			},
			kind:       SituationRestorePending,
			restore:    RestoreAvailable,
			token:      testToken,
			target:     900001,
			legacySeen: true,
		},
		{
			name: "legacy marker without state is orphan",
			evidence: Evidence{
				Setup:  SetupMissing,
				Marker: MarkerLegacy,
				Legacy: &LegacyEvidence{
					BackupToken:     testToken,
					BackupTarget:    900001,
					BackupAvailable: true,
				},
			},
			kind:       SituationOrphan,
			restore:    RestoreAvailable,
			token:      testToken,
			target:     900001,
			legacySeen: true,
		},
		{
			name: "legacy marker without backup is orphan and not restorable",
			evidence: Evidence{
				Setup:  SetupMissing,
				Marker: MarkerLegacy,
			},
			kind:    SituationOrphan,
			restore: RestoreNoBackup,
		},
		{
			name: "dispatcher marker without setup record is orphan",
			evidence: Evidence{
				Setup:  SetupMissing,
				Marker: MarkerDispatcher,
			},
			kind:    SituationOrphan,
			restore: RestoreNoBackup,
		},
		{
			name: "setup installed but marker drifted is conflict",
			evidence: Evidence{
				Setup:       SetupInstalled,
				SetupBackup: true,
				Marker:      MarkerForeign,
			},
			kind:    SituationConflict,
			restore: RestoreConflict,
		},
		{
			name: "legacy marker with mismatched state is conflict",
			evidence: Evidence{
				Setup:  SetupMissing,
				Marker: MarkerLegacy,
				Legacy: &LegacyEvidence{
					BackupToken:     testToken,
					BackupTarget:    900001,
					BackupAvailable: true,
				},
				State: situationState("fedcba9876543210fedcba9876543210", domain.PhaseEnded, 900001),
			},
			kind:       SituationConflict,
			restore:    RestoreConflict,
			token:      testToken,
			target:     900001,
			legacySeen: true,
		},
		{
			name: "restored setup is restored",
			evidence: Evidence{
				Setup:  SetupRestored,
				Marker: MarkerNone,
			},
			kind:    SituationRestored,
			restore: RestoreNotNeeded,
		},
		{
			name: "steam running blocks restore",
			evidence: Evidence{
				Setup:        SetupInstalled,
				SetupBackup:  true,
				Marker:       MarkerDispatcher,
				SteamRunning: true,
			},
			kind:    SituationIdle,
			restore: RestoreSteamRunning,
		},
		{
			name: "last launch stays visible beside legacy pending configuration",
			evidence: Evidence{
				Setup:  SetupMissing,
				Marker: MarkerLegacy,
				Legacy: &LegacyEvidence{
					BackupToken:     testToken,
					BackupTarget:    900001,
					BackupAvailable: true,
				},
				State: situationState(testToken, domain.PhaseEnded, 900001),
				Last:  situationLast(900002),
			},
			kind:       SituationRestorePending,
			restore:    RestoreAvailable,
			token:      testToken,
			target:     900001,
			lastAppID:  900002,
			legacySeen: true,
		},
		{
			name: "guide marker is treated as installed",
			evidence: Evidence{
				Setup:       SetupGuide,
				SetupBackup: true,
				Marker:      MarkerDispatcher,
				State:       situationState(testToken, domain.PhaseEnded, 900001),
			},
			kind:    SituationFinished,
			restore: RestoreAvailable,
			token:   testToken,
			target:  900001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Classify(tt.evidence)
			if got.Kind != tt.kind {
				t.Fatalf("kind = %q, want %q (situation=%+v)", got.Kind, tt.kind, got)
			}
			if got.Restore != tt.restore {
				t.Fatalf("restore = %q, want %q", got.Restore, tt.restore)
			}
			if got.Token != tt.token {
				t.Fatalf("token = %q, want %q", got.Token, tt.token)
			}
			if got.TargetAppID != tt.target {
				t.Fatalf("target = %d, want %d", got.TargetAppID, tt.target)
			}
			if tt.lastAppID != 0 {
				if got.Last == nil || got.Last.AppID != tt.lastAppID {
					t.Fatalf("last = %+v, want app %d", got.Last, tt.lastAppID)
				}
			}
			if (got.Legacy != nil) != tt.legacySeen {
				t.Fatalf("legacy = %+v, want seen=%v", got.Legacy, tt.legacySeen)
			}
		})
	}
}

func TestClassifyDoesNotInferPendingFromForeignLaunchOption(t *testing.T) {
	stale := &LegacyEvidence{BackupToken: testToken, BackupTarget: 900001, BackupAvailable: true}
	got := Classify(Evidence{Setup: SetupMissing, Marker: MarkerForeign, Legacy: stale})
	if got.Kind != SituationIdle || got.Legacy != nil {
		t.Fatalf("situation = %+v, want idle without legacy", got)
	}
}
