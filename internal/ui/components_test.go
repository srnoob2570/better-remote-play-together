package ui

import (
	"math"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	fynetest "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// Details is a plain disclosure; the whole header is the toggle.
func TestDisclosureTogglesWithHeader(t *testing.T) {
	app := fynetest.NewApp()
	defer app.Quit()
	section := newDisclosure("Details", widget.NewLabel("body"))
	if section.body.Visible() {
		t.Fatal("details body should start collapsed")
	}
	section.header.Tapped(&fyne.PointEvent{})
	if !section.open || !section.body.Visible() {
		t.Fatal("details body should open when the header is tapped")
	}
	section.header.Tapped(&fyne.PointEvent{})
	if section.open || section.body.Visible() {
		t.Fatal("details body should close when the header is tapped again")
	}
}

// The hand cursor is part of the desktop affordance, not a nicety: Fyne's
// stock Button reports DefaultCursor, so every custom actionable widget has
// to opt in explicitly.
func TestActionableWidgetsUsePointerCursor(t *testing.T) {
	app := fynetest.NewApp()
	defer app.Quit()
	section := newDisclosure("Details", widget.NewLabel("body"))
	objects := []struct {
		name string
		obj  fyne.CanvasObject
	}{
		{name: "game row", obj: newGameRow()},
		{name: "filter tab", obj: newFilterTab("All", nil)},
		{name: "action button", obj: newActionButton("Prepare", nil)},
		{name: "disclosure header", obj: section.header},
	}
	for _, test := range objects {
		cursorable, ok := test.obj.(desktop.Cursorable)
		if !ok {
			t.Errorf("%s does not implement desktop.Cursorable", test.name)
			continue
		}
		if got := cursorable.Cursor(); got != desktop.PointerCursor {
			t.Errorf("%s cursor = %v, want pointer", test.name, got)
		}
	}
}

// Resizing and refreshing an icon button must not let the label drift away
// from the icon: the embedded Button has a second renderer sharing the same
// icon and label objects, and its layout is position-additive.
func TestActionButtonIconAndLabelStayCentered(t *testing.T) {
	app := fynetest.NewApp()
	defer app.Quit()

	button := newActionButtonWithIcon("Refresh", theme.ViewRefreshIcon(), nil)
	renderer := fynetest.WidgetRenderer(button)
	button.Resize(fyne.NewSize(140, 42))
	button.Refresh()
	button.Resize(fyne.NewSize(150, 42))
	button.Refresh()
	button.SetText("Refreshing…")

	var icon, label fyne.CanvasObject
	for _, object := range renderer.Objects() {
		switch object.(type) {
		case *canvas.Image:
			icon = object
		case *widget.RichText:
			label = object
		}
	}
	if icon == nil || label == nil {
		t.Fatalf("icon=%v label=%v", icon, label)
	}
	iconCenter := icon.Position().Y + icon.Size().Height/2
	labelCenter := label.Position().Y + label.Size().Height/2
	if math.Abs(float64(iconCenter-labelCenter)) > 1 {
		t.Fatalf("icon center %.1f, label center %.1f", iconCenter, labelCenter)
	}
}
