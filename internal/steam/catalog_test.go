package steam

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/srnoob2570/better-rpt/internal/domain"
	"github.com/srnoob2570/better-rpt/internal/launch"
	"github.com/srnoob2570/better-rpt/internal/steam/appinfo"
)

func materializeFixtures(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	dest := t.TempDir()
	cmd := exec.Command("bash", filepath.Join(repoRoot, "testdata", "steam", "materialize.sh"), dest)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("materialize fixtures: %v\n%s", err, output)
	}
	return dest
}

func homeWithSteam(t *testing.T, steamRoot string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", "")
	target := filepath.Join(home, ".local", "share", "Steam")
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		t.Fatalf("create home: %v", err)
	}
	if err := os.Symlink(steamRoot, target); err != nil {
		t.Fatalf("symlink Steam root: %v", err)
	}
}

func copyFile(t *testing.T, source, destination string) {
	t.Helper()
	data, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("read %s: %v", source, err)
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
		t.Fatalf("create %s: %v", filepath.Dir(destination), err)
	}
	if err := os.WriteFile(destination, data, 0o600); err != nil {
		t.Fatalf("write %s: %v", destination, err)
	}
}

func findGame(games []domain.Game, appID uint32) (domain.Game, bool) {
	for _, game := range games {
		if game.AppID == appID {
			return game, true
		}
	}
	return domain.Game{}, false
}

func TestOpenPrimaryAndSecondary(t *testing.T) {
	fixtures := materializeFixtures(t)
	homeWithSteam(t, filepath.Join(fixtures, "primary"))

	catalog, err := Open(context.Background())
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if len(catalog.Libraries) != 2 {
		t.Fatalf("libraries = %d, want 2: %+v", len(catalog.Libraries), catalog.Libraries)
	}
	if len(catalog.Games) != 10 {
		t.Fatalf("games = %d, want 10 after excluding two tools", len(catalog.Games))
	}
	if len(catalog.Warnings) != 0 {
		t.Fatalf("warnings = %v, want none", catalog.Warnings)
	}

	spacewar, ok := findGame(catalog.Games, domain.SpaceWarAppID)
	if !ok || spacewar.Name != "Spacewar" || spacewar.SizeBytes != 1024 || spacewar.Status != domain.StatusUnknown || spacewar.Kind != domain.KindProton {
		t.Fatalf("spacewar = %+v, ok=%v", spacewar, ok)
	}
	native, ok := findGame(catalog.Games, 900001)
	if !ok || native.Status != domain.StatusReady || native.Kind != domain.KindNative || native.Launch == nil || native.LibraryPath != filepath.Join(fixtures, "primary") {
		t.Fatalf("native game = %+v, ok=%v", native, ok)
	}
	if len(native.Launch.Arguments) != 1 || native.Launch.Arguments[0] != "--fixture" {
		t.Fatalf("native arguments = %v, want [--fixture]", native.Launch.Arguments)
	}
	proton, ok := findGame(catalog.Games, 900002)
	if !ok || proton.Status != domain.StatusUnknown || proton.Kind != domain.KindProton || proton.Launch == nil {
		t.Fatalf("proton game = %+v, ok=%v", proton, ok)
	}
	secondary, ok := findGame(catalog.Games, 900007)
	if !ok || secondary.Status != domain.StatusReady || secondary.Kind != domain.KindNative || secondary.LibraryPath != filepath.Join(fixtures, "secondary") {
		t.Fatalf("secondary game = %+v, ok=%v", secondary, ok)
	}
	notInstalled, ok := findGame(catalog.Games, 900005)
	if !ok || notInstalled.Status != domain.StatusMissing {
		t.Fatalf("not installed = %+v, ok=%v", notInstalled, ok)
	}
	for _, toolID := range []uint32{900003, 900004} {
		if _, ok := findGame(catalog.Games, toolID); ok {
			t.Fatalf("tool %d was not excluded", toolID)
		}
	}
	orphan, ok := findGame(catalog.Games, 900010)
	if !ok || orphan.Status != domain.StatusIncompatible || !errors.Is(orphan.Problem, appinfo.ErrAppNotFound) {
		t.Fatalf("orphan = %+v, ok=%v", orphan, ok)
	}
	for _, incompatible := range []struct {
		id  uint32
		err error
	}{
		{900006, launch.ErrExecutableMissing},
		{900009, launch.ErrNoLaunchConfig},
		{900011, launch.ErrLaunchTemplate},
		{900012, domain.ErrPathEscape},
	} {
		game, ok := findGame(catalog.Games, incompatible.id)
		if !ok || game.Status != domain.StatusIncompatible || !errors.Is(game.Problem, incompatible.err) {
			t.Fatalf("game %d = %+v, ok=%v, want incompatible with %v", incompatible.id, game, ok, incompatible.err)
		}
	}
	if _, ok := findGame(catalog.Games, 900008); ok {
		t.Fatal("legacy game leaked into the primary catalog")
	}
}

func TestOpenLegacyLibrary(t *testing.T) {
	fixtures := materializeFixtures(t)
	copyFile(t,
		filepath.Join(fixtures, "primary", "appcache", "appinfo.vdf"),
		filepath.Join(fixtures, "legacy", "appcache", "appinfo.vdf"),
	)
	homeWithSteam(t, filepath.Join(fixtures, "legacy"))

	catalog, err := Open(context.Background())
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if len(catalog.Libraries) != 1 {
		t.Fatalf("libraries = %d, want 1", len(catalog.Libraries))
	}
	legacy, ok := findGame(catalog.Games, 900008)
	if !ok || legacy.Name != "Legacy Game" || legacy.Status != domain.StatusReady || legacy.Kind != domain.KindNative {
		t.Fatalf("legacy game = %+v, ok=%v", legacy, ok)
	}
}

func TestOpenCorruptManifestIsolation(t *testing.T) {
	fixtures := materializeFixtures(t)
	homeWithSteam(t, filepath.Join(fixtures, "primary"))
	badPath := filepath.Join(fixtures, "primary", "steamapps", "appmanifest_900099.acf")
	if err := os.WriteFile(badPath, []byte(`"AppState" {`), 0o600); err != nil {
		t.Fatalf("write bad manifest: %v", err)
	}

	catalog, err := Open(context.Background())
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if len(catalog.Games) != 10 {
		t.Fatalf("games = %d, want 10 after one corrupt manifest", len(catalog.Games))
	}
	if len(catalog.Warnings) != 1 || !errors.Is(catalog.Warnings[0].Err, domain.ErrManifestCorrupt) {
		t.Fatalf("warnings = %v, want one manifest-corrupt warning", catalog.Warnings)
	}
	if _, ok := findGame(catalog.Games, 900001); !ok {
		t.Fatal("valid game disappeared with corrupt manifest")
	}
}

func TestOpenCorruptAppInfoRecordIsolation(t *testing.T) {
	fixtures := materializeFixtures(t)
	copyFile(t,
		filepath.Join(fixtures, "primary", "appcache", "appinfo_bad_key.vdf"),
		filepath.Join(fixtures, "primary", "appcache", "appinfo.vdf"),
	)
	homeWithSteam(t, filepath.Join(fixtures, "primary"))

	catalog, err := Open(context.Background())
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if len(catalog.Warnings) != 1 || !errors.Is(catalog.Warnings[0].Err, appinfo.ErrAppCorrupt) {
		t.Fatalf("warnings = %v, want one appinfo record warning", catalog.Warnings)
	}
	if _, ok := findGame(catalog.Games, 900001); !ok {
		t.Fatal("valid apps disappeared with one corrupt appinfo record")
	}
	if game, ok := findGame(catalog.Games, 900013); ok {
		t.Fatalf("corrupt app appeared in catalog: %+v", game)
	}
}

func TestOpenMalformedLibraryFoldersIsolation(t *testing.T) {
	fixtures := materializeFixtures(t)
	homeWithSteam(t, filepath.Join(fixtures, "primary"))
	libFile := filepath.Join(fixtures, "primary", "steamapps", "libraryfolders.vdf")
	if err := os.WriteFile(libFile, []byte(`"libraryfolders" {`), 0o600); err != nil {
		t.Fatalf("write libraryfolders: %v", err)
	}

	catalog, err := Open(context.Background())
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if len(catalog.Games) != 9 {
		t.Fatalf("games = %d, want 9 from the still-readable primary library", len(catalog.Games))
	}
	if len(catalog.Warnings) != 1 || !errors.Is(catalog.Warnings[0].Err, domain.ErrVDFMalformed) {
		t.Fatalf("warnings = %v, want one VDF warning", catalog.Warnings)
	}
	if _, ok := findGame(catalog.Games, 900001); !ok {
		t.Fatal("primary games disappeared with malformed libraryfolders")
	}
	if _, ok := findGame(catalog.Games, 900007); ok {
		t.Fatal("secondary game appeared without libraryfolders")
	}
}

func TestOpenMissingInstallDir(t *testing.T) {
	fixtures := materializeFixtures(t)
	homeWithSteam(t, filepath.Join(fixtures, "primary"))
	if err := os.RemoveAll(filepath.Join(fixtures, "primary", "steamapps", "common", "Native Game")); err != nil {
		t.Fatalf("remove install dir: %v", err)
	}

	catalog, err := Open(context.Background())
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	native, ok := findGame(catalog.Games, 900001)
	if !ok || native.Status != domain.StatusMissing {
		t.Fatalf("native game = %+v, ok=%v, want missing", native, ok)
	}
}

func TestOpenPathEscapeManifest(t *testing.T) {
	fixtures := materializeFixtures(t)
	homeWithSteam(t, filepath.Join(fixtures, "primary"))
	manifest := `"AppState" { "appid" "900099" "name" "Escape" "installdir" "../evil" "StateFlags" "4" "SizeOnDisk" "1" }`
	path := filepath.Join(fixtures, "primary", "steamapps", "appmanifest_900099.acf")
	if err := os.WriteFile(path, []byte(manifest), 0o600); err != nil {
		t.Fatalf("write escape manifest: %v", err)
	}

	catalog, err := Open(context.Background())
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, ok := findGame(catalog.Games, 900099); ok {
		t.Fatal("path-escape game was included")
	}
	if len(catalog.Warnings) != 1 || !errors.Is(catalog.Warnings[0].Err, domain.ErrManifestCorrupt) {
		t.Fatalf("warnings = %v, want one manifest-corrupt warning", catalog.Warnings)
	}
}

func TestLoadUnsupportedVariant(t *testing.T) {
	_, err := Load(context.Background(), Installation{Root: "/tmp/flatpak", Variant: VariantFlatpak})
	if !errors.Is(err, domain.ErrSteamUnsupported) {
		t.Fatalf("error = %v, want ErrSteamUnsupported", err)
	}
}

func TestLoadMissingSteamApps(t *testing.T) {
	root := t.TempDir()
	_, err := Load(context.Background(), Installation{Root: root, Variant: VariantNative})
	if !errors.Is(err, domain.ErrLibraryUnreadable) {
		t.Fatalf("error = %v, want ErrLibraryUnreadable", err)
	}
}

func TestOpenCanceled(t *testing.T) {
	fixtures := materializeFixtures(t)
	homeWithSteam(t, filepath.Join(fixtures, "primary"))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := Open(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}

func TestFilterGames(t *testing.T) {
	kindNative := domain.KindNative
	kindUnknown := domain.KindUnknown
	statusMissing := domain.StatusMissing
	statusUnknown := domain.StatusUnknown
	games := []domain.Game{
		{AppID: 480, Name: "Spacewar", LibraryPath: "/one", Kind: kindUnknown, Status: statusUnknown},
		{AppID: 900001, Name: "Native Game", LibraryPath: "/one", Kind: kindNative, Status: statusUnknown},
		{AppID: 900007, Name: "Secondary Native", LibraryPath: "/two", Kind: kindNative, Status: statusMissing},
	}
	catalog := Catalog{Games: games}

	if got := catalog.Search("native"); len(got) != 2 {
		t.Fatalf("search = %v, want 2", got)
	}
	if got := catalog.Search("900001"); len(got) != 1 || got[0].AppID != 900001 {
		t.Fatalf("app id search = %v", got)
	}
	if got := catalog.Filter(Filter{Kind: &kindNative}); len(got) != 2 {
		t.Fatalf("native filter = %v", got)
	}
	if got := catalog.Filter(Filter{Kind: &kindUnknown}); len(got) != 1 || got[0].AppID != 480 {
		t.Fatalf("unknown filter = %v", got)
	}
	if got := catalog.Filter(Filter{Status: &statusMissing}); len(got) != 1 || got[0].AppID != 900007 {
		t.Fatalf("missing filter = %v", got)
	}
	if got := catalog.Filter(Filter{Library: "/two"}); len(got) != 1 || got[0].AppID != 900007 {
		t.Fatalf("library filter = %v", got)
	}
	if got := FilterGames(nil, Filter{}); got == nil || len(got) != 0 {
		t.Fatalf("empty filter = %#v, want non-nil empty slice", got)
	}
}
