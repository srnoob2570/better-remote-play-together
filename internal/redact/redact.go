// Package redact removes user identifiers and secrets from diagnostic and log
// text. It is deliberately conservative: a false positive costs detail, while a
// leaked token or SteamID can identify the person sharing a report.
package redact

import (
	"net"
	"regexp"
	"strings"
)

var (
	steamIDPattern = regexp.MustCompile(`\b7656\d{13}\b`)
	tokenPattern   = regexp.MustCompile(`\b[0-9a-fA-F]{32}\b`)
	ipv4Pattern    = regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`)
	ipv6Pattern    = regexp.MustCompile(`(?i)[0-9a-f:]{2,}`)
	uidRuntime     = regexp.MustCompile(`/run/user/\d+`)
)

// Sanitizer replaces personal data with stable placeholders.
type Sanitizer struct {
	Home     string
	Username string
}

// New returns a sanitizer for the current user. Empty values are ignored.
func New(home, username string) *Sanitizer {
	return &Sanitizer{Home: strings.TrimSpace(home), Username: strings.TrimSpace(username)}
}

// Text sanitizes one string.
func (s *Sanitizer) Text(value string) string {
	if value == "" {
		return value
	}
	if s == nil {
		s = &Sanitizer{}
	}
	if s.Home != "" {
		value = strings.ReplaceAll(value, s.Home, "$HOME")
	}
	if len(s.Username) >= 3 {
		value = replaceWord(value, s.Username, "$USER")
	}
	value = uidRuntime.ReplaceAllString(value, "/run/user/$UID")
	value = steamIDPattern.ReplaceAllString(value, "[steamid]")
	value = tokenPattern.ReplaceAllString(value, "[token]")
	value = ipv4Pattern.ReplaceAllString(value, "[ip]")
	value = ipv6Pattern.ReplaceAllStringFunc(value, func(match string) string {
		if net.ParseIP(match) == nil {
			return match
		}
		return "[ip]"
	})
	return value
}

// Lines sanitizes each line independently.
func (s *Sanitizer) Lines(lines []string) []string {
	if lines == nil {
		return nil
	}
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		out = append(out, s.Text(line))
	}
	return out
}

func replaceWord(value, old, replacement string) string {
	if old == "" {
		return value
	}
	var out strings.Builder
	start := 0
	for {
		index := strings.Index(value[start:], old)
		if index < 0 {
			out.WriteString(value[start:])
			break
		}
		index += start
		beforeOK := index == 0 || !isWordByte(value[index-1])
		after := index + len(old)
		afterOK := after == len(value) || !isWordByte(value[after])
		out.WriteString(value[start:index])
		if beforeOK && afterOK {
			out.WriteString(replacement)
		} else {
			out.WriteString(old)
		}
		start = after
	}
	return out.String()
}

func isWordByte(c byte) bool {
	return c == '_' || c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}
