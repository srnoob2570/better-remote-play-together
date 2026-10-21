package config

import "fmt"

// LocalState is a read-only snapshot of the active account's localconfig.vdf.
// The document is read and parsed once so setup, last-played, stale-marker and
// overlay checks can share it.
type LocalState struct {
	Account    SteamAccount
	Path       string
	Launch     LaunchOption
	LastPlayed *LastPlayed
	document   *vdfDocument
}

// ReadLocalState selects the active account, reads its localconfig once and
// parses it once.
func ReadLocalState(steamRoot string) (LocalState, error) {
	if steamRoot == "" {
		return LocalState{}, fmt.Errorf("%w: empty Steam root", ErrLocalConfigMissing)
	}
	account, err := SelectAccount(steamRoot)
	if err != nil {
		return LocalState{}, err
	}
	path := account.LocalConfigPath(steamRoot)
	data, _, err := readLocalConfig(path)
	if err != nil {
		return LocalState{}, err
	}
	document, err := parseVDFDocument(data)
	if err != nil {
		return LocalState{}, fmt.Errorf("%w: parse %s: %v", ErrLocalConfigUnsafe, path, err)
	}
	option, err := launchOptionFromDocument(document)
	if err != nil {
		return LocalState{}, err
	}
	last, err := lastPlayedFromDocument(document)
	if err != nil {
		return LocalState{}, err
	}
	return LocalState{
		Account:    account,
		Path:       path,
		Launch:     option,
		LastPlayed: last,
		document:   document,
	}, nil
}
