// better-rpt-helper runs inside Steam's donor context and launches the target
// as a child. It is intentionally headless and CGO-free.
package main

import (
	"context"
	"flag"
	"fmt"
	"math"
	"os"
	"os/signal"
	"syscall"

	"github.com/srnoob2570/better-rpt/internal/config"
	"github.com/srnoob2570/better-rpt/internal/domain"
	"github.com/srnoob2570/better-rpt/internal/i18n"
	"github.com/srnoob2570/better-rpt/internal/session"
)

var version = "dev"

func main() {
	os.Exit(run())
}

func run() int {
	flags := flag.NewFlagSet("better-rpt-helper", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	showVersion := flags.Bool("version", false, "print the version and exit")
	donor := flags.Bool("donor", false, "run as the Steam donor helper")
	target := flags.Uint64("target", 0, "target Steam app id")
	token := flags.String("session", "", "session token")
	if err := flags.Parse(os.Args[1:]); err != nil {
		return 2
	}
	if *showVersion {
		fmt.Printf("better-rpt-helper %s\n", version)
		return 0
	}
	if err := domain.RefuseRoot(os.Geteuid()); err != nil {
		fmt.Fprintln(os.Stderr, i18n.New(string(i18n.Detect())).T("error.root"))
		return 1
	}
	if !*donor {
		fmt.Fprintln(os.Stderr, "better-rpt-helper: --donor is required")
		flags.Usage()
		return 2
	}
	if *target == 0 || *target > math.MaxUint32 || uint32(*target) == domain.SpaceWarAppID {
		fmt.Fprintf(os.Stderr, "better-rpt-helper: invalid target %d\n", *target)
		return 2
	}
	if !domain.ValidSessionToken(*token) {
		fmt.Fprintln(os.Stderr, "better-rpt-helper: invalid session token")
		return 2
	}
	paths, err := config.ResolvePaths()
	if err != nil {
		fmt.Fprintln(os.Stderr, "better-rpt-helper: resolve paths:", err)
		return 2
	}
	store := session.NewStore(paths.DataDir)
	if err := store.Ensure(); err != nil {
		fmt.Fprintln(os.Stderr, "better-rpt-helper: prepare sessions:", err)
		return 2
	}
	helper, err := session.NewHelper(session.HelperOptions{
		Store:       &store,
		Token:       *token,
		TargetAppID: uint32(*target),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "better-rpt-helper:", err)
		return 2
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if _, err := helper.Run(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "better-rpt-helper:", err)
		return 2
	}
	return 0
}
