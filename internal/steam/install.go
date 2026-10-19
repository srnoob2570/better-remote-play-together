// Package steam detects a native Steam installation and builds the catalog of
// installed games from its libraries. It only reads Steam files.
package steam

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/srnoob2570/better-rpt/internal/domain"
)

// Variant identifies how Steam was installed. Only VariantNative is supported
// for launching; the other variants are detected so the UI can explain why.
type Variant uint8

const (
	VariantNative Variant = iota
	VariantFlatpak
	VariantSnap
)

func (v Variant) String() string {
	switch v {
	case VariantNative:
		return "native"
	case VariantFlatpak:
		return "flatpak"
	case VariantSnap:
		return "snap"
	default:
		return "unknown"
	}
}

// Installation is a detected Steam root.
type Installation struct {
	Root    string
	Variant Variant
}

// UnsupportedError reports a Flatpak or Snap installation. It wraps
// domain.ErrSteamUnsupported so callers can choose the "unsupported" UI state.
type UnsupportedError struct {
	Variant Variant
	Root    string
}

func (e *UnsupportedError) Error() string {
	return fmt.Sprintf("steam %s installation at %s is not supported", e.Variant, e.Root)
}

func (e *UnsupportedError) Unwrap() error {
	return domain.ErrSteamUnsupported
}

type rootCandidate struct {
	path    string
	variant Variant
}

// Detect looks for a native Steam root first, then for Flatpak/Snap variants.
// It returns domain.ErrSteamNotFound when nothing looks like a Steam root and
// *UnsupportedError when only a non-native variant is present.
func Detect(ctx context.Context) (Installation, error) {
	if err := ctx.Err(); err != nil {
		return Installation{}, err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return Installation{}, fmt.Errorf("%w: %v", domain.ErrSteamNotFound, err)
	}

	native := []string{
		filepath.Join(home, ".local", "share", "Steam"),
		filepath.Join(home, ".steam", "steam"),
		filepath.Join(home, ".steam", "root"),
	}
	if dataHome := os.Getenv("XDG_DATA_HOME"); filepath.IsAbs(dataHome) {
		native = append([]string{filepath.Join(dataHome, "Steam")}, native...)
	}
	for _, path := range native {
		if root, ok := steamRoot(path); ok {
			return Installation{Root: root, Variant: VariantNative}, nil
		}
	}

	unsupported := []rootCandidate{
		{filepath.Join(home, ".var", "app", "com.valvesoftware.Steam", ".local", "share", "Steam"), VariantFlatpak},
		{filepath.Join(home, ".var", "app", "com.valvesoftware.Steam", ".steam", "steam"), VariantFlatpak},
		{filepath.Join(home, ".var", "app", "com.valvesoftware.Steam", ".steam", "root"), VariantFlatpak},
		{filepath.Join(home, "snap", "steam", "common", ".local", "share", "Steam"), VariantSnap},
		{filepath.Join(home, "snap", "steam", "common", ".steam", "steam"), VariantSnap},
		{filepath.Join(home, "snap", "steam", "common", ".steam", "root"), VariantSnap},
		{filepath.Join(home, "snap", "steam", "current", ".local", "share", "Steam"), VariantSnap},
	}
	for _, candidate := range unsupported {
		if root, ok := steamRoot(candidate.path); ok {
			return Installation{}, &UnsupportedError{Variant: candidate.variant, Root: root}
		}
	}

	return Installation{}, fmt.Errorf("%w: checked native Steam locations under %s", domain.ErrSteamNotFound, home)
}

// steamRoot returns the canonical root when path has a readable steamapps
// directory. It follows symlinks, which is why ~/.steam/steam resolves to the
// same root as ~/.local/share/Steam.
func steamRoot(path string) (string, bool) {
	if strings.TrimSpace(path) == "" {
		return "", false
	}
	root, err := canonicalDir(path)
	if err != nil {
		return "", false
	}
	apps, err := os.Stat(filepath.Join(root, "steamapps"))
	if err != nil || !apps.IsDir() {
		return "", false
	}
	return root, true
}

func canonicalDir(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(real)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s is not a directory", real)
	}
	return real, nil
}
