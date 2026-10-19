// Package ui is the Fyne dashboard. It only consumes the application service;
// Steam, localconfig and session mechanics stay outside this package.
package ui

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	fyneapp "fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	bapp "github.com/srnoob2570/better-rpt/internal/app"
	"github.com/srnoob2570/better-rpt/internal/domain"
	"github.com/srnoob2570/better-rpt/internal/i18n"
	"github.com/srnoob2570/better-rpt/internal/session"
	"github.com/srnoob2570/better-rpt/internal/steam"
)

type dashboard struct {
	svc     *bapp.Service
	logger  *slog.Logger
	version string
	app     fyne.App
	window  fyne.Window

	catalog    steam.Catalog
	visible    []domain.Game
	selectedID uint32
	query      string
	filter     filterKey
	loading    bool
	refreshing bool
	refreshErr error
	warnings   []string
	donorFound bool
	donorGame  domain.Game
	recovered  bool
	notice     string
	warned     map[uint32]bool
	lastError  error
	lastDetail string
	retry      func()

	preparing       bool
	prepStarted     time.Time
	prepCancel      context.CancelFunc
	cancelRequested bool
	prepCloseSteam  bool
	prepPhase       domain.Phase

	activeToken string
	activeGame  domain.Game
	activeState session.State
	activeStart time.Time
	pollCancel  context.CancelFunc

	pendingRestore string
	pendingGame    domain.Game
	endedState     session.State
	showEnded      bool
	stale          bool
	restoring      bool

	search        *widget.Entry
	clearSearch   *widget.Button
	filterSelect  *widget.Select
	list          *widget.List
	libraryStatus *widget.Label
	sidebarBox    *fyne.Container
	sidebarScroll *container.Scroll
	elapsedLabel  *widget.Label
	suppress      bool
}

// Run opens the dashboard and blocks until the window closes.
func Run(svc *bapp.Service, logger *slog.Logger, version string) {
	a := fyneapp.New()
	w := a.NewWindow(svc.T("app.title"))
	d := &dashboard{
		svc:     svc,
		logger:  logger,
		version: version,
		app:     a,
		window:  w,
		filter:  filterAll,
		loading: true,
		warned:  make(map[uint32]bool),
	}
	w.SetContent(d.buildContent())
	w.Resize(fyne.NewSize(1280, 800))
	w.CenterOnScreen()
	d.installShortcuts()
	a.Lifecycle().SetOnStarted(func() {
		go d.startup()
	})
	w.ShowAndRun()
}

func (d *dashboard) installShortcuts() {
	canvas := d.window.Canvas()
	canvas.AddShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyF, Modifier: fyne.KeyModifierControl}, func(fyne.Shortcut) {
		if d.search != nil {
			canvas.Focus(d.search)
		}
	})
	canvas.AddShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyF5}, func(fyne.Shortcut) {
		d.refreshCatalog(true)
	})
	canvas.AddShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyEscape}, func(fyne.Shortcut) {
		d.window.Canvas().Unfocus()
	})
}

func (d *dashboard) buildContent() fyne.CanvasObject {
	d.sidebarBox = container.NewVBox()
	header := d.buildHeader()
	library := d.buildLibrary()
	sidebar := container.NewVScroll(d.sidebarBox)
	sidebar.SetMinSize(fyne.NewSize(340, 0))
	d.sidebarScroll = sidebar
	d.rebuildSidebar()
	body := container.NewHSplit(library, sidebar)
	body.SetOffset(0.64)
	return container.NewBorder(header, nil, nil, nil, body)
}

func (d *dashboard) buildHeader() fyne.CanvasObject {
	title := canvas.NewText(d.svc.T("app.title"), theme.ForegroundColor())
	title.TextSize = 26
	title.TextStyle = fyne.TextStyle{Bold: true}
	subtitle := widget.NewLabel(d.svc.T("app.subtitle"))
	subtitle.Importance = widget.LowImportance
	left := container.NewVBox(title, subtitle)

	labels := []string{d.svc.T("language.es"), d.svc.T("language.en")}
	language := widget.NewSelect(labels, func(value string) {
		if d.suppress {
			return
		}
		code := i18n.Spanish
		if value == d.svc.T("language.en") {
			code = i18n.English
		}
		if code == d.svc.Language() {
			return
		}
		if err := d.svc.SetLanguage(string(code)); err != nil {
			d.setError(err, nil)
			return
		}
		d.window.SetContent(d.buildContent())
	})
	d.suppress = true
	if d.svc.Language() == i18n.English {
		language.SetSelected(d.svc.T("language.en"))
	} else {
		language.SetSelected(d.svc.T("language.es"))
	}
	d.suppress = false
	right := container.NewHBox(language)
	return container.NewBorder(nil, nil, nil, right, left)
}

func (d *dashboard) buildLibrary() fyne.CanvasObject {
	d.search = widget.NewEntry()
	d.search.SetPlaceHolder(d.svc.T("search.placeholder"))
	d.search.OnChanged = func(value string) {
		d.query = value
		if d.clearSearch != nil {
			if strings.TrimSpace(value) == "" {
				d.clearSearch.Disable()
			} else {
				d.clearSearch.Enable()
			}
		}
		d.applyFilter()
	}

	labels := make([]string, len(filterKeys))
	for index, key := range filterKeys {
		labels[index] = d.svc.T(filterLabelKey(key))
	}
	d.filterSelect = widget.NewSelect(labels, func(value string) {
		if d.suppress {
			return
		}
		for index, label := range labels {
			if value == label {
				d.filter = filterKeys[index]
				break
			}
		}
		d.applyFilter()
	})
	d.suppress = true
	d.filterSelect.SetSelected(d.svc.T(filterLabelKey(d.filter)))
	d.suppress = false

	refresh := widget.NewButton(d.svc.T("action.refresh"), func() {
		d.refreshCatalog(true)
	})
	d.clearSearch = widget.NewButton(d.svc.T("action.clearSearch"), func() {
		d.search.SetText("")
		d.query = ""
		d.applyFilter()
	})
	d.clearSearch.Disable()
	if d.query != "" {
		d.clearSearch.Enable()
		d.search.SetText(d.query)
	}
	toolbar := container.NewBorder(nil, nil, nil, container.NewHBox(d.filterSelect, d.clearSearch, refresh), d.search)

	d.list = widget.NewList(
		func() int { return len(d.visible) },
		func() fyne.CanvasObject { return newGameRow() },
		func(id widget.ListItemID, object fyne.CanvasObject) {
			if id < 0 || id >= len(d.visible) {
				return
			}
			row := object.(*gameRow)
			game := d.visible[id]
			row.SetGame(game, game.AppID == d.selectedID, d.svc.T)
		},
	)
	d.list.OnSelected = func(id widget.ListItemID) {
		if d.suppress || id < 0 || id >= len(d.visible) {
			return
		}
		d.selectedID = d.visible[id].AppID
		d.lastError = nil
		d.lastDetail = ""
		d.retry = nil
		d.rebuildSidebar()
		d.list.Refresh()
	}

	d.libraryStatus = widget.NewLabel("")
	d.libraryStatus.Wrapping = fyne.TextWrapWord
	d.libraryStatus.Importance = widget.LowImportance
	d.libraryStatus.Hide()

	library := container.NewBorder(toolbar, d.libraryStatus, nil, nil, d.list)
	d.updateLibraryStatus()
	d.applyFilter()
	return library
}

func (d *dashboard) updateLibraryStatus() {
	if d.libraryStatus == nil {
		return
	}
	text := ""
	switch {
	case d.loading:
		text = d.svc.T("library.loading")
	case d.refreshErr != nil:
		text = d.svc.T("error.refresh") + "\n" + d.refreshErr.Error()
	case len(d.visible) == 0:
		if d.query != "" || d.filter != filterAll {
			text = d.svc.T("search.empty")
		} else {
			text = d.svc.T("library.empty") + "\n" + d.svc.T("library.emptyHint")
		}
	}
	if text == "" {
		d.libraryStatus.Hide()
	} else {
		d.libraryStatus.SetText(text)
		d.libraryStatus.Show()
	}
	d.libraryStatus.Refresh()
}

func (d *dashboard) applyFilter() {
	if d.list == nil {
		return
	}
	d.visible = visibleGames(d.catalog.Games, d.query, d.filter)
	d.list.Refresh()
	if len(d.visible) == 0 {
		d.selectedID = 0
		d.suppress = true
		d.list.UnselectAll()
		d.suppress = false
		d.rebuildSidebar()
		d.updateLibraryStatus()
		return
	}
	for index, game := range d.visible {
		if game.AppID == d.selectedID {
			d.suppress = true
			d.list.Select(index)
			d.suppress = false
			d.rebuildSidebar()
			d.updateLibraryStatus()
			return
		}
	}
	d.selectedID = d.visible[0].AppID
	d.suppress = true
	d.list.Select(0)
	d.suppress = false
	d.rebuildSidebar()
	d.updateLibraryStatus()
}

func (d *dashboard) startup() {
	fyne.Do(func() { d.refreshCatalog(true) })
}

func (d *dashboard) refreshCatalog(recover bool) {
	if d.refreshing {
		return
	}
	d.refreshing = true
	if len(d.catalog.Games) == 0 {
		d.loading = true
	}
	d.refreshErr = nil
	d.updateLibraryStatus()
	go func() {
		catalog, err := d.svc.RefreshCatalog(context.Background())
		fyne.Do(func() {
			d.refreshing = false
			d.loading = false
			if err != nil {
				d.refreshErr = err
				d.setError(err, func() { d.refreshCatalog(recover) })
				d.applyFilter()
				return
			}
			d.catalog = catalog
			d.warnings = make([]string, 0, len(catalog.Warnings))
			for _, warning := range catalog.Warnings {
				d.warnings = append(d.warnings, warning.Error())
			}
			d.refreshErr = nil
			d.lastError = nil
			d.lastDetail = ""
			d.retry = nil
			if game, ok := findGame(catalog.Games, domain.SpaceWarAppID); ok {
				d.donorGame = game
				d.donorFound = true
			} else {
				d.donorGame = domain.Game{}
				d.donorFound = false
			}
			d.applyFilter()
			if recover && !d.recovered {
				d.recovered = true
				d.recoverSession()
			}
		})
	}()
}

func (d *dashboard) recoverSession() {
	go func() {
		recovery, err := d.svc.RecoverSession(context.Background())
		fyne.Do(func() {
			if err != nil {
				d.setError(err, func() { d.recoverSession() })
				return
			}
			if recovery == nil {
				return
			}
			if len(recovery.ExpiredPlans) > 0 {
				d.notice = fmt.Sprintf("%s (%d)", d.svc.T("error.session"), len(recovery.ExpiredPlans))
			}
			if recovery.State != nil && recovery.HelperAlive && recovery.State.Token != "" {
				state := *recovery.State
				d.activeToken = state.Token
				d.activeState = state
				d.activeGame = d.gameForID(state.TargetAppID)
				d.activeStart = state.StartedAt
				if d.activeStart.IsZero() {
					d.activeStart = time.Now()
				}
				d.startPoll(d.activeToken)
			} else if recovery.State != nil && recovery.State.Token != "" {
				state := *recovery.State
				d.pendingRestore = state.Token
				d.pendingGame = d.gameForID(state.TargetAppID)
				d.endedState = state
				d.showEnded = true
			}
			if recovery.Stale != nil {
				token := recovery.Stale.Backup.Meta.SessionToken
				if token == "" {
					token = recovery.State.Token
				}
				d.pendingRestore = token
				d.pendingGame = d.gameForID(recovery.State.TargetAppID)
				d.stale = true
				d.showStaleDialog()
			}
			d.rebuildSidebar()
		})
	}()
}

func (d *dashboard) showStaleDialog() {
	if d.pendingRestore == "" {
		dialog.ShowInformation(d.svc.T("stale.title"), d.svc.T("error.noBackup"), d.window)
		return
	}
	dialog.ShowConfirm(d.svc.T("stale.title"), d.svc.T("stale.body"), func(ok bool) {
		if ok {
			d.restoreSession(d.pendingRestore)
		}
	}, d.window)
}

func (d *dashboard) gameForID(appID uint32) domain.Game {
	if game, ok := findGame(d.catalog.Games, appID); ok {
		return game
	}
	return domain.Game{AppID: appID, Name: fmt.Sprintf("%s %d", d.svc.T("details.appid"), appID)}
}

func (d *dashboard) rebuildSidebar() {
	if d.sidebarBox == nil {
		return
	}
	d.elapsedLabel = nil
	d.sidebarBox.Objects = nil
	add := func(objects ...fyne.CanvasObject) {
		for _, object := range objects {
			d.sidebarBox.Add(object)
		}
	}

	if d.lastError != nil {
		add(widget.NewLabelWithStyle(d.svc.T("error.title"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
		add(wrapped(d.svc.T("error.body")))
		if d.lastDetail != "" {
			add(widget.NewAccordion(widget.NewAccordionItem(d.svc.T("error.detail"), wrapped(d.lastDetail))))
		}
		if d.retry != nil {
			button := widget.NewButton(d.svc.T("action.retry"), d.retry)
			button.Importance = widget.HighImportance
			add(button)
		}
		d.rebuildDonor(add)
		d.finishSidebar()
		return
	}

	if d.preparing {
		add(widget.NewLabelWithStyle(d.svc.T("prep.title"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
		step := prepStepForPhase(d.prepPhase)
		for index, key := range prepSteps {
			mark := "○"
			if index < step {
				mark = "✓"
			} else if index == step {
				mark = "◐"
			}
			add(wrapped(mark + "  " + d.svc.T(key)))
		}
		add(wrapped(d.svc.T("prep.cancelHint")))
		label := d.svc.T("action.cancel")
		if d.cancelRequested {
			label = d.svc.T("action.cancelRestore")
		}
		cancel := widget.NewButton(label, d.cancelPreparation)
		cancel.Importance = widget.HighImportance
		if d.prepPhase != domain.PhasePreparing {
			cancel.Disable()
			cancel.SetText(d.svc.T("session.starting"))
		}
		add(cancel)
		d.rebuildDonor(add)
		d.finishSidebar()
		return
	}

	if d.activeToken != "" {
		add(widget.NewLabelWithStyle(d.svc.T("session.title"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
		add(wrapped(d.gameName(d.activeGame)))
		d.elapsedLabel = widget.NewLabel("")
		d.elapsedLabel.Importance = widget.LowImportance
		add(d.elapsedLabel)
		d.updateElapsed()
		add(widget.NewSeparator())
		add(wrapped(d.svc.T("session.guide1")))
		add(wrapped(d.svc.T("session.guide2")))
		add(wrapped(d.svc.T("session.guide3")))
		add(wrapped(d.svc.T("session.locked")))
		button := widget.NewButton(d.svc.T("status.active"), nil)
		button.Disable()
		add(button)
		d.rebuildDonor(add)
		d.finishSidebar()
		return
	}

	if d.pendingRestore != "" && (d.showEnded || d.stale) {
		if d.stale && !d.showEnded {
			add(widget.NewLabelWithStyle(d.svc.T("stale.title"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
			add(wrapped(d.svc.T("stale.body")))
		} else {
			add(widget.NewLabelWithStyle(d.svc.T("ended.title"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
			add(wrapped(d.gameName(d.pendingGame)))
			if !d.endedState.StartedAt.IsZero() {
				end := d.endedState.EndedAt
				if end.IsZero() {
					end = time.Now()
				}
				add(wrapped(d.svc.T("ended.duration") + ": " + formatDuration(end.Sub(d.endedState.StartedAt))))
			}
			if d.endedState.Phase == domain.PhaseError && d.endedState.Error != "" {
				add(wrapped(d.svc.T("error.title") + ": " + d.endedState.Error))
			}
			add(wrapped(d.svc.T("ended.restoreHint")))
		}
		restore := widget.NewButton(d.svc.T("action.restore"), func() {
			d.confirmRestore(d.pendingRestore)
		})
		restore.Importance = widget.HighImportance
		add(restore)
		if d.showEnded {
			back := widget.NewButton(d.svc.T("action.back"), func() {
				d.showEnded = false
				d.stale = false
				d.rebuildSidebar()
			})
			back.Importance = widget.LowImportance
			add(back)
		}
		d.rebuildDonor(add)
		d.finishSidebar()
		return
	}

	game, ok := findGame(d.catalog.Games, d.selectedID)
	if !ok {
		add(wrapped(d.svc.T("primary.select")))
		if d.pendingRestore != "" {
			restore := widget.NewButton(d.svc.T("action.restore"), func() {
				d.confirmRestore(d.pendingRestore)
			})
			restore.Importance = widget.HighImportance
			add(restore)
		} else if !d.donorFound {
			addSpaceWar := widget.NewButton(d.svc.T("action.addSpacewar"), d.openSpaceWar)
			addSpaceWar.Importance = widget.HighImportance
			add(addSpaceWar)
			if len(d.catalog.Games) == 0 && !d.loading && d.refreshErr == nil {
				openSteam := widget.NewButton(d.svc.T("action.openSteam"), d.openSteam)
				openSteam.Importance = widget.LowImportance
				add(openSteam)
			}
		} else if len(d.catalog.Games) == 0 && !d.loading && d.refreshErr == nil {
			openSteam := widget.NewButton(d.svc.T("action.openSteam"), d.openSteam)
			openSteam.Importance = widget.HighImportance
			add(openSteam)
		}
		d.rebuildDonor(add)
		d.finishSidebar()
		return
	}

	add(container.NewBorder(nil, nil, nil, newStatusChip(d.svc.T(statusLabelKey(game.Status)), statusTone(game.Status)).obj,
		widget.NewLabelWithStyle(game.Name, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})))
	add(wrapped(d.svc.T(kindLabelKey(game.Kind))))
	if game.Problem != nil {
		add(wrapped(game.Problem.Error()))
	}
	details := container.NewVBox(
		wrapped(d.svc.T("details.appid")+": "+strconv.FormatUint(uint64(game.AppID), 10)),
		wrapped(d.svc.T("details.library")+": "+game.LibraryPath),
		wrapped(d.svc.T("details.kind")+": "+d.svc.T(kindLabelKey(game.Kind))),
		wrapped(d.svc.T("details.path")+": "+game.InstallDir),
	)
	if game.Launch != nil && game.Launch.ProtonName != "" {
		details.Add(wrapped(d.svc.T("details.proton") + ": " + game.Launch.ProtonName))
	}
	add(widget.NewAccordion(widget.NewAccordionItem(d.svc.T("details.title"), details)))
	add(widget.NewSeparator())
	d.rebuildDonor(add)
	if len(d.warnings) > 0 {
		content := container.NewVBox()
		for _, warning := range d.warnings {
			content.Add(wrapped(warning))
		}
		add(widget.NewAccordion(widget.NewAccordionItem(d.svc.T("library.warnings"), content)))
	}

	if !d.donorFound {
		button := widget.NewButton(d.svc.T("action.addSpacewar"), d.openSpaceWar)
		button.Importance = widget.HighImportance
		add(button)
		d.finishSidebar()
		return
	}
	if game.AppID == domain.SpaceWarAppID {
		button := widget.NewButton(d.svc.T("donor.self"), nil)
		button.Disable()
		add(button)
		d.finishSidebar()
		return
	}
	if game.Status == domain.StatusIncompatible || game.Status == domain.StatusMissing || game.Status == domain.StatusError {
		button := widget.NewButton(d.svc.T(statusLabelKey(game.Status)), nil)
		button.Disable()
		add(button)
		d.finishSidebar()
		return
	}
	if d.pendingRestore != "" {
		restore := widget.NewButton(d.svc.T("action.restore"), func() {
			d.confirmRestore(d.pendingRestore)
		})
		restore.Importance = widget.HighImportance
		add(restore)
		d.finishSidebar()
		return
	}
	button := widget.NewButton(d.svc.T("action.prepare"), d.prepareSelected)
	button.Importance = widget.HighImportance
	add(button)
	d.finishSidebar()
}

func (d *dashboard) rebuildDonor(add func(...fyne.CanvasObject)) {
	add(widget.NewLabelWithStyle(d.svc.T("donor.title"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	label, toneValue := d.svc.T("donor.ready"), toneSuccess
	switch {
	case !d.donorFound:
		label, toneValue = d.svc.T("donor.spacewarMissing"), toneError
	case d.activeToken != "":
		label, toneValue = d.svc.T("donor.active"), toneInfo
	case d.preparing:
		label, toneValue = d.svc.T("donor.preparing"), toneWarning
	case d.pendingRestore != "":
		label, toneValue = d.svc.T("donor.stale"), toneWarning
	}
	add(newStatusChip(label, toneValue).obj)
	if d.notice != "" {
		add(wrapped(d.notice))
	}
}

func (d *dashboard) finishSidebar() {
	d.sidebarBox.Refresh()
	if d.sidebarScroll != nil {
		d.sidebarScroll.Refresh()
	}
}

func (d *dashboard) gameName(game domain.Game) string {
	if game.Name != "" {
		return game.Name
	}
	return fmt.Sprintf("%s %d", d.svc.T("details.appid"), game.AppID)
}

func (d *dashboard) updateElapsed() {
	if d.elapsedLabel == nil || d.activeStart.IsZero() {
		return
	}
	d.elapsedLabel.SetText(d.svc.T("session.elapsed") + ": " + formatDuration(time.Since(d.activeStart)))
}

func (d *dashboard) prepareSelected() {
	game, ok := findGame(d.catalog.Games, d.selectedID)
	if !ok || game.AppID == domain.SpaceWarAppID || game.Status == domain.StatusIncompatible || game.Status == domain.StatusMissing || game.Status == domain.StatusError {
		return
	}
	if game.Status == domain.StatusUnknown && !d.warned[game.AppID] {
		dialog.ShowConfirm(d.svc.T("dialog.unknown.title"), d.svc.T("dialog.unknown.body"), func(ok bool) {
			if !ok {
				return
			}
			d.warned[game.AppID] = true
			d.startSession(false)
		}, d.window)
		return
	}
	d.startSession(false)
}

func (d *dashboard) startSession(closeSteam bool) {
	if d.preparing || d.activeToken != "" {
		return
	}
	game, ok := findGame(d.catalog.Games, d.selectedID)
	if !ok || game.AppID == domain.SpaceWarAppID {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	d.preparing = true
	d.prepStarted = time.Now()
	d.prepCancel = cancel
	d.prepCloseSteam = closeSteam
	d.prepPhase = domain.PhasePreparing
	d.cancelRequested = false
	d.lastError = nil
	d.lastDetail = ""
	d.retry = nil
	d.rebuildSidebar()
	go func() {
		active, err := d.svc.StartSession(ctx, session.StartOptions{
			TargetAppID: game.AppID,
			CloseSteam:  closeSteam,
			OnState: func(state session.State) {
				fyne.Do(func() {
					if !d.preparing {
						return
					}
					d.prepPhase = state.Phase
					d.rebuildSidebar()
				})
			},
		})
		fyne.Do(func() {
			d.preparing = false
			d.prepCancel = nil
			d.cancelRequested = false
			if err != nil {
				if errors.Is(err, domain.ErrSteamRunning) && !closeSteam {
					d.confirmCloseSteam(game.AppID)
					return
				}
				if errors.Is(err, context.Canceled) {
					d.rebuildSidebar()
					return
				}
				d.setError(err, func() { d.startSession(closeSteam) })
				return
			}
			d.activeToken = active.Token
			d.activeGame = active.Game
			d.activeState = active.State
			d.activeStart = active.State.StartedAt
			if d.activeStart.IsZero() {
				d.activeStart = time.Now()
			}
			d.pendingRestore = ""
			d.pendingGame = domain.Game{}
			d.endedState = session.State{}
			d.showEnded = false
			d.stale = false
			d.lastError = nil
			d.rebuildSidebar()
			d.startPoll(active.Token)
		})
	}()
}

func (d *dashboard) confirmCloseSteam(appID uint32) {
	dialog.ShowConfirm(d.svc.T("dialog.closeSteam.title"), d.svc.T("dialog.closeSteam.body"), func(ok bool) {
		if !ok {
			d.rebuildSidebar()
			return
		}
		d.startSession(true)
	}, d.window)
}

func (d *dashboard) cancelPreparation() {
	if !d.preparing || d.prepCancel == nil || d.prepPhase != domain.PhasePreparing {
		return
	}
	d.cancelRequested = true
	d.prepCancel()
	d.rebuildSidebar()
}

func (d *dashboard) startPoll(token string) {
	if d.pollCancel != nil {
		d.pollCancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	d.pollCancel = cancel
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				state, err := d.svc.SessionState(token)
				fyne.Do(func() {
					if d.activeToken != token {
						return
					}
					if err != nil {
						d.finishSession(token, session.State{})
						return
					}
					d.activeState = state
					if state.StartedAt != (time.Time{}) {
						d.activeStart = state.StartedAt
					}
					if state.Terminal() {
						d.finishSession(token, state)
						return
					}
					d.updateElapsed()
				})
			}
		}
	}()
}

func (d *dashboard) finishSession(token string, state session.State) {
	if d.pollCancel != nil {
		d.pollCancel()
		d.pollCancel = nil
	}
	game := d.activeGame
	if state.TargetAppID != 0 {
		game = d.gameForID(state.TargetAppID)
	}
	d.activeToken = ""
	d.pendingRestore = token
	d.pendingGame = game
	d.endedState = state
	d.showEnded = true
	d.stale = false
	d.rebuildSidebar()
}

func (d *dashboard) confirmRestore(token string) {
	if token == "" || d.restoring {
		return
	}
	dialog.ShowConfirm(d.svc.T("dialog.restore.title"), d.svc.T("dialog.restore.body"), func(ok bool) {
		if ok {
			d.restoreSession(token)
		}
	}, d.window)
}

func (d *dashboard) restoreSession(token string) {
	if token == "" || d.restoring {
		return
	}
	d.restoring = true
	d.lastError = nil
	d.rebuildSidebar()
	go func() {
		err := d.svc.RestoreSession(context.Background(), token)
		fyne.Do(func() {
			d.restoring = false
			if err != nil {
				d.setError(err, func() { d.restoreSession(token) })
				return
			}
			d.pendingRestore = ""
			d.pendingGame = domain.Game{}
			d.endedState = session.State{}
			d.showEnded = false
			d.stale = false
			d.lastError = nil
			d.rebuildSidebar()
			dialog.ShowInformation(d.svc.T("restore.done.title"), d.svc.T("restore.done.body"), d.window)
		})
	}()
}

func (d *dashboard) openSpaceWar() {
	dialog.ShowInformation(d.svc.T("dialog.addSpacewar.title"), d.svc.T("dialog.addSpacewar.body"), d.window)
	d.openURL("steam://install/480")
}

func (d *dashboard) openSteam() {
	d.openURL("steam://open/main")
}

func (d *dashboard) openURL(target string) {
	if err := exec.Command("xdg-open", target).Start(); err != nil {
		d.setError(err, nil)
	}
}

func (d *dashboard) setError(err error, retry func()) {
	d.lastError = err
	d.lastDetail = ""
	if err != nil {
		d.lastDetail = err.Error()
	}
	d.retry = retry
	d.rebuildSidebar()
}

func wrapped(text string) *widget.Label {
	label := widget.NewLabel(text)
	label.Wrapping = fyne.TextWrapWord
	return label
}
