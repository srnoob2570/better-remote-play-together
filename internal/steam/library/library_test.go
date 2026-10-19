package library

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/srnoob2570/better-rpt/internal/domain"
)

func TestParseLibraryFoldersModern(t *testing.T) {
	input := `"libraryfolders"
{
	"1" { "path" "/games/secondary" "apps" { "900007" "0" } }
	"0" { "path" "/games/primary" "apps" { "480" "0" } }
	"TimeNextStatsReport" "0"
	"ContentStatsID" "123"
}`
	paths, err := ParseLibraryFolders(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	want := []string{"/games/primary", "/games/secondary"}
	if len(paths) != len(want) {
		t.Fatalf("paths = %v, want %v", paths, want)
	}
	for i := range want {
		if paths[i] != want[i] {
			t.Fatalf("paths = %v, want %v", paths, want)
		}
	}
}

func TestParseLibraryFoldersLegacy(t *testing.T) {
	input := `"libraryfolders"
{
	"TimeNextStatsReport" "0"
	"ContentStatsID" "0"
	"1" "/mnt/legacy"
	"2" "/mnt/other"
}`
	paths, err := ParseLibraryFolders(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	want := []string{"/mnt/legacy", "/mnt/other"}
	if len(paths) != len(want) {
		t.Fatalf("paths = %v, want %v", paths, want)
	}
	for i := range want {
		if paths[i] != want[i] {
			t.Fatalf("paths = %v, want %v", paths, want)
		}
	}
}

func TestParseLibraryFoldersMalformed(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"missing root", `"other" { "0" { "path" "/x" } }`},
		{"missing path", `"libraryfolders" { "0" { "label" "x" } }`},
		{"unexpected type", `"libraryfolders" { "0" { "path" "/x" } "1" { "apps" { "480" "0" } } }`},
		{"empty path", `"libraryfolders" { "0" { "path" "" } }`},
		{"invalid vdf", `"libraryfolders" {`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseLibraryFolders(strings.NewReader(tt.input))
			if !errors.Is(err, domain.ErrVDFMalformed) {
				t.Fatalf("error = %v, want ErrVDFMalformed", err)
			}
		})
	}
}

func TestParseManifest(t *testing.T) {
	manifest, err := ParseManifest("appmanifest_900001.acf", strings.NewReader(manifestText("900001", "Native Game", "Native Game", "4", "1024")))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if manifest.AppID != 900001 || manifest.Name != "Native Game" || manifest.InstallDir != "Native Game" {
		t.Fatalf("manifest = %+v", manifest)
	}
	if !manifest.Installed || manifest.SizeBytes != 1024 {
		t.Fatalf("manifest install/size = %+v", manifest)
	}
}

func TestParseManifestNotInstalled(t *testing.T) {
	manifest, err := ParseManifest("appmanifest_900005.acf", strings.NewReader(manifestText("900005", "Not Installed", "Not Installed", "2", "0")))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if manifest.Installed {
		t.Fatalf("manifest unexpectedly installed: %+v", manifest)
	}
}

func TestParseManifestWithoutSize(t *testing.T) {
	input := `"AppState" { "appid" "900001" "name" "No Size" "installdir" "No Size" "StateFlags" "4" }`
	manifest, err := ParseManifest("appmanifest_900001.acf", strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if manifest.SizeBytes != 0 {
		t.Fatalf("size = %d, want 0", manifest.SizeBytes)
	}
}

func TestParseManifestErrors(t *testing.T) {
	tests := []struct {
		name string
		file string
		body string
	}{
		{"bad filename", "other.acf", manifestText("900001", "Game", "Game", "4", "1")},
		{"appid mismatch", "appmanifest_900001.acf", manifestText("900002", "Game", "Game", "4", "1")},
		{"missing name", "appmanifest_900001.acf", manifestText("900001", "", "Game", "4", "1")},
		{"missing installdir", "appmanifest_900001.acf", manifestText("900001", "Game", "", "4", "1")},
		{"invalid flags", "appmanifest_900001.acf", manifestText("900001", "Game", "Game", "x", "1")},
		{"invalid size", "appmanifest_900001.acf", manifestText("900001", "Game", "Game", "4", "x")},
		{"missing AppState", "appmanifest_900001.acf", `"other" { "appid" "900001" }`},
		{"malformed", "appmanifest_900001.acf", `"AppState" {`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseManifest(tt.file, strings.NewReader(tt.body))
			if !errors.Is(err, domain.ErrManifestCorrupt) {
				t.Fatalf("error = %v, want ErrManifestCorrupt", err)
			}
		})
	}
}

func TestReadManifestFixture(t *testing.T) {
	path := "../../../testdata/steam/fixtures/primary/steamapps/appmanifest_900001.acf"
	manifest, err := ReadManifest(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if manifest.AppID != 900001 || manifest.Name != "Native Game" || !manifest.Installed {
		t.Fatalf("manifest = %+v", manifest)
	}
}

func manifestText(appID, name, installDir, flags, size string) string {
	sizeField := ""
	if size != "" {
		sizeField = fmt.Sprintf(`"SizeOnDisk" "%s"`, size)
	}
	return fmt.Sprintf(`"AppState"
{
	"appid" "%s"
	"name" "%s"
	"installdir" "%s"
	"StateFlags" "%s"
	%s
}`, appID, name, installDir, flags, sizeField)
}
