package config

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/srnoob2570/better-rpt/internal/domain"
	"github.com/srnoob2570/better-rpt/internal/fsutil"
)

// SetupVersion is the durable dispatcher setup schema version.
const SetupVersion = 1

// SetupStatus is the durable status of the persistent dispatcher setup.
type SetupStatus string

const (
	SetupMissing   SetupStatus = "missing"
	SetupGuide     SetupStatus = "guide"
	SetupInstalled SetupStatus = "installed"
	SetupRestored  SetupStatus = "restored"
	SetupConflict  SetupStatus = "conflict"
)

// Setup is Better RPT's record of the persistent SpaceWar dispatcher. It is
// written even when the user installs the command manually through Steam, so
// the original value and conflict checks remain available.
type Setup struct {
	Version     int          `json:"version"`
	Status      SetupStatus  `json:"status"`
	Account     SteamAccount `json:"account"`
	LocalConfig string       `json:"local_config"`
	Backup      Backup       `json:"backup"`
	HelperPath  string       `json:"helper_path"`
	NewValue    string       `json:"new_value"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

// SetupSnapshot is the read-only result of inspecting setup plus the current
// SpaceWar launch option.
type SetupSnapshot struct {
	Status      SetupStatus
	Setup       *Setup
	Account     SteamAccount
	LocalConfig string
	Current     LaunchOption
}

// InstallSetupOptions describes one automatic dispatcher installation. It
// requires Steam to be closed because it writes localconfig directly.
type InstallSetupOptions struct {
	Paths        Paths
	SteamRoot    string
	HelperPath   string
	Now          time.Time
	SteamRunning RunningFunc
}

// GuideSetupOptions describes a manual dispatcher installation. It only reads
// Steam's localconfig and records a backup for a later restore.
type GuideSetupOptions struct {
	Paths      Paths
	SteamRoot  string
	HelperPath string
	Now        time.Time
}

// RestoreSetupOptions describes one dispatcher restore. It requires Steam to be
// closed; the caller shows a manual guide while Steam is open.
type RestoreSetupOptions struct {
	Paths        Paths
	SteamRoot    string
	SteamRunning RunningFunc
}

// SetupPath returns the durable setup record path.
func SetupPath(paths Paths) string {
	return filepath.Join(paths.DataDir, "setup.json")
}

// LoadSetup reads the dispatcher setup. A missing record is not an error.
func LoadSetup(paths Paths) (*Setup, error) {
	path := SetupPath(paths)
	info, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("%w: inspect %s: %v", ErrSetupInvalid, path, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%w: %s is not a regular file", ErrSetupInvalid, path)
	}
	if info.Mode().Perm()&0o077 != 0 {
		return nil, fmt.Errorf("%w: %s is not private", ErrSetupInvalid, path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%w: read %s: %v", ErrSetupInvalid, path, err)
	}
	var setup Setup
	if err := json.Unmarshal(data, &setup); err != nil {
		return nil, fmt.Errorf("%w: parse %s: %v", ErrSetupInvalid, path, err)
	}
	if err := setup.Validate(); err != nil {
		return nil, err
	}
	return &setup, nil
}

// Validate checks the durable setup shape.
func (s Setup) Validate() error {
	if s.Version != SetupVersion {
		return fmt.Errorf("%w: version %d", ErrSetupInvalid, s.Version)
	}
	switch s.Status {
	case SetupGuide, SetupInstalled, SetupRestored, SetupConflict:
	default:
		return fmt.Errorf("%w: status %q", ErrSetupInvalid, s.Status)
	}
	if s.LocalConfig == "" || s.NewValue == "" || s.HelperPath == "" {
		return fmt.Errorf("%w: incomplete record", ErrSetupInvalid)
	}
	if s.Backup.Meta.NewValue != s.NewValue || s.Backup.Meta.Kind != BackupDispatcher {
		return fmt.Errorf("%w: backup does not match setup", ErrSetupInvalid)
	}
	if s.CreatedAt.IsZero() {
		return fmt.Errorf("%w: missing created_at", ErrSetupInvalid)
	}
	return nil
}

// SaveSetup writes the setup record atomically.
func SaveSetup(paths Paths, setup Setup) error {
	if setup.Version == 0 {
		setup.Version = SetupVersion
	}
	if setup.UpdatedAt.IsZero() {
		setup.UpdatedAt = time.Now().UTC()
	}
	if err := setup.Validate(); err != nil {
		return err
	}
	if err := paths.Ensure(); err != nil {
		return fmt.Errorf("%w: %v", ErrSetupInvalid, err)
	}
	data, err := json.MarshalIndent(setup, "", "  ")
	if err != nil {
		return fmt.Errorf("%w: encode: %v", ErrSetupInvalid, err)
	}
	data = append(data, '\n')
	if err := fsutil.AtomicWriteFile(SetupPath(paths), data, 0o600); err != nil {
		return fmt.Errorf("%w: write: %v", ErrSetupInvalid, err)
	}
	return nil
}

// InstallSetup writes the persistent dispatcher and records its backup. It
// refuses to write while Steam is running and never starts or stops Steam.
func InstallSetup(options InstallSetupOptions) (Setup, error) {
	if options.HelperPath == "" {
		options.HelperPath = options.Paths.HelperPath()
	}
	newValue, err := BuildDispatcherLaunchOption(options.HelperPath)
	if err != nil {
		return Setup{}, err
	}
	now := options.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	prepared, err := prepareLaunchOption(prepareRequest{
		Paths:        options.Paths,
		SteamRoot:    options.SteamRoot,
		NewValue:     newValue,
		Kind:         BackupDispatcher,
		Now:          now,
		SteamRunning: options.SteamRunning,
	})
	if err != nil {
		return Setup{}, err
	}
	setup := Setup{
		Version:     SetupVersion,
		Status:      SetupInstalled,
		Account:     prepared.Account,
		LocalConfig: prepared.LocalConfig,
		Backup:      prepared.Backup,
		HelperPath:  options.HelperPath,
		NewValue:    newValue,
		CreatedAt:   now.UTC(),
		UpdatedAt:   now.UTC(),
	}
	if err := SaveSetup(options.Paths, setup); err != nil {
		rollback, rollbackErr := Restore(RestoreOptions{
			Paths:        options.Paths,
			SteamRoot:    options.SteamRoot,
			BackupPath:   prepared.Backup.Path,
			SteamRunning: options.SteamRunning,
		})
		_ = rollback
		if rollbackErr != nil {
			return Setup{}, fmt.Errorf("%w; rollback failed: %v", err, rollbackErr)
		}
		return Setup{}, err
	}
	return setup, nil
}

// RecordGuideSetup creates the backup and setup record for a manual paste
// without writing Steam's localconfig.
func RecordGuideSetup(options GuideSetupOptions) (Setup, error) {
	if options.SteamRoot == "" {
		return Setup{}, fmt.Errorf("%w: empty Steam root", ErrLocalConfigMissing)
	}
	if options.HelperPath == "" {
		options.HelperPath = options.Paths.HelperPath()
	}
	newValue, err := BuildDispatcherLaunchOption(options.HelperPath)
	if err != nil {
		return Setup{}, err
	}
	account, err := SelectAccount(options.SteamRoot)
	if err != nil {
		return Setup{}, err
	}
	localConfig := account.LocalConfigPath(options.SteamRoot)
	data, mode, err := readLocalConfig(localConfig)
	if err != nil {
		return Setup{}, err
	}
	edit, err := EditLaunchOption(data, newValue)
	if err != nil {
		return Setup{}, err
	}
	if IsSessionMarker(edit.OriginalValue) || IsDispatcherMarker(edit.OriginalValue) {
		return Setup{}, fmt.Errorf("%w: SpaceWar already points at Better RPT; restore it first", ErrSetupConflict)
	}
	now := options.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	meta := BackupMeta{
		Kind:                   BackupDispatcher,
		Source:                 localConfig,
		SteamID64:              account.SteamID64,
		AccountID:              account.AccountID,
		AccountName:            account.AccountName,
		AppID:                  domain.SpaceWarAppID,
		OriginalMode:           uint32(mode.Perm()),
		OriginalRawTokenBase64: base64.StdEncoding.EncodeToString([]byte(edit.OriginalRaw)),
		OriginalValue:          edit.OriginalValue,
		NewValue:               newValue,
		CreatedAppBlock:        edit.CreatedAppBlock,
		CreatedLaunchOptions:   edit.CreatedLaunchOptions,
		CreatedAt:              now.UTC(),
	}
	backup, err := CreateBackup(options.Paths, localConfig, data, meta)
	if err != nil {
		return Setup{}, err
	}
	setup := Setup{
		Version:     SetupVersion,
		Status:      SetupGuide,
		Account:     account,
		LocalConfig: localConfig,
		Backup:      backup,
		HelperPath:  options.HelperPath,
		NewValue:    newValue,
		CreatedAt:   now.UTC(),
		UpdatedAt:   now.UTC(),
	}
	if err := SaveSetup(options.Paths, setup); err != nil {
		return Setup{}, err
	}
	return setup, nil
}

// DetectSetup compares the setup record with the current launch option. It
// never writes anything.
func DetectSetup(paths Paths, steamRoot string) (SetupSnapshot, error) {
	state, err := ReadLocalState(steamRoot)
	if err != nil {
		return SetupSnapshot{}, err
	}
	return DetectSetupFromState(paths, state)
}

// DetectSetupFromState compares the setup record with an already-read
// localconfig snapshot.
func DetectSetupFromState(paths Paths, state LocalState) (SetupSnapshot, error) {
	setup, err := LoadSetup(paths)
	if err != nil {
		return SetupSnapshot{}, err
	}
	snapshot := SetupSnapshot{
		Status:      SetupMissing,
		Setup:       setup,
		Account:     state.Account,
		LocalConfig: state.Path,
		Current:     state.Launch,
	}
	if setup == nil {
		if state.Launch.Exists && IsDispatcherMarker(state.Launch.Value) {
			// The user already pasted the dispatcher through Steam. Better RPT
			// can launch with it, but has no original value to restore.
			snapshot.Status = SetupInstalled
		}
		return snapshot, nil
	}
	if !sameSteamAccount(setup.Account, state.Account) {
		snapshot.Status = SetupConflict
		return snapshot, nil
	}
	switch setup.Status {
	case SetupGuide, SetupInstalled:
		switch {
		case launchOptionMatches(state.Launch, setup.NewValue):
			snapshot.Status = SetupInstalled
		case launchOptionMatches(state.Launch, setup.Backup.Meta.OriginalValue):
			snapshot.Status = SetupGuide
		default:
			snapshot.Status = SetupConflict
		}
	case SetupRestored:
		if launchOptionMatches(state.Launch, setup.Backup.Meta.OriginalValue) {
			snapshot.Status = SetupRestored
		} else {
			snapshot.Status = SetupConflict
		}
	default:
		snapshot.Status = SetupConflict
	}
	return snapshot, nil
}

func launchOptionMatches(option LaunchOption, value string) bool {
	if value == "" {
		return !option.Exists || option.Value == ""
	}
	return option.Exists && option.Value == value
}

// RestoreSetup undoes the persistent dispatcher while Steam is closed.
func RestoreSetup(options RestoreSetupOptions) (Restored, error) {
	setup, err := LoadSetup(options.Paths)
	if err != nil {
		return Restored{}, err
	}
	if setup == nil {
		return Restored{}, ErrSetupMissing
	}
	account, current, err := ReadCurrentLaunchOption(options.SteamRoot)
	if err != nil {
		return Restored{}, err
	}
	if !sameSteamAccount(setup.Account, account) {
		return Restored{}, fmt.Errorf("%w: setup belongs to another Steam account", ErrSetupConflict)
	}
	originalRaw, err := base64.StdEncoding.DecodeString(setup.Backup.Meta.OriginalRawTokenBase64)
	if err != nil {
		return Restored{}, fmt.Errorf("%w: decode original token: %v", ErrBackupCorrupt, err)
	}
	if launchOptionMatches(current, setup.Backup.Meta.OriginalValue) {
		setup.Status = SetupRestored
		setup.UpdatedAt = time.Now().UTC()
		if err := SaveSetup(options.Paths, *setup); err != nil {
			return Restored{}, err
		}
		return Restored{
			Account:        account,
			LocalConfig:    setup.LocalConfig,
			Backup:         setup.Backup,
			PreviousMarker: setup.NewValue,
			RestoredValue:  setup.Backup.Meta.OriginalValue,
			Removed:        string(originalRaw) == "",
		}, nil
	}
	if !launchOptionMatches(current, setup.NewValue) {
		return Restored{}, fmt.Errorf("%w: current SpaceWar launch option is neither the dispatcher nor the original", ErrLocalConfigChanged)
	}
	restored, err := Restore(RestoreOptions{
		Paths:        options.Paths,
		SteamRoot:    options.SteamRoot,
		BackupPath:   setup.Backup.Path,
		SteamRunning: options.SteamRunning,
	})
	if err != nil {
		return Restored{}, err
	}
	setup.Status = SetupRestored
	setup.UpdatedAt = time.Now().UTC()
	if err := SaveSetup(options.Paths, *setup); err != nil {
		return restored, err
	}
	return restored, nil
}

func sameSteamAccount(a, b SteamAccount) bool {
	if a.SteamID64 == "" || b.SteamID64 == "" {
		return true
	}
	return a.SteamID64 == b.SteamID64 && a.AccountID == b.AccountID
}
