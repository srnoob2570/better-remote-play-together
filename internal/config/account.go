package config

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/srnoob2570/better-rpt/internal/steam/vdf"
)

const steamID64Base uint64 = 76561197960265728

// SteamAccount is one account found in loginusers.vdf.
type SteamAccount struct {
	SteamID64   string
	AccountID   uint32
	AccountName string
	PersonaName string
	MostRecent  bool
	Timestamp   int64
}

// LocalConfigPath returns the selected account's localconfig.vdf.
func (a SteamAccount) LocalConfigPath(steamRoot string) string {
	return filepath.Join(steamRoot, "userdata", strconv.FormatUint(uint64(a.AccountID), 10), "config", "localconfig.vdf")
}

// ReadAccounts reads every user from loginusers.vdf.
func ReadAccounts(steamRoot string) ([]SteamAccount, error) {
	path, err := loginUsersPath(steamRoot)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("%w: %s", ErrAccountNotFound, path)
		}
		return nil, fmt.Errorf("%w: read %s: %v", ErrAccountData, path, err)
	}
	document, err := vdf.Parse(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("%w: parse %s: %v", ErrAccountData, path, err)
	}
	rawUsers, ok := document["users"]
	if !ok {
		return nil, fmt.Errorf("%w: %s has no users object", ErrAccountData, path)
	}
	users, ok := rawUsers.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("%w: users is not an object", ErrAccountData)
	}
	if len(users) == 0 {
		return nil, fmt.Errorf("%w: %s has no users", ErrAccountNotFound, path)
	}

	accounts := make([]SteamAccount, 0, len(users))
	for steamID, raw := range users {
		fields, ok := raw.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("%w: user %s is not an object", ErrAccountData, steamID)
		}
		steamID64, err := strconv.ParseUint(steamID, 10, 64)
		if err != nil || steamID64 < steamID64Base {
			return nil, fmt.Errorf("%w: invalid SteamID64 %q", ErrAccountData, steamID)
		}
		accountID := steamID64 - steamID64Base
		if accountID > math.MaxUint32 {
			return nil, fmt.Errorf("%w: SteamID64 %q overflows account id", ErrAccountData, steamID)
		}
		timestamp, err := intField(fields, "Timestamp")
		if err != nil {
			return nil, fmt.Errorf("%w: user %s Timestamp: %v", ErrAccountData, steamID, err)
		}
		accounts = append(accounts, SteamAccount{
			SteamID64:   steamID,
			AccountID:   uint32(accountID),
			AccountName: stringField(fields, "AccountName"),
			PersonaName: stringField(fields, "PersonaName"),
			MostRecent:  truthyField(fields, "MostRecent"),
			Timestamp:   timestamp,
		})
	}
	sort.Slice(accounts, func(i, j int) bool {
		return accounts[i].SteamID64 < accounts[j].SteamID64
	})
	return accounts, nil
}

// SelectAccount tries MostRecent, then AutoLoginUser, then a unique highest
// Timestamp. A single account is unambiguous even without those fields.
func SelectAccount(steamRoot string) (SteamAccount, error) {
	accounts, err := ReadAccounts(steamRoot)
	if err != nil {
		return SteamAccount{}, err
	}
	if len(accounts) == 1 {
		return accounts[0], nil
	}

	var recent []SteamAccount
	for _, account := range accounts {
		if account.MostRecent {
			recent = append(recent, account)
		}
	}
	switch len(recent) {
	case 1:
		return recent[0], nil
	case 0:
	default:
		return SteamAccount{}, fmt.Errorf("%w: several accounts have MostRecent", ErrAccountAmbiguous)
	}

	autoLogin, err := autoLoginUser(steamRoot)
	if err != nil {
		return SteamAccount{}, err
	}
	if autoLogin != "" {
		var matches []SteamAccount
		for _, account := range accounts {
			if account.AccountName == autoLogin {
				matches = append(matches, account)
			}
		}
		switch len(matches) {
		case 1:
			return matches[0], nil
		case 0:
		default:
			return SteamAccount{}, fmt.Errorf("%w: AutoLoginUser %q matches several accounts", ErrAccountAmbiguous, autoLogin)
		}
	}

	var newest SteamAccount
	var newestCount int
	for _, account := range accounts {
		switch {
		case account.Timestamp > newest.Timestamp:
			newest = account
			newestCount = 1
		case account.Timestamp == newest.Timestamp && account.Timestamp > 0:
			newestCount++
		}
	}
	if newest.Timestamp > 0 {
		if newestCount == 1 {
			return newest, nil
		}
		return SteamAccount{}, fmt.Errorf("%w: several accounts share the newest Timestamp", ErrAccountAmbiguous)
	}
	return SteamAccount{}, fmt.Errorf("%w: cannot determine the active account", ErrAccountNotFound)
}

func loginUsersPath(steamRoot string) (string, error) {
	candidates := []string{
		filepath.Join(steamRoot, "config", "loginusers.vdf"),
		filepath.Join(steamRoot, "loginusers.vdf"),
	}
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("%w: loginusers.vdf not found under %s", ErrAccountNotFound, steamRoot)
}

func autoLoginUser(steamRoot string) (string, error) {
	var candidates []string
	candidates = append(candidates, filepath.Join(steamRoot, "registry.vdf"))
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, filepath.Join(home, ".steam", "registry.vdf"))
	}
	path := ""
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			path = candidate
			break
		}
	}
	if path == "" {
		return "", nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("%w: read %s: %v", ErrAccountData, path, err)
	}
	document, err := vdf.Parse(bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("%w: parse %s: %v", ErrAccountData, path, err)
	}
	steam, ok := vdf.Lookup(document, "Registry", "HKCU", "Software", "Valve", "Steam")
	if !ok {
		return "", nil
	}
	return stringField(steam, "AutoLoginUser"), nil
}

func stringField(fields map[string]interface{}, key string) string {
	value, ok := fields[key]
	if !ok {
		return ""
	}
	text, _ := value.(string)
	return strings.TrimSpace(text)
}

func intField(fields map[string]interface{}, key string) (int64, error) {
	text := stringField(fields, key)
	if text == "" {
		return 0, nil
	}
	value, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		return 0, err
	}
	return value, nil
}

func truthyField(fields map[string]interface{}, key string) bool {
	switch strings.ToLower(stringField(fields, key)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
