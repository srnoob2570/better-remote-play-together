package bridge

import (
	"time"

	"github.com/srnoob2570/better-rpt/internal/config"
	"github.com/srnoob2570/better-rpt/internal/domain"
	"github.com/srnoob2570/better-rpt/internal/session"
	"github.com/srnoob2570/better-rpt/internal/steam"
)

// SchemaVersion is bumped whenever a DTO or event changes incompatibly.
const SchemaVersion = 1

// Snapshot is the state the frontend can load without touching Steam.
type Snapshot struct {
	SchemaVersion int      `json:"schemaVersion"`
	Version       string   `json:"version"`
	Language      string   `json:"language"`
	Config        Config   `json:"config"`
	Catalog       *Catalog `json:"catalog,omitempty"`
}

// Config mirrors the settings the frontend needs to render.
type Config struct {
	Language string `json:"language"`
}

// Catalog is the serializable view of the installed Steam library.
type Catalog struct {
	Games    []Game    `json:"games"`
	Warnings []Warning `json:"warnings"`
}

// Game is one library row. Launch details and filesystem launch specs stay in
// Go; the frontend only receives what the dashboard displays.
type Game struct {
	AppID       uint32   `json:"appId"`
	Name        string   `json:"name"`
	InstallDir  string   `json:"installDir"`
	LibraryPath string   `json:"libraryPath"`
	SizeBytes   uint64   `json:"sizeBytes"`
	Kind        string   `json:"kind"`
	Status      string   `json:"status"`
	Problem     *Problem `json:"problem,omitempty"`
}

// Warning is a non-fatal catalog problem tied to a path.
type Warning struct {
	Path    string  `json:"path"`
	Problem Problem `json:"problem"`
}

// Problem is the shared error shape. Code is stable and translatable; Message
// is an English fallback; Detail is the original technical error.
type Problem struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

// StartSessionInput is the payload for one launch attempt. CloseSteam is kept
// for schema compatibility with the original contract; the guide-first flow
// never closes Steam.
type StartSessionInput struct {
	TargetAppID uint32 `json:"targetAppId"`
	CloseSteam  bool   `json:"closeSteam"`
}

// Active is the result of a successful start or replay.
type Active struct {
	Token string       `json:"token"`
	Game  Game         `json:"game"`
	State SessionState `json:"state"`
}

// SessionState is the durable session snapshot.
type SessionState struct {
	Token       string   `json:"token"`
	DonorAppID  uint32   `json:"donorAppId"`
	TargetAppID uint32   `json:"targetAppId"`
	Phase       string   `json:"phase"`
	Outcome     string   `json:"outcome,omitempty"`
	Error       *Problem `json:"error,omitempty"`
	ExitCode    *int     `json:"exitCode,omitempty"`
	CreatedAt   string   `json:"createdAt,omitempty"`
	UpdatedAt   string   `json:"updatedAt,omitempty"`
	StartedAt   string   `json:"startedAt,omitempty"`
	EndedAt     string   `json:"endedAt,omitempty"`
}

// Recovery is the startup snapshot after a reload.
type Recovery struct {
	State        *SessionState  `json:"state,omitempty"`
	HelperAlive  bool           `json:"helperAlive"`
	Situation    *Situation     `json:"situation,omitempty"`
	Setup        *SetupSnapshot `json:"setup,omitempty"`
	Last         *LastSession   `json:"last,omitempty"`
	Stale        *Stale         `json:"stale,omitempty"`
	ExpiredPlans []string       `json:"expiredPlans"`
}

// Stale describes a restore-pending or orphaned configuration.
type Stale struct {
	Token       string `json:"token"`
	TargetAppID uint32 `json:"targetAppId"`
	Active      bool   `json:"active"`
}

// Situation is the classified lifecycle state used by the current dashboard.
type Situation struct {
	Kind         string        `json:"kind"`
	Setup        string        `json:"setup"`
	Restore      string        `json:"restore"`
	Token        string        `json:"token,omitempty"`
	TargetAppID  uint32        `json:"targetAppId,omitempty"`
	SteamRunning bool          `json:"steamRunning"`
	State        *SessionState `json:"state,omitempty"`
	Last         *LastSession  `json:"last,omitempty"`
}

// LastSession is the durable last-launch record shown in the session panel.
type LastSession struct {
	AppID     uint32 `json:"appId"`
	Name      string `json:"name,omitempty"`
	Token     string `json:"token,omitempty"`
	Source    string `json:"source"`
	StartedAt string `json:"startedAt,omitempty"`
	EndedAt   string `json:"endedAt,omitempty"`
	Outcome   string `json:"outcome"`
	Error     string `json:"error,omitempty"`
	Kind      string `json:"kind,omitempty"`
}

// Setup is the result of installing or recording the dispatcher setup.
type Setup struct {
	Status        string `json:"status"`
	OriginalValue string `json:"originalValue,omitempty"`
	HasBackup     bool   `json:"hasBackup"`
}

// SetupSnapshot is the read-only setup state used by the guide.
type SetupSnapshot struct {
	Status        string `json:"status"`
	OriginalValue string `json:"originalValue,omitempty"`
	HasBackup     bool   `json:"hasBackup"`
	CurrentValue  string `json:"currentValue,omitempty"`
}

func mapCatalog(catalog steam.Catalog) Catalog {
	out := Catalog{
		Games:    make([]Game, 0, len(catalog.Games)),
		Warnings: make([]Warning, 0, len(catalog.Warnings)),
	}
	for _, game := range catalog.Games {
		out.Games = append(out.Games, mapGame(game))
	}
	for _, warning := range catalog.Warnings {
		out.Warnings = append(out.Warnings, Warning{
			Path:    warning.Path,
			Problem: problemValue(warning.Err),
		})
	}
	return out
}

func mapGame(game domain.Game) Game {
	out := Game{
		AppID:       game.AppID,
		Name:        game.Name,
		InstallDir:  game.InstallDir,
		LibraryPath: game.LibraryPath,
		SizeBytes:   game.SizeBytes,
		Kind:        game.Kind.String(),
		Status:      game.Status.String(),
	}
	out.Problem = mapProblem(game.Problem)
	return out
}

func mapActive(active session.Active) Active {
	return Active{
		Token: active.Token,
		Game:  mapGame(active.Game),
		State: mapSessionState(active.State),
	}
}

func mapSessionState(state session.State) SessionState {
	out := SessionState{
		Token:       state.Token,
		DonorAppID:  state.DonorAppID,
		TargetAppID: state.TargetAppID,
		Phase:       string(state.Phase),
		Outcome:     state.Outcome,
		CreatedAt:   formatTime(state.CreatedAt),
		UpdatedAt:   formatTime(state.UpdatedAt),
		StartedAt:   formatTime(state.StartedAt),
		EndedAt:     formatTime(state.EndedAt),
	}
	if state.ErrorCode != "" || state.Error != "" {
		code := state.ErrorCode
		if code == "" {
			code = "internal"
		}
		out.Error = &Problem{Code: code, Message: state.Error, Detail: state.Error}
	}
	if state.ExitCode != nil {
		exitCode := *state.ExitCode
		out.ExitCode = &exitCode
	}
	return out
}

func mapRecovery(recovery *session.Recovery) Recovery {
	out := Recovery{ExpiredPlans: []string{}}
	if recovery == nil {
		return out
	}
	out.HelperAlive = recovery.HelperAlive
	out.ExpiredPlans = append(out.ExpiredPlans, recovery.ExpiredPlans...)
	if recovery.State != nil {
		state := mapSessionState(*recovery.State)
		out.State = &state
	}
	situation := mapSituation(recovery.Situation)
	out.Situation = &situation
	out.Stale = mapStale(recovery.Situation)
	out.Last = mapLast(recovery.Last)
	if recovery.Setup.Status != "" || recovery.Setup.Setup != nil {
		setup := mapSetupSnapshot(recovery.Setup)
		out.Setup = &setup
	}
	return out
}

func mapSituation(situation session.Situation) Situation {
	out := Situation{
		Kind:         string(situation.Kind),
		Setup:        string(situation.Setup),
		Restore:      string(situation.Restore),
		Token:        situation.Token,
		TargetAppID:  situation.TargetAppID,
		SteamRunning: situation.SteamRunning,
	}
	if situation.State != nil {
		state := mapSessionState(*situation.State)
		out.State = &state
	}
	out.Last = mapLast(situation.Last)
	return out
}

func mapStale(situation session.Situation) *Stale {
	if situation.Token == "" {
		return nil
	}
	switch situation.Kind {
	case session.SituationPrepared, session.SituationActive, session.SituationRestorePending,
		session.SituationOrphan, session.SituationConflict:
	default:
		return nil
	}
	active := situation.Kind == session.SituationActive
	if situation.State != nil && situation.State.Phase == domain.PhaseActive {
		active = true
	}
	return &Stale{
		Token:       situation.Token,
		TargetAppID: situation.TargetAppID,
		Active:      active,
	}
}

func mapLast(last *session.LastLaunch) *LastSession {
	if last == nil {
		return nil
	}
	return &LastSession{
		AppID:     last.AppID,
		Name:      last.Name,
		Token:     last.Token,
		Source:    string(last.Source),
		StartedAt: formatTime(last.StartedAt),
		EndedAt:   formatTime(last.EndedAt),
		Outcome:   string(last.Outcome),
		Error:     last.Error,
		Kind:      last.Kind,
	}
}

func mapSetup(setup config.Setup) Setup {
	return Setup{
		Status:        string(setup.Status),
		OriginalValue: setup.Backup.Meta.OriginalValue,
		HasBackup:     setup.Backup.Path != "",
	}
}

func mapSetupSnapshot(snapshot config.SetupSnapshot) SetupSnapshot {
	out := SetupSnapshot{
		Status:       string(snapshot.Status),
		CurrentValue: snapshot.Current.Value,
	}
	if snapshot.Setup != nil {
		out.OriginalValue = snapshot.Setup.Backup.Meta.OriginalValue
		out.HasBackup = snapshot.Setup.Backup.Path != ""
	}
	return out
}

func mapProblem(err error) *Problem {
	if err == nil {
		return nil
	}
	problem := ProblemFromError(err)
	return &problem
}

func problemValue(err error) Problem {
	if err == nil {
		return Problem{Code: "internal", Message: "Unknown warning."}
	}
	return ProblemFromError(err)
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}
