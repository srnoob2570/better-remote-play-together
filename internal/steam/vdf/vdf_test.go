package vdf

import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	got, err := Parse(strings.NewReader(`"AppState" { "appid" "480" }`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	appState, ok := got["AppState"].(map[string]interface{})
	if !ok {
		t.Fatalf("AppState = %T, want map", got["AppState"])
	}
	if appState["appid"] != "480" {
		t.Fatalf("appid = %v, want 480", appState["appid"])
	}
}

func TestParseValidatesStructure(t *testing.T) {
	if _, err := Parse(strings.NewReader(`"AppState" { "name" "brace { and quote inside" } // trailing comment`)); err != nil {
		t.Fatalf("valid document: %v", err)
	}
	tests := []string{
		`"AppState" {`,
		`"AppState" { "name" "unterminated }`,
		`"AppState" { } }`,
	}
	for _, input := range tests {
		if _, err := Parse(strings.NewReader(input)); err == nil {
			t.Fatalf("malformed document accepted: %q", input)
		}
	}
}
