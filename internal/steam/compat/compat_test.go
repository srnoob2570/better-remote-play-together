package compat

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeCompatFile(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writeProton(t *testing.T, dir, require string) {
	t.Helper()
	writeCompatFile(t, filepath.Join(dir, "proton"), "", 0o755)
	writeCompatFile(t, filepath.Join(dir, "toolmanifest.vdf"), `"manifest"
{
	"version" "2"
	"commandline" "/proton %verb%"
	"require_tool_appid" "`+require+`"
	"use_sessions" "1"
	"compatmanager_layer_name" "proton"
}
`, 0o755)
}

func writeMappingConfig(t *testing.T, root string, entries string) {
	t.Helper()
	writeCompatFile(t, filepath.Join(root, "config", "config.vdf"), `"InstallConfigStore"
{
	"Software"
	{
		"Valve"
		{
			"Steam"
			{
				"CompatToolMapping"
				{
`+entries+`
				}
			}
		}
	}
}
`, 0o644)
}

func TestReadMappingsFixture(t *testing.T) {
	path := filepath.Join("..", "..", "..", "testdata", "steam", "fixtures", "primary", "config", "config.vdf")
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer file.Close()
	mappings, err := ReadMappings(file)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if mappings.Global == nil || mappings.Global.Name != "GE-Proton-test" {
		t.Fatalf("global = %+v", mappings.Global)
	}
	if got, ok := mappings.Exact[900002]; !ok || got.Name != "GE-Proton-test" {
		t.Fatalf("900002 = %+v, ok=%v", got, ok)
	}
	if got, ok := mappings.Exact[900007]; !ok || !got.Native() {
		t.Fatalf("900007 = %+v, ok=%v", got, ok)
	}
	if got, ok := mappings.For(900002); !ok || got.Name != "GE-Proton-test" {
		t.Fatalf("For(900002) = %+v, ok=%v", got, ok)
	}
	if got, ok := mappings.For(900099); !ok || got.Name != "GE-Proton-test" {
		t.Fatalf("global fallback = %+v, ok=%v", got, ok)
	}
}

func TestReadMappingsMalformed(t *testing.T) {
	if _, err := ReadMappings(strings.NewReader(`"InstallConfigStore" {`)); !errors.Is(err, ErrConfig) {
		t.Fatalf("error = %v, want ErrConfig", err)
	}
	if _, err := ReadMappings(strings.NewReader(`"InstallConfigStore" { "Software" { "Valve" { "Steam" { "CompatToolMapping" "nope" } } } }`)); !errors.Is(err, ErrMapping) {
		t.Fatalf("error = %v, want ErrMapping for non-object mapping", err)
	}
}

func TestToolManifest(t *testing.T) {
	manifest, err := ReadToolManifest(strings.NewReader(`"manifest"
{
	"version" "2"
	"commandline" "/proton %verb%"
	"require_tool_appid" "900004"
	"use_sessions" "1"
	"compatmanager_layer_name" "proton"
}`))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if manifest.Version != "2" || manifest.CommandLine != "/proton %verb%" ||
		manifest.RequireToolAppID != 900004 || !manifest.UseSessions ||
		manifest.CompatManagerLayerName != "proton" {
		t.Fatalf("manifest = %+v", manifest)
	}
}

func TestResolverExactAndMapping(t *testing.T) {
	root := t.TempDir()
	writeMappingConfig(t, root, `"900002" { "name" "GE-Test" "priority" "250" }`)
	writeProton(t, filepath.Join(root, "compatibilitytools.d", "GE-Test"), "900004")
	resolver := NewResolver(root, []string{root})
	if len(resolver.Warnings()) != 0 {
		t.Fatalf("warnings = %v", resolver.Warnings())
	}
	if !resolver.ForcedProton(900002) {
		t.Fatal("exact GE-Test mapping should force Proton")
	}
	resolution, err := resolver.Resolve(900002)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	want := filepath.Join(root, "compatibilitytools.d", "GE-Test", "proton")
	if resolution.ProtonPath() != want || resolution.ProtonName() != "GE-Test" {
		t.Fatalf("resolution = %+v, want %s", resolution, want)
	}
	if resolution.RequireToolAppID() != 900004 {
		t.Fatalf("require tool = %d, want 900004", resolution.RequireToolAppID())
	}
	if _, ok := resolver.MappingFor(900002); !ok {
		t.Fatal("mapping missing")
	}
}

func TestResolverNativeAndMissing(t *testing.T) {
	root := t.TempDir()
	writeMappingConfig(t, root, `"900007" { "name" "native" }`)
	resolver := NewResolver(root, []string{root})
	if resolver.ForcedProton(900007) {
		t.Fatal("native mapping forced Proton")
	}
	if _, err := resolver.Resolve(900007); !errors.Is(err, ErrNativeTool) {
		t.Fatalf("error = %v, want ErrNativeTool", err)
	}
	if _, err := resolver.Resolve(999999); !errors.Is(err, ErrProtonNotFound) {
		t.Fatalf("error = %v, want ErrProtonNotFound", err)
	}
}

func TestResolverOfficialAliases(t *testing.T) {
	root := t.TempDir()
	writeProton(t, filepath.Join(root, "steamapps", "common", "Proton 9.0"), "900004")
	writeProton(t, filepath.Join(root, "steamapps", "common", "Proton 11.0"), "900004")
	writeProton(t, filepath.Join(root, "steamapps", "common", "Proton - Experimental"), "900004")
	resolver := NewResolver(root, []string{root})

	stable, err := resolver.ResolveName("proton-stable")
	if err != nil {
		t.Fatalf("stable: %v", err)
	}
	if stable.Name != "Proton 11.0" {
		t.Fatalf("stable = %+v, want Proton 11.0", stable)
	}
	experimental, err := resolver.ResolveName("proton-experimental")
	if err != nil {
		t.Fatalf("experimental: %v", err)
	}
	if experimental.Name != "Proton - Experimental" {
		t.Fatalf("experimental = %+v", experimental)
	}
	nine, err := resolver.ResolveName("proton_9")
	if err != nil {
		t.Fatalf("proton_9: %v", err)
	}
	if nine.Name != "Proton 9.0" {
		t.Fatalf("proton_9 = %+v", nine)
	}
	if _, err := resolver.ResolveName("proton_8"); !errors.Is(err, ErrProtonNotFound) {
		t.Fatalf("missing alias error = %v, want ErrProtonNotFound", err)
	}
}

func TestResolverAmbiguousAlias(t *testing.T) {
	root := t.TempDir()
	writeProton(t, filepath.Join(root, "steamapps", "common", "Proton 9.0"), "900004")
	other := t.TempDir()
	writeProton(t, filepath.Join(other, "steamapps", "common", "Proton 9.0"), "900004")
	resolver := NewResolver(root, []string{root, other})
	if _, err := resolver.ResolveName("proton-stable"); !errors.Is(err, ErrProtonAmbiguous) {
		t.Fatalf("error = %v, want ErrProtonAmbiguous", err)
	}
}
