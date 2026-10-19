// Package vdf is the single adapter for Steam's textual KeyValues format.
// Keeping the third-party parser behind this boundary lets the library and
// configuration packages share it without leaking its API.
package vdf

import (
	"bytes"
	"fmt"
	"io"

	steamvdf "github.com/andygrunwald/vdf"
)

// NewParser returns a parser for a textual KeyValues document.
func NewParser(r io.Reader) *steamvdf.Parser {
	return steamvdf.NewParser(r)
}

// Parse reads and parses a complete KeyValues document. The third-party parser
// is deliberately tolerant of a missing closing brace, so this adapter
// validates the brace and quote balance first and fails closed.
func Parse(r io.Reader) (map[string]interface{}, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read VDF: %w", err)
	}
	if err := validateStructure(data); err != nil {
		return nil, err
	}
	return steamvdf.NewParser(bytes.NewReader(data)).Parse()
}

func validateStructure(data []byte) error {
	depth := 0
	inQuote := false
	escaped := false
	inComment := false
	for i := 0; i < len(data); i++ {
		c := data[i]
		if inComment {
			if c == '\n' {
				inComment = false
			}
			continue
		}
		if inQuote {
			switch {
			case escaped:
				escaped = false
			case c == '\\':
				escaped = true
			case c == '"':
				inQuote = false
			}
			continue
		}
		switch c {
		case '"':
			inQuote = true
		case '/':
			if i+1 < len(data) && data[i+1] == '/' {
				inComment = true
				i++
			}
		case '{':
			depth++
		case '}':
			depth--
			if depth < 0 {
				return fmt.Errorf("invalid VDF: unexpected closing brace")
			}
		}
	}
	if inQuote {
		return fmt.Errorf("invalid VDF: unterminated quoted string")
	}
	if depth != 0 {
		return fmt.Errorf("invalid VDF: unbalanced braces")
	}
	return nil
}
