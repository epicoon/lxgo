package base

import (
	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/app"
)

type JSPreprocessorConfig struct {
	*app.ComponentConfig
	Mode            string
	CorePath        string
	MapsPath        string
	ModsPath        string
	PluginsPath     string
	SysPath         string
	PluginCacheType string
	AssetLinksPath  struct {
		Inner string
		Outer string
	}
	AppConfig          string
	CssScopeRenderSide string
	ModulesSrc         []string
	ModulesLinks       []string
	ModulesIgnore      []string
	Plugins            []string
	Targets            []Target
	ModuleInjector     map[string]string
}

/** @constructor kernel.CAppComponentConfig */
func NewJSPreprocessorConfig() kernel.IAppComponentConfig {
	return &JSPreprocessorConfig{
		ComponentConfig:    app.NewComponentConfigStruct(),
		CssScopeRenderSide: "client",
		ModulesSrc:         []string{},
		ModulesLinks:       []string{},
		ModulesIgnore:      []string{},
		Plugins:            []string{},
		Targets:            []Target{},
		ModuleInjector:     map[string]string{},
	}
}

type Target struct {
	EntryPoint string
	Output     string
	Type       string
}
