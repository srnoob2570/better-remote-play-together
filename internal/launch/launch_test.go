package launch

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/srnoob2570/better-rpt/internal/domain"
	"github.com/srnoob2570/better-rpt/internal/steam/appinfo"
)

func writeFile(t *testing.T, path string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte("fixture\n"), mode); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatalf("chmod: %v", err)
	}
}

func app(launches ...appinfo.Launch) appinfo.App {
	return appinfo.App{
		ID:        900001,
		InfoState: 2,
		Common:    appinfo.Common{Name: "Game", Type: "Game", InstallDir: "Game"},
		Launch:    launches,
	}
}

func TestSplitArguments(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{"simple", "--foo bar", []string{"--foo", "bar"}},
		{"quoted", `--logfile "output log.txt" -timestamps`, []string{"--logfile", "output log.txt", "-timestamps"}},
		{"empty quoted", `""`, []string{""}},
		{"literal env", `STEAM_RUNTIME=0`, []string{"STEAM_RUNTIME=0"}},
		{"percent literal", `-path %USERPROFILE% -pm`, []string{"-path", "%USERPROFILE%", "-pm"}},
		{"windows path", `C:\Games\game.exe --x`, []string{`C:\Games\game.exe`, "--x"}},
		{"escaped space", `a\ b`, []string{"a b"}},
		{"escaped quote", `a\"b`, []string{`a"b`}},
		{"escaped backslash", `a\\b`, []string{`a\b`}},
		{"trailing backslash", `a\`, []string{`a\`}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SplitArguments(tt.in)
			if err != nil {
				t.Fatalf("split: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("args = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestSplitArgumentsUnterminated(t *testing.T) {
	if _, err := SplitArguments(`"unterminated`); !errors.Is(err, ErrArguments) {
		t.Fatalf("error = %v, want ErrArguments", err)
	}
}

func TestSelectNative(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "bin", "game.x86_64"), 0o755)
	writeFile(t, filepath.Join(dir, "game.exe"), 0o755)
	app := app(
		appinfo.Launch{Key: "0", OSList: "linux", Arch: "64", Executable: "bin/game.x86_64", Arguments: "--foo bar", WorkingDir: "bin", Type: "default"},
		appinfo.Launch{Key: "1", OSList: "windows", Executable: "game.exe", Type: "default"},
	)
	selected, err := Select(app, dir)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if selected.Kind != domain.KindNative || selected.OS != OSLinux {
		t.Fatalf("selected = %+v, want native linux", selected)
	}
	if selected.Executable != filepath.Join(dir, "bin", "game.x86_64") {
		t.Fatalf("executable = %q", selected.Executable)
	}
	if !reflect.DeepEqual(selected.Arguments, []string{"--foo", "bar"}) {
		t.Fatalf("arguments = %#v", selected.Arguments)
	}
	if selected.WorkingDir != filepath.Join(dir, "bin") {
		t.Fatalf("working dir = %q", selected.WorkingDir)
	}
	if err := selected.Spec(900001).Validate(); err != nil {
		t.Fatalf("native spec: %v", err)
	}
}

func TestSelectProtonCaseInsensitive(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "Game.exe"), 0o644)
	app := app(appinfo.Launch{Key: "0", OSList: "windows", Executable: "game.EXE", Type: "default"})
	selected, err := Select(app, dir)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if selected.Kind != domain.KindProton || selected.OS != OSWindows {
		t.Fatalf("selected = %+v, want proton windows", selected)
	}
	if selected.Executable != filepath.Join(dir, "Game.exe") {
		t.Fatalf("executable = %q", selected.Executable)
	}
}

func TestSelectNativeMissingFallsBackToWindows(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "Game.exe"), 0o644)
	app := app(
		appinfo.Launch{Key: "0", OSList: "linux", Executable: "missing.x86_64", Type: "default"},
		appinfo.Launch{Key: "1", OSList: "windows", Executable: "Game.exe", Type: "default"},
	)
	selected, err := Select(app, dir)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if selected.Kind != domain.KindProton {
		t.Fatalf("kind = %s, want proton", selected.Kind)
	}
}

func TestSelectPrefersDefault(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "default.x86_64"), 0o755)
	writeFile(t, filepath.Join(dir, "none.x86_64"), 0o755)
	app := app(
		appinfo.Launch{Key: "0", OSList: "linux", Executable: "none.x86_64", Type: "none"},
		appinfo.Launch{Key: "1", OSList: "linux", Executable: "default.x86_64", Type: "default"},
	)
	selected, err := Select(app, dir)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if selected.Executable != filepath.Join(dir, "default.x86_64") {
		t.Fatalf("executable = %q, want default", selected.Executable)
	}
}

func TestSelectTemplateOnly(t *testing.T) {
	dir := t.TempDir()
	app := app(appinfo.Launch{Key: "0", OSList: "linux", Executable: "game.x86_64", Arguments: "--x %command%", Type: "default"})
	if _, err := Select(app, dir); !errors.Is(err, ErrLaunchTemplate) {
		t.Fatalf("error = %v, want ErrLaunchTemplate", err)
	}
}

func TestSelectOptionOnly(t *testing.T) {
	dir := t.TempDir()
	app := app(appinfo.Launch{Key: "0", OSList: "linux", Executable: "game.x86_64", Type: "option1"})
	if _, err := Select(app, dir); !errors.Is(err, ErrNoLaunchForOS) {
		t.Fatalf("error = %v, want ErrNoLaunchForOS", err)
	}
}

func TestSelectRejectsNonGame(t *testing.T) {
	dir := t.TempDir()
	app := app()
	app.Common.Type = "Tool"
	if _, err := Select(app, dir); !errors.Is(err, ErrNotGame) {
		t.Fatalf("error = %v, want ErrNotGame", err)
	}
}

func TestSelectRejectsInfoState(t *testing.T) {
	dir := t.TempDir()
	app := app()
	app.InfoState = 1
	if _, err := Select(app, dir); !errors.Is(err, ErrInfoState) {
		t.Fatalf("error = %v, want ErrInfoState", err)
	}
}

func TestSelectNoLaunchConfig(t *testing.T) {
	dir := t.TempDir()
	app := app()
	if _, err := Select(app, dir); !errors.Is(err, ErrNoLaunchConfig) {
		t.Fatalf("error = %v, want ErrNoLaunchConfig", err)
	}
}

func TestSelectMissingExecutable(t *testing.T) {
	dir := t.TempDir()
	app := app(appinfo.Launch{Key: "0", OSList: "linux", Executable: "missing.x86_64", Type: "default"})
	if _, err := Select(app, dir); !errors.Is(err, ErrExecutableMissing) {
		t.Fatalf("error = %v, want ErrExecutableMissing", err)
	}
}

func TestSelectPathEscape(t *testing.T) {
	dir := t.TempDir()
	app := app(appinfo.Launch{Key: "0", OSList: "linux", Executable: "../evil", Type: "default"})
	if _, err := Select(app, dir); !errors.Is(err, domain.ErrPathEscape) {
		t.Fatalf("error = %v, want ErrPathEscape", err)
	}
}

func TestSelectAmbiguousCase(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "Game.exe"), 0o644)
	writeFile(t, filepath.Join(dir, "game.exe"), 0o644)
	app := app(appinfo.Launch{Key: "0", OSList: "windows", Executable: "GAME.EXE", Type: "default"})
	if _, err := Select(app, dir); !errors.Is(err, ErrAmbiguousExecutable) {
		t.Fatalf("error = %v, want ErrAmbiguousExecutable", err)
	}
}

func TestSelectWorkingDirMissing(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "game.x86_64"), 0o755)
	app := app(appinfo.Launch{Key: "0", OSList: "linux", Executable: "game.x86_64", WorkingDir: "missing", Type: "default"})
	if _, err := Select(app, dir); !errors.Is(err, ErrWorkingDir) {
		t.Fatalf("error = %v, want ErrWorkingDir", err)
	}
}

func TestSelectSymlinkEscape(t *testing.T) {
	dir := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside")
	writeFile(t, outside, 0o755)
	if err := os.Symlink(outside, filepath.Join(dir, "game.x86_64")); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	app := app(appinfo.Launch{Key: "0", OSList: "linux", Executable: "game.x86_64", Type: "default"})
	if _, err := Select(app, dir); !errors.Is(err, domain.ErrPathEscape) {
		t.Fatalf("error = %v, want ErrPathEscape", err)
	}
}

func TestSelectPrefersUnrestricted(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "free.x86_64"), 0o755)
	writeFile(t, filepath.Join(dir, "beta.x86_64"), 0o755)
	app := app(
		appinfo.Launch{Key: "0", OSList: "linux", Executable: "beta.x86_64", Type: "default", BetaKeys: []string{"beta"}},
		appinfo.Launch{Key: "1", OSList: "linux", Executable: "free.x86_64", Type: "none"},
	)
	selected, err := Select(app, dir)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if selected.Restricted || selected.Executable != filepath.Join(dir, "free.x86_64") {
		t.Fatalf("selected = %+v, want unrestricted free launch", selected)
	}
}

func TestSelectRejectsServer(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "server.x86_64"), 0o755)
	app := app(appinfo.Launch{Key: "server", OSList: "linux", Executable: "server.x86_64", Type: "default"})
	if _, err := Select(app, dir); !errors.Is(err, ErrNoLaunchForOS) {
		t.Fatalf("error = %v, want ErrNoLaunchForOS", err)
	}
}

func TestSelectInfersWindows(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "Game.exe"), 0o644)
	app := app(appinfo.Launch{Key: "0", Executable: "Game.exe", Type: "default"})
	selected, err := Select(app, dir)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if selected.Kind != domain.KindProton || selected.OS != OSWindows {
		t.Fatalf("selected = %+v, want inferred proton", selected)
	}
}

func TestSelectBetaRestricted(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "game.x86_64"), 0o755)
	app := app(appinfo.Launch{Key: "0", OSList: "linux", Executable: "game.x86_64", Type: "default", BetaKeys: []string{"beta"}})
	selected, err := Select(app, dir)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if !selected.Restricted || selected.Kind != domain.KindNative {
		t.Fatalf("selected = %+v, want restricted native", selected)
	}
}

func TestSelectArchPrefers64(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "32.x86_64"), 0o755)
	writeFile(t, filepath.Join(dir, "64.x86_64"), 0o755)
	app := app(
		appinfo.Launch{Key: "0", OSList: "linux", Executable: "32.x86_64", Arch: "32", Type: "default"},
		appinfo.Launch{Key: "1", OSList: "linux", Executable: "64.x86_64", Arch: "64", Type: "default"},
	)
	selected, err := Select(app, dir)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if selected.Executable != filepath.Join(dir, "64.x86_64") {
		t.Fatalf("executable = %q, want 64-bit", selected.Executable)
	}
}
