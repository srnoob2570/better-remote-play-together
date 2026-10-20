package config

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/srnoob2570/better-rpt/internal/domain"
	"github.com/srnoob2570/better-rpt/internal/fsutil"
)

const backupRetention = 10

// BackupKind distinguishes the legacy per-session transaction from the
// persistent dispatcher setup.
type BackupKind string

const (
	BackupSession    BackupKind = "session"
	BackupDispatcher BackupKind = "dispatcher"
)

// BackupMeta is the durable description of one localconfig backup.
type BackupMeta struct {
	Version                int        `json:"version"`
	Kind                   BackupKind `json:"kind,omitempty"`
	Source                 string     `json:"source"`
	SteamID64              string     `json:"steam_id_64"`
	AccountID              uint32     `json:"account_id"`
	AccountName            string     `json:"account_name"`
	AppID                  uint32     `json:"app_id"`
	TargetAppID            uint32     `json:"target_app_id"`
	SessionToken           string     `json:"session_token"`
	OriginalSHA256         string     `json:"original_sha256"`
	OriginalMode           uint32     `json:"original_mode"`
	OriginalRawTokenBase64 string     `json:"original_raw_token_base64,omitempty"`
	OriginalValue          string     `json:"original_value"`
	NewValue               string     `json:"new_value"`
	CreatedAppBlock        bool       `json:"created_app_block"`
	CreatedLaunchOptions   bool       `json:"created_launch_options"`
	CreatedAt              time.Time  `json:"created_at"`
}

// Backup is a backup file and its metadata.
type Backup struct {
	Path     string
	MetaPath string
	Meta     BackupMeta
}

// CreateBackup writes data to a new 0600 backup plus a 0600 metadata sibling.
// The directory is private and the current backup is never pruned here.
func CreateBackup(paths Paths, source string, data []byte, meta BackupMeta) (Backup, error) {
	if meta.Kind == "" {
		meta.Kind = BackupSession
	}
	if source == "" || meta.Source == "" || meta.AppID != domain.SpaceWarAppID || meta.NewValue == "" {
		return Backup{}, fmt.Errorf("%w: invalid backup metadata", ErrBackup)
	}
	switch meta.Kind {
	case BackupSession:
		if meta.TargetAppID == 0 || meta.TargetAppID == domain.SpaceWarAppID ||
			!domain.ValidSessionToken(meta.SessionToken) {
			return Backup{}, fmt.Errorf("%w: invalid session backup metadata", ErrBackup)
		}
	case BackupDispatcher:
		if meta.TargetAppID != 0 || meta.SessionToken != "" {
			return Backup{}, fmt.Errorf("%w: invalid dispatcher backup metadata", ErrBackup)
		}
	default:
		return Backup{}, fmt.Errorf("%w: unknown kind %q", ErrBackup, meta.Kind)
	}
	dir := paths.BackupsDir()
	if err := fsutil.EnsurePrivateDir(dir); err != nil {
		return Backup{}, fmt.Errorf("%w: %v", ErrBackup, err)
	}
	if meta.CreatedAt.IsZero() {
		meta.CreatedAt = time.Now().UTC()
	} else {
		meta.CreatedAt = meta.CreatedAt.UTC()
	}
	meta.Version = 1
	sum := sha256.Sum256(data)
	meta.OriginalSHA256 = hex.EncodeToString(sum[:])

	base := fmt.Sprintf("localconfig-%s-%s",
		meta.CreatedAt.Format("20060102T150405.000000000Z"),
		meta.OriginalSHA256[:12],
	)
	backupPath := filepath.Join(dir, base+".vdf")
	for suffix := 1; ; suffix++ {
		metaPath := strings.TrimSuffix(backupPath, ".vdf") + ".json"
		backupExists, backupErr := pathExists(backupPath)
		metaExists, metaErr := pathExists(metaPath)
		if backupErr != nil {
			return Backup{}, fmt.Errorf("%w: inspect %s: %v", ErrBackup, backupPath, backupErr)
		}
		if metaErr != nil {
			return Backup{}, fmt.Errorf("%w: inspect %s: %v", ErrBackup, metaPath, metaErr)
		}
		if !backupExists && !metaExists {
			break
		}
		backupPath = filepath.Join(dir, fmt.Sprintf("%s-%d.vdf", base, suffix))
	}
	metaPath := strings.TrimSuffix(backupPath, ".vdf") + ".json"

	if err := fsutil.AtomicWriteFile(backupPath, data, 0o600); err != nil {
		return Backup{}, fmt.Errorf("%w: write backup: %v", ErrBackup, err)
	}
	encoded, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		_ = os.Remove(backupPath)
		return Backup{}, fmt.Errorf("%w: encode metadata: %v", ErrBackup, err)
	}
	encoded = append(encoded, '\n')
	if err := fsutil.AtomicWriteFile(metaPath, encoded, 0o600); err != nil {
		_ = os.Remove(backupPath)
		return Backup{}, fmt.Errorf("%w: write metadata: %v", ErrBackup, err)
	}
	return Backup{Path: backupPath, MetaPath: metaPath, Meta: meta}, nil
}

// LoadBackup reads a backup and verifies its content against the metadata hash.
func LoadBackup(path string) (Backup, error) {
	metaPath := path
	if strings.HasSuffix(metaPath, ".vdf") {
		metaPath = strings.TrimSuffix(metaPath, ".vdf") + ".json"
	} else if !strings.HasSuffix(metaPath, ".json") {
		metaPath += ".json"
	}
	data, err := os.ReadFile(metaPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Backup{}, fmt.Errorf("%w: %s", ErrBackupMissing, metaPath)
		}
		return Backup{}, fmt.Errorf("%w: read %s: %v", ErrBackup, metaPath, err)
	}
	var meta BackupMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return Backup{}, fmt.Errorf("%w: parse %s: %v", ErrBackupCorrupt, metaPath, err)
	}
	if meta.Kind == "" {
		meta.Kind = BackupSession
	}
	if meta.Version != 1 || meta.Source == "" || meta.OriginalSHA256 == "" ||
		meta.AppID != domain.SpaceWarAppID || meta.NewValue == "" {
		return Backup{}, fmt.Errorf("%w: incomplete metadata %s", ErrBackupCorrupt, metaPath)
	}
	switch meta.Kind {
	case BackupSession:
		if !domain.ValidSessionToken(meta.SessionToken) ||
			meta.TargetAppID == 0 || meta.TargetAppID == domain.SpaceWarAppID {
			return Backup{}, fmt.Errorf("%w: incomplete session metadata %s", ErrBackupCorrupt, metaPath)
		}
	case BackupDispatcher:
		if meta.SessionToken != "" || meta.TargetAppID != 0 {
			return Backup{}, fmt.Errorf("%w: incomplete dispatcher metadata %s", ErrBackupCorrupt, metaPath)
		}
	default:
		return Backup{}, fmt.Errorf("%w: unknown kind %q in %s", ErrBackupCorrupt, meta.Kind, metaPath)
	}
	metaInfo, err := os.Stat(metaPath)
	if err != nil {
		return Backup{}, fmt.Errorf("%w: stat %s: %v", ErrBackupCorrupt, metaPath, err)
	}
	if metaInfo.Mode().Perm()&0o077 != 0 {
		return Backup{}, fmt.Errorf("%w: metadata is not private: %s", ErrBackupCorrupt, metaPath)
	}
	backupPath := strings.TrimSuffix(metaPath, ".json") + ".vdf"
	backupData, err := os.ReadFile(backupPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Backup{}, fmt.Errorf("%w: %s", ErrBackupMissing, backupPath)
		}
		return Backup{}, fmt.Errorf("%w: read %s: %v", ErrBackup, backupPath, err)
	}
	sum := sha256.Sum256(backupData)
	if hex.EncodeToString(sum[:]) != meta.OriginalSHA256 {
		return Backup{}, fmt.Errorf("%w: hash mismatch for %s", ErrBackupCorrupt, backupPath)
	}
	info, err := os.Stat(backupPath)
	if err != nil {
		return Backup{}, fmt.Errorf("%w: stat %s: %v", ErrBackup, backupPath, err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		return Backup{}, fmt.Errorf("%w: backup is not private: %s", ErrBackupCorrupt, backupPath)
	}
	return Backup{Path: backupPath, MetaPath: metaPath, Meta: meta}, nil
}

// ListBackups returns backups newest first. It ignores unrelated files but
// fails on a corrupt metadata sibling.
func ListBackups(paths Paths) ([]Backup, error) {
	dir := paths.BackupsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("%w: read backup directory: %v", ErrBackup, err)
	}
	info, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("%w: stat backup directory: %v", ErrBackup, err)
	}
	if !info.IsDir() || info.Mode().Perm()&0o077 != 0 {
		return nil, fmt.Errorf("%w: backup directory is not private: %s", ErrBackup, dir)
	}
	backups := make([]Backup, 0)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(paths.BackupsDir(), strings.TrimSuffix(entry.Name(), ".json"))
		backup, err := LoadBackup(path)
		if err != nil {
			return nil, err
		}
		backups = append(backups, backup)
	}
	sort.SliceStable(backups, func(i, j int) bool {
		if backups[i].Meta.CreatedAt.Equal(backups[j].Meta.CreatedAt) {
			return backups[i].Path > backups[j].Path
		}
		return backups[i].Meta.CreatedAt.After(backups[j].Meta.CreatedAt)
	})
	return backups, nil
}

// FindBackupBySession returns the newest backup for a session token.
func FindBackupBySession(paths Paths, token string) (Backup, error) {
	backups, err := ListBackups(paths)
	if err != nil {
		return Backup{}, err
	}
	for _, backup := range backups {
		if backup.Meta.Kind == BackupDispatcher {
			continue
		}
		if backup.Meta.SessionToken == token {
			return backup, nil
		}
	}
	return Backup{}, fmt.Errorf("%w: session %s", ErrBackupMissing, token)
}

// PruneBackups keeps the newest keep backups, never deleting protect.
func pathExists(path string) (bool, error) {
	_, err := os.Lstat(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func PruneBackups(paths Paths, keep int, protect string) error {
	if keep < 1 {
		return fmt.Errorf("%w: invalid retention %d", ErrBackup, keep)
	}
	backups, err := ListBackups(paths)
	if err != nil {
		return err
	}
	removed := 0
	for _, backup := range backups {
		if removed < len(backups)-keep {
			if backup.Path == protect || backup.MetaPath == protect {
				continue
			}
			if err := os.Remove(backup.Path); err != nil && !errors.Is(err, fs.ErrNotExist) {
				return fmt.Errorf("%w: remove %s: %v", ErrBackup, backup.Path, err)
			}
			if err := os.Remove(backup.MetaPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
				return fmt.Errorf("%w: remove %s: %v", ErrBackup, backup.MetaPath, err)
			}
			removed++
		}
	}
	return nil
}
