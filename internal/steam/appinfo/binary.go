package appinfo

import (
	"encoding/binary"
	"fmt"
	"math"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

const (
	typeObject   byte = 0
	typeString   byte = 1
	typeInt32    byte = 2
	typeFloat32  byte = 3
	typePointer  byte = 4
	typeWide     byte = 5
	typeColor    byte = 6
	typeUint64   byte = 7
	typeEnd      byte = 8
	typeBlob     byte = 9
	typeInt64    byte = 10
	typeAltEnd   byte = 11
	maxVDFDepth       = 64
	maxVDFFields      = 1 << 20
)

type binaryParser struct {
	data    []byte
	strings []string
	pos     int
	fields  int
}

func (p *binaryParser) parseDocument() (map[string]any, error) {
	typ, err := p.readByte()
	if err != nil {
		return nil, err
	}
	if typ != typeObject {
		return nil, fmt.Errorf("document root is type %d, want object", typ)
	}
	key, err := p.readKey()
	if err != nil {
		return nil, err
	}
	root, err := p.parseObject(1)
	if err != nil {
		return nil, err
	}
	if p.pos < len(p.data) {
		end, err := p.readByte()
		if err != nil {
			return nil, err
		}
		if end != typeEnd && end != typeAltEnd {
			return nil, fmt.Errorf("document terminator is type %d", end)
		}
	}
	for p.pos < len(p.data) {
		b, err := p.readByte()
		if err != nil {
			return nil, err
		}
		if b != 0 {
			return nil, fmt.Errorf("trailing bytes after document")
		}
	}
	return map[string]any{key: root}, nil
}

func (p *binaryParser) parseObject(depth int) (map[string]any, error) {
	if depth > maxVDFDepth {
		return nil, fmt.Errorf("VDF object nesting exceeds %d", maxVDFDepth)
	}
	obj := make(map[string]any)
	for {
		typ, err := p.readByte()
		if err != nil {
			return nil, err
		}
		if typ == typeEnd || typ == typeAltEnd {
			return obj, nil
		}
		key, err := p.readKey()
		if err != nil {
			return nil, err
		}
		value, err := p.readValue(typ, depth)
		if err != nil {
			return nil, fmt.Errorf("field %q: %w", key, err)
		}
		obj[key] = value
		p.fields++
		if p.fields > maxVDFFields {
			return nil, fmt.Errorf("VDF field count exceeds %d", maxVDFFields)
		}
	}
}

func (p *binaryParser) readValue(typ byte, depth int) (any, error) {
	switch typ {
	case typeObject:
		return p.parseObject(depth + 1)
	case typeString:
		return p.readString()
	case typeInt32:
		value, err := p.readU32()
		return int32(value), err
	case typeFloat32:
		value, err := p.readU32()
		return math.Float32frombits(value), err
	case typePointer:
		return p.readU32()
	case typeWide:
		return p.readWideString()
	case typeColor:
		return p.readU32()
	case typeUint64:
		return p.readU64()
	case typeBlob:
		length, err := p.readU32()
		if err != nil {
			return nil, err
		}
		if uint64(length) > uint64(len(p.data)-p.pos) {
			return nil, fmt.Errorf("blob length %d exceeds remaining data", length)
		}
		value := p.data[p.pos : p.pos+int(length)]
		p.pos += int(length)
		return append([]byte(nil), value...), nil
	case typeInt64:
		value, err := p.readU64()
		return int64(value), err
	case typeEnd, typeAltEnd:
		return nil, fmt.Errorf("unexpected end marker in value position")
	default:
		return nil, fmt.Errorf("unknown VDF type %d", typ)
	}
}

func (p *binaryParser) readKey() (string, error) {
	index, err := p.readU32()
	if err != nil {
		return "", err
	}
	if uint64(index) >= uint64(len(p.strings)) {
		return "", fmt.Errorf("key index %d outside string table (%d entries)", index, len(p.strings))
	}
	return p.strings[index], nil
}

func (p *binaryParser) readString() (string, error) {
	n := p.findNUL()
	if n < 0 {
		return "", fmt.Errorf("unterminated string")
	}
	value := string(p.data[p.pos : p.pos+n])
	if !utf8.ValidString(value) {
		return "", fmt.Errorf("string is not UTF-8")
	}
	p.pos += n + 1
	return value, nil
}

func (p *binaryParser) readWideString() (string, error) {
	length, err := p.readU32()
	if err != nil {
		return "", err
	}
	if length == 0 {
		return "", nil
	}
	if uint64(length) > uint64(len(p.data)-p.pos) {
		return "", fmt.Errorf("wide string length %d exceeds remaining data", length)
	}
	if length%2 != 0 {
		return "", fmt.Errorf("wide string length %d is not even", length)
	}
	raw := p.data[p.pos : p.pos+int(length)]
	p.pos += int(length)
	units := make([]uint16, 0, length/2)
	for i := 0; i < len(raw); i += 2 {
		units = append(units, binary.LittleEndian.Uint16(raw[i:i+2]))
	}
	value := string(utf16.Decode(units))
	return strings.TrimRight(value, "\x00"), nil
}

func (p *binaryParser) findNUL() int {
	for i := p.pos; i < len(p.data); i++ {
		if p.data[i] == 0 {
			return i - p.pos
		}
	}
	return -1
}

func (p *binaryParser) readByte() (byte, error) {
	if p.pos >= len(p.data) {
		return 0, fmt.Errorf("unexpected end of VDF data")
	}
	value := p.data[p.pos]
	p.pos++
	return value, nil
}

func (p *binaryParser) readU32() (uint32, error) {
	if len(p.data)-p.pos < 4 {
		return 0, fmt.Errorf("unexpected end of VDF data")
	}
	value := binary.LittleEndian.Uint32(p.data[p.pos : p.pos+4])
	p.pos += 4
	return value, nil
}

func (p *binaryParser) readU64() (uint64, error) {
	if len(p.data)-p.pos < 8 {
		return 0, fmt.Errorf("unexpected end of VDF data")
	}
	value := binary.LittleEndian.Uint64(p.data[p.pos : p.pos+8])
	p.pos += 8
	return value, nil
}
