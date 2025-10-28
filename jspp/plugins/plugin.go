package plugins

import (
	"os"
	"path/filepath"

	"github.com/epicoon/lxgo/jspp"
	"github.com/epicoon/lxgo/jspp/internal/i18n"
	"github.com/epicoon/lxgo/kernel"
	"gopkg.in/yaml.v3"
)

/** @interface conventions.IPlugin */
type Plugin struct {
	name string
	path string

	pp         jspp.IPreprocessor
	app        kernel.IApp
	config     jspp.IPluginConfig
	pathfinder kernel.IPathfinder
	i18n       jspp.II18nMap
}

var _ jspp.IPlugin = (*Plugin)(nil)

/** @constructor */
func NewPlugin() *Plugin {
	return &Plugin{}
}

func (p *Plugin) Init(pp jspp.IPreprocessor, name, path string) {
	p.pp = pp
	p.app = pp.App()
	p.name = name
	p.path = p.app.Pathfinder().GetAbsPath(path)
	p.pathfinder = newPluginPathfinder(p)
}

func (p *Plugin) SetConfig(c jspp.IPluginConfig) {
	p.config = c
}

func (p *Plugin) Name() string {
	return p.name
}

func (p *Plugin) Path() string {
	return p.path
}

func (p *Plugin) CConfig() jspp.CPluginConfig {
	return NewConfig
}

func (p *Plugin) Preprocessor() jspp.IPreprocessor {
	return p.pp
}

func (p *Plugin) App() kernel.IApp {
	return p.app
}

func (p *Plugin) Config() jspp.IPluginConfig {
	return p.config
}

func (p *Plugin) Pathfinder() kernel.IPathfinder {
	return p.pathfinder
}

func (p *Plugin) I18n() jspp.II18nMap {
	if p.i18n != nil {
		return p.i18n
	}

	trMap := make(map[string]map[string]string)
	paths := p.Config().I18n()
	for _, path := range paths {
		fullPath := p.Pathfinder().GetAbsPath(path)
		file, err := os.Open(fullPath)
		if err != nil {
			p.pp.LogError("Can not read i18n file '%s' for plugin '%s': %s", fullPath, p.Name(), err)
			continue
		}
		defer file.Close()

		decoder := yaml.NewDecoder(file)
		trI := make(map[string]map[string]string)
		if err := decoder.Decode(trI); err != nil {
			p.pp.LogError("Can not decode i18n file '%s' for plugin '%s': %s", fullPath, p.Name(), err)
			continue
		}

		for lang, trs := range trI {
			_, exists := trMap[lang]
			if !exists {
				trMap[lang] = make(map[string]string, len(trs))
			}
			for key, tr := range trI[lang] {
				_, exists := trMap[lang][key]
				if exists {
					p.pp.LogError("Duplicate translation in i18n files for plugin '%s': key - %s", p.Name(), key)
					continue
				}
				trMap[lang][key] = tr
			}
		}
	}

	p.i18n = i18n.NewI18nMap(trMap)
	return p.i18n
}

func (p *Plugin) AjaxHandlers() kernel.HttpResourcesList {
	return make(kernel.HttpResourcesList, 0)
}

/** @abstract */
func (p *Plugin) BeforeRender() {
	// Pass
}

/** @abstract */
func (p *Plugin) AfterRender(info *jspp.PluginRenderInfo) {
	// Pass
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * PRIVATE
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

func initPlugin(plugin jspp.IPlugin, pp jspp.IPreprocessor, name, path string) error {
	if plugin.Config() != nil {
		return nil
	}

	plugin.Init(pp, name, path)

	cConf := plugin.CConfig()
	conf := cConf()
	confPath := filepath.Join(plugin.Path(), "lx-plugin.yaml")
	if err := conf.Load(confPath); err != nil {
		return err
	}

	conf.SetPlugin(plugin)
	plugin.SetConfig(conf)
	return nil
}
