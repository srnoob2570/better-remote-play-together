package config

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"time"

	"github.com/srnoob2570/better-rpt/internal/domain"
	"github.com/srnoob2570/better-rpt/internal/fsutil"
	"github.com/srnoob2570/better-rpt/internal/steam/vdf"
)

// RunningFunc reports whether any Steam client process is alive. The caller
// must supply it; a missing guard is a programming error, not permission to
// write while Steam is open.
type RunningFunc func() (bool, error)

// PrepareOptions describes one per-session SpaceWar launch-option edit.
type PrepareOptions struct {
	Paths        Paths
	SteamRoot    string
	HelperPath   string
	TargetAppID  uint32
	SessionToken string
	Now          time.Time
	SteamRunning RunningFunc
}

// Prepared contains the backup and original values needed to restore.
type Prepared struct {
	Account              SteamAccount
	LocalConfig          string
	Backup               Backup
	OriginalRaw          string
	OriginalValue        string
	NewValue             string
	CreatedAppBlock      bool
	CreatedLaunchOptions bool
	PruneError           error
}

// RestoreOptions describes one restore attempt.
type RestoreOptions struct {
	Paths         Paths
	SteamRoot     string
	BackupPath    string
	SessionToken  string
	ActiveSession bool
	SteamRunning  RunningFunc
}

// Restored reports the restored values.
type Restored struct {
	Account        SteamAccount
	LocalConfig    string
	Backup         Backup
	PreviousMarker string
	RestoredValue  string
	Removed        bool
}

// Stale describes a leftover marker with no live session.
type Stale struct {
	Account      SteamAccount
	LocalConfig  string
	Backup       Backup
	CurrentRaw   string
	CurrentValue string
	Active       bool
}

// Prepare backs up and writes the SpaceWar launch option for one legacy
// per-session transaction. New installations use InstallSetup instead.
func Prepare(options PrepareOptions) (Prepared, error) {
	if options.HelperPath == "" {
		options.HelperPath = options.Paths.HelperPath()
	}
	newValue, err := BuildLaunchOption(options.HelperPath, options.TargetAppID, options.SessionToken)
	if err != nil {
		return Prepared{}, err
	}
	return prepareLaunchOption(prepareRequest{
		Paths:        options.Paths,
		SteamRoot:    options.SteamRoot,
		TargetAppID:  options.TargetAppID,
		SessionToken: options.SessionToken,
		NewValue:     newValue,
		Kind:         BackupSession,
		Now:          options.Now,
		SteamRunning: options.SteamRunning,
	})
}

type prepareRequest struct {
	Paths        Paths
	SteamRoot    string
	TargetAppID  uint32
	SessionToken string
	NewValue     string
	Kind         BackupKind
	Now          time.Time
	SteamRunning RunningFunc
}

func prepareLaunchOption(options prepareRequest) (Prepared, error) {
	if options.SteamRunning == nil {
		return Prepared{}, ErrSteamGuardMissing
	}
	if options.SteamRoot == "" {
		return Prepared{}, fmt.Errorf("%w: empty Steam root", ErrLocalConfigMissing)
	}
	if options.NewValue == "" {
		return Prepared{}, fmt.Errorf("%w: empty launch option", ErrLocalConfigMissing)
	}
	running, err := options.SteamRunning()
	if err != nil {
		return Prepared{}, fmt.Errorf("Steam running check failed: %w", err)
	}
	if running {
		return Prepared{}, fmt.Errorf("%w: close Steam before preparing", domain.ErrSteamRunning)
	}

	account, err := SelectAccount(options.SteamRoot)
	if err != nil {
		return Prepared{}, err
	}
	localConfig := account.LocalConfigPath(options.SteamRoot)
	data, mode, err := readLocalConfig(localConfig)
	if err != nil {
		return Prepared{}, err
	}
	edit, err := EditLaunchOption(data, options.NewValue)
	if err != nil {
		return Prepared{}, err
	}
	if edit.OriginalRaw != "" && edit.OriginalValue == options.NewValue {
		return Prepared{}, ErrAlreadyPrepared
	}

	originalRaw := edit.OriginalRaw
	originalValue := edit.OriginalValue
	createdAppBlock := edit.CreatedAppBlock
	createdLaunchOptions := edit.CreatedLaunchOptions
	if options.Kind == BackupSession {
		previous, hasPrevious, err := findStaleBackup(options.Paths, account, localConfig, edit.OriginalValue)
		if err != nil {
			return Prepared{}, err
		}
		if hasPrevious {
			if previous.Meta.OriginalRawTokenBase64 == "" && previous.Meta.OriginalValue != "" {
				return Prepared{}, fmt.Errorf("%w: previous backup has no original launch option token", ErrBackupCorrupt)
			}
			raw, err := base64.StdEncoding.DecodeString(previous.Meta.OriginalRawTokenBase64)
			if err != nil {
				return Prepared{}, fmt.Errorf("%w: decode previous original token: %v", ErrBackupCorrupt, err)
			}
			originalRaw = string(raw)
			originalValue = previous.Meta.OriginalValue
			createdAppBlock = previous.Meta.CreatedAppBlock
			createdLaunchOptions = previous.Meta.CreatedLaunchOptions
		} else if IsSessionMarker(edit.OriginalValue) {
			return Prepared{}, fmt.Errorf("%w: current SpaceWar launch option is from Better RPT but has no backup", ErrBackupMissing)
		}
	} else {
		if IsSessionMarker(edit.OriginalValue) {
			return Prepared{}, fmt.Errorf("%w: restore the previous Better RPT session first", ErrSetupConflict)
		}
		if IsDispatcherMarker(edit.OriginalValue) && edit.OriginalValue != options.NewValue {
			return Prepared{}, fmt.Errorf("%w: a different dispatcher is already installed", ErrSetupConflict)
		}
	}

	meta := BackupMeta{
		Kind:                   options.Kind,
		Source:                 localConfig,
		SteamID64:              account.SteamID64,
		AccountID:              account.AccountID,
		AccountName:            account.AccountName,
		AppID:                  domain.SpaceWarAppID,
		TargetAppID:            options.TargetAppID,
		SessionToken:           options.SessionToken,
		OriginalMode:           uint32(mode.Perm()),
		OriginalRawTokenBase64: base64.StdEncoding.EncodeToString([]byte(originalRaw)),
		OriginalValue:          originalValue,
		NewValue:               options.NewValue,
		CreatedAppBlock:        createdAppBlock,
		CreatedLaunchOptions:   createdLaunchOptions,
		CreatedAt:              options.Now,
	}
	backup, err := CreateBackup(options.Paths, localConfig, data, meta)
	if err != nil {
		return Prepared{}, err
	}
	if running, err := options.SteamRunning(); err != nil {
		return Prepared{}, fmt.Errorf("Steam running check failed: %w", err)
	} else if running {
		return Prepared{}, fmt.Errorf("%w: Steam started during preparation", domain.ErrSteamRunning)
	}
	current, currentMode, err := readLocalConfig(localConfig)
	if err != nil {
		return Prepared{}, err
	}
	if !bytes.Equal(current, data) || currentMode != mode {
		return Prepared{}, fmt.Errorf("%w: localconfig changed during preparation", ErrLocalConfigChanged)
	}
	if err := writeLocalConfig(localConfig, edit.Data, mode); err != nil {
		rollbackErr := writeLocalConfig(localConfig, data, mode)
		if rollbackErr != nil {
			return Prepared{}, fmt.Errorf("%w: %v; backup=%s; rollback failed: %v",
				ErrLocalConfigUnsafe, err, backup.Path, rollbackErr)
		}
		return Prepared{}, fmt.Errorf("%w: %v; original restored; backup=%s",
			ErrLocalConfigUnsafe, err, backup.Path)
	}
	if err := verifyPrepared(localConfig, data, options.NewValue); err != nil {
		rollbackErr := writeLocalConfig(localConfig, data, mode)
		if rollbackErr != nil {
			return Prepared{}, fmt.Errorf("%w: %v; backup=%s; rollback failed: %v",
				ErrLocalConfigVerify, err, backup.Path, rollbackErr)
		}
		return Prepared{}, fmt.Errorf("%w: %v; original restored; backup=%s",
			ErrLocalConfigVerify, err, backup.Path)
	}

	prepared := Prepared{
		Account:              account,
		LocalConfig:          localConfig,
		Backup:               backup,
		OriginalRaw:          originalRaw,
		OriginalValue:        originalValue,
		NewValue:             options.NewValue,
		CreatedAppBlock:      createdAppBlock,
		CreatedLaunchOptions: createdLaunchOptions,
	}
	if err := PruneBackups(options.Paths, backupRetention, backup.Path); err != nil {
		prepared.PruneError = err
	}
	return prepared, nil
}

// Restore undoes a prepared edit, but only while the current decoded value is
// exactly the marker recorded in the backup.
func Restore(options RestoreOptions) (Restored, error) {
	if options.ActiveSession {
		return Restored{}, ErrSessionActive
	}
	if options.SteamRunning == nil {
		return Restored{}, ErrSteamGuardMissing
	}
	if options.SteamRoot == "" {
		return Restored{}, fmt.Errorf("%w: empty Steam root", ErrLocalConfigMissing)
	}
	running, err := options.SteamRunning()
	if err != nil {
		return Restored{}, fmt.Errorf("Steam running check failed: %w", err)
	}
	if running {
		return Restored{}, fmt.Errorf("%w: close Steam before restoring", domain.ErrSteamRunning)
	}

	var backup Backup
	switch {
	case options.BackupPath != "":
		backup, err = LoadBackup(options.BackupPath)
	case options.SessionToken != "":
		backup, err = FindBackupBySession(options.Paths, options.SessionToken)
	default:
		return Restored{}, fmt.Errorf("%w: specify a backup or session", ErrBackupMissing)
	}
	if err != nil {
		return Restored{}, err
	}
	if options.SessionToken != "" && backup.Meta.SessionToken != options.SessionToken {
		return Restored{}, fmt.Errorf("%w: session mismatch", ErrBackupMissing)
	}

	account := SteamAccount{
		SteamID64:   backup.Meta.SteamID64,
		AccountID:   backup.Meta.AccountID,
		AccountName: backup.Meta.AccountName,
	}
	localConfig := account.LocalConfigPath(options.SteamRoot)
	if backup.Meta.Source != "" && filepath.Clean(backup.Meta.Source) != filepath.Clean(localConfig) {
		return Restored{}, fmt.Errorf("%w: backup source does not match selected account", ErrBackupCorrupt)
	}
	data, mode, err := readLocalConfig(localConfig)
	if err != nil {
		return Restored{}, err
	}
	originalRaw, err := base64.StdEncoding.DecodeString(backup.Meta.OriginalRawTokenBase64)
	if err != nil {
		return Restored{}, fmt.Errorf("%w: decode original token: %v", ErrBackupCorrupt, err)
	}
	spec := RestoreSpec{
		NewValue:             backup.Meta.NewValue,
		OriginalRaw:          string(originalRaw),
		OriginalValue:        backup.Meta.OriginalValue,
		CreatedAppBlock:      backup.Meta.CreatedAppBlock,
		CreatedLaunchOptions: backup.Meta.CreatedLaunchOptions,
	}
	restoredData, err := RestoreLaunchOption(data, spec)
	if err != nil {
		return Restored{}, err
	}
	if running, err := options.SteamRunning(); err != nil {
		return Restored{}, fmt.Errorf("Steam running check failed: %w", err)
	} else if running {
		return Restored{}, fmt.Errorf("%w: Steam started during restore", domain.ErrSteamRunning)
	}
	current, currentMode, err := readLocalConfig(localConfig)
	if err != nil {
		return Restored{}, err
	}
	if !bytes.Equal(current, data) || currentMode != mode {
		return Restored{}, fmt.Errorf("%w: localconfig changed during restore", ErrLocalConfigChanged)
	}
	if err := writeLocalConfig(localConfig, restoredData, mode); err != nil {
		rollbackErr := writeLocalConfig(localConfig, data, mode)
		if rollbackErr != nil {
			return Restored{}, fmt.Errorf("%w: %v; backup=%s; rollback failed: %v",
				ErrLocalConfigUnsafe, err, backup.Path, rollbackErr)
		}
		return Restored{}, fmt.Errorf("%w: %v; original restored; backup=%s",
			ErrLocalConfigUnsafe, err, backup.Path)
	}
	if err := verifyRestored(localConfig, data, spec); err != nil {
		rollbackErr := writeLocalConfig(localConfig, data, mode)
		if rollbackErr != nil {
			return Restored{}, fmt.Errorf("%w: %v; backup=%s; rollback failed: %v",
				ErrLocalConfigVerify, err, backup.Path, rollbackErr)
		}
		return Restored{}, fmt.Errorf("%w: %v; original restored; backup=%s",
			ErrLocalConfigVerify, err, backup.Path)
	}
	return Restored{
		Account:        account,
		LocalConfig:    localConfig,
		Backup:         backup,
		PreviousMarker: spec.NewValue,
		RestoredValue:  spec.OriginalValue,
		Removed:        spec.OriginalRaw == "",
	}, nil
}

// DetectStale finds our marker in the active account's localconfig and a
// matching backup. It does not restore anything.
func DetectStale(paths Paths, steamRoot string, activeSession bool) (*Stale, error) {
	if steamRoot == "" {
		return nil, fmt.Errorf("%w: empty Steam root", ErrLocalConfigMissing)
	}
	account, err := SelectAccount(steamRoot)
	if err != nil {
		return nil, err
	}
	localConfig := account.LocalConfigPath(steamRoot)
	data, _, err := readLocalConfig(localConfig)
	if err != nil {
		return nil, err
	}
	option, err := ReadLaunchOption(data)
	if err != nil {
		return nil, err
	}
	if !option.Exists {
		return nil, nil
	}
	backup, ok, err := findStaleBackup(paths, account, localConfig, option.Value)
	if err != nil {
		return nil, err
	}
	if ok {
		return &Stale{
			Account:      account,
			LocalConfig:  localConfig,
			Backup:       backup,
			CurrentRaw:   option.Raw,
			CurrentValue: option.Value,
			Active:       activeSession,
		}, nil
	}
	if IsSessionMarker(option.Value) {
		return &Stale{
			Account:      account,
			LocalConfig:  localConfig,
			CurrentRaw:   option.Raw,
			CurrentValue: option.Value,
			Active:       activeSession,
		}, nil
	}
	return nil, nil
}

// findStaleBackup returns the backup whose written marker owns the current
// launch-option value. It only matches backups for the selected account.
func findStaleBackup(paths Paths, account SteamAccount, localConfig, value string) (Backup, bool, error) {
	if value == "" {
		return Backup{}, false, nil
	}
	if IsDispatcherMarker(value) {
		return Backup{}, false, nil
	}
	backups, err := ListBackups(paths)
	if err != nil {
		return Backup{}, false, err
	}
	for _, backup := range backups {
		if backup.Meta.Kind == BackupDispatcher {
			continue
		}
		if backup.Meta.NewValue != value {
			continue
		}
		if backup.Meta.Source != "" && filepath.Clean(backup.Meta.Source) != filepath.Clean(localConfig) {
			continue
		}
		if backup.Meta.AccountID != 0 && backup.Meta.AccountID != account.AccountID {
			continue
		}
		if backup.Meta.SteamID64 != "" && backup.Meta.SteamID64 != account.SteamID64 {
			continue
		}
		return backup, true, nil
	}
	return Backup{}, false, nil
}

// IsSessionMarker reports whether a launch-option value has the shape Better
// RPT writes. It lets startup detect an orphan marker even when its backup is
// missing; it is not an authorization to clear the value.
func IsSessionMarker(value string) bool {
	const suffix = " # %command%"
	if !strings.HasSuffix(value, suffix) {
		return false
	}
	body := strings.TrimSuffix(value, suffix)
	index := strings.LastIndex(body, "--session ")
	if index < 0 {
		return false
	}
	token := body[index+len("--session "):]
	return domain.ValidSessionToken(token) &&
		strings.Contains(body, "--donor") &&
		strings.Contains(body, "--target ")
}

// ReadCurrentLaunchOption reads the active account's current launch option
// without modifying anything.
func ReadCurrentLaunchOption(steamRoot string) (SteamAccount, LaunchOption, error) {
	if steamRoot == "" {
		return SteamAccount{}, LaunchOption{}, fmt.Errorf("%w: empty Steam root", ErrLocalConfigMissing)
	}
	account, err := SelectAccount(steamRoot)
	if err != nil {
		return SteamAccount{}, LaunchOption{}, err
	}
	data, _, err := readLocalConfig(account.LocalConfigPath(steamRoot))
	if err != nil {
		return SteamAccount{}, LaunchOption{}, err
	}
	option, err := ReadLaunchOption(data)
	if err != nil {
		return SteamAccount{}, LaunchOption{}, err
	}
	return account, option, nil
}

func verifyPrepared(path string, original []byte, expected string) error {
	updated, _, err := readLocalConfig(path)
	if err != nil {
		return err
	}
	option, err := ReadLaunchOption(updated)
	if err != nil {
		return err
	}
	if !option.Exists || option.Value != expected {
		return fmt.Errorf("%w: launch option is not the expected marker", ErrLocalConfigVerify)
	}
	return verifySemanticExceptLaunchOption(original, updated)
}

func verifyRestored(path string, before []byte, spec RestoreSpec) error {
	updated, _, err := readLocalConfig(path)
	if err != nil {
		return err
	}
	option, err := ReadLaunchOption(updated)
	if err != nil {
		return err
	}
	if spec.OriginalRaw != "" {
		if !option.Exists || option.Value != spec.OriginalValue {
			return fmt.Errorf("%w: original launch option is not present", ErrLocalConfigVerify)
		}
	} else if option.Exists && option.Value != "" {
		return fmt.Errorf("%w: launch option was not cleared", ErrLocalConfigVerify)
	}
	return verifySemanticExceptLaunchOption(before, updated)
}

func verifySemanticExceptLaunchOption(original, updated []byte) error {
	before, err := vdf.Parse(bytes.NewReader(original))
	if err != nil {
		return fmt.Errorf("%w: parse original: %v", ErrLocalConfigVerify, err)
	}
	after, err := vdf.Parse(bytes.NewReader(updated))
	if err != nil {
		return fmt.Errorf("%w: parse updated: %v", ErrLocalConfigVerify, err)
	}
	if err := setMapPath(before, launchOptionFullPath, "\x00better-rpt-placeholder"); err != nil {
		return fmt.Errorf("%w: original path: %v", ErrLocalConfigVerify, err)
	}
	if err := setMapPath(after, launchOptionFullPath, "\x00better-rpt-placeholder"); err != nil {
		return fmt.Errorf("%w: updated path: %v", ErrLocalConfigVerify, err)
	}
	if !reflect.DeepEqual(before, after) {
		return fmt.Errorf("%w: unrelated localconfig data changed", ErrLocalConfigVerify)
	}
	return nil
}

var launchOptionFullPath = []string{
	"UserLocalConfigStore", "Software", "Valve", "Steam", "apps", "480", "LaunchOptions",
}

func setMapPath(root map[string]interface{}, path []string, value string) error {
	current := root
	for index, key := range path {
		if index == len(path)-1 {
			current[key] = value
			return nil
		}
		raw, exists := current[key]
		if !exists {
			next := make(map[string]interface{})
			current[key] = next
			current = next
			continue
		}
		next, ok := raw.(map[string]interface{})
		if !ok {
			return fmt.Errorf("%q is not an object", key)
		}
		current = next
	}
	return nil
}

func readLocalConfig(path string) ([]byte, os.FileMode, error) {
	if os.Geteuid() == 0 {
		return nil, 0, fmt.Errorf("%w: refusing to edit Steam as root", domain.ErrRoot)
	}
	if size, err := syscall.Listxattr(path, nil); err == nil && size > 0 {
		return nil, 0, fmt.Errorf("%w: %s has extended attributes or ACLs", ErrLocalConfigUnsafe, path)
	} else if err != nil && !errors.Is(err, syscall.ENOTSUP) && !errors.Is(err, syscall.ENOSYS) {
		if errors.Is(err, syscall.ENOENT) {
			return nil, 0, fmt.Errorf("%w: %s", ErrLocalConfigMissing, path)
		}
		return nil, 0, fmt.Errorf("%w: inspect %s: %v", ErrLocalConfigUnsafe, path, err)
	}

	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW, 0)
	if err != nil {
		switch {
		case errors.Is(err, syscall.ENOENT):
			return nil, 0, fmt.Errorf("%w: %s", ErrLocalConfigMissing, path)
		case errors.Is(err, syscall.ELOOP):
			return nil, 0, fmt.Errorf("%w: %s is a symlink", ErrLocalConfigUnsafe, path)
		default:
			return nil, 0, fmt.Errorf("%w: open %s: %v", ErrLocalConfigUnsafe, path, err)
		}
	}
	file := os.NewFile(uintptr(fd), path)
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, 0, fmt.Errorf("%w: stat %s: %v", ErrLocalConfigUnsafe, path, err)
	}
	if !info.Mode().IsRegular() {
		return nil, 0, fmt.Errorf("%w: %s is not a regular file", ErrLocalConfigUnsafe, path)
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return nil, 0, fmt.Errorf("%w: cannot inspect owner of %s", ErrLocalConfigUnsafe, path)
	}
	if stat.Uid != uint32(os.Geteuid()) {
		return nil, 0, fmt.Errorf("%w: %s is owned by uid %d", ErrLocalConfigUnsafe, path, stat.Uid)
	}
	if info.Mode().Perm()&0o200 == 0 {
		return nil, 0, fmt.Errorf("%w: %s is not writable by its owner", ErrLocalConfigUnsafe, path)
	}
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: read %s: %v", ErrLocalConfigUnsafe, path, err)
	}
	after, err := file.Stat()
	if err != nil || after.Size() != int64(len(data)) || !after.ModTime().Equal(info.ModTime()) {
		return nil, 0, fmt.Errorf("%w: %s changed while reading", ErrLocalConfigChanged, path)
	}
	return data, info.Mode().Perm(), nil
}

func writeLocalConfig(path string, data []byte, mode os.FileMode) error {
	if os.Geteuid() == 0 {
		return fmt.Errorf("%w: refusing to edit Steam as root", domain.ErrRoot)
	}
	if mode == 0 {
		mode = 0o600
	}
	if err := fsutil.AtomicWriteFile(path, data, mode); err != nil {
		return fmt.Errorf("%w: write %s: %v", ErrLocalConfigUnsafe, path, err)
	}
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("%w: stat after write %s: %v", ErrLocalConfigUnsafe, path, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%w: %s is not regular after write", ErrLocalConfigUnsafe, path)
	}
	if stat, ok := info.Sys().(*syscall.Stat_t); !ok || stat.Uid != uint32(os.Geteuid()) {
		return fmt.Errorf("%w: %s owner changed after write", ErrLocalConfigUnsafe, path)
	}
	return nil
}
