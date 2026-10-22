package diagnostics

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/srnoob2570/better-rpt/internal/config"
	"github.com/srnoob2570/better-rpt/internal/domain"
	"github.com/srnoob2570/better-rpt/internal/steam"
)

func (e *evaluator) checkSteam(ctx context.Context) {
	installation, err := e.steamInstall(ctx)
	if err != nil {
		var unsupported *steam.UnsupportedError
		if errors.As(err, &unsupported) {
			e.add(CheckSteam, StatusFail, "diag.steam.install.unsupported", []any{unsupported.Variant.String()},
				[]string{"variant=" + unsupported.Variant.String(), "root=" + unsupported.Root},
				"diag.steam.install.unsupportedNext", "docs/compatibility.md")
			e.add(CheckVariant, StatusFail, "diag.steam.variant.unsupported", []any{unsupported.Variant.String()},
				[]string{"variant=" + unsupported.Variant.String()}, "diag.steam.variant.next",
				"docs/compatibility.md")
		} else {
			e.add(CheckSteam, StatusFail, "diag.steam.install.missing", nil, []string{err.Error()},
				"diag.steam.install.next", "docs/compatibility.md", "docs/known-issues.md")
			e.add(CheckVariant, StatusSkip, "diag.steam.variant.skip", nil, nil, "")
		}
		e.add(CheckSteamRun, StatusSkip, "diag.steam.running.skip", nil, nil, "")
		e.add(CheckSpaceWar, StatusSkip, "diag.steam.spacewar.skip", nil, nil, "")
		e.add(CheckDispatcher, StatusSkip, "diag.steam.dispatcher.skip", nil, nil, "")
		e.checkBackups()
		e.add(CheckSteamPerm, StatusSkip, "diag.steam.permissions.skip", nil, nil, "")
		e.add(CheckOverlay, StatusSkip, "diag.steam.overlay.skip", nil, nil, "")
		e.add(CheckControls, StatusSkip, "diag.steam.controllers.skip", nil, nil, "")
		e.add(CheckWarnings, StatusSkip, "diag.steam.warnings.skip", nil, nil, "")
		return
	}

	root := e.steamRoot()
	e.add(CheckSteam, StatusOK, "", nil, []string{
		"variant=" + installation.Variant.String(),
		"root=" + root,
	}, "")
	e.add(CheckVariant, StatusOK, "", nil, []string{"variant=" + installation.Variant.String()}, "")

	processes, processErr := e.runningSteam(ctx)
	running := len(processes) > 0
	if processErr != nil {
		e.add(CheckSteamRun, StatusSkip, "diag.steam.running.error", nil, []string{processErr.Error()}, "")
	} else if running {
		evidence := []string{}
		for _, process := range processes {
			evidence = append(evidence, fmt.Sprintf("pid=%d name=%s", process.PID, process.Name))
		}
		e.add(CheckSteamRun, StatusOK, "diag.steam.running.open", nil, evidence, "")
	} else {
		e.add(CheckSteamRun, StatusOK, "diag.steam.running.closed", nil, nil, "")
	}

	catalog, catalogErr := e.steamCatalog(ctx)
	if catalogErr != nil {
		e.add(CheckSpaceWar, StatusFail, "diag.steam.spacewar.catalog", nil, []string{catalogErr.Error()},
			"diag.steam.spacewar.next", "docs/compatibility.md")
		e.add(CheckWarnings, StatusSkip, "diag.steam.warnings.skip", nil, nil, "")
	} else {
		spaceWar := false
		for _, game := range catalog.Games {
			if game.AppID == domain.SpaceWarAppID &&
				game.Status != domain.StatusMissing &&
				game.Status != domain.StatusError &&
				game.Status != domain.StatusIncompatible {
				spaceWar = true
				break
			}
		}
		if spaceWar {
			e.add(CheckSpaceWar, StatusOK, "", nil, []string{"appid=480"}, "")
		} else {
			e.add(CheckSpaceWar, StatusFail, "diag.steam.spacewar.missing", nil,
				[]string{"appid=480"}, "diag.steam.spacewar.next", "docs/compatibility.md")
		}
		e.checkWarnings(catalog)
	}

	e.checkDispatcher(ctx)
	e.checkBackups()
	e.checkPermissions(running)
	e.checkOverlay()
	e.checkControllers()
}

func (e *evaluator) checkDispatcher(ctx context.Context) {
	snapshot, err := e.setupSnapshot(ctx)
	if err != nil {
		if errors.Is(err, domain.ErrRoot) {
			e.add(CheckDispatcher, StatusSkip, "diag.steam.dispatcher.root", nil, nil,
				"diag.steam.dispatcher.rootNext", "docs/known-issues.md")
			return
		}
		e.add(CheckDispatcher, StatusWarn, "diag.steam.dispatcher.error", nil, []string{err.Error()},
			"diag.steam.dispatcher.next", "docs/compatibility.md")
		return
	}
	status := string(snapshot.Status)
	evidence := []string{"status=" + status}
	if snapshot.Current.Exists {
		evidence = append(evidence, "launch_option="+snapshot.Current.Value)
	}
	switch snapshot.Status {
	case config.SetupInstalled:
		if config.IsDispatcherMarker(snapshot.Current.Value) {
			e.add(CheckDispatcher, StatusOK, "", nil, evidence, "")
		} else {
			e.add(CheckDispatcher, StatusFail, "diag.steam.dispatcher.conflict", nil, evidence,
				"diag.steam.dispatcher.next", "docs/compatibility.md")
		}
	case config.SetupGuide:
		e.add(CheckDispatcher, StatusWarn, "diag.steam.dispatcher.guide", nil, evidence,
			"diag.steam.dispatcher.guideNext", "docs/compatibility.md")
	case config.SetupConflict:
		e.add(CheckDispatcher, StatusFail, "diag.steam.dispatcher.conflict", nil, evidence,
			"diag.steam.dispatcher.next", "docs/compatibility.md")
	case config.SetupRestored:
		e.add(CheckDispatcher, StatusWarn, "diag.steam.dispatcher.restored", nil, evidence,
			"diag.steam.dispatcher.next", "docs/compatibility.md")
	default:
		e.add(CheckDispatcher, StatusWarn, "diag.steam.dispatcher.missing", nil, evidence,
			"diag.steam.dispatcher.missingNext", "docs/compatibility.md")
	}
}

func (e *evaluator) checkBackups() {
	if e.deps.ListBackups == nil {
		e.add(CheckBackups, StatusSkip, "diag.steam.backups.skip", nil, nil, "")
		return
	}
	backups, err := e.deps.ListBackups(e.paths)
	if err != nil {
		e.add(CheckBackups, StatusWarn, "diag.steam.backups.error", nil, []string{err.Error()},
			"diag.steam.backups.next", "docs/known-issues.md")
		return
	}
	if len(backups) == 0 {
		e.add(CheckBackups, StatusWarn, "diag.steam.backups.none", nil, nil,
			"diag.steam.backups.next", "docs/compatibility.md")
		return
	}
	kinds := make([]string, 0, len(backups))
	for _, backup := range backups {
		kinds = append(kinds, string(backup.Meta.Kind))
	}
	e.add(CheckBackups, StatusOK, "", nil, []string{
		fmt.Sprintf("backups=%d", len(backups)),
		"kinds=" + strings.Join(kinds, ","),
	}, "")
}

func (e *evaluator) checkPermissions(running bool) {
	if e.deps.SelectAccount == nil {
		e.add(CheckSteamPerm, StatusSkip, "diag.steam.permissions.skip", nil, nil, "")
		return
	}
	root := e.steamRoot()
	account, err := e.deps.SelectAccount(root)
	if err != nil {
		e.add(CheckSteamPerm, StatusWarn, "diag.steam.permissions.account", nil, []string{err.Error()},
			"diag.steam.permissions.next", "docs/known-issues.md")
		return
	}
	files := []struct {
		name string
		path string
	}{
		{"localconfig.vdf", account.LocalConfigPath(root)},
		{"config.vdf", e.configVDF()},
	}
	evidence := []string{}
	failed := make([]string, 0)
	for _, file := range files {
		if file.path == "" {
			continue
		}
		info, err := e.deps.Stat(file.path)
		if err != nil {
			evidence = append(evidence, file.name+": "+err.Error())
			failed = append(failed, file.name)
			continue
		}
		writable := e.writable(file.path, info.IsDir())
		evidence = append(evidence, fmt.Sprintf("%s: exists=true writable=%t", file.name, writable))
		if !writable {
			failed = append(failed, file.name)
		}
	}
	if len(failed) == 0 {
		e.add(CheckSteamPerm, StatusOK, "", nil, evidence, "")
		return
	}
	next := "diag.steam.permissions.next"
	if running {
		next = "diag.steam.permissions.runningNext"
	}
	status := StatusFail
	if running {
		status = StatusWarn
	}
	e.add(CheckSteamPerm, status, "diag.steam.permissions.fail", []any{len(failed)}, evidence, next,
		"docs/appimage.md", "docs/known-issues.md")
}

func (e *evaluator) checkOverlay() {
	if e.deps.InspectOverlay == nil {
		e.add(CheckOverlay, StatusSkip, "diag.steam.overlay.skip", nil, nil, "")
		return
	}
	inspection, err := e.deps.InspectOverlay(e.steamRoot())
	if err != nil {
		if errors.Is(err, domain.ErrRoot) {
			e.add(CheckOverlay, StatusSkip, "diag.steam.overlay.root", nil, nil, "")
			return
		}
		e.add(CheckOverlay, StatusWarn, "diag.steam.overlay.error", nil, []string{err.Error()},
			"diag.steam.overlay.next", "docs/compatibility.md")
		return
	}
	if !inspection.Exists {
		e.add(CheckOverlay, StatusSkip, "diag.steam.overlay.missing", nil, nil, "")
		return
	}
	evidence := []string{fmt.Sprintf("OverlayAppEnable=%t", inspection.Enabled)}
	if inspection.Enabled {
		e.add(CheckOverlay, StatusOK, "", nil, evidence, "")
	} else {
		e.add(CheckOverlay, StatusWarn, "diag.steam.overlay.disabled", nil, evidence,
			"diag.steam.overlay.next", "docs/compatibility.md")
	}
}

func (e *evaluator) checkControllers() {
	if e.deps.InspectControls == nil {
		e.add(CheckControls, StatusSkip, "diag.steam.controllers.skip", nil, nil, "")
		return
	}
	inspection, err := e.deps.InspectControls(e.steamRoot())
	if err != nil {
		if errors.Is(err, domain.ErrRoot) {
			e.add(CheckControls, StatusSkip, "diag.steam.controllers.root", nil, nil, "")
			return
		}
		e.add(CheckControls, StatusWarn, "diag.steam.controllers.error", nil, []string{err.Error()},
			"diag.steam.controllers.next", "docs/compatibility.md")
		return
	}
	evidence := []string{
		fmt.Sprintf("missing=%d", len(inspection.Missing)),
		fmt.Sprintf("autosave=%d", len(inspection.Autosave)),
		fmt.Sprintf("already=%d", len(inspection.Already)),
		fmt.Sprintf("existing=%d", len(inspection.Existing)),
	}
	pending := len(inspection.Missing) + len(inspection.Autosave)
	if pending > 0 {
		if len(inspection.Missing) > 0 {
			evidence = append(evidence, "missing_sets="+strings.Join(inspection.Missing, ","))
		}
		if len(inspection.Autosave) > 0 {
			evidence = append(evidence, "autosave_sets="+strings.Join(inspection.Autosave, ","))
		}
		e.add(CheckControls, StatusWarn, "diag.steam.controllers.missing", []any{pending},
			evidence, "diag.steam.controllers.next", "docs/compatibility.md")
		return
	}
	if len(inspection.Warnings) > 0 {
		for _, warning := range inspection.Warnings {
			evidence = append(evidence, warning.Error())
		}
		e.add(CheckControls, StatusWarn, "diag.steam.controllers.error", nil, evidence,
			"diag.steam.controllers.next", "docs/compatibility.md")
		return
	}
	if len(inspection.Already) == 0 && len(inspection.Existing) == 0 {
		e.add(CheckControls, StatusSkip, "diag.steam.controllers.none", nil, evidence, "")
		return
	}
	e.add(CheckControls, StatusOK, "", nil, evidence, "")
}

func (e *evaluator) checkWarnings(catalog *steam.Catalog) {
	if len(catalog.Warnings) == 0 {
		e.add(CheckWarnings, StatusOK, "", nil, nil, "")
		return
	}
	evidence := make([]string, 0, len(catalog.Warnings))
	for _, warning := range catalog.Warnings {
		evidence = append(evidence, warning.Error())
	}
	e.add(CheckWarnings, StatusWarn, "diag.steam.warnings.warn", []any{len(catalog.Warnings)}, evidence,
		"diag.steam.warnings.next", "docs/compatibility.md", "docs/known-issues.md")
}
