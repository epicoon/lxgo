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

// TestI18nMap_Localize_ParenInsideKeyDoesNotEndCallEarly is a regression
// test: the lx.i18n(...) call's closing paren is located via
// FindMatchingBrace, which used to count ')' characters with no awareness
// of string literals - a key containing its own ')' would cut the call
// short.
func TestI18nMap_Localize_ParenInsideKeyDoesNotEndCallEarly(t *testing.T) {
	m := NewI18nMap(map[string]map[string]string{
		"en": {"greeting)odd": "Hello"},
	})
	got := m.Localize(`x = lx.i18n('greeting)odd');`, "en")
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
	if !strings.Contains(got, "let i18n_name=userName;") {
		t.Fatalf("expected the param bound in a let statement under its mangled name, got %q", got)
	}
	if !strings.Contains(got, "return `Hello ${i18n_name}`") {
		t.Fatalf("expected the template's placeholder rewritten to the same mangled name, got %q", got)
	}
}

// TestI18nMap_Localize_ParamNameCollidesWithItsOwnValue is a regression
// test: a bound param whose value expression is the very same identifier as
// its name (the everyday case - {points} shorthand, or the equivalent
// explicit {points: points}, both meaning "bind ${points} in the template
// to the surrounding points variable") used to generate "let points =
// points;" - the local declaration shadows the outer variable of the same
// name for the WHOLE let statement, including its own initializer, so the
// right-hand side "points" resolves to the not-yet-initialized local
// instead of the outer one: "ReferenceError: Cannot access 'points' before
// initialization" at runtime. Binding the local under a mangled name no
// source identifier could ever collide with avoids that entirely.
func TestI18nMap_Localize_ParamNameCollidesWithItsOwnValue(t *testing.T) {
	m := NewI18nMap(map[string]map[string]string{
		"en": {"sold": "Sold goods: +${points}"},
	})
	got := m.Localize("x = lx.i18n('sold', {points});", "en")
	if strings.Contains(got, "let points=points;") {
		t.Fatalf("expected the self-shadowing declaration NOT to appear, got %q", got)
	}
	if !strings.Contains(got, "let i18n_points=points;") {
		t.Fatalf("expected the value bound under a mangled name, got %q", got)
	}
	if !strings.Contains(got, "return `Sold goods: +${i18n_points}`") {
		t.Fatalf("expected the template's placeholder rewritten to the same mangled name, got %q", got)
	}
}

func TestI18nMap_Localize_MultipleOccurrences(t *testing.T) {
	m := NewI18nMap(map[string]map[string]string{"en": {"a": "A", "b": "B"}})
	got := m.Localize(`lx.i18n('a'); lx.i18n('b');`, "en")
	if got != `'A'; 'B';` {
		t.Fatalf("got %q", got)
	}
}

// TestI18nMap_Localize_MultipleCallsInArrayLiteral covers the shape that
// actually broke in production: several lx.i18n(...) calls back to back in
// one array, each already carrying a module-scoped key (as produced by
// addModuleI18nPrefix in the compiler package, which stores module
// dictionaries under "module-{{name}}-{{key}}" - see
// Compiler.applyModuleI18n). Before that prefixing bug was fixed at the
// source, the second call's key would end up merged into the first (see
// files_compiler_test.go) - this pins the correct, independent handling of
// each call once the input is well-formed.
func TestI18nMap_Localize_MultipleCallsInArrayLiteral(t *testing.T) {
	m := NewI18nMap(map[string]map[string]string{
		"en": {"module-Calendar-monday": "Mo", "module-Calendar-tuesday": "Tu", "module-Calendar-wednesday": "We"},
	})
	got := m.Localize(`[lx.i18n('module-Calendar-monday'), lx.i18n('module-Calendar-tuesday'), lx.i18n('module-Calendar-wednesday')]`, "en")
	if got != `['Mo', 'Tu', 'We']` {
		t.Fatalf("got %q", got)
	}
}

// TestI18nMap_Localize_UntranslatedKeyDoesNotStopLaterOnes is a regression
// test: the loop's own "find the next lx.i18n(...)" regex used to be
// reassigned, inside the untranslated-key fallback branch, to the
// module-prefix-stripping regex - a completely different pattern reused
// under the same variable name. Once any key in the text had no
// translation, that reassignment silently broke the loop's own search on
// its next iteration (the module-prefix pattern almost never matches
// mid-text), ending the whole pass early - every lx.i18n(...) call after
// the first untranslated one was left completely untouched, even ones with
// a perfectly good translation.
func TestI18nMap_Localize_UntranslatedKeyDoesNotStopLaterOnes(t *testing.T) {
	m := NewI18nMap(map[string]map[string]string{
		"en": {"root.newGameHint": "Start a new game"},
	})
	got := m.Localize(`a = lx.i18n(root.pointsTable); b = lx.i18n(root.newGameHint);`, "en")
	want := `a = 'root.pointsTable'; b = 'Start a new game';`
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
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

// TestExtractParams_JsShorthandProperty is a regression test: {points} is
// valid JS object shorthand for {points: points} - the param split used to
// require an explicit "name: value" pair and silently drop any item with no
// ':' in it, so a shorthand-only placeholders object translated to no
// params at all.
func TestExtractParams_JsShorthandProperty(t *testing.T) {
	key, params := extractParams("sellGoods.points, {points}")
	if key != "sellGoods.points" {
		t.Fatalf("got key=%q, want sellGoods.points", key)
	}
	want := map[string]string{"points": "points"}
	if !reflect.DeepEqual(params, want) {
		t.Fatalf("got params=%#v, want %#v", params, want)
	}
}

// TestExtractParams_MixedShorthandAndExplicit checks shorthand and explicit
// "name: value" properties combined in the same object.
func TestExtractParams_MixedShorthandAndExplicit(t *testing.T) {
	key, params := extractParams("greeting, {points, name: userName}")
	if key != "greeting" {
		t.Fatalf("got key=%q, want greeting", key)
	}
	want := map[string]string{"points": "points", "name": "userName"}
	if !reflect.DeepEqual(params, want) {
		t.Fatalf("got params=%#v, want %#v", params, want)
	}
}
