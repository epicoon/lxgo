package app_test

import (
	"testing"

	"github.com/epicoon/lxgo/kernel/app"
	"github.com/epicoon/lxgo/kernel/apptest"
)

// TestAppPathfinder_GetAbsPath_Empty is a regression test: GetAbsPath("")
// used to panic on path[0] (no length check before the "@alias" prefix
// check) - found while testing lxgo-jspp's executor, which passes an
// unset (empty) config path straight through to GetAbsPath.
func TestAppPathfinder_GetAbsPath_Empty(t *testing.T) {
	a, err := apptest.New()
	if err != nil {
		t.Fatalf("apptest.New: %v", err)
	}
	pf := app.NewAppPathfinder(a)

	got := pf.GetAbsPath("")
	if got != pf.GetRoot() {
		t.Fatalf("expected GetAbsPath(\"\") to resolve to the root (%q), got %q", pf.GetRoot(), got)
	}
}
