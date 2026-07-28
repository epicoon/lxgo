package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/epicoon/lxgo/jspp"
	"github.com/epicoon/lxgo/jspp/internal/base"
	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/apptest"
)

// fakePluginManager is a minimal jspp.IPluginManager for
// TestCheckPluginPath_InvalidClientFileType - only NewData is actually
// exercised by checkPluginPath.
type fakePluginManager struct {
	lastName, lastPath, lastPlugin string
}

var _ jspp.IPluginManager = (*fakePluginManager)(nil)

func (m *fakePluginManager) Path() string    { return "" }
func (m *fakePluginManager) Has(string) bool { return false }
func (m *fakePluginManager) Load() error     { return nil }
func (m *fakePluginManager) Reset()          {}
func (m *fakePluginManager) NewData(name, path, plugin string) jspp.IPluginData {
	m.lastName, m.lastPath, m.lastPlugin = name, path, plugin
	return &fakePluginData{name: name, path: path}
}
func (m *fakePluginManager) Save(data []jspp.IPluginData) error { return nil }
func (m *fakePluginManager) Get(string) jspp.IPlugin            { return nil }
func (m *fakePluginManager) SetRoutes(jspp.PluginRoutesList)    {}
func (m *fakePluginManager) Render(jspp.IPlugin, string) (*jspp.PluginRenderInfo, error) {
	return nil, nil
}
func (m *fakePluginManager) HtmlPage(string, string) (string, error) { return "", nil }

type fakePluginData struct{ name, path string }

func (d *fakePluginData) Name() string { return d.name }
func (d *fakePluginData) Path() string { return d.path }

// fakePreprocessor is a minimal jspp.IPreprocessor for
// TestCheckPluginPath_InvalidClientFileType - built by hand (rather than via
// component.SetAppComponent) because internal/utils can't import
// jspp/component (it imports internal/utils itself - a real cycle) nor
// jspp/plugins (same, via internal/utils's own BuildMaps use in
// PluginManager.Reset()).
type fakePreprocessor struct {
	app    kernel.IApp
	config *base.JSPreprocessorConfig
	pm     jspp.IPluginManager

	errs []string
}

var _ jspp.IPreprocessor = (*fakePreprocessor)(nil)

func (p *fakePreprocessor) SetApp(kernel.IApp)                    {}
func (p *fakePreprocessor) SetConfig(kernel.IAppComponentConfig)  {}
func (p *fakePreprocessor) GetConfig() kernel.IAppComponentConfig { return nil }
func (p *fakePreprocessor) Name() string                          { return "fakeJSPreprocessor" }
func (p *fakePreprocessor) App() kernel.IApp                      { return p.app }
func (p *fakePreprocessor) CConfig() kernel.CAppComponentConfig   { return nil }
func (p *fakePreprocessor) AfterInit()                            {}
func (p *fakePreprocessor) LogCategory() string                   { return "fakeJSPreprocessor" }
func (p *fakePreprocessor) Log(msg string, params ...any)         {}
func (p *fakePreprocessor) LogWarning(msg string, params ...any)  {}
func (p *fakePreprocessor) LogError(msg string, params ...any) {
	p.errs = append(p.errs, msg)
}
func (p *fakePreprocessor) Run() error   { return nil }
func (p *fakePreprocessor) Final() error { return nil }

func (p *fakePreprocessor) Config() *base.JSPreprocessorConfig         { return p.config }
func (p *fakePreprocessor) Pathfinder() kernel.IPathfinder             { return p.app.Pathfinder() }
func (p *fakePreprocessor) ModulesMap() jspp.IModulesMap               { return nil }
func (p *fakePreprocessor) PluginManager() jspp.IPluginManager         { return p.pm }
func (p *fakePreprocessor) CompilerBuilder() jspp.ICompilerBuilder     { return nil }
func (p *fakePreprocessor) JSExecutorBuilder() jspp.IJSExecutorBuilder { return nil }

// TestCheckPluginPath_InvalidClientFileType is a regression test for task
// 0105: `file = rawFile.(string)` in checkPluginPath used to panic on a
// non-string `client.file` value in lx-plugin.yaml, instead of logging and
// falling back to "Plugin.js" like the sibling `key` field does.
func TestCheckPluginPath_InvalidClientFileType(t *testing.T) {
	app, err := apptest.New()
	if err != nil {
		t.Fatalf("apptest.New: %v", err)
	}

	// Root outside the app's own pathfinder root, and a non-empty
	// PluginsPath, so checkPluginPath takes the "copy from client.file"
	// branch where the bug lives. makeCopy (further down that branch, past
	// the code this test targets) expects a "module directory" layout -
	// the immediate parent dir named the same as the leaf - to take its
	// copyDir path instead of trying to copyFile a directory.
	pluginRoot := t.TempDir()
	pluginDir := filepath.Join(pluginRoot, "myPlugin", "myPlugin")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatalf("mkdir pluginDir: %v", err)
	}
	yamlContent := "name: myPlugin\nclient:\n  file: 42\n"
	if err := os.WriteFile(filepath.Join(pluginDir, "lx-plugin.yaml"), []byte(yamlContent), 0644); err != nil {
		t.Fatalf("write lx-plugin.yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "Plugin.js"), []byte("// plugin"), 0644); err != nil {
		t.Fatalf("write Plugin.js: %v", err)
	}

	pluginsPath := t.TempDir()
	pp := &fakePreprocessor{
		app: app,
		config: &base.JSPreprocessorConfig{
			PluginsPath: pluginsPath,
		},
		pm: &fakePluginManager{},
	}

	info, err := os.Stat(pluginDir)
	if err != nil {
		t.Fatalf("stat pluginDir: %v", err)
	}

	var ppMap []jspp.IPluginData
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("checkPluginPath panicked on a non-string client.file: %v", r)
			}
		}()
		err = checkPluginPath(pp, pluginDir, info, &ppMap)
	}()
	if err != nil {
		t.Fatalf("checkPluginPath: %v", err)
	}

	if len(pp.errs) == 0 {
		t.Fatalf("expected the invalid client.file value to be logged")
	}
	if len(ppMap) != 1 {
		t.Fatalf("expected exactly one plugin entry, got %d", len(ppMap))
	}
}
