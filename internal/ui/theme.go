package ui

import (
	"embed"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// The interface borrows its palette from the application icon: a deep ink
// ground, warm signal amber and oxidized teal. Everything else is neutral.
var (
	colorInk           = color.NRGBA{R: 0x0B, G: 0x14, B: 0x18, A: 0xFF}
	colorSessionField  = color.NRGBA{R: 0x0A, G: 0x1B, B: 0x1D, A: 0xFF}
	colorSurface       = color.NRGBA{R: 0x11, G: 0x1F, B: 0x24, A: 0xFF}
	colorSurfaceRaised = color.NRGBA{R: 0x16, G: 0x28, B: 0x2E, A: 0xFF}
	colorSurfaceQuiet  = color.NRGBA{R: 0x0C, G: 0x17, B: 0x1B, A: 0xFF}
	colorSelected      = color.NRGBA{R: 0x2A, G: 0x21, B: 0x18, A: 0xFF}
	colorHover         = color.NRGBA{R: 0x24, G: 0x1D, B: 0x15, A: 0xFF}
	colorLine          = color.NRGBA{R: 0x24, G: 0x3A, B: 0x41, A: 0xFF}
	colorLineSoft      = color.NRGBA{R: 0x1A, G: 0x2C, B: 0x32, A: 0xFF}
	colorText          = color.NRGBA{R: 0xF2, G: 0xF5, B: 0xF4, A: 0xFF}
	colorMuted         = color.NRGBA{R: 0x8C, G: 0xA0, B: 0xA6, A: 0xFF}
	colorFaint         = color.NRGBA{R: 0x84, G: 0x9B, B: 0xA2, A: 0xFF}
	colorAmber         = color.NRGBA{R: 0xF4, G: 0xA2, B: 0x61, A: 0xFF}
	colorTeal          = color.NRGBA{R: 0x2A, G: 0x9D, B: 0x8F, A: 0xFF}
	colorSeam          = color.NRGBA{R: 0x1B, G: 0x3A, B: 0x3A, A: 0xFF}
	colorMint          = color.NRGBA{R: 0x7B, G: 0xD6, B: 0xA8, A: 0xFF}
	colorCyan          = color.NRGBA{R: 0x7F, G: 0xC8, B: 0xD8, A: 0xFF}
	colorCoral         = color.NRGBA{R: 0xEE, G: 0x7B, B: 0x66, A: 0xFF}
	colorGold          = color.NRGBA{R: 0xE3, G: 0xB3, B: 0x4F, A: 0xFF}
)

// Custom type sizes used by the library and the session stage.
const (
	sizeNameRowTitle   fyne.ThemeSizeName = "rowTitle"
	sizeNameFieldValue fyne.ThemeSizeName = "fieldValue"
)

//go:embed assets/fonts/*.ttf
var fontFS embed.FS

//go:embed assets/better-rpt.png
var appIconData []byte

//go:embed assets/better-rpt-small.png
var appIconSmallData []byte

var (
	fontSans           = loadFont("assets/fonts/IBMPlexSans-Regular.ttf")
	fontSansBold       = loadFont("assets/fonts/IBMPlexSans-SemiBold.ttf")
	fontSansItalic     = loadFont("assets/fonts/IBMPlexSans-Italic.ttf")
	fontSansBoldItalic = loadFont("assets/fonts/IBMPlexSans-SemiBoldItalic.ttf")
	fontMono           = loadFont("assets/fonts/IBMPlexMono-Regular.ttf")
	appIcon            = fyne.NewStaticResource("better-rpt.png", appIconData)
	appIconSmall       = fyne.NewStaticResource("better-rpt-small.png", appIconSmallData)
)

func loadFont(path string) fyne.Resource {
	data, err := fontFS.ReadFile(path)
	if err != nil {
		panic("ui: embedded font missing: " + path)
	}
	return fyne.NewStaticResource(path, data)
}

// controlTheme is a fixed dark theme. The app is a full-screen companion to
// Steam, so it deliberately ignores the desktop light/dark preference.
type controlTheme struct {
	fallback fyne.Theme
}

func newControlTheme() fyne.Theme {
	return &controlTheme{fallback: theme.DefaultTheme()}
}

func (t *controlTheme) Color(name fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return colorInk
	case theme.ColorNameForeground:
		return colorText
	case theme.ColorNameForegroundOnPrimary, theme.ColorNameForegroundOnSuccess,
		theme.ColorNameForegroundOnWarning, theme.ColorNameForegroundOnError:
		return colorInk
	case theme.ColorNameButton:
		return colorSurfaceRaised
	case theme.ColorNameDisabledButton:
		return colorSurfaceQuiet
	case theme.ColorNameDisabled:
		return colorMuted
	case theme.ColorNamePrimary, theme.ColorNameHyperlink:
		return colorAmber
	case theme.ColorNameFocus:
		return colorAmber
	case theme.ColorNameSelection:
		return colorSelected
	case theme.ColorNameHover:
		return colorHover
	case theme.ColorNamePressed:
		return color.NRGBA{R: 0x38, G: 0x2A, B: 0x1C, A: 0xFF}
	case theme.ColorNameInputBackground:
		return colorSurfaceQuiet
	case theme.ColorNameInputBorder:
		return colorLine
	case theme.ColorNameHeaderBackground:
		return colorSurface
	case theme.ColorNameMenuBackground:
		return colorSurface
	case theme.ColorNameOverlayBackground:
		return color.NRGBA{R: 0x04, G: 0x09, B: 0x0B, A: 0xD6}
	case theme.ColorNameShadow:
		return color.NRGBA{A: 0x66}
	case theme.ColorNameSeparator:
		return colorLineSoft
	case theme.ColorNameScrollBar:
		return color.NRGBA{R: 0x2F, G: 0x46, B: 0x4D, A: 0xFF}
	case theme.ColorNameScrollBarBackground:
		return color.Transparent
	case theme.ColorNamePlaceHolder:
		return colorFaint
	case theme.ColorNameSuccess:
		return colorMint
	case theme.ColorNameWarning:
		return colorGold
	case theme.ColorNameError:
		return colorCoral
	case theme.ColorNameInnerWindowBorder, theme.ColorNameInnerWindowBorderInactive:
		return colorLine
	default:
		return t.fallback.Color(name, theme.VariantDark)
	}
}

func (t *controlTheme) Font(style fyne.TextStyle) fyne.Resource {
	switch {
	case style.Symbol:
		return t.fallback.Font(style)
	case style.Monospace:
		return fontMono
	case style.Bold && style.Italic:
		return fontSansBoldItalic
	case style.Bold:
		return fontSansBold
	case style.Italic:
		return fontSansItalic
	default:
		return fontSans
	}
}

func (t *controlTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return t.fallback.Icon(name)
}

func (t *controlTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNameText:
		return 15
	case theme.SizeNameHeadingText:
		return 26
	case theme.SizeNameSubHeadingText:
		return 16
	case theme.SizeNameCaptionText:
		return 12
	case sizeNameRowTitle:
		return 17
	case sizeNameFieldValue:
		return 14
	case theme.SizeNamePadding:
		return 12
	case theme.SizeNameInnerPadding:
		return 10
	case theme.SizeNameLineSpacing:
		return 5
	case theme.SizeNameInlineIcon:
		return 18
	case theme.SizeNameInputBorder:
		return 1
	case theme.SizeNameInputRadius:
		return 4
	case theme.SizeNameButtonRadius:
		return 4
	case theme.SizeNameSelectionRadius:
		return 3
	case theme.SizeNameCardRadius:
		return 8
	case theme.SizeNameDialogRadius:
		return 8
	case theme.SizeNamePopupRadius:
		return 8
	case theme.SizeNameMenuRadius:
		return 6
	case theme.SizeNameSeparatorThickness:
		return 1
	case theme.SizeNameScrollBar:
		return 8
	case theme.SizeNameScrollBarSmall:
		return 4
	case theme.SizeNameScrollBarRadius:
		return 4
	case theme.SizeNameSplitThickness:
		return 8
	case theme.SizeNameWindowTitleBarHeight:
		return 32
	default:
		return t.fallback.Size(name)
	}
}

func toneColor(value tone) color.Color {
	switch value {
	case toneSuccess:
		return colorMint
	case toneWarning:
		return colorGold
	case toneError:
		return colorCoral
	case toneInfo:
		return colorCyan
	default:
		return colorMuted
	}
}

func tint(value color.Color, alpha uint8) color.NRGBA {
	r, g, b, _ := value.RGBA()
	return color.NRGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: alpha}
}
