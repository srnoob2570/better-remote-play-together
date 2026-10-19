package launch

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/srnoob2570/better-rpt/internal/domain"
)

type commandFixture struct {
	steamRoot  string
	library    string
	installDir string
	proton     string
	compatData string
}

func newCommandFixture(t *testing.T) commandFixture {
	t.Helper()
	base := t.TempDir()
	library := filepath.Join(base, "library")
	steamRoot := filepath.Join(base, "steam")
	installDir := filepath.Join(library, "steamapps", "common", "Game")
	writeFile(t, filepath.Join(installDir, "Game.exe"), 0o644)
	writeFile(t, filepath.Join(installDir, "game"), 0o755)
	proton := filepath.Join(library, "steamapps", "common", "Proton Test", "proton")
	writeFile(t, proton, 0o755)
	compatData := filepath.Join(steamRoot, "steamapps", "compatdata", "900001")
	if err := os.MkdirAll(filepath.Dir(compatData), 0o700); err != nil {
		t.Fatalf("create compatdata parent: %v", err)
	}
	if err := os.Mkdir(compatData, 0o700); err != nil {
		t.Fatalf("create compatdata: %v", err)
	}
	return commandFixture{steamRoot: steamRoot, library: library, installDir: installDir, proton: proton, compatData: compatData}
}

func (f commandFixture) nativeSpec() domain.LaunchSpec {
	return domain.LaunchSpec{
		AppID:      900001,
		Kind:       domain.KindNative,
		Executable: filepath.Join(f.installDir, "game"),
		Arguments:  []string{"--fixture", "literal value", "%USERPROFILE%"},
		WorkingDir: f.installDir,
		InstallDir: f.installDir,
	}
}

func (f commandFixture) protonSpec() domain.LaunchSpec {
	return domain.LaunchSpec{
		AppID:          900001,
		Kind:           domain.KindProton,
		Executable:     filepath.Join(f.installDir, "Game.exe"),
		Arguments:      []string{"--fixture", "literal value"},
		WorkingDir:     f.installDir,
		InstallDir:     f.installDir,
		SteamRoot:      f.steamRoot,
		LibraryPaths:   []string{f.library},
		CompatDataPath: f.compatData,
		ProtonPath:     f.proton,
		ProtonName:     "Proton Test",
	}
}

func envValue(env []string, key string) (string, bool) {
	prefix := key + "="
	for _, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			return strings.TrimPrefix(entry, prefix), true
		}
	}
	return "", false
}

func TestBuildCommandNative(t *testing.T) {
	fixture := newCommandFixture(t)
	spec := fixture.nativeSpec()
	baseEnv := []string{
		"PATH=/usr/bin",
		"DISPLAY=:1",
		"WAYLAND_DISPLAY=wayland-0",
		"SteamAppId=480",
		"SteamGameId=480",
		"SteamOverlayGameId=480",
		"STEAM_COMPAT_MOUNTS=keep-me",
	}
	command, err := BuildCommand(context.Background(), spec, baseEnv)
	if err != nil {
		t.Fatalf("build native: %v", err)
	}
	wantArgs := append([]string{spec.Executable}, spec.Arguments...)
	if strings.Join(command.Args, "\x00") != strings.Join(wantArgs, "\x00") {
		t.Fatalf("args = %q, want %q", command.Args, wantArgs)
	}
	if command.Dir != fixture.installDir {
		t.Fatalf("dir = %q, want %q", command.Dir, fixture.installDir)
	}
	if command.SysProcAttr == nil || !command.SysProcAttr.Setpgid {
		t.Fatal("native command does not start a new process group")
	}
	for key, want := range map[string]string{
		"SteamAppId":          "900001",
		"SteamGameId":         "480",
		"SteamOverlayGameId":  "480",
		"DISPLAY":             ":1",
		"WAYLAND_DISPLAY":     "wayland-0",
		"STEAM_COMPAT_MOUNTS": "keep-me",
	} {
		if got, ok := envValue(command.Env, key); !ok || got != want {
			t.Fatalf("env %s = %q, %v; want %q", key, got, ok, want)
		}
	}
	if _, ok := envValue(command.Env, "STEAM_COMPAT_APP_ID"); ok {
		t.Fatal("native command unexpectedly has Proton env")
	}
}

func TestBuildCommandProton(t *testing.T) {
	fixture := newCommandFixture(t)
	spec := fixture.protonSpec()
	command, err := BuildCommand(context.Background(), spec, []string{
		"PATH=/usr/bin",
		"SteamAppId=480",
		"SteamGameId=480",
		"SteamOverlayGameId=480",
		"STEAM_COMPAT_APP_ID=480",
		"SteamClientLaunch=1",
	})
	if err != nil {
		t.Fatalf("build proton: %v", err)
	}
	wantArgs := append([]string{spec.ProtonPath, "run", spec.Executable}, spec.Arguments...)
	if strings.Join(command.Args, "\x00") != strings.Join(wantArgs, "\x00") {
		t.Fatalf("args = %q, want %q", command.Args, wantArgs)
	}
	if command.Dir != fixture.installDir {
		t.Fatalf("dir = %q", command.Dir)
	}
	if command.SysProcAttr == nil || !command.SysProcAttr.Setpgid {
		t.Fatal("proton command does not start a new process group")
	}
	for key, want := range map[string]string{
		"SteamAppId":                       "900001",
		"SteamGameId":                      "480",
		"SteamOverlayGameId":               "480",
		"STEAM_COMPAT_APP_ID":              "480",
		"STEAM_COMPAT_CLIENT_INSTALL_PATH": fixture.steamRoot,
		"STEAM_COMPAT_CLIENT_INSTALL":      fixture.steamRoot,
		"STEAM_COMPAT_DATA_PATH":           fixture.compatData,
		"WINEPREFIX":                       filepath.Join(fixture.compatData, "pfx"),
		"STEAM_COMPAT_INSTALL_PATH":        fixture.installDir,
		"STEAM_COMPAT_LIBRARY_PATHS":       filepath.Join(fixture.library, "steamapps"),
		"SteamClientLaunch":                "1",
	} {
		if got, ok := envValue(command.Env, key); !ok || got != want {
			t.Fatalf("env %s = %q, %v; want %q", key, got, ok, want)
		}
	}
	if _, ok := envValue(command.Env, "STEAM_COMPAT_MOUNTS"); ok {
		t.Fatal("BuildCommand added STEAM_COMPAT_MOUNTS")
	}
}

func TestBuildCommandUsesTargetSteamAppIdKeepsDonorSession(t *testing.T) {
	fixture := newCommandFixture(t)
	baseEnv := []string{
		"PATH=/usr/bin",
		"SteamAppId=480",
		"SteamGameId=480",
		"STEAM_COMPAT_APP_ID=480",
		"SteamOverlayGameId=480",
		"SteamClientLaunch=1",
	}
	cases := []struct {
		name string
		spec domain.LaunchSpec
	}{
		{name: "native", spec: fixture.nativeSpec()},
		{name: "proton", spec: fixture.protonSpec()},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			command, err := BuildCommand(context.Background(), testCase.spec, baseEnv)
			if err != nil {
				t.Fatalf("build %s: %v", testCase.name, err)
			}
			for key, want := range map[string]string{
				"SteamAppId":          "900001",
				"SteamGameId":         "480",
				"STEAM_COMPAT_APP_ID": "480",
				"SteamOverlayGameId":  "480",
				"SteamClientLaunch":   "1",
			} {
				if got, ok := envValue(command.Env, key); !ok || got != want {
					t.Fatalf("%s: env %s = %q, %v; want %q", testCase.name, key, got, ok, want)
				}
			}
			if got, ok := envValue(command.Env, "STEAM_COMPAT_DATA_PATH"); testCase.spec.Kind == domain.KindProton {
				if !ok || got != fixture.compatData {
					t.Fatalf("proton compatdata = %q, %v; want %q", got, ok, fixture.compatData)
				}
			} else if ok {
				t.Fatalf("native command unexpectedly has target compatdata %q", got)
			}
		})
	}
}

func TestBuildCommandRejectsTemplate(t *testing.T) {
	fixture := newCommandFixture(t)
	spec := fixture.nativeSpec()
	spec.Arguments = []string{"%COMMAND%"}
	if _, err := BuildCommand(context.Background(), spec, nil); !errors.Is(err, ErrLaunchTemplate) {
		t.Fatalf("error = %v, want ErrLaunchTemplate", err)
	}
}

func TestBuildCommandRejectsDonor(t *testing.T) {
	fixture := newCommandFixture(t)
	spec := fixture.nativeSpec()
	spec.AppID = domain.SpaceWarAppID
	if _, err := BuildCommand(context.Background(), spec, nil); !errors.Is(err, ErrCommand) {
		t.Fatalf("error = %v, want ErrCommand", err)
	}
}

func TestBuildCommandRejectsMissingExecutable(t *testing.T) {
	fixture := newCommandFixture(t)
	spec := fixture.nativeSpec()
	spec.Executable = filepath.Join(fixture.installDir, "missing")
	if _, err := BuildCommand(context.Background(), spec, nil); !errors.Is(err, ErrExecutableMissing) {
		t.Fatalf("error = %v, want ErrExecutableMissing", err)
	}
}

func TestBuildCommandRejectsNonExecutableNative(t *testing.T) {
	fixture := newCommandFixture(t)
	spec := fixture.nativeSpec()
	spec.Executable = filepath.Join(fixture.installDir, "Game.exe")
	if _, err := BuildCommand(context.Background(), spec, nil); !errors.Is(err, ErrExecutableInvalid) {
		t.Fatalf("error = %v, want ErrExecutableInvalid", err)
	}
}

func TestBuildCommandRejectsPathEscape(t *testing.T) {
	fixture := newCommandFixture(t)
	outside := filepath.Join(filepath.Dir(fixture.installDir), "outside")
	writeFile(t, outside, 0o755)
	spec := fixture.nativeSpec()
	spec.Executable = outside
	if _, err := BuildCommand(context.Background(), spec, nil); !errors.Is(err, domain.ErrPathEscape) {
		t.Fatalf("error = %v, want ErrPathEscape", err)
	}
}

func TestBuildCommandRejectsUnwritableCompatData(t *testing.T) {
	fixture := newCommandFixture(t)
	if err := os.Remove(fixture.compatData); err != nil {
		t.Fatalf("remove compatdata: %v", err)
	}
	if err := os.WriteFile(fixture.compatData, []byte("not a dir"), 0o600); err != nil {
		t.Fatalf("write compatdata file: %v", err)
	}
	if _, err := BuildCommand(context.Background(), fixture.protonSpec(), nil); !errors.Is(err, ErrCompatDataUnwritable) {
		t.Fatalf("error = %v, want ErrCompatDataUnwritable", err)
	}
}

func TestBuildCommandRejectsProtonOutsideTools(t *testing.T) {
	fixture := newCommandFixture(t)
	outside := filepath.Join(t.TempDir(), "proton")
	writeFile(t, outside, 0o755)
	spec := fixture.protonSpec()
	spec.ProtonPath = outside
	if _, err := BuildCommand(context.Background(), spec, nil); !errors.Is(err, ErrCommand) {
		t.Fatalf("error = %v, want ErrCommand", err)
	}
}

func TestBuildCommandRejectsInstallOutsideLibraries(t *testing.T) {
	fixture := newCommandFixture(t)
	outsideLibrary := t.TempDir()
	spec := fixture.protonSpec()
	spec.LibraryPaths = []string{outsideLibrary}
	if _, err := BuildCommand(context.Background(), spec, nil); !errors.Is(err, ErrCommand) {
		t.Fatalf("error = %v, want ErrCommand", err)
	}
}
