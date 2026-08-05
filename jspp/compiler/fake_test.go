package compiler

import (
	"github.com/epicoon/lxgo/jspp"
	"github.com/epicoon/lxgo/jspp/internal/base"
	"github.com/epicoon/lxgo/kernel"
)

// fakePreprocessor is a minimal jspp.IPreprocessor for whitebox tests in
// this package - built by hand (rather than via component.SetAppComponent)
// because this package can't import jspp/component: component imports this
// package itself, a real cycle.
type fakePreprocessor struct {
	config *base.JSPreprocessorConfig
	mm     jspp.IModulesMap

	errs []string
}

var _ jspp.IPreprocessor = (*fakePreprocessor)(nil)

func newFakePreprocessor() *fakePreprocessor {
	return &fakePreprocessor{
		config: &base.JSPreprocessorConfig{},
		mm:     &fakeModulesMap{},
	}
}

func (p *fakePreprocessor) SetApp(kernel.IApp)                    {}
func (p *fakePreprocessor) SetConfig(kernel.IAppComponentConfig)  {}
func (p *fakePreprocessor) GetConfig() kernel.IAppComponentConfig { return nil }
func (p *fakePreprocessor) Name() string                          { return "fakeJSPreprocessor" }
func (p *fakePreprocessor) App() kernel.IApp                      { return nil }
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

func (p *fakePreprocessor) Config() *base.JSPreprocessorConfig { return p.config }
func (p *fakePreprocessor) Pathfinder() kernel.IPathfinder     { return nil }
func (p *fakePreprocessor) ModulesMap() jspp.IModulesMap       { return p.mm }
func (p *fakePreprocessor) PluginManager() jspp.IPluginManager { return nil }
func (p *fakePreprocessor) CompilerBuilder() jspp.ICompilerBuilder {
	return Builder().SetPreprocessor(p)
}
func (p *fakePreprocessor) JSExecutorBuilder() jspp.IJSExecutorBuilder { return nil }

// fakeModulesMap is a minimal jspp.IModulesMap - Each is a no-op (no
// registered widgets), which is all lxml.Parser.Widgets() needs to not
// panic when nothing has been registered.
type fakeModulesMap struct{}

var _ jspp.IModulesMap = (*fakeModulesMap)(nil)

func (m *fakeModulesMap) Path() string        { return "" }
func (m *fakeModulesMap) Has(key string) bool { return false }
func (m *fakeModulesMap) Load() error         { return nil }
func (m *fakeModulesMap) Reset()              {}
func (m *fakeModulesMap) NewData(name, path string) jspp.IJSModuleData {
	return &fakeModuleData{name: name, path: path}
}
func (m *fakeModulesMap) Get(moduleName string) jspp.IJSModuleData { return nil }
func (m *fakeModulesMap) Save(data []jspp.IJSModuleData) error     { return nil }
func (m *fakeModulesMap) Each(f func(data jspp.IJSModuleData))     {}

type fakeModuleData struct {
	name, path string
	data       map[string]any
}

func (d *fakeModuleData) AddData(key string, val any) {
	if d.data == nil {
		d.data = map[string]any{}
	}
	d.data[key] = val
}
func (d *fakeModuleData) Name() string         { return d.name }
func (d *fakeModuleData) Path() string         { return d.path }
func (d *fakeModuleData) Data() map[string]any { return d.data }
func (d *fakeModuleData) HasData() bool        { return len(d.data) > 0 }
