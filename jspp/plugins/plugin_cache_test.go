package plugins_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/epicoon/lxgo/jspp/component"
	"github.com/epicoon/lxgo/jspp/plugins"
	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/apptest"
)

// A render whose root snippet fails to execute (here: the built core.js
// isn't there) must not be written to the plugin cache - otherwise the broken
// result keeps being served from cache on every later request, even after
// the cause is fixed.
func TestPluginRender_FailedSnippetIsNotCached(t *testing.T) {
	for _, cacheType := range []string{plugins.CACHE_ON, plugins.CACHE_DEV} {
		t.Run(cacheType, func(t *testing.T) {
			sysPath := t.TempDir()
			pluginsPath := t.TempDir()
			app, err := apptest.New(kernel.Dict{
				"Components": kernel.Dict{
					"JSPreprocessor": kernel.Dict{
						"SysPath":     sysPath,
						"MapsPath":    sysPath,
						"PluginsPath": pluginsPath,
						"CorePath":    filepath.Join(sysPath, "missing", "core.js"),
						"AssetLinksPath": kernel.Dict{
							"Inner": filepath.Join(sysPath, "assets"),
							"Outer": "/web",
						},
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

			dir := t.TempDir()
			if err := os.MkdirAll(filepath.Join(dir, "snippets"), 0755); err != nil {
				t.Fatalf("mkdir snippets: %v", err)
			}
			if err := os.WriteFile(filepath.Join(dir, "snippets", "_root.js"), []byte("new lx.Box({geom: true});"), 0644); err != nil {
				t.Fatalf("write snippet: %v", err)
			}
			yamlPath := writeYAML(t, dir,
				"name: cachedPlugin\ncacheType: "+cacheType+"\nserver:\n  rootSnippet: snippets/_root.js\n")

			p := plugins.NewPlugin()
			p.Init(pp)
			p.SetName("cachedPlugin")
			p.SetPath(dir)
			cfg := plugins.NewConfig()
			cfg.SetPlugin(p)
			if err := cfg.Load(yamlPath); err != nil {
				t.Fatalf("Load: %v", err)
			}
			p.SetConfig(cfg)

			if _, err := pp.PluginManager().Render(p, "en-EN"); err != nil {
				t.Logf("Render: %v", err)
			}

			if _, err := os.Stat(filepath.Join(pluginsPath, "lx_cache", "cachedPlugin")); !os.IsNotExist(err) {
				t.Fatalf("failed render was cached (stat err = %v), want no cache written", err)
			}
		})
	}
}
