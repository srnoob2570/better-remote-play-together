// Package update implements self-updates from the project's GitHub Releases.
// It supports the self-contained AppImage and the Debian package, verifies
// every download against SHA256SUMS and never requires a token.
package update

import "errors"

var (
	// ErrNetwork means the request could not reach GitHub.
	ErrNetwork = errors.New("update: network request failed")
	// ErrTimeout means the request exceeded its deadline.
	ErrTimeout = errors.New("update: network request timed out")
	// ErrRateLimited means GitHub rejected the unauthenticated request.
	ErrRateLimited = errors.New("update: GitHub rate limit reached")
	// ErrInvalidResponse means the release payload or a URL was unsafe.
	ErrInvalidResponse = errors.New("update: invalid release response")
	// ErrNoAsset means the release lacks a usable asset for this system.
	ErrNoAsset = errors.New("update: release asset not found")
	// ErrChecksum means SHA256SUMS is missing, malformed or does not match.
	ErrChecksum = errors.New("update: checksum verification failed")
	// ErrDownload means the asset could not be stored.
	ErrDownload = errors.New("update: download failed")
	// ErrTooLarge means the asset exceeds the size limit.
	ErrTooLarge = errors.New("update: download exceeds the size limit")
	// ErrInstall means the package manager or the file replacement failed.
	ErrInstall = errors.New("update: installation failed")
	// ErrNoPkexec means polkit's pkexec is not available.
	ErrNoPkexec = errors.New("update: pkexec is not available")
	// ErrCanceled means the user or the app canceled the operation.
	ErrCanceled = errors.New("update: operation canceled")
	// ErrBusy means another update operation is already running.
	ErrBusy = errors.New("update: another operation is in progress")
	// ErrNotChecked means there is no checked release to install.
	ErrNotChecked = errors.New("update: no update has been checked")
	// ErrNoUpdate means the last check found no newer release.
	ErrNoUpdate = errors.New("update: no update is available")
	// ErrUnsupported means this installation cannot be updated in place.
	ErrUnsupported = errors.New("update: automatic installation is not supported")
	// ErrNoRestart means no pending binary is available to relaunch.
	ErrNoRestart = errors.New("update: there is no pending restart")
	// ErrRestart means the new binary could not be started.
	ErrRestart = errors.New("update: could not restart the application")
)
