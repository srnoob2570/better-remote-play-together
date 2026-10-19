package steam

import (
	"path/filepath"
	"strconv"
	"strings"

	"github.com/srnoob2570/better-rpt/internal/domain"
)

// Filter is the reusable UI filter. Nil Kind and Status mean "any". Library
// is an exact library root; an empty value means all libraries.
type Filter struct {
	Query   string
	Kind    *domain.Kind
	Status  *domain.Status
	Library string
}

// Search filters by name or app ID.
func (c Catalog) Search(query string) []domain.Game {
	return FilterGames(c.Games, Filter{Query: query})
}

// Filter applies the side-panel filters to this catalog.
func (c Catalog) Filter(filter Filter) []domain.Game {
	return FilterGames(c.Games, filter)
}

// FilterGames applies a filter without depending on the catalog type. It
// preserves catalog order and always returns a non-nil slice.
func FilterGames(games []domain.Game, filter Filter) []domain.Game {
	query := strings.ToLower(strings.TrimSpace(filter.Query))
	result := make([]domain.Game, 0, len(games))
	for _, game := range games {
		if query != "" &&
			!strings.Contains(strings.ToLower(game.Name), query) &&
			!strings.Contains(strconv.FormatUint(uint64(game.AppID), 10), query) {
			continue
		}
		if filter.Kind != nil && game.Kind != *filter.Kind {
			continue
		}
		if filter.Status != nil && game.Status != *filter.Status {
			continue
		}
		if filter.Library != "" && !samePath(game.LibraryPath, filter.Library) {
			continue
		}
		result = append(result, game)
	}
	return result
}

func samePath(a, b string) bool {
	if a == "" || b == "" {
		return a == b
	}
	return filepath.Clean(a) == filepath.Clean(b)
}
