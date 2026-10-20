package ui

import (
	"math"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	fynetest "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/srnoob2570/better-rpt/internal/domain"
)

// textBaseline returns the y coordinate of the first text line inside a
// widget.Label, relative to the label itself. RichText draws its first line at
// the label's inner padding, and the driver reports the distance from that
// line's top to its baseline.
func textBaseline(label *widget.Label) float32 {
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

func testApp(t *testing.T) fyne.App {
	t.Helper()
	app := fynetest.NewApp()
	app.Settings().SetTheme(newControlTheme())
	t.Cleanup(app.Quit)
	return app
}

func TestGameRowTextCentersOnHighlight(t *testing.T) {
	testApp(t)

	row := newGameRow()
	row.SetGame(domain.Game{
		Name:   "HELLDIVERS™ 2",
		Kind:   domain.KindProton,
		Status: domain.StatusUnknown,
	}, true, func(key string) string { return key }, "")
	row.Resize(fyne.NewSize(700, 40))
	row.root.Layout.Layout(row.root.Objects, row.root.Size())

	// The selected highlight is inset 4px at the bottom of each 40px row.
	highlightCenter := float32(18)
	for name, label := range map[string]*widget.Label{
		"title":  row.name,
		"kind":   row.kind,
		"status": row.status,
	} {
		center := label.Position().Y + label.MinSize().Height/2
		if math.Abs(float64(center-highlightCenter)) > 1 {
			t.Fatalf("%s center=%.1f, highlight center=%.1f", name, center, highlightCenter)
		}
	}
	iconCenter := row.icon.Position().Y + row.icon.Size().Height/2
	if math.Abs(float64(iconCenter-highlightCenter)) > 1 {
		t.Fatalf("icon center=%.1f, highlight center=%.1f", iconCenter, highlightCenter)
	}
}

func TestFieldRowsShareBaseline(t *testing.T) {
	testApp(t)

	key := captionLabel("Proton:")
	value := fieldValueLabel("Sí")
	field := container.New(&fieldLayout{labelWidth: 58}, key, value)
	field.Resize(fyne.NewSize(200, 24))
	field.Layout.Layout(field.Objects, field.Size())

	keyBaseline := key.Position().Y + textBaseline(key)
	valueBaseline := value.Position().Y + textBaseline(value)
	if math.Abs(float64(keyBaseline-valueBaseline)) > 1 {
		t.Fatalf("field baselines: key=%.1f value=%.1f", keyBaseline, valueBaseline)
	}
}

func TestSessionHeaderTextCenters(t *testing.T) {
	testApp(t)

	tabs := newSessionTabGroup([]string{"Sesión", "Detalles"}, 0, nil)
	hint := captionLabel("Remote Play Together no está confirmado.")
	hint.Wrapping = fyne.TextWrapWord
	hint.Show()
	header := container.New(&sessionHeaderLayout{gap: 24}, tabs.obj, hint)
	header.Resize(fyne.NewSize(440, 30))
	header.Layout.Layout(header.Objects, header.Size())
	tabs.obj.Layout.Layout(tabs.obj.Objects, tabs.obj.Size())

	tab := tabs.tabs[0]
	fynetest.WidgetRenderer(tab).Layout(tab.Size())
	tabCenter := tab.Position().Y + tab.label.Position().Y + tab.label.MinSize().Height/2
	hintCenter := hint.Position().Y + labelFirstLineCenter(hint)
	if math.Abs(float64(tabCenter-hintCenter)) > 1 {
		t.Fatalf("session header text centers: tab=%.1f hint=%.1f", tabCenter, hintCenter)
	}
}
