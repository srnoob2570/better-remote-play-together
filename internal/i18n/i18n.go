// Package i18n contains the small, static Spanish/English copy catalogue used
// by the GUI. The Spanish catalogue is the fallback for unknown languages and
// for keys that are missing from the selected language.
package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Language is a supported UI language.
type Language string

const (
	Spanish Language = "es"
	English Language = "en"
)

//go:embed catalogs/*.json
var catalogFS embed.FS

var catalogs = loadCatalogs()

func loadCatalogs() map[Language]map[string]string {
	out := make(map[Language]map[string]string, 2)
	for _, lang := range []Language{Spanish, English} {
		name := "catalogs/" + string(lang) + ".json"
		data, err := catalogFS.ReadFile(name)
		if err != nil {
			panic("i18n: embedded catalog missing: " + name)
		}
		var catalog map[string]string
		if err := json.Unmarshal(data, &catalog); err != nil {
			panic("i18n: invalid embedded catalog " + name + ": " + err.Error())
		}
		out[lang] = catalog
	}
	return out
}

// Supported reports whether lang is one of the bundled languages. It accepts
// only the language code, not a full locale.
func Supported(lang string) bool {
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case string(Spanish), string(English):
		return true
	default:
		return false
	}
}

// Normalize maps English locales to English and everything else to Spanish.
func Normalize(lang string) Language {
	if strings.EqualFold(strings.TrimSpace(lang), string(English)) {
		return English
	}
	return Spanish
}

// Detect reads the usual POSIX locale variables and falls back to Spanish.
// LC_ALL takes precedence over LC_MESSAGES, which takes precedence over LANG.
func Detect() Language {
	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		value := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
		switch {
		case strings.HasPrefix(value, string(English)):
			return English
		case strings.HasPrefix(value, string(Spanish)):
			return Spanish
		}
	}
	return Spanish
}

// Translator resolves copy for one language.
type Translator struct {
	lang Language
}

// New returns a translator for lang. Unknown languages fall back to Spanish.
func New(lang string) *Translator {
	return &Translator{lang: Normalize(lang)}
}

// Language returns the resolved language.
func (t *Translator) Language() Language {
	return t.lang
}

// T returns the requested key, falling back to Spanish and then to the key.
// When args are provided, the resolved value is formatted with fmt.Sprintf.
func (t *Translator) T(key string, args ...any) string {
	value, ok := catalogs[t.lang][key]
	if !ok {
		value, ok = catalogs[Spanish][key]
	}
	if !ok {
		return key
	}
	if len(args) == 0 {
		return value
	}
	return fmt.Sprintf(value, args...)
}
