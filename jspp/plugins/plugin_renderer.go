package plugins

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/epicoon/lxgo/jspp"
	"github.com/epicoon/lxgo/jspp/internal/compiler"
	"github.com/epicoon/lxgo/kernel/conv"
	lxErrors "github.com/epicoon/lxgo/kernel/errors"
	"github.com/epicoon/lxgo/kernel/utils"
)

type pluginRenderer struct {
	*lxErrors.ErrorsCollector

	pp     jspp.IPreprocessor
	plugin jspp.IPlugin
	conf   *pluginConf
	lang   string
	title  string
	icon   string

	compiled   bool
	key        string
	rootSR     *snippetRenderer
	nested     []*pluginRenderer
	nestedConf []*nestedPluginConf
	depFiles   map[string]bool

	html            string
	rootSnippetKey  string
	serverCodeBlank string

	assets jspp.IAssets
	output pluginOutput
}

type pluginOutput struct {
	Conf     map[string]any          `json:"conf"`
	Js       string                  `json:"js"`
	Snippets map[string]*snippetConf `json:"snippets"`
}

type pluginConf struct {
	// Common
	name       string
	cssScope   string
	imagePaths map[string]string

	// Server
	params map[string]any

	// Client
	data   map[string]any
	onLoad []string
}

type nestedPluginConf struct {
	Hash     string         `dict:"hash"`
	Name     string         `dict:"name"`
	CssScope string         `dict:"cssScope"`
	Params   map[string]any `dict:"params"`
	OnLoad   string         `dict:"onLoad"`
}

func newPluginRenderer(pp jspp.IPreprocessor, plugin jspp.IPlugin, hash, lang string) *pluginRenderer {
	if hash == "" {
		str := plugin.Path() + utils.GenRandomHash(16)
		hashMd5 := md5.New()
		io.WriteString(hashMd5, str)
		hash = fmt.Sprintf("%x", hashMd5.Sum(nil))
	}

	return &pluginRenderer{
		ErrorsCollector: lxErrors.NewErrorsCollector(),
		pp:              pp,
		lang:            lang,
		plugin:          plugin,
		key:             hash,
		assets:          &compiler.Assets{},
		conf:            &pluginConf{},
		nestedConf:      make([]*nestedPluginConf, 0),
		depFiles:        make(map[string]bool),
	}
}

func (r *pluginRenderer) run() *jspp.PluginRenderInfo {
	r.plugin.BeforeRender()
	r.compile()

	result := &jspp.PluginRenderInfo{
		Root: r.key,
		Lx:   make(map[string]any, 1),
	}
	commonAssets := compiler.Assets{}
	r.eachContext(func(pr *pluginRenderer) {
		commonAssets.Merge(pr.assets)

		if result.Html == "" {
			result.Html = pr.html
		} else {
			re := regexp.MustCompile(fmt.Sprintf(`lx-plugin="%s"[^>]*?>`, pr.key))
			result.Html = re.ReplaceAllStringFunc(result.Html, func(match string) string {
				return match + pr.html
			})
		}

		result.Lx[pr.key] = pr.output
	})

	for _, as := range commonAssets.All() {
		switch {
		case as.IsJS():
			result.Assets.Scripts = append(result.Assets.Scripts, as.Path())
		case as.IsCSS():
			result.Assets.Css = append(result.Assets.Css, as.Path())
		case as.IsModule():
			result.Assets.Modules = append(result.Assets.Modules, as.Path())
		}
	}

	al := newAssetLinker(r.pp, r.plugin.Pathfinder())
	result.Assets.Scripts = al.getAssetsSlice(result.Assets.Scripts)

	al.reset()
	result.Assets.Css = al.getAssetsSlice(result.Assets.Css)

	r.plugin.AfterRender(result)
	return result
}

func (r *pluginRenderer) compileProcess() {
	if r.compileMainJs(); r.HasErrors() {
		r.pp.LogError("error while plugin '%s' JS-compile: %v", r.plugin.Name(), r.GetFirstError())
	}
	// Get: r.output.Js

	r.compileSnippet()
	// Get: r.output.Snippets
	// Get: r.nestedConf
}

func (r *pluginRenderer) compile() {
	if r.compiled {
		return
	}

	r.preparePluginConf()

	// Compile or load cache
	cache := newPluginCache(r)
	switch cache.Type() {
	case CACHE_OFF:
		r.compileProcess()
	case CACHE_ON:
		if !cache.Exists() {
			r.compileProcess()
			if err := cache.Save(); err != nil {
				r.pp.LogError("can not save cache for plugin '%s': %v", r.plugin.Name(), err)
			}
		} else if err := cache.Load(); err != nil {
			r.pp.LogError("can not load cache for plugin '%s', recompiling: %v", r.plugin.Name(), err)
			r.compileProcess()
		}
	case CACHE_DEV:
		if !cache.Exists() || cache.DepsChanged() {
			r.compileProcess()
			if err := cache.Save(); err != nil {
				r.pp.LogError("can not save cache for plugin '%s': %v", r.plugin.Name(), err)
			}
		} else if err := cache.Load(); err != nil {
			r.pp.LogError("can not load cache for plugin '%s', recompiling: %v", r.plugin.Name(), err)
			r.compileProcess()
		}
	default:
		r.compileProcess()
	}

	// Compile nested plugins
	for _, conf := range r.nestedConf {
		plugin := r.pp.PluginManager().Get(conf.Name)
		if plugin == nil {
			r.pp.LogError("plugin '%s' not found", conf.Name)
			continue
		}

		nested := newPluginRenderer(r.pp, plugin, conf.Hash, r.lang)
		nested.conf.cssScope = conf.CssScope
		nested.conf.params = conf.Params
		nested.conf.onLoad = []string{conf.OnLoad}

		r.nested = append(r.nested, nested)
		nested.compile()
	}

	r.compilePluginConf()
	if r.HasErrors() {
		r.pp.LogError("error while plugin '%s' config compile: %v", r.plugin.Name(), r.GetFirstError())
	}

	r.compiled = true
}

func (r *pluginRenderer) compileSnippet() {
	snippetPath := r.plugin.Config().Server().RootSnippet()
	snippetPath = r.plugin.Pathfinder().GetAbsPath(snippetPath)
	r.rootSR = newSnippetRenderer(r, "root", snippetPath, map[string]any{})
	r.rootSR.run()
	r.rootSnippetKey = r.rootSR.snippet.key
	r.html = r.rootSR.html
	r.output.Snippets = r.rootSR.output
}

func (r *pluginRenderer) compileMainJs() {
	plugin := r.plugin

	var code string
	filePath := plugin.Pathfinder().GetAbsPath(plugin.Config().Client().File())

	_, err := os.Stat(filePath)
	if err == nil {
		d, err := os.ReadFile(filePath)
		if err != nil {
			r.CollectErrorf("can not read file for plugin '%s': %v", plugin.Name(), err)
			code = "class Plugin extends lx.Plugin {}"
		} else {
			code = string(d)
			r.depFiles[filePath] = true
		}
	} else {
		if errors.Is(err, os.ErrNotExist) {
			code = "class Plugin extends lx.Plugin {}"
		} else {
			r.CollectErrorf("problem with client file for plugin '%s': %v", plugin.Name(), err)
			return
		}
	}

	hash := md5.New()
	io.WriteString(hash, plugin.Pathfinder().GetRoot())
	pluginKey := fmt.Sprintf("%x", hash.Sum(nil))

	initMethods := fmt.Sprintf("static getKey(){return '%s';}", pluginKey)
	core := plugin.Config().Client().Core()
	if core != "" {
		initMethods += fmt.Sprintf("getCoreClass(){return %s;}", core)
	}

	css := plugin.Config().CssAssets()
	if len(css) > 0 {
		cssList := strings.Join(css, ",")
		initMethods += fmt.Sprintf("static getCssAssetClasses(){return [%s];}", cssList)

		cssInitCall := "this.getCssAssetClasses().forEach(ac=>(new ac()).init(css));"
		re := regexp.MustCompile(`static\s+initCss\s*\(\s*css\s*\)\s*{`)
		if re.MatchString(code) {
			code = re.ReplaceAllStringFunc(code, func(match string) string {
				return match + cssInitCall
			})
		} else {
			initMethods += fmt.Sprintf("static initCss(css){%s}", cssInitCall)
		}
	}

	guiNodes := plugin.Config().Client().GuiNodes()
	if len(guiNodes) > 0 {
		guiNodesObj := make([]string, 0, len(guiNodes))
		for key, class := range guiNodes {
			guiNodesObj = append(guiNodesObj, fmt.Sprintf("%s:%s", key, class))
		}
		guiNodesObjStr := strings.Join(guiNodesObj, ",")
		initMethods += fmt.Sprintf("getGuiNodeClasses(){return {%s};}", guiNodesObjStr)
	}

	pattern := `class Plugin[^{]*{`
	re := regexp.MustCompile(pattern)
	loc := re.FindStringIndex(code)
	if loc == nil {
		r.CollectErrorf("plugin class does not exist for '%s'", plugin.Name())
		return
	}
	code = code[:loc[1]] + initMethods + code[loc[1]:]

	require := plugin.Config().Require()
	require = append(require, plugin.Config().Client().Require()...)
	if len(require) > 0 {
		requireStr := ""
		for _, req := range require {
			requireStr += fmt.Sprintf("@lx:require %s;\n", req)
		}
		code = requireStr + code
	}

	compiler := r.pp.CompilerBuilder().
		BuildModules(false).
		SetLang(r.lang).
		SetI18n(r.plugin.I18n()).
		SetClientContext().
		SetFilePath(filepath.Join(plugin.Path(), "_.js")).
		SetPathfinder(plugin.Pathfinder()).
		SetCode(code).
		SetUnwrapped().
		Compiler()
	pCode, err := compiler.Run()
	if err != nil {
		r.CollectErrorf("can not compile code for plugin '%s': %v", plugin.Name(), err)
		return
	}

	ff := compiler.CompiledFiles()
	for _, fn := range ff {
		r.depFiles[fn] = true
	}

	r.addAssets(compiler)

	pattern = `if\('([^']+)' in ([^)]+)\)return`
	re = regexp.MustCompile(pattern)
	sub := re.FindStringSubmatch(pCode)
	if len(sub) == 3 {
		loc = re.FindStringIndex(pCode)
		pCode = pCode[:loc[1]] + "}" + pCode[loc[1]:]
		pCode = pCode[:loc[1]-6] + "{__plugin__=new " + sub[2] + "." + sub[1] + "(config); " + pCode[loc[1]-6:]
	}

	pattern = `Plugin\.__afterDefinition\(\);`
	re = regexp.MustCompile(pattern)
	loc = re.FindStringIndex(pCode)
	pCode = pCode[:loc[1]] + "__plugin__=new Plugin(config);" + pCode[loc[1]:]

	r.output.Js = `(config)=>{let __plugin__=null;` + pCode + `return __plugin__;}`
}

func (r *pluginRenderer) preparePluginConf() {
	plugin := r.plugin

	r.conf.name = plugin.Name()

	imgs := plugin.Config().Images()
	al := newAssetLinker(r.pp, r.plugin.Pathfinder())
	r.conf.imagePaths = al.getAssetsMap(imgs)
}

func (r *pluginRenderer) compilePluginConf() {
	conf := make(map[string]any, 5)

	conf["name"] = r.conf.name
	conf["cssScope"] = r.conf.cssScope
	conf["imagePaths"] = r.conf.imagePaths
	conf["rsk"] = r.rootSnippetKey

	if len(r.conf.data) > 0 {
		conf["data"] = r.conf.data
	}

	if len(r.conf.onLoad) > 0 {
		conf["onLoad"] = r.conf.onLoad
	}

	r.output.Conf = conf
}

func (r *pluginRenderer) nestPlugins(list []nestedPluginConf) {
	for _, conf := range list {
		r.nestedConf = append(r.nestedConf, &conf)
	}
}

func (r *pluginRenderer) applyBuildData(rawConf map[string]any) {
	conf := struct {
		Title  string         `dict:"title"`
		Icon   string         `dict:"icon"`
		Data   map[string]any `dict:"data"`
		OnLoad []string       `dict:"onLoad"`
	}{}
	conv.MapToStruct(rawConf, &conf)

	if conf.Title != "" && r.title != "" {
		r.pp.LogError("Plugin %s already had title %s", r.plugin.Name(), r.title)
	}
	r.title = conf.Title
	if conf.Icon != "" && r.icon != "" {
		r.pp.LogError("Plugin %s already had icon %s", r.plugin.Name(), r.icon)
	}
	r.icon = conf.Icon

	if r.conf.data == nil {
		r.conf.data = conf.Data
	} else {
		maps.Copy(r.conf.data, conf.Data)
	}

	r.conf.onLoad = append(r.conf.onLoad, conf.OnLoad...)
}

func (r *pluginRenderer) addAssets(compiler jspp.ICompiler) {
	r.assets.Merge(compiler.Assets())
	for _, m := range compiler.CompiledModules() {
		r.assets.AddModule(m)
	}
}

func (r *pluginRenderer) getServerCode(data map[string]any) string {
	bData, err := json.Marshal(data)
	if err != nil {
		r.pp.LogError("error while plugin runtime data serialization for '%s': %v", r.plugin.Name(), err)
		return ""
	}
	pluginData := string(bData)

	blank := r.getServerCodeBlank()
	if blank == "" {
		return ""
	}

	return fmt.Sprintf(blank, pluginData)
}

func (r *pluginRenderer) getServerCodeBlank() string {
	if r.serverCodeBlank == "" {
		plugin := r.plugin

		path := plugin.Config().Server().File()
		if path == "" {
			return r.defaultServerCodeBlank()
		}

		var code string
		filePath := plugin.Pathfinder().GetAbsPath(plugin.Config().Server().File())

		_, err := os.Stat(filePath)
		if err == nil {
			d, err := os.ReadFile(filePath)
			if err != nil {
				r.CollectErrorf("can not read file for plugin '%s': %v", plugin.Name(), err)
				return r.defaultServerCodeBlank()
			}
			code = string(d)
		} else {
			if errors.Is(err, os.ErrNotExist) {
				return r.defaultServerCodeBlank()
			} else {
				r.CollectErrorf("problem with client file for plugin '%s': %v", plugin.Name(), err)
				return ""
			}
		}

		compiler := r.pp.CompilerBuilder().
			SetLang(r.lang).
			SetI18n(r.plugin.I18n()).
			SetServerContext().
			SetFilePath(filepath.Join(plugin.Path(), "_.js")).
			SetPathfinder(plugin.Pathfinder()).
			SetCode(code).
			SetUnwrapped().
			Compiler()
		pCode, err := compiler.Run()
		if err != nil {
			r.CollectErrorf("can not compile code for plugin '%s': %v", plugin.Name(), err)
			return ""
		}

		r.addAssets(compiler)

		pattern := `Plugin\.__afterDefinition\(\);`
		re := regexp.MustCompile(pattern)
		loc := re.FindStringIndex(pCode)
		r.serverCodeBlank = "@lx:use lx.Plugin; (()=>{" +
			pCode[:loc[1]] +
			"lx.globalContext.$plugin=new Plugin(%s);" +
			pCode[loc[1]:] +
			"})()"
	}

	return r.serverCodeBlank
}

func (r *pluginRenderer) defaultServerCodeBlank() string {
	if r.serverCodeBlank == "" {
		r.serverCodeBlank = `
			@lx:use lx.Plugin;
			const $plugin = new lx.Plugin(%s);
			lx.globalContext.$plugin = $plugin;
		`
	}
	return r.serverCodeBlank
}

func (r *pluginRenderer) eachContext(f func(pr *pluginRenderer)) {
	f(r)
	for _, nr := range r.nested {
		nr.eachContext(f)
	}
}
