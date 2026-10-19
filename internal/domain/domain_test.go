package domain

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestValidSessionToken(t *testing.T) {
	tests := []struct {
		name  string
		token string
		want  bool
	}{
		{"valid", "0123456789abcdef0123456789abcdef", true},
		{"uppercase", "0123456789ABCDEF0123456789abcdef", false},
		{"short", "0123", false},
		{"nonhex", "0123456789abcdef0123456789abcdeg", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidSessionToken(tt.token); got != tt.want {
				t.Fatalf("ValidSessionToken(%q) = %v, want %v", tt.token, got, tt.want)
			}
		})
	}
}

func TestStatusAndKindStrings(t *testing.T) {
	if got := KindNative.String(); got != "native" {
		t.Fatalf("native = %q", got)
	}
	if got := KindProton.String(); got != "proton" {
		t.Fatalf("proton = %q", got)
	}
	if got := KindUnknown.String(); got != "unknown" {
		t.Fatalf("unknown = %q", got)
	}
	for status, want := range map[Status]string{
		StatusUnknown:      "unknown",
		StatusReady:        "ready",
		StatusIncompatible: "incompatible",
		StatusMissing:      "missing",
		StatusError:        "error",
		StatusActive:       "active",
	} {
		if got := status.String(); got != want {
			t.Fatalf("status %d = %q, want %q", status, got, want)
		}
	}
}

func TestLaunchSpecValidate(t *testing.T) {
	native := LaunchSpec{AppID: 900001, Kind: KindNative, Executable: "game.x86_64"}
	if err := native.Validate(); err != nil {
		t.Fatalf("valid native spec: %v", err)
	}
	proton := LaunchSpec{AppID: 900001, Kind: KindProton, Executable: "Game.exe", ProtonPath: "/proton"}
	if err := proton.Validate(); err != nil {
		t.Fatalf("valid proton spec: %v", err)
	}

	invalid := []LaunchSpec{
		{Kind: KindNative, Executable: "game"},
		{AppID: 1, Kind: KindUnknown, Executable: "game"},
		{AppID: 1, Kind: KindNative},
		{AppID: 1, Kind: KindProton, Executable: "game.exe"},
	}
	for _, spec := range invalid {
		if err := spec.Validate(); !errors.Is(err, ErrInvalidLaunchSpec) {
			t.Fatalf("spec %+v error = %v, want ErrInvalidLaunchSpec", spec, err)
		}
	}
}

func TestSessionValidate(t *testing.T) {
	valid := Session{
		Token:       "0123456789abcdef0123456789abcdef",
		DonorAppID:  SpaceWarAppID,
		TargetAppID: 900001,
		Phase:       PhaseActive,
		StartedAt:   time.Now(),
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid session: %v", err)
	}

	tests := []struct {
		name   string
		mutate func(*Session)
	}{
		{"bad token", func(s *Session) { s.Token = "nope" }},
		{"bad donor", func(s *Session) { s.DonorAppID = 1 }},
		{"target is donor", func(s *Session) { s.TargetAppID = SpaceWarAppID }},
		{"bad phase", func(s *Session) { s.Phase = Phase("bogus") }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := valid
			tt.mutate(&s)
			if err := s.Validate(); !errors.Is(err, ErrInvalidSession) {
				t.Fatalf("error = %v, want ErrInvalidSession", err)
			}
		})
	}
}

func TestRefuseRoot(t *testing.T) {
	if err := RefuseRoot(0); !errors.Is(err, ErrRoot) {
		t.Fatalf("root error = %v", err)
	}
	if err := RefuseRoot(1000); err != nil {
		t.Fatalf("non-root error = %v", err)
	}
	if !strings.Contains(ErrRoot.Error(), "root") {
		t.Fatalf("root error text = %q", ErrRoot.Error())
	}
}
