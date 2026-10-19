// Package compat reads Steam's compatibility-tool configuration and resolves
// the Proton installation used for a target. It never launches a tool and never
// writes Steam configuration.
package compat

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/srnoob2570/better-rpt/internal/steam/vdf"
)

var (
	// ErrConfig means config.vdf or its compatibility section is unreadable.
	ErrConfig = errors.New("Steam compatibility configuration is unreadable")
	// ErrMapping means a CompatToolMapping entry is malformed.
	ErrMapping = errors.New("Steam compatibility mapping is invalid")
	// ErrProtonNotFound means the configured Proton installation is not present.
	ErrProtonNotFound = errors.New("configured Proton is not installed")
	// ErrProtonAmbiguous means more than one installation matches the mapping.
	ErrProtonAmbiguous = errors.New("configured Proton is ambiguous")
	// ErrToolManifest means a Proton toolmanifest.vdf is invalid.
	ErrToolManifest = errors.New("Proton tool manifest is invalid")
	// ErrNativeTool means a mapping explicitly selects native launching.
	ErrNativeTool = errors.New("compatibility mapping selects native")
)

// Warning records a non-fatal compatibility discovery problem.
type Warning struct {
	Path string
	Err  error
}

func (w Warning) Error() string {
	if w.Path == "" {
		return w.Err.Error()
	}
	return fmt.Sprintf("%s: %v", w.Path, w.Err)
}

func (w Warning) Unwrap() error {
	return w.Err
}

// Mapping is one CompatToolMapping entry.
type Mapping struct {
	AppID    uint32
	Name     string
	Config   string
	Priority int
}

// Native reports whether the mapping selects native launching.
func (m Mapping) Native() bool {
	return strings.EqualFold(strings.TrimSpace(m.Name), "native")
}

// Mappings contains exact AppID mappings and the global AppID 0 entry.
type Mappings struct {
	Exact  map[uint32]Mapping
	Global *Mapping
}

// For returns the exact mapping for appID, otherwise the global mapping.
func (m Mappings) For(appID uint32) (Mapping, bool) {
	if exact, ok := m.Exact[appID]; ok {
		return exact, true
	}
	if m.Global != nil {
		return *m.Global, true
	}
	return Mapping{}, false
}

// ReadMappings parses config.vdf's CompatToolMapping section.
func ReadMappings(r io.Reader) (Mappings, error) {
	document, err := vdf.Parse(r)
	if err != nil {
		return Mappings{}, fmt.Errorf("%w: %v", ErrConfig, err)
	}
	steam, ok := nestedMap(document, []string{"InstallConfigStore", "Software", "Valve", "Steam"})
	if !ok {
		return Mappings{}, fmt.Errorf("%w: missing InstallConfigStore/Software/Valve/Steam", ErrConfig)
	}
	raw, exists := steam["CompatToolMapping"]
	if !exists {
		return Mappings{Exact: make(map[uint32]Mapping)}, nil
	}
	entries, ok := raw.(map[string]interface{})
	if !ok {
		return Mappings{}, fmt.Errorf("%w: CompatToolMapping is not an object", ErrMapping)
	}

	mappings := Mappings{Exact: make(map[uint32]Mapping, len(entries))}
	for key, value := range entries {
		id, err := strconv.ParseUint(key, 10, 32)
		if err != nil {
			// Valve also stores a few metadata keys here in some versions.
			continue
		}
		fields, ok := value.(map[string]interface{})
		if !ok {
			return Mappings{}, fmt.Errorf("%w: app %s is not an object", ErrMapping, key)
		}
		name := field(fields, "name")
		if strings.TrimSpace(name) == "" {
			return Mappings{}, fmt.Errorf("%w: app %s has no tool name", ErrMapping, key)
		}
		priority := 0
		if rawPriority := field(fields, "priority"); rawPriority != "" {
			priority, err = strconv.Atoi(rawPriority)
			if err != nil {
				return Mappings{}, fmt.Errorf("%w: app %s priority %q", ErrMapping, key, rawPriority)
			}
		}
		mapping := Mapping{
			AppID:    uint32(id),
			Name:     strings.TrimSpace(name),
			Config:   field(fields, "config"),
			Priority: priority,
		}
		if id == 0 {
			copy := mapping
			mappings.Global = &copy
		} else {
			mappings.Exact[uint32(id)] = mapping
		}
	}
	return mappings, nil
}

// LoadMappings reads config.vdf from path.
func LoadMappings(path string) (Mappings, error) {
	file, err := os.Open(path)
	if err != nil {
		return Mappings{}, err
	}
	defer file.Close()
	return ReadMappings(file)
}

func nestedMap(root map[string]interface{}, path []string) (map[string]interface{}, bool) {
	current := root
	for _, key := range path {
		raw, ok := current[key]
		if !ok {
			return nil, false
		}
		next, ok := raw.(map[string]interface{})
		if !ok {
			return nil, false
		}
		current = next
	}
	return current, true
}

func field(fields map[string]interface{}, key string) string {
	raw, ok := fields[key]
	if !ok {
		return ""
	}
	text, _ := raw.(string)
	return strings.TrimSpace(text)
}
