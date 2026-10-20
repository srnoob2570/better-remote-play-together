package bridge

import (
	"context"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"
)

var contractDTOs = map[string]any{
	"Snapshot":          Snapshot{},
	"Config":            Config{},
	"Catalog":           Catalog{},
	"Game":              Game{},
	"Warning":           Warning{},
	"Problem":           Problem{},
	"StartSessionInput": StartSessionInput{},
	"Active":            Active{},
	"SessionState":      SessionState{},
	"Recovery":          Recovery{},
	"Stale":             Stale{},
	"Situation":         Situation{},
	"LastSession":       LastSession{},
	"Setup":             Setup{},
	"SetupSnapshot":     SetupSnapshot{},
}

func TestDTOsAreSerializable(t *testing.T) {
	errorType := reflect.TypeOf((*error)(nil)).Elem()
	contextType := reflect.TypeOf((*context.Context)(nil)).Elem()
	timeType := reflect.TypeOf(time.Time{})
	jsonName := regexp.MustCompile(`^[a-z][A-Za-z0-9]*$`)

	for name, value := range contractDTOs {
		t.Run(name, func(t *testing.T) {
			typ := reflect.TypeOf(value)
			if typ.Kind() != reflect.Struct {
				t.Fatalf("%s is %s, want struct", name, typ.Kind())
			}
			for index := 0; index < typ.NumField(); index++ {
				field := typ.Field(index)
				tag := field.Tag.Get("json")
				if tag == "" || tag == "-" {
					t.Fatalf("%s.%s has no usable json tag", name, field.Name)
				}
				jsonNamePart := strings.Split(tag, ",")[0]
				if !jsonName.MatchString(jsonNamePart) {
					t.Fatalf("%s.%s json name %q is not camelCase", name, field.Name, jsonNamePart)
				}
				if field.Type == errorType || field.Type == contextType || field.Type == timeType {
					t.Fatalf("%s.%s uses forbidden type %s", name, field.Name, field.Type)
				}
				switch field.Type.Kind() {
				case reflect.Func, reflect.Chan, reflect.UnsafePointer:
					t.Fatalf("%s.%s uses non-serializable type %s", name, field.Name, field.Type)
				}
			}
		})
	}
}

func TestTypeScriptContractMatchesDTOs(t *testing.T) {
	path := "../../frontend/src/bridge/types.ts"
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	source := string(data)
	fieldPattern := regexp.MustCompile(`(?m)^\s*([A-Za-z][A-Za-z0-9]*)\??\s*:`)

	for name, value := range contractDTOs {
		t.Run(name, func(t *testing.T) {
			body, ok := interfaceBody(source, name)
			if !ok {
				t.Fatalf("TypeScript interface %s not found in %s", name, path)
			}
			tsFields := make([]string, 0)
			for _, match := range fieldPattern.FindAllStringSubmatch(body, -1) {
				tsFields = append(tsFields, match[1])
			}
			goFields := jsonFieldNames(value)
			sort.Strings(tsFields)
			sort.Strings(goFields)
			if !reflect.DeepEqual(tsFields, goFields) {
				t.Fatalf("%s fields differ:\n  Go: %v\n  TS: %v", name, goFields, tsFields)
			}
		})
	}
}

func interfaceBody(source, name string) (string, bool) {
	pattern := regexp.MustCompile(`(?s)export\s+interface\s+` + regexp.QuoteMeta(name) + `\s*\{(.*?)\n\}`)
	match := pattern.FindStringSubmatch(source)
	if match == nil {
		return "", false
	}
	return match[1], true
}

func jsonFieldNames(value any) []string {
	typ := reflect.TypeOf(value)
	names := make([]string, 0, typ.NumField())
	for index := 0; index < typ.NumField(); index++ {
		tag := typ.Field(index).Tag.Get("json")
		names = append(names, strings.Split(tag, ",")[0])
	}
	return names
}
