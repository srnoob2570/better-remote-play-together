package bridge

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/srnoob2570/better-rpt/internal/config"
	"github.com/srnoob2570/better-rpt/internal/domain"
	"github.com/srnoob2570/better-rpt/internal/helper"
	"github.com/srnoob2570/better-rpt/internal/launch"
	"github.com/srnoob2570/better-rpt/internal/session"
	"github.com/srnoob2570/better-rpt/internal/steam/appinfo"
	"github.com/srnoob2570/better-rpt/internal/steam/compat"
)

// ProblemError carries a stable bridge code. The wrapped error, when present,
// stays in Detail for the collapsed technical disclosure.
type ProblemError struct {
	Problem Problem
}

func (e *ProblemError) Error() string {
	if e == nil {
		return ""
	}
	if e.Problem.Message != "" {
		return e.Problem.Message
	}
	return e.Problem.Code
}

func newProblemError(code, message, detail string) error {
	return &ProblemError{Problem: Problem{Code: code, Message: message, Detail: detail}}
}

// ErrorCode maps a Go sentinel to a stable bridge code. Unknown errors fall
// back to "internal" so the frontend always has a translatable value.
func ErrorCode(err error) string {
	if err == nil {
		return ""
	}
	var problem *ProblemError
	if errors.As(err, &problem) && problem.Problem.Code != "" {
		return problem.Problem.Code
	}
	if errors.Is(err, context.Canceled) {
		return "session_canceled"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	for _, entry := range errorCodes {
		if errors.Is(err, entry.err) {
			return entry.code
		}
	}
	return "internal"
}

// ProblemFromError builds the wire problem. Message is an English fallback;
// Detail preserves the original error for the technical disclosure.
func ProblemFromError(err error) Problem {
	if err == nil {
		return Problem{Code: "internal", Message: "Unknown error."}
	}
	var problem *ProblemError
	if errors.As(err, &problem) {
		out := problem.Problem
		if out.Code == "" {
			out.Code = "internal"
		}
		if out.Message == "" {
			out.Message = err.Error()
		}
		if out.Detail == "" {
			out.Detail = err.Error()
		}
		return out
	}
	code := ErrorCode(err)
	message := messageForCode(code)
	if message == "" {
		message = err.Error()
	}
	return Problem{Code: code, Message: message, Detail: err.Error()}
}

// MarshalError is the application.Options.MarshalError hook. It always emits
// code, message and detail keys, even when the detail is empty.
func MarshalError(err error) []byte {
	problem := ProblemFromError(err)
	if problem.Detail == "" {
		problem.Detail = problem.Message
	}
	data, marshalErr := json.Marshal(struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Detail  string `json:"detail"`
	}{
		Code:    problem.Code,
		Message: problem.Message,
		Detail:  problem.Detail,
	})
	if marshalErr != nil {
		return []byte(`{"code":"internal","message":"An unexpected error occurred.","detail":"error marshalling failed"}`)
	}
	return data
}

func messageForCode(code string) string {
	return errorMessages[code]
}

var errorMessages = map[string]string{
	"root":                "Better RPT cannot run as root.",
	"paths_unavailable":   "The user data directories are unavailable.",
	"instance_locked":     "Another Better RPT instance is already running.",
	"config_invalid":      "The configuration is invalid.",
	"steam_not_found":     "Steam was not found.",
	"steam_unsupported":   "The Steam installation is not supported.",
	"library_unreadable":  "The Steam library could not be read.",
	"vdf_malformed":       "A Steam VDF file is malformed.",
	"manifest_corrupt":    "A Steam app manifest is corrupt.",
	"path_escape":         "A path escaped its allowed directory.",
	"spacewar_missing":    "SpaceWar is not installed.",
	"steam_running":       "Steam must be closed for this action.",
	"session_busy":        "Another session is already active.",
	"session_canceled":    "The session was canceled.",
	"session_active":      "A Better RPT session is active.",
	"invalid_session":     "The session is invalid.",
	"invalid_launch_spec": "The game cannot be launched.",
	"helper_missing":      "The helper is missing.",
	"no_launch_option":    "The launch option is missing.",
	"unsupported":         "The value is not supported.",
	"timeout":             "The operation timed out.",
	"setup_missing":       "The SpaceWar setup is missing.",
	"setup_conflict":      "The SpaceWar setup conflicts with Steam.",
	"setup_invalid":       "The SpaceWar setup record is invalid.",
	"internal":            "An unexpected error occurred.",
}

var errorCodes = []struct {
	err  error
	code string
}{
	{domain.ErrRoot, "root"},
	{domain.ErrPaths, "paths_unavailable"},
	{domain.ErrInstanceLocked, "instance_locked"},
	{domain.ErrConfig, "config_invalid"},
	{domain.ErrSteamNotFound, "steam_not_found"},
	{domain.ErrSteamUnsupported, "steam_unsupported"},
	{domain.ErrLibraryUnreadable, "library_unreadable"},
	{domain.ErrVDFMalformed, "vdf_malformed"},
	{domain.ErrManifestCorrupt, "manifest_corrupt"},
	{domain.ErrPathEscape, "path_escape"},
	{domain.ErrSpaceWarMissing, "spacewar_missing"},
	{domain.ErrSteamRunning, "steam_running"},
	{domain.ErrInvalidSession, "invalid_session"},
	{domain.ErrInvalidLaunchSpec, "invalid_launch_spec"},
	{domain.ErrHelperMissing, "helper_missing"},
	{domain.ErrNoLaunchOption, "no_launch_option"},
	{domain.ErrUnsupported, "unsupported"},

	{config.ErrAccountNotFound, "account_not_found"},
	{config.ErrAccountAmbiguous, "account_ambiguous"},
	{config.ErrAccountData, "account_data"},
	{config.ErrLocalConfigMissing, "local_config_missing"},
	{config.ErrLocalConfigUnsafe, "local_config_unsafe"},
	{config.ErrLocalConfigChanged, "local_config_changed"},
	{config.ErrLocalConfigVerify, "local_config_verify"},
	{config.ErrLocalConfigInvalid, "local_config_invalid"},
	{config.ErrBackup, "backup"},
	{config.ErrBackupCorrupt, "backup_corrupt"},
	{config.ErrBackupMissing, "backup_missing"},
	{config.ErrSteamGuardMissing, "steam_guard_missing"},
	{config.ErrAlreadyPrepared, "already_prepared"},
	{config.ErrSessionActive, "session_active"},
	{config.ErrSetupMissing, "setup_missing"},
	{config.ErrSetupConflict, "setup_conflict"},
	{config.ErrSetupInvalid, "setup_invalid"},
	{config.ErrControllerConfig, "controller_config"},
	{config.ErrControllerConfigVerify, "controller_config_verify"},

	{session.ErrStore, "store"},
	{session.ErrPlanMissing, "plan_missing"},
	{session.ErrPlanExpired, "plan_expired"},
	{session.ErrPlanInvalid, "plan_invalid"},
	{session.ErrStateMissing, "invalid_session"},
	{session.ErrStateInvalid, "invalid_session"},
	{session.ErrIntentMissing, "intent_missing"},
	{session.ErrIntentExpired, "intent_expired"},
	{session.ErrIntentInvalid, "intent_invalid"},
	{session.ErrLastLaunchMissing, "last_launch_missing"},
	{session.ErrLastLaunchInvalid, "last_launch_invalid"},
	{session.ErrSessionBusy, "session_busy"},
	{session.ErrEnvironment, "environment"},
	{session.ErrTargetExited, "target_exited"},
	{session.ErrTimeout, "timeout"},

	{helper.ErrSourceMissing, "helper_source_missing"},
	{helper.ErrSourceInvalid, "helper_source_invalid"},
	{helper.ErrInstall, "helper_install"},
	{helper.ErrActiveSession, "session_active"},

	{launch.ErrNotGame, "not_game"},
	{launch.ErrInfoState, "info_state"},
	{launch.ErrNoLaunchConfig, "no_launch_config"},
	{launch.ErrNoLaunchForOS, "no_launch_for_os"},
	{launch.ErrLaunchTemplate, "launch_template"},
	{launch.ErrExecutableMissing, "executable_missing"},
	{launch.ErrExecutableInvalid, "executable_invalid"},
	{launch.ErrAmbiguousExecutable, "ambiguous_executable"},
	{launch.ErrArguments, "arguments"},
	{launch.ErrWorkingDir, "working_dir"},
	{launch.ErrCommand, "command"},
	{launch.ErrCompatDataUnwritable, "compat_data_unwritable"},

	{appinfo.ErrNotAppInfo, "appinfo_not_appinfo"},
	{appinfo.ErrUnsupportedVersion, "appinfo_unsupported_version"},
	{appinfo.ErrCorrupt, "appinfo_corrupt"},
	{appinfo.ErrAppCorrupt, "appinfo_app_corrupt"},
	{appinfo.ErrAppNotFound, "appinfo_app_not_found"},

	{compat.ErrConfig, "compat_config"},
	{compat.ErrMapping, "compat_mapping"},
	{compat.ErrProtonNotFound, "proton_not_found"},
	{compat.ErrProtonAmbiguous, "proton_ambiguous"},
	{compat.ErrToolManifest, "tool_manifest"},
	{compat.ErrNativeTool, "native_tool"},
}
