package bridge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/srnoob2570/better-rpt/internal/config"
	"github.com/srnoob2570/better-rpt/internal/domain"
	"github.com/srnoob2570/better-rpt/internal/helper"
	"github.com/srnoob2570/better-rpt/internal/session"
)

func TestErrorCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"root", domain.ErrRoot, "root"},
		{"wrapped path", fmt.Errorf("x: %w", domain.ErrPaths), "paths_unavailable"},
		{"instance", domain.ErrInstanceLocked, "instance_locked"},
		{"config", domain.ErrConfig, "config_invalid"},
		{"steam missing", domain.ErrSteamNotFound, "steam_not_found"},
		{"unsupported", domain.ErrUnsupported, "unsupported"},
		{"spacewar", domain.ErrSpaceWarMissing, "spacewar_missing"},
		{"session busy", session.ErrSessionBusy, "session_busy"},
		{"setup missing", config.ErrSetupMissing, "setup_missing"},
		{"helper source", helper.ErrSourceMissing, "helper_source_missing"},
		{"canceled", context.Canceled, "session_canceled"},
		{"deadline", context.DeadlineExceeded, "timeout"},
		{"custom", &ProblemError{Problem: Problem{Code: "custom"}}, "custom"},
		{"unknown", errors.New("boom"), "internal"},
		{"nil", nil, ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := ErrorCode(test.err); got != test.want {
				t.Fatalf("ErrorCode() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestProblemFromError(t *testing.T) {
	err := fmt.Errorf("wrapped: %w", domain.ErrSpaceWarMissing)
	got := ProblemFromError(err)
	if got.Code != "spacewar_missing" {
		t.Fatalf("code = %q", got.Code)
	}
	if got.Message == "" || got.Detail != err.Error() {
		t.Fatalf("problem = %+v", got)
	}
	custom := &ProblemError{Problem: Problem{Code: "custom", Message: "Custom message."}}
	got = ProblemFromError(custom)
	if got.Code != "custom" || got.Message != "Custom message." || got.Detail == "" {
		t.Fatalf("custom = %+v", got)
	}
	if got := ProblemFromError(nil); got.Code != "internal" {
		t.Fatalf("nil = %+v", got)
	}
}

func TestMarshalErrorAlwaysHasDetail(t *testing.T) {
	data := MarshalError(fmt.Errorf("wrapped: %w", domain.ErrSteamRunning))
	var got map[string]string
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got["code"] != "steam_running" {
		t.Fatalf("code = %q", got["code"])
	}
	if got["message"] == "" || got["detail"] == "" {
		t.Fatalf("payload = %+v", got)
	}

	data = MarshalError(nil)
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got["code"] != "internal" || got["detail"] == "" {
		t.Fatalf("nil payload = %+v", got)
	}
}
