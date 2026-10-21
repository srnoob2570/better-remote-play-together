package appinfo

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

const fixtureDir = "../../../testdata/steam/fixtures/primary/appcache"

func openFixture(t *testing.T, name string) *File {
	t.Helper()
	file, err := Open(filepath.Join(fixtureDir, name))
	if err != nil {
		t.Fatalf("open %s: %v", name, err)
	}
	return file
}

func TestOpenValidFixture(t *testing.T) {
	file := openFixture(t, "appinfo.vdf")
	if got := len(file.Warnings()); got != 0 {
		t.Fatalf("warnings = %d, want 0: %v", got, file.Warnings())
	}
	apps := file.Apps()
	if len(apps) != 12 {
		t.Fatalf("apps = %d, want 12", len(apps))
	}

	native, ok := file.App(900001)
	if !ok {
		t.Fatal("app 900001 missing")
	}
	if native.Common.Name != "Native Game" || native.Common.Type != "Game" || native.Common.InstallDir != "Native Game" {
		t.Fatalf("native common = %+v", native.Common)
	}
	if len(native.Launch) != 2 {
		t.Fatalf("native launches = %d, want 2", len(native.Launch))
	}
	if native.Launch[0].OSList != "linux" || native.Launch[0].Executable != "game.x86_64" || native.Launch[0].Arch != "64" {
		t.Fatalf("native linux launch = %+v", native.Launch[0])
	}
	if native.Launch[1].OSList != "windows" || native.Launch[1].Executable != "game.exe" {
		t.Fatalf("native windows launch = %+v", native.Launch[1])
	}

	proton, ok := file.App(900002)
	if !ok || len(proton.Launch) != 1 || proton.Launch[0].OSList != "windows" {
		t.Fatalf("proton app = %+v, ok=%v", proton, ok)
	}
	tool, ok := file.App(900003)
	if !ok || tool.Common.Type != "Tool" {
		t.Fatalf("tool app = %+v, ok=%v", tool, ok)
	}
	noLaunch, ok := file.App(900009)
	if !ok || len(noLaunch.Launch) != 0 {
		t.Fatalf("no-launch app = %+v, ok=%v", noLaunch, ok)
	}
	if _, ok := file.App(900010); ok {
		t.Fatal("900010 unexpectedly present")
	}
}

func TestOpenUnsupportedVersion(t *testing.T) {
	_, err := Open(filepath.Join(fixtureDir, "appinfo_v40.vdf"))
	if !errors.Is(err, ErrUnsupportedVersion) {
		t.Fatalf("error = %v, want ErrUnsupportedVersion", err)
	}
}

func TestOpenCorrupt(t *testing.T) {
	_, err := Open(filepath.Join(fixtureDir, "appinfo_corrupt.vdf"))
	if !errors.Is(err, ErrCorrupt) {
		t.Fatalf("error = %v, want ErrCorrupt", err)
	}
}

func TestOpenBadKeyOnlySkipsThatApp(t *testing.T) {
	file := openFixture(t, "appinfo_bad_key.vdf")
	if got := len(file.Warnings()); got != 0 {
		t.Fatalf("warnings before lookup = %d, want 0", got)
	}
	if _, ok := file.App(900013); ok {
		t.Fatal("corrupt app was returned")
	}
	warnings := file.Warnings()
	if len(warnings) != 1 {
		t.Fatalf("warnings = %d, want 1: %v", len(warnings), warnings)
	}
	if warnings[0].AppID != 900013 || !errors.Is(warnings[0].Err, ErrAppCorrupt) {
		t.Fatalf("warning = %+v, want corrupt 900013", warnings[0])
	}
	if _, ok := file.App(900001); !ok {
		t.Fatal("valid app disappeared after a bad-key record")
	}
}

func TestOpenMissingFile(t *testing.T) {
	_, err := Open(filepath.Join(t.TempDir(), "missing.vdf"))
	if !errors.Is(err, ErrNotAppInfo) {
		t.Fatalf("error = %v, want ErrNotAppInfo", err)
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("error = %v, want os.ErrNotExist too", err)
	}
}

func TestParseTruncated(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(fixtureDir, "appinfo.vdf"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	for _, size := range []int{0, 8, 16, len(data) / 2, len(data) - 1} {
		if _, err := Parse(bytes.NewReader(data[:size]), int64(size)); err == nil {
			t.Fatalf("truncated %d bytes accepted", size)
		}
	}
}

func TestParseRejectsBadMagic(t *testing.T) {
	data := make([]byte, 32)
	if _, err := Parse(bytes.NewReader(data), int64(len(data))); !errors.Is(err, ErrNotAppInfo) {
		t.Fatalf("error = %v, want ErrNotAppInfo", err)
	}
}

func TestParseChecksBinaryHash(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(fixtureDir, "appinfo.vdf"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	// The first entry starts at 16; its data starts at 24 and its binary SHA-1
	// is at data+40. Flip one bit without changing the entry size.
	data[24+40] ^= 0xff
	file, err := Parse(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	apps := file.Apps()
	if len(apps) != 11 {
		t.Fatalf("apps = %d, want 11 after skipping one bad hash", len(apps))
	}
	if warnings := file.Warnings(); len(warnings) != 1 || !errors.Is(warnings[0].Err, ErrAppCorrupt) {
		t.Fatalf("warnings = %v, want one corrupt app", warnings)
	}
}

func BenchmarkOpenFixture(b *testing.B) {
	path := filepath.Join(fixtureDir, "appinfo.vdf")
	for i := 0; i < b.N; i++ {
		if _, err := Open(path); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkOpenRealAppInfo(b *testing.B) {
	path, ok := realAppInfoPath(b)
	if !ok {
		return
	}
	for i := 0; i < b.N; i++ {
		if _, err := Open(path); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkOpenAndLookupRealAppInfo(b *testing.B) {
	path, ok := realAppInfoPath(b)
	if !ok {
		return
	}
	benchmarkOpenAndLookup(b, path, 50)
}

func BenchmarkOpenAndLookupFixture(b *testing.B) {
	benchmarkOpenAndLookup(b, filepath.Join(fixtureDir, "appinfo.vdf"), 0)
}

func benchmarkOpenAndLookup(b *testing.B, path string, limit int) {
	b.Helper()
	indexed, err := Open(path)
	if err != nil {
		b.Fatal(err)
	}
	apps := indexed.Apps()
	if len(apps) == 0 {
		b.Skip("no usable app records")
	}
	if limit <= 0 || limit > len(apps) {
		limit = len(apps)
	}
	ids := make([]uint32, limit)
	for i := range ids {
		ids[i] = apps[i].ID
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		file, err := Open(path)
		if err != nil {
			b.Fatal(err)
		}
		for _, id := range ids {
			if _, ok := file.App(id); !ok {
				b.Fatalf("app %d missing after lookup", id)
			}
		}
	}
}

func realAppInfoPath(b *testing.B) (string, bool) {
	b.Helper()
	home, err := os.UserHomeDir()
	if err != nil {
		b.Skip("no home directory")
	}
	path := filepath.Join(home, ".local", "share", "Steam", "appcache", "appinfo.vdf")
	if _, err := os.Stat(path); err != nil {
		b.Skip("real appinfo.vdf unavailable")
	}
	return path, true
}
