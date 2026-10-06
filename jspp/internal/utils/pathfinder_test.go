package utils_test

import (
	"testing"

	"github.com/epicoon/lxgo/jspp"
	"github.com/epicoon/lxgo/jspp/component"
	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/apptest"
)

func newTestPreprocessor(t *testing.T) jspp.IPreprocessor {
	t.Helper()
	return newTestPreprocessorWithParams(t, nil)
}

func newTestPreprocessorWithParams(t *testing.T, params kernel.Dict) jspp.IPreprocessor {
	t.Helper()
	sysPath := t.TempDir()
	cfg := kernel.Dict{
		"Components": kernel.Dict{
			"JSPreprocessor": kernel.Dict{
				"SysPath":  sysPath,
				"MapsPath": sysPath,
			},
		},
	}
	if params != nil {
		cfg["Params"] = params
	}
	app, err := apptest.New(cfg)
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

// TestPathfinder_GetAbsPath_Empty is a regression test: GetAbsPath("")
// used to panic on path[0] with no length check first.
func TestPathfinder_GetAbsPath_Empty(t *testing.T) {
	pp := newTestPreprocessor(t)

	if got := pp.Pathfinder().GetAbsPath(""); got != "" {
		t.Fatalf("expected GetAbsPath(\"\") to return \"\", got %q", got)
	}
}

func TestPathfinder_GetAbsPath_Param_SubstitutesConfigValue(t *testing.T) {
	pp := newTestPreprocessorWithParams(t, kernel.Dict{"StaticSrc": "http://localhost:8092"})

	got := pp.Pathfinder().GetAbsPath("{@param(Params.StaticSrc)}/runtime/web/lib/three.js")
	want := "http://localhost:8092/runtime/web/lib/three.js"
	if got != want {
		t.Fatalf("GetAbsPath = %q, want %q", got, want)
	}
}

// Plain concatenation, not filepath.Join/path.Join - a URL config value
// joined that way would lose the "/" after its scheme.
func TestPathfinder_GetAbsPath_Param_DoesNotMangleURLScheme(t *testing.T) {
	pp := newTestPreprocessorWithParams(t, kernel.Dict{"StaticSrc": "https://cdn.example.com"})

	got := pp.Pathfinder().GetAbsPath("{@param(Params.StaticSrc)}/x.js")
	if got != "https://cdn.example.com/x.js" {
		t.Fatalf("GetAbsPath = %q, want the scheme's \"//\" intact", got)
	}
}

func TestPathfinder_GetAbsPath_Param_MissingConfigParam_ReturnsEmpty(t *testing.T) {
	pp := newTestPreprocessor(t)

	if got := pp.Pathfinder().GetAbsPath("{@param(Params.DoesNotExist)}/x.js"); got != "" {
		t.Fatalf("GetAbsPath = %q, want \"\" for a missing config param", got)
	}
}

func TestPathfinder_GetAbsPath_Param_NonStringConfigParam_ReturnsEmpty(t *testing.T) {
	pp := newTestPreprocessorWithParams(t, kernel.Dict{"Count": 42})

	if got := pp.Pathfinder().GetAbsPath("{@param(Params.Count)}/x.js"); got != "" {
		t.Fatalf("GetAbsPath = %q, want \"\" for a non-string config param", got)
	}
}
