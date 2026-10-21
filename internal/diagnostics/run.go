package diagnostics

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/srnoob2570/better-rpt/internal/config"
	"github.com/srnoob2570/better-rpt/internal/i18n"
	"github.com/srnoob2570/better-rpt/internal/redact"
	"github.com/srnoob2570/better-rpt/internal/session"
	"github.com/srnoob2570/better-rpt/internal/steam"
)

// Run executes every check and returns a localized report. It never writes to
// Steam or to the user's configuration.
func Run(ctx context.Context, options Options) (report Report) {
	if ctx == nil {
		ctx = context.Background()
	}
	deps := options.Deps
	if deps == nil {
		paths, err := config.ResolvePaths()
		deps = DefaultDeps(paths, err)
	}
	if deps.Now == nil {
		deps.Now = time.Now
	}
	if deps.Getenv == nil {
		deps.Getenv = os.Getenv
	}
	language := options.Language
	if language == "" {
		language = string(i18n.Detect())
	}
	e := &evaluator{
		deps:    deps,
		tr:      i18n.New(language),
		version: options.Version,
		lang:    string(i18n.New(language).Language()),
		now:     deps.Now().UTC(),
		options: options,
		redact:  redact.New(deps.Home, deps.Username),
	}
	if e.now.IsZero() {
		e.now = time.Now().UTC()
	}
	if !options.Now.IsZero() {
		e.now = options.Now.UTC()
	}
	e.format = e.detectFormat()
	e.executable = e.findExecutable()
	defer func() {
		if recovered := recover(); recovered != nil {
			e.add("diagnostics.internal", StatusFail, "diag.internal.fail", []any{
				fmt.Sprint(recovered),
			}, nil, "")
			report = e.report()
		}
	}()
	e.run(ctx)
	return e.report()
}

type evaluator struct {
	deps       *Deps
	tr         *i18n.Translator
	version    string
	lang       string
	now        time.Time
	format     string
	executable string
	options    Options
	redact     *redact.Sanitizer
	checks     []Check

	osEvidence     []string
	glibcEvidence  []string
	glibcNext      string
	mountEvidence  []string
	mountNext      string
	macEvidence    []string
	macNext        string
	configEvidence []string
	configNext     string
	xdgEvidence    []string
	xdgNext        string
	lockEvidence   []string
	lockNext       string
	helperEvidence []string
	helperNext     string
	logsEvidence   []string
	logsNext       string

	steam       steamState
	paths       config.Paths
	pathsErr    error
	setupLoaded bool
	setup       config.SetupSnapshot
	setupErr    error
}

type steamState struct {
	detected  bool
	install   steam.Installation
	detectErr error

	catalogTried bool
	catalog      *steam.Catalog
	catalogErr   error

	runningTried bool
	running      []steam.Process
	runningErr   error
}

func (e *evaluator) run(ctx context.Context) {
	e.paths = e.deps.Paths
	e.pathsErr = e.deps.PathsErr

	e.checkEnvironment(ctx)
	e.checkAppImage(ctx)
	e.checkDependencies(ctx)
	e.checkSteam(ctx)
	e.checkApp(ctx)
	e.checkSession(ctx)
	e.add(CheckLimits, StatusSkip, "diag.limits.skip", nil, []string{
		e.t("diag.limits.evidence"),
	}, "", "docs/compatibility.md", "docs/known-issues.md")
}

func (e *evaluator) report() Report {
	summary := Summary{Total: len(e.checks)}
	for _, check := range e.checks {
		switch check.Status {
		case StatusOK:
			summary.Counts.OK++
		case StatusWarn:
			summary.Counts.Warn++
		case StatusFail:
			summary.Counts.Fail++
		default:
			summary.Counts.Skip++
		}
	}
	switch {
	case summary.Counts.Fail > 0:
		summary.Verdict = StatusFail
	case summary.Counts.Warn > 0:
		summary.Verdict = StatusWarn
	default:
		summary.Verdict = StatusOK
	}
	return Report{
		SchemaVersion: SchemaVersion,
		GeneratedAt:   e.now.Format(time.RFC3339),
		Version:       e.version,
		Format:        e.format,
		Language:      e.lang,
		Summary:       summary,
		Checks:        e.checks,
	}
}

func (e *evaluator) add(id string, status Status, messageKey string, args []any, evidence []string, nextKey string, docs ...string) {
	message := e.t("diag.status." + string(status))
	if messageKey != "" {
		message = e.t(messageKey, args...)
	}
	next := ""
	if nextKey != "" {
		next = e.t(nextKey)
	}
	items := make([]string, 0, len(evidence))
	for _, item := range evidence {
		item = strings.TrimSpace(e.redact.Text(item))
		if item == "" {
			continue
		}
		if len(item) > 1200 {
			item = item[:1200] + "…"
		}
		items = append(items, item)
	}
	e.checks = append(e.checks, Check{
		ID:       id,
		Status:   status,
		Title:    e.t("diag.check." + id + ".title"),
		Message:  e.redact.Text(message),
		Evidence: items,
		NextStep: e.redact.Text(next),
		Docs:     docs,
	})
}

func (e *evaluator) t(key string, args ...any) string {
	return e.tr.T(key, args...)
}

func (e *evaluator) env(key string) string {
	if e.deps.Getenv == nil {
		return ""
	}
	return strings.TrimSpace(e.deps.Getenv(key))
}

func (e *evaluator) detectFormat() string {
	if value := strings.ToLower(e.env("BETTER_RPT_FORMAT")); value != "" {
		return value
	}
	if e.env("APPIMAGE") != "" || e.env("APPDIR") != "" {
		return "appimage"
	}
	executable := e.findExecutable()
	if strings.Contains(strings.ToLower(executable), "appimage") {
		return "appimage"
	}
	if e.deps.Stat != nil && (strings.HasPrefix(executable, "/usr/") || strings.HasPrefix(executable, "/opt/")) {
		if info, err := e.deps.Stat("/usr/lib/better-rpt/better-rpt-helper"); err == nil && info.Mode().IsRegular() {
			return "deb"
		}
	}
	return "dev"
}

func (e *evaluator) findExecutable() string {
	if e.deps.Executable == nil {
		return ""
	}
	value, err := e.deps.Executable()
	if err != nil {
		return ""
	}
	return value
}

func (e *evaluator) steamInstall(ctx context.Context) (steam.Installation, error) {
	if e.steam.detected {
		return e.steam.install, e.steam.detectErr
	}
	e.steam.detected = true
	if e.options.Catalog != nil {
		e.steam.install = e.options.Catalog.Installation
		e.steam.catalog = e.options.Catalog
		e.steam.catalogTried = true
		return e.steam.install, nil
	}
	if e.deps.DetectSteam == nil {
		e.steam.detectErr = fmt.Errorf("steam detector unavailable")
		return steam.Installation{}, e.steam.detectErr
	}
	installation, err := e.deps.DetectSteam(ctx)
	e.steam.install = installation
	e.steam.detectErr = err
	return installation, err
}

func (e *evaluator) steamCatalog(ctx context.Context) (*steam.Catalog, error) {
	if e.steam.catalogTried {
		return e.steam.catalog, e.steam.catalogErr
	}
	e.steam.catalogTried = true
	if e.options.Catalog != nil {
		catalog := *e.options.Catalog
		e.steam.catalog = &catalog
		return e.steam.catalog, nil
	}
	installation, err := e.steamInstall(ctx)
	if err != nil {
		e.steam.catalogErr = err
		return nil, err
	}
	if e.deps.LoadCatalog == nil {
		e.steam.catalogErr = fmt.Errorf("steam catalog loader unavailable")
		return nil, e.steam.catalogErr
	}
	catalog, err := e.deps.LoadCatalog(ctx, installation)
	if err != nil {
		e.steam.catalogErr = err
		return nil, err
	}
	e.steam.catalog = &catalog
	return e.steam.catalog, nil
}

func (e *evaluator) runningSteam(ctx context.Context) ([]steam.Process, error) {
	if e.steam.runningTried {
		return e.steam.running, e.steam.runningErr
	}
	e.steam.runningTried = true
	if e.deps.ScanSteam == nil {
		e.steam.runningErr = fmt.Errorf("steam process scanner unavailable")
		return nil, e.steam.runningErr
	}
	processes, err := e.deps.ScanSteam(e.procRoot())
	e.steam.running = processes
	e.steam.runningErr = err
	return processes, err
}

func (e *evaluator) procRoot() string {
	if e.deps.ProcRoot != "" {
		return e.deps.ProcRoot
	}
	return "/proc"
}

func (e *evaluator) steamRoot() string {
	if !e.steam.detected || e.steam.detectErr != nil {
		return ""
	}
	if e.steam.install.Variant != steam.VariantNative {
		return ""
	}
	return e.steam.install.Root
}

func (e *evaluator) setupSnapshot(ctx context.Context) (config.SetupSnapshot, error) {
	if e.setupLoaded {
		return e.setup, e.setupErr
	}
	e.setupLoaded = true
	root := e.steamRoot()
	if root == "" {
		e.setupErr = fmt.Errorf("steam root unavailable")
		return e.setup, e.setupErr
	}
	if e.deps.DetectSetup == nil {
		e.setupErr = fmt.Errorf("setup detector unavailable")
		return e.setup, e.setupErr
	}
	e.setup, e.setupErr = e.deps.DetectSetup(e.paths, root)
	return e.setup, e.setupErr
}

func (e *evaluator) activeSession() (*session.State, error) {
	if e.deps.ActiveSession == nil {
		return nil, fmt.Errorf("session reader unavailable")
	}
	return e.deps.ActiveSession(e.paths)
}

func (e *evaluator) lastLaunch() (*session.LastLaunch, error) {
	if e.deps.ReadLastLaunch == nil {
		return nil, fmt.Errorf("last-launch reader unavailable")
	}
	return e.deps.ReadLastLaunch(e.paths)
}

func (e *evaluator) writable(path string, directory bool) bool {
	if e.deps.CanWrite == nil {
		return false
	}
	return e.deps.CanWrite(path, directory) == nil
}

func (e *evaluator) nearestWritable(path string) bool {
	for {
		info, err := e.deps.Stat(path)
		if err == nil {
			if !info.IsDir() {
				return e.writable(filepath.Dir(path), true)
			}
			return e.writable(path, true)
		}
		parent := filepath.Dir(path)
		if parent == path {
			return false
		}
		path = parent
	}
}

func (e *evaluator) configVDF() string {
	root := e.steamRoot()
	if root == "" {
		return ""
	}
	return filepath.Join(root, "config", "config.vdf")
}
