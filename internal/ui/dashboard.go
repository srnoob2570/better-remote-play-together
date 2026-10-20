// Package ui is the Fyne dashboard. It only consumes the application service;
// Steam, localconfig and session mechanics stay outside this package.
package ui

import (
	"context"
	"errors"
	"fmt"
	"image/color"
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
	"github.com/srnoob2570/better-rpt/internal/config"
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
	prepPhase       domain.Phase

	activeToken string
	activeGame  domain.Game
	activeState session.State
	activeStart time.Time
	pollCancel  context.CancelFunc

	pendingRestore string
	pendingGame    domain.Game
	restoring      bool
	restoringSetup bool
	setup          config.SetupSnapshot
	situation      session.Situation
	last           *session.LastLaunch

	search        *widget.Entry
	clearSearch   *actionButton
	filterTabs    *tabGroup
	refreshButton *actionButton
	countLabel    *widget.Label
	list          *widget.List
	empty         *emptyState
	bannerBox     *fyne.Container
	libraryTop    *fyne.Container
	listArea      *fyne.Container
	listTable     *fyne.Container
	searchRow     *fyne.Container
	sidebarBox    *fyne.Container
	sidebarScroll *container.Scroll
	sessionTabs   *tabGroup
	sessionHeader *fyne.Container
	sessionHint   *widget.Label
	sessionTab    int
	elapsedLabel  *canvas.Text
	suppress      bool
}

// Run opens the dashboard and blocks until the window closes.
func Run(svc *bapp.Service, logger *slog.Logger, version string) {
	a := fyneapp.New()
	a.Settings().SetTheme(newControlTheme())
	a.SetIcon(appIcon)
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
	d.empty = newEmptyState()
	d.sidebarBox = vbox(14)
	header := d.buildHeader()
	library := d.buildLibrary()

	content := container.NewVScroll(pad(d.sidebarBox, 0, 12, 24, 24))
	content.SetMinSize(fyne.NewSize(260, 0))
	d.sidebarScroll = content

	d.sessionTabs = newSessionTabGroup(
		[]string{d.svc.T("session.section"), d.svc.T("details.title"), d.svc.T("session.tabGuide")},
		d.sessionTab,
		func(index int) {
			if index == d.sessionTab {
				return
			}
			d.sessionTab = index
			d.rebuildSidebar()
		},
	)
	d.sessionHint = captionLabel("")
	d.sessionHint.Alignment = fyne.TextAlignTrailing
	d.sessionHint.Wrapping = fyne.TextWrapWord
	d.sessionHint.Importance = widget.WarningImportance
	d.sessionHint.Hide()
	d.sessionHeader = container.New(
		&sessionHeaderLayout{gap: 24},
		d.sessionTabs.obj,
		d.sessionHint,
	)
	d.rebuildSidebar()

	tabs := pad(d.sessionHeader, 24, 12, 24, 24)
	panel := container.NewStack(
		canvas.NewRectangle(colorSessionField),
		container.NewBorder(tabs, nil, nil, nil, content),
	)

	body := container.New(
		&twoPaneLayout{panelRatio: 0.40, minPanel: 400},
		pad(library, 18, 18, 26, 0),
		panel,
	)
	root := container.NewBorder(header, nil, nil, nil, body)
	return container.NewStack(canvas.NewRectangle(colorInk), root)
}

func (d *dashboard) buildHeader() fyne.CanvasObject {
	title := canvas.NewText(d.svc.T("app.title"), colorText)
	title.TextSize = 18
	title.TextStyle = fyne.TextStyle{Bold: true}
	subtitle := canvas.NewText(d.svc.T("app.subtitle"), colorMuted)
	subtitle.TextSize = 12
	wordmark := vbox(1, title, subtitle)

	logo := canvas.NewImageFromResource(appIconSmall)
	logo.FillMode = canvas.ImageFillContain
	logo.CornerRadius = 7
	logo.SetMinSize(fyne.NewSize(28, 28))
	logoBox := container.NewCenter(container.NewGridWrap(fyne.NewSize(28, 28), logo))
	left := hbox(12, logoBox, wordmark)

	labels := []string{d.svc.T("language.es"), d.svc.T("language.en")}
	selected := 0
	if d.svc.Language() == i18n.English {
		selected = 1
	}
	language := newTabGroup(labels, selected, func(index int) {
		if d.suppress {
			return
		}
		code := i18n.Spanish
		if index == 1 {
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

	right := hbox(22, versionLabel(d.version), language.obj)
	inner := container.NewBorder(nil, nil, left, container.NewCenter(right), nil)
	return vbox(0, tall(pad(inner, 2, 2, 26, 22), 42), widget.NewSeparator())
}

func versionLabel(version string) fyne.CanvasObject {
	if version == "" || version == "dev" {
		return vSpace(0)
	}
	return captionLabel("v" + version)
}

func (d *dashboard) buildLibrary() fyne.CanvasObject {
	d.search = widget.NewEntry()
	d.search.SetPlaceHolder(d.svc.T("search.placeholder"))
	if d.query != "" {
		d.search.SetText(d.query)
	}
	d.search.OnChanged = func(value string) {
		d.query = value
		d.updateClearSearch()
		d.applyFilter()
	}

	d.clearSearch = newActionButton(d.svc.T("action.clear"), func() {
		d.search.SetText("")
	})
	d.clearSearch.Importance = widget.LowImportance
	d.searchRow = container.NewBorder(nil, nil, nil, container.NewCenter(d.clearSearch), tall(d.search, 42))
	d.updateClearSearch()

	labels := make([]string, len(filterKeys))
	for index, key := range filterKeys {
		labels[index] = d.svc.T(filterLabelKey(key))
	}
	d.filterTabs = newTabGroup(labels, filterIndex(d.filter), func(index int) {
		if d.suppress {
			return
		}
		d.filter = filterKeys[index]
		d.applyFilter()
	})

	d.refreshButton = newActionButtonWithIcon(d.svc.T("action.refresh"), theme.ViewRefreshIcon(), func() {
		d.refreshCatalog(true)
	})
	d.refreshButton.Importance = widget.LowImportance
	if d.refreshing {
		d.setRefreshButtonBusy(true)
	}
	refresh := tall(container.NewThemeOverride(
		d.refreshButton,
		&compactButtonTheme{Theme: fyne.CurrentApp().Settings().Theme()},
	), 36)

	d.countLabel = captionLabel("")
	libraryTitle := container.NewThemeOverride(
		headingLabel(d.svc.T("library.title")),
		&listPaddingTheme{Theme: fyne.CurrentApp().Settings().Theme()},
	)
	libraryHeader := container.NewBorder(nil, nil, libraryTitle, container.NewCenter(d.countLabel), nil)
	filterRow := container.NewBorder(nil, nil, nil, container.NewCenter(refresh), d.filterTabs.obj)
	toolbar := vbox(14, libraryHeader, d.searchRow, filterRow)

	d.list = widget.NewList(
		func() int { return len(d.visible) },
		func() fyne.CanvasObject { return newGameRow() },
		func(id widget.ListItemID, object fyne.CanvasObject) {
			if id < 0 || id >= len(d.visible) {
				return
			}
			row := object.(*gameRow)
			game := d.visible[id]
			row.SetGame(game, game.AppID == d.selectedID, d.svc.T, steamIconPath(game))
		},
	)
	d.list.HideSeparators = true
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

	d.bannerBox = vbox(0)
	d.bannerBox.Hide()
	d.libraryTop = vbox(14, toolbar, d.bannerBox)
	header := container.NewThemeOverride(
		vbox(0, vSpace(8), tall(newLibraryTableHeader(d.svc.T), 28)),
		&listPaddingTheme{Theme: fyne.CurrentApp().Settings().Theme()},
	)
	listView := container.NewThemeOverride(d.list, &listPaddingTheme{Theme: fyne.CurrentApp().Settings().Theme()})
	d.listTable = container.New(&listTableLayout{gap: 4}, header, listView)
	d.listArea = container.NewStack(d.listTable, d.empty.obj)
	library := container.NewBorder(d.libraryTop, nil, nil, nil, d.listArea)
	d.applyFilter()
	return pad(library, 0, 0, 0, 18)
}

// listPaddingTheme tightens the List's built-in separator gap. The global
// theme keeps 12px for widgets, but 12px between table rows made the selected
// row look inset at the top and flush at the bottom.
type listPaddingTheme struct {
	fyne.Theme
}

// compactButtonTheme tightens a text action's icon gap and label inset. The
// stock button adds Padding plus the RichText inner inset between icon and
// label, which reads as a hole in a small toolbar action.
type compactButtonTheme struct {
	fyne.Theme
}

func (t compactButtonTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNamePadding:
		return 4
	case theme.SizeNameInnerPadding:
		return 4
	}
	return t.Theme.Size(name)
}

func (t listPaddingTheme) Size(name fyne.ThemeSizeName) float32 {
	if name == theme.SizeNamePadding {
		return 4
	}
	if name == theme.SizeNameInnerPadding {
		return 0
	}
	return t.Theme.Size(name)
}

func (t listPaddingTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	if name == theme.ColorNameSelection || name == theme.ColorNameHover {
		return color.Transparent
	}
	return t.Theme.Color(name, variant)
}

func (d *dashboard) updateClearSearch() {
	if d.clearSearch == nil {
		return
	}
	if strings.TrimSpace(d.query) == "" {
		d.clearSearch.Hide()
	} else {
		d.clearSearch.Show()
	}
	if d.searchRow != nil {
		d.searchRow.Refresh()
	}
}

func (d *dashboard) updateLibraryStatus() {
	if d.list == nil || d.countLabel == nil || d.empty == nil {
		return
	}
	count := len(d.visible)
	if count == 1 {
		d.countLabel.SetText(d.svc.T("library.countOne"))
	} else {
		d.countLabel.SetText(fmt.Sprintf(d.svc.T("library.count"), count))
	}
	d.countLabel.Refresh()

	if d.refreshErr != nil && len(d.visible) > 0 {
		d.showLibraryBanner()
	} else {
		d.hideLibraryBanner()
	}

	showList := false
	switch {
	case d.loading && len(d.catalog.Games) == 0:
		d.empty.Set(d.svc.T("library.loading"), "", "", nil)
		d.empty.Show()
	case d.refreshErr != nil && len(d.visible) == 0:
		d.empty.Set(d.svc.T("error.refresh"), d.refreshErr.Error(), d.svc.T("action.retry"), func() {
			d.refreshCatalog(true)
		})
		d.empty.Show()
	case len(d.visible) == 0 && (d.query != "" || d.filter != filterAll):
		d.empty.Set(d.svc.T("search.empty"), "", d.svc.T("action.resetFilters"), d.resetFilters)
		d.empty.Show()
	case len(d.visible) == 0:
		d.empty.Set(d.svc.T("library.empty"), d.svc.T("library.emptyHint"), d.svc.T("action.refresh"), func() {
			d.refreshCatalog(true)
		})
		d.empty.SetSecondary(d.svc.T("action.openSteam"), d.openSteam)
		d.empty.Show()
	default:
		showList = true
		d.empty.Hide()
	}
	if showList {
		if d.listTable != nil {
			d.listTable.Show()
		}
		d.list.Show()
	} else {
		if d.listTable != nil {
			d.listTable.Hide()
		}
		d.list.Hide()
	}
	if d.listArea != nil {
		d.listArea.Refresh()
	}
}

func (d *dashboard) showLibraryBanner() {
	if d.bannerBox == nil {
		return
	}
	retry := newActionButton(d.svc.T("action.retry"), func() {
		d.refreshCatalog(true)
	})
	retry.Importance = widget.LowImportance
	d.bannerBox.Objects = []fyne.CanvasObject{
		newCallout(d.svc.T("error.refresh"), d.refreshErr.Error(), toneError, retry).obj,
	}
	d.bannerBox.Show()
	d.bannerBox.Refresh()
	if d.libraryTop != nil {
		d.libraryTop.Refresh()
	}
}

func (d *dashboard) hideLibraryBanner() {
	if d.bannerBox == nil {
		return
	}
	if d.bannerBox.Visible() {
		d.bannerBox.Objects = nil
		d.bannerBox.Hide()
		d.bannerBox.Refresh()
		if d.libraryTop != nil {
			d.libraryTop.Refresh()
		}
	}
}

func (d *dashboard) resetFilters() {
	d.suppress = true
	d.query = ""
	d.filter = filterAll
	if d.search != nil {
		d.search.SetText("")
	}
	if d.filterTabs != nil {
		d.filterTabs.selectTab(filterIndex(filterAll))
	}
	d.suppress = false
	d.updateClearSearch()
	d.applyFilter()
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
	index := 0
	for position, game := range d.visible {
		if game.AppID == d.selectedID {
			index = position
			break
		}
	}
	d.selectedID = d.visible[index].AppID
	d.suppress = true
	d.list.Select(index)
	d.list.ScrollTo(index)
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
	d.setRefreshButtonBusy(true)
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
			d.setRefreshButtonBusy(false)
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

func (d *dashboard) setRefreshButtonBusy(busy bool) {
	if d.refreshButton == nil {
		return
	}
	if busy {
		d.refreshButton.SetText(d.svc.T("action.refreshing"))
		d.refreshButton.Disable()
		return
	}
	d.refreshButton.SetText(d.svc.T("action.refresh"))
	d.refreshButton.Enable()
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
				d.refreshSituation()
				return
			}
			d.applyRecovery(recovery)
			d.rebuildSidebar()
		})
	}()
}

func (d *dashboard) applyRecovery(recovery *session.Recovery) {
	d.situation = recovery.Situation
	d.last = recovery.Last
	d.setup = recovery.Setup
	if recovery.State != nil && recovery.HelperAlive {
		state := *recovery.State
		d.activeToken = state.Token
		d.activeState = state
		d.activeGame = d.gameForID(state.TargetAppID)
		d.activeStart = state.StartedAt
		if d.activeStart.IsZero() {
			d.activeStart = time.Now()
		}
		d.startPoll(d.activeToken)
		return
	}
	if recovery.Situation.Kind == session.SituationRestorePending ||
		recovery.Situation.Kind == session.SituationOrphan ||
		recovery.Situation.Kind == session.SituationConflict {
		d.pendingRestore = recovery.Situation.Token
		d.pendingGame = d.gameForID(recovery.Situation.TargetAppID)
	} else {
		d.pendingRestore = ""
		d.pendingGame = domain.Game{}
	}
}

// refreshSituation re-reads classification and last-launch state after an
// action, without re-running recovery side effects.
func (d *dashboard) refreshSituation() {
	situation, err := d.svc.SessionSituation()
	if err != nil {
		d.setError(err, func() { d.refreshSituation() })
		return
	}
	setup, _ := d.svc.SetupSnapshot()
	d.situation = situation
	d.last = situation.Last
	d.setup = setup
	if situation.Kind == session.SituationRestorePending ||
		situation.Kind == session.SituationOrphan ||
		situation.Kind == session.SituationConflict {
		d.pendingRestore = situation.Token
		d.pendingGame = d.gameForID(situation.TargetAppID)
	} else {
		d.pendingRestore = ""
		d.pendingGame = domain.Game{}
	}
	d.rebuildSidebar()
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
	d.updateSessionHint()

	if d.sessionTab == 1 {
		d.sidebarDetails()
		d.finishSidebar()
		return
	}
	if d.sessionTab == 2 {
		d.sidebarGuide()
		d.finishSidebar()
		return
	}

	switch {
	case d.lastError != nil:
		d.sidebarError()
	case d.preparing:
		d.sidebarPreparing()
	case d.activeToken != "":
		d.sidebarActive()
	case d.last != nil:
		d.sidebarLast()
	default:
		d.sidebarSelection()
	}
	d.finishSidebar()
}

func (d *dashboard) add(objects ...fyne.CanvasObject) {
	for _, object := range objects {
		d.sidebarBox.Add(object)
	}
}

func (d *dashboard) sidebarError() {
	var extra []fyne.CanvasObject
	if d.lastDetail != "" {
		extra = append(extra, newDisclosure(
			d.svc.T("error.detail"),
			pathLabel(d.lastDetail),
		))
	}
	d.add(newCallout(d.svc.T("error.title"), d.svc.T("error.body"), toneError, extra...).obj)
	d.sidebarNotice()
	if d.retry != nil {
		d.add(tall(primaryButton(d.svc.T("action.retry"), d.retry), 48))
	}
	d.addPendingRestore()
}

func (d *dashboard) sidebarPreparing() {
	d.add(headingLabel(d.svc.T("prep.title")))
	if game, ok := d.selectedGame(); ok {
		d.add(captionWrapLabel(d.gameName(game)))
	}
	step := prepStepForPhase(d.prepPhase)
	if step > len(prepSteps)-1 {
		step = len(prepSteps) - 1
	}
	d.add(captionLabel(fmt.Sprintf(d.svc.T("prep.step"), step+1, len(prepSteps))))

	rows := make([]fyne.CanvasObject, 0, len(prepSteps))
	for index, key := range prepSteps {
		rows = append(rows, d.prepStepRow(index, step, d.svc.T(key)))
	}
	d.add(vbox(10, rows...))
	d.sidebarNotice()

	label := d.svc.T("action.cancel")
	if d.cancelRequested {
		label = d.svc.T("action.cancelRestore")
	}
	cancel := newActionButton(label, d.cancelPreparation)
	if d.prepPhase != domain.PhasePreparing {
		cancel.Disable()
		cancel.SetText(d.svc.T("session.starting"))
	}
	d.add(captionWrapLabel(d.svc.T("prep.cancelHint")), tall(cancel, 48))
}

func (d *dashboard) prepStepRow(index, current int, text string) fyne.CanvasObject {
	state := toneNeutral
	label := bodyLabel(text)
	label.Importance = widget.LowImportance
	switch {
	case index < current:
		state = toneSuccess
	case index == current:
		state = toneWarning
		label.Importance = widget.MediumImportance
		label.TextStyle = fyne.TextStyle{Bold: true}
	}
	return numberedRow(index+1, label, state)
}

func numberedRow(number int, label *widget.Label, state tone) fyne.CanvasObject {
	index := canvas.NewText(strconv.Itoa(number), toneColor(state))
	index.TextSize = 12
	index.TextStyle = fyne.TextStyle{Bold: true}
	indexBox := container.NewCenter(container.NewGridWrap(fyne.NewSize(22, 18), index))
	return container.NewBorder(nil, nil, indexBox, nil, label)
}

func (d *dashboard) sidebarActive() {
	d.add(headingLabel(d.svc.T("session.title")))
	d.add(bodyLabel(d.gameName(d.activeGame)))

	timer := canvas.NewText("00:00", colorText)
	timer.TextSize = 30
	timer.TextStyle = fyne.TextStyle{Monospace: true}
	d.elapsedLabel = timer
	d.updateElapsed()
	d.add(vbox(4, captionLabel(d.svc.T("session.elapsed")), timer))

	d.add(widget.NewSeparator())
	d.add(sectionLabel(d.svc.T("session.guide")))
	guide := []string{"session.guide1", "session.guide2", "session.guide3"}
	rows := make([]fyne.CanvasObject, 0, len(guide))
	for index, key := range guide {
		rows = append(rows, numberedRow(index+1, bodyLabel(d.svc.T(key)), toneInfo))
	}
	d.add(vbox(10, rows...))
	d.add(captionWrapLabel(d.svc.T("session.locked")))
	d.sidebarNotice()

	active := primaryButton(d.svc.T("status.active"), nil)
	active.Disable()
	d.add(captionWrapLabel(d.svc.T("session.actionHint")), tall(active, 48))
}

// addPendingRestore offers the pending restore without blocking a new
// preparation. A new session supersedes the stale transaction and keeps the
// original launch-option value.
func (d *dashboard) addPendingRestore() {
	d.addPendingCallout()
}

func (d *dashboard) sidebarLast() {
	if d.last == nil {
		d.sidebarSelection()
		return
	}
	game, found := findGame(d.catalog.Games, d.last.AppID)
	if !found {
		game = d.gameForID(d.last.AppID)
	}
	d.add(headingLabel(d.svc.T("last.title")))
	d.add(d.gameCover(game))
	d.add(d.gameInfo(game))

	extra := make([]fyne.CanvasObject, 0, 5)
	extra = append(extra, captionLabel(d.svc.T(lastSourceKey(d.last.Source))))
	extra = append(extra, statusLabel(d.svc.T(lastOutcomeKey(d.last.Outcome)), outcomeTone(d.last.Outcome)))
	if !d.last.StartedAt.IsZero() {
		extra = append(extra, monoLabel(d.svc.T("last.started")+": "+d.last.StartedAt.Local().Format("2006-01-02 15:04")))
	}
	if !d.last.EndedAt.IsZero() && !d.last.StartedAt.IsZero() {
		extra = append(extra, monoLabel(d.svc.T("last.duration")+": "+formatDuration(d.last.EndedAt.Sub(d.last.StartedAt))))
	}
	if d.last.Error != "" {
		problem := captionWrapLabel(d.svc.T("error.title") + ": " + d.last.Error)
		problem.Importance = widget.WarningImportance
		extra = append(extra, problem)
	}

	installable := found && game.Status != domain.StatusIncompatible &&
		game.Status != domain.StatusMissing && game.Status != domain.StatusError &&
		game.Status != domain.StatusActive
	if d.situation.Setup == session.SetupInstalled && installable {
		d.add(tall(primaryButton(d.svc.T("action.playAgain"), d.replaySession), 48))
	} else {
		reason := d.svc.T("last.setupMissing")
		if !installable {
			reason = d.svc.T(statusLabelKey(game.Status))
		}
		disabled := primaryButton(d.svc.T("action.playAgain"), nil)
		disabled.Disable()
		d.add(captionWrapLabel(reason), tall(disabled, 48))
	}
	d.addRestoreAction()
	viewGuide := newActionButton(d.svc.T("action.viewGuide"), d.openGuide)
	viewGuide.Importance = widget.LowImportance
	d.add(tall(viewGuide, 40))
	d.add(captionWrapLabel(d.svc.T("last.hint")))
	d.add(newCallout(d.svc.T(situationLabelKey(d.situation.Kind)), "", situationTone(d.situation.Kind), extra...).obj)
	d.addPendingCallout()
	d.sidebarNotice()

	if game, ok := d.selectedGame(); ok && game.AppID != d.last.AppID {
		d.add(widget.NewSeparator())
		d.sidebarSelection()
	}
}

func outcomeTone(outcome session.LaunchOutcome) tone {
	switch outcome {
	case session.LaunchOK:
		return toneSuccess
	case session.LaunchPending, session.LaunchTimeout:
		return toneWarning
	case session.LaunchError:
		return toneError
	default:
		return toneNeutral
	}
}

func (d *dashboard) addRestoreAction() {
	label := d.svc.T("action.restore")
	if d.situation.Restore != session.RestoreAvailable {
		button := newActionButton(label, nil)
		button.Disable()
		d.add(captionWrapLabel(d.svc.T(restoreReasonKey(d.situation.Restore))), tall(button, 48))
		return
	}
	d.add(tall(primaryButton(label, d.confirmRestoreCurrent), 48))
}

func (d *dashboard) addPendingCallout() {
	switch d.situation.Kind {
	case session.SituationRestorePending, session.SituationOrphan, session.SituationConflict:
	default:
		return
	}
	title := d.svc.T("pending.title")
	body := d.svc.T("pending.body")
	if d.situation.Kind == session.SituationOrphan || d.situation.Kind == session.SituationConflict {
		title = d.svc.T("pending.orphanTitle")
		body = d.svc.T("pending.orphanBody")
	} else if d.pendingGame.AppID != 0 {
		body = fmt.Sprintf(d.svc.T("pending.body"), d.gameName(d.pendingGame))
	}
	extra := []fyne.CanvasObject{}
	if d.situation.Restore == session.RestoreNoBackup {
		extra = append(extra, captionWrapLabel(d.svc.T("pending.noBackup")))
	}
	d.add(newCallout(title, body, toneWarning, extra...).obj)
	if d.situation.Restore == session.RestoreAvailable {
		d.add(tall(newActionButton(d.svc.T("pending.restore"), d.confirmRestoreCurrent), 48))
	} else {
		d.add(captionWrapLabel(d.svc.T(restoreReasonKey(d.situation.Restore))))
	}
	guide := newActionButton(d.svc.T("pending.guide"), d.openGuide)
	guide.Importance = widget.LowImportance
	d.add(tall(guide, 40))
}

func (d *dashboard) sidebarGuide() {
	d.add(headingLabel(d.svc.T("guide.title")))
	d.add(bodyLabel(d.svc.T("guide.intro")))

	statusBody := d.svc.T("guide.setupMissing")
	switch d.situation.Setup {
	case session.SetupGuide:
		statusBody = d.svc.T("guide.setupGuide")
	case session.SetupInstalled:
		statusBody = d.svc.T("guide.setupInstalled")
	case session.SetupRestored:
		statusBody = d.svc.T("guide.setupRestored")
	case session.SetupConflict:
		statusBody = d.svc.T("guide.setupConflict")
	}
	d.add(newCallout(d.svc.T("setup.title")+": "+d.svc.T(setupLabelKey(d.situation.Setup)), statusBody, situationTone(d.situation.Kind)).obj)

	d.add(sectionLabel(d.svc.T("guide.command")))
	command, err := d.svc.DispatcherCommand()
	if err != nil {
		d.add(captionWrapLabel(err.Error()))
	} else {
		d.add(monoWrapLabel(command))
		copyButton := newActionButton(d.svc.T("action.copyCommand"), func() {
			d.window.Clipboard().SetContent(command)
			d.notice = d.svc.T("guide.copyDone")
			d.rebuildSidebar()
		})
		copyButton.Importance = widget.LowImportance
		d.add(tall(copyButton, 40))
	}
	d.add(captionWrapLabel(d.svc.T("guide.commandHint")))

	record := newActionButton(d.svc.T("action.prepareGuide"), d.recordGuideSetup)
	record.Importance = widget.LowImportance
	d.add(tall(record, 40))
	refresh := newActionButton(d.svc.T("action.refreshSetup"), d.refreshSituation)
	refresh.Importance = widget.LowImportance
	d.add(tall(refresh, 40))

	if d.situation.SteamRunning {
		d.add(captionWrapLabel(d.svc.T("guide.steamOpen")))
		install := newActionButton(d.svc.T("action.installAuto"), nil)
		install.Disable()
		d.add(captionWrapLabel(d.svc.T("guide.autoUnavailable")), tall(install, 40))
	} else {
		d.add(captionWrapLabel(d.svc.T("guide.steamClosed")))
		d.add(tall(primaryButton(d.svc.T("action.installAuto"), d.installSetup), 48))
	}

	d.add(widget.NewSeparator())
	steps := []struct{ title, body string }{
		{"guide.step1.title", "guide.step1.body"},
		{"guide.step2.title", "guide.step2.body"},
		{"guide.step3.title", "guide.step3.body"},
		{"guide.step4.title", "guide.step4.body"},
		{"guide.step5.title", "guide.step5.body"},
		{"guide.step6.title", "guide.step6.body"},
	}
	rows := make([]fyne.CanvasObject, 0, len(steps))
	for index, step := range steps {
		rows = append(rows, numberedRow(index+1, bodyLabel(d.svc.T(step.title)+". "+d.svc.T(step.body)), toneInfo))
	}
	d.add(vbox(12, rows...))

	if d.setup.Setup != nil && d.setup.Setup.Backup.Path != "" {
		d.add(sectionLabel(d.svc.T("guide.originalValue")))
		original := d.setup.Setup.Backup.Meta.OriginalValue
		if original == "" {
			d.add(captionWrapLabel(d.svc.T("guide.originalEmpty")))
		} else {
			d.add(pathLabel(original))
		}
		d.add(captionWrapLabel(d.svc.T("guide.restoreHint")))
	}

	d.add(sectionLabel(d.svc.T("guide.images")))
	d.add(guideFigure(d.svc.T("guide.image.launchOptions"), guideImageLaunchOptions, d.svc.T("guide.image.placeholder")))
	d.add(guideFigure(d.svc.T("guide.image.command"), guideImageCommand, d.svc.T("guide.image.placeholder")))
	d.add(guideFigure(d.svc.T("guide.image.popup"), guideImagePopup, d.svc.T("guide.image.placeholder")))
	d.add(guideFigure(d.svc.T("guide.image.invite"), guideImageInvite, d.svc.T("guide.image.placeholder")))
	d.sidebarNotice()
}

func (d *dashboard) openGuide() {
	d.sessionTab = 2
	if d.sessionTabs != nil {
		d.sessionTabs.selectTab(2)
	}
	d.rebuildSidebar()
}

func (d *dashboard) replaySession() {
	if d.preparing || d.activeToken != "" {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	d.preparing = true
	d.prepStarted = time.Now()
	d.prepCancel = cancel
	d.prepPhase = domain.PhasePreparing
	d.cancelRequested = false
	d.lastError = nil
	d.rebuildSidebar()
	go func() {
		active, err := d.svc.ReplaySession(ctx)
		fyne.Do(func() {
			d.preparing = false
			d.prepCancel = nil
			d.cancelRequested = false
			if err != nil {
				if errors.Is(err, config.ErrSetupMissing) {
					d.sessionTab = 2
					d.notice = d.svc.T("error.setupRequired")
					d.rebuildSidebar()
					return
				}
				d.setError(err, d.replaySession)
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
			d.lastError = nil
			d.rebuildSidebar()
			d.startPoll(active.Token)
		})
	}()
}

func (d *dashboard) recordGuideSetup() {
	go func() {
		_, err := d.svc.RecordGuideSetup(context.Background())
		fyne.Do(func() {
			if err != nil {
				d.setError(err, d.recordGuideSetup)
				return
			}
			d.notice = d.svc.T("guide.recordDone")
			d.refreshSituation()
		})
	}()
}

func (d *dashboard) installSetup() {
	go func() {
		_, err := d.svc.InstallSetup(context.Background())
		fyne.Do(func() {
			if err != nil {
				if errors.Is(err, domain.ErrSteamRunning) {
					d.notice = d.svc.T("guide.autoUnavailable")
					d.refreshSituation()
					return
				}
				d.setError(err, d.installSetup)
				return
			}
			d.notice = d.svc.T("guide.installDone")
			d.refreshSituation()
		})
	}()
}

func (d *dashboard) sidebarSelection() {
	game, ok := findGame(d.catalog.Games, d.selectedID)
	if !ok {
		d.add(headingLabel(d.svc.T("primary.select")))
		hint := bodyLabel(d.svc.T("selection.hint"))
		hint.Importance = widget.LowImportance
		d.add(hint)
		d.sidebarNotice()
		switch {
		case !d.donorFound:
			d.add(
				captionWrapLabel(d.svc.T("selection.donorHint")),
				tall(primaryButton(d.svc.T("action.addSpacewar"), d.openSpaceWar), 48),
			)
			if len(d.catalog.Games) == 0 && !d.loading && d.refreshErr == nil {
				openSteam := newActionButton(d.svc.T("action.openSteam"), d.openSteam)
				openSteam.Importance = widget.LowImportance
				d.add(tall(openSteam, 40))
			}
		default:
			button := primaryButton(d.svc.T("primary.select"), nil)
			button.Disable()
			d.add(tall(button, 48))
		}
		d.addPendingCallout()
		return
	}

	d.add(d.gameCover(game))
	d.add(d.gameInfo(game))

	switch {
	case !d.donorFound:
		d.add(
			captionWrapLabel(d.svc.T("selection.donorHint")),
			tall(primaryButton(d.svc.T("action.addSpacewar"), d.openSpaceWar), 48),
		)
	case game.AppID == domain.SpaceWarAppID:
		button := primaryButton(d.svc.T("donor.self"), nil)
		button.Disable()
		d.add(tall(button, 48))
	case game.Status == domain.StatusIncompatible || game.Status == domain.StatusMissing || game.Status == domain.StatusError:
		button := primaryButton(d.svc.T(statusLabelKey(game.Status)), nil)
		button.Disable()
		d.add(tall(button, 48))
	case d.situation.Setup == session.SetupInstalled:
		d.add(
			captionWrapLabel(d.svc.T("selection.actionHint")),
			tall(primaryButton(d.svc.T("action.play"), d.prepareSelected), 48),
		)
	default:
		d.add(
			captionWrapLabel(d.svc.T("selection.guideHint")),
			tall(primaryButton(d.svc.T("action.viewGuide"), d.openGuide), 48),
		)
	}
	d.addPendingCallout()

	if len(d.warnings) > 0 {
		content := vbox(8)
		for _, warning := range d.warnings {
			content.Add(captionWrapLabel(warning))
		}
		d.add(newDisclosure(d.svc.T("library.warnings"), content))
	}
	d.sidebarNotice()
}

func (d *dashboard) gameCover(game domain.Game) fyne.CanvasObject {
	const coverHeight float32 = 128
	if path := steamCoverPath(game); path != "" {
		image := canvas.NewImageFromFile(path)
		image.FillMode = canvas.ImageFillCover
		image.CornerRadius = 4
		image.SetMinSize(fyne.NewSize(0, coverHeight))
		return tall(image, coverHeight)
	}

	background := canvas.NewRectangle(colorSurfaceRaised)
	background.CornerRadius = 4
	name := widget.NewLabelWithStyle(d.gameName(game), fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	name.SizeName = theme.SizeNameHeadingText
	name.Truncation = fyne.TextTruncateEllipsis
	name.Wrapping = fyne.TextWrapOff
	kind := captionLabel(d.svc.T(kindLabelKey(game.Kind)))
	kind.Alignment = fyne.TextAlignCenter
	return tall(
		container.New(&coverFallbackLayout{}, background, name, kind),
		coverHeight,
	)
}

type coverFallbackLayout struct{}

func (coverFallbackLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) < 3 {
		return
	}
	background, name, kind := objects[0], objects[1], objects[2]
	background.Resize(size)

	width := fyne.Max(0, size.Width-40)
	nameHeight := fyne.Max(24, name.MinSize().Height)
	kindHeight := fyne.Max(14, kind.MinSize().Height)
	total := nameHeight + 4 + kindHeight
	y := fyne.Max(8, (size.Height-total)/2)

	name.Resize(fyne.NewSize(width, nameHeight))
	name.Move(fyne.NewPos(20, y))
	kind.Resize(fyne.NewSize(width, kindHeight))
	kind.Move(fyne.NewPos(20, y+nameHeight+4))
}

func (coverFallbackLayout) MinSize([]fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(0, 128)
}

func (d *dashboard) gameInfo(game domain.Game) fyne.CanvasObject {
	name := widget.NewLabelWithStyle(d.gameName(game), fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	name.SizeName = theme.SizeNameHeadingText
	name.Wrapping = fyne.TextWrapWord

	protonValue := d.svc.T("common.unknown")
	switch game.Kind {
	case domain.KindProton:
		protonValue = d.svc.T("common.yes")
	case domain.KindNative:
		protonValue = d.svc.T("common.no")
	}
	proton := fieldValueLabel(protonValue)
	proton.Importance = widget.LowImportance

	status := statusLabel(d.svc.T(statusLabelKey(game.Status)), statusTone(game.Status))
	status.SizeName = sizeNameFieldValue

	grid := container.New(
		&infoGridLayout{gap: 12, leftShare: 0.5},
		infoField(d.svc.T("details.proton"), proton),
		infoField(d.svc.T("library.columnStatus"), status),
		infoField(d.svc.T("details.appid"), monoLabel(strconv.FormatUint(uint64(game.AppID), 10))),
	)

	content := vbox(12, name, grid)
	if game.Problem != nil {
		problem := captionWrapLabel(game.Problem.Error())
		problem.Importance = widget.WarningImportance
		content.Add(problem)
	}
	return content
}

func (d *dashboard) detailsView(game domain.Game) fyne.CanvasObject {
	rows := []fyne.CanvasObject{
		detailRow(d.svc.T("details.appid"), strconv.FormatUint(uint64(game.AppID), 10), true, false),
		detailRow(d.svc.T("details.library"), game.LibraryPath, false, true),
		detailRow(d.svc.T("details.kind"), d.svc.T(kindLabelKey(game.Kind)), false, false),
		detailRow(d.svc.T("details.path"), game.InstallDir, false, true),
	}
	if game.Launch != nil && game.Launch.ProtonName != "" {
		rows = append(rows, detailRow(d.svc.T("details.proton"), game.Launch.ProtonName, false, false))
	}
	return vbox(12, rows...)
}

func (d *dashboard) detailsGame() (domain.Game, bool) {
	if game, ok := d.selectedGame(); ok {
		return game, true
	}
	if d.activeGame.AppID != 0 {
		return d.activeGame, true
	}
	if d.pendingGame.AppID != 0 {
		return d.pendingGame, true
	}
	return domain.Game{}, false
}

func (d *dashboard) updateSessionHint() {
	if d.sessionHint == nil {
		return
	}
	text := ""
	if game, ok := d.detailsGame(); ok && game.Status == domain.StatusUnknown {
		text = d.svc.T("compat.unknownHint")
	}
	d.sessionHint.SetText(text)
	if text == "" {
		d.sessionHint.Hide()
	} else {
		d.sessionHint.Show()
	}
	d.sessionHint.Refresh()
	if d.sessionHeader != nil {
		d.sessionHeader.Refresh()
	}
}

func (d *dashboard) sidebarDetails() {
	game, ok := d.detailsGame()
	if !ok {
		d.add(sectionLabel(d.svc.T("details.emptyTitle")))
		hint := bodyLabel(d.svc.T("details.emptyHint"))
		hint.Importance = widget.LowImportance
		d.add(hint)
		return
	}

	name := widget.NewLabelWithStyle(d.gameName(game), fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	name.SizeName = sizeNameRowTitle
	name.Wrapping = fyne.TextWrapWord
	d.add(name, d.detailsView(game))
}

func (d *dashboard) sidebarNotice() {
	if d.notice == "" {
		return
	}
	notice := captionWrapLabel(d.notice)
	notice.Importance = widget.WarningImportance
	d.add(notice)
}

func (d *dashboard) finishSidebar() {
	d.sidebarBox.Refresh()
	if d.sidebarScroll != nil {
		d.sidebarScroll.Refresh()
	}
}

func (d *dashboard) selectedGame() (domain.Game, bool) {
	return findGame(d.catalog.Games, d.selectedID)
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
	d.elapsedLabel.Text = formatDuration(time.Since(d.activeStart))
	d.elapsedLabel.Refresh()
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
			d.startSession()
		}, d.window)
		return
	}
	d.startSession()
}

func (d *dashboard) startSession() {
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
	d.prepPhase = domain.PhasePreparing
	d.cancelRequested = false
	d.lastError = nil
	d.lastDetail = ""
	d.retry = nil
	d.rebuildSidebar()
	go func() {
		active, err := d.svc.StartSession(ctx, session.StartOptions{
			TargetAppID: game.AppID,
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
				switch {
				case errors.Is(err, config.ErrSetupMissing):
					d.sessionTab = 2
					d.notice = d.svc.T("error.setupRequired")
					d.rebuildSidebar()
					return
				case errors.Is(err, config.ErrSetupConflict):
					d.sessionTab = 2
					d.notice = d.svc.T("error.setupConflict")
					d.rebuildSidebar()
					return
				case errors.Is(err, context.Canceled):
					d.refreshSituation()
					return
				}
				d.setError(err, d.startSession)
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
			d.lastError = nil
			d.rebuildSidebar()
			d.startPoll(active.Token)
		})
	}()
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
						d.finishSession()
						return
					}
					d.activeState = state
					if state.StartedAt != (time.Time{}) {
						d.activeStart = state.StartedAt
					}
					if state.Terminal() {
						d.finishSession()
						return
					}
					d.updateElapsed()
				})
			}
		}
	}()
}

func (d *dashboard) finishSession() {
	if d.pollCancel != nil {
		d.pollCancel()
		d.pollCancel = nil
	}
	d.activeToken = ""
	d.refreshSituation()
}

func (d *dashboard) confirmRestoreCurrent() {
	if d.situation.Restore != session.RestoreAvailable || d.restoring || d.restoringSetup {
		return
	}
	if d.situation.Setup == session.SetupInstalled && d.setup.Setup != nil {
		d.confirmRestoreSetup()
		return
	}
	if d.pendingRestore != "" {
		d.confirmRestore(d.pendingRestore)
		return
	}
	d.confirmRestoreSetup()
}

func (d *dashboard) confirmRestoreSetup() {
	dialog.ShowConfirm(d.svc.T("dialog.restore.title"), d.svc.T("dialog.restore.body"), func(ok bool) {
		if ok {
			d.restoreSetup()
		}
	}, d.window)
}

func (d *dashboard) restoreSetup() {
	if d.restoringSetup {
		return
	}
	d.restoringSetup = true
	d.lastError = nil
	d.rebuildSidebar()
	go func() {
		_, err := d.svc.RestoreSetup(context.Background())
		fyne.Do(func() {
			d.restoringSetup = false
			if err != nil {
				if errors.Is(err, domain.ErrSteamRunning) {
					d.sessionTab = 2
					d.notice = d.svc.T(restoreReasonKey(session.RestoreSteamRunning))
					d.refreshSituation()
					return
				}
				if errors.Is(err, config.ErrLocalConfigChanged) {
					d.sessionTab = 2
					d.notice = d.svc.T(restoreReasonKey(session.RestoreConflict))
					d.refreshSituation()
					return
				}
				d.setError(err, d.restoreSetup)
				return
			}
			d.lastError = nil
			d.refreshSituation()
			dialog.ShowInformation(d.svc.T("restore.done.title"), d.svc.T("restore.done.body"), d.window)
		})
	}()
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
				if errors.Is(err, domain.ErrSteamRunning) {
					d.sessionTab = 2
					d.notice = d.svc.T(restoreReasonKey(session.RestoreSteamRunning))
					d.refreshSituation()
					return
				}
				if errors.Is(err, config.ErrLocalConfigChanged) {
					d.sessionTab = 2
					d.notice = d.svc.T(restoreReasonKey(session.RestoreConflict))
					d.refreshSituation()
					return
				}
				d.setError(err, func() { d.restoreSession(token) })
				return
			}
			d.pendingRestore = ""
			d.pendingGame = domain.Game{}
			d.lastError = nil
			d.refreshSituation()
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
