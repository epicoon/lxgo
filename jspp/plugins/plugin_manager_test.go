package plugins_test

import (
	"testing"

	"github.com/epicoon/lxgo/jspp"
	"github.com/epicoon/lxgo/jspp/component"
	"github.com/epicoon/lxgo/jspp/plugins"
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

// TestPluginManager_Save_PopulatesInMemoryCache is a regression test:
// Save() used to allocate dataSlice at the right length but never fill it
// from the plugins argument, leaving m.data full of zero-valued entries
// (empty Name/Path/Plugin) after every Save() - Has()/Get() would then
// misbehave for anything saved in the same process, even though the
// on-disk file was written correctly.
func TestPluginManager_Save_PopulatesInMemoryCache(t *testing.T) {
	pp := newTestPreprocessor(t)
	m := plugins.NewMap(pp)

	data := []jspp.IPluginData{
		m.NewData("alpha", "/path/alpha", ""),
		m.NewData("beta", "/path/beta", "SomeCtor"),
	}
	if err := m.Save(data); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if !m.Has("alpha") {
		t.Fatalf("expected Has(alpha) to be true right after Save()")
	}
	if !m.Has("beta") {
		t.Fatalf("expected Has(beta) to be true right after Save()")
	}
	if m.Has("") {
		t.Fatalf("expected Has(\"\") to be false - the old bug left zero-valued entries keyed by \"\"")
	}
}
