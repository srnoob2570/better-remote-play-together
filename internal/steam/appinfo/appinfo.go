// Package appinfo parses Steam's binary appinfo.vdf cache (format v41).
//
// It reads the cache into memory, validates the header, entry boundaries and
// string table, and decodes an app record on demand. A corrupt app record is
// skipped without discarding the rest of the cache. It never writes to Steam.
package appinfo

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"sync"
	"unicode/utf8"
)

const (
	magicV41 = 0x07564429
	magicV40 = 0x07564428

	headerSize        = 16
	fixedEntrySize    = 60
	maxAppEntries     = 1 << 20
	maxStringEntries  = 1 << 20
	maxParserWarnings = 100
)

var (
	// ErrNotAppInfo means the file is missing or is not an appinfo cache.
	ErrNotAppInfo = errors.New("not a Steam appinfo file")
	// ErrUnsupportedVersion means the magic belongs to a version this parser
	// does not understand.
	ErrUnsupportedVersion = errors.New("unsupported Steam appinfo version")
	// ErrCorrupt means the file-level structure (header, footer, string table
	// or entry boundaries) is invalid.
	ErrCorrupt = errors.New("appinfo file is corrupt")
	// ErrAppCorrupt means one app record is invalid; other records remain usable.
	ErrAppCorrupt = errors.New("appinfo record is corrupt")
	// ErrAppNotFound means the cache has no usable record for an app ID.
	ErrAppNotFound = errors.New("appinfo record not found")
)

// Common is the app's common metadata.
type Common struct {
	Name       string
	Type       string
	OSList     string
	InstallDir string
}

// Launch is one entry of config.launch.
type Launch struct {
	Key         string
	OSList      string
	Arch        string
	Executable  string
	Arguments   string
	WorkingDir  string
	Type        string
	Description string
	BetaKeys    []string
	OwnsDLC     bool
}

// App is one usable appinfo record.
type App struct {
	ID          uint32
	InfoState   uint32
	LastUpdated uint32
	Common      Common
	Launch      []Launch
}

// Warning records an app record that was skipped.
type Warning struct {
	AppID  uint32
	Offset int64
	Err    error
}

func (w Warning) Error() string {
	return fmt.Sprintf("appinfo app %d at offset %d: %v", w.AppID, w.Offset, w.Err)
}

func (w Warning) Unwrap() error {
	return w.Err
}

// File is an indexed appinfo cache. Entry boundaries are validated on open;
// each app record is decoded on demand, so callers only pay for the records
// they actually read.
type File struct {
	mu       sync.Mutex
	strings  []string
	entries  map[uint32]appEntry
	ids      []uint32
	decoded  map[uint32]App
	failed   map[uint32]bool
	warnings []Warning
}

type appEntry struct {
	data   []byte
	offset int64
}

// Open reads and indexes an appinfo.vdf file. A file being rewritten by Steam
// gets one retry; other failures are returned immediately. Records are decoded
// by App or Apps.
func Open(path string) (*File, error) {
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		f, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("%w: open %s: %w", ErrNotAppInfo, path, err)
		}
		info, statErr := f.Stat()
		if statErr != nil {
			_ = f.Close()
			return nil, fmt.Errorf("%w: stat %s: %v", ErrCorrupt, path, statErr)
		}
		file, parseErr := Parse(f, info.Size())
		closeErr := f.Close()
		if parseErr == nil && closeErr == nil {
			return file, nil
		}
		if parseErr == nil {
			parseErr = fmt.Errorf("%w: close %s: %v", ErrCorrupt, path, closeErr)
		}
		lastErr = parseErr
		if !errors.Is(parseErr, ErrCorrupt) || attempt > 0 {
			break
		}
	}
	return nil, lastErr
}

// Parse parses size bytes from r. The caller retains ownership of r.
func Parse(r io.ReaderAt, size int64) (*File, error) {
	if size <= 0 {
		return nil, fmt.Errorf("%w: empty file", ErrCorrupt)
	}
	if size > int64(^uint(0)>>1) {
		return nil, fmt.Errorf("%w: file too large", ErrCorrupt)
	}
	data := make([]byte, int(size))
	n, err := r.ReadAt(data, 0)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("%w: read: %v", ErrCorrupt, err)
	}
	if int64(n) != size {
		return nil, fmt.Errorf("%w: short read: got %d of %d bytes", ErrCorrupt, n, size)
	}
	return parseData(data)
}

func parseData(data []byte) (*File, error) {
	if len(data) < headerSize {
		return nil, fmt.Errorf("%w: short header", ErrCorrupt)
	}
	magic := binary.LittleEndian.Uint32(data[0:4])
	switch magic {
	case magicV41:
	case magicV40:
		return nil, fmt.Errorf("%w: v40", ErrUnsupportedVersion)
	default:
		return nil, fmt.Errorf("%w: magic 0x%08x", ErrNotAppInfo, magic)
	}
	tableOffset64 := int64(binary.LittleEndian.Uint64(data[8:16]))
	if tableOffset64 < headerSize+4 || tableOffset64 > int64(len(data))-4 {
		return nil, fmt.Errorf("%w: string table offset %d outside file", ErrCorrupt, tableOffset64)
	}
	tableOffset := int(tableOffset64)
	if binary.LittleEndian.Uint32(data[tableOffset-4:tableOffset]) != 0 {
		return nil, fmt.Errorf("%w: missing entry footer before string table", ErrCorrupt)
	}
	strings, err := parseStringTable(data[tableOffset:])
	if err != nil {
		return nil, err
	}

	file := &File{
		strings: strings,
		entries: make(map[uint32]appEntry),
		decoded: make(map[uint32]App),
		failed:  make(map[uint32]bool),
	}
	pos := headerSize
	sawFooter := false
	entryCount := 0
	for pos < tableOffset {
		if tableOffset-pos < 4 {
			return nil, fmt.Errorf("%w: truncated app id", ErrCorrupt)
		}
		appID := binary.LittleEndian.Uint32(data[pos : pos+4])
		pos += 4
		if appID == 0 {
			if pos != tableOffset {
				return nil, fmt.Errorf("%w: entry footer is not immediately before string table", ErrCorrupt)
			}
			sawFooter = true
			break
		}
		if tableOffset-pos < 4 {
			return nil, fmt.Errorf("%w: truncated size for app %d", ErrCorrupt, appID)
		}
		size := int(binary.LittleEndian.Uint32(data[pos : pos+4]))
		pos += 4
		if size < 0 || size > tableOffset-pos {
			return nil, fmt.Errorf("%w: app %d size %d exceeds entry area", ErrCorrupt, appID, size)
		}
		end := pos + size
		if size < fixedEntrySize {
			file.addWarning(Warning{
				AppID:  appID,
				Offset: int64(pos),
				Err:    fmt.Errorf("%w: entry is %d bytes, need at least %d", ErrAppCorrupt, size, fixedEntrySize),
			})
			pos = end
			continue
		}
		entryCount++
		if entryCount > maxAppEntries {
			return nil, fmt.Errorf("%w: too many app entries", ErrCorrupt)
		}
		file.entries[appID] = appEntry{data: data[pos:end], offset: int64(pos)}
		pos = end
	}
	if !sawFooter {
		return nil, fmt.Errorf("%w: missing entry footer", ErrCorrupt)
	}
	file.ids = make([]uint32, 0, len(file.entries))
	for id := range file.entries {
		file.ids = append(file.ids, id)
	}
	sort.Slice(file.ids, func(i, j int) bool { return file.ids[i] < file.ids[j] })
	return file, nil
}

func (f *File) addWarning(warning Warning) {
	if len(f.warnings) >= maxParserWarnings {
		return
	}
	f.warnings = append(f.warnings, warning)
}

func parseStringTable(data []byte) ([]string, error) {
	if len(data) < 4 {
		return nil, fmt.Errorf("%w: truncated string table", ErrCorrupt)
	}
	count := binary.LittleEndian.Uint32(data[0:4])
	if uint64(count) > uint64(len(data)-4) {
		return nil, fmt.Errorf("%w: impossible string count %d", ErrCorrupt, count)
	}
	if count > maxStringEntries {
		return nil, fmt.Errorf("%w: too many strings %d", ErrCorrupt, count)
	}
	capacity := count
	if capacity > 4096 {
		capacity = 4096
	}
	strings := make([]string, 0, capacity)
	pos := 4
	for i := uint32(0); i < count; i++ {
		n := bytes.IndexByte(data[pos:], 0)
		if n < 0 {
			return nil, fmt.Errorf("%w: unterminated string %d", ErrCorrupt, i)
		}
		value := string(data[pos : pos+n])
		if !validUTF8(value) {
			return nil, fmt.Errorf("%w: string %d is not UTF-8", ErrCorrupt, i)
		}
		strings = append(strings, value)
		pos += n + 1
	}
	for _, b := range data[pos:] {
		if b != 0 {
			return nil, fmt.Errorf("%w: trailing string table data", ErrCorrupt)
		}
	}
	return strings, nil
}

func parseAppRecord(appID uint32, data []byte, strings []string, offset int64) (App, error) {
	if len(data) < fixedEntrySize {
		return App{}, fmt.Errorf("%w: short app record", ErrAppCorrupt)
	}
	infoState := binary.LittleEndian.Uint32(data[0:4])
	lastUpdated := binary.LittleEndian.Uint32(data[4:8])
	vdf := data[fixedEntrySize:]
	binaryHash := data[40:60]
	if !allZero(binaryHash) {
		sum := sha1.Sum(vdf)
		if !bytes.Equal(sum[:], binaryHash) {
			return App{}, fmt.Errorf("%w: binary VDF SHA-1 mismatch", ErrAppCorrupt)
		}
	}

	p := binaryParser{data: vdf, strings: strings}
	root, err := p.parseDocument()
	if err != nil {
		return App{}, fmt.Errorf("%w: %v", ErrAppCorrupt, err)
	}
	rootObject, ok := root["appinfo"].(map[string]any)
	if !ok {
		return App{}, fmt.Errorf("%w: missing appinfo root object", ErrAppCorrupt)
	}
	common, launch, err := extractApp(rootObject)
	if err != nil {
		return App{}, fmt.Errorf("%w: %v", ErrAppCorrupt, err)
	}
	return App{
		ID:          appID,
		InfoState:   infoState,
		LastUpdated: lastUpdated,
		Common:      common,
		Launch:      launch,
	}, nil
}

// Apps decodes and returns every usable record in app ID order. Prefer App
// when only specific IDs are needed.
func (f *File) Apps() []App {
	if f == nil {
		return nil
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	apps := make([]App, 0, len(f.ids))
	for _, id := range f.ids {
		if app, ok := f.appLocked(id); ok {
			apps = append(apps, app)
		}
	}
	return apps
}

// App returns one usable record, decoding it on first use.
func (f *File) App(id uint32) (App, bool) {
	if f == nil {
		return App{}, false
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.appLocked(id)
}

func (f *File) appLocked(id uint32) (App, bool) {
	if app, ok := f.decoded[id]; ok {
		return app, true
	}
	if f.failed[id] {
		return App{}, false
	}
	entry, ok := f.entries[id]
	if !ok {
		return App{}, false
	}
	app, err := parseAppRecord(id, entry.data, f.strings, entry.offset)
	if err != nil {
		f.addWarning(Warning{AppID: id, Offset: entry.offset, Err: err})
		f.failed[id] = true
		return App{}, false
	}
	f.decoded[id] = app
	return app, true
}

// Warnings returns a copy of the records that were skipped. Per-record
// warnings appear after the record is decoded by App or Apps.
func (f *File) Warnings() []Warning {
	if f == nil {
		return nil
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]Warning(nil), f.warnings...)
}

func allZero(b []byte) bool {
	for _, v := range b {
		if v != 0 {
			return false
		}
	}
	return true
}

func validUTF8(s string) bool {
	return utf8.ValidString(s)
}
