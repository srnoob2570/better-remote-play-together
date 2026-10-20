package config

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/srnoob2570/better-rpt/internal/domain"
)

func setupOptions(t *testing.T, root string) InstallSetupOptions {
	t.Helper()
	paths := testPaths(t)
	return InstallSetupOptions{
		Paths:        paths,
		SteamRoot:    root,
		HelperPath:   filepath.Join(paths.DataDir, "bin", "better-rpt-helper"),
		SteamRunning: func() (bool, error) { return false, nil },
	}
}

func readFixture(t *testing.T, root string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, "userdata", "11111111", "config", "localconfig.vdf"))
	if err != nil {
		t.Fatalf("read localconfig: %v", err)
	}
	return data
}

func TestInstallSetupAndRestore(t *testing.T) {
	root := transactionRoot(t, "localconfig.vdf")
	original := readFixture(t, root)
	options := setupOptions(t, root)

	setup, err := InstallSetup(options)
	if err != nil {
		t.Fatalf("install setup: %v", err)
	}
	if setup.Status != SetupInstalled {
		t.Fatalf("status = %q", setup.Status)
	}
	if setup.Backup.Meta.Kind != BackupDispatcher {
		t.Fatalf("backup kind = %q", setup.Backup.Meta.Kind)
	}
	want, err := BuildDispatcherLaunchOption(options.HelperPath)
	if err != nil {
		t.Fatalf("build dispatcher: %v", err)
	}
	if setup.NewValue != want {
		t.Fatalf("new value = %q, want %q", setup.NewValue, want)
	}
	current := readFixture(t, root)
	if !bytes.Contains(current, []byte(setup.NewValue)) {
		t.Fatalf("localconfig does not contain dispatcher %q", setup.NewValue)
	}
	if bytes.Equal(current, original) {
		t.Fatal("install did not write localconfig")
	}

	loaded, err := LoadSetup(options.Paths)
	if err != nil || loaded == nil || loaded.Status != SetupInstalled {
		t.Fatalf("loaded = %+v, err=%v", loaded, err)
	}
	snapshot, err := DetectSetup(options.Paths, root)
	if err != nil {
		t.Fatalf("detect setup: %v", err)
	}
	if snapshot.Status != SetupInstalled || !IsDispatcherMarker(snapshot.Current.Value) {
		t.Fatalf("snapshot = %+v", snapshot)
	}

	restored, err := RestoreSetup(RestoreSetupOptions{
		Paths:        options.Paths,
		SteamRoot:    root,
		SteamRunning: options.SteamRunning,
	})
	if err != nil {
		t.Fatalf("restore setup: %v", err)
	}
	if !restored.Removed && restored.RestoredValue != "" {
		t.Fatalf("restored = %+v", restored)
	}
	if after := readFixture(t, root); !bytes.Equal(after, original) {
		t.Fatal("restore did not reproduce the original localconfig")
	}
	loaded, err = LoadSetup(options.Paths)
	if err != nil || loaded == nil || loaded.Status != SetupRestored {
		t.Fatalf("restored setup = %+v, err=%v", loaded, err)
	}
	snapshot, err = DetectSetup(options.Paths, root)
	if err != nil || snapshot.Status != SetupRestored {
		t.Fatalf("restored snapshot = %+v, err=%v", snapshot, err)
	}
}

func mustBuildDispatcher(t *testing.T, helper string) string {
	t.Helper()
	value, err := BuildDispatcherLaunchOption(helper)
	if err != nil {
		t.Fatalf("build dispatcher: %v", err)
	}
	return value
}

func TestInstallSetupRefusesSteamRunningWithoutWriting(t *testing.T) {
	root := transactionRoot(t, "localconfig.vdf")
	original := readFixture(t, root)
	options := setupOptions(t, root)
	options.SteamRunning = func() (bool, error) { return true, nil }
	if _, err := InstallSetup(options); !errors.Is(err, domain.ErrSteamRunning) {
		t.Fatalf("error = %v, want ErrSteamRunning", err)
	}
	if after := readFixture(t, root); !bytes.Equal(after, original) {
		t.Fatal("localconfig changed while Steam was running")
	}
	if setup, err := LoadSetup(options.Paths); err != nil || setup != nil {
		t.Fatalf("setup = %+v, err=%v", setup, err)
	}
}

func TestInstallSetupRefusesLegacySession(t *testing.T) {
	root := transactionRoot(t, "localconfig.vdf")
	paths := testPaths(t)
	legacy, err := Prepare(PrepareOptions{
		Paths:        paths,
		SteamRoot:    root,
		HelperPath:   filepath.Join(paths.DataDir, "bin", "better-rpt-helper"),
		TargetAppID:  900001,
		SessionToken: sessionToken,
		SteamRunning: func() (bool, error) { return false, nil },
	})
	if err != nil {
		t.Fatalf("prepare legacy: %v", err)
	}
	options := setupOptions(t, root)
	options.Paths = paths
	if _, err := InstallSetup(options); !errors.Is(err, ErrSetupConflict) {
		t.Fatalf("error = %v, want ErrSetupConflict", err)
	}
	_, option, err := ReadCurrentLaunchOption(root)
	if err != nil || option.Value != legacy.NewValue {
		t.Fatalf("legacy marker changed: %+v, err=%v", option, err)
	}
}

func TestRecordGuideSetupBacksUpWithoutWriting(t *testing.T) {
	root := transactionRoot(t, "localconfig.vdf")
	original := readFixture(t, root)
	options := GuideSetupOptions{
		Paths:      testPaths(t),
		SteamRoot:  root,
		HelperPath: filepath.Join(t.TempDir(), "better-rpt-helper"),
	}
	setup, err := RecordGuideSetup(options)
	if err != nil {
		t.Fatalf("record guide: %v", err)
	}
	if setup.Status != SetupGuide {
		t.Fatalf("status = %q", setup.Status)
	}
	if setup.Backup.Meta.Kind != BackupDispatcher {
		t.Fatalf("kind = %q", setup.Backup.Meta.Kind)
	}
	if after := readFixture(t, root); !bytes.Equal(after, original) {
		t.Fatal("guide setup wrote localconfig")
	}
	snapshot, err := DetectSetup(options.Paths, root)
	if err != nil || snapshot.Status != SetupGuide {
		t.Fatalf("snapshot = %+v, err=%v", snapshot, err)
	}

	// Simulate the user pasting the dispatcher in Steam's UI while Steam was
	// open. Better RPT only reads the result afterwards.
	data, mode, err := readLocalConfig(setup.LocalConfig)
	if err != nil {
		t.Fatalf("read localconfig: %v", err)
	}
	edit, err := EditLaunchOption(data, setup.NewValue)
	if err != nil {
		t.Fatalf("edit launch option: %v", err)
	}
	if err := writeLocalConfig(setup.LocalConfig, edit.Data, mode); err != nil {
		t.Fatalf("write simulated paste: %v", err)
	}
	snapshot, err = DetectSetup(options.Paths, root)
	if err != nil || snapshot.Status != SetupInstalled {
		t.Fatalf("installed snapshot = %+v, err=%v", snapshot, err)
	}
	if _, err := RestoreSetup(RestoreSetupOptions{
		Paths:        options.Paths,
		SteamRoot:    root,
		SteamRunning: func() (bool, error) { return false, nil },
	}); err != nil {
		t.Fatalf("restore guide setup: %v", err)
	}
	if after := readFixture(t, root); !bytes.Equal(after, original) {
		t.Fatal("guide restore did not reproduce the original")
	}
}

func TestDetectSetupConflictDoesNotOverwriteUserChange(t *testing.T) {
	root := transactionRoot(t, "localconfig.vdf")
	options := setupOptions(t, root)
	setup, err := InstallSetup(options)
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	data, mode, err := readLocalConfig(setup.LocalConfig)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	edit, err := EditLaunchOption(data, "cd /tmp/user-change && ./run # %command%")
	if err != nil {
		t.Fatalf("edit: %v", err)
	}
	if err := writeLocalConfig(setup.LocalConfig, edit.Data, mode); err != nil {
		t.Fatalf("write user change: %v", err)
	}
	snapshot, err := DetectSetup(options.Paths, root)
	if err != nil || snapshot.Status != SetupConflict {
		t.Fatalf("snapshot = %+v, err=%v", snapshot, err)
	}
	if _, err := RestoreSetup(RestoreSetupOptions{
		Paths:        options.Paths,
		SteamRoot:    root,
		SteamRunning: func() (bool, error) { return false, nil },
	}); !errors.Is(err, ErrLocalConfigChanged) {
		t.Fatalf("restore error = %v, want ErrLocalConfigChanged", err)
	}
	_, option, err := ReadCurrentLaunchOption(root)
	if err != nil || option.Value != "cd /tmp/user-change && ./run # %command%" {
		t.Fatalf("user change overwritten: %+v, err=%v", option, err)
	}
}

func TestRestoreSetupMissing(t *testing.T) {
	root := transactionRoot(t, "localconfig.vdf")
	if _, err := RestoreSetup(RestoreSetupOptions{
		Paths:        testPaths(t),
		SteamRoot:    root,
		SteamRunning: func() (bool, error) { return false, nil },
	}); !errors.Is(err, ErrSetupMissing) {
		t.Fatalf("error = %v, want ErrSetupMissing", err)
	}
}

func TestRecordGuideSetupWithoutLaunchOption(t *testing.T) {
	root := transactionRoot(t, "localconfig_no_launchoptions.vdf")
	original := readFixture(t, root)
	options := GuideSetupOptions{
		Paths:      testPaths(t),
		SteamRoot:  root,
		HelperPath: filepath.Join(t.TempDir(), "better-rpt-helper"),
	}
	setup, err := RecordGuideSetup(options)
	if err != nil {
		t.Fatalf("record guide: %v", err)
	}
	if setup.Backup.Meta.OriginalValue != "" {
		t.Fatalf("original = %q, want empty", setup.Backup.Meta.OriginalValue)
	}
	snapshot, err := DetectSetup(options.Paths, root)
	if err != nil || snapshot.Status != SetupGuide {
		t.Fatalf("snapshot = %+v, err=%v, want guide", snapshot, err)
	}
	if _, err := RestoreSetup(RestoreSetupOptions{
		Paths:        options.Paths,
		SteamRoot:    root,
		SteamRunning: func() (bool, error) { return false, nil },
	}); err != nil {
		t.Fatalf("restore guide setup: %v", err)
	}
	if after := readFixture(t, root); !bytes.Equal(after, original) {
		t.Fatal("guide restore did not reproduce the original without a launch option")
	}
}
