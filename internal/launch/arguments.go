package launch

import (
	"fmt"
	"strings"
)

// SplitArguments lexes Steam's shell-like launch arguments without invoking a
// shell. It supports double quotes and backslash escapes for quotes, spaces and
// backslashes. Percent variables and globs are preserved literally.
func SplitArguments(s string) ([]string, error) {
	args := make([]string, 0)
	var current strings.Builder
	started := false
	inQuote := false

	flush := func() {
		if started {
			args = append(args, current.String())
			current.Reset()
			started = false
		}
	}

	for i := 0; i < len(s); i++ {
		c := s[i]
		if inQuote {
			switch c {
			case '"':
				inQuote = false
				started = true
			case '\\':
				if i+1 < len(s) && (s[i+1] == '"' || s[i+1] == '\\') {
					current.WriteByte(s[i+1])
					i++
				} else {
					current.WriteByte(c)
				}
				started = true
			default:
				current.WriteByte(c)
				started = true
			}
			continue
		}

		switch c {
		case ' ', '\t', '\n', '\r':
			flush()
		case '"':
			inQuote = true
			started = true
		case '\\':
			if i+1 < len(s) && (s[i+1] == '"' || s[i+1] == '\\' || isSpace(s[i+1])) {
				current.WriteByte(s[i+1])
				i++
			} else {
				current.WriteByte(c)
			}
			started = true
		default:
			current.WriteByte(c)
			started = true
		}
	}
	if inQuote {
		return nil, fmt.Errorf("%w: unterminated double quote", ErrArguments)
	}
	flush()
	return args, nil
}

func isSpace(c byte) bool {
	switch c {
	case ' ', '\t', '\n', '\r':
		return true
	default:
		return false
	}
}
