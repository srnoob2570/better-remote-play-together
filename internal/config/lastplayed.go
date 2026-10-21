package config

import (
	"bytes"
	"fmt"
	"strconv"
	"time"

	"github.com/srnoob2570/better-rpt/internal/domain"
	"github.com/srnoob2570/better-rpt/internal/steam/vdf"
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
	if steamRoot == "" {
		return nil, fmt.Errorf("%w: empty Steam root", ErrLocalConfigMissing)
	}
	account, err := SelectAccount(steamRoot)
	if err != nil {
		return nil, err
	}
	data, _, err := readLocalConfig(account.LocalConfigPath(steamRoot))
	if err != nil {
		return nil, err
	}
	root, err := vdf.Parse(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("%w: parse %s: %v", ErrLocalConfigInvalid, account.LocalConfigPath(steamRoot), err)
	}
	apps, ok := vdf.Lookup(root, launchOptionPath...)
	if !ok {
		return nil, nil
	}
	var newest *LastPlayed
	for key, raw := range apps {
		appID, err := strconv.ParseUint(key, 10, 32)
		if err != nil || appID == 0 || uint32(appID) == domain.SpaceWarAppID {
			continue
		}
		app, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		epoch, ok := parseEpoch(app["LastPlayed"])
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
