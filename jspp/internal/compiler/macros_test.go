package compiler

import (
	"strings"
	"testing"
)

func TestApplyMacros(t *testing.T) {
	c := &Compiler{}
	src := "@lx:macros GREET { console.log('hi'); };\n" +
		"function f() { lx>>>GREET }\n" +
		"function g() { lx>>>GREET }\n"

	got := c.applyMacros(src)

	if !strings.Contains(got, "console.log('hi');") {
		t.Fatalf("expected the macro body substituted at both use sites, got %q", got)
	}
	if strings.Contains(got, "@lx:macros") {
		t.Fatalf("expected the macro declaration itself to be stripped, got %q", got)
	}
	if strings.Contains(got, "lx>>>GREET") {
		t.Fatalf("expected every lx>>>GREET use site replaced, got %q", got)
	}
}

// TestApplyMacros_BraceInsideStringDoesNotEndMacroBodyEarly is a
// regression test: the macro body's closing brace is located via
// FindMatchingBrace, which used to count '}' characters with no awareness
// of string literals - a macro body containing a string with its own '}'
// (e.g. a template string embedding an object literal) would cut the body
// short.
func TestApplyMacros_BraceInsideStringDoesNotEndMacroBodyEarly(t *testing.T) {
	c := &Compiler{}
	src := "@lx:macros GREET { console.log('a } b'); };\n" +
		"function f() { lx>>>GREET }\n"

	got := c.applyMacros(src)

	if !strings.Contains(got, "console.log('a } b');") {
		t.Fatalf("expected the full macro body (including the '}' inside the string) substituted, got %q", got)
	}
	if strings.Contains(got, "@lx:macros") {
		t.Fatalf("expected the macro declaration itself to be stripped, got %q", got)
	}
}

func TestApplyMacros_NoDeclarationLeavesCodeUnchanged(t *testing.T) {
	c := &Compiler{}
	src := "plain code with no macros"
	got := c.applyMacros(src)
	if got != src {
		t.Fatalf("expected unchanged code, got %q", got)
	}
}

func TestApplyMacros_UnmatchedUseSiteIsLeftAlone(t *testing.T) {
	// A use site for a macro that was never declared shouldn't be touched
	// (and must not panic on a nil map lookup).
	c := &Compiler{}
	got := c.applyMacros("lx>>>NEVER_DECLARED")
	if got != "lx>>>NEVER_DECLARED" {
		t.Fatalf("expected the unmatched use site left as-is, got %q", got)
	}
}
