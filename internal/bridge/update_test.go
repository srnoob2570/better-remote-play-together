package bridge

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/srnoob2570/better-rpt/internal/update"
)

type fakeUpdater struct {
	check      update.Check
	checkErr   error
	startup    update.Check
	startupErr error
	install    update.InstallResult
	installErr error
	restartErr error
	closed     bool
	restarted  bool
	progress   []update.Progress
}

func (f *fakeUpdater) Check(context.Context) (update.Check, error) {
	return f.check, f.checkErr
}

func (f *fakeUpdater) CheckStartup(context.Context) (update.Check, error) {
	return f.startup, f.startupErr
}

func (f *fakeUpdater) Install(_ context.Context, emit func(update.Progress)) (update.InstallResult, error) {
	for _, progress := range f.progress {
		emit(progress)
	}
	return f.install, f.installErr
}

func (f *fakeUpdater) Restart() error {
	if f.restartErr != nil {
		return f.restartErr
	}
	f.restarted = true
	return nil
}

func (f *fakeUpdater) Close() {
	f.closed = true
}

type quittingEmitter struct {
	emitted int
	quit    int
}

func (e *quittingEmitter) Emit(string, any) { e.emitted++ }
func (e *quittingEmitter) Quit()            { e.quit++ }

func TestCheckForUpdateMapsResult(t *testing.T) {
	checkedAt := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	fake := &fakeUpdater{check: update.Check{
		CurrentVersion: "0.2.0",
		LatestVersion:  "0.2.1",
		Tag:            "v0.2.1",
		ReleaseURL:     "https://example.test/release",
		Available:      true,
		CanInstall:     true,
		InstallKind:    update.KindAppImage,
		CheckedAt:      checkedAt,
	}}
	bridge := New(&fakeApp{}, "0.2.0", nil)
	bridge.updater = fake
	got, err := bridge.CheckForUpdate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.LatestVersion != "0.2.1" || got.InstallKind != "appimage" || !got.CanInstall {
		t.Fatalf("check = %+v", got)
	}
	if got.CheckedAt != "2026-10-21T12:00:00Z" {
		t.Fatalf("checkedAt = %q", got.CheckedAt)
	}
}

func TestCheckForUpdateOnStartupMapsCached(t *testing.T) {
	fake := &fakeUpdater{startup: update.Check{
		CurrentVersion: "0.2.0",
		LatestVersion:  "0.2.1",
		Available:      true,
		Cached:         true,
		InstallKind:    update.KindDeb,
	}}
	bridge := New(&fakeApp{}, "0.2.0", nil)
	bridge.updater = fake
	got, err := bridge.CheckForUpdateOnStartup(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !got.Cached || got.InstallKind != "deb" {
		t.Fatalf("check = %+v", got)
	}
}

func TestInstallUpdateEmitsProgressAndMapsResult(t *testing.T) {
	fake := &fakeUpdater{
		progress: []update.Progress{
			{Stage: update.StageDownloading, Downloaded: 10, Total: 20, Percent: 50},
			{Stage: update.StageInstalling, Percent: 100},
		},
		install: update.InstallResult{
			Status:      update.StatusReplaced,
			InstallKind: update.KindAppImage,
			Path:        "/home/test/BetterRPT.AppImage",
			Restart:     true,
		},
	}
	recorder := &eventRecorder{}
	bridge := New(&fakeApp{}, "0.2.0", recorder)
	bridge.updater = fake
	got, err := bridge.InstallUpdate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "replaced" || !got.Restart || got.InstallKind != "appimage" {
		t.Fatalf("result = %+v", got)
	}
	events := recorder.snapshot()
	if len(events) != 2 || events[0].name != EventUpdateProgress {
		t.Fatalf("events = %+v", events)
	}
	progress, ok := events[0].data.(UpdateProgress)
	if !ok || progress.Percent != 50 || progress.Stage != "downloading" {
		t.Fatalf("progress = %+v", events[0].data)
	}
}

func TestRestartToUpdateLaunchesAndQuits(t *testing.T) {
	fake := &fakeUpdater{}
	emitter := &quittingEmitter{}
	bridge := New(&fakeApp{}, "0.2.0", emitter)
	bridge.updater = fake
	if err := bridge.RestartToUpdate(); err != nil {
		t.Fatal(err)
	}
	if !fake.restarted || emitter.quit != 1 {
		t.Fatalf("restarted = %v, quit = %d", fake.restarted, emitter.quit)
	}
}

func TestRestartToUpdatePropagatesFailure(t *testing.T) {
	fake := &fakeUpdater{restartErr: update.ErrRestart}
	emitter := &quittingEmitter{}
	bridge := New(&fakeApp{}, "0.2.0", emitter)
	bridge.updater = fake
	if err := bridge.RestartToUpdate(); !errors.Is(err, update.ErrRestart) {
		t.Fatalf("error = %v", err)
	}
	if emitter.quit != 0 {
		t.Fatal("app quit after a failed restart")
	}
}

func TestShutdownClosesUpdater(t *testing.T) {
	fake := &fakeUpdater{}
	bridge := New(&fakeApp{}, "0.2.0", nil)
	bridge.updater = fake
	bridge.Shutdown()
	if !fake.closed {
		t.Fatal("updater was not closed")
	}
}
