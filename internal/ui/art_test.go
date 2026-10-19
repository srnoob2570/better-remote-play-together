package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/srnoob2570/better-rpt/internal/domain"
)

func TestSteamArtPaths(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "appcache", "librarycache", "123")
	if err := os.MkdirAll(filepath.Join(dir, "hero"), 0o755); err != nil {
		t.Fatal(err)
	}
	hero := filepath.Join(dir, "hero", "library_hero.jpg")
	icon := filepath.Join(dir, strings.Repeat("a", 40)+".jpg")
	for _, path := range []string{hero, icon} {
		if err := os.WriteFile(path, []byte("test"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	game := domain.Game{AppID: 123, LibraryPath: root}
	if got := steamCoverPath(game); got != hero {
		t.Fatalf("cover = %q, want %q", got, hero)
	}
	if got := steamIconPath(game); got != icon {
		t.Fatalf("icon = %q, want %q", got, icon)
	}

	missing := domain.Game{AppID: 999, LibraryPath: root}
	if got := steamCoverPath(missing); got != "" {
		t.Fatalf("missing cover = %q, want empty", got)
	}
	if got := steamIconPath(missing); got != "" {
		t.Fatalf("missing icon = %q, want empty", got)
	}
}
