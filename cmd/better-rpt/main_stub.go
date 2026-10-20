//go:build !gtk3

package main

import (
	"fmt"
	"os"
)

// Wails v3 defaults to GTK4, which is not available on the Ubuntu 22.04
// baseline. Keep the default `go test ./...` free of the GTK toolchain and
// make an accidental untagged build fail loudly instead of producing a binary
// that cannot open a window.
func main() {
	fmt.Fprintln(os.Stderr, "better-rpt: build with -tags gtk3 (see docs/build.md)")
	os.Exit(1)
}
