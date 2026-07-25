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

// Config is the default jspp.IPluginConfig implementation - a parsed lx-plugin.yaml.
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

// NewConfig constructs an empty Config, ready for Load.
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

// SetPlugin binds the config (and its server/client/page sections) to its owning plugin.
func (c *Config) SetPlugin(plugin jspp.IPlugin) {
	c.plugin = plugin
	c.server.plugin = plugin
	c.client.plugin = plugin
	c.page.plugin = plugin
}

// Load reads and parses the lx-plugin.yaml file at path.
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

// Name returns the plugin's configured name.
func (c *Config) Name() string {
	return get[string](&c.absConfig, "name", "", "")
}

// Images returns the plugin's configured image base paths, keyed by prefix
// ("default" if unprefixed) - {"default": ""} if unconfigured.
func (c *Config) Images() (result map[string]string) {
	defer func() {
		if c.images == nil {
			c.images = map[string]string{
				"default": "",
			}
		}
		result = c.images
	}()

	if c.images != nil {
		return c.images
	}

	if !kernelConfig.HasParam(c.data, "images") {
		return map[string]string{
			"default": "",
		}
	}

	res, err := kernelConfig.GetParam[kernel.Config](c.data, "images")
	if err != nil {
		img, err := kernelConfig.GetParam[string](c.data, "images")
		if err == nil {
			c.images = map[string]string{
				"default": img,
			}
			return c.images
		}
		c.logError("can not get config param 'images' for plugin '%s': %v", c.plugin.Name(), err)
		return
	}

	m := res.ToMap()
	c.images = make(map[string]string, len(m))
	for key, val := range m {
		s, ok := val.(string)
		if !ok {
			c.logError("can not get config param 'images' for plugin '%s': %v", c.plugin.Name(), err)
			return
		}
		c.images[key] = s
	}
	return c.images
}

// I18n returns the plugin's configured translation file paths.
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

// CacheType returns the plugin's configured cache mode ("inherit" by default).
func (c *Config) CacheType() string {
	return get[string](&c.absConfig, "cacheType", "inherit", "")
}

// Require returns extra files/directories the plugin always needs, merged with Client().Require.
func (c *Config) Require() []string {
	return get[[]string](&c.absConfig, "require", make([]string, 0), "")
}

// CssAssets returns the plugin's configured lx.PluginCssAsset subclasses to plug in.
func (c *Config) CssAssets() []string {
	return get[[]string](&c.absConfig, "cssAssets", make([]string, 0), "")
}

// Server returns the plugin's server-side config section.
func (c *Config) Server() jspp.IPluginServerConfig {
	return c.server
}

// Client returns the plugin's client-side config section.
func (c *Config) Client() jspp.IPluginClientConfig {
	return c.client
}

// Page returns the plugin's page-rendering config section.
func (c *Config) Page() jspp.IPluginPageConfig {
	return c.page
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * IMPLEMENTATION cnv.IPluginServerConfig
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

// Key returns the Go DI key for a custom jspp.IPlugin implementation, or "" to use the plain base plugin.
func (c *serverConfig) Key() string {
	return get[string](&c.absConfig, "key", "", "server")
}

// File returns the server-side entry file, if any.
func (c *serverConfig) File() string {
	return get[string](&c.absConfig, "file", "", "server")
}

// RootSnippet returns the snippet rendered as the plugin's root ("snippets/_root.js" by default).
func (c *serverConfig) RootSnippet() string {
	return get[string](&c.absConfig, "rootSnippet", "snippets/_root.js", "server")
}

// Snippets returns the directories snippets are looked up in ([]string{"snippets"} by default).
func (c *serverConfig) Snippets() []string {
	return get[[]string](&c.absConfig, "snippets", []string{"snippets"}, "server")
}

// SnippetsMap returns short-name aliases for snippets, resolved to absolute
// paths - see jspp.IPluginServerConfig.SnippetsMap for the config format.
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

// Require returns extra files/directories the server side needs, merged into jspp.IPluginConfig.Require.
func (c *serverConfig) Require() []string {
	return get[[]string](&c.absConfig, "require", make([]string, 0), "server")
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * IMPLEMENTATION cnv.IPluginClientConfig
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

// File returns the client-side entry file ("Plugin.js" by default).
func (c *clientConfig) File() string {
	return get[string](&c.absConfig, "file", "Plugin.js", "client")
}

// Require returns extra files/directories the client side needs, merged
// with jspp.IPluginConfig.Require into one list.
func (c *clientConfig) Require() []string {
	return get[[]string](&c.absConfig, "require", make([]string, 0), "client")
}

// Core returns the plugin's core JS class name, if customized.
func (c *clientConfig) Core() string {
	return get[string](&c.absConfig, "core", "", "client")
}

// GuiNodes returns the plugin's configured GUI node class names, keyed by node name.
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

// Title returns the page's <title> ("lx" by default).
func (c *pageConfig) Title() string {
	return get[string](&c.absConfig, "title", "lx", "page")
}

// Icon returns the page's favicon ("data:," - a blank icon - by default).
func (c *pageConfig) Icon() string {
	return get[string](&c.absConfig, "icon", "data:,", "page")
}

// Template returns the page's wrapping template, or nil to use the built-in default.
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
