package bridge

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/srnoob2570/better-rpt/internal/diagnostics"
	"github.com/srnoob2570/better-rpt/internal/fsutil"
)

// DiagnosticsResult is the in-app report plus the Markdown the copy button
// places on the clipboard.
type DiagnosticsResult struct {
	Report   diagnostics.Report `json:"report"`
	Markdown string             `json:"markdown"`
}

// RunDiagnostics runs the same read-only engine as `--diagnose`. It uses the
// service's resolved paths and cached catalog, but never takes the instance
// lock or writes Steam configuration.
func (b *Bridge) RunDiagnostics(ctx context.Context) (DiagnosticsResult, error) {
	options := diagnostics.Options{
		Version:  b.version,
		Language: string(b.app.Language()),
		Deps:     diagnostics.DefaultDeps(b.app.Paths(), nil),
	}
	if catalog, ok := b.app.Catalog(); ok {
		options.Catalog = &catalog
	}
	report := diagnostics.Run(contextOrBackground(ctx), options)
	return DiagnosticsResult{
		Report:   report,
		Markdown: diagnostics.RenderMarkdown(report),
	}, nil
}

// SaveDiagnostics writes the Markdown report to the private diagnostics
// directory. The UI only passes text produced by RunDiagnostics; the size cap
// prevents an accidental large write.
func (b *Bridge) SaveDiagnostics(markdown string) (string, error) {
	if len(markdown) == 0 {
		return "", newProblemError("diagnostics_empty", "The diagnostic report is empty.", "")
	}
	if len(markdown) > 1<<20 {
		return "", newProblemError("diagnostics_too_large", "The diagnostic report is too large.", "")
	}
	dir := b.app.Paths().DiagnosticsDir()
	if err := fsutil.EnsurePrivateDir(dir); err != nil {
		return "", fmt.Errorf("prepare diagnostics directory: %w", err)
	}
	stamp := time.Now().UTC().Format("20060102T150405Z")
	for suffix := 0; suffix < 1000; suffix++ {
		name := "better-rpt-diagnostic-" + stamp
		if suffix > 0 {
			name += fmt.Sprintf("-%d", suffix)
		}
		path := filepath.Join(dir, name+".md")
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			if os.IsExist(err) {
				continue
			}
			return "", fmt.Errorf("save diagnostic report: %w", err)
		}
		if _, err := file.WriteString(markdown); err != nil {
			_ = file.Close()
			_ = os.Remove(path)
			return "", fmt.Errorf("save diagnostic report: %w", err)
		}
		if err := file.Close(); err != nil {
			return "", fmt.Errorf("close diagnostic report: %w", err)
		}
		return path, nil
	}
	return "", fmt.Errorf("save diagnostic report: no available filename")
}
