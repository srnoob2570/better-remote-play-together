package ui

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/srnoob2570/better-rpt/internal/domain"
)

// Steam keeps the library artwork in appcache next to the client. Covers are
// usually cached as a wide hero/header image and icons as a 40-hex .jpg at the
// root of the app's librarycache directory. The game's library path is the
// first place to look; the default client locations cover secondary libraries
// whose appcache lives under the main Steam install.

func steamArtDirs(game domain.Game) []string {
	roots := make([]string, 0, 4)
	if game.LibraryPath != "" {
		roots = append(roots, game.LibraryPath)
	}
	if home, err := os.UserHomeDir(); err == nil {
		roots = append(roots,
			filepath.Join(home, ".local", "share", "Steam"),
			filepath.Join(home, ".steam", "steam"),
			filepath.Join(home, ".var", "app", "com.valvesoftware.Steam", ".local", "share", "Steam"),
		)
	}

	seen := make(map[string]bool, len(roots))
	dirs := make([]string, 0, len(roots))
	for _, root := range roots {
		dir := filepath.Join(root, "appcache", "librarycache", strconv.FormatUint(uint64(game.AppID), 10))
		if seen[dir] {
			continue
		}
		seen[dir] = true
		dirs = append(dirs, dir)
	}
	return dirs
}

func steamCoverPath(game domain.Game) string {
	for _, dir := range steamArtDirs(game) {
		if path := coverIn(dir); path != "" {
			return path
		}
	}
	return ""
}

func coverIn(dir string) string {
	names := []string{
		"library_hero.jpg",
		"library_header.jpg",
		"header.jpg",
		"library_capsule.jpg",
		"library_header_latam.jpg",
		"library_600x900.jpg",
	}
	for _, name := range names {
		if path := existingFile(filepath.Join(dir, name)); path != "" {
			return path
		}
		matches, err := filepath.Glob(filepath.Join(dir, "*", name))
		if err != nil {
			continue
		}
		sort.Strings(matches)
		if path := existingFile(matches...); path != "" {
			return path
		}
	}
	return ""
}

func steamIconPath(game domain.Game) string {
	for _, dir := range steamArtDirs(game) {
		if path := iconIn(dir); path != "" {
			return path
		}
	}
	return ""
}

func iconIn(dir string) string {
	entries, err := os.ReadDir(dir)
	if err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && isSteamIconName(entry.Name()) {
				return filepath.Join(dir, entry.Name())
			}
		}
	}
	for _, name := range []string{"icon.jpg", "icon.png"} {
		if path := existingFile(filepath.Join(dir, name)); path != "" {
			return path
		}
	}
	return ""
}

func isSteamIconName(name string) bool {
	if len(name) != 44 || !strings.HasSuffix(name, ".jpg") {
		return false
	}
	for _, char := range name[:40] {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
			return false
		}
	}
	return true
}

func existingFile(paths ...string) string {
	for _, path := range paths {
		info, err := os.Stat(path)
		if err == nil && !info.IsDir() {
			return path
		}
	}
	return ""
}
