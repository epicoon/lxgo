package plugins

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/epicoon/lxgo/jspp"
	"github.com/epicoon/lxgo/jspp/internal/handlers"
	"github.com/epicoon/lxgo/jspp/internal/utils"
	"github.com/epicoon/lxgo/kernel"
)

/** @interface conventions.IPluginsMap */
type PluginManager struct {
	pp      jspp.IPreprocessor
	data    map[string]pluginData
	plugins map[string]jspp.IPlugin
}

var _ jspp.IPluginManager = (*PluginManager)(nil)

func NewMap(pp jspp.IPreprocessor) jspp.IPluginManager {
	return &PluginManager{pp: pp}
}

func (m *PluginManager) Path() string {
	app := m.pp.App()
	dir := app.Pathfinder().GetAbsPath(m.pp.Config().MapsPath)
	return filepath.Join(dir, "_plugins.json")
}

func (m *PluginManager) Has(key string) bool {
	if m.data == nil {
		if err := m.Load(); err != nil {
			m.pp.LogError("can not load plugins map: %v", err)
			return false
		}
	}
	_, ok := m.data[key]
	return ok
}

func (m *PluginManager) Get(pluginName string) jspp.IPlugin {
	if p, ok := m.plugins[pluginName]; ok {
		return p
	}

	if m.data == nil {
		if err := m.Load(); err != nil {
			m.pp.LogError("can not load plugins map: %v", err)
			return nil
		}
	}

	pluginData, ok := m.data[pluginName]
	if !ok {
		return nil
	}

	var plugin jspp.IPlugin
	if pluginData.Plugin() != "" {
		app := m.pp.App()
		p := app.DIContainer().Get(pluginData.Plugin())
		attempt, ok := p.(jspp.IPlugin)
		if ok {
			plugin = attempt
		} else {
			m.pp.LogError("Invalid constructor for plugin %s", pluginData.Plugin())
		}
	}

	if plugin == nil {
		plugin = NewPlugin()
	}

	if err := initPlugin(plugin, m.pp, pluginData.Ename, pluginData.Epath); err != nil {
		m.pp.LogError("Can not init plugin %s: %v", pluginName, err)
		return nil
	}

	if m.plugins == nil {
		m.plugins = make(map[string]jspp.IPlugin, 1)
	}
	m.plugins[pluginName] = plugin

	return plugin
}

func (m *PluginManager) Load() error {
	path := m.Path()

	d, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("can not read file %s: %w", path, err)
	}

	var plugins []pluginData
	if err := json.Unmarshal(d, &plugins); err != nil {
		return fmt.Errorf("can not parse JSON from %s: %w", path, err)
	}

	m.data = mapping(plugins)

	return nil
}

func (m *PluginManager) Reset() {
	utils.BuildMaps(m.pp, utils.MapBuilderOptions{Plugins: true})
}

func (m *PluginManager) NewData(name, path, plugin string) jspp.IPluginData {
	return &pluginData{Ename: name, Epath: path, Eplugin: plugin}
}

func (m *PluginManager) Save(plugins []jspp.IPluginData) error {
	filePath := m.Path()
	dir := filepath.Dir(filePath)

	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return fmt.Errorf("can not make directory: %w", err)
	}

	jsonData, err := json.MarshalIndent(plugins, "", "  ")
	if err != nil {
		return fmt.Errorf("can not serialize JSON: %w", err)
	}

	if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
		return fmt.Errorf("can not write file: %w", err)
	}

	dataSlice := make([]pluginData, len(plugins))
	m.data = mapping(dataSlice)

	return nil
}

func (m *PluginManager) SetRoutes(list jspp.PluginRoutesList) {
	router := m.pp.App().Router()

	for url := range list {
		router.RegisterResource(url, "GET", handlers.NewPluginPageHandler)
	}

	router.AddMiddleware(func(ctx kernel.IHandleContext) error {
		r := ctx.Route()
		pluginName, exists := list[r]
		if !exists {
			return nil
		}

		ctx.Set("jspp", m.pp)
		ctx.Set("pluginName", pluginName)
		return nil
	})
}

func (m *PluginManager) Render(plugin jspp.IPlugin, lang string) (*jspp.PluginRenderInfo, error) {
	renderer := newPluginRenderer(m.pp, plugin, "", lang)
	res := renderer.run()
	if renderer.HasErrors() {
		return nil, renderer.GetFirstError()
	}
	return res, nil
}

func (m *PluginManager) HtmlPage(pluginName, lang string) (string, error) {
	plugin := m.pp.PluginManager().Get(pluginName)
	if plugin == nil {
		return "", fmt.Errorf("can not find plugin '%s'", pluginName)
	}
	r := newRenderer(m, plugin, lang)
	return r.render()
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * PRIVATE
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

func mapping(d []pluginData) map[string]pluginData {
	result := make(map[string]pluginData, len(d))
	for _, val := range d {
		result[val.Name()] = val
	}
	return result
}

type pluginData struct {
	Epath   string `json:"path"`
	Ename   string `json:"name"`
	Eplugin string `json:"plugin,omitempty"`
}

var _ jspp.IPluginData = (*pluginData)(nil)

func (d *pluginData) Path() string {
	return d.Epath
}

func (d *pluginData) Name() string {
	return d.Ename
}

func (d *pluginData) Plugin() string {
	return d.Eplugin
}
