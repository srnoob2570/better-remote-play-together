package config

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/srnoob2570/better-rpt/internal/domain"
)

func copyTestFile(t *testing.T, source, destination string, mode os.FileMode) {
	t.Helper()
	data, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("read %s: %v", source, err)
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(destination), err)
	}
	if err := os.WriteFile(destination, data, mode); err != nil {
		t.Fatalf("write %s: %v", destination, err)
	}
}

func transactionRoot(t *testing.T, localConfigName string) string {
	t.Helper()
	root := t.TempDir()
	fixture := filepath.Join("..", "..", "testdata", "steam", "fixtures", "primary")
	copyTestFile(t, filepath.Join(fixture, "loginusers.vdf"), filepath.Join(root, "loginusers.vdf"), 0o644)
	copyTestFile(t, filepath.Join(fixture, "registry.vdf"), filepath.Join(root, "registry.vdf"), 0o644)
	copyTestFile(t,
		filepath.Join(fixture, "userdata", "11111111", "config", localConfigName),
		filepath.Join(root, "userdata", "11111111", "config", "localconfig.vdf"),
		0o775,
	)
	return root
}

func transactionOptions(t *testing.T, root string) PrepareOptions {
	t.Helper()
	paths := testPaths(t)
	return PrepareOptions{
		Paths:        paths,
		SteamRoot:    root,
		HelperPath:   filepath.Join(paths.DataDir, "bin", "better-rpt-helper"),
		TargetAppID:  900001,
		SessionToken: sessionToken,
		SteamRunning: func() (bool, error) { return false, nil },
	}
}

func TestPrepareAndRestoreExistingOption(t *testing.T) {
	root := transactionRoot(t, "localconfig.vdf")
	original, err := os.ReadFile(filepath.Join(root, "userdata", "11111111", "config", "localconfig.vdf"))
	if err != nil {
		t.Fatalf("read original: %v", err)
	}
	options := transactionOptions(t, root)
	prepared, err := Prepare(options)
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if prepared.Account.AccountID != 11111111 {
		t.Fatalf("account = %+v", prepared.Account)
	}
	if prepared.CreatedAppBlock || prepared.CreatedLaunchOptions {
		t.Fatalf("created flags = %+v", prepared)
	}
	if prepared.OriginalRaw != `""` || prepared.OriginalValue != "" {
		t.Fatalf("original = %q/%q", prepared.OriginalRaw, prepared.OriginalValue)
	}
	decoded, err := base64.StdEncoding.DecodeString(prepared.Backup.Meta.OriginalRawTokenBase64)
	if err != nil || string(decoded) != `""` {
		t.Fatalf("backup raw = %q, err=%v", decoded, err)
	}
	if prepared.PruneError != nil {
		t.Fatalf("prune: %v", prepared.PruneError)
	}

	_, option, err := ReadCurrentLaunchOption(root)
	if err != nil || !option.Exists || option.Value != prepared.NewValue {
		t.Fatalf("current option = %+v, err=%v", option, err)
	}
	current, err := os.ReadFile(prepared.LocalConfig)
	if err != nil {
		t.Fatalf("read prepared: %v", err)
	}
	if !bytes.Contains(current, []byte("cd /opt/other && ./launcher # %command%")) {
		t.Fatal("unrelated 900001 launch option changed")
	}
	if !bytes.Contains(current, []byte("webstorage-decoy")) {
		t.Fatal("decoy changed")
	}

	stale, err := DetectStale(options.Paths, root, false)
	if err != nil || stale == nil || stale.Backup.Path != prepared.Backup.Path {
		t.Fatalf("stale = %+v, err=%v", stale, err)
	}
	if stale.CurrentValue != prepared.NewValue {
		t.Fatalf("stale value = %q", stale.CurrentValue)
	}

	restored, err := Restore(RestoreOptions{
		Paths:        options.Paths,
		SteamRoot:    root,
		BackupPath:   prepared.Backup.Path,
		SteamRunning: options.SteamRunning,
	})
	if err != nil {
		t.Fatalf("restore: %v", err)
	}
	if restored.Removed || restored.RestoredValue != "" {
		t.Fatalf("restored = %+v", restored)
	}
	after, err := os.ReadFile(prepared.LocalConfig)
	if err != nil {
		t.Fatalf("read restored: %v", err)
	}
	if !bytes.Equal(after, original) {
		t.Fatal("restore did not reproduce the original localconfig bytes")
	}
	stale, err = DetectStale(options.Paths, root, false)
	if err != nil || stale != nil {
		t.Fatalf("stale after restore = %+v, err=%v", stale, err)
	}
}

func TestPrepareRefusesSteamRunning(t *testing.T) {
	root := transactionRoot(t, "localconfig.vdf")
	path := filepath.Join(root, "userdata", "11111111", "config", "localconfig.vdf")
	before, _ := os.ReadFile(path)
	options := transactionOptions(t, root)
	options.SteamRunning = func() (bool, error) { return true, nil }
	if _, err := Prepare(options); !errors.Is(err, domain.ErrSteamRunning) {
		t.Fatalf("error = %v, want ErrSteamRunning", err)
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatal("localconfig changed while Steam was running")
	}
}

func TestPrepareRequiresGuard(t *testing.T) {
	root := transactionRoot(t, "localconfig.vdf")
	options := transactionOptions(t, root)
	options.SteamRunning = nil
	if _, err := Prepare(options); !errors.Is(err, ErrSteamGuardMissing) {
		t.Fatalf("error = %v, want ErrSteamGuardMissing", err)
	}
}

func TestPrepareAlreadyPrepared(t *testing.T) {
	root := transactionRoot(t, "localconfig.vdf")
	options := transactionOptions(t, root)
	if _, err := Prepare(options); err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if _, err := Prepare(options); !errors.Is(err, ErrAlreadyPrepared) {
		t.Fatalf("error = %v, want ErrAlreadyPrepared", err)
	}
}

func TestPrepareSupersedesStaleMarkerAndRestoresOriginal(t *testing.T) {
	root := transactionRoot(t, "localconfig.vdf")
	localConfig := filepath.Join(root, "userdata", "11111111", "config", "localconfig.vdf")
	original, err := os.ReadFile(localConfig)
	if err != nil {
		t.Fatalf("read original: %v", err)
	}
	options := transactionOptions(t, root)
	first, err := Prepare(options)
	if err != nil {
		t.Fatalf("first prepare: %v", err)
	}
	options.SessionToken = "fedcba9876543210fedcba9876543210"
	second, err := Prepare(options)
	if err != nil {
		t.Fatalf("second prepare: %v", err)
	}
	if second.Backup.Path == first.Backup.Path {
		t.Fatal("second prepare reused the stale backup")
	}
	if second.OriginalRaw != first.OriginalRaw || second.OriginalValue != first.OriginalValue {
		t.Fatalf("second original = %q/%q, want %q/%q",
			second.OriginalRaw, second.OriginalValue, first.OriginalRaw, first.OriginalValue)
	}
	if second.CreatedAppBlock != first.CreatedAppBlock || second.CreatedLaunchOptions != first.CreatedLaunchOptions {
		t.Fatalf("second created flags = %+v, want %+v", second, first)
	}
	if second.Backup.Meta.OriginalRawTokenBase64 != first.Backup.Meta.OriginalRawTokenBase64 {
		t.Fatalf("second backup original raw = %q, want %q",
			second.Backup.Meta.OriginalRawTokenBase64, first.Backup.Meta.OriginalRawTokenBase64)
	}
	if _, err := Restore(RestoreOptions{
		Paths:        options.Paths,
		SteamRoot:    root,
		BackupPath:   second.Backup.Path,
		SteamRunning: options.SteamRunning,
	}); err != nil {
		t.Fatalf("restore second: %v", err)
	}
	after, err := os.ReadFile(localConfig)
	if err != nil {
		t.Fatalf("read restored: %v", err)
	}
	if !bytes.Equal(after, original) {
		t.Fatal("restoring the superseding session did not reproduce the original bytes")
	}
}

func TestPrepareRefusesOrphanMarkerWithoutBackup(t *testing.T) {
	root := transactionRoot(t, "localconfig.vdf")
	options := transactionOptions(t, root)
	prepared, err := Prepare(options)
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if err := os.Remove(prepared.Backup.Path); err != nil {
		t.Fatalf("remove backup: %v", err)
	}
	if err := os.Remove(prepared.Backup.MetaPath); err != nil {
		t.Fatalf("remove metadata: %v", err)
	}
	options.SessionToken = "fedcba9876543210fedcba9876543210"
	if _, err := Prepare(options); !errors.Is(err, ErrBackupMissing) {
		t.Fatalf("error = %v, want ErrBackupMissing", err)
	}
	_, option, err := ReadCurrentLaunchOption(root)
	if err != nil {
		t.Fatalf("read current: %v", err)
	}
	if option.Value != prepared.NewValue {
		t.Fatal("orphan marker was overwritten")
	}
}

func TestPrepareMissingAppAndRestore(t *testing.T) {
	root := transactionRoot(t, "localconfig_missing.vdf")
	options := transactionOptions(t, root)
	prepared, err := Prepare(options)
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if !prepared.CreatedAppBlock || !prepared.CreatedLaunchOptions {
		t.Fatalf("created flags = %+v", prepared)
	}
	if _, err := Restore(RestoreOptions{
		Paths:        options.Paths,
		SteamRoot:    root,
		BackupPath:   prepared.Backup.Path,
		SteamRunning: options.SteamRunning,
	}); err != nil {
		t.Fatalf("restore: %v", err)
	}
	_, option, err := ReadCurrentLaunchOption(root)
	if err != nil {
		t.Fatalf("read current: %v", err)
	}
	if option.Exists {
		t.Fatalf("created app block was not removed: %+v", option)
	}
}

func TestPrepareMissingLaunchOptionsAndRestore(t *testing.T) {
	root := transactionRoot(t, "localconfig_no_launchoptions.vdf")
	options := transactionOptions(t, root)
	prepared, err := Prepare(options)
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if prepared.CreatedAppBlock || !prepared.CreatedLaunchOptions {
		t.Fatalf("created flags = %+v", prepared)
	}
	if _, err := Restore(RestoreOptions{
		Paths:        options.Paths,
		SteamRoot:    root,
		BackupPath:   prepared.Backup.Path,
		SteamRunning: options.SteamRunning,
	}); err != nil {
		t.Fatalf("restore: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(root, "userdata", "11111111", "config", "localconfig.vdf"))
	if err != nil {
		t.Fatalf("read restored: %v", err)
	}
	if !bytes.Contains(data, []byte(`"cloud"`)) {
		t.Fatal("unrelated data in app 480 disappeared")
	}
}

func TestDetectStaleWithoutBackup(t *testing.T) {
	root := transactionRoot(t, "localconfig.vdf")
	options := transactionOptions(t, root)
	prepared, err := Prepare(options)
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if err := os.Remove(prepared.Backup.Path); err != nil {
		t.Fatalf("remove backup: %v", err)
	}
	if err := os.Remove(prepared.Backup.MetaPath); err != nil {
		t.Fatalf("remove metadata: %v", err)
	}
	stale, err := DetectStale(options.Paths, root, false)
	if err != nil {
		t.Fatalf("detect stale: %v", err)
	}
	if stale == nil || stale.Backup.Path != "" || !IsSessionMarker(stale.CurrentValue) {
		t.Fatalf("stale = %+v, want marker without backup", stale)
	}
}

func TestRestoreRefusesChangedMarker(t *testing.T) {
	root := transactionRoot(t, "localconfig.vdf")
	options := transactionOptions(t, root)
	prepared, err := Prepare(options)
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	data, err := os.ReadFile(prepared.LocalConfig)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	changed := bytes.Replace(data, []byte(prepared.NewValue), []byte("someone-else"), 1)
	if bytes.Equal(changed, data) {
		t.Fatal("marker not found")
	}
	if err := writeLocalConfig(prepared.LocalConfig, changed, 0o775); err != nil {
		t.Fatalf("write changed: %v", err)
	}
	if _, err := Restore(RestoreOptions{
		Paths:        options.Paths,
		SteamRoot:    root,
		BackupPath:   prepared.Backup.Path,
		SteamRunning: options.SteamRunning,
	}); !errors.Is(err, ErrLocalConfigChanged) {
		t.Fatalf("error = %v, want ErrLocalConfigChanged", err)
	}
	after, _ := os.ReadFile(prepared.LocalConfig)
	if !bytes.Equal(after, changed) {
		t.Fatal("restore overwrote a changed launch option")
	}
}

func TestRestoreActiveSession(t *testing.T) {
	root := transactionRoot(t, "localconfig.vdf")
	options := transactionOptions(t, root)
	prepared, err := Prepare(options)
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if _, err := Restore(RestoreOptions{
		Paths:         options.Paths,
		SteamRoot:     root,
		BackupPath:    prepared.Backup.Path,
		ActiveSession: true,
		SteamRunning:  options.SteamRunning,
	}); !errors.Is(err, ErrSessionActive) {
		t.Fatalf("error = %v, want ErrSessionActive", err)
	}
}

func TestRestoreRequiresGuard(t *testing.T) {
	root := transactionRoot(t, "localconfig.vdf")
	options := transactionOptions(t, root)
	prepared, err := Prepare(options)
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if _, err := Restore(RestoreOptions{
		Paths:      options.Paths,
		SteamRoot:  root,
		BackupPath: prepared.Backup.Path,
	}); !errors.Is(err, ErrSteamGuardMissing) {
		t.Fatalf("error = %v, want ErrSteamGuardMissing", err)
	}
}

func TestPrepareMalformedLeavesFileIntact(t *testing.T) {
	root := transactionRoot(t, "localconfig_malformed.vdf")
	path := filepath.Join(root, "userdata", "11111111", "config", "localconfig.vdf")
	before, _ := os.ReadFile(path)
	options := transactionOptions(t, root)
	if _, err := Prepare(options); err == nil {
		t.Fatal("malformed localconfig accepted")
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatal("malformed localconfig was modified")
	}
}

func readXattr(t *testing.T, path, name string) []byte {
	t.Helper()
	size, err := syscall.Getxattr(path, name, nil)
	if err != nil {
		t.Fatalf("read xattr %s: %v", name, err)
	}
	value := make([]byte, size)
	n, err := syscall.Getxattr(path, name, value)
	if err != nil {
		t.Fatalf("read xattr %s: %v", name, err)
	}
	return value[:n]
}

// posixACLBlob builds a system.posix_acl_access value with one named-user
// entry, which is what makes the kernel store the ACL as an xattr.
func posixACLBlob(owner, named, group, mask, other uint16) []byte {
	const (
		aclUserObj  = 0x01
		aclUser     = 0x02
		aclGroupObj = 0x04
		aclMask     = 0x10
		aclOther    = 0x20
		undefinedID = 0xffffffff
	)
	entries := []struct {
		tag  uint16
		perm uint16
		id   uint32
	}{
		{aclUserObj, owner, undefinedID},
		{aclUser, named, uint32(os.Geteuid())},
		{aclGroupObj, group, undefinedID},
		{aclMask, mask, undefinedID},
		{aclOther, other, undefinedID},
	}
	blob := make([]byte, 4, 4+len(entries)*8)
	binary.LittleEndian.PutUint32(blob, 2)
	for _, entry := range entries {
		raw := make([]byte, 8)
		binary.LittleEndian.PutUint16(raw[0:2], entry.tag)
		binary.LittleEndian.PutUint16(raw[2:4], entry.perm)
		binary.LittleEndian.PutUint32(raw[4:8], entry.id)
		blob = append(blob, raw...)
	}
	return blob
}

func TestPreparePreservesExtendedAttributes(t *testing.T) {
	root := transactionRoot(t, "localconfig.vdf")
	path := filepath.Join(root, "userdata", "11111111", "config", "localconfig.vdf")
	if err := syscall.Setxattr(path, "user.better-rpt-test", []byte("kept"), 0); err != nil {
		t.Skipf("xattrs unsupported: %v", err)
	}
	defer func() { _ = syscall.Removexattr(path, "user.better-rpt-test") }()

	if _, err := Prepare(transactionOptions(t, root)); err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if got := readXattr(t, path, "user.better-rpt-test"); string(got) != "kept" {
		t.Fatalf("xattr after prepare = %q, want %q", got, "kept")
	}
}

func TestPreparePreservesPOSIXACL(t *testing.T) {
	root := transactionRoot(t, "localconfig.vdf")
	path := filepath.Join(root, "userdata", "11111111", "config", "localconfig.vdf")
	acl := posixACLBlob(0o6, 0o4, 0o4, 0o4, 0o0)
	if err := syscall.Setxattr(path, "system.posix_acl_access", acl, 0); err != nil {
		t.Skipf("POSIX ACLs unsupported: %v", err)
	}
	defer func() { _ = syscall.Removexattr(path, "system.posix_acl_access") }()

	if _, err := Prepare(transactionOptions(t, root)); err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if got := readXattr(t, path, "system.posix_acl_access"); !bytes.Equal(got, acl) {
		t.Fatalf("ACL after prepare = %x, want %x", got, acl)
	}
}
