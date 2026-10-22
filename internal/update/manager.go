package update

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// Check is the result of comparing the running version with the latest release.
type Check struct {
	CurrentVersion string
	LatestVersion  string
	Tag            string
	ReleaseURL     string
	Available      bool
	CanInstall     bool
	InstallKind    Kind
	Development    bool
	CheckedAt      time.Time
	Cached         bool
}

// Progress describes one stage of a download or installation.
type Progress struct {
	Stage      string
	Downloaded int64
	Total      int64
	Percent    int
}

const (
	StageDownloading = "downloading"
	StageVerifying   = "verifying"
	StageInstalling  = "installing"
)

// InstallResult describes where the verified update ended up.
type InstallResult struct {
	Status      string
	InstallKind Kind
	Path        string
	Restart     bool
}

const (
	StatusReplaced  = "replaced"
	StatusInstalled = "installed"
	StatusSaved     = "saved"
)

// Manager owns release checks, cached state, downloads and installation. It is
// safe for concurrent use and serializes every network or install operation.
type Manager struct {
	version      string
	cacheFile    string
	downloadsDir string
	apiURL       string
	client       *http.Client
	now          func() time.Time
	env          environment
	pkexecPath   string

	opCh    chan struct{}
	opOnce  sync.Once
	stateMu sync.Mutex
	closed  bool
	cancel  context.CancelFunc
	done    chan struct{}

	installOnce sync.Once
	install     installInfo

	lastRelease   *Release
	lastAvailable bool
	restartPath   string
}

// New builds a manager. cacheFile is usually $XDG_CONFIG_HOME/better-rpt/update.json.
func New(version, cacheFile string) *Manager {
	downloadsDir := ""
	if home, err := os.UserHomeDir(); err == nil {
		downloadsDir = filepath.Join(home, "Descargas")
	}
	return &Manager{
		version:      version,
		cacheFile:    cacheFile,
		downloadsDir: downloadsDir,
		apiURL:       apiURL,
		client:       newHTTPClient(),
		now:          time.Now,
		env:          defaultEnvironment(),
		pkexecPath:   "/usr/bin/pkexec",
		opCh:         make(chan struct{}, 1),
	}
}

func newHTTPClient() *http.Client {
	return &http.Client{
		CheckRedirect: func(request *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("%w: too many redirects", ErrInvalidResponse)
			}
			if request.URL == nil || request.URL.Scheme != "https" {
				return fmt.Errorf("%w: redirect to a non-HTTPS URL", ErrInvalidResponse)
			}
			return nil
		},
	}
}

// installInfo returns the install kind detected for this process. Detection is
// invariant for the process lifetime, so it runs at most once.
func (m *Manager) installInfo() installInfo {
	m.installOnce.Do(func() {
		m.install = detectInstall(m.env)
	})
	return m.install
}

// Check always queries GitHub and records the result in the cache.
func (m *Manager) Check(ctx context.Context) (Check, error) {
	runCtx, finish, err := m.begin(ctx)
	if err != nil {
		return Check{}, err
	}
	defer finish()
	return m.check(runCtx, false)
}

// CheckStartup reuses a successful check younger than 24 hours. Development
// builds are reported without touching the network.
func (m *Manager) CheckStartup(ctx context.Context) (Check, error) {
	runCtx, finish, err := m.begin(ctx)
	if err != nil {
		return Check{}, err
	}
	defer finish()
	return m.check(runCtx, true)
}

func (m *Manager) check(ctx context.Context, useCache bool) (Check, error) {
	now := m.now().UTC()
	info := m.installInfo()
	current, currentErr := ParseVersion(m.version)

	if useCache && (currentErr != nil || m.version == "dev") {
		m.remember(nil, false)
		return Check{
			CurrentVersion: m.version,
			ReleaseURL:     latestPage,
			InstallKind:    KindDevelopment,
			Development:    true,
			CheckedAt:      now,
		}, nil
	}

	if useCache {
		if state, ok := readCache(m.cacheFile, now); ok {
			release := releaseForTag(state.Tag)
			available := currentErr == nil && release.Version.Compare(current) > 0
			m.remember(&release, available)
			return m.buildCheck(current, currentErr, info, release, available, state.CheckedAt, true), nil
		}
	}

	release, err := m.fetchLatest(ctx)
	if err != nil {
		return Check{}, err
	}
	// A cache failure must not turn a successful network check into a failed
	// update check.
	_ = writeCache(m.cacheFile, cacheState{CheckedAt: now, Tag: release.Tag})
	available := currentErr == nil && release.Version.Compare(current) > 0
	m.remember(&release, available)
	return m.buildCheck(current, currentErr, info, release, available, now, false), nil
}

func (m *Manager) buildCheck(current Version, currentErr error, info installInfo, release Release, available bool, checkedAt time.Time, cached bool) Check {
	development := info.Kind == KindDevelopment || currentErr != nil
	return Check{
		CurrentVersion: m.version,
		LatestVersion:  release.Version.String(),
		Tag:            release.Tag,
		ReleaseURL:     release.URL,
		Available:      available,
		CanInstall:     available && info.Kind != KindDevelopment && currentErr == nil,
		InstallKind:    info.Kind,
		Development:    development,
		CheckedAt:      checkedAt,
		Cached:         cached,
	}
}

func (m *Manager) fetchLatest(ctx context.Context) (Release, error) {
	ctx, cancel := context.WithTimeout(ctx, metadataTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, m.apiURL, nil)
	if err != nil {
		return Release{}, fmt.Errorf("%w: %v", ErrInvalidResponse, err)
	}
	request.Header.Set("User-Agent", m.userAgent())
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	response, err := m.client.Do(request)
	if err != nil {
		return Release{}, mapRequestError(err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.Request == nil || response.Request.URL == nil || response.Request.URL.Scheme != "https" {
		return Release{}, fmt.Errorf("%w: final API URL is not HTTPS", ErrInvalidResponse)
	}
	if err := statusError(response.StatusCode); err != nil {
		return Release{}, err
	}
	data, err := readLimited(response.Body, maxMetadata)
	if err != nil {
		return Release{}, err
	}
	return parseRelease(data)
}

// Install downloads, verifies and applies the release found by the last check.
func (m *Manager) Install(ctx context.Context, progress func(Progress)) (InstallResult, error) {
	if progress == nil {
		progress = func(Progress) {}
	}
	runCtx, finish, err := m.begin(ctx)
	if err != nil {
		return InstallResult{}, err
	}
	defer finish()

	m.stateMu.Lock()
	release := m.lastRelease
	available := m.lastAvailable
	m.restartPath = ""
	m.stateMu.Unlock()
	if release == nil {
		return InstallResult{}, ErrNotChecked
	}
	if !available {
		return InstallResult{}, ErrNoUpdate
	}
	current, err := ParseVersion(m.version)
	if err != nil {
		return InstallResult{}, ErrUnsupported
	}
	if release.Version.Compare(current) <= 0 {
		return InstallResult{}, ErrNoUpdate
	}
	info := m.installInfo()
	if info.Kind == KindDevelopment {
		return InstallResult{}, fmt.Errorf("%w: development build", ErrUnsupported)
	}
	if runtime.GOARCH != "amd64" {
		return InstallResult{}, fmt.Errorf("%w: architecture %s", ErrNoAsset, runtime.GOARCH)
	}
	asset, err := selectAsset(*release, info.Kind, runtime.GOARCH)
	if err != nil {
		return InstallResult{}, err
	}
	checksums, err := selectChecksums(*release)
	if err != nil {
		return InstallResult{}, err
	}

	var result InstallResult
	switch info.Kind {
	case KindAppImage:
		result, err = m.installAppImage(runCtx, checksums.URL, asset, release.Version.String(), info.Path, progress)
	case KindDeb:
		result, err = m.installDeb(runCtx, checksums.URL, asset, progress)
	default:
		err = fmt.Errorf("%w: %s", ErrUnsupported, info.Kind)
	}
	if err != nil {
		return InstallResult{}, err
	}
	if result.Restart && result.Path != "" {
		m.stateMu.Lock()
		m.restartPath = result.Path
		m.stateMu.Unlock()
	}
	return result, nil
}

// Restart starts the verified binary and leaves the caller to quit this
// instance. It never starts a file that was not installed by this manager.
func (m *Manager) Restart() error {
	m.stateMu.Lock()
	path := m.restartPath
	closed := m.closed
	m.stateMu.Unlock()
	if closed {
		return ErrCanceled
	}
	if path == "" {
		return ErrNoRestart
	}
	info, err := m.env.stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("%w: %s is not executable", ErrRestart, path)
	}
	command := exec.Command(path)
	command.Stdin = nil
	command.Stdout = nil
	command.Stderr = nil
	// The new process waits briefly for this process to release the
	// single-instance lock during restart.
	command.Env = append(os.Environ(), "BETTER_RPT_WAIT_FOR_INSTANCE_LOCK=1")
	if err := command.Start(); err != nil {
		return fmt.Errorf("%w: %v", ErrRestart, err)
	}
	return nil
}

// Close cancels an in-flight check or download and waits for cleanup. A dpkg
// process that already started is allowed to finish instead of being killed.
func (m *Manager) Close() {
	m.stateMu.Lock()
	m.closed = true
	cancel := m.cancel
	done := m.done
	m.stateMu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
}

func (m *Manager) begin(ctx context.Context) (context.Context, func(), error) {
	if ctx == nil {
		ctx = context.Background()
	}
	m.opOnce.Do(func() {
		if m.opCh == nil {
			m.opCh = make(chan struct{}, 1)
		}
	})
	select {
	case m.opCh <- struct{}{}:
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	}
	m.stateMu.Lock()
	if m.closed {
		m.stateMu.Unlock()
		<-m.opCh
		return nil, nil, ErrCanceled
	}
	runCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	m.cancel = cancel
	m.done = done
	m.stateMu.Unlock()

	finish := func() {
		m.stateMu.Lock()
		if m.done == done {
			m.done = nil
		}
		m.cancel = nil
		m.stateMu.Unlock()
		cancel()
		close(done)
		<-m.opCh
	}
	return runCtx, finish, nil
}

func (m *Manager) remember(release *Release, available bool) {
	m.stateMu.Lock()
	m.lastRelease = release
	m.lastAvailable = available
	m.stateMu.Unlock()
}

func (m *Manager) userAgent() string {
	version := m.version
	if version == "" {
		version = "dev"
	}
	return "better-rpt/" + version
}

func (m *Manager) report(stage string, downloaded, total int64, progress func(Progress)) {
	percent := 0
	if total > 0 {
		percent = int(downloaded * 100 / total)
		if percent < 0 {
			percent = 0
		}
		if percent > 100 {
			percent = 100
		}
	} else if stage != StageDownloading {
		percent = 100
	}
	progress(Progress{
		Stage:      stage,
		Downloaded: downloaded,
		Total:      total,
		Percent:    percent,
	})
}
