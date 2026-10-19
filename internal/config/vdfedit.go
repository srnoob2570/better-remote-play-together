package config

import (
	"bytes"
	"fmt"
	"strings"
)

type vdfTokenKind uint8

const (
	vdfTokenEOF vdfTokenKind = iota
	vdfTokenString
	vdfTokenOpen
	vdfTokenClose
	vdfTokenCondition
)

type vdfToken struct {
	kind       vdfTokenKind
	start, end int
	raw, value string
}

type vdfLexer struct {
	data []byte
	pos  int
}

type vdfNode struct {
	Key         string
	KeyRaw      string
	KeyStart    int
	KeyEnd      int
	Object      *vdfObject
	Raw         string
	Value       string
	ValueStart  int
	ValueEnd    int
	End         int
	Conditional bool
}

type vdfObject struct {
	Open     int
	Close    int
	Children []*vdfNode
}

type vdfDocument struct {
	Data []byte
	Root *vdfObject
}

// LaunchOption is the current value of SpaceWar's LaunchOptions token.
type LaunchOption struct {
	Exists bool
	Raw    string
	Value  string
}

// EditResult describes a surgical launch-option edit.
type EditResult struct {
	Data                 []byte
	OriginalRaw          string
	OriginalValue        string
	NewValue             string
	CreatedAppBlock      bool
	CreatedLaunchOptions bool
}

// RestoreSpec is the durable information needed to undo one edit.
type RestoreSpec struct {
	NewValue             string
	OriginalRaw          string
	OriginalValue        string
	CreatedAppBlock      bool
	CreatedLaunchOptions bool
}

var launchOptionPath = []string{
	"UserLocalConfigStore", "Software", "Valve", "Steam", "apps",
}

const launchOptionKey = "LaunchOptions"
const spaceWarKey = "480"

func parseVDFDocument(data []byte) (*vdfDocument, error) {
	lexer := &vdfLexer{data: data}
	if bytes.HasPrefix(data, []byte{0xEF, 0xBB, 0xBF}) {
		lexer.pos = 3
	}
	root := &vdfObject{Open: 0, Close: len(data)}
	for {
		lexer.skipSpace()
		if lexer.pos >= len(data) {
			break
		}
		if data[lexer.pos] == '}' {
			return nil, fmt.Errorf("%w: unexpected closing brace at byte %d", ErrLocalConfigUnsafe, lexer.pos)
		}
		node, err := parseVDFPair(lexer)
		if err != nil {
			return nil, err
		}
		root.Children = append(root.Children, node)
	}
	return &vdfDocument{Data: data, Root: root}, nil
}

func parseVDFPair(lexer *vdfLexer) (*vdfNode, error) {
	key, err := lexer.next()
	if err != nil {
		return nil, err
	}
	if key.kind != vdfTokenString {
		return nil, fmt.Errorf("%w: expected key at byte %d", ErrLocalConfigUnsafe, key.start)
	}
	lexer.skipSpace()
	value, err := lexer.next()
	if err != nil {
		return nil, err
	}
	node := &vdfNode{
		Key:      key.value,
		KeyRaw:   key.raw,
		KeyStart: key.start,
		KeyEnd:   key.end,
	}
	switch value.kind {
	case vdfTokenOpen:
		object, err := parseVDFObject(lexer, value.start)
		if err != nil {
			return nil, err
		}
		node.Object = object
		node.ValueStart = value.start
		node.ValueEnd = object.Close + 1
		node.End = node.ValueEnd
	case vdfTokenString:
		node.Raw = value.raw
		node.Value = value.value
		node.ValueStart = value.start
		node.ValueEnd = value.end
		node.End = value.end
	default:
		return nil, fmt.Errorf("%w: key %q has no scalar or object value", ErrLocalConfigUnsafe, node.Key)
	}
	lexer.skipSpace()
	if lexer.pos < len(lexer.data) && lexer.data[lexer.pos] == '[' {
		condition, err := lexer.next()
		if err != nil {
			return nil, err
		}
		if condition.kind != vdfTokenCondition {
			return nil, fmt.Errorf("%w: malformed conditional for %q", ErrLocalConfigUnsafe, node.Key)
		}
		node.Conditional = true
		node.End = condition.end
	}
	return node, nil
}

func parseVDFObject(lexer *vdfLexer, open int) (*vdfObject, error) {
	object := &vdfObject{Open: open}
	for {
		lexer.skipSpace()
		if lexer.pos >= len(lexer.data) {
			return nil, fmt.Errorf("%w: unterminated object at byte %d", ErrLocalConfigUnsafe, open)
		}
		if lexer.data[lexer.pos] == '}' {
			object.Close = lexer.pos
			lexer.pos++
			return object, nil
		}
		node, err := parseVDFPair(lexer)
		if err != nil {
			return nil, err
		}
		object.Children = append(object.Children, node)
	}
}

func (l *vdfLexer) skipSpace() {
	for {
		for l.pos < len(l.data) && isVDFSpace(l.data[l.pos]) {
			l.pos++
		}
		if l.pos+1 >= len(l.data) || l.data[l.pos] != '/' {
			return
		}
		switch l.data[l.pos+1] {
		case '/':
			l.pos += 2
			for l.pos < len(l.data) && l.data[l.pos] != '\n' {
				l.pos++
			}
		case '*':
			l.pos += 2
			for l.pos+1 < len(l.data) && !(l.data[l.pos] == '*' && l.data[l.pos+1] == '/') {
				l.pos++
			}
			if l.pos+1 >= len(l.data) {
				return
			}
			l.pos += 2
		default:
			return
		}
	}
}

func (l *vdfLexer) next() (vdfToken, error) {
	l.skipSpace()
	if l.pos >= len(l.data) {
		return vdfToken{kind: vdfTokenEOF, start: len(l.data), end: len(l.data)}, nil
	}
	start := l.pos
	switch l.data[l.pos] {
	case '{':
		l.pos++
		return vdfToken{kind: vdfTokenOpen, start: start, end: l.pos, raw: "{"}, nil
	case '}':
		l.pos++
		return vdfToken{kind: vdfTokenClose, start: start, end: l.pos, raw: "}"}, nil
	case '[':
		l.pos++
		for l.pos < len(l.data) && l.data[l.pos] != ']' {
			l.pos++
		}
		if l.pos >= len(l.data) {
			return vdfToken{}, fmt.Errorf("%w: unterminated condition", ErrLocalConfigUnsafe)
		}
		l.pos++
		return vdfToken{kind: vdfTokenCondition, start: start, end: l.pos, raw: string(l.data[start:l.pos])}, nil
	case ']':
		return vdfToken{}, fmt.Errorf("%w: unexpected ] at byte %d", ErrLocalConfigUnsafe, start)
	case '"':
		l.pos++
		var decoded strings.Builder
		for l.pos < len(l.data) {
			c := l.data[l.pos]
			if c == '\\' {
				if l.pos+1 >= len(l.data) {
					return vdfToken{}, fmt.Errorf("%w: trailing escape at byte %d", ErrLocalConfigUnsafe, l.pos)
				}
				next := l.data[l.pos+1]
				switch next {
				case 'n':
					decoded.WriteByte('\n')
				case 't':
					decoded.WriteByte('\t')
				case 'r':
					decoded.WriteByte('\r')
				default:
					decoded.WriteByte(next)
				}
				l.pos += 2
				continue
			}
			if c == '"' {
				l.pos++
				return vdfToken{
					kind:  vdfTokenString,
					start: start,
					end:   l.pos,
					raw:   string(l.data[start:l.pos]),
					value: decoded.String(),
				}, nil
			}
			decoded.WriteByte(c)
			l.pos++
		}
		return vdfToken{}, fmt.Errorf("%w: unterminated quoted string at byte %d", ErrLocalConfigUnsafe, start)
	default:
		for l.pos < len(l.data) && !isVDFSpace(l.data[l.pos]) && !strings.ContainsRune("{}[]\"", rune(l.data[l.pos])) {
			l.pos++
		}
		if l.pos == start {
			return vdfToken{}, fmt.Errorf("%w: invalid token at byte %d", ErrLocalConfigUnsafe, start)
		}
		raw := string(l.data[start:l.pos])
		return vdfToken{kind: vdfTokenString, start: start, end: l.pos, raw: raw, value: raw}, nil
	}
}

func isVDFSpace(c byte) bool {
	switch c {
	case ' ', '\t', '\r', '\n':
		return true
	default:
		return false
	}
}

func uniqueChild(object *vdfObject, key string) (*vdfNode, error) {
	var found *vdfNode
	count := 0
	for _, child := range object.Children {
		if child.Key != key {
			continue
		}
		count++
		found = child
		if child.Conditional {
			return nil, fmt.Errorf("%w: conditional key %q", ErrLocalConfigUnsafe, key)
		}
	}
	if count > 1 {
		return nil, fmt.Errorf("%w: duplicate key %q", ErrLocalConfigUnsafe, key)
	}
	return found, nil
}

func findObject(document *vdfDocument, path []string) (*vdfObject, error) {
	current := document.Root
	for _, key := range path {
		child, err := uniqueChild(current, key)
		if err != nil {
			return nil, err
		}
		if child == nil {
			return nil, nil
		}
		if child.Object == nil {
			return nil, fmt.Errorf("%w: %q must be an object", ErrLocalConfigUnsafe, key)
		}
		current = child.Object
	}
	return current, nil
}

func findLaunchOption(document *vdfDocument) (*vdfNode, *vdfNode, *vdfObject, error) {
	apps, err := findObject(document, launchOptionPath)
	if err != nil {
		return nil, nil, nil, err
	}
	if apps == nil {
		return nil, nil, nil, fmt.Errorf("%w: apps object is missing", ErrLocalConfigUnsafe)
	}
	app, err := uniqueChild(apps, spaceWarKey)
	if err != nil {
		return nil, nil, nil, err
	}
	if app == nil {
		return nil, nil, apps, nil
	}
	if app.Object == nil {
		return nil, nil, nil, fmt.Errorf("%w: app 480 is not an object", ErrLocalConfigUnsafe)
	}
	launch, err := uniqueChild(app.Object, launchOptionKey)
	if err != nil {
		return nil, nil, nil, err
	}
	return launch, app, apps, nil
}

// ReadLaunchOption reads SpaceWar's LaunchOptions without modifying data.
func ReadLaunchOption(data []byte) (LaunchOption, error) {
	document, err := parseVDFDocument(data)
	if err != nil {
		return LaunchOption{}, err
	}
	launch, app, apps, err := findLaunchOption(document)
	if err != nil {
		return LaunchOption{}, err
	}
	if apps == nil {
		return LaunchOption{}, fmt.Errorf("%w: the correct apps object is missing", ErrLocalConfigUnsafe)
	}
	if app == nil || launch == nil {
		return LaunchOption{}, nil
	}
	if launch.Object != nil {
		return LaunchOption{}, fmt.Errorf("%w: LaunchOptions is an object", ErrLocalConfigUnsafe)
	}
	return LaunchOption{Exists: true, Raw: launch.Raw, Value: launch.Value}, nil
}

// EditLaunchOption replaces or inserts SpaceWar's LaunchOptions, touching only
// that token (or the minimal missing parents). It returns the original raw token
// so a later restore can put it back exactly.
func EditLaunchOption(data []byte, newValue string) (EditResult, error) {
	token, err := encodeVDFValue(newValue)
	if err != nil {
		return EditResult{}, err
	}
	document, err := parseVDFDocument(data)
	if err != nil {
		return EditResult{}, err
	}
	launch, app, apps, err := findLaunchOption(document)
	if err != nil {
		return EditResult{}, err
	}
	if apps == nil {
		return EditResult{}, fmt.Errorf("%w: the correct apps object is missing", ErrLocalConfigUnsafe)
	}

	result := EditResult{NewValue: newValue}
	var updated []byte
	switch {
	case app == nil:
		updated, err = insertAppBlock(data, apps, token)
		result.CreatedAppBlock = true
		result.CreatedLaunchOptions = true
	case app.Object == nil:
		return EditResult{}, fmt.Errorf("%w: app 480 is not an object", ErrLocalConfigUnsafe)
	case launch == nil:
		updated, err = insertLaunchOption(data, app.Object, token)
		result.CreatedLaunchOptions = true
	case launch.Object != nil:
		return EditResult{}, fmt.Errorf("%w: LaunchOptions is an object", ErrLocalConfigUnsafe)
	default:
		result.OriginalRaw = launch.Raw
		result.OriginalValue = launch.Value
		updated = replaceSpan(data, launch.ValueStart, launch.ValueEnd, token)
	}
	if err != nil {
		return EditResult{}, err
	}
	if _, err := parseVDFDocument(updated); err != nil {
		return EditResult{}, fmt.Errorf("edited VDF is invalid: %w", err)
	}
	check, err := ReadLaunchOption(updated)
	if err != nil {
		return EditResult{}, err
	}
	if !check.Exists || check.Value != newValue {
		return EditResult{}, fmt.Errorf("%w: launch option was not written", ErrLocalConfigVerify)
	}
	result.Data = updated
	return result, nil
}

// RestoreLaunchOption undoes one edit only if the current decoded value is
// exactly the marker Better RPT wrote. It never overwrites a changed value.
func RestoreLaunchOption(data []byte, spec RestoreSpec) ([]byte, error) {
	document, err := parseVDFDocument(data)
	if err != nil {
		return nil, err
	}
	launch, app, _, err := findLaunchOption(document)
	if err != nil {
		return nil, err
	}
	if launch == nil || launch.Value != spec.NewValue {
		return nil, fmt.Errorf("%w: current launch option is not ours", ErrLocalConfigChanged)
	}

	var updated []byte
	switch {
	case spec.OriginalRaw != "":
		updated = replaceSpan(data, launch.ValueStart, launch.ValueEnd, []byte(spec.OriginalRaw))
	case spec.CreatedAppBlock:
		if app == nil || app.Object == nil || len(app.Object.Children) != 1 {
			return nil, fmt.Errorf("%w: app 480 has user changes", ErrLocalConfigChanged)
		}
		only := app.Object.Children[0]
		if only.Key != launchOptionKey || only.Value != spec.NewValue {
			return nil, fmt.Errorf("%w: app 480 has user changes", ErrLocalConfigChanged)
		}
		updated = removeNode(data, app)
	case spec.CreatedLaunchOptions:
		updated = removeNode(data, launch)
	default:
		empty, err := encodeVDFValue("")
		if err != nil {
			return nil, err
		}
		updated = replaceSpan(data, launch.ValueStart, launch.ValueEnd, empty)
	}

	if _, err := parseVDFDocument(updated); err != nil {
		return nil, fmt.Errorf("restored VDF is invalid: %w", err)
	}
	if spec.OriginalRaw != "" {
		check, err := ReadLaunchOption(updated)
		if err != nil {
			return nil, err
		}
		if !check.Exists || check.Value != spec.OriginalValue {
			return nil, fmt.Errorf("%w: original launch option was not restored", ErrLocalConfigVerify)
		}
	}
	return updated, nil
}

func insertAppBlock(data []byte, apps *vdfObject, token []byte) ([]byte, error) {
	indent, ok := objectChildIndent(data, apps)
	if !ok {
		return nil, fmt.Errorf("%w: cannot determine apps indentation", ErrLocalConfigUnsafe)
	}
	unit, ok := objectIndentUnit(data, apps)
	if !ok {
		return nil, fmt.Errorf("%w: cannot determine apps indentation", ErrLocalConfigUnsafe)
	}
	block := fmt.Sprintf("%s\"%s\"\n%s{\n%s%s\"%s\"\t\t%s\n%s}\n",
		indent, spaceWarKey,
		indent,
		indent, unit, launchOptionKey, string(token),
		indent,
	)
	return insertAtObjectClose(data, apps, block), nil
}

func insertLaunchOption(data []byte, app *vdfObject, token []byte) ([]byte, error) {
	indent, ok := objectChildIndent(data, app)
	if !ok {
		return nil, fmt.Errorf("%w: cannot determine 480 indentation", ErrLocalConfigUnsafe)
	}
	line := fmt.Sprintf("%s\"%s\"\t\t%s\n", indent, launchOptionKey, string(token))
	return insertAtObjectClose(data, app, line), nil
}

func insertAtObjectClose(data []byte, object *vdfObject, content string) []byte {
	lineStart := startOfLine(data, object.Close)
	if len(bytes.TrimSpace(data[lineStart:object.Close])) == 0 {
		return replaceSpan(data, lineStart, lineStart, []byte(content))
	}
	return replaceSpan(data, object.Close, object.Close, []byte("\n"+content))
}

func objectChildIndent(data []byte, object *vdfObject) (string, bool) {
	closeStart := startOfLine(data, object.Close)
	if len(bytes.TrimSpace(data[closeStart:object.Close])) == 0 {
		return string(data[closeStart:object.Close]) + "\t", true
	}
	return "\t", true
}

func objectIndentUnit(data []byte, object *vdfObject) (string, bool) {
	closeStart := startOfLine(data, object.Close)
	closeIndent := string(data[closeStart:object.Close])
	if len(object.Children) == 0 {
		return "\t", true
	}
	first := object.Children[0]
	childStart := startOfLine(data, first.KeyStart)
	if len(bytes.TrimSpace(data[childStart:first.KeyStart])) != 0 {
		return "\t", true
	}
	childIndent := string(data[childStart:first.KeyStart])
	if strings.HasPrefix(childIndent, closeIndent) && len(childIndent) > len(closeIndent) {
		return childIndent[len(closeIndent):], true
	}
	return "\t", true
}

func replaceSpan(data []byte, start, end int, replacement []byte) []byte {
	result := make([]byte, 0, len(data)-(end-start)+len(replacement))
	result = append(result, data[:start]...)
	result = append(result, replacement...)
	result = append(result, data[end:]...)
	return result
}

func removeNode(data []byte, node *vdfNode) []byte {
	start := startOfLine(data, node.KeyStart)
	if len(bytes.TrimSpace(data[start:node.KeyStart])) != 0 {
		start = node.KeyStart
	}
	end := node.End
	if end < len(data) && data[end] == '\r' {
		end++
	}
	if end < len(data) && data[end] == '\n' {
		end++
	}
	return replaceSpan(data, start, end, nil)
}

func startOfLine(data []byte, pos int) int {
	if pos > len(data) {
		pos = len(data)
	}
	if index := bytes.LastIndexByte(data[:pos], '\n'); index >= 0 {
		return index + 1
	}
	return 0
}

func encodeVDFValue(value string) ([]byte, error) {
	for i := 0; i < len(value); i++ {
		if value[i] < 0x20 || value[i] == 0x7f {
			return nil, fmt.Errorf("%w: launch option contains control characters", ErrLocalConfigUnsafe)
		}
	}
	var encoded bytes.Buffer
	encoded.WriteByte('"')
	for i := 0; i < len(value); i++ {
		switch value[i] {
		case '\\', '"':
			encoded.WriteByte('\\')
		}
		encoded.WriteByte(value[i])
	}
	encoded.WriteByte('"')
	return encoded.Bytes(), nil
}
