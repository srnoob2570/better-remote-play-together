package redact

import "testing"

func TestSanitizerText(t *testing.T) {
	s := New("/home/alice", "alice")
	input := `/home/alice/.local/share/Steam/userdata/76561198012345678/config/localconfig.vdf ` +
		`user=alice token=fedcba9876543210fedcba9876543210 ip=192.168.1.20 ip6=2001:db8::1 ` +
		`hash=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef`
	out := s.Text(input)
	for _, forbidden := range []string{
		"/home/alice", "alice", "76561198012345678",
		"fedcba9876543210fedcba9876543210", "192.168.1.20", "2001:db8::1",
	} {
		if contains(out, forbidden) {
			t.Fatalf("sanitized text still contains %q: %s", forbidden, out)
		}
	}
	if !contains(out, "$HOME") || !contains(out, "$USER") {
		t.Fatalf("expected placeholders, got %s", out)
	}
	if !contains(out, "hash=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef") {
		t.Fatalf("helper SHA-256 was redacted: %s", out)
	}
}

func TestSanitizerKeepsTimestamps(t *testing.T) {
	s := New("/home/alice", "alice")
	input := "2026-10-21T12:34:56Z level=info ts=12:34:56"
	if got := s.Text(input); got != input {
		t.Fatalf("timestamp changed: %q", got)
	}
}

func TestSanitizerKeepsShortUsernamesAndEmpty(t *testing.T) {
	s := New("", "ab")
	if got := s.Text("ab is not a username boundary"); got != "ab is not a username boundary" {
		t.Fatalf("unexpected change: %s", got)
	}
	if got := s.Text(""); got != "" {
		t.Fatalf("empty text changed: %q", got)
	}
}

func contains(value, needle string) bool {
	for index := 0; index+len(needle) <= len(value); index++ {
		if value[index:index+len(needle)] == needle {
			return true
		}
	}
	return false
}
