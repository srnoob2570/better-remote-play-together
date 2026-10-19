package config

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/srnoob2570/better-rpt/internal/domain"
	"github.com/srnoob2570/better-rpt/internal/steam/vdf"
)

const sessionToken = "0123456789abcdef0123456789abcdef"

func fixtureLocalConfig(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "testdata", "steam", "fixtures", "primary", "userdata", "11111111", "config", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return data
}

func TestReadLaunchOptionFixture(t *testing.T) {
	data := fixtureLocalConfig(t, "localconfig.vdf")
	option, err := ReadLaunchOption(data)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !option.Exists || option.Value != "" || option.Raw != `""` {
		t.Fatalf("option = %+v, want an existing empty quoted token", option)
	}

	_, err = parseVDFDocument(data)
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
}

func TestEditExistingLaunchOptionAndRestore(t *testing.T) {
	original := fixtureLocalConfig(t, "localconfig.vdf")
	marker := "'/tmp/helper' --donor --target 900001 --session " + sessionToken + " # %command%"
	edit, err := EditLaunchOption(original, marker)
	if err != nil {
		t.Fatalf("edit: %v", err)
	}
	if edit.CreatedAppBlock || edit.CreatedLaunchOptions {
		t.Fatalf("created flags = %+v, want false", edit)
	}
	if edit.OriginalRaw != `""` || edit.OriginalValue != "" {
		t.Fatalf("original = %q/%q", edit.OriginalRaw, edit.OriginalValue)
	}
	option, err := ReadLaunchOption(edit.Data)
	if err != nil {
		t.Fatalf("read edited: %v", err)
	}
	if !option.Exists || option.Value != marker {
		t.Fatalf("edited option = %+v", option)
	}
	if !bytes.Contains(edit.Data, []byte("cd /opt/other && ./launcher # %command%")) {
		t.Fatal("unrelated app 900001 launch option changed")
	}
	if !bytes.Contains(edit.Data, []byte("webstorage-decoy")) || !bytes.Contains(edit.Data, []byte("root-decoy")) {
		t.Fatal("decoy app blocks changed")
	}
	if _, err := vdf.Parse(bytes.NewReader(edit.Data)); err != nil {
		t.Fatalf("third-party parse edited: %v", err)
	}

	restored, err := RestoreLaunchOption(edit.Data, RestoreSpec{
		NewValue:      edit.NewValue,
		OriginalRaw:   edit.OriginalRaw,
		OriginalValue: edit.OriginalValue,
	})
	if err != nil {
		t.Fatalf("restore: %v", err)
	}
	if !bytes.Equal(restored, original) {
		t.Fatal("restore did not reproduce the original bytes")
	}
}

func TestEditMissingAppAndRestore(t *testing.T) {
	original := fixtureLocalConfig(t, "localconfig_missing.vdf")
	marker := "marker-missing-app"
	edit, err := EditLaunchOption(original, marker)
	if err != nil {
		t.Fatalf("edit: %v", err)
	}
	if !edit.CreatedAppBlock || !edit.CreatedLaunchOptions {
		t.Fatalf("created flags = %+v, want both true", edit)
	}
	option, err := ReadLaunchOption(edit.Data)
	if err != nil {
		t.Fatalf("read edited: %v", err)
	}
	if !option.Exists || option.Value != marker {
		t.Fatalf("edited option = %+v", option)
	}
	if _, err := vdf.Parse(bytes.NewReader(edit.Data)); err != nil {
		t.Fatalf("third-party parse edited: %v", err)
	}

	restored, err := RestoreLaunchOption(edit.Data, RestoreSpec{
		NewValue:             edit.NewValue,
		CreatedAppBlock:      edit.CreatedAppBlock,
		CreatedLaunchOptions: edit.CreatedLaunchOptions,
	})
	if err != nil {
		t.Fatalf("restore: %v", err)
	}
	option, err = ReadLaunchOption(restored)
	if err != nil {
		t.Fatalf("read restored: %v", err)
	}
	if option.Exists {
		t.Fatalf("option still exists after removing created block: %+v", option)
	}
	if !bytes.Contains(restored, []byte("900001")) {
		t.Fatal("unrelated data disappeared")
	}
}

func TestEditMissingLaunchOptionsAndRestore(t *testing.T) {
	original := fixtureLocalConfig(t, "localconfig_no_launchoptions.vdf")
	edit, err := EditLaunchOption(original, "marker-no-launchoptions")
	if err != nil {
		t.Fatalf("edit: %v", err)
	}
	if edit.CreatedAppBlock || !edit.CreatedLaunchOptions {
		t.Fatalf("created flags = %+v, want only LaunchOptions", edit)
	}
	restored, err := RestoreLaunchOption(edit.Data, RestoreSpec{
		NewValue:             edit.NewValue,
		CreatedLaunchOptions: true,
	})
	if err != nil {
		t.Fatalf("restore: %v", err)
	}
	option, err := ReadLaunchOption(restored)
	if err != nil {
		t.Fatalf("read restored: %v", err)
	}
	if option.Exists {
		t.Fatalf("LaunchOptions still exists: %+v", option)
	}
	if !bytes.Contains(restored, []byte(`"cloud"`)) {
		t.Fatal("unrelated 480 data disappeared")
	}
}

func TestEditFailsClosedOnDuplicate(t *testing.T) {
	data := fixtureLocalConfig(t, "localconfig_duplicate.vdf")
	if _, err := EditLaunchOption(data, "x"); !errors.Is(err, ErrLocalConfigUnsafe) {
		t.Fatalf("error = %v, want ErrLocalConfigUnsafe", err)
	}
}

func TestEditFailsClosedOnMalformed(t *testing.T) {
	data := fixtureLocalConfig(t, "localconfig_malformed.vdf")
	if _, err := EditLaunchOption(data, "x"); err == nil {
		t.Fatal("malformed VDF accepted")
	}
}

func TestEditFailsClosedOnConditional(t *testing.T) {
	data := []byte(`"UserLocalConfigStore" { "Software" { "Valve" { "Steam" { "apps" { "480" { "LaunchOptions" "" [$WIN32] } } } } } }`)
	if _, err := EditLaunchOption(data, "x"); !errors.Is(err, ErrLocalConfigUnsafe) {
		t.Fatalf("error = %v, want ErrLocalConfigUnsafe", err)
	}
}

func TestMissingAppsFailsClosed(t *testing.T) {
	data := []byte(`"UserLocalConfigStore" { "Software" { "Valve" { "Steam" { "apps_other" { } } } } }`)
	if _, err := EditLaunchOption(data, "marker"); !errors.Is(err, ErrLocalConfigUnsafe) {
		t.Fatalf("edit error = %v, want ErrLocalConfigUnsafe", err)
	}
	if _, err := ReadLaunchOption(data); !errors.Is(err, ErrLocalConfigUnsafe) {
		t.Fatalf("read error = %v, want ErrLocalConfigUnsafe", err)
	}
}

func TestEditPreservesCommentsAndCRLF(t *testing.T) {
	data := []byte("\"UserLocalConfigStore\"\r\n{\r\n\t// keep me\r\n\t\"Software\" { \"Valve\" { \"Steam\" { \"apps\" { \"480\" { \"LaunchOptions\" \"old \\\"value\\\"\" } } } } }\r\n}\r\n")
	edit, err := EditLaunchOption(data, "new value")
	if err != nil {
		t.Fatalf("edit: %v", err)
	}
	if !bytes.Contains(edit.Data, []byte("// keep me")) || !bytes.Contains(edit.Data, []byte("\r\n")) {
		t.Fatal("comment or CRLF was not preserved")
	}
	restored, err := RestoreLaunchOption(edit.Data, RestoreSpec{
		NewValue:      edit.NewValue,
		OriginalRaw:   edit.OriginalRaw,
		OriginalValue: edit.OriginalValue,
	})
	if err != nil {
		t.Fatalf("restore: %v", err)
	}
	if !bytes.Equal(restored, data) {
		t.Fatal("restore did not reproduce CRLF fixture")
	}
}

func TestReadLaunchOptionDoesNotUseDecoyApps(t *testing.T) {
	data := []byte(`"apps" { "480" { "LaunchOptions" "root-decoy" } }`)
	if _, err := ReadLaunchOption(data); !errors.Is(err, ErrLocalConfigUnsafe) {
		t.Fatalf("error = %v, want ErrLocalConfigUnsafe", err)
	}
}

func TestRestoreRejectsChangedValue(t *testing.T) {
	original := fixtureLocalConfig(t, "localconfig.vdf")
	edit, err := EditLaunchOption(original, "ours")
	if err != nil {
		t.Fatalf("edit: %v", err)
	}
	changed := bytes.Replace(edit.Data, []byte(`"ours"`), []byte(`"theirs"`), 1)
	if _, err := RestoreLaunchOption(changed, RestoreSpec{
		NewValue:      edit.NewValue,
		OriginalRaw:   edit.OriginalRaw,
		OriginalValue: edit.OriginalValue,
	}); !errors.Is(err, ErrLocalConfigChanged) {
		t.Fatalf("error = %v, want ErrLocalConfigChanged", err)
	}
}

func TestBuildLaunchOption(t *testing.T) {
	helper := "/home/user/Steam Data/better-rpt/bin/better-rpt-helper"
	got, err := BuildLaunchOption(helper, 900001, sessionToken)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if !strings.Contains(got, `'/home/user/Steam Data/better-rpt/bin/better-rpt-helper'`) {
		t.Fatalf("helper is not single-quoted: %s", got)
	}
	if !strings.HasSuffix(got, " # %command%") || !strings.Contains(got, "--target 900001") {
		t.Fatalf("launch option = %q", got)
	}

	for _, invalid := range []struct {
		name   string
		helper string
		target uint32
		token  string
	}{
		{"relative helper", "bin/helper", 900001, sessionToken},
		{"donor target", helper, domain.SpaceWarAppID, sessionToken},
		{"zero target", helper, 0, sessionToken},
		{"bad token", helper, 900001, "not-a-token"},
		{"control helper", "/tmp/help\ner", 900001, sessionToken},
	} {
		if _, err := BuildLaunchOption(invalid.helper, invalid.target, invalid.token); err == nil {
			t.Fatalf("%s: invalid launch option accepted", invalid.name)
		}
	}
}

func TestBuildLaunchOptionEscapesSingleQuotes(t *testing.T) {
	got, err := BuildLaunchOption("/tmp/it's here/helper", 900001, sessionToken)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if !strings.Contains(got, `'/tmp/it'\''s here/helper'`) {
		t.Fatalf("single quote escaping = %q", got)
	}
}
