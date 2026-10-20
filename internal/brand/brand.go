// Package brand exposes the application artwork shared by the native shell.
package brand

import _ "embed"

// Icon is the application icon used by the window manager and the about box.
//
//go:embed better-rpt.png
var Icon []byte
