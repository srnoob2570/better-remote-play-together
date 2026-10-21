// Package art resolves Steam library artwork without exposing filesystem
// paths to the frontend. Only files under appcache/librarycache/<appid> are
// considered, and the resolved path must stay inside the canonical app
// directory even after following symlinks.
package art

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/srnoob2570/better-rpt/internal/domain"
)

// Kind is the artwork variant requested by the frontend.
type Kind string

const (
	Cover Kind = "cover"
	Icon  Kind = "icon"
)

// MaxFileSize is the largest artwork file served by the endpoint. Steam cover
// artwork is far smaller; the limit prevents an oversized cache file from
// exhausting memory or bandwidth.
const MaxFileSize int64 = 8 << 20

var (
	// ErrNotFound means no artwork exists for the game.
	ErrNotFound = errors.New("art: artwork not found")
	// ErrUnsafe means a candidate escaped the allowlisted app directory or was
	// not a regular file.
	ErrUnsafe = errors.New("art: candidate is outside the library cache")
	// ErrTooLarge means the candidate exceeded MaxFileSize.
	ErrTooLarge = errors.New("art: artwork exceeds the size limit")
)

var coverNames = []string{
	"library_hero.jpg",
	"library_header.jpg",
	"header.jpg",
	"library_capsule.jpg",
	"library_header_latam.jpg",
	"library_600x900.jpg",
}

// ParseKind validates a frontend artwork kind.
func ParseKind(value string) (Kind, bool) {
	switch Kind(value) {
	case Cover:
		return Cover, true
	case Icon:
		return Icon, true
	default:
		return "", false
	}
}

// CoverPath returns the best cover for game, or ErrNotFound. The path is
// canonical and is guaranteed to be a regular file inside the allowlisted app
// directory.
func CoverPath(game domain.Game) (string, error) {
	return resolve(game, Cover)
}

// IconPath returns the best icon for game, or ErrNotFound.
func IconPath(game domain.Game) (string, error) {
	return resolve(game, Icon)
}

func resolve(game domain.Game, kind Kind) (string, error) {
	if game.AppID == 0 {
		return "", ErrNotFound
	}

	unsafe := false
	tooLarge := false
	for _, root := range artRoots(game) {
		canonicalRoot, ok := canonicalDir(root)
		if !ok {
			continue
		}
		base := filepath.Join(canonicalRoot, "appcache", "librarycache", strconv.FormatUint(uint64(game.AppID), 10))
		appDir, ok := canonicalDir(base)
		if !ok || appDir != base {
			// The app directory itself must not be a symlink: following it
			// could move the allowlisted tree outside the Steam cache.
			continue
		}
		var found string
		forEachCandidate(base, kind, func(candidate string) bool {
			path, err := validate(base, candidate)
			if err == nil {
				found = path
				return true
			}
			switch {
			case errors.Is(err, ErrTooLarge):
				tooLarge = true
			case errors.Is(err, ErrUnsafe):
				unsafe = true
			}
			return false
		})
		if found != "" {
			return found, nil
		}
	}
	if tooLarge {
		return "", ErrTooLarge
	}
	if unsafe {
		return "", ErrUnsafe
	}
	return "", ErrNotFound
}

func artRoots(game domain.Game) []string {
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
	unique := make([]string, 0, len(roots))
	for _, root := range roots {
		clean := filepath.Clean(root)
		if seen[clean] {
			continue
		}
		seen[clean] = true
		unique = append(unique, clean)
	}
	return unique
}

// forEachCandidate visits artwork candidates in preference order. It stops
// when visit returns true, so the cache directory is only globbed when a
// higher-preference candidate was not usable.
func forEachCandidate(base string, kind Kind, visit func(string) bool) {
	if kind == Icon {
		for _, candidate := range iconCandidates(base) {
			if visit(candidate) {
				return
			}
		}
		return
	}

	for _, name := range coverNames {
		if visit(filepath.Join(base, name)) {
			return
		}
		matches, err := filepath.Glob(filepath.Join(base, "*", name))
		if err != nil {
			continue
		}
		sort.Strings(matches)
		for _, match := range matches {
			if visit(match) {
				return
			}
		}
	}
}

func iconCandidates(base string) []string {
	out := make([]string, 0, 3)
	entries, err := os.ReadDir(base)
	if err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && isSteamIconName(entry.Name()) {
				out = append(out, filepath.Join(base, entry.Name()))
			}
		}
	}
	for _, name := range []string{"icon.jpg", "icon.png"} {
		out = append(out, filepath.Join(base, name))
	}
	return out
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

func canonicalDir(dir string) (string, bool) {
	absolute, err := filepath.Abs(dir)
	if err != nil {
		return "", false
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", false
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.IsDir() {
		return "", false
	}
	return filepath.Clean(resolved), true
}

func validate(base, candidate string) (string, error) {
	absolute, err := filepath.Abs(candidate)
	if err != nil {
		return "", ErrNotFound
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", ErrNotFound
	}
	relative, err := filepath.Rel(base, resolved)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		return "", ErrUnsafe
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", ErrNotFound
	}
	if !info.Mode().IsRegular() {
		return "", ErrUnsafe
	}
	if info.Size() > MaxFileSize {
		return "", ErrTooLarge
	}
	return resolved, nil
}
