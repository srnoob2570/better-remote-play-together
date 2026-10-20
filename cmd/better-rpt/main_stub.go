//go:build !gtk3

package main

import (
	"fmt"
	"os"

	"github.com/srnoob2570/better-rpt/internal/diagnostics"
)

// Wails v3 defaults to GTK4, which is not available on the Ubuntu 22.04
// baseline. Keep the default `go test ./...` free of the GTK toolchain and
// make an accidental untagged build fail loudly instead of producing a binary
// that cannot open a window. Diagnostics is pure Go and remains available.
func main() {
	if handled, code := diagnostics.Main(os.Args[1:], os.Stdout, os.Stderr, "dev"); handled {
		os.Exit(code)
	}
	fmt.Fprintln(os.Stderr, "better-rpt: build with -tags gtk3 (see docs/build.md)")
	os.Exit(1)
}
