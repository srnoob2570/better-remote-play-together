package ui

import (
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/srnoob2570/better-rpt/internal/domain"
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
	"prep.closeSteam",
	"prep.backup",
	"prep.launchOption",
	"prep.startDonor",
	"prep.waitHelper",
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
		return 5
	case domain.PhaseTargetStarting:
		return 6
	case domain.PhaseActive, domain.PhaseEnded, domain.PhaseError:
		return len(prepSteps)
	default:
		return 0
	}
}
