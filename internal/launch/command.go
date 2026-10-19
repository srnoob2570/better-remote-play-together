package launch

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/srnoob2570/better-rpt/internal/domain"
)

// BuildCommand turns a validated LaunchSpec into an exec.Cmd. It never uses a
// shell, never expands variables or globs, and never starts the process.
func BuildCommand(ctx context.Context, spec domain.LaunchSpec, baseEnv []string) (*exec.Cmd, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := spec.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCommand, err)
	}
	if spec.AppID == domain.SpaceWarAppID {
		return nil, fmt.Errorf("%w: SpaceWar cannot be a target", ErrCommand)
	}
	for _, argument := range spec.Arguments {
		if strings.Contains(strings.ToLower(argument), "%command%") {
			return nil, fmt.Errorf("%w: argument uses %%command%%", ErrLaunchTemplate)
		}
		if strings.ContainsRune(argument, 0) {
			return nil, fmt.Errorf("%w: argument contains NUL", ErrCommand)
		}
	}
	if strings.TrimSpace(spec.InstallDir) == "" || !filepath.IsAbs(spec.InstallDir) {
		return nil, fmt.Errorf("%w: install directory must be absolute", ErrCommand)
	}
	installDir, err := canonicalDir(spec.InstallDir)
	if err != nil {
		return nil, fmt.Errorf("%w: install directory: %v", ErrCommand, err)
	}
	executable, err := validateExecutable(installDir, spec.Executable, spec.Kind == domain.KindNative)
	if err != nil {
		return nil, err
	}
	workingDir := spec.WorkingDir
	if strings.TrimSpace(workingDir) == "" {
		workingDir = installDir
	}
	workingDir, err = validateWorkingDir(installDir, workingDir)
	if err != nil {
		return nil, err
	}

	env := []string{
		"SteamAppId=" + strconv.FormatUint(uint64(spec.AppID), 10),
		"SteamGameId=" + strconv.FormatUint(uint64(spec.AppID), 10),
	}
	var command *exec.Cmd
	switch spec.Kind {
	case domain.KindNative:
		command = exec.CommandContext(ctx, executable, spec.Arguments...)
	case domain.KindProton:
		protonPath, protonEnv, err := protonEnvironment(spec, installDir)
		if err != nil {
			return nil, err
		}
		env = append(env, protonEnv...)
		arguments := make([]string, 0, len(spec.Arguments)+2)
		arguments = append(arguments, "run", executable)
		arguments = append(arguments, spec.Arguments...)
		command = exec.CommandContext(ctx, protonPath, arguments...)
	default:
		return nil, fmt.Errorf("%w: unsupported kind %s", ErrCommand, spec.Kind)
	}
	command.Dir = workingDir
	command.Env = mergeEnvironment(baseEnv, env)
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return command, nil
}

func protonEnvironment(spec domain.LaunchSpec, installDir string) (string, []string, error) {
	steamRoot, err := canonicalDir(spec.SteamRoot)
	if err != nil {
		return "", nil, fmt.Errorf("%w: Steam root: %v", ErrCommand, err)
	}
	protonPath, err := validateProtonPath(spec.ProtonPath, steamRoot, spec.LibraryPaths)
	if err != nil {
		return "", nil, err
	}
	libraries, err := validateLibraries(spec.LibraryPaths, installDir)
	if err != nil {
		return "", nil, err
	}
	compatData, err := validateCompatData(steamRoot, spec.CompatDataPath, spec.AppID)
	if err != nil {
		return "", nil, err
	}
	libraryPaths := make([]string, 0, len(libraries))
	for _, library := range libraries {
		libraryPaths = append(libraryPaths, filepath.Join(library, "steamapps"))
	}
	env := []string{
		"STEAM_COMPAT_APP_ID=" + strconv.FormatUint(uint64(spec.AppID), 10),
		"STEAM_COMPAT_CLIENT_INSTALL_PATH=" + steamRoot,
		"STEAM_COMPAT_CLIENT_INSTALL=" + steamRoot,
		"STEAM_COMPAT_DATA_PATH=" + compatData,
		"WINEPREFIX=" + filepath.Join(compatData, "pfx"),
		"STEAM_COMPAT_INSTALL_PATH=" + installDir,
		"STEAM_COMPAT_LIBRARY_PATHS=" + strings.Join(libraryPaths, string(os.PathListSeparator)),
	}
	return protonPath, env, nil
}

func validateExecutable(base, path string, requireExecutable bool) (string, error) {
	if strings.TrimSpace(path) == "" || !filepath.IsAbs(path) {
		return "", fmt.Errorf("%w: executable path must be absolute", ErrExecutableInvalid)
	}
	real, err := filepath.EvalSymlinks(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", fmt.Errorf("%w: %s", ErrExecutableMissing, path)
		}
		return "", fmt.Errorf("%w: %s: %v", ErrExecutableInvalid, path, err)
	}
	info, err := os.Stat(real)
	if err != nil {
		return "", fmt.Errorf("%w: %s: %v", ErrExecutableInvalid, path, err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("%w: %s is not a regular file", ErrExecutableInvalid, path)
	}
	if requireExecutable && info.Mode().Perm()&0o111 == 0 {
		return "", fmt.Errorf("%w: %s is not executable", ErrExecutableInvalid, path)
	}
	if err := ensureWithin(base, real); err != nil {
		return "", err
	}
	return real, nil
}

func validateWorkingDir(installDir, path string) (string, error) {
	if strings.TrimSpace(path) == "" || !filepath.IsAbs(path) {
		return "", fmt.Errorf("%w: working directory must be absolute", ErrWorkingDir)
	}
	real, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", fmt.Errorf("%w: %s: %v", ErrWorkingDir, path, err)
	}
	info, err := os.Stat(real)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("%w: %s is not a directory", ErrWorkingDir, path)
	}
	if err := ensureWithin(installDir, real); err != nil {
		return "", fmt.Errorf("%w: %v", ErrWorkingDir, err)
	}
	return real, nil
}

func validateProtonPath(path, steamRoot string, libraryPaths []string) (string, error) {
	if strings.TrimSpace(path) == "" || !filepath.IsAbs(path) {
		return "", fmt.Errorf("%w: Proton path must be absolute", ErrExecutableInvalid)
	}
	real, err := filepath.EvalSymlinks(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", fmt.Errorf("%w: %s", ErrExecutableMissing, path)
		}
		return "", fmt.Errorf("%w: %s: %v", ErrExecutableInvalid, path, err)
	}
	info, err := os.Stat(real)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
		return "", fmt.Errorf("%w: Proton executable is invalid: %s", ErrExecutableInvalid, path)
	}
	if isWithin(filepath.Join(steamRoot, "compatibilitytools.d"), real) {
		return real, nil
	}
	for _, root := range libraryPaths {
		library, err := canonicalDir(root)
		if err != nil {
			continue
		}
		if isWithin(filepath.Join(library, "steamapps", "common"), real) {
			return real, nil
		}
	}
	return "", fmt.Errorf("%w: Proton path is outside Steam: %s", ErrCommand, path)
}

func validateLibraries(paths []string, installDir string) ([]string, error) {
	if len(paths) == 0 {
		return nil, fmt.Errorf("%w: no Steam libraries", ErrCommand)
	}
	result := make([]string, 0, len(paths))
	seen := make(map[string]bool)
	installAllowed := false
	for _, path := range paths {
		if strings.TrimSpace(path) == "" || !filepath.IsAbs(path) {
			return nil, fmt.Errorf("%w: library path must be absolute: %q", ErrCommand, path)
		}
		real, err := canonicalDir(path)
		if err != nil {
			return nil, fmt.Errorf("%w: library %s: %v", ErrCommand, path, err)
		}
		if seen[real] {
			continue
		}
		seen[real] = true
		result = append(result, real)
		if isWithin(filepath.Join(real, "steamapps", "common"), installDir) {
			installAllowed = true
		}
	}
	if !installAllowed {
		return nil, fmt.Errorf("%w: install directory is outside the selected libraries", ErrCommand)
	}
	return result, nil
}

func validateCompatData(steamRoot, compatData string, appID uint32) (string, error) {
	if strings.TrimSpace(compatData) == "" || !filepath.IsAbs(compatData) {
		return "", fmt.Errorf("%w: compat data path must be absolute", ErrCommand)
	}
	clean := filepath.Clean(compatData)
	expected := filepath.Join(filepath.Clean(steamRoot), "steamapps", "compatdata", strconv.FormatUint(uint64(appID), 10))
	if clean != expected {
		return "", fmt.Errorf("%w: compat data path %q, want %q", ErrCommand, clean, expected)
	}
	base := filepath.Join(filepath.Clean(steamRoot), "steamapps", "compatdata")
	if info, err := os.Lstat(clean); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("%w: compat data path is a symlink", ErrCommand)
		}
		if !info.IsDir() {
			return "", fmt.Errorf("%w: %s is not a directory", ErrCompatDataUnwritable, clean)
		}
		if err := accessWritable(clean); err != nil {
			return "", fmt.Errorf("%w: %s: %v", ErrCompatDataUnwritable, clean, err)
		}
		return clean, nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return "", fmt.Errorf("%w: %s: %v", ErrCompatDataUnwritable, clean, err)
	}

	current := filepath.Dir(clean)
	for {
		info, err := os.Stat(current)
		if err == nil {
			if !info.IsDir() {
				return "", fmt.Errorf("%w: %s is not a directory", ErrCompatDataUnwritable, current)
			}
			real, err := filepath.EvalSymlinks(current)
			if err != nil || !isWithin(base, real) {
				return "", fmt.Errorf("%w: %s escapes compatdata", ErrCommand, current)
			}
			if err := accessWritable(real); err != nil {
				return "", fmt.Errorf("%w: %s: %v", ErrCompatDataUnwritable, current, err)
			}
			return clean, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return "", fmt.Errorf("%w: %s: %v", ErrCompatDataUnwritable, current, err)
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return "", fmt.Errorf("%w: no writable parent for %s", ErrCompatDataUnwritable, clean)
}

func accessWritable(path string) error {
	// Linux W_OK and X_OK values; syscall.Access exposes the call but not the
	// constants in Go's syscall package.
	const (
		writeAccess   = 0x2
		executeAccess = 0x1
	)
	return syscall.Access(path, writeAccess|executeAccess)
}

func canonicalDir(path string) (string, error) {
	real, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(real)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s is not a directory", path)
	}
	return real, nil
}

func isWithin(base, path string) bool {
	relative, err := filepath.Rel(base, path)
	if err != nil {
		return false
	}
	return relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(os.PathSeparator)))
}

func mergeEnvironment(base, overrides []string) []string {
	if base == nil {
		base = os.Environ()
	}
	result := make([]string, 0, len(base)+len(overrides))
	index := make(map[string]int, len(base)+len(overrides))
	add := func(entry string) {
		key := entry
		if split := strings.IndexByte(entry, '='); split >= 0 {
			key = entry[:split]
		}
		if position, ok := index[key]; ok {
			result[position] = entry
			return
		}
		index[key] = len(result)
		result = append(result, entry)
	}
	for _, entry := range base {
		add(entry)
	}
	for _, entry := range overrides {
		add(entry)
	}
	return result
}
