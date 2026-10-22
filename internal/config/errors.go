package config

import "errors"

var (
	// ErrAccountNotFound means no Steam account could be selected.
	ErrAccountNotFound = errors.New("account not found")
	// ErrAccountAmbiguous means several accounts are equally plausible.
	ErrAccountAmbiguous = errors.New("account is ambiguous")
	// ErrAccountData means loginusers/registry data could not be read.
	ErrAccountData = errors.New("login data is unreadable")
	// ErrLocalConfigMissing means the selected account has no localconfig.vdf.
	ErrLocalConfigMissing = errors.New("localconfig.vdf is missing")
	// ErrLocalConfigUnsafe means the file or target path cannot be edited safely.
	ErrLocalConfigUnsafe = errors.New("localconfig.vdf is unsafe to edit")
	// ErrLocalConfigChanged means the current launch option is not the one Better
	// RPT wrote, so restoring would overwrite someone else's change.
	ErrLocalConfigChanged = errors.New("launch option changed outside Better RPT")
	// ErrLocalConfigVerify means the post-write semantic verification failed.
	ErrLocalConfigVerify = errors.New("localconfig.vdf verification failed")
	// ErrLocalConfigInvalid means the localconfig could not be parsed safely.
	ErrLocalConfigInvalid = errors.New("localconfig.vdf is invalid")
	// ErrBackup means a backup could not be created or read.
	ErrBackup = errors.New("localconfig backup failed")
	// ErrBackupCorrupt means a backup's metadata or content hash is invalid.
	ErrBackupCorrupt = errors.New("localconfig backup is corrupt")
	// ErrBackupMissing means no backup matches the requested session or account.
	ErrBackupMissing = errors.New("localconfig backup not found")
	// ErrSteamGuardMissing means the caller did not provide a running check, so
	// the transaction must not write.
	ErrSteamGuardMissing = errors.New("running Steam check is required")
	// ErrAlreadyPrepared means localconfig already contains this exact launch
	// option; a second preparation would create a duplicate session.
	ErrAlreadyPrepared = errors.New("launch option is already prepared")
	// ErrSessionActive means restoration was attempted while a session is live.
	ErrSessionActive = errors.New("a Better RPT session is active")
	// ErrSetupMissing means no persistent dispatcher setup is recorded.
	ErrSetupMissing = errors.New("dispatcher setup is missing")
	// ErrSetupConflict means the current Steam configuration no longer matches
	// the setup record, so Better RPT must not overwrite it.
	ErrSetupConflict = errors.New("dispatcher setup conflicts with Steam")
	// ErrSetupInvalid means the setup record is malformed.
	ErrSetupInvalid = errors.New("dispatcher setup is invalid")
	// ErrControllerConfig means a SpaceWar Steam Input configset could not be
	// inspected or updated safely.
	ErrControllerConfig = errors.New("controller config failed")
	// ErrControllerConfigVerify means the post-write controller config did not
	// contain the expected template.
	ErrControllerConfigVerify = errors.New("controller config verification failed")
)
