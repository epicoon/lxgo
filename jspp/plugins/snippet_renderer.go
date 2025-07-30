package plugins

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/epicoon/lxgo/jspp"
	"github.com/epicoon/lxgo/kernel/conv"
)

type snippetRenderer struct {
	pp     jspp.IPreprocessor
	pr     *pluginRenderer
	plugin jspp.IPlugin
	nested []*snippetRenderer

	snippet *snippet
	path    string

	html   string
	output map[string]*snippetConf
}

func newSnippetRenderer(pr *pluginRenderer, hash, path string, params map[string]any) *snippetRenderer {
	return &snippetRenderer{
		pp:      pr.pp,
		pr:      pr,
		plugin:  pr.plugin,
		path:    path,
		snippet: newSnippet(hash, params),
	}
}

func (r *snippetRenderer) run() {
	plugin := r.plugin

	cacheType := plugin.Config().CacheType()
	switch cacheType {
	// 	case PluginCacheManager::CACHE_BUILD:
	// 		return $this->buildProcess(true);
	case CACHE_NONE:
		r.buildProcess(false)
		return

		// 	case PluginCacheManager::CACHE_STRICT:
		// 		$result = $this->getCache();
		// 		if (!$result) {
		// 			\lx::devLog(['_' => [__FILE__, __CLASS__, __METHOD__, __LINE__],
		// 				'__trace__' => debug_backtrace(
		// 					DEBUG_BACKTRACE_PROVIDE_OBJECT & DEBUG_BACKTRACE_IGNORE_ARGS
		// 				),
		// 				'msg' => 'There is the strict cache option for the plugin without cache',
		// 				'plugin' => $this->getPlugin()->name,
		// 				'snippet' => $this->snippet->getFile()->getName(),
		// 			]);
		// 			$result = '[]';
		// 		}
		// 		return $result;
		// 	case PluginCacheManager::CACHE_ON:
		// 		return $this->getCache() ?? $this->buildProcess(true);
		// 	case PluginCacheManager::CACHE_SMART:
		// 		return $this->getSmartCache();
	}
}

func (r *snippetRenderer) buildProcess(renewCache bool) {
	if !r.runSnippetCode() {
		return
	}

	r.output = make(map[string]*snippetConf, 1)
	collectSnippets(r, r.output, &r.html)

	//TODO
	_ = renewCache
	// jsonData, err := json.Marshal(snippetsData)
	// if err != nil {
	// 	r.jspp.LogError("can not serialize snippets data for plugin %s: %v", r.plugin.Name(), err)
	// 	return "[]"
	// }
	// cache := string(jsonData)
	// if renewCache {
	// 	// 	$this->cacheData->renew(
	// 	// 		$this->getKey(),
	// 	// 		$snippets,
	// 	// 		$snippetsData,
	// 	// 		$cache
	// 	// 	);
	// }
}

func (sr *snippetRenderer) runSnippetCode() bool {
	plugin := sr.plugin
	pr := sr.pr

	// Prepare plugin data
	pData := sr.getPluginData()
	bData, err := json.Marshal(pData)
	if err != nil {
		sr.pp.LogError("error while plugin runtime data serialization for '%s': %v", plugin.Name(), err)
		return false
	}
	pluginData := string(bData)

	// Prepare snippet data
	sDataStruct := struct {
		Params map[string]any `json:"params"`
	}{
		Params: sr.snippet.params,
	}
	sData, err := json.Marshal(sDataStruct)
	if err != nil {
		sr.pp.LogError("error white snippet '%s' data serialization for plugin '%s': %v", sr.path, plugin.Name(), err)
		return false
	}
	snippetData := string(sData)

	// Prepare code edges
	prev := fmt.Sprintf(`
		@lx:use lx.Plugin;
		const $plugin = new lx.Plugin(%s);
		lx.globalContext.$plugin = $plugin;
		const $snippet = new lx.Snippet(%s);
		lx.globalContext.$snippet = $snippet;
		lx.app.start({
			root: $snippet.widget,
		});
	`, pluginData, snippetData)
	post := `
		return {
			app: lx.app.getResult(),
			plugin: $plugin.getResult(),
			snippet: $snippet.getResult(),
			subPlugins: $snippet.getSubPlugins(),
			dependencies: {}
				.lxMerge(lx.app.getDependencies())
				.lxMerge($plugin.getDependencies())
		};
	`
	res := struct {
		Plugin     map[string]any     `dict:"plugin"`
		Snippet    map[string]any     `dict:"snippet"`
		SubPlugins []nestedPluginConf `dict:"subPlugins"`
	}{}

	compiler := pr.pp.CompilerBuilder().
		SetLang(pr.lang).
		SetI18n(pr.plugin.I18n()).
		SetServerContext().
		SetUnwrapped().
		SetPrevCode(prev).
		SetPostCode(post).
		SetFilePath(sr.path).
		SetPathfinder(plugin.Pathfinder()).
		Compiler()
	code, err := compiler.Run()
	if err != nil {
		sr.pp.LogError("can not compile snippet code '%s': %s", sr.path, err)
		return false
	}
	executor := sr.pr.pp.ExecutorBuilder().
		SetCode(code).
		Executor()
	rawRes, err := executor.Exec()
	if err != nil {
		sr.pp.LogError("can not execute snippet code '%s': %s", sr.path, err)
		return false
	}
	if rawRes.Fatal() != "" {
		sr.pp.LogError("can not execute snippet code '%s':\n%s", sr.path, rawRes.Fatal())
		return false
	}

	conv.MapToStruct(rawRes.Result().(map[string]any), &res)
	sr.fillSnippet(res.Snippet)
	pr.addAssets(compiler)
	pr.applyBuildData(res.Plugin)

	//TODO
	// // Зависимости сниппету запомнить для кэша?
	// $snippet->setDependencies($dependencies, $compiler->getCompiledFiles());

	// Process nested plugins
	pr.nestPlugins(res.SubPlugins)

	// Build contexts tree with nested snippets
	for _, innSR := range sr.nested {
		innSR.runSnippetCode()
	}

	return true
}

func (sr *snippetRenderer) getPluginData() map[string]any {
	data := make(map[string]any, 5)
	conf := sr.pr.conf

	data["name"] = conf.name
	data["cssScope"] = conf.cssScope
	data["imagePaths"] = conf.imagePaths
	data["params"] = conf.params

	return data
}

func (sr *snippetRenderer) fillSnippet(data map[string]any) {
	sr.snippet.html = data["html"].(string)
	sr.snippet.clientData.JS = data["js"].(string)

	sData, ok := data["data"].(map[string]any)
	if ok && len(sData) > 0 {
		sr.snippet.clientData.Data = sData
	}

	self := data["selfData"].(map[string]any)
	if len(self) > 0 {
		sr.snippet.clientData.Self = self
	}

	lx := data["lx"].([]any)
	if len(lx) > 0 {
		sr.snippet.clientData.Lx = lx
	}

	for i, rawElemData := range sr.snippet.clientData.Lx {
		elemData := rawElemData.(map[string]any)
		rawSnippetInfo, exists := elemData["snippetInfo"]
		if !exists {
			continue
		}

		delete(elemData, "snippetInfo")

		inf := struct {
			Hash   string         `dict:"hash"`
			Path   string         `dict:"path"`
			Params map[string]any `dict:"params"`
		}{}
		if err := conv.MapToStruct(rawSnippetInfo.(map[string]any), &inf); err != nil {
			sr.pp.LogError("wrong snippet settings '%v' for plugin '%s': %v", rawSnippetInfo, sr.plugin.Name(), err)
		}

		if inf.Path == "" {
			sr.pp.LogError("wrong snippet path for plugin '%s'", sr.plugin.Name())
			continue
		}

		sr.addSnippet(inf.Hash, inf.Path, inf.Params)
		sr.snippet.clientData.Lx[i] = elemData
	}
}

func (sr *snippetRenderer) addSnippet(hash, path string, params map[string]any) *snippetRenderer {
	paths := make([]string, 0)
	file := path
	if !strings.HasSuffix(file, ".js") {
		file += ".js"
	}

	// Check config snippets
	fromSnippetsArr := make([]string, 0)
	snippets := sr.plugin.Config().Server().Snippets()
	for _, dir := range snippets {
		fullPath := sr.plugin.Pathfinder().GetAbsPath(filepath.Join(dir, file))
		if _, err := os.Stat(fullPath); err == nil {
			fromSnippetsArr = append(fromSnippetsArr, fullPath)
		} else if !os.IsNotExist(err) {
			sr.pp.LogError("problem with snippet '%s.%s', file '%s': %v", sr.plugin.Name(), path, fullPath, err)
		}
	}
	if len(fromSnippetsArr) > 1 {
		sr.pp.LogError("there are several '%s' snippet files: %v for plugin %s", path, fromSnippetsArr, sr.plugin.Name())
		return nil
	}
	fromSnippets := ""
	if len(fromSnippetsArr) == 1 {
		fromSnippets = fromSnippetsArr[0]
	}
	if fromSnippets != "" {
		if !slices.Contains(paths, fromSnippets) {
			paths = append(paths, fromSnippets)
		}
	}

	// Check config snippets map
	fromSnippetsMap := sr.plugin.Config().Server().SnippetsMap()[path]
	if fromSnippetsMap != "" {
		if !slices.Contains(paths, fromSnippetsMap) {
			paths = append(paths, fromSnippetsMap)
		}
	}

	// Check relative file
	fromRelative := ""
	dir := filepath.Dir(sr.path)
	fullPath := filepath.Join(dir, file)
	if _, err := os.Stat(fullPath); err == nil {
		fromRelative = fullPath
	} else if !os.IsNotExist(err) {
		sr.pp.LogError("problem with snippet '%s.%s', file '%s': %v", sr.plugin.Name(), path, fullPath, err)
	}
	if fromRelative != "" {
		if !slices.Contains(paths, fromRelative) {
			paths = append(paths, fromRelative)
		}
	}

	if len(paths) == 0 {
		sr.pp.LogError("snippet '%s.%s' not found", sr.plugin.Name(), path)
		return nil
	} else if len(paths) > 1 {
		sr.pp.LogError("there are several paths for snippet '%s.%s': %v", sr.plugin.Name(), path, paths)
		return nil
	}

	snippetPath := paths[0]

	newSR := newSnippetRenderer(sr.pr, hash, snippetPath, params)
	sr.nested = append(sr.nested, newSR)

	return newSR
}

func collectSnippets(sr *snippetRenderer, list map[string]*snippetConf, html *string) {
	list[sr.snippet.key] = sr.snippet.clientData

	if *html == "" {
		*html = sr.snippet.html
	} else {
		re := regexp.MustCompile(fmt.Sprintf(`lx-snippet="%s"[^>]*?>`, sr.snippet.key))
		*html = re.ReplaceAllStringFunc(*html, func(match string) string {
			return match + sr.snippet.html
		})
	}

	for _, innSR := range sr.nested {
		collectSnippets(innSR, list, html)
	}
}
