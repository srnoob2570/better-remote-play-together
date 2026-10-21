package update

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/srnoob2570/better-rpt/internal/fsutil"
)

const cacheTTL = 24 * time.Hour

type cacheState struct {
	CheckedAt time.Time `json:"checkedAt"`
	Tag       string    `json:"tag"`
}

func readCache(path string, now time.Time) (cacheState, bool) {
	if path == "" {
		return cacheState{}, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return cacheState{}, false
	}
	var state cacheState
	if err := json.Unmarshal(data, &state); err != nil {
		return cacheState{}, false
	}
	if state.Tag == "" || state.CheckedAt.IsZero() {
		return cacheState{}, false
	}
	if _, err := ParseVersion(state.Tag); err != nil {
		return cacheState{}, false
	}
	age := now.Sub(state.CheckedAt)
	if age < 0 || age >= cacheTTL {
		return cacheState{}, false
	}
	return state, true
}

func writeCache(path string, state cacheState) error {
	if path == "" {
		return nil
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode update cache: %w", err)
	}
	data = append(data, '\n')
	if err := fsutil.AtomicWriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write update cache: %w", err)
	}
	return nil
}
