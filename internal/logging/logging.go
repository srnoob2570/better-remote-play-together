// Package logging owns the persistent, size-limited application log. The log
// is written next to the user's data so a diagnostic report can include the
// error that happened before the GUI opened.
package logging

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/srnoob2570/better-rpt/internal/config"
	"github.com/srnoob2570/better-rpt/internal/fsutil"
	"github.com/srnoob2570/better-rpt/internal/redact"
)

const (
	// MaxBytes is the size of one log file before rotation.
	MaxBytes int64 = 1 << 20
	// MaxBackups is the number of rotated files kept beside the current log.
	MaxBackups = 3
)

// Path returns the active log file path.
func Path(paths config.Paths) string {
	return filepath.Join(paths.LogsDir(), "better-rpt.log")
}

// Open creates the persistent logger. It always returns a usable logger: if
// the file cannot be opened, the error is returned and logging falls back to
// stderr only.
func Open(paths config.Paths, version string) (*slog.Logger, func() error, error) {
	home, _ := os.UserHomeDir()
	sanitizer := redact.New(home, username())
	stderr := &sanitizingWriter{next: os.Stderr, sanitizer: sanitizer}

	writer, openErr := newRotatingWriter(Path(paths), MaxBytes, MaxBackups)
	var output io.Writer = stderr
	closeFn := func() error { return nil }
	if openErr == nil {
		output = io.MultiWriter(stderr, &sanitizingWriter{next: writer, sanitizer: sanitizer})
		closeFn = writer.Close
	}
	logger := slog.New(slog.NewTextHandler(output, &slog.HandlerOptions{Level: slog.LevelInfo}))
	if version != "" {
		logger = logger.With("version", version)
	}
	if openErr != nil {
		return logger, closeFn, openErr
	}
	return logger, closeFn, nil
}

// Tail returns the last lines of the active log, already read-only. A missing
// log is reported as fs.ErrNotExist.
func Tail(paths config.Paths, lines int) ([]string, error) {
	if lines < 1 {
		lines = 200
	}
	file, err := os.Open(Path(paths))
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	offset := info.Size() - 256<<10
	if offset < 0 {
		offset = 0
	}
	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		return nil, err
	}
	data, err := io.ReadAll(io.LimitReader(file, 256<<10))
	if err != nil {
		return nil, err
	}
	all := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(all) > lines {
		all = all[len(all)-lines:]
	}
	if len(all) == 1 && all[0] == "" {
		return nil, nil
	}
	return all, nil
}

type sanitizingWriter struct {
	next      io.Writer
	sanitizer *redact.Sanitizer
}

func (w *sanitizingWriter) Write(data []byte) (int, error) {
	if w.next == nil {
		return len(data), nil
	}
	safe := w.sanitizer.Text(string(data))
	n, err := w.next.Write([]byte(safe))
	if err != nil {
		return n, err
	}
	// The sanitized text can be shorter than the input. Report the original
	// length so io.MultiWriter and slog never retry the write.
	return len(data), nil
}

type rotatingWriter struct {
	path     string
	maxBytes int64
	backups  int

	mu   sync.Mutex
	file *os.File
	size int64
}

func newRotatingWriter(path string, maxBytes int64, backups int) (*rotatingWriter, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("empty log path")
	}
	if maxBytes <= 0 {
		maxBytes = MaxBytes
	}
	if backups < 1 {
		backups = MaxBackups
	}
	if err := fsutil.EnsurePrivateDir(filepath.Dir(path)); err != nil {
		return nil, fmt.Errorf("prepare log directory: %w", err)
	}
	writer := &rotatingWriter{path: path, maxBytes: maxBytes, backups: backups}
	if err := writer.openLocked(); err != nil {
		return nil, err
	}
	return writer, nil
}

func (w *rotatingWriter) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return 0, errors.New("log file is closed")
	}
	if w.size > 0 && w.size+int64(len(data)) > w.maxBytes {
		if err := w.rotateLocked(); err != nil {
			return 0, err
		}
	}
	n, err := w.file.Write(data)
	w.size += int64(n)
	return n, err
}

func (w *rotatingWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	return err
}

func (w *rotatingWriter) openLocked() error {
	file, err := os.OpenFile(w.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open log: %w", err)
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return fmt.Errorf("stat log: %w", err)
	}
	w.file = file
	w.size = info.Size()
	if w.size > w.maxBytes {
		return w.rotateLocked()
	}
	return nil
}

func (w *rotatingWriter) rotateLocked() error {
	if w.file != nil {
		if err := w.file.Close(); err != nil {
			return fmt.Errorf("close log for rotation: %w", err)
		}
		w.file = nil
	}
	for index := w.backups - 1; index >= 1; index-- {
		from := fmt.Sprintf("%s.%d", w.path, index)
		to := fmt.Sprintf("%s.%d", w.path, index+1)
		if err := os.Rename(from, to); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("rotate %s: %w", from, err)
		}
	}
	if err := os.Rename(w.path, w.path+".1"); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("rotate current log: %w", err)
	}
	_ = os.Remove(fmt.Sprintf("%s.%d", w.path, w.backups+1))
	return w.openLocked()
}

func username() string {
	for _, key := range []string{"USER", "LOGNAME"} {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return ""
}
