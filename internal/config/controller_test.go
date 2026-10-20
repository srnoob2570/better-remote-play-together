package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const controllerTestAccountID uint32 = 12345

func controllerTestRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	login := filepath.Join(root, "config", "loginusers.vdf")
	writeControllerTestFile(t, login, `"users"
{
	"76561197960278073"
	{
		"AccountName"		"test"
		"PersonaName"		"Test"
		"MostRecent"		"1"
		"Timestamp"		"1"
	}
}
`)
	return root
}

func controllerTestConfigDir(root string) string {
	return filepath.Join(root, "steamapps", "common", "Steam Controller Configs",
		"12345", "config")
}

func controllerTestTemplatesDir(root string) string {
	return filepath.Join(root, "controller_base", "templates")
}

func writeControllerTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create parent of %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func readControllerTestFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func TestEnsureGenericControllerLayoutsAddsMissingTemplateOnce(t *testing.T) {
	root := controllerTestRoot(t)
	configDir := controllerTestConfigDir(root)
	templatesDir := controllerTestTemplatesDir(root)

	genericPath := filepath.Join(configDir, "configset_controller_generic.vdf")
	xboxPath := filepath.Join(configDir, "configset_controller_xboxone.vdf")
	touchPath := filepath.Join(configDir, "configset_controller_mobile_touch.vdf")
	writeControllerTestFile(t, genericPath, "\"controller_config\"\n{\n\t\"123\"\n\t{\n\t\t\"autosave\"\t\t\"1\"\n\t}\n}\n")
	writeControllerTestFile(t, xboxPath, "\"controller_config\"\n{\n}\n")
	touchConfig := "\"controller_config\"\n{\n\t\"480\"\n\t{\n\t\t\"autosave\"\t\t\"1\"\n\t}\n}\n"
	writeControllerTestFile(t, touchPath, touchConfig)
	writeControllerTestFile(t, filepath.Join(templatesDir, "controller_generic_gamepad_joystick.vdf"), "\"controller_mappings\"\n{\n}\n")
	writeControllerTestFile(t, filepath.Join(templatesDir, "controller_xboxone_gamepad_joystick.vdf"), "\"controller_mappings\"\n{\n}\n")

	result, err := EnsureGenericControllerLayouts(root)
	if err != nil {
		t.Fatalf("ensure layouts: %v", err)
	}
	if len(result.Applied) != 2 {
		t.Fatalf("applied = %v, want generic and xboxone", result.Applied)
	}
	if got := readControllerTestFile(t, genericPath); !strings.Contains(got, `"480"`) ||
		!strings.Contains(got, `"controller_generic_gamepad_joystick.vdf"`) ||
		!strings.Contains(got, `"123"`) ||
		!strings.Contains(got, "\t\"480\"\n\t{\n\t\t\"template\"\t\t\"controller_generic_gamepad_joystick.vdf\"\n\t}\n") {
		t.Fatalf("generic configset missing template or unrelated entry:\n%s", got)
	}
	if got := readControllerTestFile(t, xboxPath); !strings.Contains(got, `"controller_xboxone_gamepad_joystick.vdf"`) {
		t.Fatalf("xboxone configset missing template:\n%s", got)
	}
	if got := readControllerTestFile(t, touchPath); got != touchConfig {
		t.Fatalf("mobile touch config changed:\n%s", got)
	}

	firstGeneric := readControllerTestFile(t, genericPath)
	second, err := EnsureGenericControllerLayouts(root)
	if err != nil {
		t.Fatalf("ensure layouts twice: %v", err)
	}
	if len(second.Applied) != 0 {
		t.Fatalf("second applied = %v, want none", second.Applied)
	}
	if len(second.Already) != 2 {
		t.Fatalf("second already = %v, want generic and xboxone", second.Already)
	}
	if got := readControllerTestFile(t, genericPath); got != firstGeneric {
		t.Fatalf("idempotent write changed content:\n%s", got)
	}
}

func TestEnsureGenericControllerLayoutsPreservesExistingEntry(t *testing.T) {
	root := controllerTestRoot(t)
	configDir := controllerTestConfigDir(root)
	path := filepath.Join(configDir, "configset_controller_generic.vdf")
	original := "\"controller_config\"\n{\n\t\"480\"\n\t{\n\t\t\"template\"\t\t\"custom_gamepad.vdf\"\n\t}\n}\n"
	writeControllerTestFile(t, path, original)
	writeControllerTestFile(t, filepath.Join(controllerTestTemplatesDir(root), "controller_generic_gamepad_joystick.vdf"), "\"controller_mappings\"\n{\n}\n")

	result, err := EnsureGenericControllerLayouts(root)
	if err != nil {
		t.Fatalf("ensure layouts: %v", err)
	}
	if len(result.Existing) != 1 || result.Existing[0] != "configset_controller_generic.vdf" {
		t.Fatalf("existing = %v, want generic", result.Existing)
	}
	if got := readControllerTestFile(t, path); got != original {
		t.Fatalf("existing layout was overwritten:\n%s", got)
	}
}

func TestEnsureGenericControllerLayoutsPreservesAutosave(t *testing.T) {
	root := controllerTestRoot(t)
	configDir := controllerTestConfigDir(root)
	path := filepath.Join(configDir, "configset_controller_generic.vdf")
	original := "\"controller_config\"\n{\n\t\"480\"\n\t{\n\t\t\"autosave\"\t\t\"1\"\n\t}\n}\n"
	writeControllerTestFile(t, path, original)
	writeControllerTestFile(t, filepath.Join(controllerTestTemplatesDir(root), "controller_generic_gamepad_joystick.vdf"), "\"controller_mappings\"\n{\n}\n")

	result, err := EnsureGenericControllerLayouts(root)
	if err != nil {
		t.Fatalf("ensure layouts: %v", err)
	}
	if len(result.Existing) != 1 {
		t.Fatalf("existing = %v, want the autosave entry", result.Existing)
	}
	if got := readControllerTestFile(t, path); got != original {
		t.Fatalf("autosave layout was overwritten:\n%s", got)
	}
}

func TestEnsureGenericControllerLayoutsMissingConfigDir(t *testing.T) {
	root := controllerTestRoot(t)
	result, err := EnsureGenericControllerLayouts(root)
	if err != nil {
		t.Fatalf("ensure layouts: %v", err)
	}
	if len(result.Applied) != 0 || len(result.Warnings) != 0 {
		t.Fatalf("result = %+v, want empty", result)
	}
}

func TestEnsureGenericControllerLayoutsMissingTemplateSkips(t *testing.T) {
	root := controllerTestRoot(t)
	configDir := controllerTestConfigDir(root)
	path := filepath.Join(configDir, "configset_controller_xboxone.vdf")
	original := "\"controller_config\"\n{\n}\n"
	writeControllerTestFile(t, path, original)

	result, err := EnsureGenericControllerLayouts(root)
	if err != nil {
		t.Fatalf("ensure layouts: %v", err)
	}
	if len(result.Skipped) != 1 {
		t.Fatalf("skipped = %v, want xboxone", result.Skipped)
	}
	if got := readControllerTestFile(t, path); got != original {
		t.Fatalf("missing-template config changed:\n%s", got)
	}
}
