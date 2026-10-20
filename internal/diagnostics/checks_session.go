package diagnostics

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/srnoob2570/better-rpt/internal/session"
)

func (e *evaluator) checkSession(_ context.Context) {
	active, activeErr := e.activeSession()
	if activeErr != nil && !errors.Is(activeErr, session.ErrStateMissing) {
		e.add(CheckSession, StatusWarn, "diag.session.state.error", nil, []string{activeErr.Error()},
			"diag.session.state.next", "docs/known-issues.md")
		return
	}

	if active != nil {
		helperAlive := false
		if e.deps.ProcessAlive != nil {
			helperAlive = e.deps.ProcessAlive(active.HelperPID, active.HelperStartTime)
		}
		targetAlive := false
		if active.TargetPID > 0 && e.deps.ProcessAlive != nil {
			targetAlive = e.deps.ProcessAlive(active.TargetPID, active.TargetStartTime)
		}
		evidence := []string{
			"phase=" + string(active.Phase),
			fmt.Sprintf("target_appid=%d", active.TargetAppID),
			fmt.Sprintf("helper_pid=%d alive=%t", active.HelperPID, helperAlive),
		}
		if active.TargetPID > 0 {
			evidence = append(evidence, fmt.Sprintf("target_pid=%d alive=%t", active.TargetPID, targetAlive))
		}
		if active.ExitCode != nil {
			evidence = append(evidence, fmt.Sprintf("exit_code=%d", *active.ExitCode))
		}
		if active.Error != "" {
			evidence = append(evidence, "last_error="+active.Error)
		}
		if !helperAlive || (active.TargetPID > 0 && !targetAlive) {
			e.add(CheckSession, StatusWarn, "diag.session.state.stale", nil, evidence,
				"diag.session.state.staleNext", "docs/known-issues.md")
			return
		}
		e.add(CheckSession, StatusOK, "diag.session.state.active", nil, evidence, "")
		return
	}

	last, lastErr := e.lastLaunch()
	if lastErr != nil {
		if errors.Is(lastErr, session.ErrLastLaunchMissing) {
			e.add(CheckSession, StatusSkip, "diag.session.state.none", nil, nil, "")
			return
		}
		e.add(CheckSession, StatusWarn, "diag.session.state.error", nil, []string{lastErr.Error()},
			"diag.session.state.next", "docs/known-issues.md")
		return
	}
	if last == nil {
		e.add(CheckSession, StatusSkip, "diag.session.state.none", nil, nil, "")
		return
	}
	evidence := []string{
		fmt.Sprintf("appid=%d", last.AppID),
		"outcome=" + string(last.Outcome),
		"source=" + string(last.Source),
	}
	if !last.StartedAt.IsZero() {
		evidence = append(evidence, "started_at="+last.StartedAt.UTC().Format("2006-01-02T15:04:05Z"))
	}
	if !last.EndedAt.IsZero() {
		evidence = append(evidence, "ended_at="+last.EndedAt.UTC().Format("2006-01-02T15:04:05Z"))
	}
	if last.Error != "" {
		evidence = append(evidence, "last_error="+last.Error)
	}
	if last.Token != "" && e.deps.ListStates != nil {
		if states, err := e.deps.ListStates(e.paths); err == nil {
			for _, state := range states {
				if state.Token != last.Token {
					continue
				}
				evidence = append(evidence, "state_phase="+string(state.Phase))
				helperAlive := false
				if e.deps.ProcessAlive != nil {
					helperAlive = e.deps.ProcessAlive(state.HelperPID, state.HelperStartTime)
				}
				evidence = append(evidence,
					fmt.Sprintf("helper_pid=%d alive=%t", state.HelperPID, helperAlive))
				if state.TargetPID > 0 {
					targetAlive := false
					if e.deps.ProcessAlive != nil {
						targetAlive = e.deps.ProcessAlive(state.TargetPID, state.TargetStartTime)
					}
					evidence = append(evidence,
						fmt.Sprintf("target_pid=%d alive=%t", state.TargetPID, targetAlive))
				}
				if state.ExitCode != nil {
					evidence = append(evidence, fmt.Sprintf("exit_code=%d", *state.ExitCode))
				}
				break
			}
		}
	}
	switch last.Outcome {
	case session.LaunchOK:
		e.add(CheckSession, StatusOK, "diag.session.state.last", nil, evidence, "")
	case session.LaunchCanceled:
		e.add(CheckSession, StatusOK, "diag.session.state.canceled", nil, evidence, "")
	default:
		e.add(CheckSession, StatusWarn, "diag.session.state.lastError", []any{strings.ToUpper(string(last.Outcome))},
			evidence, "diag.session.state.next", "docs/known-issues.md")
	}
}
