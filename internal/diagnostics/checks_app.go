package diagnostics

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
)

func (e *evaluator) checkApp(ctx context.Context) {
	if e.pathsErr != nil {
		e.add(CheckConfig, StatusFail, "diag.app.config.paths", nil, []string{e.pathsErr.Error()},
			"diag.app.config.next", "docs/known-issues.md")
		e.add(CheckXDG, StatusFail, "diag.app.xdg.paths", nil, []string{e.pathsErr.Error()},
			"diag.app.xdg.next", "docs/known-issues.md")
		e.add(CheckLock, StatusSkip, "diag.app.lock.skip", nil, nil, "")
		e.add(CheckHelper, StatusSkip, "diag.app.helper.skip", nil, nil, "")
		e.add(CheckLogs, StatusSkip, "diag.app.logs.skip", nil, nil, "")
		return
	}

	configStatus := e.checkConfig()
	configKey := "diag.app.config.ok"
	if configStatus == StatusFail {
		configKey = "diag.app.config.fail"
	} else if configStatus == StatusSkip {
		configKey = "diag.app.config.skip"
	}
	e.add(CheckConfig, configStatus, configKey, nil, e.configEvidence, e.configNext,
		"docs/known-issues.md")

	xdgStatus := e.checkXDG()
	xdgKey := "diag.app.xdg.ok"
	if xdgStatus == StatusFail {
		xdgKey = "diag.app.xdg.fail"
	}
	e.add(CheckXDG, xdgStatus, xdgKey, nil, e.xdgEvidence, e.xdgNext,
		"docs/known-issues.md")

	lockStatus := e.checkLock()
	lockKey := "diag.app.lock.ok"
	if lockStatus == StatusWarn {
		lockKey = "diag.app.lock.warn"
	} else if lockStatus == StatusSkip {
		lockKey = "diag.app.lock.skip"
	}
	e.add(CheckLock, lockStatus, lockKey, nil, e.lockEvidence, e.lockNext,
		"docs/known-issues.md")

	helperStatus := e.checkHelper(ctx)
	helperKey := "diag.app.helper.ok"
	switch helperStatus {
	case StatusWarn:
		helperKey = "diag.app.helper.warn"
	case StatusFail:
		helperKey = "diag.app.helper.fail"
	case StatusSkip:
		helperKey = "diag.app.helper.skip"
	}
	e.add(CheckHelper, helperStatus, helperKey, nil, e.helperEvidence, e.helperNext,
		"docs/known-issues.md", "docs/appimage.md")

	logsStatus := e.checkLogs()
	logsKey := "diag.app.logs.ok"
	if logsStatus == StatusWarn {
		logsKey = "diag.app.logs.warn"
	} else if logsStatus == StatusSkip {
		logsKey = "diag.app.logs.skip"
	}
	e.add(CheckLogs, logsStatus, logsKey, nil, e.logsEvidence, e.logsNext,
		"docs/known-issues.md")
}

func (e *evaluator) checkConfig() Status {
	e.configEvidence = nil
	e.configNext = ""
	if e.deps.LoadConfig == nil {
		return StatusSkip
	}
	cfg, err := e.deps.LoadConfig(e.paths)
	if err != nil {
		e.configEvidence = []string{err.Error()}
		e.configNext = "diag.app.config.next"
		return StatusFail
	}
	e.configEvidence = []string{"config=valid", "language=" + firstNonEmpty(cfg.Language, "(auto)")}
	return StatusOK
}

func (e *evaluator) checkXDG() Status {
	e.xdgEvidence = nil
	e.xdgNext = ""
	dirs := []struct {
		name string
		path string
	}{
		{"config", e.paths.ConfigDir},
		{"data", e.paths.DataDir},
		{"runtime", e.paths.RuntimeDir},
	}
	failed := 0
	for _, dir := range dirs {
		if strings.TrimSpace(dir.path) == "" {
			e.xdgEvidence = append(e.xdgEvidence, dir.name+"=(empty)")
			failed++
			continue
		}
		info, err := e.deps.Stat(dir.path)
		if err == nil {
			if !info.IsDir() {
				e.xdgEvidence = append(e.xdgEvidence, dir.name+"=not a directory")
				failed++
				continue
			}
			writable := e.writable(dir.path, true)
			e.xdgEvidence = append(e.xdgEvidence, fmt.Sprintf("%s=exists,writable=%t", dir.name, writable))
			if !writable {
				failed++
			}
			continue
		}
		if !errors.Is(err, fs.ErrNotExist) {
			e.xdgEvidence = append(e.xdgEvidence, dir.name+"="+err.Error())
			failed++
			continue
		}
		creatable := e.nearestWritable(dir.path)
		e.xdgEvidence = append(e.xdgEvidence, fmt.Sprintf("%s=missing,creatable=%t", dir.name, creatable))
		if !creatable {
			failed++
		}
	}
	if failed > 0 {
		e.xdgNext = "diag.app.xdg.next"
		return StatusFail
	}
	return StatusOK
}

func (e *evaluator) checkLock() Status {
	e.lockEvidence = nil
	e.lockNext = ""
	path := filepath.Join(e.paths.RuntimeDir, "app.lock")
	data, err := e.deps.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			e.lockEvidence = []string{"app.lock=free"}
			return StatusOK
		}
		e.lockEvidence = []string{"app.lock=" + err.Error()}
		e.lockNext = "diag.app.lock.next"
		return StatusWarn
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		e.lockEvidence = []string{"app.lock=invalid"}
		e.lockNext = "diag.app.lock.next"
		return StatusWarn
	}
	if e.deps.PIDAlive != nil && e.deps.PIDAlive(pid) {
		e.lockEvidence = []string{fmt.Sprintf("app.lock=held pid=%d", pid)}
		return StatusOK
	}
	e.lockEvidence = []string{fmt.Sprintf("app.lock=stale pid=%d", pid)}
	e.lockNext = "diag.app.lock.next"
	return StatusWarn
}

func (e *evaluator) checkHelper(ctx context.Context) Status {
	e.helperEvidence = nil
	e.helperNext = ""
	if e.deps.InspectHelper == nil {
		return StatusSkip
	}
	info, err := e.deps.InspectHelper(e.paths)
	if err != nil {
		e.helperEvidence = []string{err.Error()}
		e.helperNext = "diag.app.helper.next"
		return StatusWarn
	}
	e.helperEvidence = append(e.helperEvidence, "path="+info.Path)
	if info.ExpectedSource != "" {
		e.helperEvidence = append(e.helperEvidence, "source="+info.ExpectedSource)
	} else if info.SourceError == "" {
		e.helperEvidence = append(e.helperEvidence, "source=missing")
	}
	if !info.Exists {
		e.helperEvidence = append(e.helperEvidence, "exists=false")
		e.helperNext = "diag.app.helper.missingNext"
		return StatusWarn
	}
	e.helperEvidence = append(e.helperEvidence,
		fmt.Sprintf("exists=true mode=%04o uid=%d", info.Mode.Perm(), info.UID),
		"sha256="+info.SHA256,
	)
	if info.Mode.Perm()&0o111 == 0 {
		e.helperNext = "diag.app.helper.next"
		return StatusFail
	}
	if e.deps.EUID != 0 && info.UID != uint32(e.deps.EUID) {
		e.helperNext = "diag.app.helper.ownerNext"
		return StatusFail
	}
	if info.SourceError != "" {
		e.helperEvidence = append(e.helperEvidence, "source="+info.SourceError)
		e.helperNext = "diag.app.helper.next"
		return StatusWarn
	}
	if info.ExpectedSHA256 != "" {
		e.helperEvidence = append(e.helperEvidence,
			"expected_sha256="+info.ExpectedSHA256,
			fmt.Sprintf("matches_packaged=%t", info.Matches),
		)
		if !info.Matches {
			e.helperNext = "diag.app.helper.mismatchNext"
			return StatusWarn
		}
	}
	if e.deps.Run != nil {
		output, runErr := e.deps.Run(ctx, info.Path, "--version")
		if runErr != nil {
			e.helperEvidence = append(e.helperEvidence, "version=error")
			e.helperNext = "diag.app.helper.next"
			return StatusWarn
		}
		version := helperVersion(string(output))
		e.helperEvidence = append(e.helperEvidence, "version="+firstNonEmpty(version, "desconocida"))
		if version != "" && e.version != "" && version != e.version {
			e.helperNext = "diag.app.helper.versionNext"
			return StatusWarn
		}
	}
	return StatusOK
}

func (e *evaluator) checkLogs() Status {
	e.logsEvidence = nil
	e.logsNext = ""
	if e.deps.ReadLogTail == nil {
		return StatusSkip
	}
	lines, err := e.deps.ReadLogTail(e.paths, 200)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return StatusSkip
		}
		e.logsEvidence = []string{err.Error()}
		e.logsNext = "diag.app.logs.next"
		return StatusWarn
	}
	if len(lines) == 0 {
		return StatusSkip
	}
	e.logsEvidence = append(e.logsEvidence, fmt.Sprintf("lines=%d", len(lines)))
	last := ""
	for _, line := range lines {
		if strings.Contains(line, "level=ERROR") {
			last = line
		}
	}
	if last == "" {
		return StatusOK
	}
	e.logsEvidence = append(e.logsEvidence, "last_error="+last)
	e.logsNext = "diag.app.logs.next"
	return StatusWarn
}

func helperVersion(output string) string {
	fields := strings.Fields(strings.TrimSpace(output))
	if len(fields) == 0 {
		return ""
	}
	return fields[len(fields)-1]
}
