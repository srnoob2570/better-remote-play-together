package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReadLocalStateSharesLaunchAndLastPlayed(t *testing.T) {
	root := transactionRoot(t, "localconfig_lastplayed.vdf")
	state, err := ReadLocalState(root)
	if err != nil {
		t.Fatalf("read local state: %v", err)
	}
	if state.Account.AccountID != 11111111 || state.Path == "" {
		t.Fatalf("state account = %+v, path = %q", state.Account, state.Path)
	}
	if state.LastPlayed == nil || state.LastPlayed.AppID != 900002 {
		t.Fatalf("last played = %+v, want 900002", state.LastPlayed)
	}
	want := time.Unix(1700000200, 0).UTC()
	if !state.LastPlayed.At.Equal(want) {
		t.Fatalf("last played at = %s, want %s", state.LastPlayed.At, want)
	}
	if _, err := state.InspectOverlay(); err != nil {
		t.Fatalf("inspect overlay from state: %v", err)
	}
	if _, err := DetectSetupFromState(testPaths(t), state); err != nil {
		t.Fatalf("detect setup from state: %v", err)
	}
}

func BenchmarkLocalStateShared(b *testing.B) {
	root := benchmarkTransactionRoot(b)
	paths := Paths{DataDir: b.TempDir()}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		state, err := ReadLocalState(root)
		if err != nil {
			b.Fatal(err)
		}
		if _, err := DetectSetupFromState(paths, state); err != nil {
			b.Fatal(err)
		}
		if state.LastPlayed == nil {
			b.Fatal("last played missing")
		}
	}
}

func BenchmarkLocalStateSeparate(b *testing.B) {
	root := benchmarkTransactionRoot(b)
	paths := Paths{DataDir: b.TempDir()}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := ReadLastPlayed(root); err != nil {
			b.Fatal(err)
		}
		if _, err := DetectSetup(paths, root); err != nil {
			b.Fatal(err)
		}
		if _, err := InspectOverlay(root); err != nil {
			b.Fatal(err)
		}
	}
}

func benchmarkTransactionRoot(b *testing.B) string {
	b.Helper()
	root := b.TempDir()
	fixture := filepath.Join("..", "..", "testdata", "steam", "fixtures", "primary")
	copyBenchmarkFile(b, filepath.Join(fixture, "loginusers.vdf"), filepath.Join(root, "loginusers.vdf"), 0o644)
	copyBenchmarkFile(b, filepath.Join(fixture, "registry.vdf"), filepath.Join(root, "registry.vdf"), 0o644)
	copyBenchmarkFile(b,
		filepath.Join(fixture, "userdata", "11111111", "config", "localconfig_lastplayed.vdf"),
		filepath.Join(root, "userdata", "11111111", "config", "localconfig.vdf"),
		0o775,
	)
	return root
}

func copyBenchmarkFile(b *testing.B, source, destination string, mode os.FileMode) {
	b.Helper()
	data, err := os.ReadFile(source)
	if err != nil {
		b.Fatalf("read %s: %v", source, err)
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		b.Fatalf("mkdir %s: %v", filepath.Dir(destination), err)
	}
	if err := os.WriteFile(destination, data, mode); err != nil {
		b.Fatalf("write %s: %v", destination, err)
	}
}
