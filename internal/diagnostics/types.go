// Package diagnostics runs read-only checks and renders a support report. It is
// shared by `better-rpt --diagnose` and the in-app diagnostics screen.
package diagnostics

import (
	"time"

	"github.com/srnoob2570/better-rpt/internal/steam"
)

// SchemaVersion is the version of the JSON report contract.
const SchemaVersion = 1

// Status is the outcome of one check.
type Status string

const (
	StatusOK   Status = "ok"
	StatusWarn Status = "warn"
	StatusFail Status = "fail"
	StatusSkip Status = "skip"
)

// Stable check identifiers. They are part of the support contract and must not
// be translated or renamed.
const (
	CheckVersion    = "system.version"
	CheckFormat     = "system.format"
	CheckOS         = "system.os"
	CheckKernel     = "system.kernel"
	CheckArch       = "system.arch"
	CheckGlibc      = "system.glibc"
	CheckLocale     = "system.locale"
	CheckSessionEnv = "system.session"
	CheckDesktop    = "system.desktop"
	CheckRoot       = "system.root"
	CheckFUSE       = "appimage.fuse"
	CheckMount      = "appimage.mount"
	CheckMAC        = "appimage.mac"
	CheckELF        = "deps.elf"
	CheckSteam      = "steam.install"
	CheckVariant    = "steam.variant"
	CheckSteamRun   = "steam.running"
	CheckSpaceWar   = "steam.spacewar"
	CheckDispatcher = "steam.dispatcher"
	CheckBackups    = "steam.backups"
	CheckSteamPerm  = "steam.permissions"
	CheckOverlay    = "steam.overlay"
	CheckControls   = "steam.controllers"
	CheckWarnings   = "steam.warnings"
	CheckConfig     = "app.config"
	CheckXDG        = "app.xdg"
	CheckLock       = "app.lock"
	CheckHelper     = "app.helper"
	CheckLogs       = "app.logs"
	CheckSession    = "session.state"
	CheckLimits     = "limits.rpt"
)

// Counts is the number of checks in each status.
type Counts struct {
	OK   int `json:"ok"`
	Warn int `json:"warn"`
	Fail int `json:"fail"`
	Skip int `json:"skip"`
}

// Summary is the report verdict and its counts.
type Summary struct {
	Verdict Status `json:"verdict"`
	Counts  Counts `json:"counts"`
	Total   int    `json:"total"`
}

// Check is one localized diagnostic result.
type Check struct {
	ID       string   `json:"id"`
	Status   Status   `json:"status"`
	Title    string   `json:"title"`
	Message  string   `json:"message"`
	Evidence []string `json:"evidence"`
	NextStep string   `json:"nextStep,omitempty"`
	Docs     []string `json:"docs,omitempty"`
}

// Report is the complete machine-readable and human-readable diagnostic result.
type Report struct {
	SchemaVersion int     `json:"schema_version"`
	GeneratedAt   string  `json:"generated_at"`
	Version       string  `json:"version"`
	Format        string  `json:"format"`
	Language      string  `json:"language"`
	Summary       Summary `json:"summary"`
	Checks        []Check `json:"checks"`
}

// Options controls one diagnostic run.
type Options struct {
	Version      string
	Language     string
	Catalog      *steam.Catalog
	CatalogKnown bool
	Deps         *Deps
	Now          time.Time
}
