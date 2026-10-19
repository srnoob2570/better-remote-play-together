package compat

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Source says where a Proton-like tool was installed.
type Source uint8

const (
	SourceOfficial Source = iota
	SourceCompatibilityTools
)

func (s Source) String() string {
	if s == SourceCompatibilityTools {
		return "compatibilitytools.d"
	}
	return "steamapps/common"
}

// Install is one Proton-capable tool directory.
type Install struct {
	Name       string
	Dir        string
	ProtonPath string
	Source     Source
	Official   bool
	Manifest   ToolManifest
}

// Resolution ties a mapping to an installed Proton.
type Resolution struct {
	Mapping Mapping
	Install Install
}

// ProtonPath is the direct executable to run.
func (r Resolution) ProtonPath() string { return r.Install.ProtonPath }

// ProtonName is the installed directory name.
func (r Resolution) ProtonName() string { return r.Install.Name }

// RequireToolAppID is diagnostic: the MVP runs Proton directly and does not
// nest the SteamLinuxRuntime wrapper.
func (r Resolution) RequireToolAppID() uint32 { return r.Install.Manifest.RequireToolAppID }

// Resolver discovers Proton tools and applies config.vdf mappings.
type Resolver struct {
	SteamRoot    string
	LibraryRoots []string
	Mappings     Mappings
	installs     []Install
	warnings     []Warning
}

// NewResolver loads config.vdf mappings and discovers Proton installations.
// Missing or malformed config and bad tool directories are warnings; callers
// can still list native games and explain why Proton is unavailable.
func NewResolver(steamRoot string, libraryRoots []string) *Resolver {
	resolver := &Resolver{
		SteamRoot:    filepath.Clean(steamRoot),
		LibraryRoots: normalizeRoots(steamRoot, libraryRoots),
		Mappings:     Mappings{Exact: make(map[uint32]Mapping)},
	}
	configPath := filepath.Join(resolver.SteamRoot, "config", "config.vdf")
	if mappings, err := LoadMappings(configPath); err != nil {
		resolver.warnings = append(resolver.warnings, Warning{Path: configPath, Err: fmt.Errorf("%w: %v", ErrConfig, err)})
	} else {
		resolver.Mappings = mappings
	}
	installs, warnings := DiscoverInstalls(resolver.SteamRoot, resolver.LibraryRoots)
	resolver.installs = installs
	resolver.warnings = append(resolver.warnings, warnings...)
	return resolver
}

// Installs returns a copy of discovered Proton tools.
func (r *Resolver) Installs() []Install {
	if r == nil {
		return nil
	}
	return append([]Install(nil), r.installs...)
}

// Warnings returns non-fatal discovery/configuration problems.
func (r *Resolver) Warnings() []Warning {
	if r == nil {
		return nil
	}
	return append([]Warning(nil), r.warnings...)
}

// Exact returns the per-app mapping, if one exists.
func (r *Resolver) Exact(appID uint32) (Mapping, bool) {
	if r == nil {
		return Mapping{}, false
	}
	mapping, ok := r.Mappings.Exact[appID]
	return mapping, ok
}

// ForcedProton reports whether an exact mapping other than native selects
// Proton. A global mapping is not used to override an available native launch.
func (r *Resolver) ForcedProton(appID uint32) bool {
	mapping, ok := r.Exact(appID)
	return ok && !mapping.Native()
}

// MappingFor returns the exact mapping, then the global mapping.
func (r *Resolver) MappingFor(appID uint32) (Mapping, bool) {
	if r == nil {
		return Mapping{}, false
	}
	return r.Mappings.For(appID)
}

// Resolve resolves the mapping that applies to appID. Native or missing
// mappings return ErrNativeTool / ErrProtonNotFound.
func (r *Resolver) Resolve(appID uint32) (Resolution, error) {
	mapping, ok := r.MappingFor(appID)
	if !ok {
		return Resolution{}, fmt.Errorf("%w: no CompatToolMapping for %d", ErrProtonNotFound, appID)
	}
	if mapping.Native() {
		return Resolution{}, fmt.Errorf("%w: mapping %q is native", ErrNativeTool, mapping.Name)
	}
	install, err := r.ResolveName(mapping.Name)
	if err != nil {
		return Resolution{}, err
	}
	return Resolution{Mapping: mapping, Install: install}, nil
}

// ResolveName finds an installed tool by exact name or by a documented alias.
func (r *Resolver) ResolveName(name string) (Install, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Install{}, fmt.Errorf("%w: empty Proton name", ErrProtonNotFound)
	}
	if strings.EqualFold(name, "native") {
		return Install{}, fmt.Errorf("%w: mapping is native", ErrNativeTool)
	}
	if install, ok, err := r.exactInstall(name); err != nil {
		return Install{}, err
	} else if ok {
		return install, nil
	}

	lower := strings.ToLower(name)
	switch {
	case lower == "proton-stable":
		return r.newestStable()
	case lower == "proton-experimental" || lower == "proton_experimental":
		return r.uniqueOfficial("Proton - Experimental")
	default:
		if major, ok := protonMajorAlias(lower); ok {
			return r.uniqueOfficial(fmt.Sprintf("Proton %d.0", major))
		}
	}
	return Install{}, fmt.Errorf("%w: configured tool %q is not installed", ErrProtonNotFound, name)
}

func (r *Resolver) exactInstall(name string) (Install, bool, error) {
	var exact []Install
	var folded []Install
	for _, install := range r.installs {
		if install.Name == name {
			exact = append(exact, install)
		} else if strings.EqualFold(install.Name, name) {
			folded = append(folded, install)
		}
	}
	matches := exact
	if len(matches) == 0 {
		matches = folded
	}
	switch len(matches) {
	case 0:
		return Install{}, false, nil
	case 1:
		return matches[0], true, nil
	default:
		return Install{}, false, fmt.Errorf("%w: name %q matches %d installations", ErrProtonAmbiguous, name, len(matches))
	}
}

func (r *Resolver) uniqueOfficial(name string) (Install, error) {
	var matches []Install
	for _, install := range r.installs {
		if install.Official && strings.EqualFold(install.Name, name) {
			matches = append(matches, install)
		}
	}
	switch len(matches) {
	case 0:
		return Install{}, fmt.Errorf("%w: official Proton %q is not installed", ErrProtonNotFound, name)
	case 1:
		return matches[0], nil
	default:
		return Install{}, fmt.Errorf("%w: official Proton %q matches %d installations", ErrProtonAmbiguous, name, len(matches))
	}
}

func (r *Resolver) newestStable() (Install, error) {
	best := -1
	var matches []Install
	for _, install := range r.installs {
		if !install.Official {
			continue
		}
		major, minor, ok := officialVersion(install.Name)
		if !ok {
			continue
		}
		score := major*1000 + minor
		switch {
		case score > best:
			best = score
			matches = []Install{install}
		case score == best:
			matches = append(matches, install)
		}
	}
	switch len(matches) {
	case 0:
		return Install{}, fmt.Errorf("%w: no official stable Proton is installed", ErrProtonNotFound)
	case 1:
		return matches[0], nil
	default:
		return Install{}, fmt.Errorf("%w: newest official stable Proton is ambiguous", ErrProtonAmbiguous)
	}
}

func protonMajorAlias(name string) (int, bool) {
	index := strings.IndexAny(name, "_-")
	if index < 0 || !strings.HasPrefix(name[:index], "proton") {
		return 0, false
	}
	rest := name[index+1:]
	if strings.Contains(rest, ".") {
		return 0, false
	}
	major, err := strconv.Atoi(rest)
	if err != nil || major <= 0 {
		return 0, false
	}
	return major, true
}

var officialProtonPattern = regexp.MustCompile(`^Proton ([0-9]+)\.([0-9]+)$`)

func officialVersion(name string) (int, int, bool) {
	match := officialProtonPattern.FindStringSubmatch(name)
	if match == nil {
		return 0, 0, false
	}
	major, _ := strconv.Atoi(match[1])
	minor, _ := strconv.Atoi(match[2])
	return major, minor, true
}

func normalizeRoots(steamRoot string, roots []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(roots)+1)
	add := func(root string) {
		root = strings.TrimSpace(root)
		if root == "" {
			return
		}
		root = filepath.Clean(root)
		if !filepath.IsAbs(root) || seen[root] {
			return
		}
		seen[root] = true
		result = append(result, root)
	}
	add(steamRoot)
	for _, root := range roots {
		add(root)
	}
	sort.Strings(result)
	return result
}

// DiscoverInstalls scans compatibilitytools.d in the Steam root and Proton*
// directories under every library's steamapps/common. It requires an executable
// "proton" and a readable toolmanifest.vdf.
func DiscoverInstalls(steamRoot string, libraryRoots []string) ([]Install, []Warning) {
	installs := make([]Install, 0)
	warnings := make([]Warning, 0)
	seenDirs := make(map[string]bool)

	addDir := func(dir string, source Source, official bool) {
		canonical, err := filepath.EvalSymlinks(dir)
		if err != nil {
			warnings = append(warnings, Warning{Path: dir, Err: fmt.Errorf("%w: %v", ErrConfig, err)})
			return
		}
		if seenDirs[canonical] {
			return
		}
		protonPath := filepath.Join(canonical, "proton")
		info, err := os.Stat(protonPath)
		if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
			if official || source == SourceCompatibilityTools {
				// A directory without a usable proton is not a Proton tool; do
				// not turn every unrelated compatibility directory into a warning.
				return
			}
			return
		}
		manifest, err := LoadToolManifest(filepath.Join(canonical, "toolmanifest.vdf"))
		if err != nil {
			warnings = append(warnings, Warning{Path: canonical, Err: err})
			return
		}
		seenDirs[canonical] = true
		installs = append(installs, Install{
			Name:       filepath.Base(canonical),
			Dir:        canonical,
			ProtonPath: protonPath,
			Source:     source,
			Official:   official,
			Manifest:   manifest,
		})
	}

	if steamRoot != "" {
		toolsDir := filepath.Join(steamRoot, "compatibilitytools.d")
		entries, err := os.ReadDir(toolsDir)
		switch {
		case err == nil:
			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}
				addDir(filepath.Join(toolsDir, entry.Name()), SourceCompatibilityTools, false)
			}
		case errors.Is(err, fs.ErrNotExist):
		default:
			warnings = append(warnings, Warning{Path: toolsDir, Err: fmt.Errorf("%w: %v", ErrConfig, err)})
		}
	}

	for _, root := range libraryRoots {
		common := filepath.Join(root, "steamapps", "common")
		entries, err := os.ReadDir(common)
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				warnings = append(warnings, Warning{Path: common, Err: fmt.Errorf("%w: %v", ErrConfig, err)})
			}
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "Proton") {
				continue
			}
			addDir(filepath.Join(common, entry.Name()), SourceOfficial, true)
		}
	}

	sort.SliceStable(installs, func(i, j int) bool {
		if installs[i].Name == installs[j].Name {
			return installs[i].Dir < installs[j].Dir
		}
		return installs[i].Name < installs[j].Name
	})
	return installs, warnings
}
