// Package fsutil contains the small filesystem primitives shared by the
// configuration and session layers.
package fsutil

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// EnsurePrivateDir creates dir if needed and verifies it is a real directory
// owned by the current user with mode 0700. Existing permissions are tightened.
func EnsurePrivateDir(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create directory %s: %w", dir, err)
	}
	info, err := os.Lstat(dir)
	if err != nil {
		return fmt.Errorf("inspect directory %s: %w", dir, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing symlink directory %s", dir)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", dir)
	}
	if info.Mode().Perm()&0o077 != 0 {
		if err := os.Chmod(dir, 0o700); err != nil {
			return fmt.Errorf("tighten directory %s: %w", dir, err)
		}
	}
	if stat, ok := info.Sys().(*syscall.Stat_t); ok && int(stat.Uid) != os.Geteuid() {
		return fmt.Errorf("directory %s is owned by uid %d", dir, stat.Uid)
	}
	return nil
}

// AtomicWriteFile writes data to path using a temporary file in the same
// directory, fsyncs it, renames it over path and fsyncs the directory.
// Extended attributes and POSIX ACLs of an existing path are preserved; the
// write fails before replacing the original if one cannot be preserved.
// The parent directory must already exist and be private.
func AtomicWriteFile(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temporary file for %s: %w", path, err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if err := tmp.Chmod(perm); err != nil {
		tmp.Close()
		return fmt.Errorf("set permissions on temporary file: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write temporary file: %w", err)
	}
	if err := copyXattrs(path, tmpName); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("sync temporary file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temporary file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("replace %s: %w", path, err)
	}
	return syncDir(dir)
}

const (
	posixACLAccess = "system.posix_acl_access"
	selinuxContext = "security.selinux"
)

// copyXattrs mirrors source's extended attributes onto destination, which is
// the temporary file that will replace it. A missing source has nothing to
// preserve. SELinux labels are assigned by the kernel on creation; they are
// copied only when the new label differs from the old one.
func copyXattrs(source, destination string) error {
	names, err := listXattrs(source)
	if errors.Is(err, syscall.ENOENT) || errors.Is(err, syscall.ENOTSUP) || errors.Is(err, syscall.ENOSYS) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("list extended attributes of %s: %w", source, err)
	}
	present := make(map[string]bool, len(names))
	for _, name := range names {
		present[name] = true
	}
	// A new file inherits the directory's default ACL. The replacement must
	// keep the replaced file's ACL, not gain one the original did not have.
	if !present[posixACLAccess] {
		if err := syscall.Removexattr(destination, posixACLAccess); err != nil &&
			!errors.Is(err, syscall.ENODATA) && !errors.Is(err, syscall.ENOTSUP) && !errors.Is(err, syscall.ENOSYS) {
			return fmt.Errorf("remove inherited ACL from %s: %w", destination, err)
		}
	}
	for _, name := range names {
		value, err := getXattr(source, name)
		if errors.Is(err, syscall.ENODATA) {
			continue
		}
		if err != nil {
			return fmt.Errorf("read extended attribute %q of %s: %w", name, source, err)
		}
		if name == selinuxContext {
			current, err := getXattr(destination, name)
			if err == nil && bytes.Equal(current, value) {
				continue
			}
			if errors.Is(err, syscall.ENODATA) {
				// No label on the new file means SELinux is not labeling
				// files, so the old label has no effect.
				continue
			}
		}
		if err := syscall.Setxattr(destination, name, value, 0); err != nil {
			return fmt.Errorf("preserve extended attribute %q on %s: %w", name, destination, err)
		}
	}
	return nil
}

func listXattrs(path string) ([]string, error) {
	for attempt := 0; attempt < 3; attempt++ {
		size, err := syscall.Listxattr(path, nil)
		if err != nil {
			return nil, err
		}
		if size == 0 {
			return nil, nil
		}
		buffer := make([]byte, size)
		n, err := syscall.Listxattr(path, buffer)
		if errors.Is(err, syscall.ERANGE) {
			continue
		}
		if err != nil {
			return nil, err
		}
		var names []string
		for _, raw := range bytes.Split(buffer[:n], []byte{0}) {
			if len(raw) > 0 {
				names = append(names, string(raw))
			}
		}
		return names, nil
	}
	return nil, syscall.ERANGE
}

func getXattr(path, name string) ([]byte, error) {
	for attempt := 0; attempt < 3; attempt++ {
		size, err := syscall.Getxattr(path, name, nil)
		if err != nil {
			return nil, err
		}
		value := make([]byte, size)
		n, err := syscall.Getxattr(path, name, value)
		if errors.Is(err, syscall.ERANGE) {
			continue
		}
		if err != nil {
			return nil, err
		}
		return value[:n], nil
	}
	return nil, syscall.ERANGE
}

func syncDir(dir string) error {
	f, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("open directory %s: %w", dir, err)
	}
	defer f.Close()
	if err := f.Sync(); err != nil && !errors.Is(err, syscall.EINVAL) {
		return fmt.Errorf("sync directory %s: %w", dir, err)
	}
	return nil
}
