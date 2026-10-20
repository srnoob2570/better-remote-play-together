//go:build gtk3 && !production

package main

import (
	"io/fs"
	"os"
)

// Development builds read the generated frontend from disk. `wails3 dev`
// proxies requests to the Vite server before this filesystem is consulted.
var assets fs.FS = os.DirFS("cmd/better-rpt/dist")
