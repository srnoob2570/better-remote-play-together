package i18n

import (
	"strings"
	"testing"
)

func TestCatalogsHaveSameKeys(t *testing.T) {
	for key := range catalogs[Spanish] {
		if _, ok := catalogs[English][key]; !ok {
			t.Errorf("English catalogue is missing %q", key)
		}
	}
	for key := range catalogs[English] {
		if _, ok := catalogs[Spanish][key]; !ok {
			t.Errorf("Spanish catalogue is missing %q", key)
		}
	}
}

func TestCatalogsAreEmbeddedAndNonEmpty(t *testing.T) {
	for _, lang := range []Language{Spanish, English} {
		catalog := catalogs[lang]
		if len(catalog) == 0 {
			t.Fatalf("%s catalogue is empty", lang)
		}
		for key, value := range catalog {
			if key == "" {
				t.Errorf("%s catalogue has an empty key", lang)
			}
			if value == "" {
				t.Errorf("%s catalogue has an empty value for %q", lang, key)
			}
		}
	}
}

func TestNormalizeAndSupported(t *testing.T) {
	if got := Normalize("en_US"); got != Spanish {
		t.Fatalf("Normalize(en_US) = %q, want fallback es", got)
	}
	if got := Normalize("EN"); got != English {
		t.Fatalf("Normalize(EN) = %q", got)
	}
	if !Supported("es") || !Supported("EN") {
		t.Fatal("supported language rejected")
	}
	if Supported("fr") || Supported("") {
		t.Fatal("unsupported language accepted")
	}
}

func TestDetect(t *testing.T) {
	t.Setenv("LC_ALL", "en_US.UTF-8")
	t.Setenv("LC_MESSAGES", "")
	t.Setenv("LANG", "es_ES.UTF-8")
	if got := Detect(); got != English {
		t.Fatalf("Detect() = %q, want en", got)
	}

	t.Setenv("LC_ALL", "")
	t.Setenv("LC_MESSAGES", "es_MX.UTF-8")
	if got := Detect(); got != Spanish {
		t.Fatalf("Detect() = %q, want es", got)
	}

	t.Setenv("LC_MESSAGES", "")
	t.Setenv("LANG", "C")
	if got := Detect(); got != Spanish {
		t.Fatalf("Detect() fallback = %q, want es", got)
	}
}

func TestTranslator(t *testing.T) {
	if got := New("en").T("app.title"); got != "Better RPT" {
		t.Fatalf("title = %q", got)
	}
	if got := New("fr").T("app.language"); got != "Idioma" {
		t.Fatalf("fallback = %q", got)
	}
	if got := New("en").T("does.not.exist"); got != "does.not.exist" {
		t.Fatalf("missing key = %q", got)
	}
}

func TestTranslatorFormatsArguments(t *testing.T) {
	tests := []struct {
		name string
		lang string
		key  string
		args []any
		want string
	}{
		{name: "Spanish count", lang: "es", key: "library.count", args: []any{2}, want: "2 juegos"},
		{name: "English count", lang: "en", key: "library.count", args: []any{2}, want: "2 games"},
		{name: "Spanish step", lang: "es", key: "prep.step", args: []any{2, 5}, want: "Paso 2 de 5"},
		{name: "English step", lang: "en", key: "prep.step", args: []any{2, 5}, want: "Step 2 of 5"},
		{name: "Spanish current step", lang: "es", key: "prep.current", args: []any{4, 5, "Pedir a Steam que abra SpaceWar"}, want: "Paso 4 de 5 · Pedir a Steam que abra SpaceWar"},
		{name: "English current step", lang: "en", key: "prep.current", args: []any{4, 5, "Ask Steam to open SpaceWar"}, want: "Step 4 of 5 · Ask Steam to open SpaceWar"},
		{name: "pending game", lang: "en", key: "pending.body", args: []any{"Hades"}, want: "SpaceWar still has a configuration from an earlier session for Hades. It is not the last session; you can restore it or open the guide."},
		{name: "unsupported falls back", lang: "fr", key: "library.count", args: []any{3}, want: "3 juegos"},
		{name: "missing key is not formatted", lang: "en", key: "does.not.exist", args: []any{1}, want: "does.not.exist"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := New(test.lang).T(test.key, test.args...); got != test.want {
				t.Fatalf("T(%q, %v) = %q, want %q", test.key, test.args, got, test.want)
			}
		})
	}
}

func TestInterpolatedCatalogsKeepPlaceholders(t *testing.T) {
	tests := []struct {
		key     string
		spanish string
		english string
	}{
		{key: "library.count", spanish: "%d juegos", english: "%d games"},
		{key: "prep.step", spanish: "Paso %d de %d", english: "Step %d of %d"},
		{key: "prep.current", spanish: "Paso %d de %d · %s", english: "Step %d of %d · %s"},
	}
	for _, test := range tests {
		if got := catalogs[Spanish][test.key]; got != test.spanish {
			t.Errorf("Spanish %q = %q, want %q", test.key, got, test.spanish)
		}
		if got := catalogs[English][test.key]; got != test.english {
			t.Errorf("English %q = %q, want %q", test.key, got, test.english)
		}
	}
	if !strings.Contains(catalogs[Spanish]["pending.body"], "%s") {
		t.Error("Spanish pending.body is missing its placeholder")
	}
	if !strings.Contains(catalogs[English]["pending.body"], "%s") {
		t.Error("English pending.body is missing its placeholder")
	}
}
