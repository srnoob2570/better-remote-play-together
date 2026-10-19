package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/srnoob2570/better-rpt/internal/domain"
)

type statusChip struct {
	text *canvas.Text
	rect *canvas.Rectangle
	obj  *fyne.Container
}

func newStatusChip(label string, tone tone) *statusChip {
	foreground := toneColor(tone)
	text := canvas.NewText(label, foreground)
	text.TextSize = 12
	text.Alignment = fyne.TextAlignCenter
	rect := canvas.NewRectangle(tint(foreground, 0x30))
	rect.CornerRadius = 8
	rect.SetMinSize(fyne.NewSize(104, 24))
	chip := &statusChip{
		text: text,
		rect: rect,
		obj:  container.NewStack(rect, container.NewCenter(text)),
	}
	return chip
}

func (c *statusChip) Set(label string, tone tone) {
	foreground := toneColor(tone)
	c.text.Text = label
	c.text.Color = foreground
	c.rect.FillColor = tint(foreground, 0x30)
	c.text.Refresh()
	c.rect.Refresh()
	c.obj.Refresh()
}

func tint(value color.Color, alpha uint8) color.NRGBA {
	r, g, b, _ := value.RGBA()
	return color.NRGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: alpha}
}

func toneColor(value tone) color.Color {
	switch value {
	case toneSuccess:
		return theme.SuccessColor()
	case toneWarning:
		return theme.WarningColor()
	case toneError:
		return theme.ErrorColor()
	case toneInfo:
		return theme.PrimaryColor()
	default:
		return theme.ForegroundColor()
	}
}

type gameRow struct {
	widget.BaseWidget

	name     *widget.Label
	kind     *widget.Label
	chip     *statusChip
	selected *canvas.Rectangle
	root     *fyne.Container
}

func newGameRow() *gameRow {
	row := &gameRow{
		name: widget.NewLabel(""),
		kind: widget.NewLabel(""),
		chip: newStatusChip("", toneNeutral),
	}
	row.name.TextStyle = fyne.TextStyle{Bold: true}
	row.name.Truncation = fyne.TextTruncateEllipsis
	row.kind.Importance = widget.LowImportance
	row.selected = canvas.NewRectangle(color.NRGBA{A: 0})
	row.selected.Hide()
	right := container.NewHBox(row.kind, row.chip.obj)
	content := container.NewBorder(nil, nil, nil, right, container.NewPadded(row.name))
	row.root = container.NewStack(row.selected, content)
	row.ExtendBaseWidget(row)
	return row
}

func (r *gameRow) SetGame(game domain.Game, selected bool, translate func(string) string) {
	r.name.SetText(game.Name)
	r.kind.SetText(translate(kindLabelKey(game.Kind)))
	r.chip.Set(translate(statusLabelKey(game.Status)), statusTone(game.Status))
	if selected {
		r.selected.FillColor = tint(theme.PrimaryColor(), 0x20)
		r.selected.Show()
	} else {
		r.selected.Hide()
	}
	r.selected.Refresh()
	r.Refresh()
}

func (r *gameRow) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(r.root)
}
