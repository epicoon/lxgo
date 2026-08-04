package lxml_test

import (
	"strings"
	"testing"

	"github.com/epicoon/lxgo/jspp/internal/lxml"
)

// TestParseText_RepeatedMethodCall is a regression test: calling the same
// method twice in a row (`#method(a) #method(b)`) used to compile both
// calls with the SAME (last-written) args, since WidgetNode.Methods was a
// plain map[string]string keyed by method name - the second call's args
// silently overwrote the first's before compilation ever ran. Methods now
// holds one args entry per call, consumed in order.
func TestParseText_RepeatedMethodCall(t *testing.T) {
	pp := newTestPreprocessor(t)
	src := "<*root>\n" +
		"    <lx.Box> #method(1) #method(2)\n" +
		"<&root>\n"

	code, err := lxml.NewParser(pp).ParseText(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	first := strings.Index(code, ".method(1)")
	second := strings.Index(code, ".method(2)")
	if first == -1 || second == -1 {
		t.Fatalf("expected both distinct calls .method(1) and .method(2) in the compiled code, got: %s", code)
	}
	if first >= second {
		t.Fatalf("expected .method(1) to come before .method(2) in the compiled code, got: %s", code)
	}
}

// TestParseText_MethodArgWithParenInsideStringIsNotCutShort is a regression
// test: #method(...)'s closing paren is located via FindMatchingBrace,
// which used to count ')' characters with no awareness of string literals
// - an argument containing its own ')' (a string value, in real markup)
// would cut the method call short.
func TestParseText_MethodArgWithParenInsideStringIsNotCutShort(t *testing.T) {
	pp := newTestPreprocessor(t)
	src := "<*root>\n" +
		"    <lx.Box> #method('a)b')\n" +
		"<&root>\n"

	code, err := lxml.NewParser(pp).ParseText(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(code, ".method('a)b')") {
		t.Fatalf("expected the full method argument (including the ')' inside the string) in compiled code, got: %s", code)
	}
}

// TestParseText_RepeatedMethodCall_MixedWithOtherMethod checks that
// interleaving a repeated method with a different one doesn't confuse the
// per-name call-index tracking.
func TestParseText_RepeatedMethodCall_MixedWithOtherMethod(t *testing.T) {
	pp := newTestPreprocessor(t)
	src := "<*root>\n" +
		"    <lx.Box> #method(1) #other(x) #method(2)\n" +
		"<&root>\n"

	code, err := lxml.NewParser(pp).ParseText(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, want := range []string{".method(1)", ".other(x)", ".method(2)"} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in compiled code, got: %s", want, code)
		}
	}
}

// TestParseText_InterpolationInTextAttribute checks the already-documented
// behavior (doc/lxml.md): ${expr} inside a quoted widget text attribute
// compiles to double-quoted-string concatenation.
func TestParseText_InterpolationInTextAttribute(t *testing.T) {
	pp := newTestPreprocessor(t)
	src := "<lx.Box> \"hello ${name}!\"\n"

	code, err := lxml.NewParser(pp).ParseText(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(code, `text:"hello "+name+"!"`) {
		t.Fatalf("expected interpolation compiled to string concatenation, got: %s", code)
	}
}

// TestParseText_InterpolationInRawHTML is a regression test: raw HTML nested
// under a widget compiles into a `html:` field wrapped in JS backticks
// (a real template literal), so ${expr} there is already valid JS and needs
// no rewriting - procInserts used to rewrite it into "+expr+" regardless of
// context, splicing double-quote concatenation syntax into the middle of a
// backtick string and corrupting it (e.g. `<div class="${cls}">` used to
// come out as `<div class=""+cls>`, breaking the markup).
func TestParseText_InterpolationInRawHTML(t *testing.T) {
	pp := newTestPreprocessor(t)
	src := "<lx.Box> @root\n" +
		"  <div class=\"${cls}\">insert: ${val}</div>\n"

	code, err := lxml.NewParser(pp).ParseText(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "html:`  <div class=\"${cls}\">insert: ${val}</div>`"
	if !strings.Contains(code, want) {
		t.Fatalf("expected raw HTML's ${...} left untouched inside the backtick literal, got: %s", code)
	}
}

// TestParseText_InterpolationInTextAttributeAndRawHTMLTogether covers both
// contexts appearing in the same compiled widget - each must be resolved
// independently by procInserts's quote-tracking (double-quoted text still
// concatenated, backtick-quoted HTML left alone).
func TestParseText_InterpolationInTextAttributeAndRawHTMLTogether(t *testing.T) {
	pp := newTestPreprocessor(t)
	src := "<lx.Box> \"hi ${name}\"\n" +
		"  <div>insert: ${val}</div>\n"

	code, err := lxml.NewParser(pp).ParseText(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(code, `text:"hi "+name,`) {
		t.Fatalf("expected the text attribute's ${...} concatenated, got: %s", code)
	}
	if !strings.Contains(code, "html:`  <div>insert: ${val}</div>`") {
		t.Fatalf("expected the raw HTML's ${...} left untouched, got: %s", code)
	}
}
