package library

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/srnoob2570/better-rpt/internal/domain"
	textvdf "github.com/srnoob2570/better-rpt/internal/steam/vdf"
)

// Manifest is the subset of appmanifest_*.acf that the catalog needs.
type Manifest struct {
	AppID      uint32
	Name       string
	InstallDir string // relative to steamapps/common
	StateFlags uint32
	SizeBytes  uint64
	Installed  bool
}

// ReadManifest opens and parses one appmanifest file.
func ReadManifest(path string) (Manifest, error) {
	f, err := os.Open(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("%w: open %s: %v", domain.ErrLibraryUnreadable, path, err)
	}
	defer f.Close()
	return ParseManifest(filepath.Base(path), f)
}

// ParseManifest parses an appmanifest. filename is used to validate the app ID
// and must look like appmanifest_<id>.acf.
func ParseManifest(filename string, r io.Reader) (Manifest, error) {
	fileAppID, err := appIDFromFilename(filename)
	if err != nil {
		return Manifest{}, err
	}
	doc, err := textvdf.Parse(r)
	if err != nil {
		return Manifest{}, fmt.Errorf("%w: %s: %v", domain.ErrManifestCorrupt, filename, err)
	}
	root, ok := doc["AppState"].(map[string]interface{})
	if !ok {
		return Manifest{}, fmt.Errorf("%w: %s: missing AppState", domain.ErrManifestCorrupt, filename)
	}

	rawAppID, ok := stringField(root, "appid")
	if !ok {
		return Manifest{}, fmt.Errorf("%w: %s: missing appid", domain.ErrManifestCorrupt, filename)
	}
	parsedAppID, err := strconv.ParseUint(strings.TrimSpace(rawAppID), 10, 32)
	if err != nil || uint32(parsedAppID) != fileAppID {
		return Manifest{}, fmt.Errorf("%w: %s: appid %q does not match filename", domain.ErrManifestCorrupt, filename, rawAppID)
	}

	name, ok := stringField(root, "name")
	if !ok || strings.TrimSpace(name) == "" {
		return Manifest{}, fmt.Errorf("%w: %s: missing name", domain.ErrManifestCorrupt, filename)
	}
	installDir, ok := stringField(root, "installdir")
	if !ok || strings.TrimSpace(installDir) == "" {
		return Manifest{}, fmt.Errorf("%w: %s: missing installdir", domain.ErrManifestCorrupt, filename)
	}

	rawFlags, ok := stringField(root, "StateFlags")
	if !ok {
		return Manifest{}, fmt.Errorf("%w: %s: missing StateFlags", domain.ErrManifestCorrupt, filename)
	}
	flags, err := strconv.ParseUint(strings.TrimSpace(rawFlags), 10, 32)
	if err != nil {
		return Manifest{}, fmt.Errorf("%w: %s: invalid StateFlags %q", domain.ErrManifestCorrupt, filename, rawFlags)
	}

	var size uint64
	if rawSize, ok := stringField(root, "SizeOnDisk"); ok && strings.TrimSpace(rawSize) != "" {
		size, err = strconv.ParseUint(strings.TrimSpace(rawSize), 10, 64)
		if err != nil {
			return Manifest{}, fmt.Errorf("%w: %s: invalid SizeOnDisk %q", domain.ErrManifestCorrupt, filename, rawSize)
		}
	}

	return Manifest{
		AppID:      fileAppID,
		Name:       strings.TrimSpace(name),
		InstallDir: strings.TrimSpace(installDir),
		StateFlags: uint32(flags),
		SizeBytes:  size,
		Installed:  flags&4 != 0,
	}, nil
}

func appIDFromFilename(filename string) (uint32, error) {
	base := filepath.Base(filename)
	if !strings.HasPrefix(base, "appmanifest_") || !strings.HasSuffix(base, ".acf") {
		return 0, fmt.Errorf("%w: invalid manifest filename %q", domain.ErrManifestCorrupt, base)
	}
	raw := strings.TrimSuffix(strings.TrimPrefix(base, "appmanifest_"), ".acf")
	id, err := strconv.ParseUint(raw, 10, 32)
	if err != nil || id == 0 {
		return 0, fmt.Errorf("%w: invalid app id in %q", domain.ErrManifestCorrupt, base)
	}
	return uint32(id), nil
}

func stringField(root map[string]interface{}, key string) (string, bool) {
	value, ok := root[key].(string)
	return value, ok
}
