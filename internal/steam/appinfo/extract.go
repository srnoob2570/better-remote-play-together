package appinfo

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

func extractApp(root map[string]any) (Common, []Launch, error) {
	common := Common{}
	if raw, ok := root["common"]; ok {
		obj, ok := raw.(map[string]any)
		if !ok {
			return Common{}, nil, fmt.Errorf("common is not an object")
		}
		common.Name = stringValue(obj["name"])
		common.Type = stringValue(obj["type"])
		common.OSList = stringValue(obj["oslist"])
		common.InstallDir = stringValue(obj["installdir"])
	}

	configRaw, ok := root["config"]
	if !ok {
		return common, nil, nil
	}
	config, ok := configRaw.(map[string]any)
	if !ok {
		return Common{}, nil, fmt.Errorf("config is not an object")
	}
	launchRaw, ok := config["launch"]
	if !ok {
		return common, nil, nil
	}
	launchObj, ok := launchRaw.(map[string]any)
	if !ok {
		return Common{}, nil, fmt.Errorf("config.launch is not an object")
	}

	launches := make([]Launch, 0, len(launchObj))
	for key, raw := range launchObj {
		if _, err := strconv.Atoi(key); err != nil {
			continue // metadata, not a launch entry
		}
		obj, ok := raw.(map[string]any)
		if !ok {
			return Common{}, nil, fmt.Errorf("launch %q is not an object", key)
		}
		launch := Launch{
			Key:         key,
			OSList:      stringValue(obj["oslist"]),
			Arch:        stringValue(obj["osarch"]),
			Executable:  stringValue(obj["executable"]),
			Arguments:   stringValue(obj["arguments"]),
			WorkingDir:  stringValue(obj["workingdir"]),
			Type:        stringValue(obj["type"]),
			Description: stringValue(obj["description"]),
		}
		if cfgRaw, ok := obj["config"]; ok {
			cfg, ok := cfgRaw.(map[string]any)
			if !ok {
				return Common{}, nil, fmt.Errorf("launch %q config is not an object", key)
			}
			if launch.OSList == "" {
				launch.OSList = stringValue(cfg["oslist"])
			}
			if launch.Arch == "" {
				launch.Arch = stringValue(cfg["osarch"])
			}
			launch.OwnsDLC = truthy(cfg["ownsdlc"])
		}
		launch.OwnsDLC = launch.OwnsDLC || truthy(obj["ownsdlc"])
		launch.BetaKeys = betaKeys(obj)
		launches = append(launches, launch)
	}
	sort.SliceStable(launches, func(i, j int) bool {
		return launchKeyLess(launches[i].Key, launches[j].Key)
	})
	return common, launches, nil
}

func stringValue(v any) string {
	s, _ := v.(string)
	return s
}

func truthy(v any) bool {
	switch value := v.(type) {
	case nil:
		return false
	case bool:
		return value
	case string:
		switch strings.ToLower(strings.TrimSpace(value)) {
		case "1", "true", "yes", "on":
			return true
		}
	case int32:
		return value != 0
	case int64:
		return value != 0
	case uint32:
		return value != 0
	case uint64:
		return value != 0
	}
	return false
}

func betaKeys(obj map[string]any) []string {
	var keys []string
	for _, field := range []string{"BetaKey", "BetaKeys"} {
		value := stringValue(obj[field])
		for _, part := range strings.Split(value, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				keys = append(keys, part)
			}
		}
	}
	return keys
}

func launchKeyLess(a, b string) bool {
	ai, aerr := strconv.Atoi(a)
	bi, berr := strconv.Atoi(b)
	switch {
	case aerr == nil && berr == nil:
		return ai < bi
	case aerr == nil:
		return true
	case berr == nil:
		return false
	default:
		return a < b
	}
}
