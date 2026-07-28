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
