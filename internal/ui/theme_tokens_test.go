package ui

import (
	"image/color"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

var cssPropertyPattern = regexp.MustCompile(`(?m)^\s*(--[a-z-]+):\s*([^;]+);`)

func TestDesignTokensMatchTheme(t *testing.T) {
	data, err := os.ReadFile("../../design/tokens.css")
	if err != nil {
		t.Fatalf("read design tokens: %v", err)
	}
	properties := make(map[string]string)
	for _, match := range cssPropertyPattern.FindAllStringSubmatch(string(data), -1) {
		properties[match[1]] = strings.TrimSpace(match[2])
	}

	colors := map[string]color.NRGBA{
		"ink":       colorInk,
		"session":   colorSessionField,
		"surface":   colorSurface,
		"raised":    colorSurfaceRaised,
		"quiet":     colorSurfaceQuiet,
		"selected":  colorSelected,
		"hover":     colorHover,
		"line":      colorLine,
		"line-soft": colorLineSoft,
		"cream":     colorText,
		"muted-ink": colorMuted,
		"faint":     colorFaint,
		"amber":     colorAmber,
		"teal":      colorTeal,
		"seam":      colorSeam,
		"mint":      colorMint,
		"cyan":      colorCyan,
		"coral":     colorCoral,
		"gold":      colorGold,
	}
	for name, want := range colors {
		got, ok := properties["--color-"+name]
		if !ok {
			t.Errorf("design tokens are missing --color-%s", name)
			continue
		}
		if got != colorHex(want) {
			t.Errorf("--color-%s = %q, want %q", name, got, colorHex(want))
		}
	}

	sizes := map[string]float32{
		"text":                15,
		"heading":             26,
		"subheading":          16,
		"caption":             12,
		"row-title":           17,
		"field-value":         14,
		"padding":             12,
		"inner-padding":       10,
		"line-spacing":        5,
		"inline-icon":         18,
		"input-border":        1,
		"input-radius":        4,
		"button-radius":       4,
		"selection-radius":    3,
		"card-radius":         8,
		"dialog-radius":       8,
		"popup-radius":        8,
		"menu-radius":         6,
		"separator-thickness": 1,
		"scrollbar":           8,
		"scrollbar-small":     4,
		"scrollbar-radius":    4,
		"split-thickness":     8,
		"titlebar-height":     32,
	}
	for name, want := range sizes {
		got, ok := properties["--size-"+name]
		if !ok {
			t.Errorf("design tokens are missing --size-%s", name)
			continue
		}
		value, err := strconv.ParseFloat(strings.TrimSuffix(got, "px"), 32)
		if err != nil {
			t.Errorf("--size-%s = %q: %v", name, got, err)
			continue
		}
		if float32(value) != want {
			t.Errorf("--size-%s = %v, want %v", name, value, want)
		}
	}

	if got := properties["--font-ui"]; !strings.HasPrefix(got, `"IBM Plex Sans"`) {
		t.Errorf("--font-ui = %q, want IBM Plex Sans", got)
	}
	if got := properties["--font-code"]; !strings.HasPrefix(got, `"IBM Plex Mono"`) {
		t.Errorf("--font-code = %q, want IBM Plex Mono", got)
	}
}

func colorHex(value color.NRGBA) string {
	const digits = "0123456789abcdef"
	buf := []byte{'#', 0, 0, 0, 0, 0, 0}
	for i, part := range []uint8{value.R, value.G, value.B} {
		buf[1+i*2] = digits[part>>4]
		buf[2+i*2] = digits[part&0x0f]
	}
	return string(buf)
}
