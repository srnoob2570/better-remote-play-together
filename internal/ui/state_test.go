package ui

import (
	"reflect"
	"testing"
	"time"

	"github.com/srnoob2570/better-rpt/internal/domain"
)

func sampleGames() []domain.Game {
	return []domain.Game{
		{AppID: 900002, Name: "Zeta", Kind: domain.KindProton, Status: domain.StatusUnknown},
		{AppID: 900001, Name: "Alpha", Kind: domain.KindNative, Status: domain.StatusReady},
		{AppID: 900003, Name: "Beta", Kind: domain.KindNative, Status: domain.StatusIncompatible},
	}
}

func TestVisibleGamesSortAndFilters(t *testing.T) {
	games := sampleGames()
	got := visibleGames(games, "", filterAll)
	if len(got) != 3 || got[0].Name != "Alpha" || got[1].Name != "Beta" || got[2].Name != "Zeta" {
		t.Fatalf("sorted games = %+v", got)
	}
	if got := visibleGames(games, "", filterReady); len(got) != 1 || got[0].AppID != 900001 {
		t.Fatalf("ready filter = %+v", got)
	}
	if got := visibleGames(games, "", filterUnknown); len(got) != 1 || got[0].AppID != 900002 {
		t.Fatalf("unknown filter = %+v", got)
	}
	if got := visibleGames(games, "", filterIncompatible); len(got) != 1 || got[0].AppID != 900003 {
		t.Fatalf("incompatible filter = %+v", got)
	}
}

func TestVisibleGamesExcludesDonorRuntime(t *testing.T) {
	games := append(sampleGames(), domain.Game{
		AppID:  domain.SpaceWarAppID,
		Name:   "Spacewar",
		Kind:   domain.KindProton,
		Status: domain.StatusUnknown,
	})
	if got := visibleGames(games, "", filterAll); len(got) != 3 {
		t.Fatalf("donor runtime leaked into the library: %+v", got)
	}
	if got := visibleGames(games, "spacewar", filterAll); len(got) != 0 {
		t.Fatalf("donor runtime matched a search: %+v", got)
	}
}

func TestVisibleGamesSearchNameAndAppID(t *testing.T) {
	games := sampleGames()
	if got := visibleGames(games, "alp", filterAll); len(got) != 1 || got[0].AppID != 900001 {
		t.Fatalf("name search = %+v", got)
	}
	if got := visibleGames(games, "900002", filterAll); len(got) != 1 || got[0].Name != "Zeta" {
		t.Fatalf("app id search = %+v", got)
	}
	if got := visibleGames(games, "missing", filterAll); len(got) != 0 {
		t.Fatalf("empty search = %+v", got)
	}
	if games[0].Name != "Zeta" {
		t.Fatal("visibleGames mutated the input order")
	}
}

func TestFindGame(t *testing.T) {
	if game, ok := findGame(sampleGames(), 900001); !ok || game.Name != "Alpha" {
		t.Fatalf("find = %+v, %v", game, ok)
	}
	if _, ok := findGame(sampleGames(), 1); ok {
		t.Fatal("unexpected game")
	}
}

func TestStatusAndKindLabels(t *testing.T) {
	cases := map[domain.Status]string{
		domain.StatusReady:        "status.ready",
		domain.StatusUnknown:      "status.unknown",
		domain.StatusIncompatible: "status.incompatible",
		domain.StatusMissing:      "status.missing",
		domain.StatusError:        "status.error",
		domain.StatusActive:       "status.active",
	}
	for status, want := range cases {
		if got := statusLabelKey(status); got != want {
			t.Fatalf("status %v key = %q, want %q", status, got, want)
		}
	}
	if got := kindLabelKey(domain.KindNative); got != "kind.native" {
		t.Fatalf("native key = %q", got)
	}
	if got := kindLabelKey(domain.KindProton); got != "kind.proton" {
		t.Fatalf("proton key = %q", got)
	}
}

func TestStatusTone(t *testing.T) {
	cases := map[domain.Status]tone{
		domain.StatusReady:        toneSuccess,
		domain.StatusUnknown:      toneWarning,
		domain.StatusIncompatible: toneError,
		domain.StatusMissing:      toneError,
		domain.StatusError:        toneError,
		domain.StatusActive:       toneInfo,
	}
	for status, want := range cases {
		if got := statusTone(status); got != want {
			t.Fatalf("status %v tone = %v, want %v", status, got, want)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	cases := []struct {
		duration time.Duration
		want     string
	}{
		{0, "00:00"},
		{9 * time.Second, "00:09"},
		{65 * time.Second, "01:05"},
		{time.Hour + 2*time.Minute + 3*time.Second, "1 h 02 min"},
		{-time.Second, "00:00"},
	}
	for _, test := range cases {
		if got := formatDuration(test.duration); got != test.want {
			t.Fatalf("formatDuration(%s) = %q, want %q", test.duration, got, test.want)
		}
	}
}

func TestPrepStepForPhase(t *testing.T) {
	cases := map[domain.Phase]int{
		domain.PhasePreparing:      0,
		domain.PhaseDonorStarting:  5,
		domain.PhaseTargetStarting: 6,
		domain.PhaseActive:         7,
		domain.PhaseEnded:          7,
		domain.PhaseError:          7,
	}
	for phase, want := range cases {
		if got := prepStepForPhase(phase); got != want {
			t.Fatalf("phase %s step = %d, want %d", phase, got, want)
		}
	}
}

func TestPrepSteps(t *testing.T) {
	if len(prepSteps) != 7 {
		t.Fatalf("prep steps = %d, want 7", len(prepSteps))
	}
	if !reflect.DeepEqual(filterKeys, []filterKey{filterAll, filterReady, filterUnknown, filterIncompatible}) {
		t.Fatalf("filter keys = %v", filterKeys)
	}
}
