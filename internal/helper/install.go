package helper

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/srnoob2570/better-rpt/internal/config"
	"github.com/srnoob2570/better-rpt/internal/fsutil"
)

const installRecordVersion = 1

type fileSignature struct {
	Size       int64     `json:"size"`
	ModTime    time.Time `json:"mod_time"`
	ChangeTime time.Time `json:"change_time,omitempty"`
}

func (s fileSignature) equal(other fileSignature) bool {
	return s.Size == other.Size &&
		s.ModTime.Equal(other.ModTime) &&
		s.ChangeTime.Equal(other.ChangeTime)
}

type installRecord struct {
	Version     int           `json:"version"`
	SourceKind  string        `json:"source_kind"`
	Source      fileSignature `json:"source"`
	Destination fileSignature `json:"destination"`
	SHA256      string        `json:"sha256"`
}

// Errors returned by the stable helper installer.
var (
	ErrSourceMissing = errors.New("stable helper source not found")
	ErrSourceInvalid = errors.New("stable helper source is invalid")
	ErrInstall       = errors.New("stable helper installation failed")
	ErrActiveSession = errors.New("cannot replace the helper during an active session")
)

// Installed describes one installer result.
type Installed struct {
	Source      string
	Destination string
	SHA256      string
	Changed     bool
}

// Installer copies the packaged helper to the stable XDG path.
type Installer struct {
	DataDir    string
	AppDir     string
	SystemPath string
}

// NewInstaller uses the packaged helper locations from a .deb or AppImage.
func NewInstaller(paths config.Paths) *Installer {
	return &Installer{
		DataDir:    paths.DataDir,
		AppDir:     os.Getenv("APPDIR"),
		SystemPath: "/usr/lib/better-rpt/better-rpt-helper",
	}
}

// Destination returns the stable helper path.
func (i *Installer) Destination() string {
	return filepath.Join(i.DataDir, "bin", "better-rpt-helper")
}

// Sources returns candidate packaged helpers in preference order. An AppImage
// uses its own bundle; a .deb uses /usr/lib.
func (i *Installer) Sources() []string {
	sources := make([]string, 0, 2)
	if i.AppDir != "" {
		sources = append(sources, filepath.Join(i.AppDir, "usr", "lib", "better-rpt", "better-rpt-helper"))
	}
	if i.SystemPath != "" {
		sources = append(sources, i.SystemPath)
	}
	return sources
}

// Install copies the first valid packaged helper. It refuses to replace a
// different helper while a session is active.
func (i *Installer) Install(activeSession bool) (Installed, error) {
	source, sourceInfo, err := i.findSource()
	if err != nil {
		return Installed{}, err
	}
	destination := i.Destination()
	if err := fsutil.EnsurePrivateDir(filepath.Dir(destination)); err != nil {
		return Installed{}, fmt.Errorf("%w: %v", ErrInstall, err)
	}
	if recorded, ok := i.recordedHash(source, sourceInfo, destination); ok {
		current, err := os.Lstat(source)
		if err != nil {
			return Installed{}, fmt.Errorf("%w: inspect %s: %v", ErrSourceInvalid, source, err)
		}
		if signature(sourceInfo).equal(signature(current)) {
			if err := ensureMode(destination, 0o755); err != nil {
				return Installed{}, err
			}
			i.writeRecord(source, sourceInfo, destination, recorded)
			return Installed{Source: source, Destination: destination, SHA256: recorded}, nil
		}
		sourceInfo = current
	}
	data, sum, sourceInfo, err := i.readSourceContent(source, sourceInfo)
	if err != nil {
		return Installed{}, err
	}
	if existing, err := hashFile(destination); err == nil {
		if existing == sum {
			if err := ensureMode(destination, 0o755); err != nil {
				return Installed{}, err
			}
			i.writeRecord(source, sourceInfo, destination, sum)
			return Installed{Source: source, Destination: destination, SHA256: sum}, nil
		}
		if activeSession {
			return Installed{}, fmt.Errorf("%w: destination %s differs", ErrActiveSession, destination)
		}
	}
	if err := fsutil.AtomicWriteFile(destination, data, 0o755); err != nil {
		return Installed{}, fmt.Errorf("%w: %v", ErrInstall, err)
	}
	i.writeRecord(source, sourceInfo, destination, sum)
	return Installed{Source: source, Destination: destination, SHA256: sum, Changed: true}, nil
}

func (i *Installer) findSource() (string, fs.FileInfo, error) {
	for _, candidate := range i.Sources() {
		info, err := os.Lstat(candidate)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return "", nil, fmt.Errorf("%w: inspect %s: %v", ErrSourceInvalid, candidate, err)
		}
		if !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 || info.Mode().Perm()&0o022 != 0 {
			return "", nil, fmt.Errorf("%w: %s", ErrSourceInvalid, candidate)
		}
		return candidate, info, nil
	}
	return "", nil, ErrSourceMissing
}

func (i *Installer) readSourceContent(source string, info fs.FileInfo) ([]byte, string, fs.FileInfo, error) {
	data, err := os.ReadFile(source)
	if err != nil {
		return nil, "", nil, fmt.Errorf("%w: read %s: %v", ErrSourceInvalid, source, err)
	}
	// Re-read the metadata after the content. If the file changed while it
	// was being read, the signature no longer matches and the record is not
	// safe to use on the next launch.
	current, err := os.Lstat(source)
	if err != nil || !signature(info).equal(signature(current)) {
		return nil, "", nil, fmt.Errorf("%w: %s changed while reading", ErrSourceInvalid, source)
	}
	sum := sha256.Sum256(data)
	return data, hex.EncodeToString(sum[:]), current, nil
}

func (i *Installer) readSource() (string, fs.FileInfo, []byte, string, error) {
	source, info, err := i.findSource()
	if err != nil {
		return "", nil, nil, "", err
	}
	data, sum, current, err := i.readSourceContent(source, info)
	if err != nil {
		return "", nil, nil, "", err
	}
	return source, current, data, sum, nil
}

func (i *Installer) recordedHash(source string, sourceInfo fs.FileInfo, destination string) (string, bool) {
	data, err := os.ReadFile(i.recordPath())
	if err != nil {
		return "", false
	}
	var record installRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return "", false
	}
	if record.Version != installRecordVersion || !validHash(record.SHA256) ||
		record.SourceKind != i.sourceKind(source) ||
		!record.Source.equal(signature(sourceInfo)) {
		return "", false
	}
	info, err := os.Lstat(destination)
	if err != nil || !info.Mode().IsRegular() ||
		!record.Destination.equal(signature(info)) {
		return "", false
	}
	return record.SHA256, true
}

func (i *Installer) writeRecord(source string, sourceInfo fs.FileInfo, destination, sum string) {
	info, err := os.Lstat(destination)
	if err != nil || !info.Mode().IsRegular() {
		return
	}
	record := installRecord{
		Version:     installRecordVersion,
		SourceKind:  i.sourceKind(source),
		Source:      signature(sourceInfo),
		Destination: signature(info),
		SHA256:      sum,
	}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return
	}
	data = append(data, '\n')
	_ = fsutil.AtomicWriteFile(i.recordPath(), data, 0o600)
}

func (i *Installer) recordPath() string {
	return filepath.Join(i.DataDir, "bin", ".better-rpt-helper.install.json")
}

func (i *Installer) sourceKind(source string) string {
	if i.AppDir != "" &&
		filepath.Clean(source) == filepath.Clean(filepath.Join(i.AppDir, "usr", "lib", "better-rpt", "better-rpt-helper")) {
		return "appimage"
	}
	return "system"
}

func signature(info fs.FileInfo) fileSignature {
	result := fileSignature{Size: info.Size(), ModTime: info.ModTime().UTC()}
	if raw, ok := info.Sys().(*syscall.Stat_t); ok {
		result.ChangeTime = time.Unix(raw.Ctim.Sec, raw.Ctim.Nsec).UTC()
	}
	return result
}

func validHash(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func hashFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func ensureMode(path string, mode os.FileMode) error {
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("%w: stat %s: %v", ErrInstall, path, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%w: %s is not a regular file", ErrInstall, path)
	}
	if info.Mode().Perm() != mode {
		if err := os.Chmod(path, mode); err != nil {
			return fmt.Errorf("%w: chmod %s: %v", ErrInstall, path, err)
		}
	}
	return nil
}
