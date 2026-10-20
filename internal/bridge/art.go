package bridge

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/srnoob2570/better-rpt/internal/art"
	"github.com/srnoob2570/better-rpt/internal/domain"
	"github.com/srnoob2570/better-rpt/internal/steam"
)

const artPrefix = "/api/art/"

// CatalogProvider is the part of app.Service needed to resolve artwork. It is
// intentionally small so the endpoint can be tested without Wails or a UI.
type CatalogProvider interface {
	Catalog() (steam.Catalog, bool)
}

// ArtMiddleware serves /api/art/ from the same origin as the Wails asset
// server and passes every other request to next.
func ArtMiddleware(provider CatalogProvider, logger *slog.Logger) func(http.Handler) http.Handler {
	handler := NewArtHandler(provider, logger)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, artPrefix) {
				handler.ServeHTTP(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// NewArtHandler returns the same-origin artwork handler. It never exposes a
// filesystem path: the only inputs are the App ID and the cover/icon enum.
func NewArtHandler(provider CatalogProvider, logger *slog.Logger) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &artHandler{provider: provider, log: logger}
}

type artHandler struct {
	provider CatalogProvider
	log      *slog.Logger
}

func (h *artHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	rest := strings.TrimPrefix(r.URL.Path, artPrefix)
	if rest == r.URL.Path || rest == "" || strings.HasSuffix(rest, "/") {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	parts := strings.Split(rest, "/")
	if len(parts) != 2 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	appID, err := strconv.ParseUint(parts[0], 10, 32)
	if err != nil || appID == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	kind, ok := art.ParseKind(parts[1])
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	game, ok := h.game(uint32(appID))
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var path string
	if kind == art.Cover {
		path, err = art.CoverPath(game)
	} else {
		path, err = art.IconPath(game)
	}
	if err != nil {
		if errors.Is(err, art.ErrNotFound) || errors.Is(err, art.ErrUnsafe) || errors.Is(err, art.ErrTooLarge) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		h.log.Error("resolve artwork", "appId", appID, "kind", kind, "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	h.serveFile(w, r, path)
}

func (h *artHandler) game(appID uint32) (game domain.Game, ok bool) {
	if h.provider == nil {
		return domain.Game{}, false
	}
	catalog, ok := h.provider.Catalog()
	if !ok {
		return domain.Game{}, false
	}
	for _, candidate := range catalog.Games {
		if candidate.AppID == appID {
			return candidate, true
		}
	}
	return domain.Game{}, false
}

func (h *artHandler) serveFile(w http.ResponseWriter, r *http.Request, path string) {
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		h.log.Error("open artwork", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		h.log.Error("stat artwork", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !info.Mode().IsRegular() || info.Size() > art.MaxFileSize {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	header := make([]byte, 512)
	read, readErr := io.ReadFull(file, header)
	if readErr != nil && readErr != io.EOF && readErr != io.ErrUnexpectedEOF {
		h.log.Error("read artwork header", "error", readErr)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	contentType := imageContentType(header[:read])
	if contentType == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		h.log.Error("rewind artwork", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	etag := fmt.Sprintf(`W/"%x-%x"`, info.Size(), info.ModTime().UnixNano())
	w.Header().Set("Cache-Control", "private, max-age=3600")
	w.Header().Set("ETag", etag)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if matchesETag(r.Header.Get("If-None-Match"), etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.FormatInt(info.Size(), 10))
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodHead {
		return
	}
	if _, err := io.Copy(w, file); err != nil {
		h.log.Warn("copy artwork", "error", err)
	}
}

func imageContentType(data []byte) string {
	if len(data) >= 12 && bytes.Equal(data[:4], []byte("RIFF")) && bytes.Equal(data[8:12], []byte("WEBP")) {
		return "image/webp"
	}
	switch contentType := http.DetectContentType(data); contentType {
	case "image/jpeg", "image/png", "image/webp":
		return contentType
	default:
		return ""
	}
}

func matchesETag(header, etag string) bool {
	if header == "" {
		return false
	}
	for _, raw := range strings.Split(header, ",") {
		candidate := strings.TrimSpace(raw)
		if candidate == "*" || candidate == etag {
			return true
		}
		if strings.TrimPrefix(candidate, "W/") == strings.TrimPrefix(etag, "W/") {
			return true
		}
	}
	return false
}
