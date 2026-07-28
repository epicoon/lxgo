package plugins_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/epicoon/lxgo/jspp/plugins"
)

func newTestPlugin(t *testing.T, path string) *plugins.Plugin {
	t.Helper()
	pp := newTestPreprocessor(t)
	p := plugins.NewPlugin()
	p.Init(pp)
	p.SetName("testPlugin")
	p.SetPath(path)
	return p
}

func writeYAML(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, "lx-plugin.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write lx-plugin.yaml: %v", err)
	}
	return path
}

func TestConfig_Load_Defaults(t *testing.T) {
	dir := t.TempDir()
	yamlPath := writeYAML(t, dir, "name: myPlugin\n")

	p := newTestPlugin(t, dir)
	cfg := plugins.NewConfig()
	cfg.SetPlugin(p)
	if err := cfg.Load(yamlPath); err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Name() != "myPlugin" {
		t.Fatalf("Name() = %q, want myPlugin", cfg.Name())
	}
	if cfg.CacheType() != "inherit" {
		t.Fatalf("CacheType() = %q, want the default 'inherit'", cfg.CacheType())
	}
	if len(cfg.Require()) != 0 {
		t.Fatalf("Require() = %#v, want empty", cfg.Require())
	}
	if got := cfg.Client().File(); got != "Plugin.js" {
		t.Fatalf("Client().File() = %q, want the default Plugin.js", got)
	}
	if got := cfg.Server().RootSnippet(); got != "snippets/_root.js" {
		t.Fatalf("Server().RootSnippet() = %q, want the default", got)
	}
	if got := cfg.Page().Title(); got != "lx" {
		t.Fatalf("Page().Title() = %q, want the default lx", got)
	}
	images := cfg.Images()
	if len(images) != 1 || images["default"] != "" {
		t.Fatalf("Images() = %#v, want {default: \"\"}", images)
	}
	if got := cfg.I18n(); got != nil {
		t.Fatalf("I18n() = %#v, want nil when unconfigured", got)
	}
}

func TestConfig_Load_ConfiguredValues(t *testing.T) {
	dir := t.TempDir()
	yamlPath := writeYAML(t, dir, `
name: myPlugin
cacheType: on
require:
  - a.js
  - b.js
i18n: lang.yaml
images:
  default: img/
  icons: img/icons/
client:
  file: Custom.js
  guiNodes:
    mainBox: MainBox
server:
  key: myKey
`)

	p := newTestPlugin(t, dir)
	cfg := plugins.NewConfig()
	cfg.SetPlugin(p)
	if err := cfg.Load(yamlPath); err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.CacheType() != "on" {
		t.Fatalf("CacheType() = %q, want on", cfg.CacheType())
	}
	if got := cfg.Require(); len(got) != 2 || got[0] != "a.js" || got[1] != "b.js" {
		t.Fatalf("Require() = %#v", got)
	}
	if got := cfg.I18n(); len(got) != 1 || got[0] != "lang.yaml" {
		t.Fatalf("I18n() = %#v, want a single-element slice (string coerced to []string)", got)
	}
	images := cfg.Images()
	if images["default"] != "img/" || images["icons"] != "img/icons/" {
		t.Fatalf("Images() = %#v", images)
	}
	if got := cfg.Client().File(); got != "Custom.js" {
		t.Fatalf("Client().File() = %q, want Custom.js", got)
	}
	guiNodes := cfg.Client().GuiNodes()
	if guiNodes["mainBox"] != "MainBox" {
		t.Fatalf("Client().GuiNodes() = %#v", guiNodes)
	}
	if got := cfg.Server().Key(); got != "myKey" {
		t.Fatalf("Server().Key() = %q, want myKey", got)
	}
}

func TestConfig_Images_PlainString(t *testing.T) {
	dir := t.TempDir()
	yamlPath := writeYAML(t, dir, "name: myPlugin\nimages: img/\n")

	p := newTestPlugin(t, dir)
	cfg := plugins.NewConfig()
	cfg.SetPlugin(p)
	if err := cfg.Load(yamlPath); err != nil {
		t.Fatalf("Load: %v", err)
	}

	images := cfg.Images()
	if len(images) != 1 || images["default"] != "img/" {
		t.Fatalf("Images() = %#v, want {default: img/}", images)
	}
}
