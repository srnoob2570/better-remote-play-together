package launch

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/srnoob2570/better-rpt/internal/domain"
	"github.com/srnoob2570/better-rpt/internal/steam/appinfo"
)

// OS is a target platform used to choose a launch entry.
type OS string

const (
	OSLinux   OS = "linux"
	OSWindows OS = "windows"
)

// Selected is a resolved target launch. The executable and working directory
// are absolute and verified to stay inside the install directory.
type Selected struct {
	Kind       domain.Kind
	OS         OS
	Executable string
	Arguments  []string
	WorkingDir string
	Launch     appinfo.Launch
	Restricted bool // the entry depends on a beta or DLC
}

// Spec converts the selected launch into the domain launch description. Proton
// resolution is filled by the compatibility layer; ProtonPath is empty here.
func (s Selected) Spec(appID uint32) domain.LaunchSpec {
	return domain.LaunchSpec{
		AppID:      appID,
		Kind:       s.Kind,
		Executable: s.Executable,
		Arguments:  append([]string(nil), s.Arguments...),
		WorkingDir: s.WorkingDir,
	}
}

type candidate struct {
	launch     appinfo.Launch
	os         OS
	restricted bool
}

// Select chooses and resolves one launch for app. It prefers a Linux/native
// launch, then falls back to a Windows/Proton launch if the native executable
// is missing or invalid. The caller resolves Proton itself later.
func Select(app appinfo.App, installDir string) (Selected, error) {
	var firstErr error
	for _, targetOS := range []OS{OSLinux, OSWindows} {
		selected, err := SelectOS(app, installDir, targetOS)
		if err == nil {
			return selected, nil
		}
		if firstErr == nil || errors.Is(firstErr, ErrNoLaunchForOS) {
			firstErr = err
		}
	}
	if firstErr != nil {
		return Selected{}, firstErr
	}
	return Selected{}, ErrNoLaunchForOS
}

// SelectOS chooses and resolves a launch restricted to one target OS. It is
// used when an explicit Steam compatibility mapping forces Proton even though
// a native launch exists.
func SelectOS(app appinfo.App, installDir string, target OS) (Selected, error) {
	if app.InfoState != 2 {
		return Selected{}, fmt.Errorf("%w: state %d", ErrInfoState, app.InfoState)
	}
	if !strings.EqualFold(strings.TrimSpace(app.Common.Type), "Game") {
		return Selected{}, fmt.Errorf("%w: common.type %q", ErrNotGame, app.Common.Type)
	}
	if len(app.Launch) == 0 {
		return Selected{}, ErrNoLaunchConfig
	}
	if strings.TrimSpace(installDir) == "" {
		return Selected{}, fmt.Errorf("%w: empty install directory", ErrExecutableInvalid)
	}

	candidates, err := candidatesForOS(app, target)
	if len(candidates) == 0 {
		if err != nil {
			return Selected{}, err
		}
		return Selected{}, ErrNoLaunchForOS
	}
	var firstErr error
	for _, candidate := range candidates {
		selected, resolveErr := resolveCandidate(candidate, installDir)
		if resolveErr == nil {
			return selected, nil
		}
		if firstErr == nil || errors.Is(firstErr, ErrNoLaunchForOS) {
			firstErr = resolveErr
		}
	}
	if firstErr != nil {
		return Selected{}, firstErr
	}
	return Selected{}, ErrNoLaunchForOS
}

func candidatesForOS(app appinfo.App, target OS) ([]candidate, error) {
	usable := make([]candidate, 0)
	restricted := make([]candidate, 0)
	var templateErr error

	for _, launch := range app.Launch {
		if !launchMatchesOS(launch, target) {
			continue
		}
		if launchBlocked(launch) {
			continue
		}
		if strings.TrimSpace(launch.Executable) == "" {
			continue
		}
		if strings.Contains(strings.ToLower(launch.Arguments), "%command%") {
			if templateErr == nil {
				templateErr = fmt.Errorf("%w: launch %q", ErrLaunchTemplate, launch.Key)
			}
			continue
		}
		c := candidate{
			launch:     launch,
			os:         target,
			restricted: len(launch.BetaKeys) > 0 || launch.OwnsDLC,
		}
		if c.restricted {
			restricted = append(restricted, c)
		} else {
			usable = append(usable, c)
		}
	}

	chosen := usable
	if len(chosen) == 0 {
		chosen = restricted
	}
	if len(chosen) == 0 {
		if templateErr != nil {
			return nil, templateErr
		}
		return nil, fmt.Errorf("%w: %s", ErrNoLaunchForOS, target)
	}
	sort.SliceStable(chosen, func(i, j int) bool {
		return candidateLess(chosen[i], chosen[j])
	})
	return chosen, nil
}

func candidateLess(a, b candidate) bool {
	if ar, br := typeRank(a.launch.Type), typeRank(b.launch.Type); ar != br {
		return ar < br
	}
	if ar, br := archRank(a.launch.Arch), archRank(b.launch.Arch); ar != br {
		return ar < br
	}
	return launchKeyLess(a.launch.Key, b.launch.Key)
}

func typeRank(typ string) int {
	switch strings.ToLower(strings.TrimSpace(typ)) {
	case "default":
		return 0
	case "", "none":
		return 1
	default:
		return 2
	}
}

func archRank(arch string) int {
	switch strings.ToLower(strings.TrimSpace(arch)) {
	case "64", "amd64", "x86_64":
		return 0
	case "":
		return 1
	default:
		return 2
	}
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

func launchBlocked(launch appinfo.Launch) bool {
	key := strings.ToLower(strings.TrimSpace(launch.Key))
	typ := strings.ToLower(strings.TrimSpace(launch.Type))
	if strings.HasPrefix(key, "option") || strings.HasPrefix(typ, "option") {
		return true
	}
	blocked := map[string]bool{
		"server": true, "editor": true, "config": true, "vr": true,
		"openxr": true, "manual": true, "safemode": true,
	}
	return blocked[key] || blocked[typ]
}

func launchMatchesOS(launch appinfo.Launch, target OS) bool {
	if list := strings.TrimSpace(launch.OSList); list != "" {
		for _, part := range strings.Split(list, ",") {
			if strings.EqualFold(strings.TrimSpace(part), string(target)) {
				return true
			}
		}
		return false
	}
	return inferOS(launch.Executable) == target
}

func inferOS(executable string) OS {
	ext := filepath.Ext(strings.ReplaceAll(executable, "\\", "/"))
	if strings.EqualFold(ext, ".exe") {
		return OSWindows
	}
	return OSLinux
}

func resolveCandidate(c candidate, installDir string) (Selected, error) {
	kind := domain.KindNative
	if c.os == OSWindows {
		kind = domain.KindProton
	}
	executable, err := resolveExecutable(installDir, c.launch.Executable, kind == domain.KindNative)
	if err != nil {
		return Selected{}, err
	}
	arguments, err := SplitArguments(c.launch.Arguments)
	if err != nil {
		return Selected{}, err
	}
	workingDir, err := resolveWorkingDir(installDir, c.launch.WorkingDir)
	if err != nil {
		return Selected{}, err
	}
	return Selected{
		Kind:       kind,
		OS:         c.os,
		Executable: executable,
		Arguments:  arguments,
		WorkingDir: workingDir,
		Launch:     c.launch,
		Restricted: c.restricted,
	}, nil
}

func resolveExecutable(installDir, raw string, requireExecutable bool) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return "", fmt.Errorf("%w: empty executable", ErrExecutableMissing)
	}
	path, err := resolvePath(installDir, raw)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrExecutableInvalid, err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("%w: not a regular file", ErrExecutableInvalid)
	}
	if requireExecutable && info.Mode().Perm()&0o111 == 0 {
		return "", fmt.Errorf("%w: no execute permission", ErrExecutableInvalid)
	}
	if err := ensureWithin(installDir, path); err != nil {
		return "", err
	}
	return path, nil
}

func resolveWorkingDir(installDir, raw string) (string, error) {
	clean := strings.TrimSpace(raw)
	if clean == "" || clean == "." {
		info, err := os.Stat(installDir)
		if err != nil || !info.IsDir() {
			return "", fmt.Errorf("%w: install directory is unavailable", ErrWorkingDir)
		}
		return installDir, nil
	}
	path, err := resolvePath(installDir, clean)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrWorkingDir, err)
	}
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("%w: %s", ErrWorkingDir, path)
	}
	if err := ensureWithin(installDir, path); err != nil {
		return "", fmt.Errorf("%w: %w", ErrWorkingDir, err)
	}
	return path, nil
}

func resolvePath(base, raw string) (string, error) {
	relative, err := relativePath(raw)
	if err != nil {
		return "", err
	}
	baseReal, err := filepath.EvalSymlinks(base)
	if err != nil {
		return "", fmt.Errorf("%w: install directory: %v", ErrExecutableInvalid, err)
	}
	exact := filepath.Join(baseReal, relative)
	if _, statErr := os.Stat(exact); statErr == nil {
		return exact, nil
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return "", fmt.Errorf("%w: %v", ErrExecutableInvalid, statErr)
	}
	return resolveCaseInsensitive(baseReal, relative)
}

func relativePath(raw string) (string, error) {
	normalized := strings.ReplaceAll(strings.TrimSpace(raw), "\\", "/")
	if normalized == "" {
		return "", fmt.Errorf("%w: empty path", domain.ErrPathEscape)
	}
	if filepath.IsAbs(normalized) || strings.HasPrefix(normalized, "/") {
		return "", fmt.Errorf("%w: absolute path %q", domain.ErrPathEscape, raw)
	}
	clean := filepath.Clean(normalized)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%w: path %q", domain.ErrPathEscape, raw)
	}
	return clean, nil
}

func resolveCaseInsensitive(base, relative string) (string, error) {
	current := base
	parts := strings.Split(relative, string(filepath.Separator))
	for _, part := range parts {
		entries, err := os.ReadDir(current)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return "", fmt.Errorf("%w: %s", ErrExecutableMissing, filepath.Join(current, part))
			}
			return "", fmt.Errorf("%w: %v", ErrExecutableInvalid, err)
		}
		matches := make([]string, 0, 1)
		for _, entry := range entries {
			if strings.EqualFold(entry.Name(), part) {
				matches = append(matches, entry.Name())
			}
		}
		switch len(matches) {
		case 0:
			return "", fmt.Errorf("%w: %s", ErrExecutableMissing, filepath.Join(current, part))
		case 1:
			current = filepath.Join(current, matches[0])
		default:
			return "", fmt.Errorf("%w: %s matches %v", ErrAmbiguousExecutable, filepath.Join(current, part), matches)
		}
	}
	return current, nil
}

func ensureWithin(base, path string) error {
	baseReal, err := filepath.EvalSymlinks(base)
	if err != nil {
		return fmt.Errorf("%w: install directory: %v", ErrExecutableInvalid, err)
	}
	pathReal, err := filepath.EvalSymlinks(path)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrExecutableInvalid, err)
	}
	relative, err := filepath.Rel(baseReal, pathReal)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return fmt.Errorf("%w: %s escapes %s", domain.ErrPathEscape, pathReal, baseReal)
	}
	return nil
}
