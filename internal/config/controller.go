package config

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	controllerConfigRoot    = "controller_config"
	controllerConfigSetGlob = "configset_controller_"
	controllerConfigSetExt  = ".vdf"
	controllerMobileTouch   = "mobile_touch"
	controllerAppID         = "480"
	controllerTemplateKey   = "template"
	controllerTemplateTail  = "_gamepad_joystick.vdf"
)

// ControllerLayoutResult describes the idempotent SpaceWar Steam Input pass.
// Applied lists configsets that gained a 480 entry; Already lists ones that
// already pointed at the expected template; Existing lists ones left alone
// because SpaceWar already had a custom/autosave entry.
type ControllerLayoutResult struct {
	Applied  []string
	Already  []string
	Existing []string
	Skipped  []string
	Warnings []error
}

// EnsureGenericControllerLayouts makes SpaceWar use the generic gamepad
// template for physical controller types. It only adds a missing 480 entry and
// never overwrites an existing custom layout. Steam may cache the selection
// while it is running; the entry is still durable for the next reload.
func EnsureGenericControllerLayouts(steamRoot string) (ControllerLayoutResult, error) {
	var result ControllerLayoutResult
	if steamRoot == "" {
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
		outcome, err := ensureControllerConfigSet(filepath.Join(configDir, name), template)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Errorf("%s: %w", name, err))
			continue
		}
		switch outcome {
		case controllerConfigApplied:
			result.Applied = append(result.Applied, name)
		case controllerConfigAlready:
			result.Already = append(result.Already, name)
		case controllerConfigExisting:
			result.Existing = append(result.Existing, name)
		}
	}
	return result, nil
}

type controllerConfigOutcome uint8

const (
	controllerConfigApplied controllerConfigOutcome = iota
	controllerConfigAlready
	controllerConfigExisting
)

func ensureControllerConfigSet(path, template string) (controllerConfigOutcome, error) {
	data, mode, err := readLocalConfig(path)
	if err != nil {
		return 0, err
	}
	document, err := parseVDFDocument(data)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrControllerConfig, err)
	}
	root, err := controllerConfigObject(document)
	if err != nil {
		return 0, err
	}
	if entry := findVDFChild(root, controllerAppID); entry != nil {
		if value, ok := scalarVDFChild(entry, controllerTemplateKey); ok && value == template {
			return controllerConfigAlready, nil
		}
		return controllerConfigExisting, nil
	}
	updated, err := insertControllerTemplate(data, root.Object, template)
	if err != nil {
		return 0, err
	}
	if err := verifyControllerTemplate(updated, template); err != nil {
		return 0, err
	}
	if err := writeLocalConfig(path, updated, mode); err != nil {
		return 0, err
	}
	return controllerConfigApplied, nil
}

func controllerConfigObject(document *vdfDocument) (*vdfNode, error) {
	var root *vdfNode
	for _, child := range document.Root.Children {
		if child.Key != controllerConfigRoot {
			continue
		}
		if root != nil {
			return nil, fmt.Errorf("%w: duplicate %q object", ErrControllerConfig, controllerConfigRoot)
		}
		if child.Object == nil {
			return nil, fmt.Errorf("%w: %q is not an object", ErrControllerConfig, controllerConfigRoot)
		}
		root = child
	}
	if root == nil {
		return nil, fmt.Errorf("%w: missing %q object", ErrControllerConfig, controllerConfigRoot)
	}
	return root, nil
}

func findVDFChild(object *vdfNode, key string) *vdfNode {
	if object == nil || object.Object == nil {
		return nil
	}
	for _, child := range object.Object.Children {
		if child.Key == key {
			return child
		}
	}
	return nil
}

func scalarVDFChild(object *vdfNode, key string) (string, bool) {
	child := findVDFChild(object, key)
	if child == nil || child.Object != nil {
		return "", false
	}
	return child.Value, true
}

func insertControllerTemplate(data []byte, object *vdfObject, template string) ([]byte, error) {
	if object == nil || object.Close < 0 || object.Close > len(data) {
		return nil, fmt.Errorf("%w: invalid %q object", ErrControllerConfig, controllerConfigRoot)
	}
	indent := "\t"
	if len(object.Children) > 0 {
		start := object.Children[0].KeyStart
		lineStart := bytes.LastIndexByte(data[:start], '\n') + 1
		if candidate := string(data[lineStart:start]); candidate != "" && strings.Trim(candidate, " \t") == "" {
			indent = candidate
		}
	}
	block := fmt.Sprintf("%s%q\n%s{\n%s%q\t\t%q\n%s}\n",
		indent, controllerAppID,
		indent,
		indent+"\t", controllerTemplateKey, template,
		indent,
	)
	at := object.Close
	prefix := ""
	if at > 0 && data[at-1] != '\n' {
		prefix = "\n"
	}
	updated := make([]byte, 0, len(data)+len(block)+len(prefix))
	updated = append(updated, data[:at]...)
	updated = append(updated, prefix...)
	updated = append(updated, block...)
	updated = append(updated, data[at:]...)
	return updated, nil
}

func verifyControllerTemplate(data []byte, template string) error {
	document, err := parseVDFDocument(data)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrControllerConfigVerify, err)
	}
	root, err := controllerConfigObject(document)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrControllerConfigVerify, err)
	}
	entry := findVDFChild(root, controllerAppID)
	value, ok := scalarVDFChild(entry, controllerTemplateKey)
	if !ok || value != template {
		return fmt.Errorf("%w: %s template missing after edit", ErrControllerConfigVerify, controllerAppID)
	}
	return nil
}
