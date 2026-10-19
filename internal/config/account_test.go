package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeText(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func loginUsers(mostRecent string, timestamp string, secondRecent string, secondTimestamp string) string {
	return `"users"
{
	"76561197971376839"
	{
		"AccountName" "fixture"
		"PersonaName" "Fixture User"
		"MostRecent" "` + mostRecent + `"
		"Timestamp" "` + timestamp + `"
	}
	"76561197982487950"
	{
		"AccountName" "fixture2"
		"PersonaName" "Second Fixture User"
		"MostRecent" "` + secondRecent + `"
		"Timestamp" "` + secondTimestamp + `"
	}
}
`
}

func TestSelectAccountFixture(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "steam", "fixtures", "primary")
	account, err := SelectAccount(root)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if account.AccountID != 11111111 || account.AccountName != "fixture" || !account.MostRecent {
		t.Fatalf("account = %+v", account)
	}
	wantPath := filepath.Join(root, "userdata", "11111111", "config", "localconfig.vdf")
	if got := account.LocalConfigPath(root); got != wantPath {
		t.Fatalf("localconfig path = %q, want %q", got, wantPath)
	}
}

func TestSelectAccountAmbiguousMostRecent(t *testing.T) {
	root := t.TempDir()
	writeText(t, filepath.Join(root, "loginusers.vdf"), loginUsers("1", "100", "1", "200"))
	if _, err := SelectAccount(root); !errors.Is(err, ErrAccountAmbiguous) {
		t.Fatalf("error = %v, want ErrAccountAmbiguous", err)
	}
}

func TestSelectAccountAutoLogin(t *testing.T) {
	root := t.TempDir()
	writeText(t, filepath.Join(root, "loginusers.vdf"), loginUsers("0", "100", "0", "200"))
	writeText(t, filepath.Join(root, "registry.vdf"), `"Registry" { "HKCU" { "Software" { "Valve" { "Steam" { "AutoLoginUser" "fixture2" } } } } }`)
	account, err := SelectAccount(root)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if account.AccountName != "fixture2" {
		t.Fatalf("account = %+v, want fixture2", account)
	}
}

func TestSelectAccountNewestTimestamp(t *testing.T) {
	root := t.TempDir()
	writeText(t, filepath.Join(root, "loginusers.vdf"), loginUsers("0", "100", "0", "200"))
	account, err := SelectAccount(root)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if account.AccountName != "fixture2" {
		t.Fatalf("account = %+v, want newest fixture2", account)
	}
}

func TestSelectAccountTimestampTie(t *testing.T) {
	root := t.TempDir()
	writeText(t, filepath.Join(root, "loginusers.vdf"), loginUsers("0", "200", "0", "200"))
	if _, err := SelectAccount(root); !errors.Is(err, ErrAccountAmbiguous) {
		t.Fatalf("error = %v, want ErrAccountAmbiguous", err)
	}
}

func TestSelectAccountSingleUserFallback(t *testing.T) {
	root := t.TempDir()
	writeText(t, filepath.Join(root, "loginusers.vdf"), `"users" { "76561197971376839" { "AccountName" "only" } }`)
	account, err := SelectAccount(root)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if account.AccountID != 11111111 || account.AccountName != "only" {
		t.Fatalf("account = %+v", account)
	}
}

func TestSelectAccountMissing(t *testing.T) {
	root := t.TempDir()
	if _, err := SelectAccount(root); !errors.Is(err, ErrAccountNotFound) {
		t.Fatalf("error = %v, want ErrAccountNotFound", err)
	}
}

func TestSelectAccountMalformed(t *testing.T) {
	root := t.TempDir()
	writeText(t, filepath.Join(root, "loginusers.vdf"), `"users" {`)
	if _, err := SelectAccount(root); !errors.Is(err, ErrAccountData) {
		t.Fatalf("error = %v, want ErrAccountData", err)
	}
}

func TestReadAccountsInvalidSteamID(t *testing.T) {
	root := t.TempDir()
	writeText(t, filepath.Join(root, "loginusers.vdf"), `"users" { "123" { "AccountName" "bad" } }`)
	if _, err := ReadAccounts(root); !errors.Is(err, ErrAccountData) || !strings.Contains(err.Error(), "SteamID64") {
		t.Fatalf("error = %v, want invalid SteamID64", err)
	}
}
