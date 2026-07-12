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

func (c *pluginCache) Save() {
	langRoot := c.langRoot()

	_ = os.MkdirAll(langRoot, os.ModePerm)

	// Serialize
	snippetsBytes, _ := json.Marshal(c.r.output.Snippets)
	nestedBytes, _ := json.Marshal(c.r.nestedConf)

	// Data hash
	hashMd5 := md5.New()
	io.WriteString(hashMd5,
		c.r.output.Js+
			string(snippetsBytes)+
			string(nestedBytes),
	)
	dataHash := fmt.Sprintf("%x", hashMd5.Sum(nil))

	cacheRoot := filepath.Join(langRoot, dataHash)

	_ = os.RemoveAll(cacheRoot)
	_ = os.MkdirAll(cacheRoot, os.ModePerm)

	// Params hash
	paramsHash := c.paramsHash()

	// Map
	cacheMap, _ := c.readMap()
	cacheMap[paramsHash] = dataHash
	c.writeMap(cacheMap)

	// Deps
	deps := c.collectDeps()
	depsBytes, _ := json.MarshalIndent(deps, "", " ")

	// Note root snippet key
	js := c.r.rootSnippetKey + ":::" + c.r.output.Js

	// Save files
	_ = os.WriteFile(filepath.Join(cacheRoot, "js"), []byte(js), 0644)
	_ = os.WriteFile(filepath.Join(cacheRoot, "html"), []byte(c.r.html), 0644)
	_ = os.WriteFile(filepath.Join(cacheRoot, "snippets"), snippetsBytes, 0644)
	_ = os.WriteFile(filepath.Join(cacheRoot, "nested"), nestedBytes, 0644)
	_ = os.WriteFile(filepath.Join(cacheRoot, "deps.json"), depsBytes, 0644)
}

func (c *pluginCache) Load() {
	cacheMap, err := c.readMap()
	if err != nil {
		return
	}

	paramsHash := c.paramsHash()
	dataHash, ok := cacheMap[paramsHash]
	if !ok {
		return
	}

	langRoot := c.langRoot()
	cacheRoot := filepath.Join(langRoot, dataHash)

	js, err0 := os.ReadFile(filepath.Join(cacheRoot, "js"))
	html, err1 := os.ReadFile(filepath.Join(cacheRoot, "html"))
	snippets, err2 := os.ReadFile(filepath.Join(cacheRoot, "snippets"))
	nested, err3 := os.ReadFile(filepath.Join(cacheRoot, "nested"))

	if err0 != nil || err1 != nil || err2 != nil || err3 != nil {
		return
	}

	jsData := string(js)
	parts := strings.SplitN(jsData, ":::", 2)
	c.r.rootSnippetKey = parts[0]
	c.r.output.Js = parts[1]

	c.r.html = string(html)

	_ = json.Unmarshal(snippets, &c.r.output.Snippets)
	_ = json.Unmarshal(nested, &c.r.nestedConf)
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
		return result, err
	}

	err = json.Unmarshal(data, &result)
	return result, err
}

func (c *pluginCache) writeMap(m map[string]string) {
	data, _ := json.MarshalIndent(m, "", " ")
	_ = os.WriteFile(c.mapPath(), data, 0644)
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
