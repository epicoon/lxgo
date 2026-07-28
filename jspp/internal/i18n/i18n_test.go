package i18n

import (
	"reflect"
	"strings"
	"testing"
)

func TestI18nMap_IsEmpty(t *testing.T) {
	if !NewI18nMap(nil).IsEmpty() {
		t.Fatalf("expected a nil map to be empty")
	}
	if NewI18nMap(map[string]map[string]string{"en": {"a": "b"}}).IsEmpty() {
		t.Fatalf("expected a non-empty map to report non-empty")
	}
}

func TestI18nMap_Get(t *testing.T) {
	m := NewI18nMap(map[string]map[string]string{
		"en": {"greeting": "Hello"},
	})
	if got := m.Get("en", "greeting"); got != "Hello" {
		t.Fatalf("got %q, want Hello", got)
	}
	if got := m.Get("en", "missing"); got != "" {
		t.Fatalf("expected empty string for a missing key, got %q", got)
	}
	if got := m.Get("fr", "greeting"); got != "" {
		t.Fatalf("expected empty string for a missing language, got %q", got)
	}
}

func TestI18nMap_Localize_FoundTranslation(t *testing.T) {
	m := NewI18nMap(map[string]map[string]string{
		"en": {"greeting": "Hello"},
	})
	got := m.Localize(`x = lx.i18n('greeting');`, "en")
	if got != `x = 'Hello';` {
		t.Fatalf("got %q", got)
	}
}

func TestI18nMap_Localize_MissingTranslation_StripsModulePrefix(t *testing.T) {
	m := NewI18nMap(map[string]map[string]string{"en": {}})
	got := m.Localize(`x = lx.i18n('module-mymod-greeting');`, "en")
	if got != `x = 'greeting';` {
		t.Fatalf("got %q", got)
	}
}

func TestI18nMap_Localize_MissingLanguage(t *testing.T) {
	m := NewI18nMap(map[string]map[string]string{"en": {"greeting": "Hello"}})
	got := m.Localize(`x = lx.i18n('greeting');`, "fr")
	if got != `x = 'greeting';` {
		t.Fatalf("got %q, want the key itself since 'fr' has no translations", got)
	}
}

func TestI18nMap_Localize_WithParams(t *testing.T) {
	m := NewI18nMap(map[string]map[string]string{
		"en": {"greeting": "Hello ${name}"},
	})
	got := m.Localize("x = lx.i18n('greeting', {name: userName});", "en")
	if !strings.Contains(got, "let name=userName;") {
		t.Fatalf("expected the param bound in a let statement, got %q", got)
	}
	if !strings.Contains(got, "return `Hello ${name}`") {
		t.Fatalf("expected the translation returned as a template literal, got %q", got)
	}
}

func TestI18nMap_Localize_MultipleOccurrences(t *testing.T) {
	m := NewI18nMap(map[string]map[string]string{"en": {"a": "A", "b": "B"}})
	got := m.Localize(`lx.i18n('a'); lx.i18n('b');`, "en")
	if got != `'A'; 'B';` {
		t.Fatalf("got %q", got)
	}
}

func TestExtractParams_NoParams(t *testing.T) {
	key, params := extractParams("greeting")
	if key != "greeting" || len(params) != 0 {
		t.Fatalf("got key=%q params=%#v", key, params)
	}
}

func TestExtractParams_WithParams(t *testing.T) {
	key, params := extractParams("greeting, {name: userName, count: n}")
	if key != "greeting" {
		t.Fatalf("got key=%q, want greeting", key)
	}
	want := map[string]string{"name": "userName", "count": "n"}
	if !reflect.DeepEqual(params, want) {
		t.Fatalf("got params=%#v, want %#v", params, want)
	}
}
