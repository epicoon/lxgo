package plugins

import (
	"fmt"
	"path/filepath"

	"github.com/epicoon/lxgo/jspp"
	"github.com/epicoon/lxgo/kernel"
	kernelConfig "github.com/epicoon/lxgo/kernel/config"
	"github.com/epicoon/lxgo/kernel/conv"
)

type absConfig struct {
	plugin jspp.IPlugin
	data   *kernel.Config
}

type Config struct {
	absConfig
	server *serverConfig
	client *clientConfig
	page   *pageConfig
	images map[string]string
	i18n   []string
}

var _ jspp.IPluginConfig = (*Config)(nil)

type serverConfig struct {
	absConfig

	//TODO lasy load to the rest of params

	snippetsMap map[string]string
}

var _ jspp.IPluginServerConfig = (*serverConfig)(nil)

type clientConfig struct {
	absConfig
	//TODO lasy load
}

type pageConfig struct {
	absConfig

	//TODO lasy load

	noTpl bool
	tpl   *jspp.PluginTemplate
}

var _ jspp.IPluginClientConfig = (*clientConfig)(nil)

func NewConfig() jspp.IPluginConfig {
	return &Config{
		server: &serverConfig{},
		client: &clientConfig{},
		page:   &pageConfig{},
	}
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * IMPLEMENTATION jspp.IPluginConfig
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

func (c *Config) SetPlugin(plugin jspp.IPlugin) {
	c.plugin = plugin
	c.server.plugin = plugin
	c.client.plugin = plugin
	c.page.plugin = plugin
}

func (c *Config) Load(path string) error {
	data, err := kernelConfig.Load(path)
	if err != nil {
		return err
	}

	c.data = data

	if !kernelConfig.HasParam(c.data, "server") {
		c.server.data = &kernel.Config{}
	} else {
		res, err := kernelConfig.GetParam[kernel.Config](c.data, "server")
		if err != nil {
			return fmt.Errorf("can not get config param 'server' for plugin '%s': %v", path, err)
		}
		c.server.data = &res
	}

	if !kernelConfig.HasParam(c.data, "client") {
		c.client.data = &kernel.Config{}
	} else {
		res, err := kernelConfig.GetParam[kernel.Config](c.data, "client")
		if err != nil {
			return fmt.Errorf("can not get config param 'client' for plugin '%s': %v", path, err)
		}
		c.client.data = &res
	}

	if !kernelConfig.HasParam(c.data, "page") {
		c.page.data = &kernel.Config{}
	} else {
		res, err := kernelConfig.GetParam[kernel.Config](c.data, "page")
		if err != nil {
			return fmt.Errorf("can not get config param 'page' for plugin '%s': %v", path, err)
		}
		c.page.data = &res
	}

	return nil
}

func (c *Config) Name() string {
	return get[string](&c.absConfig, "name", "", "")
}

func (c *Config) Images() map[string]string {
	if c.images != nil {
		return c.images
	}

	if !kernelConfig.HasParam(c.data, "images") {
		return map[string]string{
			"default": "",
		}
	}

	res, err := kernelConfig.GetParam[map[string]string](c.data, "images")
	if err != nil {
		img, err := kernelConfig.GetParam[string](c.data, "images")
		if err == nil {
			res = map[string]string{
				"default": img,
			}
		} else {
			c.logError("can not get config param 'images' for plugin '%s': %v", c.plugin.Name(), err)
			res = map[string]string{
				"default": "",
			}
		}
	}

	c.images = res
	return c.images
}

func (c *Config) I18n() []string {
	if c.i18n != nil {
		return c.i18n
	}

	if !kernelConfig.HasParam(c.data, "i18n") {
		return nil
	}

	res, err := kernelConfig.GetParam[[]string](c.data, "i18n")
	if err != nil {
		str, err := kernelConfig.GetParam[string](c.data, "i18n")
		if err == nil {
			res = []string{str}
		} else {
			c.logError("can not get config param 'i18n' for plugin '%s': %v", c.plugin.Name(), err)
			res = []string{}
		}
	}

	return res
}

func (c *Config) CacheType() string {
	return get[string](&c.absConfig, "cacheType", "on", "")
}

func (c *Config) Require() []string {
	return get[[]string](&c.absConfig, "require", make([]string, 0), "")
}

func (c *Config) CssAssets() []string {
	return get[[]string](&c.absConfig, "cssAssets", make([]string, 0), "")
}

func (c *Config) Server() jspp.IPluginServerConfig {
	return c.server
}

func (c *Config) Client() jspp.IPluginClientConfig {
	return c.client
}

func (c *Config) Page() jspp.IPluginPageConfig {
	return c.page
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * IMPLEMENTATION cnv.IPluginServerConfig
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

func (c *serverConfig) Key() string {
	return get[string](&c.absConfig, "key", "", "server")
}

func (c *serverConfig) RootSnippet() string {
	return get[string](&c.absConfig, "rootSnippet", "snippets/_root.js", "server")
}

func (c *serverConfig) Snippets() []string {
	return get[[]string](&c.absConfig, "snippets", []string{"snippets"}, "server")
}

func (c *serverConfig) SnippetsMap() map[string]string {
	if c.snippetsMap != nil {
		return c.snippetsMap
	}

	raw := get[kernel.Config](&c.absConfig, "snippetsMap", kernel.Config{}, "server")
	if len(raw) == 0 {
		c.snippetsMap = make(map[string]string)
		return c.snippetsMap
	}

	c.snippetsMap = make(map[string]string, len(raw))
	for sName, data := range raw {
		path := c.serializePath(data)
		if path != "" {
			c.snippetsMap[sName] = path
		} else {
			c.plugin.Preprocessor().LogError(fmt.Sprintf("unserializable snippet path for '%s' in plugin '%s'", sName, c.plugin.Name()))
		}
	}

	return c.snippetsMap
}

func (c *serverConfig) Require() []string {
	return get[[]string](&c.absConfig, "require", make([]string, 0), "server")
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * IMPLEMENTATION cnv.IPluginCilentConfig
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

func (c *clientConfig) File() string {
	return get[string](&c.absConfig, "file", "Plugin.js", "client")
}

func (c *clientConfig) Require() []string {
	return get[[]string](&c.absConfig, "require", make([]string, 0), "client")
}

func (c *clientConfig) Core() string {
	return get[string](&c.absConfig, "core", "", "client")
}

func (c *clientConfig) GuiNodes() map[string]string {
	list := get[kernel.Config](&c.absConfig, "guiNodes", kernel.Config{}, "client")
	if len(list) == 0 {
		return make(map[string]string, 0)
	}

	res := make(map[string]string, len(list))
	for key, val := range list {
		sVal, ok := val.(string)
		if !ok {
			c.logError("wrong type of guiNode name for key '%s' for plugin '%s'", key, c.plugin.Name())
			return make(map[string]string, 0)
		}
		res[key] = sVal
	}
	return res
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * IMPLEMENTATION cnv.IPluginPageConfig
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

func (c *pageConfig) Title() string {
	return get[string](&c.absConfig, "title", "lx", "page")
}

func (c *pageConfig) Icon() string {
	return get[string](&c.absConfig, "icon", "data:,", "page")
}

func (c *pageConfig) Template() *jspp.PluginTemplate {
	if c.noTpl {
		return nil
	}

	if c.tpl == nil {
		raw := get[kernel.Config](&c.absConfig, "template", nil, "page")
		if raw == nil {
			c.noTpl = true
			return nil
		}

		c.tpl = &jspp.PluginTemplate{}
		dict := kernel.Dict(raw)
		conv.DictToStruct(&dict, c.tpl)
	}
	return c.tpl
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * PRIVATE
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

func (sc *serverConfig) serializePath(pathData any) string {
	// "path/to/snippet.js"
	sPath, ok := pathData.(string)
	if ok {
		return sc.plugin.Pathfinder().GetAbsPath(sPath)
	}

	mPath, ok := pathData.(kernel.Config)
	if !ok {
		return ""
	}

	// {"path": "path/to/snippet.js"}
	if len(mPath) == 1 {
		path, ok := mPath["path"].(string)
		if !ok {
			return ""
		}
		return sc.plugin.Pathfinder().GetAbsPath(path)
	}

	// {"plugin": "PluginName", "path": "path/to/snippet.js"}
	// OR
	// {"plugin": "PluginName", "snippet": "SnippetName"}
	if len(mPath) == 2 {
		plugin, ok := mPath["plugin"].(string)
		if !ok {
			return ""
		}

		path, ok := mPath["path"].(string)
		if ok {
			sPath := filepath.Join(fmt.Sprintf("{plugin:%s}", plugin), path)
			return sc.plugin.Pathfinder().GetAbsPath(sPath)
		}

		snippet, ok := mPath["snippet"].(string)
		if ok {
			sPath := fmt.Sprintf("{snippet:%s.%s}", plugin, snippet)
			return sc.plugin.Pathfinder().GetAbsPath(sPath)
		}
	}

	return ""
}

func (c *absConfig) logError(msg string, params ...any) {
	c.plugin.Preprocessor().LogError(msg, params...)
}

func get[T any](c *absConfig, key string, defaultVal T, errParamPrefix string) T {
	if !kernelConfig.HasParam(c.data, key) {
		return defaultVal
	}
	if errParamPrefix != "" {
		errParamPrefix += "."
	}
	res, err := kernelConfig.GetParam[T](c.data, key)
	if err != nil {
		c.logError("can not get config param '%s%s' for plugin '%s': %v", errParamPrefix, key, c.plugin.Name(), err)
		return defaultVal
	}
	return res
}
