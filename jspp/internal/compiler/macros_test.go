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
