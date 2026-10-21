package config

import (
	"strconv"
	"time"

	"github.com/srnoob2570/better-rpt/internal/domain"
)

// LastPlayed is Steam's most recent non-donor app launch, read from the same
// localconfig.vdf Better RPT already inspects.
type LastPlayed struct {
	AppID uint32
	At    time.Time
}

// ReadLastPlayed returns the newest LastPlayed entry excluding SpaceWar. Steam
// writes this field when a game is launched from its own UI or library.
func ReadLastPlayed(steamRoot string) (*LastPlayed, error) {
	state, err := ReadLocalState(steamRoot)
	if err != nil {
		return nil, err
	}
	return state.LastPlayed, nil
}

func lastPlayedFromDocument(document *vdfDocument) (*LastPlayed, error) {
	apps, err := findObject(document, launchOptionPath)
	if err != nil {
		return nil, err
	}
	if apps == nil {
		return nil, nil
	}
	var newest *LastPlayed
	for _, appNode := range apps.Children {
		appID, err := strconv.ParseUint(appNode.Key, 10, 32)
		if err != nil || appID == 0 || uint32(appID) == domain.SpaceWarAppID {
			continue
		}
		if appNode.Object == nil {
			continue
		}
		var last *vdfNode
		for _, child := range appNode.Object.Children {
			if child.Key == "LastPlayed" {
				last = child
			}
		}
		if last == nil || last.Object != nil {
			continue
		}
		epoch, ok := parseEpoch(last.Value)
		if !ok {
			continue
		}
		at := time.Unix(epoch, 0).UTC()
		if newest == nil || at.After(newest.At) {
			newest = &LastPlayed{AppID: uint32(appID), At: at}
		}
	}
	return newest, nil
}

func parseEpoch(raw interface{}) (int64, bool) {
	var value string
	switch typed := raw.(type) {
	case string:
		value = typed
	case int64:
		return typed, typed > 0
	case int:
		return int64(typed), typed > 0
	case float64:
		return int64(typed), typed > 0
	default:
		return 0, false
	}
	epoch, err := strconv.ParseInt(value, 10, 64)
	return epoch, err == nil && epoch > 0
}
