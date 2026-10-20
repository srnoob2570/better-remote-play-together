package ui

import (
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/srnoob2570/better-rpt/internal/domain"
	"github.com/srnoob2570/better-rpt/internal/session"
)

type filterKey string

const (
	filterAll          filterKey = "all"
	filterReady        filterKey = "ready"
	filterUnknown      filterKey = "unknown"
	filterIncompatible filterKey = "incompatible"
)

var filterKeys = []filterKey{filterAll, filterReady, filterUnknown, filterIncompatible}

type tone uint8

const (
	toneNeutral tone = iota
	toneSuccess
	toneWarning
	toneError
	toneInfo
)

var prepSteps = []string{
	"prep.verify",
	"prep.installHelper",
	"prep.armIntent",
	"prep.startDonor",
	"prep.waitHelper",
}

// restoreReasonKey returns the visible explanation for a disabled restore
// action. It is empty when restore is available.
func restoreReasonKey(state session.RestoreState) string {
	switch state {
	case session.RestoreAvailable:
		return ""
	case session.RestoreSteamRunning:
		return "restore.reason.steamRunning"
	case session.RestoreSessionActive:
		return "restore.reason.sessionActive"
	case session.RestoreConflict:
		return "restore.reason.conflict"
	case session.RestoreNoBackup:
		return "restore.reason.noBackup"
	default:
		return "restore.reason.notNeeded"
	}
}

func situationLabelKey(kind session.SituationKind) string {
	switch kind {
	case session.SituationPrepared:
		return "situation.prepared"
	case session.SituationActive:
		return "situation.active"
	case session.SituationFinished:
		return "situation.finished"
	case session.SituationRestorePending:
		return "situation.restorePending"
	case session.SituationRestored:
		return "situation.restored"
	case session.SituationOrphan:
		return "situation.orphan"
	case session.SituationConflict:
		return "situation.conflict"
	default:
		return "situation.idle"
	}
}

func situationTone(kind session.SituationKind) tone {
	switch kind {
	case session.SituationActive, session.SituationPrepared:
		return toneInfo
	case session.SituationRestorePending, session.SituationOrphan:
		return toneWarning
	case session.SituationConflict:
		return toneError
	case session.SituationFinished, session.SituationRestored:
		return toneSuccess
	default:
		return toneNeutral
	}
}

func setupLabelKey(state session.SetupState) string {
	switch state {
	case session.SetupGuide:
		return "setup.guide"
	case session.SetupInstalled:
		return "setup.installed"
	case session.SetupRestored:
		return "setup.restored"
	case session.SetupConflict:
		return "setup.conflict"
	default:
		return "setup.missing"
	}
}

func lastOutcomeKey(outcome session.LaunchOutcome) string {
	switch outcome {
	case session.LaunchOK:
		return "last.outcome.ok"
	case session.LaunchError:
		return "last.outcome.error"
	case session.LaunchTimeout:
		return "last.outcome.timeout"
	case session.LaunchCanceled:
		return "last.outcome.canceled"
	case session.LaunchUnknown:
		return "last.outcome.unknown"
	default:
		return "last.outcome.pending"
	}
}

func lastSourceKey(source session.LaunchSource) string {
	if source == session.SourceSteam {
		return "last.source.steam"
	}
	return "last.source.betterRpt"
}

// lastOwnsSidebar reports whether the full last-session card should replace the
// selected-game card. A different selected game always keeps the primary card.
func lastOwnsSidebar(selectedID uint32, last *session.LastLaunch) bool {
	return last != nil && (selectedID == 0 || last.AppID == selectedID)
}

// visibleGames returns the sorted library subset matching a text query and one
// of the four dashboard filters. It never mutates the input slice. SpaceWar is
// the donor runtime, not a launchable target, so it stays out of the library.
func visibleGames(games []domain.Game, query string, filter filterKey) []domain.Game {
	query = strings.ToLower(strings.TrimSpace(query))
	result := make([]domain.Game, 0, len(games))
	for _, game := range games {
		if game.AppID == domain.SpaceWarAppID {
			continue
		}
		if filter != filterAll && !matchesFilter(game.Status, filter) {
			continue
		}
		if query != "" {
			name := strings.ToLower(game.Name)
			appID := strconv.FormatUint(uint64(game.AppID), 10)
			if !strings.Contains(name, query) && !strings.Contains(appID, query) {
				continue
			}
		}
		result = append(result, game)
	}
	sort.SliceStable(result, func(i, j int) bool {
		left, right := strings.ToLower(result[i].Name), strings.ToLower(result[j].Name)
		if left == right {
			return result[i].AppID < result[j].AppID
		}
		return left < right
	})
	return result
}

func matchesFilter(status domain.Status, filter filterKey) bool {
	switch filter {
	case filterReady:
		return status == domain.StatusReady
	case filterUnknown:
		return status == domain.StatusUnknown
	case filterIncompatible:
		return status == domain.StatusIncompatible
	default:
		return true
	}
}

func findGame(games []domain.Game, appID uint32) (domain.Game, bool) {
	for _, game := range games {
		if game.AppID == appID {
			return game, true
		}
	}
	return domain.Game{}, false
}

func statusLabelKey(status domain.Status) string {
	switch status {
	case domain.StatusReady:
		return "status.ready"
	case domain.StatusUnknown:
		return "status.unknown"
	case domain.StatusIncompatible:
		return "status.incompatible"
	case domain.StatusMissing:
		return "status.missing"
	case domain.StatusError:
		return "status.error"
	case domain.StatusActive:
		return "status.active"
	default:
		return "status.unknown"
	}
}

func kindLabelKey(kind domain.Kind) string {
	if kind == domain.KindProton {
		return "kind.proton"
	}
	return "kind.native"
}

func statusTone(status domain.Status) tone {
	switch status {
	case domain.StatusReady:
		return toneSuccess
	case domain.StatusUnknown:
		return toneWarning
	case domain.StatusIncompatible, domain.StatusError, domain.StatusMissing:
		return toneError
	case domain.StatusActive:
		return toneInfo
	default:
		return toneNeutral
	}
}

func filterIndex(filter filterKey) int {
	for index, key := range filterKeys {
		if key == filter {
			return index
		}
	}
	return 0
}

func filterLabelKey(filter filterKey) string {
	switch filter {
	case filterReady:
		return "filter.ready"
	case filterUnknown:
		return "filter.unknown"
	case filterIncompatible:
		return "filter.incompatible"
	default:
		return "filter.all"
	}
}

func formatDuration(duration time.Duration) string {
	if duration < 0 {
		duration = 0
	}
	total := int(duration.Seconds())
	hours := total / 3600
	minutes := (total % 3600) / 60
	seconds := total % 60
	if hours > 0 {
		return strconv.Itoa(hours) + " h " + twoDigits(minutes) + " min"
	}
	return twoDigits(minutes) + ":" + twoDigits(seconds)
}

func twoDigits(value int) string {
	if value < 10 {
		return "0" + strconv.Itoa(value)
	}
	return strconv.Itoa(value)
}

func prepStepForPhase(phase domain.Phase) int {
	switch phase {
	case domain.PhaseDonorStarting:
		return 3
	case domain.PhaseTargetStarting:
		return 4
	case domain.PhaseActive, domain.PhaseEnded, domain.PhaseError:
		return len(prepSteps)
	default:
		return 0
	}
}
