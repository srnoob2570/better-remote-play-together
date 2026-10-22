package bridge

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/srnoob2570/better-rpt/internal/app"
	"github.com/srnoob2570/better-rpt/internal/domain"
	"github.com/srnoob2570/better-rpt/internal/steam"
)

var _ CatalogProvider = (*app.Service)(nil)

type fakeCatalog struct {
	catalog steam.Catalog
	ok      bool
}

func (f fakeCatalog) Catalog() (steam.Catalog, bool) {
	return f.catalog, f.ok
}

func TestArtHandlerGETAndHEAD(t *testing.T) {
	root := t.TempDir()
	writeArt(t, root, 123, "library_hero.jpg", jpegBytes(t))
	writeArt(t, root, 123, "icon.png", pngBytes(t))
	server := httptest.NewServer(NewArtHandler(catalogFor(root, 123), testLogger()))
	defer func() { server.Close() }()

	response, err := http.Get(server.URL + "/api/art/123/cover")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("GET cover status = %d, want 200", response.StatusCode)
	}
	if len(body) == 0 {
		t.Fatal("GET cover returned an empty body")
	}
	if got := response.Header.Get("Content-Type"); got != "image/jpeg" {
		t.Fatalf("Content-Type = %q, want image/jpeg", got)
	}
	if got := response.Header.Get("Cache-Control"); got != "private, max-age=3600" {
		t.Fatalf("Cache-Control = %q", got)
	}
	if got := response.Header.Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q", got)
	}
	if response.Header.Get("ETag") == "" {
		t.Fatal("ETag is empty")
	}
	if got := response.Header.Get("Content-Length"); got != strconv.Itoa(len(body)) {
		t.Fatalf("Content-Length = %q, want %d", got, len(body))
	}

	request, _ := http.NewRequest(http.MethodHead, server.URL+"/api/art/123/icon", nil)
	response, err = http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(response.Body)
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("HEAD icon status = %d, want 200", response.StatusCode)
	}
	if len(body) != 0 {
		t.Fatalf("HEAD returned %d bytes", len(body))
	}
	if got := response.Header.Get("Content-Type"); got != "image/png" {
		t.Fatalf("HEAD Content-Type = %q, want image/png", got)
	}
	if response.Header.Get("Content-Length") == "" {
		t.Fatal("HEAD Content-Length is empty")
	}
}

func TestArtHandlerValidation(t *testing.T) {
	root := t.TempDir()
	writeArt(t, root, 123, "library_hero.jpg", jpegBytes(t))
	provider := catalogFor(root, 123)
	server := httptest.NewServer(NewArtHandler(provider, testLogger()))
	defer func() { server.Close() }()

	tests := []struct {
		name   string
		method string
		path   string
		status int
	}{
		{"zero app id", http.MethodGet, "/api/art/0/cover", http.StatusBadRequest},
		{"negative app id", http.MethodGet, "/api/art/-1/cover", http.StatusBadRequest},
		{"non numeric app id", http.MethodGet, "/api/art/abc/cover", http.StatusBadRequest},
		{"overflow app id", http.MethodGet, "/api/art/4294967296/cover", http.StatusBadRequest},
		{"unknown kind", http.MethodGet, "/api/art/123/poster", http.StatusBadRequest},
		{"missing kind", http.MethodGet, "/api/art/123", http.StatusBadRequest},
		{"extra segment", http.MethodGet, "/api/art/123/cover/extra", http.StatusBadRequest},
		{"trailing slash", http.MethodGet, "/api/art/123/cover/", http.StatusBadRequest},
		{"unknown game", http.MethodGet, "/api/art/999/cover", http.StatusNotFound},
		{"missing art", http.MethodGet, "/api/art/123/icon", http.StatusNotFound},
		{"post", http.MethodPost, "/api/art/123/cover", http.StatusMethodNotAllowed},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request, _ := http.NewRequest(test.method, server.URL+test.path, nil)
			response, err := http.DefaultClient.Do(request)
			if err != nil {
				t.Fatal(err)
			}
			_ = response.Body.Close()
			if response.StatusCode != test.status {
				t.Fatalf("status = %d, want %d", response.StatusCode, test.status)
			}
			if test.status == http.StatusMethodNotAllowed && response.Header.Get("Allow") != "GET, HEAD" {
				t.Fatalf("Allow = %q", response.Header.Get("Allow"))
			}
		})
	}

	empty := httptest.NewServer(NewArtHandler(fakeCatalog{}, testLogger()))
	defer func() { empty.Close() }()
	response, err := http.Get(empty.URL + "/api/art/123/cover")
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("empty catalog status = %d, want 404", response.StatusCode)
	}
}

func TestArtHandlerETag(t *testing.T) {
	root := t.TempDir()
	writeArt(t, root, 123, "library_hero.jpg", jpegBytes(t))
	server := httptest.NewServer(NewArtHandler(catalogFor(root, 123), testLogger()))
	defer func() { server.Close() }()

	response, err := http.Get(server.URL + "/api/art/123/cover")
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	etag := response.Header.Get("ETag")
	if etag == "" {
		t.Fatal("ETag is empty")
	}

	request, _ := http.NewRequest(http.MethodGet, server.URL+"/api/art/123/cover", nil)
	request.Header.Set("If-None-Match", etag)
	response, err = http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if response.StatusCode != http.StatusNotModified {
		t.Fatalf("conditional status = %d, want 304", response.StatusCode)
	}
	if len(body) != 0 {
		t.Fatalf("304 returned %d bytes", len(body))
	}
}

func TestArtHandlerRejectsDisguisedHTML(t *testing.T) {
	root := t.TempDir()
	writeArt(t, root, 123, "library_hero.jpg", []byte("<html><body>not an image</body></html>"))
	server := httptest.NewServer(NewArtHandler(catalogFor(root, 123), testLogger()))
	defer func() { server.Close() }()

	response, err := http.Get(server.URL + "/api/art/123/cover")
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("disguised HTML status = %d, want 404", response.StatusCode)
	}
}

func TestArtHandlerRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	base := artBase(root, 123)
	outside := filepath.Join(t.TempDir(), "outside.jpg")
	if err := os.WriteFile(outside, jpegBytes(t), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(base, "library_hero.jpg")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	server := httptest.NewServer(NewArtHandler(catalogFor(root, 123), testLogger()))
	defer func() { server.Close() }()

	response, err := http.Get(server.URL + "/api/art/123/cover")
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("symlink escape status = %d, want 404", response.StatusCode)
	}
}

func TestArtHandlerRejectsOversized(t *testing.T) {
	root := t.TempDir()
	path := writeArt(t, root, 123, "library_hero.jpg", jpegBytes(t))
	if err := os.Truncate(path, 8<<20+1); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(NewArtHandler(catalogFor(root, 123), testLogger()))
	defer func() { server.Close() }()

	response, err := http.Get(server.URL + "/api/art/123/cover")
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("oversized status = %d, want 404", response.StatusCode)
	}
}

func TestArtMiddlewarePassesThrough(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})
	handler := ArtMiddleware(fakeCatalog{}, testLogger())(next)
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	record := httptest.NewRecorder()
	handler.ServeHTTP(record, request)
	if !called || record.Code != http.StatusNoContent {
		t.Fatalf("next handler was not called: called=%v status=%d", called, record.Code)
	}
}

func catalogFor(root string, appID uint32) fakeCatalog {
	return fakeCatalog{
		ok: true,
		catalog: steam.Catalog{Games: []domain.Game{{
			AppID:       appID,
			Name:        "Test game",
			LibraryPath: root,
		}}},
	}
}

func artBase(root string, appID uint32) string {
	base := filepath.Join(root, "appcache", "librarycache", strconv.FormatUint(uint64(appID), 10))
	if err := os.MkdirAll(base, 0o755); err != nil {
		panic(err)
	}
	return base
}

func writeArt(t *testing.T, root string, appID uint32, name string, data []byte) string {
	t.Helper()
	path := filepath.Join(artBase(root, appID), name)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func jpegBytes(t *testing.T) []byte {
	t.Helper()
	var buffer bytes.Buffer
	image := image.NewRGBA(image.Rect(0, 0, 1, 1))
	image.Set(0, 0, color.RGBA{R: 0x11, G: 0x22, B: 0x33, A: 0xff})
	if err := jpeg.Encode(&buffer, image, nil); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func pngBytes(t *testing.T) []byte {
	t.Helper()
	var buffer bytes.Buffer
	image := image.NewRGBA(image.Rect(0, 0, 1, 1))
	image.Set(0, 0, color.RGBA{R: 0x11, G: 0x22, B: 0x33, A: 0xff})
	if err := png.Encode(&buffer, image); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
