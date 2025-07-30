package jspp

import (
	"github.com/epicoon/lxgo/jspp/internal/base"
	"github.com/epicoon/lxgo/kernel"
)

const APP_COMPONENT_KEY = "lxgo_jspp"

type IPreprocessor interface {
	Name() string
	App() kernel.IApp
	Config() *base.JSPreprocessorConfig
	Pathfinder() kernel.IPathfinder
	ModulesMap() IModulesMap
	PluginManager() IPluginManager
	CompilerBuilder() ICompilerBuilder
	ExecutorBuilder() IExecutorBuilder
	LogError(msg string, params ...any)
}

type IMap interface {
	Path() string
	Has(key string) bool
	Load() error
	Reset()
}

type IModulesMap interface {
	IMap
	NewData(name, path string) IJSModuleData
	Get(moduleName string) IJSModuleData
	Save(data []IJSModuleData) error
	Each(f func(data IJSModuleData))
}

type IJSModuleData interface {
	AddData(key string, val any)
	Name() string
	Path() string
	Data() map[string]any
	HasData() bool
}

type II18nMap interface {
	IsEmpty() bool
	Get(lang string, key string) string
	Localize(text string, lang string) string
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * COMPILER
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

type ICompilerBuilder interface {
	SetPreprocessor(pp IPreprocessor) ICompilerBuilder
	SetApp(app kernel.IApp) ICompilerBuilder
	SetConfig(c *base.JSPreprocessorConfig) ICompilerBuilder
	SetPathfinder(pf kernel.IPathfinder) ICompilerBuilder
	SetLang(lang string) ICompilerBuilder
	SetI18n(i18n II18nMap) ICompilerBuilder
	SetAppContext() ICompilerBuilder
	SetClientContext() ICompilerBuilder
	SetServerContext() ICompilerBuilder
	SetFilePath(filePath string) ICompilerBuilder
	SetCompiledModules(list []string) ICompilerBuilder
	SetPrevCode(code string) ICompilerBuilder
	SetCode(code string) ICompilerBuilder
	SetPostCode(code string) ICompilerBuilder
	UseModules(modules []string) ICompilerBuilder
	SetUnwrapped() ICompilerBuilder
	BuildModules(val bool) ICompilerBuilder
	IgnoreModules(list []string) ICompilerBuilder
	Compiler() ICompiler
}

type ICompiler interface {
	Run() (string, error)
	Pathfinder() kernel.IPathfinder
	Mode() string
	CompiledModules() []string
	ModulesCode() string
	CleanCode() string
	Assets() IAssets
}

type IAssets interface {
	AddJS(path string)
	AddCSS(path string)
	AddModule(name string)
	Merge(asset IAssets)
	All() []IAsset
}

type IAsset interface {
	Type() string
	IsJS() bool
	IsCSS() bool
	IsModule() bool
	Path() string
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * EXECUTOR
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

type IExecutor interface {
	Exec() (IExecResult, error)
}

type IExecutorBuilder interface {
	Executor() IExecutor
	SetPreprocessor(IPreprocessor) IExecutorBuilder
	SetCode(code string) IExecutorBuilder
}

type IExecResult interface {
	Log() map[string][]string
	Errors() map[string][]string
	Dumps() []string
	Fatal() string
	Result() any
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * PLUGIN
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

type CPlugin func(name, path string) IPlugin
type CPluginConfig func() IPluginConfig

type PluginRenderInfo struct {
	Html   string         `json:"html"`
	Root   string         `json:"root"`
	Lx     map[string]any `json:"lx"`
	Assets struct {
		Modules []string `json:"modules,omitempty"`
		Scripts []string `json:"scripts,omitempty"`
		Css     []string `json:"css,omitempty"`
	} `json:"assets"`
}

type PluginRoutesList map[string]string

type IPluginManager interface {
	IMap
	NewData(name, path, plugin string) IPluginData
	Save(data []IPluginData) error
	Get(pluginName string) IPlugin
	SetRoutes(list PluginRoutesList)
	Render(plugin IPlugin, lang string) (*PluginRenderInfo, error)
	HtmlPage(pluginName, lang string) (string, error)
}

type IPluginData interface {
	Name() string
	Path() string
}

type IPlugin interface {
	Init(pp IPreprocessor, name, path string)
	SetConfig(c IPluginConfig)
	CConfig() CPluginConfig
	Name() string
	Path() string
	Preprocessor() IPreprocessor
	App() kernel.IApp
	Config() IPluginConfig
	Pathfinder() kernel.IPathfinder
	I18n() II18nMap
	BeforeRender()
	AfterRender(info *PluginRenderInfo)
}

type IPluginConfig interface {
	SetPlugin(plugin IPlugin)
	Load(path string) error
	Name() string
	Images() map[string]string
	I18n() []string
	CacheType() string
	Require() []string
	CssAssets() []string
	Server() IPluginServerConfig
	Client() IPluginClientConfig
	Page() IPluginPageConfig
}

type IPluginServerConfig interface {
	Key() string
	RootSnippet() string
	Snippets() []string
	/**
	 * Config examples:
	 * snippetsMap:
	 *   ext1: "/absolute/path/to/snippet/file.js"
	 *   ext2:
	 *     path: "@alias/snippetFile.js"
	 *   ext3:
	 *     plugin: SomePlugin
	 *     path: path/to/snippet/file.js
	 *   ext4:
	 *     plugin: SomePlugin
	 *     snippet: snippetName
	 */
	SnippetsMap() map[string]string
	Require() []string
}

type IPluginClientConfig interface {
	File() string
	Require() []string
	Core() string
	GuiNodes() map[string]string
}

type IPluginPageConfig interface {
	Title() string
	Icon() string
	Template() PluginTemplate
}

type PluginTemplate struct {
	Namespace string `dict:"namespace"`
	Block     string `dict:"block"`
}
