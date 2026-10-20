package art

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/srnoob2570/better-rpt/internal/domain"
)

func TestParseKind(t *testing.T) {
	for _, value := range []string{"cover", "icon"} {
		if _, ok := ParseKind(value); !ok {
			t.Fatalf("ParseKind(%q) rejected", value)
		}
	}
	if _, ok := ParseKind("poster"); ok {
		t.Fatal("ParseKind accepted an unknown kind")
	}
}

func TestCoverPreference(t *testing.T) {
	root := t.TempDir()
	base := cacheDir(t, root, 123)
	writeFile(t, filepath.Join(base, "header.jpg"), []byte("header"))
	writeFile(t, filepath.Join(base, "library_hero.jpg"), []byte("hero"))

	game := domain.Game{AppID: 123, LibraryPath: root}
	path, err := CoverPath(game)
	if err != nil {
		t.Fatalf("CoverPath() error = %v", err)
	}
	if want := filepath.Join(base, "library_hero.jpg"); path != want {
		t.Fatalf("CoverPath() = %q, want %q", path, want)
	}

	if err := os.Remove(filepath.Join(base, "library_hero.jpg")); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(base, "hash", "library_header.jpg"), []byte("header in hash"))
	path, err = CoverPath(game)
	if err != nil {
		t.Fatalf("CoverPath() subdirectory error = %v", err)
	}
	if want := filepath.Join(base, "hash", "library_header.jpg"); path != want {
		t.Fatalf("CoverPath() subdirectory = %q, want %q", path, want)
	}
}

func TestRootOrder(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	libraryRoot := t.TempDir()

	writeFile(t, filepath.Join(cacheDir(t, home, 123), "library_hero.jpg"), []byte("home"))
	writeFile(t, filepath.Join(cacheDir(t, libraryRoot, 123), "library_hero.jpg"), []byte("library"))

	path, err := CoverPath(domain.Game{AppID: 123, LibraryPath: libraryRoot})
	if err != nil {
		t.Fatalf("CoverPath() error = %v", err)
	}
	if want := filepath.Join(cacheDir(t, libraryRoot, 123), "library_hero.jpg"); path != want {
		t.Fatalf("CoverPath() = %q, want %q", path, want)
	}
}

func TestIconPreference(t *testing.T) {
	root := t.TempDir()
	base := cacheDir(t, root, 123)
	hexName := strings.Repeat("a", 40) + ".jpg"
	writeFile(t, filepath.Join(base, hexName), []byte("hex"))
	writeFile(t, filepath.Join(base, "icon.jpg"), []byte("icon jpg"))
	writeFile(t, filepath.Join(base, "icon.png"), []byte("icon png"))

	game := domain.Game{AppID: 123, LibraryPath: root}
	path, err := IconPath(game)
	if err != nil {
		t.Fatalf("IconPath() error = %v", err)
	}
	if want := filepath.Join(base, hexName); path != want {
		t.Fatalf("IconPath() = %q, want %q", path, want)
	}

	if err := os.Remove(filepath.Join(base, hexName)); err != nil {
		t.Fatal(err)
	}
	path, err = IconPath(game)
	if err != nil {
		t.Fatalf("IconPath() fallback error = %v", err)
	}
	if want := filepath.Join(base, "icon.jpg"); path != want {
		t.Fatalf("IconPath() fallback = %q, want %q", path, want)
	}
}

func TestCoverSubdirectoryDepth(t *testing.T) {
	root := t.TempDir()
	base := cacheDir(t, root, 123)
	writeFile(t, filepath.Join(base, "one", "two", "library_hero.jpg"), []byte("too deep"))

	_, err := CoverPath(domain.Game{AppID: 123, LibraryPath: root})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("CoverPath() error = %v, want ErrNotFound", err)
	}
}

func TestMissingArtwork(t *testing.T) {
	_, err := CoverPath(domain.Game{AppID: 123, LibraryPath: t.TempDir()})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("CoverPath() error = %v, want ErrNotFound", err)
	}
	_, err = IconPath(domain.Game{})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("IconPath() empty game error = %v, want ErrNotFound", err)
	}
}

func TestSymlinkEscapeRejected(t *testing.T) {
	root := t.TempDir()
	base := cacheDir(t, root, 123)
	outside := filepath.Join(t.TempDir(), "outside.jpg")
	writeFile(t, outside, []byte("outside"))
	if err := os.Symlink(outside, filepath.Join(base, "library_hero.jpg")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	path, err := CoverPath(domain.Game{AppID: 123, LibraryPath: root})
	if err == nil {
		t.Fatalf("CoverPath() = %q, want rejection", path)
	}
	if !errors.Is(err, ErrUnsafe) {
		t.Fatalf("CoverPath() error = %v, want ErrUnsafe", err)
	}
}

func TestAppDirectorySymlinkRejected(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	writeFile(t, filepath.Join(outside, "library_hero.jpg"), []byte("outside"))
	if err := os.MkdirAll(filepath.Join(root, "appcache", "librarycache"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "appcache", "librarycache", "123")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	path, err := CoverPath(domain.Game{AppID: 123, LibraryPath: root})
	if err == nil {
		t.Fatalf("CoverPath() = %q, want rejection", path)
	}
}

func TestSymlinkInsideAllowed(t *testing.T) {
	root := t.TempDir()
	base := cacheDir(t, root, 123)
	real := filepath.Join(base, "actual-cover.jpg")
	writeFile(t, real, []byte("cover"))
	if err := os.Symlink(real, filepath.Join(base, "library_hero.jpg")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	path, err := CoverPath(domain.Game{AppID: 123, LibraryPath: root})
	if err != nil {
		t.Fatalf("CoverPath() error = %v", err)
	}
	if path != real {
		t.Fatalf("CoverPath() = %q, want canonical %q", path, real)
	}
}

func TestNonRegularRejected(t *testing.T) {
	root := t.TempDir()
	base := cacheDir(t, root, 123)
	if err := os.Mkdir(filepath.Join(base, "library_hero.jpg"), 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := CoverPath(domain.Game{AppID: 123, LibraryPath: root})
	if !errors.Is(err, ErrUnsafe) {
		t.Fatalf("CoverPath() error = %v, want ErrUnsafe", err)
	}
}

func TestOversizedRejected(t *testing.T) {
	root := t.TempDir()
	base := cacheDir(t, root, 123)
	path := filepath.Join(base, "library_hero.jpg")
	writeFile(t, path, []byte("cover"))
	if err := os.Truncate(path, MaxFileSize+1); err != nil {
		t.Fatal(err)
	}

	_, err := CoverPath(domain.Game{AppID: 123, LibraryPath: root})
	if !errors.Is(err, ErrTooLarge) {
		t.Fatalf("CoverPath() error = %v, want ErrTooLarge", err)
	}
}

func cacheDir(t *testing.T, root string, appID uint32) string {
	t.Helper()
	dir := filepath.Join(root, "appcache", "librarycache", strconvUint32(appID))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func writeFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func strconvUint32(value uint32) string {
	return strconv.FormatUint(uint64(value), 10)
}
