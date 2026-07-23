package plugins

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/epicoon/lxgo/jspp"
)

const (
	CACHE_OFF     = "off"
	CACHE_ON      = "on"
	CACHE_DEV     = "dev"
	CACHE_INHERIT = "inherit"
)

type pluginCache struct {
	r *pluginRenderer
}

func newPluginCache(r *pluginRenderer) *pluginCache {
	return &pluginCache{r: r}
}

func (c *pluginCache) Type() string {
	tp := c.r.plugin.Config().CacheType()
	if tp == CACHE_INHERIT {
		return c.r.pp.Config().PluginCacheType
	}
	return tp
}

func (c *pluginCache) Exists() bool {
	langRoot := c.langRoot()

	paramsHash := c.paramsHash()

	cacheMap, err := c.readMap()
	if err != nil {
		return false
	}

	dataHash, ok := cacheMap[paramsHash]
	if !ok {
		return false
	}

	cacheRoot := filepath.Join(langRoot, dataHash)

	if _, err := os.Stat(cacheRoot); err != nil {
		return false
	}

	return true
}

func (c *pluginCache) DepsChanged() bool {
	cacheMap, err := c.readMap()
	if err != nil {
		return true
	}

	paramsHash := c.paramsHash()
	dataHash, ok := cacheMap[paramsHash]
	if !ok {
		return true
	}

	langRoot := c.langRoot()
	cacheRoot := filepath.Join(langRoot, dataHash)

	// Cache created time
	cacheInfo, err := os.Stat(cacheRoot)
	if err != nil {
		return true
	}
	cacheTime := cacheInfo.ModTime()

	// Read file dependencied list
	depsPath := filepath.Join(cacheRoot, "deps.json")
	data, err := os.ReadFile(depsPath)
	if err != nil {
		return true
	}

	var deps []string
	if err := json.Unmarshal(data, &deps); err != nil {
		return true
	}

	for _, dep := range deps {
		info, err := os.Stat(dep)
		if err != nil {
			// File is not exist → dependencies are changed
			return true
		}

		// File is newer than cache → cache is deprecated
		if info.ModTime().After(cacheTime) {
			return true
		}
	}

	return false
}

func (c *pluginCache) Save() error {
	langRoot := c.langRoot()

	if err := os.MkdirAll(langRoot, os.ModePerm); err != nil {
		return fmt.Errorf("can not create cache dir '%s': %w", langRoot, err)
	}

	// Serialize
	snippetsBytes, err := json.Marshal(c.r.output.Snippets)
	if err != nil {
		return fmt.Errorf("can not marshal snippets for plugin '%s': %w", c.r.plugin.Name(), err)
	}
	nestedBytes, err := json.Marshal(c.r.nestedConf)
	if err != nil {
		return fmt.Errorf("can not marshal nested plugins config for plugin '%s': %w", c.r.plugin.Name(), err)
	}
	assetsBytes, err := json.Marshal(cachedAssetsFrom(c.r.assets))
	if err != nil {
		return fmt.Errorf("can not marshal assets for plugin '%s': %w", c.r.plugin.Name(), err)
	}

	// Data hash
	hashMd5 := md5.New()
	io.WriteString(hashMd5,
		c.r.output.Js+
			string(snippetsBytes)+
			string(nestedBytes),
	)
	dataHash := fmt.Sprintf("%x", hashMd5.Sum(nil))

	cacheRoot := filepath.Join(langRoot, dataHash)

	if err := os.RemoveAll(cacheRoot); err != nil {
		return fmt.Errorf("can not clear cache dir '%s': %w", cacheRoot, err)
	}
	if err := os.MkdirAll(cacheRoot, os.ModePerm); err != nil {
		return fmt.Errorf("can not create cache dir '%s': %w", cacheRoot, err)
	}

	// Params hash
	paramsHash := c.paramsHash()

	// Map
	cacheMap, err := c.readMap()
	if err != nil {
		return fmt.Errorf("can not read cache map for plugin '%s': %w", c.r.plugin.Name(), err)
	}
	cacheMap[paramsHash] = dataHash
	if err := c.writeMap(cacheMap); err != nil {
		return fmt.Errorf("can not write cache map for plugin '%s': %w", c.r.plugin.Name(), err)
	}

	// Deps
	deps := c.collectDeps()
	depsBytes, err := json.MarshalIndent(deps, "", " ")
	if err != nil {
		return fmt.Errorf("can not marshal deps for plugin '%s': %w", c.r.plugin.Name(), err)
	}

	// Note root snippet key
	js := c.r.rootSnippetKey + ":::" + c.r.output.Js

	// Save files
	if err := os.WriteFile(filepath.Join(cacheRoot, "js"), []byte(js), 0644); err != nil {
		return fmt.Errorf("can not write cache file 'js' for plugin '%s': %w", c.r.plugin.Name(), err)
	}
	if err := os.WriteFile(filepath.Join(cacheRoot, "html"), []byte(c.r.html), 0644); err != nil {
		return fmt.Errorf("can not write cache file 'html' for plugin '%s': %w", c.r.plugin.Name(), err)
	}
	if err := os.WriteFile(filepath.Join(cacheRoot, "snippets"), snippetsBytes, 0644); err != nil {
		return fmt.Errorf("can not write cache file 'snippets' for plugin '%s': %w", c.r.plugin.Name(), err)
	}
	if err := os.WriteFile(filepath.Join(cacheRoot, "nested"), nestedBytes, 0644); err != nil {
		return fmt.Errorf("can not write cache file 'nested' for plugin '%s': %w", c.r.plugin.Name(), err)
	}
	if err := os.WriteFile(filepath.Join(cacheRoot, "deps.json"), depsBytes, 0644); err != nil {
		return fmt.Errorf("can not write cache file 'deps.json' for plugin '%s': %w", c.r.plugin.Name(), err)
	}
	if err := os.WriteFile(filepath.Join(cacheRoot, "assets.json"), assetsBytes, 0644); err != nil {
		return fmt.Errorf("can not write cache file 'assets.json' for plugin '%s': %w", c.r.plugin.Name(), err)
	}

	return nil
}

func (c *pluginCache) Load() error {
	cacheMap, err := c.readMap()
	if err != nil {
		return fmt.Errorf("can not read cache map for plugin '%s': %w", c.r.plugin.Name(), err)
	}

	paramsHash := c.paramsHash()
	dataHash, ok := cacheMap[paramsHash]
	if !ok {
		return fmt.Errorf("cache miss for plugin '%s': no entry for params hash '%s'", c.r.plugin.Name(), paramsHash)
	}

	langRoot := c.langRoot()
	cacheRoot := filepath.Join(langRoot, dataHash)

	js, err := os.ReadFile(filepath.Join(cacheRoot, "js"))
	if err != nil {
		return fmt.Errorf("can not read cache file 'js' for plugin '%s': %w", c.r.plugin.Name(), err)
	}
	html, err := os.ReadFile(filepath.Join(cacheRoot, "html"))
	if err != nil {
		return fmt.Errorf("can not read cache file 'html' for plugin '%s': %w", c.r.plugin.Name(), err)
	}
	snippets, err := os.ReadFile(filepath.Join(cacheRoot, "snippets"))
	if err != nil {
		return fmt.Errorf("can not read cache file 'snippets' for plugin '%s': %w", c.r.plugin.Name(), err)
	}
	nested, err := os.ReadFile(filepath.Join(cacheRoot, "nested"))
	if err != nil {
		return fmt.Errorf("can not read cache file 'nested' for plugin '%s': %w", c.r.plugin.Name(), err)
	}
	assetsData, err := os.ReadFile(filepath.Join(cacheRoot, "assets.json"))
	if err != nil {
		return fmt.Errorf("can not read cache file 'assets.json' for plugin '%s': %w", c.r.plugin.Name(), err)
	}

	jsData := string(js)
	parts := strings.SplitN(jsData, ":::", 2)
	if len(parts) != 2 {
		return fmt.Errorf("corrupted cache file 'js' for plugin '%s': missing ':::' separator", c.r.plugin.Name())
	}

	var snippetsConf map[string]*snippetConf
	if err := json.Unmarshal(snippets, &snippetsConf); err != nil {
		return fmt.Errorf("can not parse cache file 'snippets' for plugin '%s': %w", c.r.plugin.Name(), err)
	}
	var nestedConf []*nestedPluginConf
	if err := json.Unmarshal(nested, &nestedConf); err != nil {
		return fmt.Errorf("can not parse cache file 'nested' for plugin '%s': %w", c.r.plugin.Name(), err)
	}
	var assetsConf []cachedAsset
	if err := json.Unmarshal(assetsData, &assetsConf); err != nil {
		return fmt.Errorf("can not parse cache file 'assets.json' for plugin '%s': %w", c.r.plugin.Name(), err)
	}

	c.r.rootSnippetKey = parts[0]
	c.r.output.Js = parts[1]
	c.r.html = string(html)
	c.r.output.Snippets = snippetsConf
	c.r.nestedConf = nestedConf
	applyCachedAssets(c.r.assets, assetsConf)

	return nil
}

func (c *pluginCache) langRoot() string {
	path := c.r.pp.Config().PluginsPath

	root := filepath.Join(path, "lx_cache")
	pluginRoot := filepath.Join(root, c.r.plugin.Name())

	return filepath.Join(pluginRoot, c.r.lang)
}

func (c *pluginCache) paramsHash() string {
	str := c.buildParamsString()

	hash := md5.New()
	io.WriteString(hash, str)

	return fmt.Sprintf("%x", hash.Sum(nil))
}

func (c *pluginCache) buildParamsString() string {
	var parts []string

	parts = append(parts, c.r.conf.cssScope)

	keys := make([]string, 0, len(c.r.conf.params))
	for k := range c.r.conf.params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		vBytes, _ := json.Marshal(c.r.conf.params[k])
		parts = append(parts, k+":"+string(vBytes))
	}

	return strings.Join(parts, ",")
}

func (c *pluginCache) mapPath() string {
	return filepath.Join(c.langRoot(), "_map.json")
}

func (c *pluginCache) readMap() (map[string]string, error) {
	path := c.mapPath()

	result := map[string]string{}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return result, nil
		}
		return nil, fmt.Errorf("can not read cache map file '%s': %w", path, err)
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("can not parse cache map file '%s': %w", path, err)
	}
	return result, nil
}

func (c *pluginCache) writeMap(m map[string]string) error {
	data, err := json.MarshalIndent(m, "", " ")
	if err != nil {
		return fmt.Errorf("can not marshal cache map: %w", err)
	}
	if err := os.WriteFile(c.mapPath(), data, 0644); err != nil {
		return fmt.Errorf("can not write cache map file '%s': %w", c.mapPath(), err)
	}
	return nil
}

func (c *pluginCache) collectDeps() []string {
	deps := make([]string, len(c.r.depFiles))
	i := 0
	for file := range c.r.depFiles {
		deps[i] = file
		i++
	}
	return deps
}

// cachedAsset is the on-disk JSON shape for one jspp.IAsset entry - Asset's
// own fields aren't exported, so this package keeps its own serializable
// mirror, round-tripped only through the public IAssets methods.
type cachedAsset struct {
	Path string `json:"path"`
	Type string `json:"type"`
}

const (
	cachedAssetJS     = "js"
	cachedAssetCSS    = "css"
	cachedAssetModule = "module"
)

func cachedAssetsFrom(assets jspp.IAssets) []cachedAsset {
	all := assets.All()
	res := make([]cachedAsset, 0, len(all))
	for _, a := range all {
		var tp string
		switch {
		case a.IsJS():
			tp = cachedAssetJS
		case a.IsCSS():
			tp = cachedAssetCSS
		case a.IsModule():
			tp = cachedAssetModule
		default:
			continue
		}
		res = append(res, cachedAsset{Path: a.Path(), Type: tp})
	}
	return res
}

func applyCachedAssets(assets jspp.IAssets, cached []cachedAsset) {
	for _, a := range cached {
		switch a.Type {
		case cachedAssetJS:
			assets.AddJS(a.Path)
		case cachedAssetCSS:
			assets.AddCSS(a.Path)
		case cachedAssetModule:
			assets.AddModule(a.Path)
		}
	}
}
