package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCheckReportsAvailableAndCaches(t *testing.T) {
	server, _ := newReleaseServer(t, "0.2.1", []byte("new binary"), "")
	target := newAppImage(t, "old binary")
	manager := testManager(t, "0.2.0", server)
	manager.env = appImageEnvironment(target)

	check, err := manager.Check(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !check.Available || !check.CanInstall {
		t.Fatalf("check = %+v", check)
	}
	if check.LatestVersion != "0.2.1" || check.InstallKind != KindAppImage {
		t.Fatalf("check = %+v", check)
	}
	if check.Cached || !strings.HasSuffix(check.ReleaseURL, "/releases/tag/v0.2.1") {
		t.Fatalf("check = %+v", check)
	}
	cache, ok := readCache(manager.cacheFile, time.Now())
	if !ok || cache.Tag != "v0.2.1" {
		t.Fatalf("cache = %+v, ok = %v", cache, ok)
	}

	cached, err := manager.CheckStartup(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !cached.Cached || !cached.Available {
		t.Fatalf("cached check = %+v", cached)
	}
}

func TestCheckStartupUsesFreshCacheWithoutNetwork(t *testing.T) {
	manager := New("0.2.0", filepath.Join(t.TempDir(), "update.json"))
	manager.apiURL = "https://127.0.0.1:1/releases/latest"
	if err := writeCache(manager.cacheFile, cacheState{
		CheckedAt: time.Now().Add(-time.Hour),
		Tag:       "v0.2.1",
	}); err != nil {
		t.Fatal(err)
	}
	manager.env = developmentEnvironment()
	check, err := manager.CheckStartup(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !check.Cached || !check.Available || check.LatestVersion != "0.2.1" {
		t.Fatalf("check = %+v", check)
	}
}

func TestCheckStartupDetectsInstallOnce(t *testing.T) {
	manager := New("0.2.0", filepath.Join(t.TempDir(), "update.json"))
	if err := writeCache(manager.cacheFile, cacheState{
		CheckedAt: time.Now().Add(-time.Hour),
		Tag:       "v0.2.1",
	}); err != nil {
		t.Fatal(err)
	}
	calls := 0
	env := developmentEnvironment()
	env.executable = func() (string, error) {
		calls++
		return "/opt/better-rpt", nil
	}
	manager.env = env

	for range 2 {
		check, err := manager.CheckStartup(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if check.InstallKind != KindDevelopment {
			t.Fatalf("install kind = %v, want development", check.InstallKind)
		}
	}
	if calls != 1 {
		t.Fatalf("install detection calls = %d, want 1", calls)
	}
}

func TestCheckStartupRefreshesExpiredCache(t *testing.T) {
	server, _ := newReleaseServer(t, "0.2.1", []byte("new binary"), "")
	manager := testManager(t, "0.2.0", server)
	manager.env = developmentEnvironment()
	if err := writeCache(manager.cacheFile, cacheState{
		CheckedAt: time.Now().Add(-25 * time.Hour),
		Tag:       "v0.1.0",
	}); err != nil {
		t.Fatal(err)
	}
	check, err := manager.CheckStartup(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if check.Cached || !check.Available || check.LatestVersion != "0.2.1" {
		t.Fatalf("check = %+v", check)
	}
}

func TestDevelopmentVersionNeverInstalls(t *testing.T) {
	server, _ := newReleaseServer(t, "0.2.1", []byte("new binary"), "")
	manager := testManager(t, "dev", server)
	manager.env = developmentEnvironment()
	check, err := manager.Check(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !check.Development || check.CanInstall || check.Available {
		t.Fatalf("check = %+v", check)
	}
	if _, err := manager.Install(context.Background(), nil); !errors.Is(err, ErrNoUpdate) {
		t.Fatalf("Install error = %v, want ErrNoUpdate", err)
	}
}

func TestInstallReplacesAppImageAtomically(t *testing.T) {
	content := []byte("new verified binary")
	server, _ := newReleaseServer(t, "0.2.1", content, "")
	target := newAppImage(t, "old binary")
	manager := testManager(t, "0.2.0", server)
	manager.env = appImageEnvironment(target)
	if _, err := manager.Check(context.Background()); err != nil {
		t.Fatal(err)
	}
	var stages []string
	result, err := manager.Install(context.Background(), func(progress Progress) {
		stages = append(stages, progress.Stage)
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != StatusReplaced || !result.Restart || result.Path != target {
		t.Fatalf("result = %+v", result)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(content) {
		t.Fatalf("target = %q", data)
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Fatalf("mode = %o", info.Mode().Perm())
	}
	if len(stages) == 0 || stages[0] != StageDownloading || !containsStage(stages, StageVerifying) || !containsStage(stages, StageInstalling) {
		t.Fatalf("stages = %v", stages)
	}
	leftovers, err := filepath.Glob(filepath.Join(filepath.Dir(target), ".better-rpt-download-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(leftovers) != 0 {
		t.Fatalf("leftovers = %v", leftovers)
	}
}

func TestInstallRejectsChecksumMismatch(t *testing.T) {
	other := sha256.Sum256([]byte("other binary"))
	server, _ := newReleaseServer(t, "0.2.1", []byte("new binary"), hex.EncodeToString(other[:]))
	target := newAppImage(t, "old binary")
	manager := testManager(t, "0.2.0", server)
	manager.env = appImageEnvironment(target)
	if _, err := manager.Check(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Install(context.Background(), nil); !errors.Is(err, ErrChecksum) {
		t.Fatalf("Install error = %v, want ErrChecksum", err)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "old binary" {
		t.Fatalf("target changed to %q", data)
	}
	leftovers, err := filepath.Glob(filepath.Join(filepath.Dir(target), ".better-rpt-download-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(leftovers) != 0 {
		t.Fatalf("leftovers = %v", leftovers)
	}
}

func TestAppImageFallsBackToDownloadsWhenTargetIsNotWritable(t *testing.T) {
	content := []byte("fallback binary")
	server, _ := newReleaseServer(t, "0.2.1", content, "")
	manager := testManager(t, "0.2.0", server)
	target := filepath.Join(t.TempDir(), "missing", "BetterRPT-0.2.0-x86_64.AppImage")
	release := Release{
		Tag:     "v0.2.1",
		Version: Version{Major: 0, Minor: 2, Patch: 1},
		Assets: []Asset{
			{Name: "BetterRPT-0.2.1-x86_64.AppImage", URL: server.URL + "/asset.AppImage"},
			{Name: checksumName, URL: server.URL + "/SHA256SUMS"},
		},
	}
	asset, err := selectAsset(release, KindAppImage, "amd64")
	if err != nil {
		t.Fatal(err)
	}
	checksums, err := selectChecksums(release)
	if err != nil {
		t.Fatal(err)
	}
	result, err := manager.installAppImage(
		context.Background(),
		checksums.URL,
		asset,
		"0.2.1",
		target,
		func(Progress) {},
	)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(manager.downloadsDir, "BetterRPT-0.2.1-x86_64.AppImage")
	if result.Status != StatusSaved || result.Restart || result.Path != want {
		t.Fatalf("result = %+v", result)
	}
	data, err := os.ReadFile(want)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(content) {
		t.Fatalf("saved file = %q", data)
	}
	info, err := os.Stat(want)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Fatalf("mode = %o", info.Mode().Perm())
	}
}

func TestRestartStartsReplacementAndMarksLockWait(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "restarted")
	script := filepath.Join(dir, "BetterRPT.AppImage")
	body := "#!/bin/sh\nprintf '%s' \"$BETTER_RPT_WAIT_FOR_INSTANCE_LOCK\" > \"$RESTART_MARKER\"\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("RESTART_MARKER", marker)
	manager := New("0.2.0", filepath.Join(t.TempDir(), "update.json"))
	manager.restartPath = script
	if err := manager.Restart(); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		data, err := os.ReadFile(marker)
		if err == nil {
			if string(data) != "1" {
				t.Fatalf("lock wait marker = %q", data)
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("replacement did not start: %v", err)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestInstallWithoutAssetsFailsClosed(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/releases/latest" {
			t.Fatalf("unexpected path %s", request.URL.Path)
		}
		_, _ = writer.Write([]byte(`{"tag_name":"v0.2.1","assets":[]}`))
	}))
	defer server.Close()
	target := newAppImage(t, "old binary")
	manager := testManager(t, "0.2.0", server)
	manager.env = appImageEnvironment(target)
	if _, err := manager.Check(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Install(context.Background(), nil); !errors.Is(err, ErrNoAsset) {
		t.Fatalf("Install error = %v, want ErrNoAsset", err)
	}
	if data, _ := os.ReadFile(target); string(data) != "old binary" {
		t.Fatalf("target changed to %q", data)
	}
}

func TestInstallDebUsesPkexecArguments(t *testing.T) {
	server, _ := newReleaseServer(t, "0.2.1", []byte("new package"), "")
	manager := testManager(t, "0.2.0", server)
	argsFile := filepath.Join(t.TempDir(), "args")
	t.Setenv("BETTER_RPT_TEST_ARGS", argsFile)
	script := filepath.Join(t.TempDir(), "pkexec")
	scriptBody := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$BETTER_RPT_TEST_ARGS\"\n"
	if err := os.WriteFile(script, []byte(scriptBody), 0o755); err != nil {
		t.Fatal(err)
	}
	manager.env = environment{
		getenv:       func(string) string { return "" },
		executable:   func() (string, error) { return "/usr/bin/better-rpt", nil },
		stat:         os.Stat,
		evalSymlinks: filepath.EvalSymlinks,
		lookPath: func(name string) (string, error) {
			switch name {
			case "dpkg":
				return "/usr/bin/dpkg", nil
			case "pkexec":
				return script, nil
			default:
				return "", errors.New("not found")
			}
		},
		output: func(context.Context, string, ...string) ([]byte, error) {
			return []byte("better-rpt: /usr/bin/better-rpt\n"), nil
		},
	}
	if _, err := manager.Check(context.Background()); err != nil {
		t.Fatal(err)
	}
	result, err := manager.Install(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != StatusInstalled || !result.Restart || result.Path != "/usr/bin/better-rpt" {
		t.Fatalf("result = %+v", result)
	}
	data, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 3 || lines[0] != "dpkg" || lines[1] != "-i" || !strings.HasSuffix(lines[2], ".deb") {
		t.Fatalf("pkexec args = %q", lines)
	}
}

func TestCloseCancelsPendingCheck(t *testing.T) {
	started := make(chan struct{})
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		close(started)
		<-request.Context().Done()
	}))
	defer server.Close()
	manager := testManager(t, "0.2.0", server)
	manager.env = developmentEnvironment()
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = manager.Check(context.Background())
	}()
	<-started
	manager.Close()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Check did not stop after Close")
	}
}

func testManager(t *testing.T, version string, server *httptest.Server) *Manager {
	t.Helper()
	manager := New(version, filepath.Join(t.TempDir(), "update.json"))
	manager.apiURL = server.URL + "/releases/latest"
	manager.client = server.Client()
	manager.downloadsDir = filepath.Join(t.TempDir(), "Descargas")
	return manager
}

func newAppImage(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "BetterRPT-0.2.0-x86_64.AppImage")
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func appImageEnvironment(target string) environment {
	return environment{
		getenv: func(key string) string {
			if key == "APPIMAGE" {
				return target
			}
			return ""
		},
		executable:   func() (string, error) { return target, nil },
		stat:         os.Stat,
		evalSymlinks: filepath.EvalSymlinks,
		lookPath:     func(string) (string, error) { return "", errors.New("not found") },
		output:       func(context.Context, string, ...string) ([]byte, error) { return nil, errors.New("unused") },
	}
}

func developmentEnvironment() environment {
	return environment{
		getenv:       func(string) string { return "" },
		executable:   func() (string, error) { return "/opt/better-rpt", nil },
		stat:         os.Stat,
		evalSymlinks: filepath.EvalSymlinks,
		lookPath:     func(string) (string, error) { return "", errors.New("not found") },
		output:       func(context.Context, string, ...string) ([]byte, error) { return nil, errors.New("unused") },
	}
}

func newReleaseServer(t *testing.T, version string, content []byte, checksumOverride string) (*httptest.Server, string) {
	t.Helper()
	digest := sha256.Sum256(content)
	checksum := hex.EncodeToString(digest[:])
	if checksumOverride != "" {
		checksum = checksumOverride
	}
	var server *httptest.Server
	server = httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/releases/latest":
			payload := fmt.Sprintf(`{
				"tag_name":"v%s",
				"draft":false,
				"prerelease":false,
				"assets":[
					{"name":"better-rpt_%s-1_amd64.deb","size":%d,"browser_download_url":"%s/asset.deb"},
					{"name":"BetterRPT-%s-x86_64.AppImage","size":%d,"browser_download_url":"%s/asset.AppImage"},
					{"name":"SHA256SUMS","size":200,"browser_download_url":"%s/SHA256SUMS"}
				]
			}`, version, version, len(content), server.URL, version, len(content), server.URL, server.URL)
			_, _ = writer.Write([]byte(payload))
		case "/SHA256SUMS":
			payload := fmt.Sprintf("%s  BetterRPT-%s-x86_64.AppImage\n%s  better-rpt_%s-1_amd64.deb\n",
				checksum, version, checksum, version)
			_, _ = writer.Write([]byte(payload))
		case "/asset.AppImage", "/asset.deb":
			_, _ = writer.Write(content)
		default:
			http.NotFound(writer, request)
		}
	}))
	t.Cleanup(server.Close)
	return server, checksum
}

func containsStage(stages []string, want string) bool {
	for _, stage := range stages {
		if stage == want {
			return true
		}
	}
	return false
}
