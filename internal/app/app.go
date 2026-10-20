package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/srnoob2570/better-rpt/internal/config"
	"github.com/srnoob2570/better-rpt/internal/domain"
	"github.com/srnoob2570/better-rpt/internal/i18n"
	"github.com/srnoob2570/better-rpt/internal/steam"
)

// Service is the application-layer entry point. The UI consumes it; it does
// not import Fyne and can be exercised without a display server.
type Service struct {
	paths config.Paths
	log   *slog.Logger

	mu         sync.RWMutex
	cfg        config.Config
	tr         *i18n.Translator
	lock       *instanceLock
	catalog    steam.Catalog
	hasCatalog bool
	sessions   sessionRunner
}

// New resolves the user paths, loads the configuration and takes the
// single-instance lock. It refuses to run as root.
func New(logger *slog.Logger) (*Service, error) {
	if err := domain.RefuseRoot(os.Geteuid()); err != nil {
		return nil, err
	}
	paths, err := config.ResolvePaths()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrPaths, err)
	}
	return NewWithPaths(paths, logger)
}

// NewWithPaths is the testable constructor. It does not check the effective
// uid; the command entry points own that policy.
func NewWithPaths(paths config.Paths, logger *slog.Logger) (*Service, error) {
	if logger == nil {
		logger = slog.Default()
	}
	if err := paths.Ensure(); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrPaths, err)
	}
	lock, err := acquireInstanceLock(paths.RuntimeDir)
	if err != nil {
		return nil, err
	}
	cfg, err := config.Load(paths)
	if err != nil {
		lock.Close()
		return nil, fmt.Errorf("%w: %v", domain.ErrConfig, err)
	}
	lang := cfg.Language
	if lang == "" {
		lang = string(i18n.Detect())
	}
	svc := &Service{
		paths: paths,
		log:   logger,
		cfg:   cfg,
		tr:    i18n.New(lang),
		lock:  lock,
	}
	logger.Info("better-rpt started", "config", paths.ConfigFile, "language", svc.tr.Language())
	return svc, nil
}

// Close releases the single-instance lock. It is safe to call more than once.
func (s *Service) Close() error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.lock == nil {
		return nil
	}
	err := s.lock.Close()
	s.lock = nil
	return err
}

// Paths returns the immutable user paths.
func (s *Service) Paths() config.Paths {
	return s.paths
}

// Config returns a copy of the current configuration.
func (s *Service) Config() config.Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg
}

// Language returns the effective UI language.
func (s *Service) Language() i18n.Language {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.tr.Language()
}

// T translates a UI key with the current language.
func (s *Service) T(key string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.tr.T(key)
}

// RefreshCatalog detects Steam and rebuilds the installed-game catalog.
// Library and manifest warnings are returned in the catalog and logged.
func (s *Service) RefreshCatalog(ctx context.Context) (steam.Catalog, error) {
	catalog, err := steam.Open(ctx)
	if err != nil {
		return steam.Catalog{}, err
	}
	layout, layoutErr := config.EnsureGenericControllerLayouts(catalog.Installation.Root)
	if layoutErr != nil {
		s.log.Warn("controller layout", "error", layoutErr)
		catalog.Warnings = append(catalog.Warnings, steam.Warning{Path: "Steam controller config", Err: layoutErr})
	} else {
		if len(layout.Applied) > 0 {
			s.log.Info("controller layout applied", "configsets", layout.Applied)
		}
		if len(layout.Existing) > 0 {
			s.log.Warn("controller layout kept", "configsets", layout.Existing)
			catalog.Warnings = append(catalog.Warnings, steam.Warning{
				Path: "Steam controller config",
				Err:  fmt.Errorf("SpaceWar already has a controller layout for %v; it was not changed", layout.Existing),
			})
		}
		if len(layout.Skipped) > 0 {
			s.log.Debug("controller layout skipped", "configsets", layout.Skipped)
		}
		for _, warning := range layout.Warnings {
			s.log.Warn("controller layout", "error", warning)
			catalog.Warnings = append(catalog.Warnings, steam.Warning{Path: "Steam controller config", Err: warning})
		}
	}
	s.mu.Lock()
	s.catalog = catalog
	s.hasCatalog = true
	s.mu.Unlock()
	for _, warning := range catalog.Warnings {
		s.log.Warn("steam catalog warning", "path", warning.Path, "error", warning.Err)
	}
	s.log.Info("steam catalog refreshed", "libraries", len(catalog.Libraries), "games", len(catalog.Games), "warnings", len(catalog.Warnings))
	return catalog, nil
}

// Catalog returns the last successful refresh and whether one exists.
func (s *Service) Catalog() (steam.Catalog, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.catalog, s.hasCatalog
}

// SetLanguage validates, persists and applies a new UI language.
func (s *Service) SetLanguage(lang string) error {
	if !i18n.Supported(lang) {
		return fmt.Errorf("%w: language %q", domain.ErrUnsupported, lang)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	cfg := s.cfg
	cfg.Language = lang
	if err := config.Save(s.paths, cfg); err != nil {
		return err
	}
	s.cfg = cfg
	s.tr = i18n.New(lang)
	s.log.Info("language changed", "language", lang)
	return nil
}
