//go:build gtk3

package main

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/srnoob2570/better-rpt/internal/app"
	"github.com/srnoob2570/better-rpt/internal/brand"
	"github.com/srnoob2570/better-rpt/internal/bridge"
	"github.com/srnoob2570/better-rpt/internal/domain"
	"github.com/srnoob2570/better-rpt/internal/i18n"
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

	if err := runGUI(svc, logger); err != nil {
		logger.Error("gui failed", "error", err)
		return 1
	}
	return 0
}

func runGUI(svc *app.Service, logger *slog.Logger) error {
	logger.Info("gui starting", "version", version)

	emitter := &wailsEmitter{}
	b := bridge.New(svc, version, emitter)
	wailsApp := application.New(application.Options{
		Name:        svc.T("app.title"),
		Description: svc.T("app.subtitle"),
		Icon:        brand.Icon,
		Logger:      logger,
		LogLevel:    slog.LevelInfo,
		Services: []application.Service{
			application.NewService(b),
		},
		MarshalError: bridge.MarshalError,
		Assets: application.AssetOptions{
			Handler:    application.BundledAssetFileServer(assets),
			Middleware: bridge.ArtMiddleware(svc, logger),
		},
		OnShutdown: b.Shutdown,
	})
	emitter.app = wailsApp

	wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            svc.T("app.title"),
		Width:            1280,
		Height:           800,
		MinWidth:         1024,
		MinHeight:        600,
		InitialPosition:  application.WindowCentered,
		BackgroundColour: application.NewRGB(0x0b, 0x14, 0x18),
		Linux: application.LinuxWindow{
			Icon: brand.Icon,
		},
		URL: "/",
	})

	err := wailsApp.Run()
	logger.Info("gui stopped")
	return err
}

type wailsEmitter struct {
	app *application.App
}

func (e *wailsEmitter) Emit(name string, data any) {
	if e.app != nil {
		e.app.Event.Emit(name, data)
	}
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
