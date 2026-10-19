package compat

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/srnoob2570/better-rpt/internal/steam/vdf"
)

// ToolManifest is the subset of toolmanifest.vdf used by Better RPT.
type ToolManifest struct {
	Version                 string
	CommandLine             string
	RequireToolAppID        uint32
	UseSessions             bool
	UseToolSubprocessReaper bool
	CompatManagerLayerName  string
}

// ReadToolManifest parses a Proton toolmanifest.vdf.
func ReadToolManifest(r io.Reader) (ToolManifest, error) {
	document, err := vdf.Parse(r)
	if err != nil {
		return ToolManifest{}, fmt.Errorf("%w: %v", ErrToolManifest, err)
	}
	raw, ok := document["manifest"]
	if !ok {
		return ToolManifest{}, fmt.Errorf("%w: missing manifest object", ErrToolManifest)
	}
	fields, ok := raw.(map[string]interface{})
	if !ok {
		return ToolManifest{}, fmt.Errorf("%w: manifest is not an object", ErrToolManifest)
	}
	manifest := ToolManifest{
		Version:                 field(fields, "version"),
		CommandLine:             field(fields, "commandline"),
		UseSessions:             truthy(field(fields, "use_sessions")),
		UseToolSubprocessReaper: truthy(field(fields, "use_tool_subprocess_reaper")),
		CompatManagerLayerName:  field(fields, "compatmanager_layer_name"),
	}
	if rawID := field(fields, "require_tool_appid"); rawID != "" {
		id, err := strconv.ParseUint(rawID, 10, 32)
		if err != nil {
			return ToolManifest{}, fmt.Errorf("%w: require_tool_appid %q", ErrToolManifest, rawID)
		}
		manifest.RequireToolAppID = uint32(id)
	}
	return manifest, nil
}

// LoadToolManifest reads toolmanifest.vdf from path.
func LoadToolManifest(path string) (ToolManifest, error) {
	file, err := os.Open(path)
	if err != nil {
		return ToolManifest{}, fmt.Errorf("%w: open %s: %v", ErrToolManifest, path, err)
	}
	defer file.Close()
	return ReadToolManifest(file)
}

func truthy(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
