package i18n

import "testing"

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
