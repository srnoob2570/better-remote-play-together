package fsutil

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestAtomicWriteFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	if err := AtomicWriteFile(path, []byte("hello"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("data = %q", data)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("mode = %o, want 600", got)
	}

	if err := AtomicWriteFile(path, []byte("world"), 0o600); err != nil {
		t.Fatalf("replace: %v", err)
	}
	data, err = os.ReadFile(path)
	if err != nil {
		t.Fatalf("read replacement: %v", err)
	}
	if string(data) != "world" {
		t.Fatalf("replacement = %q", data)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("temporary files left behind: %v", entries)
	}
}

func TestAtomicWriteFilePreservesExtendedAttributes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := AtomicWriteFile(path, []byte("hello"), 0o640); err != nil {
		t.Fatalf("write: %v", err)
	}
	name := "user.better-rpt-test"
	if err := syscall.Setxattr(path, name, []byte("kept"), 0); err != nil {
		t.Skipf("xattrs unsupported: %v", err)
	}

	if err := AtomicWriteFile(path, []byte("world"), 0o640); err != nil {
		t.Fatalf("replace: %v", err)
	}
	size, err := syscall.Getxattr(path, name, nil)
	if err != nil {
		t.Fatalf("read xattr: %v", err)
	}
	value := make([]byte, size)
	if _, err := syscall.Getxattr(path, name, value); err != nil {
		t.Fatalf("read xattr: %v", err)
	}
	if string(value) != "kept" {
		t.Fatalf("xattr = %q, want %q", value, "kept")
	}
}

func TestEnsurePrivateDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "private")
	if err := EnsurePrivateDir(dir); err != nil {
		t.Fatalf("create: %v", err)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o700 {
		t.Fatalf("mode = %o, want 700", got)
	}

	open := filepath.Join(t.TempDir(), "open")
	if err := os.Mkdir(open, 0o755); err != nil {
		t.Fatalf("mkdir open: %v", err)
	}
	if err := EnsurePrivateDir(open); err != nil {
		t.Fatalf("tighten: %v", err)
	}
	info, err = os.Stat(open)
	if err != nil {
		t.Fatalf("stat open: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o700 {
		t.Fatalf("tightened mode = %o, want 700", got)
	}

	readonly := filepath.Join(t.TempDir(), "readonly")
	if err := os.Mkdir(readonly, 0o500); err != nil {
		t.Fatalf("mkdir readonly: %v", err)
	}
	if err := EnsurePrivateDir(readonly); err != nil {
		t.Fatalf("restore owner write: %v", err)
	}
	info, err = os.Stat(readonly)
	if err != nil {
		t.Fatalf("stat readonly: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o700 {
		t.Fatalf("readonly mode = %o, want 700", got)
	}
}

func TestEnsurePrivateDirRejectsSymlink(t *testing.T) {
	base := t.TempDir()
	target := filepath.Join(base, "target")
	if err := os.Mkdir(target, 0o700); err != nil {
		t.Fatalf("mkdir target: %v", err)
	}
	link := filepath.Join(base, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	err := EnsurePrivateDir(link)
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("error = %v, want symlink rejection", err)
	}
}
