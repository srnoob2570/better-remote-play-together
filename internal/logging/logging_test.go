package logging

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/srnoob2570/better-rpt/internal/config"
)

func TestOpenRedactsAndRotates(t *testing.T) {
	dir := t.TempDir()
	paths := config.Paths{DataDir: filepath.Join(dir, "data")}
	logger, closeFn, err := Open(paths, "0.2.0")
	if err != nil {
		t.Fatal(err)
	}
	logger.Info("secret", "path", filepath.Join(dir, "alice", "token"),
		"token", "fedcba9876543210fedcba9876543210")
	if err := closeFn(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(Path(paths))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "fedcba9876543210fedcba9876543210") {
		t.Fatalf("token leaked: %s", data)
	}
	if !strings.Contains(string(data), "[token]") {
		t.Fatalf("expected token placeholder: %s", data)
	}
	if !strings.Contains(string(data), "version=0.2.0") {
		t.Fatalf("expected version attribute: %s", data)
	}
}

func TestRotatingWriterKeepsBackups(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.log")
	writer, err := newRotatingWriter(path, 20, 2)
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()
	for i := 0; i < 5; i++ {
		if _, err := writer.Write([]byte(strings.Repeat("x", 15) + "\n")); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := os.Stat(path + ".1"); err != nil {
		t.Fatalf("missing backup .1: %v", err)
	}
	if _, err := os.Stat(path + ".2"); err != nil {
		t.Fatalf("missing backup .2: %v", err)
	}
}

func TestTail(t *testing.T) {
	dir := t.TempDir()
	paths := config.Paths{DataDir: filepath.Join(dir, "data")}
	logger, closeFn, err := Open(paths, "test")
	if err != nil {
		t.Fatal(err)
	}
	logger.Info("one")
	logger.Warn("two")
	if err := closeFn(); err != nil {
		t.Fatal(err)
	}
	lines, err := Tail(paths, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 1 || !strings.Contains(lines[0], "two") {
		t.Fatalf("unexpected tail: %#v", lines)
	}
}
