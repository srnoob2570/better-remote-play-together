package config

import "errors"

var (
	// ErrAccountNotFound means no Steam account could be selected.
	ErrAccountNotFound = errors.New("Steam account not found")
	// ErrAccountAmbiguous means several accounts are equally plausible.
	ErrAccountAmbiguous = errors.New("Steam account is ambiguous")
	// ErrAccountData means loginusers/registry data could not be read.
	ErrAccountData = errors.New("Steam login data is unreadable")
	// ErrLocalConfigMissing means the selected account has no localconfig.vdf.
	ErrLocalConfigMissing = errors.New("Steam localconfig.vdf is missing")
	// ErrLocalConfigUnsafe means the file or target path cannot be edited safely.
	ErrLocalConfigUnsafe = errors.New("Steam localconfig.vdf is unsafe to edit")
	// ErrLocalConfigChanged means the current launch option is not the one Better
	// RPT wrote, so restoring would overwrite someone else's change.
	ErrLocalConfigChanged = errors.New("Steam launch option changed outside Better RPT")
	// ErrLocalConfigVerify means the post-write semantic verification failed.
	ErrLocalConfigVerify = errors.New("Steam localconfig.vdf verification failed")
	// ErrBackup means a backup could not be created or read.
	ErrBackup = errors.New("Steam localconfig backup failed")
	// ErrBackupCorrupt means a backup's metadata or content hash is invalid.
	ErrBackupCorrupt = errors.New("Steam localconfig backup is corrupt")
	// ErrBackupMissing means no backup matches the requested session or account.
	ErrBackupMissing = errors.New("Steam localconfig backup not found")
	// ErrSteamGuardMissing means the caller did not provide a running check, so
	// the transaction must not write.
	ErrSteamGuardMissing = errors.New("Steam running check is required")
	// ErrAlreadyPrepared means localconfig already contains this exact launch
	// option; a second preparation would create a duplicate session.
	ErrAlreadyPrepared = errors.New("Steam launch option is already prepared")
	// ErrSessionActive means restoration was attempted while a session is live.
	ErrSessionActive = errors.New("a Better RPT session is active")
)
