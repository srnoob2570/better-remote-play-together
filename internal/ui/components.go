package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/srnoob2570/better-rpt/internal/domain"
)

func hbox(gap float32, objects ...fyne.CanvasObject) *fyne.Container {
	return container.New(layout.NewCustomPaddedHBoxLayout(gap), objects...)
}

func vbox(gap float32, objects ...fyne.CanvasObject) *fyne.Container {
	return container.New(layout.NewCustomPaddedVBoxLayout(gap), objects...)
}

func pad(object fyne.CanvasObject, top, bottom, left, right float32) fyne.CanvasObject {
	return container.New(layout.NewCustomPaddedLayout(top, bottom, left, right), object)
}

func vSpace(height float32) fyne.CanvasObject {
	space := canvas.NewRectangle(color.Transparent)
	space.SetMinSize(fyne.NewSize(0, height))
	return space
}

// tall forces a widget to a fixed height while keeping the parent width. The
// regular VBox layout only respects minimum heights.
func tall(object fyne.CanvasObject, height float32) fyne.CanvasObject {
	return container.New(&fixedHeightLayout{height: height}, object)
}

type fixedHeightLayout struct {
	height float32
}

func (l *fixedHeightLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	for _, object := range objects {
		object.Move(fyne.NewPos(0, 0))
		object.Resize(fyne.NewSize(size.Width, l.height))
	}
}

func (l *fixedHeightLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	width := float32(0)
	for _, object := range objects {
		width = fyne.Max(width, object.MinSize().Width)
	}
	return fyne.NewSize(width, l.height)
}

// scrollColumn keeps a vertical scroll at the viewport width. Fyne's Scroll
// expands its content to the content's MinSize, which preserves a label's old
// wrap width after the window shrinks and clips text under the scrollbar.
type scrollColumn struct{}

func (l *scrollColumn) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	for _, object := range objects {
		object.Move(fyne.NewPos(0, 0))
		object.Resize(size)
	}
}

func (l *scrollColumn) MinSize(objects []fyne.CanvasObject) fyne.Size {
	height := float32(0)
	for _, object := range objects {
		height = fyne.Max(height, object.MinSize().Height)
	}
	return fyne.NewSize(0, height)
}

// twoPaneLayout gives the session stage a fixed share of the window and lets
// the library take the rest. Unlike HSplit it has no draggable handle, so the
// seam reads as a change of field instead of a scrollbar.
type twoPaneLayout struct {
	panelRatio float32
	minPanel   float32
}

func (l *twoPaneLayout) panelWidth(size fyne.Size) float32 {
	width := size.Width * l.panelRatio
	if width < l.minPanel {
		width = l.minPanel
	}
	return fyne.Min(width, size.Width)
}

func (l *twoPaneLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) < 2 {
		return
	}
	panel := l.panelWidth(size)
	library := fyne.Max(0, size.Width-panel)
	objects[0].Move(fyne.NewPos(0, 0))
	objects[0].Resize(fyne.NewSize(library, size.Height))
	objects[1].Move(fyne.NewPos(library, 0))
	objects[1].Resize(fyne.NewSize(panel, size.Height))
}

func (l *twoPaneLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	height := float32(0)
	for _, object := range objects {
		height = fyne.Max(height, object.MinSize().Height)
	}
	if len(objects) < 2 {
		return fyne.NewSize(0, height)
	}
	return fyne.NewSize(
		objects[0].MinSize().Width+fyne.Max(l.minPanel, objects[1].MinSize().Width),
		height,
	)
}

// fieldLayout gives the key a fixed gutter and the value the remaining width,
// with no layout padding between them. This keeps the value columns aligned
// and gives short values such as "Proton" the whole remainder of their cell.
type fieldLayout struct {
	labelWidth float32
}

func (l fieldLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) < 2 {
		return
	}
	baseline := float32(0)
	for _, object := range objects[:2] {
		if label, ok := object.(*widget.Label); ok {
			baseline = fyne.Max(baseline, labelTextBaseline(label))
		}
	}
	objects[0].Move(fyne.NewPos(0, fieldTextY(objects[0], baseline)))
	objects[0].Resize(fyne.NewSize(l.labelWidth, size.Height))
	objects[1].Move(fyne.NewPos(l.labelWidth, fieldTextY(objects[1], baseline)))
	objects[1].Resize(fyne.NewSize(fyne.Max(0, size.Width-l.labelWidth), size.Height))
}

func fieldTextY(object fyne.CanvasObject, baseline float32) float32 {
	label, ok := object.(*widget.Label)
	if !ok {
		return 0
	}
	return baseline - labelTextBaseline(label)
}

func (l fieldLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) < 2 {
		return fyne.NewSize(l.labelWidth, 0)
	}
	return fyne.NewSize(
		l.labelWidth+objects[1].MinSize().Width,
		fyne.Max(objects[0].MinSize().Height, objects[1].MinSize().Height),
	)
}

func infoField(label string, value fyne.CanvasObject) fyne.CanvasObject {
	return container.New(&fieldLayout{labelWidth: 58}, captionLabel(label+":"), value)
}

// infoGridLayout gives the game-info fields two columns with a wider left
// side for the game name and state, and a narrower right side for the short
// values (Proton, App ID). Equal columns force the name to break mid-word.
type infoGridLayout struct {
	gap       float32
	leftShare float32
}

func (l infoGridLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) < 3 {
		return
	}
	left := (size.Width - l.gap) * l.leftShare
	right := size.Width - l.gap - left
	firstRow := fyne.Max(objects[0].MinSize().Height, objects[1].MinSize().Height)
	secondRow := objects[2].MinSize().Height
	if len(objects) > 3 {
		secondRow = fyne.Max(secondRow, objects[3].MinSize().Height)
	}

	objects[0].Move(fyne.NewPos(0, 0))
	objects[0].Resize(fyne.NewSize(left, firstRow))
	objects[1].Move(fyne.NewPos(left+l.gap, 0))
	objects[1].Resize(fyne.NewSize(right, firstRow))
	objects[2].Move(fyne.NewPos(0, firstRow+l.gap))
	objects[2].Resize(fyne.NewSize(left, secondRow))
	if len(objects) > 3 {
		objects[3].Move(fyne.NewPos(left+l.gap, firstRow+l.gap))
		objects[3].Resize(fyne.NewSize(right, secondRow))
	}
}

func (l infoGridLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) < 3 {
		return fyne.NewSize(260, 0)
	}
	firstRow := fyne.Max(objects[0].MinSize().Height, objects[1].MinSize().Height)
	secondRow := objects[2].MinSize().Height
	if len(objects) > 3 {
		secondRow = fyne.Max(secondRow, objects[3].MinSize().Height)
	}
	return fyne.NewSize(260, firstRow+l.gap+secondRow)
}

func sectionLabel(text string) *widget.Label {
	label := widget.NewLabelWithStyle(text, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	label.SizeName = theme.SizeNameSubHeadingText
	return label
}

func headingLabel(text string) *widget.Label {
	label := widget.NewLabelWithStyle(text, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	label.SizeName = theme.SizeNameHeadingText
	label.Wrapping = fyne.TextWrapWord
	return label
}

func bodyLabel(text string) *widget.Label {
	label := widget.NewLabel(text)
	label.Wrapping = fyne.TextWrapWord
	return label
}

func fieldValueLabel(text string) *widget.Label {
	label := widget.NewLabel(text)
	label.SizeName = sizeNameFieldValue
	label.Wrapping = fyne.TextWrapWord
	return label
}

func captionLabel(text string) *widget.Label {
	label := widget.NewLabel(text)
	label.SizeName = theme.SizeNameCaptionText
	label.Importance = widget.LowImportance
	return label
}

// labelTextBaseline is the distance from a label's top to the baseline of its
// first line. Fyne labels draw their text from the top of the widget, so two
// labels with different type sizes cannot be aligned by centering their boxes.
func labelTextBaseline(label *widget.Label) float32 {
	th := label.Theme()
	sizeName := label.SizeName
	if sizeName == "" {
		sizeName = theme.SizeNameText
	}
	_, baseline := fyne.CurrentApp().Driver().RenderedTextSize(
		label.Text,
		th.Size(sizeName),
		label.TextStyle,
		nil,
	)
	return th.Size(theme.SizeNameInnerPadding) + baseline
}

// captionWrapLabel is the prose variant of captionLabel. Keep captionLabel
// single-line so it reports its full width inside horizontal layouts.
func captionWrapLabel(text string) *widget.Label {
	label := captionLabel(text)
	label.Wrapping = fyne.TextWrapWord
	return label
}

func monoLabel(text string) *widget.Label {
	label := widget.NewLabel(text)
	label.SizeName = theme.SizeNameCaptionText
	label.TextStyle = fyne.TextStyle{Monospace: true}
	label.Importance = widget.LowImportance
	return label
}

// monoWrapLabel is monoLabel for long commands that must remain readable in a
// narrow sidebar; it breaks instead of forcing horizontal clipping.
func monoWrapLabel(text string) *widget.Label {
	label := monoLabel(text)
	label.Wrapping = fyne.TextWrapBreak
	return label
}

// pathLabel breaks inside long unspaced values so a Steam path cannot force
// the inspector wider than the window.
func pathLabel(text string) *widget.Label {
	label := bodyLabel(text)
	label.Wrapping = fyne.TextWrapBreak
	return label
}

// statusLabel carries the state in words. Colour is a redundant cue, never
// the only signal.
func statusLabel(text string, value tone) *widget.Label {
	label := captionLabel(text)
	label.Importance = statusImportance(value)
	return label
}

func statusImportance(value tone) widget.Importance {
	switch value {
	case toneSuccess:
		return widget.SuccessImportance
	case toneWarning:
		return widget.WarningImportance
	case toneError:
		return widget.DangerImportance
	case toneInfo:
		return widget.HighImportance
	default:
		return widget.LowImportance
	}
}

type gameRow struct {
	widget.BaseWidget

	name         *widget.Label
	kind         *widget.Label
	status       *widget.Label
	icon         *canvas.Image
	iconFallback *fyne.Container
	iconInitial  *canvas.Text
	selectedBg   *canvas.Rectangle
	hoverBg      *canvas.Rectangle
	selectedBar  *canvas.Rectangle
	separator    *canvas.Rectangle
	root         *fyne.Container
}

func newGameRow() *gameRow {
	iconFallback := canvas.NewRectangle(colorSurfaceRaised)
	iconFallback.CornerRadius = 3
	iconInitial := canvas.NewText("?", colorText)
	iconInitial.TextSize = 11
	iconInitial.TextStyle = fyne.TextStyle{Bold: true}
	iconInitial.Alignment = fyne.TextAlignCenter
	fallback := container.NewStack(iconFallback, container.NewCenter(iconInitial))

	row := &gameRow{
		name:         widget.NewLabel(""),
		kind:         widget.NewLabel(""),
		status:       statusLabel("", toneNeutral),
		icon:         canvas.NewImageFromFile(""),
		iconFallback: fallback,
		iconInitial:  iconInitial,
		selectedBg:   canvas.NewRectangle(color.Transparent),
		hoverBg:      canvas.NewRectangle(color.Transparent),
		selectedBar:  canvas.NewRectangle(colorAmber),
		separator:    canvas.NewRectangle(colorLineSoft),
	}
	row.icon.FillMode = canvas.ImageFillStretch
	row.icon.CornerRadius = 3
	row.icon.Hide()
	row.iconFallback.Hide()
	row.name.SizeName = sizeNameRowTitle
	row.name.TextStyle = fyne.TextStyle{Bold: true}
	row.name.Truncation = fyne.TextTruncateEllipsis
	row.name.Wrapping = fyne.TextWrapOff
	row.kind.SizeName = theme.SizeNameCaptionText
	row.kind.Importance = widget.LowImportance
	row.kind.Truncation = fyne.TextTruncateEllipsis
	row.kind.Wrapping = fyne.TextWrapOff
	row.status.Truncation = fyne.TextTruncateEllipsis
	row.status.Wrapping = fyne.TextWrapOff
	row.selectedBg.Hide()
	row.hoverBg.Hide()
	row.selectedBar.Hide()
	row.root = container.New(&gameRowLayout{row: row},
		row.selectedBg, row.hoverBg, row.selectedBar, row.separator,
		row.icon, row.iconFallback, row.name, row.kind, row.status,
	)
	row.ExtendBaseWidget(row)
	return row
}

var (
	_ desktop.Hoverable  = (*gameRow)(nil)
	_ desktop.Cursorable = (*gameRow)(nil)
)

func (r *gameRow) SetGame(game domain.Game, selected bool, translate func(string) string, iconPath string) {
	r.name.SetText(game.Name)
	r.kind.SetText(translate(kindLabelKey(game.Kind)))
	r.status.SetText(translate(statusLabelKey(game.Status)))
	r.status.Importance = statusImportance(statusTone(game.Status))
	r.status.Refresh()

	initial := "?"
	if runes := []rune(game.Name); len(runes) > 0 {
		initial = string(runes[0])
	}
	r.iconInitial.Text = initial
	r.iconInitial.Refresh()
	if iconPath != "" {
		r.icon.File = iconPath
		r.icon.Refresh()
		r.icon.Show()
		r.iconFallback.Hide()
	} else {
		r.icon.Hide()
		r.iconFallback.Show()
		r.iconFallback.Refresh()
	}

	if selected {
		r.selectedBg.FillColor = colorSelected
		r.selectedBg.Show()
		r.selectedBar.Show()
	} else {
		r.selectedBg.Hide()
		r.selectedBar.Hide()
	}
	r.selectedBg.Refresh()
	r.selectedBar.Refresh()
	r.Refresh()
}

func (r *gameRow) MouseIn(*desktop.MouseEvent) {
	r.showHover()
}

func (r *gameRow) MouseMoved(*desktop.MouseEvent) {
	r.showHover()
}

func (r *gameRow) showHover() {
	if r.selectedBg.Hidden {
		r.hoverBg.FillColor = colorHover
		r.hoverBg.Show()
		r.hoverBg.Refresh()
	}
}

func (r *gameRow) MouseOut() {
	r.hoverBg.Hide()
	r.hoverBg.Refresh()
}

func (r *gameRow) Cursor() desktop.Cursor {
	return desktop.PointerCursor
}

func (r *gameRow) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(r.root)
}

type gameRowLayout struct {
	row *gameRow
}

const (
	libraryGutter      float32 = 12
	libraryColumnGap   float32 = 20
	libraryKindWidth   float32 = 76
	libraryStatusWidth float32 = 112
	libraryArtSize     float32 = 24
	libraryArtGap      float32 = 10
)

type libraryColumns struct {
	nameX     float32
	nameWidth float32
	kindX     float32
	statusX   float32
}

// libraryColumnsFor anchors the type and status columns to the right edge.
// The cells inside each column are left aligned, so their text never shifts
// when a game name or a status word changes length.
func libraryColumnsFor(width float32) libraryColumns {
	statusX := width - libraryGutter - libraryStatusWidth
	kindX := statusX - libraryColumnGap - libraryKindWidth
	return libraryColumns{
		nameX:     libraryGutter,
		nameWidth: fyne.Max(0, kindX-libraryColumnGap-libraryGutter),
		kindX:     kindX,
		statusX:   statusX,
	}
}

func placeLibraryCell(object fyne.CanvasObject, x, width, height float32) {
	size := object.MinSize()
	object.Resize(fyne.NewSize(width, size.Height))
	object.Move(fyne.NewPos(x, (height-size.Height)/2))
}

func (l gameRowLayout) Layout(_ []fyne.CanvasObject, size fyne.Size) {
	row := l.row
	const rowInset float32 = 4
	highlightHeight := fyne.Max(0, size.Height-rowInset)
	row.selectedBg.Resize(fyne.NewSize(size.Width, highlightHeight))
	row.selectedBg.Move(fyne.NewPos(0, 0))
	row.hoverBg.Resize(fyne.NewSize(size.Width, highlightHeight))
	row.hoverBg.Move(fyne.NewPos(0, 0))
	row.selectedBar.Resize(fyne.NewSize(3, highlightHeight))
	row.selectedBar.Move(fyne.NewPos(0, 0))
	row.separator.Resize(fyne.NewSize(fyne.Max(0, size.Width-2*libraryGutter), 1))
	row.separator.Move(fyne.NewPos(libraryGutter, size.Height-1))

	columns := libraryColumnsFor(size.Width)
	iconY := (highlightHeight - libraryArtSize) / 2
	row.icon.Resize(fyne.NewSize(libraryArtSize, libraryArtSize))
	row.icon.Move(fyne.NewPos(columns.nameX, iconY))
	row.iconFallback.Resize(fyne.NewSize(libraryArtSize, libraryArtSize))
	row.iconFallback.Move(fyne.NewPos(columns.nameX, iconY))

	nameX := columns.nameX + libraryArtSize + libraryArtGap
	nameWidth := fyne.Max(0, columns.nameWidth-libraryArtSize-libraryArtGap)
	placeLibraryCell(row.name, nameX, nameWidth, highlightHeight)
	placeLibraryCell(row.kind, columns.kindX, libraryKindWidth, highlightHeight)
	placeLibraryCell(row.status, columns.statusX, libraryStatusWidth, highlightHeight)
}

func (gameRowLayout) MinSize(_ []fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(360, 40)
}

type tableHeaderLayout struct{}

func (tableHeaderLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	rule, game, kind, status := objects[0], objects[1], objects[2], objects[3]
	rule.Resize(fyne.NewSize(size.Width, 1))
	rule.Move(fyne.NewPos(0, size.Height-1))
	columns := libraryColumnsFor(size.Width)
	placeLibraryCell(game, columns.nameX, columns.nameWidth, size.Height)
	placeLibraryCell(kind, columns.kindX, libraryKindWidth, size.Height)
	placeLibraryCell(status, columns.statusX, libraryStatusWidth, size.Height)
}

func (tableHeaderLayout) MinSize([]fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(360, 28)
}

func newLibraryTableHeader(translate func(string) string) fyne.CanvasObject {
	return container.New(&tableHeaderLayout{},
		canvas.NewRectangle(colorLineSoft),
		captionLabel(translate("library.columnGame")),
		captionLabel(translate("library.columnKind")),
		captionLabel(translate("library.columnStatus")),
	)
}

// listTableLayout joins the table header and the virtualized list without the
// BorderLayout theme padding, so the first row uses the same 4px gap as the
// separators between the rows below it.
type listTableLayout struct {
	gap float32
}

func (l listTableLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) < 2 {
		return
	}
	header, list := objects[0], objects[1]
	headerHeight := header.MinSize().Height
	header.Move(fyne.NewPos(0, 0))
	header.Resize(fyne.NewSize(size.Width, headerHeight))
	list.Move(fyne.NewPos(0, headerHeight+l.gap))
	list.Resize(fyne.NewSize(size.Width, fyne.Max(0, size.Height-headerHeight-l.gap)))
}

func (l listTableLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) < 2 {
		return fyne.NewSize(0, 0)
	}
	header, list := objects[0].MinSize(), objects[1].MinSize()
	return fyne.NewSize(
		fyne.Max(header.Width, list.Width),
		header.Height+l.gap+list.Height,
	)
}

type filterTab struct {
	widget.BaseWidget

	label           *canvas.Text
	underline       *canvas.Rectangle
	underlineHeight float32
	root            *fyne.Container

	selected bool
	hovered  bool
	focused  bool
	onTap    func()
}

func newFilterTab(label string, onTap func()) *filterTab {
	tab := &filterTab{
		label:     canvas.NewText(label, colorMuted),
		underline: canvas.NewRectangle(color.Transparent),
		onTap:     onTap,
	}
	tab.label.TextSize = 13
	tab.underline.Hide()
	tab.root = container.New(&filterTabLayout{tab: tab}, tab.underline, tab.label)
	tab.ExtendBaseWidget(tab)
	return tab
}

var (
	_ desktop.Hoverable  = (*filterTab)(nil)
	_ desktop.Cursorable = (*filterTab)(nil)
)

func (t *filterTab) SetLabel(label string) {
	t.label.Text = label
	t.label.Refresh()
	t.Refresh()
}

func (t *filterTab) SetSelected(selected bool) {
	t.selected = selected
	t.applyStyle()
}

func (t *filterTab) Tapped(*fyne.PointEvent) {
	if t.onTap != nil {
		t.onTap()
	}
}

func (t *filterTab) MouseIn(*desktop.MouseEvent) {
	t.hovered = true
	t.applyStyle()
}

func (t *filterTab) MouseMoved(*desktop.MouseEvent) {}

func (t *filterTab) MouseOut() {
	t.hovered = false
	t.applyStyle()
}

func (t *filterTab) FocusGained() {
	t.focused = true
	t.applyStyle()
}

func (t *filterTab) FocusLost() {
	t.focused = false
	t.applyStyle()
}

func (t *filterTab) TypedRune(rune) {}

func (t *filterTab) TypedKey(event *fyne.KeyEvent) {
	if event.Name == fyne.KeyReturn || event.Name == fyne.KeySpace {
		t.Tapped(nil)
	}
}

func (t *filterTab) Cursor() desktop.Cursor {
	return desktop.PointerCursor
}

func (t *filterTab) applyStyle() {
	t.underlineHeight = 0
	switch {
	case t.selected || t.focused:
		t.underlineHeight = 2
		t.underline.FillColor = colorAmber
		t.label.Color = colorText
		t.label.TextStyle = fyne.TextStyle{Bold: true}
	case t.hovered:
		t.underlineHeight = 1
		t.underline.FillColor = colorMuted
		t.label.Color = colorText
		t.label.TextStyle = fyne.TextStyle{}
	default:
		t.label.Color = colorMuted
		t.label.TextStyle = fyne.TextStyle{}
	}
	if t.underlineHeight > 0 {
		t.underline.Show()
	} else {
		t.underline.Hide()
	}
	t.underline.Refresh()
	t.label.Refresh()
	t.Refresh()
}

func (t *filterTab) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(t.root)
}

type filterTabLayout struct {
	tab *filterTab
}

func (l filterTabLayout) Layout(_ []fyne.CanvasObject, size fyne.Size) {
	tab := l.tab
	textSize := tab.label.MinSize()
	tab.label.Resize(textSize)
	tab.label.Move(fyne.NewPos((size.Width-textSize.Width)/2, (size.Height-textSize.Height-4)/2))
	underlineWidth := textSize.Width + 8
	tab.underline.Resize(fyne.NewSize(underlineWidth, tab.underlineHeight))
	tab.underline.Move(fyne.NewPos((size.Width-underlineWidth)/2, size.Height-tab.underlineHeight))
}

func (filterTabLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	label := objects[1]
	return fyne.NewSize(label.MinSize().Width+24, 30)
}

type tabGroup struct {
	obj      *fyne.Container
	tabs     []*filterTab
	selected int
	onChange func(int)
}

func newTabGroup(labels []string, selected int, onChange func(int)) *tabGroup {
	group := &tabGroup{selected: selected, onChange: onChange}
	objects := make([]fyne.CanvasObject, len(labels))
	for index, label := range labels {
		index := index
		tab := newFilterTab(label, func() { group.selectTab(index) })
		group.tabs = append(group.tabs, tab)
		objects[index] = tab
	}
	group.obj = hbox(18, objects...)
	group.tabs[selected].SetSelected(true)
	return group
}

// newSessionTabGroup uses the same underline language as the library filters
// but at the panel's section scale, so Sesión / Detalles read as tabs rather
// than as another filter row.
func newSessionTabGroup(labels []string, selected int, onChange func(int)) *tabGroup {
	group := newTabGroup(labels, selected, onChange)
	for _, tab := range group.tabs {
		tab.label.TextSize = 18
		tab.label.Refresh()
	}
	group.obj.Refresh()
	return group
}

// sessionHeaderLayout keeps the tabs anchored left and lets the compatibility
// note sit at the far right of the same row, wrapping when the panel is narrow.
type sessionHeaderLayout struct {
	gap float32
}

func (l sessionHeaderLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) < 2 {
		return
	}
	tabs := objects[0].MinSize()
	objects[0].Resize(tabs)
	objects[0].Move(fyne.NewPos(0, (size.Height-tabs.Height)/2))

	hint := objects[1]
	if !hint.Visible() {
		return
	}
	x := tabs.Width + l.gap
	width := fyne.Max(0, size.Width-x)
	hint.Resize(fyne.NewSize(width, size.Height))
	hintY := float32(0)
	if label, ok := hint.(*widget.Label); ok {
		if center, ok := firstTabTextCenter(objects[0]); ok {
			hintY = objects[0].Position().Y + center - labelFirstLineCenter(label)
		}
	}
	hint.Move(fyne.NewPos(x, hintY))
}

// labelFirstLineCenter is the distance from a label's top to the center of its
// first text line. The note is a wrapped label, so centering its first line
// keeps the hint visually level with the single-line tabs above it.
func labelFirstLineCenter(label *widget.Label) float32 {
	th := label.Theme()
	sizeName := label.SizeName
	if sizeName == "" {
		sizeName = theme.SizeNameText
	}
	textSize, _ := fyne.CurrentApp().Driver().RenderedTextSize(
		label.Text,
		th.Size(sizeName),
		label.TextStyle,
		nil,
	)
	return th.Size(theme.SizeNameInnerPadding) + textSize.Height/2
}

// firstTabTextCenter returns the vertical center of the first tab label
// relative to the tab group. Tabs are canvas.Text objects, so their line box
// is already the visible text box.
func firstTabTextCenter(object fyne.CanvasObject) (float32, bool) {
	group, ok := object.(*fyne.Container)
	if !ok || len(group.Objects) == 0 {
		return 0, false
	}
	tab, ok := group.Objects[0].(*filterTab)
	if !ok {
		return 0, false
	}
	return tab.Position().Y + tab.label.Position().Y + tab.label.MinSize().Height/2, true
}

func (l sessionHeaderLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) == 0 {
		return fyne.NewSize(0, 0)
	}
	tabs := objects[0].MinSize()
	return fyne.NewSize(tabs.Width, tabs.Height)
}

func (g *tabGroup) selectTab(index int) {
	if index < 0 || index >= len(g.tabs) || index == g.selected {
		return
	}
	g.selected = index
	for i, tab := range g.tabs {
		tab.SetSelected(i == index)
	}
	if g.onChange != nil {
		g.onChange(index)
	}
}

// emptyState is the library's first-class blank, filtered and error view.
type emptyState struct {
	obj       *fyne.Container
	column    *fyne.Container
	title     *widget.Label
	body      *widget.Label
	action    *actionButton
	secondary *actionButton
}

func newEmptyState() *emptyState {
	state := &emptyState{
		title:     widget.NewLabelWithStyle("", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		body:      widget.NewLabel(""),
		action:    newActionButton("", nil),
		secondary: newActionButton("", nil),
	}
	state.title.SizeName = theme.SizeNameHeadingText
	state.title.Wrapping = fyne.TextWrapWord
	state.body.Alignment = fyne.TextAlignCenter
	state.body.Importance = widget.LowImportance
	state.body.Wrapping = fyne.TextWrapWord
	state.action.Importance = widget.MediumImportance
	state.secondary.Importance = widget.LowImportance
	state.action.Hide()
	state.secondary.Hide()

	state.column = vbox(0)
	state.obj = container.New(&maxWidthLayout{width: 460}, state.column)
	state.obj.Hide()
	state.rebuild()
	return state
}

func (s *emptyState) Set(title, body, actionLabel string, action func()) {
	s.title.SetText(title)
	s.body.SetText(body)
	s.secondary.Hide()
	s.secondary.OnTapped = nil
	if actionLabel == "" || action == nil {
		s.action.Hide()
		s.action.OnTapped = nil
	} else {
		s.action.SetText(actionLabel)
		s.action.OnTapped = action
		s.action.Show()
	}
	s.rebuild()
}

// SetSecondary adds the quieter follow-up action some empty states need.
func (s *emptyState) SetSecondary(label string, action func()) {
	if label == "" || action == nil {
		s.secondary.Hide()
		s.secondary.OnTapped = nil
	} else {
		s.secondary.SetText(label)
		s.secondary.OnTapped = action
		s.secondary.Show()
	}
	s.rebuild()
}

func (s *emptyState) rebuild() {
	objects := []fyne.CanvasObject{s.title}
	if s.body.Text != "" {
		objects = append(objects, vSpace(8), s.body)
	}
	buttons := make([]fyne.CanvasObject, 0, 2)
	if s.action.Visible() {
		buttons = append(buttons, s.action)
	}
	if s.secondary.Visible() {
		buttons = append(buttons, s.secondary)
	}
	if len(buttons) > 0 {
		objects = append(objects, vSpace(16), container.NewCenter(hbox(10, buttons...)))
	}
	s.column.Objects = objects
	s.column.Refresh()
	s.obj.Refresh()
}

func (s *emptyState) Show() {
	s.obj.Show()
	s.obj.Refresh()
}

func (s *emptyState) Hide() {
	s.obj.Hide()
	s.obj.Refresh()
}

type maxWidthLayout struct {
	width float32
}

func (l *maxWidthLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	width := fyne.Min(size.Width, l.width)
	for _, object := range objects {
		min := object.MinSize()
		height := fyne.Min(size.Height, min.Height)
		object.Resize(fyne.NewSize(width, height))
		object.Move(fyne.NewPos((size.Width-width)/2, (size.Height-height)/2))
	}
}

func (l *maxWidthLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	width := float32(0)
	height := float32(0)
	for _, object := range objects {
		min := object.MinSize()
		width = fyne.Min(l.width, fyne.Max(width, min.Width))
		height = fyne.Max(height, min.Height)
	}
	return fyne.NewSize(width, height)
}

type callout struct {
	obj *fyne.Container
}

func newCallout(title, body string, tone tone, extra ...fyne.CanvasObject) *callout {
	titleLabel := widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	titleLabel.SizeName = theme.SizeNameSubHeadingText
	titleLabel.Wrapping = fyne.TextWrapWord
	content := []fyne.CanvasObject{titleLabel}
	if body != "" {
		content = append(content, captionWrapLabel(body))
	}
	content = append(content, extra...)
	column := vbox(8, content...)

	background := canvas.NewRectangle(tint(toneColor(tone), 0x0c))
	background.CornerRadius = 6
	background.StrokeColor = tint(toneColor(tone), 0x33)
	background.StrokeWidth = 1

	return &callout{obj: container.NewStack(background, pad(column, 14, 14, 16, 14))}
}

// detailRow stacks a quiet key over its value; it keeps long Steam paths
// readable inside the narrow inspector.
func detailRow(key, value string, mono, breakWrap bool) fyne.CanvasObject {
	keyLabel := captionLabel(key + ":")
	valueLabel := bodyLabel(value)
	if mono {
		valueLabel.TextStyle = fyne.TextStyle{Monospace: true}
		valueLabel.SizeName = theme.SizeNameCaptionText
	}
	if breakWrap {
		valueLabel.Wrapping = fyne.TextWrapBreak
	}
	return container.New(&fieldLayout{labelWidth: 92}, keyLabel, valueLabel)
}

// actionButton is a Fyne button that reports the hand cursor. Fyne's stock
// button returns DefaultCursor, which hides that a control is clickable on
// desktop.
type actionButton struct {
	*widget.Button
	renderer    fyne.WidgetRenderer
	iconOffsetY float32
}

func newActionButton(label string, action func()) *actionButton {
	return &actionButton{Button: widget.NewButton(label, action)}
}

func newActionButtonWithIcon(label string, icon fyne.Resource, action func()) *actionButton {
	// The refresh glyph sits slightly above the optical center of its box.
	return &actionButton{Button: widget.NewButtonWithIcon(label, icon, action), iconOffsetY: 1.0}
}

func (b *actionButton) Cursor() desktop.Cursor {
	return desktop.PointerCursor
}

// CreateRenderer keeps the visible renderer so Resize can lay it out. The
// embedded Button creates a separate renderer for itself; without this the
// button keeps its old width when the window is resized.
func (b *actionButton) CreateRenderer() fyne.WidgetRenderer {
	b.renderer = b.Button.CreateRenderer()
	return b.renderer
}

// layout resets the icon and label before laying out the visible renderer.
// The embedded Button owns a second renderer that shares those two objects;
// its Layout adds the centering offset to their current positions. Without the
// reset, every resize/refresh pushes the label down again and the icon and text
// stop sharing a center line.
func (b *actionButton) layout(size fyne.Size) {
	if b.renderer == nil {
		return
	}
	for _, object := range b.renderer.Objects() {
		switch object.(type) {
		case *widget.RichText, *canvas.Image:
			object.Move(fyne.NewPos(0, 0))
		}
	}
	b.renderer.Layout(size)
	if b.iconOffsetY == 0 {
		return
	}
	for _, object := range b.renderer.Objects() {
		if icon, ok := object.(*canvas.Image); ok {
			icon.Move(icon.Position().Add(fyne.NewPos(0, b.iconOffsetY)))
		}
	}
}

func (b *actionButton) Resize(size fyne.Size) {
	b.Button.Resize(size)
	b.layout(size)
}

func (b *actionButton) Refresh() {
	b.Button.Refresh()
	b.sync()
}

func (b *actionButton) SetText(text string) {
	b.Button.SetText(text)
	b.sync()
}

func (b *actionButton) Disable() {
	b.Button.Disable()
	b.sync()
}

func (b *actionButton) Enable() {
	b.Button.Enable()
	b.sync()
}

func (b *actionButton) sync() {
	if b.renderer == nil {
		return
	}
	b.renderer.Refresh()
	b.layout(b.Button.Size())
}

var _ desktop.Cursorable = (*actionButton)(nil)

// disclosure is a plain expanding section. The stock accordion draws a card
// and its headers keep the default arrow cursor; this keeps the section in the
// panel's type hierarchy and makes the whole header actionable.
type disclosure struct {
	widget.BaseWidget

	header *actionButton
	body   *fyne.Container
	root   *fyne.Container
	open   bool
}

func newDisclosure(title string, content fyne.CanvasObject) *disclosure {
	section := &disclosure{
		body: vbox(0, content),
	}
	section.header = newActionButton(title, section.toggle)
	section.header.Alignment = widget.ButtonAlignLeading
	section.header.IconPlacement = widget.ButtonIconLeadingText
	section.header.Icon = theme.MenuDropDownIcon()
	section.header.Importance = widget.LowImportance
	section.body.Hide()
	section.root = vbox(0, section.header, section.body)
	section.ExtendBaseWidget(section)
	return section
}

func (d *disclosure) toggle() {
	d.open = !d.open
	if d.open {
		d.header.Icon = theme.MenuDropUpIcon()
		d.body.Show()
	} else {
		d.header.Icon = theme.MenuDropDownIcon()
		d.body.Hide()
	}
	d.header.Refresh()
	d.body.Refresh()
	d.Refresh()
}

// MinSize ignores the collapsed body, whose long Steam paths would otherwise
// widen the whole session stage even while hidden.
func (d *disclosure) MinSize() fyne.Size {
	d.ExtendBaseWidget(d)
	if !d.open {
		return d.header.MinSize()
	}
	return d.root.MinSize()
}

func (d *disclosure) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(d.root)
}

func primaryButton(label string, action func()) *actionButton {
	button := newActionButton(label, action)
	button.Importance = widget.HighImportance
	return button
}

// guideFigure renders one guide image with a title and an explanatory caption.
// The embedded images are documented placeholders until real Steam captures
// replace them.
func guideFigure(title string, resource fyne.Resource, caption string) fyne.CanvasObject {
	titleLabel := sectionLabel(title)
	var visual fyne.CanvasObject
	if resource != nil {
		image := canvas.NewImageFromResource(resource)
		image.FillMode = canvas.ImageFillContain
		image.SetMinSize(fyne.NewSize(1, 160))
		visual = tall(image, 160)
	} else {
		background := canvas.NewRectangle(colorSurfaceRaised)
		background.CornerRadius = 6
		visual = tall(container.NewStack(background, container.NewCenter(captionWrapLabel(caption))), 160)
	}
	objects := []fyne.CanvasObject{titleLabel, visual}
	if caption != "" {
		objects = append(objects, captionWrapLabel(caption))
	}
	return vbox(6, objects...)
}
