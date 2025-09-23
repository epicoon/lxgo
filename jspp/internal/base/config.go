package base

import (
	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/app"
)

type JSPreprocessorConfig struct {
	*app.ComponentConfig
	Mode           string
	CorePath       string
	MapsPath       string
	ModsPath       string
	AssetLinksPath struct {
		Inner string
		Outer string
	}
	AppConfig          string
	CssScopeRenderSide string
	Modules            []string
	Plugins            []string
	Targets            []Target
}

/** @constructor kernel.CAppComponentConfig */
func NewJSPreprocessorConfig() kernel.IAppComponentConfig {
	return &JSPreprocessorConfig{
		ComponentConfig:    app.NewComponentConfigStruct(),
		CssScopeRenderSide: "client",
		Modules:            []string{},
		Plugins:            []string{},
		Targets:            []Target{},
	}
}

type Target struct {
	EntryPoint string
	Output     string
	Type       string
}
