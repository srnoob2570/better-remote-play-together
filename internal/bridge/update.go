package bridge

import (
	"context"

	"github.com/srnoob2570/better-rpt/internal/update"
)

// updater is the update package surface used by the bridge. Tests provide a
// fake; production uses *update.Manager.
type updater interface {
	Check(context.Context) (update.Check, error)
	CheckStartup(context.Context) (update.Check, error)
	Install(context.Context, func(update.Progress)) (update.InstallResult, error)
	Restart() error
	Close()
}

// Quitter is implemented by the Wails emitter. Tests can omit it.
type Quitter interface {
	Quit()
}

// CheckForUpdate always queries GitHub for the latest release.
func (b *Bridge) CheckForUpdate(ctx context.Context) (UpdateCheck, error) {
	check, err := b.updater.Check(contextOrBackground(ctx))
	if err != nil {
		return UpdateCheck{}, err
	}
	return mapUpdateCheck(check), nil
}

// CheckForUpdateOnStartup reuses the cached result for up to 24 hours.
func (b *Bridge) CheckForUpdateOnStartup(ctx context.Context) (UpdateCheck, error) {
	check, err := b.updater.CheckStartup(contextOrBackground(ctx))
	if err != nil {
		return UpdateCheck{}, err
	}
	return mapUpdateCheck(check), nil
}

// InstallUpdate downloads, verifies and installs the release found by the
// last check. The Wails call context is ignored so closing a dialog does not
// cancel an in-flight download; Shutdown owns cancellation.
func (b *Bridge) InstallUpdate(_ context.Context) (UpdateInstallResult, error) {
	result, err := b.updater.Install(context.Background(), func(progress update.Progress) {
		b.emit(EventUpdateProgress, mapUpdateProgress(progress))
	})
	if err != nil {
		return UpdateInstallResult{}, err
	}
	return mapUpdateInstallResult(result), nil
}

// RestartToUpdate launches the verified binary and asks Wails to quit.
func (b *Bridge) RestartToUpdate() error {
	if err := b.updater.Restart(); err != nil {
		return err
	}
	if quitter, ok := b.emitter.(Quitter); ok {
		quitter.Quit()
	}
	return nil
}
