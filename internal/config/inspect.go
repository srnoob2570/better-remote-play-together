package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// OverlayInspection is the read-only state of Steam's in-game overlay flag for
// SpaceWar. Exists is false when Steam has never written the flag; that means
// "unknown", not "disabled".
type OverlayInspection struct {
	Exists  bool
	Enabled bool
}

// ControllerLayoutInspection is the read-only result of checking SpaceWar's
// entry in physical controller configsets. It never writes or repairs them.
type ControllerLayoutInspection struct {
	Missing  []string
	Already  []string
	Existing []string
	Skipped  []string
	Warnings []error
}

// InspectOverlay reads SpaceWar's OverlayAppEnable flag without modifying the
// Steam configuration. It requires a readable native Steam root.
func InspectOverlay(steamRoot string) (OverlayInspection, error) {
	if strings.TrimSpace(steamRoot) == "" {
		return OverlayInspection{}, fmt.Errorf("%w: empty Steam root", ErrLocalConfigMissing)
	}
	account, err := SelectAccount(steamRoot)
	if err != nil {
		return OverlayInspection{}, err
	}
	data, _, err := readLocalConfig(account.LocalConfigPath(steamRoot))
	if err != nil {
		return OverlayInspection{}, err
	}
	document, err := parseVDFDocument(data)
	if err != nil {
		return OverlayInspection{}, err
	}
	apps, err := findObject(document, launchOptionPath)
	if err != nil {
		return OverlayInspection{}, err
	}
	if apps == nil {
		return OverlayInspection{}, fmt.Errorf("%w: apps object is missing", ErrLocalConfigUnsafe)
	}
	app, err := uniqueChild(apps, spaceWarKey)
	if err != nil {
		return OverlayInspection{}, err
	}
	if app == nil || app.Object == nil {
		return OverlayInspection{}, nil
	}
	flag, err := uniqueChild(app.Object, "OverlayAppEnable")
	if err != nil {
		return OverlayInspection{}, err
	}
	if flag == nil || flag.Object != nil {
		return OverlayInspection{}, nil
	}
	enabled, err := strconv.ParseBool(strings.TrimSpace(flag.Value))
	if err != nil {
		return OverlayInspection{}, fmt.Errorf("%w: OverlayAppEnable=%q", ErrLocalConfigUnsafe, flag.Value)
	}
	return OverlayInspection{Exists: true, Enabled: enabled}, nil
}

// InspectControllerLayouts checks the physical controller configsets for a
// SpaceWar entry. EnsureGenericControllerLayouts performs the write path; this
// function shares its VDF parsing and never calls the write path.
func InspectControllerLayouts(steamRoot string) (ControllerLayoutInspection, error) {
	var result ControllerLayoutInspection
	if strings.TrimSpace(steamRoot) == "" {
		return result, fmt.Errorf("%w: empty Steam root", ErrControllerConfig)
	}
	account, err := SelectAccount(steamRoot)
	if err != nil {
		return result, fmt.Errorf("%w: %v", ErrControllerConfig, err)
	}
	configDir := filepath.Join(
		steamRoot,
		"steamapps", "common", "Steam Controller Configs",
		strconv.FormatUint(uint64(account.AccountID), 10), "config",
	)
	entries, err := os.ReadDir(configDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return result, nil
		}
		return result, fmt.Errorf("%w: read %s: %v", ErrControllerConfig, configDir, err)
	}
	templatesDir := filepath.Join(steamRoot, "controller_base", "templates")
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() ||
			!strings.HasPrefix(name, controllerConfigSetGlob) ||
			!strings.HasSuffix(name, controllerConfigSetExt) {
			continue
		}
		controllerType := strings.TrimSuffix(strings.TrimPrefix(name, controllerConfigSetGlob), controllerConfigSetExt)
		if controllerType == controllerMobileTouch {
			continue
		}
		template := "controller_" + controllerType + controllerTemplateTail
		if _, err := os.Stat(filepath.Join(templatesDir, template)); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				result.Skipped = append(result.Skipped, name)
				continue
			}
			result.Warnings = append(result.Warnings,
				fmt.Errorf("%w: inspect template %s: %v", ErrControllerConfig, template, err))
			continue
		}
		data, _, err := readLocalConfig(filepath.Join(configDir, name))
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Errorf("%s: %w", name, err))
			continue
		}
		document, err := parseVDFDocument(data)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Errorf("%s: %w", name, err))
			continue
		}
		root, err := controllerConfigObject(document)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Errorf("%s: %w", name, err))
			continue
		}
		spaceWar := findVDFChild(root, controllerAppID)
		if spaceWar == nil {
			result.Missing = append(result.Missing, name)
			continue
		}
		value, ok := scalarVDFChild(spaceWar, controllerTemplateKey)
		if ok && value == template {
			result.Already = append(result.Already, name)
			continue
		}
		result.Existing = append(result.Existing, name)
	}
	return result, nil
}
