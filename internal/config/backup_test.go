package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func testBackupMeta(token string, created time.Time) BackupMeta {
	return BackupMeta{
		Source:        "/steam/userdata/11111111/config/localconfig.vdf",
		SteamID64:     "76561197971376839",
		AccountID:     11111111,
		AccountName:   "fixture",
		AppID:         480,
		TargetAppID:   900001,
		SessionToken:  token,
		OriginalMode:  0o775,
		OriginalValue: "",
		NewValue:      "marker",
	}
}

func TestBackupRoundTrip(t *testing.T) {
	paths := testPaths(t)
	data := []byte("original localconfig\n")
	meta := testBackupMeta(sessionToken, time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC))
	backup, err := CreateBackup(paths, meta.Source, data, meta)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	for _, path := range []string{backup.Path, backup.MetaPath} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat %s: %v", path, err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("%s mode = %o, want 600", path, info.Mode().Perm())
		}
	}
	dirInfo, err := os.Stat(paths.BackupsDir())
	if err != nil {
		t.Fatalf("stat backup dir: %v", err)
	}
	if dirInfo.Mode().Perm() != 0o700 {
		t.Fatalf("backup dir mode = %o, want 700", dirInfo.Mode().Perm())
	}

	loaded, err := LoadBackup(backup.Path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.Meta.SessionToken != sessionToken || loaded.Meta.OriginalSHA256 == "" {
		t.Fatalf("loaded meta = %+v", loaded.Meta)
	}
	backups, err := ListBackups(paths)
	if err != nil || len(backups) != 1 {
		t.Fatalf("list = %v, %v", backups, err)
	}
	found, err := FindBackupBySession(paths, sessionToken)
	if err != nil || found.Path != backup.Path {
		t.Fatalf("find = %+v, %v", found, err)
	}
}

func TestBackupCorruptHash(t *testing.T) {
	paths := testPaths(t)
	backup, err := CreateBackup(paths, "source", []byte("original\n"), testBackupMeta(sessionToken, time.Now()))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := os.WriteFile(backup.Path, []byte("changed\n"), 0o600); err != nil {
		t.Fatalf("corrupt backup: %v", err)
	}
	if _, err := LoadBackup(backup.Path); !errors.Is(err, ErrBackupCorrupt) {
		t.Fatalf("error = %v, want ErrBackupCorrupt", err)
	}
	backups, err := ListBackups(paths)
	if err != nil || len(backups) != 1 {
		t.Fatalf("metadata listing = %v, %v; want one backup despite corrupt payload", backups, err)
	}
}

func TestListBackupsWithoutPayload(t *testing.T) {
	paths := testPaths(t)
	backup, err := CreateBackup(paths, "source", []byte("original\n"), testBackupMeta(sessionToken, time.Now()))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := os.Remove(backup.Path); err != nil {
		t.Fatalf("remove payload: %v", err)
	}
	backups, err := ListBackups(paths)
	if err != nil || len(backups) != 1 {
		t.Fatalf("metadata listing = %v, %v; want one backup without payload", backups, err)
	}
	if _, err := LoadBackup(backup.Path); !errors.Is(err, ErrBackupMissing) {
		t.Fatalf("load error = %v, want ErrBackupMissing", err)
	}
}

func TestBackupCorruptMetadata(t *testing.T) {
	paths := testPaths(t)
	backup, err := CreateBackup(paths, "source", []byte("original\n"), testBackupMeta(sessionToken, time.Now()))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := os.WriteFile(backup.MetaPath, []byte("{"), 0o600); err != nil {
		t.Fatalf("corrupt metadata: %v", err)
	}
	if _, err := LoadBackup(backup.Path); !errors.Is(err, ErrBackupCorrupt) {
		t.Fatalf("error = %v, want ErrBackupCorrupt", err)
	}
	if _, err := ListBackups(paths); !errors.Is(err, ErrBackupCorrupt) {
		t.Fatalf("list error = %v, want ErrBackupCorrupt", err)
	}
}

func TestRestoreBySessionVerifiesPayload(t *testing.T) {
	paths := testPaths(t)
	backup, err := CreateBackup(paths, "source", []byte("original\n"), testBackupMeta(sessionToken, time.Now()))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := os.WriteFile(backup.Path, []byte("changed\n"), 0o600); err != nil {
		t.Fatalf("corrupt backup: %v", err)
	}
	_, err = Restore(RestoreOptions{
		Paths:        paths,
		SteamRoot:    "/steam",
		SessionToken: sessionToken,
		SteamRunning: func() (bool, error) { return false, nil },
	})
	if !errors.Is(err, ErrBackupCorrupt) {
		t.Fatalf("restore error = %v, want ErrBackupCorrupt", err)
	}
}

func TestPruneKeepsNewest(t *testing.T) {
	paths := testPaths(t)
	var newest string
	for i := 0; i < 12; i++ {
		token := fmt.Sprintf("%032x", i)
		meta := testBackupMeta(token, time.Date(2026, 10, 18, 12, i, 0, 0, time.UTC))
		backup, err := CreateBackup(paths, meta.Source, []byte("original\n"), meta)
		if err != nil {
			t.Fatalf("create %d: %v", i, err)
		}
		newest = backup.Path
	}
	if err := PruneBackups(paths, 10, newest); err != nil {
		t.Fatalf("prune: %v", err)
	}
	backups, err := ListBackups(paths)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(backups) != 10 {
		t.Fatalf("backups = %d, want 10", len(backups))
	}
	found := false
	for _, backup := range backups {
		if backup.Path == newest {
			found = true
		}
	}
	if !found {
		t.Fatal("protected newest backup was removed")
	}
}

func TestListBackupsIgnoresUnrelated(t *testing.T) {
	paths := testPaths(t)
	if err := os.MkdirAll(paths.BackupsDir(), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(paths.BackupsDir(), "notes.txt"), []byte("x"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	backups, err := ListBackups(paths)
	if err != nil || len(backups) != 0 {
		t.Fatalf("backups = %v, err=%v", backups, err)
	}
}

func BenchmarkListBackupsMetadataOnly(b *testing.B) {
	paths := benchBackups(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		backups, err := ListBackups(paths)
		if err != nil || len(backups) != 10 {
			b.Fatalf("backups = %d, err=%v", len(backups), err)
		}
	}
}

func BenchmarkListBackupsVerifyPayload(b *testing.B) {
	paths := benchBackups(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		backups, err := ListBackups(paths)
		if err != nil || len(backups) != 10 {
			b.Fatalf("backups = %d, err=%v", len(backups), err)
		}
		for _, backup := range backups {
			if _, err := LoadBackup(backup.Path); err != nil {
				b.Fatal(err)
			}
		}
	}
}

func benchBackups(b *testing.B) Paths {
	b.Helper()
	paths := Paths{DataDir: b.TempDir()}
	data := bytes.Repeat([]byte("x"), 1<<20)
	for i := 0; i < 10; i++ {
		meta := testBackupMeta(fmt.Sprintf("%032x", i), time.Date(2026, 10, 18, 12, i, 0, 0, time.UTC))
		if _, err := CreateBackup(paths, meta.Source, data, meta); err != nil {
			b.Fatalf("create %d: %v", i, err)
		}
	}
	return paths
}
