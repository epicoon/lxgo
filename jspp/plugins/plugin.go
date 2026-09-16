// Package plugins provides the default jspp.IPlugin/jspp.IPluginConfig
// implementations (Plugin/Config) and the plugin manager (PluginManager) -
// see the jspp package doc's "Backend implementation" section for how to
// embed Plugin in a custom plugin type.
package plugins

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/epicoon/lxgo/jspp"
	"github.com/epicoon/lxgo/jspp/elems"
	"github.com/epicoon/lxgo/jspp/internal/i18n"
	"github.com/epicoon/lxgo/jspp/internal/md"
	"github.com/epicoon/lxgo/kernel"
	"gopkg.in/yaml.v3"
)

/** @interface conventions.IPlugin */

// Plugin is the base jspp.IPlugin implementation - embed it in your own
// plugin struct and override BeforeRender/AfterRender as needed.
type Plugin struct {
	*elems.Element

	name       string
	path       string
	config     jspp.IPluginConfig
	pathfinder kernel.IPathfinder
	i18n       jspp.II18nMap
}

var _ jspp.IPlugin = (*Plugin)(nil)

/** @constructor */

// NewPlugin constructs an uninitialized Plugin - call Init/SetName/SetPath/
// SetConfig before use (see plugins.NewMap, which does this for plugins
// resolved through the plugin manager).
func NewPlugin() *Plugin {
	return &Plugin{Element: elems.NewElement()}
}

// SetName sets the plugin's name.
func (p *Plugin) SetName(name string) {
	p.name = name
}

// SetPath sets the plugin's source path (resolved to an absolute path) and
// its own IPathfinder.
func (p *Plugin) SetPath(path string) {
	p.path = p.App().Pathfinder().GetAbsPath(path)
	p.pathfinder = newPluginPathfinder(p)
}

// SetConfig sets the plugin's parsed lx-plugin.yaml config.
func (p *Plugin) SetConfig(c jspp.IPluginConfig) {
	p.config = c
}

// Name returns the plugin's name.
func (p *Plugin) Name() string {
	return p.name
}

// Path returns the plugin's absolute source path.
func (p *Plugin) Path() string {
	return p.path
}

// CConfig returns standard config constructor - override this only if you need
// a custom IPluginConfig implementation.
func (p *Plugin) CConfig() jspp.CPluginConfig {
	return NewConfig
}

// Config returns the plugin's parsed lx-plugin.yaml config.
func (p *Plugin) Config() jspp.IPluginConfig {
	return p.config
}

// Pathfinder returns an IPathfinder rooted at the plugin's own directory.
func (p *Plugin) Pathfinder() kernel.IPathfinder {
	return p.pathfinder
}

// I18n returns the plugin's translation map, loading it from its config's
// I18n() paths on first call and caching the result.
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
			p.Preprocessor().LogError("Can not read i18n file '%s' for plugin '%s': %s", fullPath, p.Name(), err)
			continue
		}
		defer file.Close()

		decoder := yaml.NewDecoder(file)
		trI := make(map[string]map[string]string)
		if err := decoder.Decode(trI); err != nil {
			p.Preprocessor().LogError("Can not decode i18n file '%s' for plugin '%s': %s", fullPath, p.Name(), err)
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
					p.Preprocessor().LogError("Duplicate translation in i18n files for plugin '%s': key - %s", p.Name(), key)
					continue
				}
				resolved, err := resolveI18nFileRef(tr, fullPath)
				if err != nil {
					p.Preprocessor().LogError("Can not resolve i18n file reference for plugin '%s', key '%s': %s", p.Name(), key, err)
					resolved = ""
				}
				trMap[lang][key] = resolved
			}
		}
	}

	p.i18n = i18n.NewI18nMap(trMap)
	return p.i18n
}

// i18nFileRefRe matches an i18n value that is entirely a file reference:
// ${^relative/path}. The leading ^ (as opposed to a plain ${name}
// placeholder, substituted at lx.i18n(key, {params}) call sites - see
// package i18n) marks it as "read this file in", not "fill in this
// variable".
var i18nFileRefRe = regexp.MustCompile(`^\$\{\^(.+)\}$`)

// resolveI18nFileRef checks whether tr is entirely a file reference
// (${^path}) and, if so, returns the referenced file's content in its
// place - rendered through the markdown engine when the path ends in
// ".md", taken as plain text otherwise. The path is resolved relative to
// the directory of yamlPath (the i18n file tr came from). tr is returned
// unchanged, with no error, when it isn't a file reference at all.
func resolveI18nFileRef(tr, yamlPath string) (string, error) {
	sub := i18nFileRefRe.FindStringSubmatch(tr)
	if sub == nil {
		return tr, nil
	}

	fullPath := filepath.Join(filepath.Dir(yamlPath), sub[1])
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("can not read i18n-referenced file '%s': %w", fullPath, err)
	}

	if strings.HasSuffix(fullPath, ".md") {
		return md.Convert(string(data)), nil
	}
	return string(data), nil
}

/** @abstract */

// BeforeRender is a no-op - override it to run logic before the plugin's snippets are rendered.
func (p *Plugin) BeforeRender() {
	// Pass
}

/** @abstract */

// AfterRender is a no-op - override it to run logic after rendering.
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

	plugin.Init(pp)
	plugin.SetName(name)
	plugin.SetPath(path)

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
