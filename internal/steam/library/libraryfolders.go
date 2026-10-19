// Package library reads Steam's textual library metadata: libraryfolders.vdf
// and appmanifest_*.acf. It never writes to the Steam installation.
package library

import (
	"fmt"
	"io"
	"sort"
	"strconv"

	"github.com/srnoob2570/better-rpt/internal/domain"
	textvdf "github.com/srnoob2570/better-rpt/internal/steam/vdf"
)

type folderEntry struct {
	index int
	path  string
}

// ParseLibraryFolders parses both the modern libraryfolders.vdf (numeric keys
// whose values are objects with a path field) and the legacy format (numeric
// keys whose values are paths). Metadata keys are ignored.
func ParseLibraryFolders(r io.Reader) ([]string, error) {
	doc, err := textvdf.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrVDFMalformed, err)
	}
	root, ok := doc["libraryfolders"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("%w: missing libraryfolders root", domain.ErrVDFMalformed)
	}

	entries := make([]folderEntry, 0, len(root))
	for key, value := range root {
		index, err := strconv.Atoi(key)
		if err != nil {
			continue // TimeNextStatsReport, ContentStatsID, etc.
		}
		if index < 0 {
			return nil, fmt.Errorf("%w: negative library index %q", domain.ErrVDFMalformed, key)
		}
		var path string
		switch v := value.(type) {
		case string:
			path = v
		case map[string]interface{}:
			raw, ok := v["path"].(string)
			if !ok {
				return nil, fmt.Errorf("%w: library %q has no path", domain.ErrVDFMalformed, key)
			}
			path = raw
		default:
			return nil, fmt.Errorf("%w: library %q has unexpected type %T", domain.ErrVDFMalformed, key, value)
		}
		if path == "" {
			return nil, fmt.Errorf("%w: library %q has an empty path", domain.ErrVDFMalformed, key)
		}
		entries = append(entries, folderEntry{index: index, path: path})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].index < entries[j].index
	})
	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		paths = append(paths, entry.path)
	}
	return paths, nil
}
