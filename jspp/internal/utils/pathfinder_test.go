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

// TestPathfinder_GetAbsPath_Empty is a regression test: GetAbsPath("")
// used to panic on path[0] with no length check first.
func TestPathfinder_GetAbsPath_Empty(t *testing.T) {
	pp := newTestPreprocessor(t)

	if got := pp.Pathfinder().GetAbsPath(""); got != "" {
		t.Fatalf("expected GetAbsPath(\"\") to return \"\", got %q", got)
	}
}
