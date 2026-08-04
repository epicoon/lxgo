// Package jspp is a JS preprocessor component for lxgo/kernel applications -
// register it via component.SetAppComponent, then use its IPreprocessor
// (compiler, executor, modules map, plugin manager) to compile and serve
// `lx`-flavored JS on demand. See the package README/doc for the JS-side
// directive syntax (`@lx:...`) this preprocessor understands.
package jspp

import (
	"github.com/epicoon/lxgo/jspp/internal/base"
	"github.com/epicoon/lxgo/kernel"
)

// APP_COMPONENT_KEY is the key the preprocessor registers itself under -
// see component.SetAppComponent/AppComponent.
const APP_COMPONENT_KEY = "lxgo_jspp"

// IPreprocessor is the JS preprocessor app component - the entry point for
// compiling/executing `lx`-flavored JS and resolving modules/plugins. See
// component.JSPreprocessor for the default implementation.
type IPreprocessor interface {
	kernel.IAppComponent

	// Config returns the preprocessor's config.
	Config() *base.JSPreprocessorConfig

	// Pathfinder returns the preprocessor's own IPathfinder, aware of its
	// path aliases (modules, plugins, snippets, ...).
	Pathfinder() kernel.IPathfinder

	// ModulesMap returns the preprocessor's modules map.
	ModulesMap() IModulesMap

	// PluginManager returns the preprocessor's plugin manager.
	PluginManager() IPluginManager

	// CompilerBuilder returns a fresh ICompilerBuilder, pre-bound to this preprocessor.
	CompilerBuilder() ICompilerBuilder

	// JSExecutorBuilder returns a fresh IJSExecutorBuilder, pre-bound to this preprocessor.
	JSExecutorBuilder() IJSExecutorBuilder
}

// IMap is a lazily-loaded, on-disk lookup table - shared shape behind
// IModulesMap and IPluginManager.
type IMap interface {
	// Path returns the map's on-disk file path.
	Path() string

	// Has reports whether key is present, loading the map first if it hasn't been yet.
	Has(key string) bool

	// Load reads the map from disk.
	Load() error

	// Reset rebuilds the map from scratch.
	Reset()
}

// IModulesMap resolves named JS modules (`@lx:module Name;`, referenced via
// `lx.import(Name)`) to their source files - see IPreprocessor.ModulesMap.
type IModulesMap interface {
	IMap

	// NewData constructs an IJSModuleData entry for name/path, not yet saved.
	NewData(name, path string) IJSModuleData

	// Get returns the entry for moduleName, or nil if it isn't in the map.
	Get(moduleName string) IJSModuleData

	// Save persists data as the map's full contents.
	Save(data []IJSModuleData) error

	// Each calls f for every entry in the map.
	Each(f func(data IJSModuleData))
}

// IJSModuleData is one entry in the modules map - a module's name, source
// path, and arbitrary extra data attached via AddData.
type IJSModuleData interface {
	// AddData attaches an arbitrary key/value to the entry.
	AddData(key string, val any)

	// Name returns the module's name.
	Name() string

	// Path returns the module's source path.
	Path() string

	// Data returns all data attached via AddData.
	Data() map[string]any

	// HasData reports whether any data was attached via AddData.
	HasData() bool
}

// II18nMap looks up translations for `lx.i18n(...)` - see the package doc's
// internationalization section.
type II18nMap interface {
	// IsEmpty reports whether the map holds no translations.
	IsEmpty() bool

	// Get returns the translation for key in lang, or "" if there isn't one.
	Get(lang string, key string) string

	// Localize replaces every `lx.i18n('key')` call inside text with its
	// translation for lang (or the key itself if untranslated).
	Localize(text string, lang string) string
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * COMPILER
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

// ICompilerBuilder configures and builds an ICompiler - chain the Set*/other
// configuration methods (each returns the builder itself), then call
// Compiler. See IPreprocessor.CompilerBuilder.
type ICompilerBuilder interface {
	// SetPreprocessor binds the compiler to pp - normally already done by IPreprocessor.CompilerBuilder.
	SetPreprocessor(pp IPreprocessor) ICompilerBuilder

	// SetApp binds the compiler to app.
	SetApp(app kernel.IApp) ICompilerBuilder

	// SetConfig sets the preprocessor config the compiler reads settings (mode, paths) from.
	SetConfig(c *base.JSPreprocessorConfig) ICompilerBuilder

	// SetPathfinder sets the IPathfinder used to resolve source file paths.
	SetPathfinder(pf kernel.IPathfinder) ICompilerBuilder

	// SetLang sets the language used to resolve `lx.i18n(...)` calls.
	SetLang(lang string) ICompilerBuilder

	// SetI18n sets the translation map used to resolve `lx.i18n(...)` calls.
	SetI18n(i18n II18nMap) ICompilerBuilder

	// SetAppContext marks this as a whole-application compile (as opposed
	// to compiling a single plugin/snippet).
	SetAppContext() ICompilerBuilder

	// SetClientContext compiles for the browser side - strips `@lx:<context SERVER: ...`
	// blocks, keeps `CLIENT` ones.
	SetClientContext() ICompilerBuilder

	// SetServerContext compiles for the server side - strips `@lx:<context CLIENT: ...`
	// blocks, keeps `SERVER` ones.
	SetServerContext() ICompilerBuilder

	// SetFilePath sets the source file to compile - mutually exclusive with SetCode.
	SetFilePath(filePath string) ICompilerBuilder

	// SetCompiledModules seeds the set of module names already compiled
	// elsewhere in this build, so `lx.import(Name)` can skip re-including them.
	SetCompiledModules(list []string) ICompilerBuilder

	// SetPrevCode sets code to prepend to the compiled output, unprocessed.
	SetPrevCode(code string) ICompilerBuilder

	// SetCode sets the source code to compile directly - mutually exclusive with SetFilePath.
	SetCode(code string) ICompilerBuilder

	// SetPostCode sets code to append to the compiled output, unprocessed.
	SetPostCode(code string) ICompilerBuilder

	// UseModules marks the given module names as needed by this build,
	// compiling them in even without an explicit `lx.import(Name)`.
	UseModules(modules []string) ICompilerBuilder

	// SetUnwrapped skips IIFE-wrapping the compiled output.
	SetUnwrapped() ICompilerBuilder

	// BuildModules toggles whether `lx.import(Name)` module references are
	// resolved and inlined (true) or left as plain references (false).
	BuildModules(val bool) ICompilerBuilder

	// IgnoreModules excludes the given module names from compilation even
	// if referenced.
	IgnoreModules(list []string) ICompilerBuilder

	// Compiler returns the configured ICompiler, ready to Run.
	Compiler() ICompiler
}

// ICompiler compiles `lx`-flavored JS source into plain JS - see
// ICompilerBuilder to configure and construct one.
type ICompiler interface {
	// Run compiles the configured source (file or code) and returns the compiled output.
	Run() (string, error)

	// Pathfinder returns the compiler's IPathfinder.
	Pathfinder() kernel.IPathfinder

	// Mode returns the configured build mode (e.g. "DEV"), used to resolve
	// `@lx:<mode NAME: ...` blocks.
	Mode() string

	// CompiledModules returns the names of every module compiled during Run.
	CompiledModules() []string

	// ModulesCode returns the compiled code contributed by resolved modules.
	ModulesCode() string

	// CleanCode returns the compiled output without the modules code.
	CleanCode() string

	// Assets returns the extra JS/CSS/module assets (`@lx:js`/`@lx:css`)
	// collected during Run.
	Assets() IAssets

	// CompiledFiles returns the paths of every source file read during Run.
	CompiledFiles() []string
}

// IAssets is a mutable collection of extra assets (`@lx:js`/`@lx:css`
// directives, referenced modules) a compiled build depends on.
type IAssets interface {
	// AddJS registers a plain JS asset by path.
	AddJS(path string)

	// AddCSS registers a plain CSS asset by path.
	AddCSS(path string)

	// AddModule registers a JS module asset by name.
	AddModule(name string)

	// Merge adds asset's entries into this collection.
	Merge(asset IAssets)

	// All returns every asset in the collection.
	All() []IAsset
}

// IAsset is a single asset entry in an IAssets collection.
type IAsset interface {
	// Type returns the asset's kind ("js", "css" or "module").
	Type() string

	// IsJS reports whether the asset is a plain JS file.
	IsJS() bool

	// IsCSS reports whether the asset is a plain CSS file.
	IsCSS() bool

	// IsModule reports whether the asset is a JS module (by name, not path).
	IsModule() bool

	// Src returns the asset's path (or, for a module asset, its name).
	Src() string
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * JS_EXECUTOR
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

// IJSExecutor runs already-compiled JS server-side (via an embedded JS VM) -
// used for server rendering. See IJSExecutorBuilder to configure and
// construct one.
type IJSExecutor interface {
	// Exec runs the configured code and returns its result.
	Exec() (IJSExecResult, error)
}

// IJSExecutorBuilder configures and builds an IJSExecutor - chain SetPreprocessor/SetCode, then call Executor.
type IJSExecutorBuilder interface {
	// Executor returns the configured IJSExecutor, ready to Exec.
	Executor() IJSExecutor

	// SetPreprocessor binds the executor to pp - normally already done by IPreprocessor.JSExecutorBuilder.
	SetPreprocessor(IPreprocessor) IJSExecutorBuilder

	// SetCode sets the JS code to execute.
	SetCode(code string) IJSExecutorBuilder
}

// IJSExecResult is the outcome of an IJSExecutor.Exec run.
type IJSExecResult interface {
	// Log returns console.log-style output, keyed by source.
	Log() map[string][]string

	// Errors returns non-fatal errors raised during execution, keyed by source.
	Errors() map[string][]string

	// Dumps returns any explicit debug dumps produced during execution.
	Dumps() []string

	// Fatal returns the fatal error message that aborted execution, or "" if none occurred.
	Fatal() string

	// Result returns the executed code's final value.
	Result() any
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * ELEMENT
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

// IElement is the Go-side counterpart of a JS `lx.Element` (a widget or
// plugin) that needs its own backend ajax endpoints - see elems.Element for
// the default implementation, and the package doc "An element's own ajax
// channel" section for how a widget wires one up.
type IElement interface {
	// Init binds the element to its owning preprocessor/app - called before any other method.
	Init(pp IPreprocessor)

	// App returns the owning application.
	App() kernel.IApp

	// Preprocessor returns the owning IPreprocessor.
	Preprocessor() IPreprocessor

	// AjaxHandlers returns the element's own ajax routes, dispatched to via
	// `this.ajax(path, params)` on the JS side - empty by default.
	AjaxHandlers() kernel.HttpResourcesList
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * PLUGIN
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

// CPlugin constructs an IPlugin for the given name/path.
type CPlugin func(name, path string) IPlugin

// CPluginConfig constructs an IPluginConfig - see IPlugin.CConfig.
type CPluginConfig func() IPluginConfig

// PluginRenderInfo is a rendered plugin's output - see IPluginManager.Render
// and IPlugin.AfterRender.
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

// PluginRoutesList maps routes to the plugin name served at that route - see IPluginManager.SetRoutes.
type PluginRoutesList map[string]string

// IPluginManager resolves plugin names to IPlugin instances and renders
// them - see IPreprocessor.PluginManager.
type IPluginManager interface {
	IMap

	// NewData constructs an IPluginData entry for name/path/plugin (the DI
	// key for a custom jspp.IPlugin, or "" for the plain base plugin), not yet saved.
	NewData(name, path, plugin string) IPluginData

	// Save persists data as the map's full contents.
	Save(data []IPluginData) error

	// Get returns the IPlugin registered under pluginName, constructing and
	// initializing it on first access.
	Get(pluginName string) IPlugin

	// SetRoutes registers list's routes to serve each plugin as a full HTML page.
	SetRoutes(list PluginRoutesList)

	// Render renders plugin's snippets for lang and returns the result.
	Render(plugin IPlugin, lang string) (*PluginRenderInfo, error)

	// HtmlPage renders plugin as a full HTML page for lang.
	HtmlPage(plugin IPlugin, lang string) (string, error)
}

// IPluginData is one entry in the plugins map - a plugin's name, source
// path, and (via IPluginManager.NewData) its Go DI key.
type IPluginData interface {
	// Name returns the plugin's name.
	Name() string

	// Path returns the plugin's source path.
	Path() string
}

// IPlugin is a plugin's Go-side counterpart - see plugins.Plugin for the
// base implementation (no-op render hooks, no ajax endpoints) to embed and
// override, and the package doc "Backend implementation" section.
type IPlugin interface {
	IElement

	// SetName sets the plugin's name.
	SetName(name string)

	// SetPath sets the plugin's source path.
	SetPath(path string)

	// SetConfig sets the plugin's parsed lx-plugin.yaml config.
	SetConfig(c IPluginConfig)

	// CConfig returns the plugin's config constructor - override this only
	// if you need a custom IPluginConfig implementation.
	CConfig() CPluginConfig

	// Name returns the plugin's name.
	Name() string

	// Path returns the plugin's source path.
	Path() string

	// Config returns the plugin's parsed lx-plugin.yaml config.
	Config() IPluginConfig

	// Pathfinder returns an IPathfinder rooted at the plugin's own directory.
	Pathfinder() kernel.IPathfinder

	// I18n returns the plugin's translation map, loaded from its config's I18n() paths.
	I18n() II18nMap

	// BeforeRender runs before the plugin's snippets are rendered - e.g. to
	// load data the templates need. No-op by default.
	BeforeRender()

	// AfterRender runs after rendering, with the render's result. No-op by default.
	AfterRender(info *PluginRenderInfo)
}

// IPluginConfig is a parsed lx-plugin.yaml - see the package doc "The
// plugin config" section for the file format.
type IPluginConfig interface {
	// SetPlugin binds the config to its owning plugin.
	SetPlugin(plugin IPlugin)

	// Load reads and parses the lx-plugin.yaml file at path.
	Load(path string) error

	// Name returns the plugin's configured name.
	Name() string

	// Images returns the plugin's configured image base paths, keyed by prefix ("default" if unprefixed).
	Images() map[string]string

	// I18n returns the plugin's configured translation file paths.
	I18n() []string

	// CacheType returns the plugin's configured cache mode ("off"/"on"/"dev"/"inherit").
	CacheType() string

	// Require returns extra files/directories the plugin always needs, merged with Client().Require.
	Require() []string

	// CssAssets returns the plugin's configured lx.PluginCssAsset subclasses to plug in.
	CssAssets() []string

	// Server returns the plugin's server-side config section.
	Server() IPluginServerConfig

	// Client returns the plugin's client-side config section.
	Client() IPluginClientConfig

	// Page returns the plugin's page-rendering config section.
	Page() IPluginPageConfig
}

// IPluginServerConfig is an IPluginConfig's "server" section.
type IPluginServerConfig interface {
	// Key returns the Go DI key for a custom jspp.IPlugin implementation,
	// or "" to use the plain base plugin.
	Key() string

	// File returns the server-side entry file, if any.
	File() string

	// RootSnippet returns the snippet rendered as the plugin's root ("snippets/_root.js" by default).
	RootSnippet() string

	// Snippets returns the directories snippets are looked up in ([]string{"snippets"} by default).
	Snippets() []string

	// SnippetsMap returns short-name aliases for snippets, resolved to
	// absolute paths. Config examples:
	//	snippetsMap:
	//	  ext1: "/absolute/path/to/snippet/file.js"
	//	  ext2:
	//	    path: "@alias/snippetFile.js"
	//	  ext3:
	//	    plugin: SomePlugin
	//	    path: path/to/snippet/file.js
	//	  ext4:
	//	    plugin: SomePlugin
	//	    snippet: snippetName
	SnippetsMap() map[string]string

	// Require returns extra files/directories the server side needs, merged into IPluginConfig.Require.
	Require() []string
}

// IPluginClientConfig is an IPluginConfig's "client" section.
type IPluginClientConfig interface {
	// File returns the client-side entry file ("Plugin.js" by default).
	File() string

	// Require returns extra files/directories the client side needs,
	// merged with IPluginConfig.Require into one list.
	Require() []string

	// Core returns the plugin's core JS class name, if customized.
	Core() string

	// GuiNodes returns the plugin's configured GUI node class names, keyed by node name.
	GuiNodes() map[string]string
}

// IPluginPageConfig is an IPluginConfig's "page" section - used when rendering the plugin as a full page.
type IPluginPageConfig interface {
	// Title returns the page's <title> ("lx" by default).
	Title() string

	// Icon returns the page's favicon ("data:," - a blank icon - by default).
	Icon() string

	// Template returns the page's wrapping template, or nil to use the built-in default.
	Template() *PluginTemplate
}

// PluginTemplate names the template a plugin page is wrapped in.
type PluginTemplate struct {
	Namespace string `dict:"namespace"`
	Block     string `dict:"block"`
}
