package lxml_test

import (
	"testing"

	"github.com/epicoon/lxgo/jspp"
	"github.com/epicoon/lxgo/jspp/component"
	"github.com/epicoon/lxgo/jspp/internal/lxml"
	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/apptest"
)

func newTestPreprocessor(t *testing.T) jspp.IPreprocessor {
	t.Helper()
	sysPath := t.TempDir()
	app, err := apptest.New(kernel.Dict{
		"Components": kernel.Dict{
			"JSPreprocessor": kernel.Dict{
				"SysPath":  sysPath,
				"MapsPath": sysPath,
			},
		},
	})
	if err != nil {
		t.Fatalf("apptest.New: %v", err)
	}
	if err := component.SetAppComponent(app, "Components.JSPreprocessor"); err != nil {
		t.Fatalf("SetAppComponent: %v", err)
	}
	pp, err := component.AppComponent(app)
	if err != nil {
		t.Fatalf("AppComponent: %v", err)
	}
	return pp
}

// TestParseText_LinkToDefinedBlock_Compiles is the baseline: a link to a
// block that IS defined compiles cleanly, with the placeholder substituted
// for the real generated function name.
func TestParseText_LinkToDefinedBlock_Compiles(t *testing.T) {
	pp := newTestPreprocessor(t)
	src := "<*greet>\n" +
		"    <lx.Box> #text('hi')\n" +
		"<&greet>\n"

	code, err := lxml.NewParser(pp).ParseText(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code == "" {
		t.Fatalf("expected non-empty compiled code")
	}
}

// TestParseText_LinkToUndefinedBlock_Errors is a regression test: a link
// to a block with no matching definition used to compile "successfully"
// into invalid JS (a literal, unsubstituted "[|Name|]" placeholder left
// in the output) instead of raising a compile error.
func TestParseText_LinkToUndefinedBlock_Errors(t *testing.T) {
	pp := newTestPreprocessor(t)
	src := "<&neverDefined>\n"

	code, err := lxml.NewParser(pp).ParseText(src)
	if err == nil {
		t.Fatalf("expected an error for a link to an undefined block, got code: %q", code)
	}
	if code != "" {
		t.Fatalf("expected no code on error, got %q", code)
	}
}
