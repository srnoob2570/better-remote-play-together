package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	apiURL          = "https://api.github.com/repos/srnoob2570/better-remote-play-together/releases/latest"
	repoURL         = "https://github.com/srnoob2570/better-remote-play-together"
	releasePage     = repoURL + "/releases/tag/"
	latestPage      = repoURL + "/releases/latest"
	checksumName    = "SHA256SUMS"
	maxMetadata     = 1 << 20
	maxAssetBytes   = 200 << 20
	metadataTimeout = 10 * time.Second
	downloadTimeout = 5 * time.Minute
)

// Kind identifies how the running binary was installed.
type Kind string

const (
	KindAppImage    Kind = "appimage"
	KindDeb         Kind = "deb"
	KindDevelopment Kind = "development"
)

// Asset is one file published with a release.
type Asset struct {
	Name string
	URL  string
	Size int64
}

// Release is the validated subset of a GitHub release.
type Release struct {
	Tag     string
	Version Version
	URL     string
	Assets  []Asset
}

type releasePayload struct {
	TagName    string `json:"tag_name"`
	Draft      bool   `json:"draft"`
	Prerelease bool   `json:"prerelease"`
	Assets     []struct {
		Name string `json:"name"`
		Size int64  `json:"size"`
		URL  string `json:"browser_download_url"`
	} `json:"assets"`
}

func parseRelease(data []byte) (Release, error) {
	var payload releasePayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return Release{}, fmt.Errorf("%w: decode release: %v", ErrInvalidResponse, err)
	}
	if payload.Draft || payload.Prerelease {
		return Release{}, fmt.Errorf("%w: release is a draft or prerelease", ErrInvalidResponse)
	}
	tag := strings.TrimSpace(payload.TagName)
	version, err := ParseVersion(tag)
	if err != nil {
		return Release{}, fmt.Errorf("%w: %v", ErrInvalidResponse, err)
	}
	release := Release{
		Tag:     tag,
		Version: version,
		URL:     releasePage + url.PathEscape(tag),
		Assets:  make([]Asset, 0, len(payload.Assets)),
	}
	for _, item := range payload.Assets {
		release.Assets = append(release.Assets, Asset{
			Name: item.Name,
			URL:  item.URL,
			Size: item.Size,
		})
	}
	return release, nil
}

func selectAsset(release Release, kind Kind, arch string) (Asset, error) {
	if kind != KindAppImage && kind != KindDeb {
		return Asset{}, fmt.Errorf("%w: installation kind %q", ErrUnsupported, kind)
	}
	if arch != "amd64" {
		return Asset{}, fmt.Errorf("%w: architecture %q", ErrNoAsset, arch)
	}
	var matches []Asset
	for _, asset := range release.Assets {
		name := asset.Name
		if filepath.Base(name) != name {
			continue
		}
		switch kind {
		case KindAppImage:
			if strings.HasSuffix(name, ".AppImage") &&
				(strings.Contains(name, "x86_64") || strings.Contains(name, "amd64")) {
				matches = append(matches, asset)
			}
		case KindDeb:
			if strings.HasSuffix(name, ".deb") && strings.Contains(name, "amd64") {
				matches = append(matches, asset)
			}
		}
	}
	if len(matches) != 1 {
		return Asset{}, fmt.Errorf("%w: %s for %s", ErrNoAsset, kind, arch)
	}
	if !isHTTPS(matches[0].URL) {
		return Asset{}, fmt.Errorf("%w: asset %s does not use HTTPS", ErrInvalidResponse, matches[0].Name)
	}
	return matches[0], nil
}

func selectChecksums(release Release) (Asset, error) {
	for _, asset := range release.Assets {
		if asset.Name != checksumName {
			continue
		}
		if !isHTTPS(asset.URL) {
			return Asset{}, fmt.Errorf("%w: SHA256SUMS does not use HTTPS", ErrInvalidResponse)
		}
		return asset, nil
	}
	return Asset{}, fmt.Errorf("%w: SHA256SUMS", ErrNoAsset)
}

func parseChecksums(data []byte) (map[string]string, error) {
	sums := make(map[string]string)
	for lineNumber, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 2 {
			return nil, fmt.Errorf("%w: malformed line %d", ErrChecksum, lineNumber+1)
		}
		digest := strings.ToLower(fields[0])
		if len(digest) != sha256.Size*2 {
			return nil, fmt.Errorf("%w: invalid digest on line %d", ErrChecksum, lineNumber+1)
		}
		if _, err := hex.DecodeString(digest); err != nil {
			return nil, fmt.Errorf("%w: invalid digest on line %d", ErrChecksum, lineNumber+1)
		}
		name := strings.TrimPrefix(fields[1], "*")
		if name == "" || filepath.Base(name) != name {
			return nil, fmt.Errorf("%w: invalid file name on line %d", ErrChecksum, lineNumber+1)
		}
		if _, exists := sums[name]; exists {
			return nil, fmt.Errorf("%w: duplicate file %s", ErrChecksum, name)
		}
		sums[name] = digest
	}
	if len(sums) == 0 {
		return nil, fmt.Errorf("%w: empty SHA256SUMS", ErrChecksum)
	}
	return sums, nil
}

func verifyFile(path, expected string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("%w: open %s: %v", ErrChecksum, path, err)
	}
	defer func() { _ = file.Close() }()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return fmt.Errorf("%w: read %s: %v", ErrChecksum, path, err)
	}
	actual := hex.EncodeToString(hash.Sum(nil))
	if !strings.EqualFold(actual, expected) {
		return fmt.Errorf("%w: expected %s, got %s", ErrChecksum, expected, actual)
	}
	return nil
}

func (m *Manager) downloadVerified(
	ctx context.Context,
	checksumURL string,
	asset Asset,
	destination *os.File,
	progress func(Progress),
) error {
	checksumCtx, cancel := context.WithTimeout(ctx, metadataTimeout)
	defer cancel()
	data, err := fetchBytes(checksumCtx, m.client, checksumURL, m.userAgent(), maxMetadata)
	if err != nil {
		return err
	}
	sums, err := parseChecksums(data)
	if err != nil {
		return err
	}
	expected, ok := sums[asset.Name]
	if !ok {
		return fmt.Errorf("%w: no entry for %s", ErrChecksum, asset.Name)
	}
	if err := m.downloadTo(ctx, asset.URL, destination, maxAssetBytes, func(downloaded, total int64) {
		m.report(StageDownloading, downloaded, total, progress)
	}); err != nil {
		return err
	}
	if err := destination.Sync(); err != nil {
		return fmt.Errorf("%w: sync download: %v", ErrDownload, err)
	}
	if err := destination.Close(); err != nil {
		return fmt.Errorf("%w: close download: %v", ErrDownload, err)
	}
	m.report(StageVerifying, 0, 0, progress)
	if err := verifyFile(destination.Name(), expected); err != nil {
		return err
	}
	return nil
}

func (m *Manager) downloadTo(
	ctx context.Context,
	target string,
	destination io.Writer,
	limit int64,
	progress func(downloaded, total int64),
) error {
	ctx, cancel := context.WithTimeout(ctx, downloadTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidResponse, err)
	}
	if !isHTTPS(target) {
		return fmt.Errorf("%w: download URL is not HTTPS", ErrInvalidResponse)
	}
	request.Header.Set("User-Agent", m.userAgent())
	request.Header.Set("Accept", "application/octet-stream")
	response, err := m.client.Do(request)
	if err != nil {
		return mapRequestError(err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.Request == nil || response.Request.URL == nil || response.Request.URL.Scheme != "https" {
		return fmt.Errorf("%w: final download URL is not HTTPS", ErrInvalidResponse)
	}
	if err := statusError(response.StatusCode); err != nil {
		return err
	}
	total := response.ContentLength
	if total > limit {
		return fmt.Errorf("%w: %d bytes", ErrTooLarge, total)
	}
	writer := &limitedWriter{
		destination: destination,
		limit:       limit,
		total:       total,
		progress:    progress,
	}
	if _, err := io.Copy(writer, response.Body); err != nil {
		if writer.limitHit {
			return fmt.Errorf("%w: more than %d bytes", ErrTooLarge, limit)
		}
		return mapRequestError(err)
	}
	if total < 0 {
		total = writer.written
	}
	if progress != nil {
		progress(writer.written, total)
	}
	return nil
}

func fetchBytes(ctx context.Context, client *http.Client, target, userAgent string, limit int64) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidResponse, err)
	}
	if !isHTTPS(target) {
		return nil, fmt.Errorf("%w: URL is not HTTPS", ErrInvalidResponse)
	}
	request.Header.Set("User-Agent", userAgent)
	request.Header.Set("Accept", "application/vnd.github+json")
	response, err := client.Do(request)
	if err != nil {
		return nil, mapRequestError(err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.Request == nil || response.Request.URL == nil || response.Request.URL.Scheme != "https" {
		return nil, fmt.Errorf("%w: final URL is not HTTPS", ErrInvalidResponse)
	}
	if err := statusError(response.StatusCode); err != nil {
		return nil, err
	}
	data, err := readLimited(response.Body, limit)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func readLimited(reader io.Reader, limit int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, mapRequestError(err)
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("%w: response exceeds %d bytes", ErrInvalidResponse, limit)
	}
	return data, nil
}

func statusError(status int) error {
	switch {
	case status == http.StatusForbidden || status == http.StatusTooManyRequests:
		return fmt.Errorf("%w: HTTP %d", ErrRateLimited, status)
	case status < 200 || status >= 300:
		return fmt.Errorf("%w: HTTP %d", ErrInvalidResponse, status)
	default:
		return nil
	}
}

func mapRequestError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, context.DeadlineExceeded):
		return fmt.Errorf("%w: %v", ErrTimeout, err)
	case errors.Is(err, context.Canceled):
		return fmt.Errorf("%w: %v", ErrCanceled, err)
	case errors.Is(err, ErrInvalidResponse):
		return err
	default:
		return fmt.Errorf("%w: %v", ErrNetwork, err)
	}
}

type limitedWriter struct {
	destination io.Writer
	limit       int64
	total       int64
	written     int64
	limitHit    bool
	lastReport  time.Time
	progress    func(downloaded, total int64)
}

func (w *limitedWriter) Write(data []byte) (int, error) {
	if w.written+int64(len(data)) > w.limit {
		w.limitHit = true
		return 0, fmt.Errorf("%w: more than %d bytes", ErrTooLarge, w.limit)
	}
	written, err := w.destination.Write(data)
	w.written += int64(written)
	if w.progress != nil && (w.lastReport.IsZero() || time.Since(w.lastReport) >= 100*time.Millisecond || w.written == w.total) {
		w.lastReport = time.Now()
		w.progress(w.written, w.total)
	}
	return written, err
}

func isHTTPS(value string) bool {
	parsed, err := url.Parse(value)
	return err == nil && parsed.Scheme == "https" && parsed.Host != ""
}

func releaseForTag(tag string) Release {
	version, err := ParseVersion(tag)
	if err != nil {
		return Release{Tag: tag, URL: latestPage}
	}
	return Release{Tag: tag, Version: version, URL: releasePage + url.PathEscape(tag)}
}
