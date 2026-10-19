package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/srnoob2570/better-rpt/internal/domain"
	"github.com/srnoob2570/better-rpt/internal/fsutil"
	"github.com/srnoob2570/better-rpt/internal/i18n"
)

// Config is the persistent user preference file. An empty Language means
// "follow the system locale"; it is resolved at startup by the app service.
type Config struct {
	Language string `json:"language,omitempty"`
}

// Default returns the empty configuration.
func Default() Config {
	return Config{}
}

// Load reads config.json. A missing file is not an error and yields defaults.
func Load(paths Paths) (Config, error) {
	data, err := os.ReadFile(paths.ConfigFile)
	if errors.Is(err, fs.ErrNotExist) {
		return Default(), nil
	}
	if err != nil {
		return Default(), fmt.Errorf("read %s: %w", paths.ConfigFile, err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Default(), fmt.Errorf("parse %s: %w", paths.ConfigFile, err)
	}
	if cfg.Language != "" && !i18n.Supported(cfg.Language) {
		cfg.Language = ""
	}
	return cfg, nil
}

// Save writes config.json atomically with mode 0600.
func Save(paths Paths, cfg Config) error {
	if cfg.Language != "" && !i18n.Supported(cfg.Language) {
		return fmt.Errorf("%w: language %q", domain.ErrUnsupported, cfg.Language)
	}
	if err := paths.Ensure(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode configuration: %w", err)
	}
	data = append(data, '\n')
	if err := fsutil.AtomicWriteFile(paths.ConfigFile, data, 0o600); err != nil {
		return fmt.Errorf("save configuration: %w", err)
	}
	return nil
}
