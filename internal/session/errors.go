package session

import "errors"

var (
	// ErrStore means the session directory or file could not be read or written.
	ErrStore = errors.New("session store failed")
	// ErrPlanMissing means no plan exists for the requested token.
	ErrPlanMissing = errors.New("session plan is missing")
	// ErrPlanExpired means the plan is older than its allowed startup window.
	ErrPlanExpired = errors.New("session plan expired")
	// ErrPlanInvalid means the plan contents do not describe a valid session.
	ErrPlanInvalid = errors.New("session plan is invalid")
	// ErrStateMissing means no state file exists for the requested token.
	ErrStateMissing = errors.New("session state is missing")
	// ErrStateInvalid means the state file is malformed or inconsistent.
	ErrStateInvalid = errors.New("session state is invalid")
	// ErrSessionBusy means another helper holds the session lock.
	ErrSessionBusy = errors.New("another session is active")
	// ErrEnvironment means the helper was not started in the SpaceWar donor context.
	ErrEnvironment = errors.New("helper is not running in the Steam donor context")
	// ErrTargetExited means the target exited during the startup window.
	ErrTargetExited = errors.New("target exited immediately")
	// ErrTimeout means a startup or shutdown wait exceeded its deadline.
	ErrTimeout = errors.New("session wait timed out")
)
