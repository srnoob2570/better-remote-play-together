package steam

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/srnoob2570/better-rpt/internal/domain"
	"github.com/srnoob2570/better-rpt/internal/launch"
	"github.com/srnoob2570/better-rpt/internal/steam/appinfo"
	"github.com/srnoob2570/better-rpt/internal/steam/compat"
	"github.com/srnoob2570/better-rpt/internal/steam/library"
)

// Library is one readable Steam library.
type Library struct {
	Root      string
	SteamApps string
}

// Warning records a non-fatal problem in one library or manifest. A broken
// secondary library must not hide the games from the other libraries.
type Warning struct {
	Path string
	Err  error
}

func (w Warning) Error() string {
	if w.Err == nil {
		return w.Path
	}
	if w.Path == "" {
		return w.Err.Error()
	}
	return fmt.Sprintf("%s: %v", w.Path, w.Err)
}

func (w Warning) Unwrap() error {
	return w.Err
}

// Catalog is the result of a manual refresh.
type Catalog struct {
	Installation Installation
	Libraries    []Library
	Games        []domain.Game
	Warnings     []Warning
}

// Open detects Steam and builds its catalog.
func Open(ctx context.Context) (Catalog, error) {
	installation, err := Detect(ctx)
	if err != nil {
		return Catalog{}, err
	}
	return Load(ctx, installation)
}

// Load builds the catalog for an already-detected installation. Library and
// manifest failures are collected in Warnings; only a failure to read the
// primary steamapps directory is fatal.
func Load(ctx context.Context, installation Installation) (Catalog, error) {
	if err := ctx.Err(); err != nil {
		return Catalog{}, err
	}
	if installation.Variant != VariantNative {
		return Catalog{}, &UnsupportedError{Variant: installation.Variant, Root: installation.Root}
	}
	root, err := canonicalDir(installation.Root)
	if err != nil {
		return Catalog{}, fmt.Errorf("%w: %v", domain.ErrLibraryUnreadable, err)
	}
	if _, err := canonicalDir(filepath.Join(root, "steamapps")); err != nil {
		return Catalog{}, fmt.Errorf("%w: %v", domain.ErrLibraryUnreadable, err)
	}

	appInfoPath := filepath.Join(root, "appcache", "appinfo.vdf")
	appInfo, err := appinfo.Open(appInfoPath)
	if err != nil {
		return Catalog{}, err
	}
	catalog := Catalog{Installation: Installation{Root: root, Variant: VariantNative}}
	roots, warnings := discoverRoots(root)
	catalog.Warnings = append(catalog.Warnings, warnings...)
	resolver := compat.NewResolver(root, roots)
	for _, warning := range resolver.Warnings() {
		catalog.Warnings = append(catalog.Warnings, Warning{Path: warning.Path, Err: warning.Err})
	}
	seenRoots := make(map[string]bool)
	for _, rawRoot := range roots {
		if err := ctx.Err(); err != nil {
			return catalog, err
		}
		lib, games, warnings := loadLibrary(ctx, rawRoot, appInfo, resolver, root, roots)
		if lib == nil {
			catalog.Warnings = append(catalog.Warnings, warnings...)
			continue
		}
		if seenRoots[lib.Root] {
			continue
		}
		seenRoots[lib.Root] = true
		catalog.Warnings = append(catalog.Warnings, warnings...)
		catalog.Libraries = append(catalog.Libraries, *lib)
		catalog.Games = append(catalog.Games, games...)
	}
	for _, warning := range appInfo.Warnings() {
		catalog.Warnings = append(catalog.Warnings, Warning{Path: appInfoPath, Err: warning})
	}
	catalog.Games = dedupeGames(catalog.Games)
	return catalog, nil
}

func discoverRoots(primary string) ([]string, []Warning) {
	primary = filepath.Clean(primary)
	roots := []string{primary}
	seen := map[string]bool{primary: true}
	file := filepath.Join(primary, "steamapps", "libraryfolders.vdf")
	f, err := os.Open(file)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return roots, nil
		}
		return roots, []Warning{{Path: file, Err: fmt.Errorf("%w: %v", domain.ErrLibraryUnreadable, err)}}
	}
	defer func() { _ = f.Close() }()

	paths, err := library.ParseLibraryFolders(f)
	if err != nil {
		return roots, []Warning{{Path: file, Err: err}}
	}
	warnings := make([]Warning, 0)
	for _, path := range paths {
		if !filepath.IsAbs(path) {
			warnings = append(warnings, Warning{
				Path: file,
				Err:  fmt.Errorf("%w: relative library path %q", domain.ErrVDFMalformed, path),
			})
			continue
		}
		clean := filepath.Clean(path)
		if seen[clean] {
			continue
		}
		seen[clean] = true
		roots = append(roots, clean)
	}
	return roots, warnings
}

func loadLibrary(ctx context.Context, rawRoot string, appInfo *appinfo.File, resolver *compat.Resolver, steamRoot string, roots []string) (*Library, []domain.Game, []Warning) {
	root, err := canonicalDir(rawRoot)
	if err != nil {
		return nil, nil, []Warning{{Path: rawRoot, Err: fmt.Errorf("%w: %v", domain.ErrLibraryUnreadable, err)}}
	}
	apps, err := canonicalDir(filepath.Join(root, "steamapps"))
	if err != nil {
		return nil, nil, []Warning{{Path: root, Err: fmt.Errorf("%w: %v", domain.ErrLibraryUnreadable, err)}}
	}
	entries, err := os.ReadDir(apps)
	if err != nil {
		return nil, nil, []Warning{{Path: apps, Err: fmt.Errorf("%w: %v", domain.ErrLibraryUnreadable, err)}}
	}
	lib := &Library{Root: root, SteamApps: apps}
	common := filepath.Join(apps, "common")
	games := make([]domain.Game, 0)
	warnings := make([]Warning, 0)
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			warnings = append(warnings, Warning{Path: apps, Err: err})
			return lib, games, warnings
		}
		name := entry.Name()
		if entry.IsDir() || !strings.HasPrefix(name, "appmanifest_") || !strings.HasSuffix(name, ".acf") {
			continue
		}
		path := filepath.Join(apps, name)
		manifest, err := library.ReadManifest(path)
		if err != nil {
			warnings = append(warnings, Warning{Path: path, Err: err})
			continue
		}
		installDir, err := safeJoin(common, manifest.InstallDir)
		if err != nil {
			warnings = append(warnings, Warning{Path: path, Err: err})
			continue
		}
		game := domain.Game{
			AppID:       manifest.AppID,
			Name:        manifest.Name,
			InstallDir:  installDir,
			LibraryPath: root,
			SizeBytes:   manifest.SizeBytes,
			Kind:        domain.KindUnknown,
			Status:      domain.StatusUnknown,
		}
		if !manifest.Installed {
			game.Status = domain.StatusMissing
			games = append(games, game)
			continue
		}
		app, hasAppInfo := appInfo.App(manifest.AppID)
		if hasAppInfo && strings.EqualFold(strings.TrimSpace(app.Common.Type), "Tool") {
			continue
		}
		if hasAppInfo && strings.TrimSpace(app.Common.Name) != "" {
			game.Name = app.Common.Name
		}
		info, err := os.Stat(installDir)
		switch {
		case err == nil && info.IsDir():
		case err == nil:
			game.Status = domain.StatusMissing
		case errors.Is(err, fs.ErrNotExist):
			game.Status = domain.StatusMissing
		default:
			game.Status = domain.StatusError
			warnings = append(warnings, Warning{Path: installDir, Err: fmt.Errorf("%w: %v", domain.ErrLibraryUnreadable, err)})
		}
		if game.Status == domain.StatusMissing || game.Status == domain.StatusError {
			games = append(games, game)
			continue
		}
		if !hasAppInfo {
			game.Status = domain.StatusIncompatible
			game.Problem = appinfo.ErrAppNotFound
			games = append(games, game)
			continue
		}
		selected, spec, selectErr := selectLaunch(app, installDir, manifest.AppID, resolver, steamRoot)
		if selectErr != nil {
			game.Status = domain.StatusIncompatible
			game.Problem = selectErr
			games = append(games, game)
			continue
		}
		game.Kind = selected.Kind
		game.Launch = &spec
		if selected.Kind == domain.KindNative && !selected.Restricted {
			game.Status = domain.StatusReady
		} else {
			game.Status = domain.StatusUnknown
		}
		games = append(games, game)
	}
	return lib, games, warnings
}

// selectLaunch applies the exact-mapping rule before preferring a native Linux
// launch, then falls back to Windows/Proton only when native is unavailable.
func selectLaunch(app appinfo.App, installDir string, appID uint32, resolver *compat.Resolver, steamRoot string) (launch.Selected, domain.LaunchSpec, error) {
	if resolver == nil {
		return launch.Selected{}, domain.LaunchSpec{}, fmt.Errorf("%w: compatibility resolver is missing", launch.ErrCommand)
	}
	forced := resolver.ForcedProton(appID)
	exact, hasExact := resolver.Exact(appID)
	if forced {
		selected, err := launch.SelectOS(app, installDir, launch.OSWindows)
		if err != nil {
			return launch.Selected{}, domain.LaunchSpec{}, err
		}
		resolution, err := resolver.Resolve(appID)
		if err != nil {
			return launch.Selected{}, domain.LaunchSpec{}, err
		}
		spec := selected.Spec(appID)
		fillLaunchSpec(&spec, installDir, steamRoot, resolver.LibraryRoots)
		spec.ProtonPath = resolution.ProtonPath()
		spec.ProtonName = resolution.ProtonName()
		return selected, spec, nil
	}
	if hasExact && exact.Native() {
		selected, err := launch.SelectOS(app, installDir, launch.OSLinux)
		if err != nil {
			return launch.Selected{}, domain.LaunchSpec{}, err
		}
		spec := selected.Spec(appID)
		fillLaunchSpec(&spec, installDir, steamRoot, resolver.LibraryRoots)
		return selected, spec, nil
	}

	selected, nativeErr := launch.SelectOS(app, installDir, launch.OSLinux)
	if nativeErr == nil {
		spec := selected.Spec(appID)
		fillLaunchSpec(&spec, installDir, steamRoot, resolver.LibraryRoots)
		return selected, spec, nil
	}
	selected, windowsErr := launch.SelectOS(app, installDir, launch.OSWindows)
	if windowsErr != nil {
		if !errors.Is(nativeErr, launch.ErrNoLaunchForOS) {
			return launch.Selected{}, domain.LaunchSpec{}, nativeErr
		}
		return launch.Selected{}, domain.LaunchSpec{}, windowsErr
	}
	resolution, err := resolver.Resolve(appID)
	if err != nil {
		return launch.Selected{}, domain.LaunchSpec{}, err
	}
	spec := selected.Spec(appID)
	fillLaunchSpec(&spec, installDir, steamRoot, resolver.LibraryRoots)
	spec.ProtonPath = resolution.ProtonPath()
	spec.ProtonName = resolution.ProtonName()
	return selected, spec, nil
}

func fillLaunchSpec(spec *domain.LaunchSpec, installDir, steamRoot string, roots []string) {
	spec.InstallDir = installDir
	spec.SteamRoot = steamRoot
	spec.LibraryPaths = append([]string(nil), roots...)
	if spec.Kind == domain.KindProton {
		spec.CompatDataPath = filepath.Join(steamRoot, "steamapps", "compatdata", strconv.FormatUint(uint64(spec.AppID), 10))
	}
}

func safeJoin(base, relative string) (string, error) {
	if strings.TrimSpace(relative) == "" || filepath.IsAbs(relative) {
		return "", fmt.Errorf("%w: unsafe install dir %q", domain.ErrManifestCorrupt, relative)
	}
	clean := filepath.Clean(relative)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%w: install dir %q escapes steamapps/common", domain.ErrManifestCorrupt, relative)
	}
	full := filepath.Join(base, clean)
	check, err := filepath.Rel(base, full)
	if err != nil || check == ".." || strings.HasPrefix(check, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%w: install dir %q escapes steamapps/common", domain.ErrManifestCorrupt, relative)
	}
	return full, nil
}

func dedupeGames(games []domain.Game) []domain.Game {
	byID := make(map[uint32]int, len(games))
	result := make([]domain.Game, 0, len(games))
	for _, game := range games {
		if index, ok := byID[game.AppID]; ok {
			if gameRank(game.Status) > gameRank(result[index].Status) {
				result[index] = game
			}
			continue
		}
		byID[game.AppID] = len(result)
		result = append(result, game)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].AppID < result[j].AppID
	})
	return result
}

func gameRank(status domain.Status) int {
	switch status {
	case domain.StatusReady:
		return 4
	case domain.StatusUnknown:
		return 3
	case domain.StatusMissing:
		return 2
	case domain.StatusError:
		return 1
	default:
		return 0
	}
}
