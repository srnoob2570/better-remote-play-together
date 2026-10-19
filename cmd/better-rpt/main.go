package main

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/srnoob2570/better-rpt/internal/app"
	"github.com/srnoob2570/better-rpt/internal/domain"
	"github.com/srnoob2570/better-rpt/internal/i18n"
	"github.com/srnoob2570/better-rpt/internal/ui"
)

var version = "dev"

func main() {
	os.Exit(run())
}

func run() int {
	flags := flag.NewFlagSet("better-rpt", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	showVersion := flags.Bool("version", false, "print the version and exit")
	if err := flags.Parse(os.Args[1:]); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintf(os.Stderr, "unexpected arguments: %v\n", flags.Args())
		flags.Usage()
		return 2
	}
	if *showVersion {
		fmt.Printf("better-rpt %s\n", version)
		return 0
	}
	if err := domain.RefuseRoot(os.Geteuid()); err != nil {
		fatal(err)
		return 1
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	svc, err := app.New(logger)
	if err != nil {
		fatal(err)
		return 1
	}
	defer func() {
		if err := svc.Close(); err != nil {
			logger.Warn("close service", "error", err)
		}
	}()

	runGUI(svc, logger)
	return 0
}

func runGUI(svc *app.Service, logger *slog.Logger) {
	logger.Info("gui starting", "version", version)
	ui.Run(svc, logger, version)
	logger.Info("gui stopped")
}

func fatal(err error) {
	tr := i18n.New(string(i18n.Detect()))
	key := "error.generic"
	switch {
	case errors.Is(err, domain.ErrRoot):
		key = "error.root"
	case errors.Is(err, domain.ErrInstanceLocked):
		key = "error.instance"
	case errors.Is(err, domain.ErrConfig):
		key = "error.config"
	case errors.Is(err, domain.ErrPaths):
		key = "error.paths"
	}
	fmt.Fprintf(os.Stderr, "%s: %v\n", tr.T(key), err)
}
