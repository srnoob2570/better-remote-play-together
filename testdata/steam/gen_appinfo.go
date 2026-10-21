//go:build ignore

// gen_appinfo.go generates the synthetic appinfo.vdf fixtures for Better RPT.
// Run from the repository root:
//
//	go run testdata/steam/gen_appinfo.go -out testdata/steam/fixtures/primary/appcache
package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

const (
	magicV41 = 0x07564429
	magicV40 = 0x07564428
)

type field struct {
	key string
	val any // string, int32, or object
}

type object []field

type launchDef struct {
	key        string
	executable string
	arguments  string
	workdir    string
	typ        string
	desc       string
	oslist     string
	osarch     string
}

type appDef struct {
	id         uint32
	name       string
	typ        string
	installDir string
	oslist     string
	launches   []launchDef
	badKey     bool
}

var keyNames = []string{
	"appinfo",
	"common",
	"name",
	"type",
	"oslist",
	"installdir",
	"config",
	"launch",
	"0",
	"1",
	"executable",
	"arguments",
	"workingdir",
	"description",
	"osarch",
}

var keyIndex = func() map[string]uint32 {
	m := make(map[string]uint32, len(keyNames))
	for i, k := range keyNames {
		m[k] = uint32(i)
	}
	return m
}()

func main() {
	out := flag.String("out", "testdata/steam/fixtures/primary/appcache", "output directory for appinfo binaries")
	flag.Parse()

	if err := os.MkdirAll(*out, 0o755); err != nil {
		fatal(err)
	}

	apps := []appDef{
		{
			id: 480, name: "Spacewar", typ: "Game", installDir: "Spacewar",
			oslist: "windows,macos,linux",
			launches: []launchDef{{
				key: "0", executable: "SteamworksExample.exe", typ: "none",
				desc: "Play Spacewar", oslist: "windows", osarch: "64",
			}},
		},
		{
			id: 900001, name: "Native Game", typ: "Game", installDir: "Native Game",
			oslist: "linux,windows",
			launches: []launchDef{
				{
					key: "0", executable: "game.x86_64", arguments: "--fixture",
					workdir: ".", typ: "default", desc: "Play Native Game",
					oslist: "linux", osarch: "64",
				},
				{
					key: "1", executable: "game.exe", workdir: ".",
					typ: "default", desc: "Play Native Game (Windows)",
					oslist: "windows", osarch: "64",
				},
			},
		},
		{
			id: 900002, name: "Proton Game", typ: "Game", installDir: "Proton Game",
			oslist: "windows",
			launches: []launchDef{{
				key: "0", executable: "Game.exe", arguments: "-windowed",
				workdir: ".", typ: "default", desc: "Play Proton Game",
				oslist: "windows", osarch: "64",
			}},
		},
		{id: 900003, name: "Proton Test", typ: "Tool", installDir: "Proton Test", oslist: "linux"},
		{id: 900004, name: "SteamLinuxRuntime Test", typ: "Tool", installDir: "SteamLinuxRuntime_test", oslist: "linux"},
		{
			id: 900005, name: "Not Installed Game", typ: "Game", installDir: "Not Installed Game",
			oslist: "linux",
			launches: []launchDef{{
				key: "0", executable: "game.x86_64", typ: "default",
				desc: "Play Not Installed Game", oslist: "linux", osarch: "64",
			}},
		},
		{
			id: 900006, name: "Missing Exe", typ: "Game", installDir: "Missing Exe",
			oslist: "linux",
			launches: []launchDef{{
				key: "0", executable: "missing.x86_64", typ: "default",
				desc: "Play Missing Exe", oslist: "linux", osarch: "64",
			}},
		},
		{
			id: 900007, name: "Secondary Native", typ: "Game", installDir: "Secondary Native",
			oslist: "linux",
			launches: []launchDef{{
				key: "0", executable: "game.x86_64", typ: "default",
				desc: "Play Secondary Native", oslist: "linux", osarch: "64",
			}},
		},
		{
			id: 900008, name: "Legacy Game", typ: "Game", installDir: "Legacy Game",
			oslist: "linux",
			launches: []launchDef{{
				key: "0", executable: "game.x86_64", typ: "default",
				desc: "Play Legacy Game", oslist: "linux", osarch: "64",
			}},
		},
		{id: 900009, name: "No Launch Config", typ: "Game", installDir: "No Launch Config", oslist: "linux"},
		{
			id: 900011, name: "Template Launch", typ: "Game", installDir: "Template Launch",
			oslist: "linux",
			launches: []launchDef{{
				key: "0", executable: "game.x86_64", arguments: "--fixture %command%",
				typ: "default", desc: "Template Launch", oslist: "linux", osarch: "64",
			}},
		},
		{
			id: 900012, name: "Path Escape", typ: "Game", installDir: "Path Escape",
			oslist: "linux",
			launches: []launchDef{{
				key: "0", executable: "../evil", typ: "default",
				desc: "Path Escape", oslist: "linux", osarch: "64",
			}},
		},
	}

	badApp := appDef{id: 900013, name: "Bad Key", typ: "Game", installDir: "Bad Key", oslist: "linux", badKey: true}

	files := map[string][]byte{
		"appinfo.vdf":         makeFile(apps, magicV41, false),
		"appinfo_v40.vdf":     makeFile(apps, magicV40, false),
		"appinfo_corrupt.vdf": makeFile(apps, magicV41, true),
		"appinfo_bad_key.vdf": makeFile(append(apps, badApp), magicV41, false),
	}

	for name, data := range files {
		path := filepath.Join(*out, name)
		if err := os.WriteFile(path, data, 0o644); err != nil {
			fatal(fmt.Errorf("write %s: %w", path, err))
		}
		fmt.Printf("wrote %s (%d bytes)\n", path, len(data))
	}
	fmt.Printf("apps=%d keys=%d\n", len(apps), len(keyNames))
}

func makeFile(apps []appDef, magic uint32, badOffset bool) []byte {
	var entries bytes.Buffer
	for _, app := range apps {
		entries.Write(makeEntry(app))
	}
	footer := make([]byte, 4)
	tableOffset := int64(16 + entries.Len() + len(footer))
	if badOffset {
		tableOffset += 1000
	}

	var out bytes.Buffer
	writeU32(&out, magic)
	writeU32(&out, 1)
	var off [8]byte
	binary.LittleEndian.PutUint64(off[:], uint64(tableOffset))
	out.Write(off[:])
	out.Write(entries.Bytes())
	out.Write(footer)

	writeU32(&out, uint32(len(keyNames)))
	for _, k := range keyNames {
		out.WriteString(k)
		out.WriteByte(0)
	}
	return out.Bytes()
}

func makeEntry(app appDef) []byte {
	vdf := makeAppVDF(app)

	var data bytes.Buffer
	writeU32(&data, 2)           // InfoState: normal
	writeU32(&data, 0)           // last updated
	writeU64(&data, 0)           // PICS token
	data.Write(make([]byte, 20)) // VDF SHA-1: unset in synthetic fixtures
	writeU32(&data, 0)           // change number
	sum := sha1.Sum(vdf)
	data.Write(sum[:]) // binary VDF SHA-1
	data.Write(vdf)

	var entry bytes.Buffer
	writeU32(&entry, app.id)
	writeU32(&entry, uint32(data.Len()))
	entry.Write(data.Bytes())
	return entry.Bytes()
}

func makeAppVDF(app appDef) []byte {
	var root bytes.Buffer
	root.WriteByte(0) // object
	if app.badKey {
		writeU32(&root, 9999)
	} else {
		root.Write(keyBytes("appinfo"))
	}
	for _, f := range appFields(app) {
		writeField(&root, f)
	}
	root.WriteByte(8) // end root object
	root.WriteByte(8) // end document
	return root.Bytes()
}

func appFields(app appDef) object {
	common := object{
		{key: "name", val: app.name},
		{key: "type", val: app.typ},
		{key: "oslist", val: app.oslist},
		{key: "installdir", val: app.installDir},
	}
	cfg := object{{key: "installdir", val: app.installDir}}
	if len(app.launches) > 0 {
		launch := object{}
		for _, l := range app.launches {
			launch = append(launch, field{key: l.key, val: launchFields(l)})
		}
		cfg = append(cfg, field{key: "launch", val: launch})
	}
	return object{
		{key: "common", val: common},
		{key: "config", val: cfg},
	}
}

func launchFields(l launchDef) object {
	fields := object{
		{key: "executable", val: l.executable},
		{key: "arguments", val: l.arguments},
		{key: "workingdir", val: l.workdir},
		{key: "type", val: l.typ},
		{key: "description", val: l.desc},
	}
	cfg := object{
		{key: "oslist", val: l.oslist},
		{key: "osarch", val: l.osarch},
	}
	return append(fields, field{key: "config", val: cfg})
}

func writeField(buf *bytes.Buffer, f field) {
	switch v := f.val.(type) {
	case string:
		buf.WriteByte(1) // string
		buf.Write(keyBytes(f.key))
		buf.WriteString(v)
		buf.WriteByte(0)
	case int32:
		buf.WriteByte(2) // int32
		buf.Write(keyBytes(f.key))
		writeU32(buf, uint32(v))
	case object:
		buf.WriteByte(0) // object
		buf.Write(keyBytes(f.key))
		for _, child := range v {
			writeField(buf, child)
		}
		buf.WriteByte(8)
	default:
		panic(fmt.Sprintf("unsupported field type %T for %q", v, f.key))
	}
}

func keyBytes(name string) []byte {
	idx, ok := keyIndex[name]
	if !ok {
		panic("unknown key " + name)
	}
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], idx)
	return b[:]
}

func writeU32(buf *bytes.Buffer, v uint32) {
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], v)
	buf.Write(b[:])
}

func writeU64(buf *bytes.Buffer, v uint64) {
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], v)
	buf.Write(b[:])
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "gen_appinfo:", err)
	os.Exit(1)
}
