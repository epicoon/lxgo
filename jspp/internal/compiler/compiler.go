package compiler

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/epicoon/lxgo/jspp"
	"github.com/epicoon/lxgo/jspp/internal/base"
	"github.com/epicoon/lxgo/jspp/internal/i18n"
	"github.com/epicoon/lxgo/jspp/internal/lxml"
	"github.com/epicoon/lxgo/kernel"
	"gopkg.in/yaml.v3"
)

const contextClient = "CLIENT"
const contextServer = "SERVER"

type Flags struct {
	Recursive bool
	Force     bool
	Unwrapped bool
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * Compiler
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */
type Compiler struct {
	config     *base.JSPreprocessorConfig
	pp         jspp.IPreprocessor
	app        kernel.IApp
	pathfinder kernel.IPathfinder

	isApp      bool
	context    string
	filePath   string
	prevCode   string
	inputCode  string
	postCode   string
	useModules []string
	flags      Flags

	lang        string
	i18n        jspp.II18nMap
	modulesI18n map[string]map[string]string

	buildModules    bool
	compiledFiles   []string
	ignoredModules  []string
	compiledModules []string
	modulesCode     string
	cleanCode       string
	assets          jspp.IAssets
}

/** @interface */
var _ jspp.ICompiler = (*Compiler)(nil)

/** @constructor */
func newCompiler() *Compiler {
	return &Compiler{
		isApp:           false,
		compiledModules: make([]string, 0),
		buildModules:    true,
		assets:          &Assets{},
	}
}

func (c *Compiler) Run() (string, error) {
	if c.filePath == "" && c.inputCode == "" {
		return "", errors.New("nothing to compile")
	}
	if c.inputCode == "" {
		return c.buildFile()
	}
	return c.buildCode()
}

func (c *Compiler) Pathfinder() kernel.IPathfinder {
	if c.pathfinder == nil {
		c.pathfinder = c.pp.Pathfinder()
	}
	return c.pathfinder
}

func (c *Compiler) Mode() string {
	return c.config.Mode
}

func (c *Compiler) CompiledModules() []string {
	return c.compiledModules
}

func (c *Compiler) ModulesCode() string {
	return c.modulesCode
}

func (c *Compiler) CleanCode() string {
	return c.cleanCode
}

func (c *Compiler) Assets() jspp.IAssets {
	return c.assets
}

func (c *Compiler) CompiledFiles() []string {
	return c.compiledFiles
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * PRIVATE
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

func (c *Compiler) buildFile() (string, error) {
	if err := c.getCode(); err != nil {
		return "", fmt.Errorf("can not get code from file '%s': %v", c.filePath, err)
	}
	c.noteFileCompiled(c.filePath)
	return c.buildCode()
}

func (c *Compiler) getCode() error {
	data, err := os.ReadFile(c.filePath)
	if err != nil {
		return err
	}
	c.inputCode = string(data)
	return nil
}

func (c *Compiler) buildCode() (string, error) {
	filePath := c.filePath
	code := c.prevCode + c.inputCode + c.postCode
	var err error

	appConf := c.parseAppConfig()

	code = c.applyUsedModules(code)

	code, err = c.compileCodeInnerDirectives(code, filePath)
	if err != nil {
		return "", err
	}
	code, err = c.compileCodeOuterDirectives(code, filePath, !c.flags.Unwrapped)
	if err != nil {
		return "", err
	}

	c.cleanCode = code
	code = c.modulesCode + c.getAppStart(appConf) + code

	code, err = c.applyI18n(code)
	if err != nil {
		return "", err
	}

	return code, nil
}

func (c *Compiler) applyUsedModules(code string) string {
	if len(c.useModules) == 0 {
		return code
	}
	return "lx.import(" + strings.Join(c.useModules, ",") + ");" + code
}

func (c *Compiler) compileCodeInnerDirectives(code, path string) (string, error) {
	var err error
	re := regexp.MustCompile(`// *?@lx:`)
	code = re.ReplaceAllString(code, "@lx:")

	code = c.processLxml(code)

	//TODO need leave comments for DEV ?
	code = c.cutComments(code)

	code = c.cutCoordinationDirectives(code)

	code = c.applyMacros(code)
	code = c.parseMd(code, path)

	code = c.applyContext(code)
	code = c.injectDatum(code)
	code, err = applyExtendedSyntax(code, path)
	if err != nil {
		return "", err
	}
	code = c.plugDependencies(code)
	code = c.applyMode(code)

	return code, nil
}

func (c *Compiler) compileCodeOuterDirectives(code, path string, wrapped bool) (string, error) {
	code = c.markDev(code, path)

	if wrapped {
		code = `(()=>{` + code + `})();`
	}

	code, err := c.processImport(code, path)
	if err != nil {
		return "", err
	}

	return code, nil
}

func (c *Compiler) markDev(code, path string) string {
	if c.Mode() != "DEV" || path == "" {
		return code
	}
	return fmt.Sprintf("\n/* @lx-begin-js-file: %s */\n%s\n/* @lx-end-js-file: %s */\n", path, code, path)
}

func (c *Compiler) markDevInterrupting(code, path string) string {
	if c.Mode() != "DEV" || path == "" {
		return code
	}
	return fmt.Sprintf("\n/* @lx-interrupted-js-file: %s */\n%s\n/* @lx-continue-js-file: %s */\n", path, code, path)
}

// processLxml finds lx.ml(`...`) calls and replaces them with the compiled
// LXML tree. A backtick inside the template can be escaped as \` (e.g. for a
// real nested JS template literal inside a raw (...) attribute). If the call
// is immediately preceded by `const NAME = ` / `let NAME = ` / `var NAME = `,
// that assignment is absorbed into the generated output (keeping whichever
// keyword the author used) instead of being left as a separate statement.
func (c *Compiler) processLxml(code string) string {
	const marker = "lx.ml("
	reAssign := regexp.MustCompile(`(?:^|[\s;{}])(const|let|var)\s+([A-Za-z_$][\w$]*)\s*=\s*$`)

	var out strings.Builder
	n := len(code)
	i := 0
	for i < n {
		rel := strings.Index(code[i:], marker)
		if rel == -1 {
			out.WriteString(code[i:])
			break
		}
		idx := i + rel

		if idx > 0 && isLxmlIdentByte(code[idx-1]) {
			out.WriteString(code[i : idx+1])
			i = idx + 1
			continue
		}

		afterMarker := idx + len(marker)
		j := afterMarker
		for j < n && isLxmlSpaceByte(code[j]) {
			j++
		}
		if j >= n || code[j] != '`' {
			out.WriteString(code[i:afterMarker])
			i = afterMarker
			continue
		}

		blockStart := j + 1
		closeIdx := findUnescapedBacktick(code, blockStart)
		if closeIdx == -1 {
			c.pp.LogError("lx.ml(): unterminated template literal")
			out.WriteString(code[i:afterMarker])
			i = afterMarker
			continue
		}

		p := closeIdx + 1
		for p < n && isLxmlSpaceByte(code[p]) {
			p++
		}
		if p >= n || code[p] != ')' {
			c.pp.LogError("lx.ml(): expected closing ')'")
			out.WriteString(code[i:afterMarker])
			i = afterMarker
			continue
		}
		callEnd := p + 1

		segment := code[i:idx]
		keyword := ""
		name := ""
		if loc := reAssign.FindStringSubmatchIndex(segment); loc != nil {
			keyword = segment[loc[2]:loc[3]]
			name = segment[loc[4]:loc[5]]
			segment = segment[:loc[2]]
		}
		out.WriteString(segment)

		ml := code[blockStart:closeIdx]
		ml = strings.ReplaceAll(ml, "\\`", "`")

		parser := lxml.NewParser(c.pp)
		parser.SetOutput(name)
		if name != "" {
			parser.SetOutputKeyword(keyword)
		}
		compiled, err := parser.ParseText(ml)
		if err != nil {
			c.pp.LogError(err.Error())
		} else {
			out.WriteString(compiled)
		}

		i = callEnd
	}

	return out.String()
}

func isLxmlIdentByte(b byte) bool {
	return b == '_' || b == '$' ||
		(b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}

func isLxmlSpaceByte(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

// findUnescapedBacktick returns the index (>= from) of the next backtick in
// code that is not preceded by an odd number of backslashes, or -1.
func findUnescapedBacktick(code string, from int) int {
	n := len(code)
	for k := from; k < n; k++ {
		if code[k] != '`' {
			continue
		}
		bs := 0
		m := k - 1
		for m >= from && code[m] == '\\' {
			bs++
			m--
		}
		if bs%2 == 0 {
			return k
		}
	}
	return -1
}

func (c *Compiler) cutComments(code string) string {
	// //...
	singleLine1 := regexp.MustCompile(`(\r|\n|\r\n)//.*?(?:\r|\n|\r\n)`)
	code = singleLine1.ReplaceAllString(code, "$1")
	singleLine2 := regexp.MustCompile(`(?:^| +?)//.*?(?:\r|\n|\r\n)`)
	code = singleLine2.ReplaceAllString(code, "")
	// /* ... */
	multiLine := regexp.MustCompile(`/\*[\s\S]*?\*/`)
	code = multiLine.ReplaceAllString(code, "")

	return code
}

func (c *Compiler) applyContext(code string) string {
	re := regexp.MustCompile(`@lx:<context +?(.+?): *?(?:\r|\n|\r\n)([\w\W]*?)@lx:context>`)
	return re.ReplaceAllStringFunc(code, func(match string) string {
		matches := re.FindStringSubmatch(match)
		if len(matches) < 3 {
			return match
		}
		if matches[1] == c.context {
			return matches[2]
		}
		return ""
	})
}

func (c *Compiler) plugDependencies(code string) string {
	// @lx:js path;
	code = c.plugDependency(code, "js", depTypeJS)

	// @lx:css path;
	code = c.plugDependency(code, "css", depTypeCSS)

	return code
}

func (c *Compiler) plugDependency(code, key, tp string) string {
	re := regexp.MustCompile(fmt.Sprintf("@lx:%s +([^;]+?);", key))
	matches := re.FindAllStringSubmatch(code, -1)
	for _, match := range matches {
		path := match[1]
		if !strings.HasPrefix(path, "http:") && !strings.HasPrefix(path, "https:") {
			path = c.Pathfinder().GetAbsPath(path)
		}
		addAsset(c.assets, path, tp)
	}
	return re.ReplaceAllString(code, "")
}

func (c *Compiler) applyMode(code string) string {
	re := regexp.MustCompile(`@lx:<mode +?(.+?): *?(?:\r|\n|\r\n)([\w\W]*?)@lx:mode>`)
	return re.ReplaceAllStringFunc(code, func(match string) string {
		matches := re.FindStringSubmatch(match)
		if len(matches) < 3 {
			return match
		}
		if matches[1] == c.Mode() {
			return matches[2]
		}
		return ""
	})
}

func (c *Compiler) cutCoordinationDirectives(code string) string {
	pattern := regexp.MustCompile(`@lx:module\s+([^;]+?);`)
	code = pattern.ReplaceAllString(code, "")

	pattern = regexp.MustCompile(`@lx:module-data:\s*([^;]+?);`)
	code = pattern.ReplaceAllString(code, "")

	return code
}

func (c *Compiler) injectDatum(code string) string {
	// val = lx.json('path');
	// val = lx.yaml('path');
	pattern := regexp.MustCompile(`lx\.(json|yaml)\s*\(\s*['"]?(.*?)['"]?\s*\)`)
	code = pattern.ReplaceAllStringFunc(code, func(match string) string {
		matches := pattern.FindStringSubmatch(match)
		if len(matches) < 3 {
			return "null"
		}

		key := matches[1]
		path := c.Pathfinder().GetAbsPath(matches[2])
		file, err := os.Open(path)
		if err != nil {
			c.pp.LogError("Can not open file %s: %v", path, err)
			return "null"
		}
		defer file.Close()

		var data any
		switch key {
		case "json":
			if err := json.NewDecoder(file).Decode(&data); err != nil {
				c.pp.LogError("Can not decode json file %s: %v", path, err)
				return "null"
			}
		case "yaml":
			if err := yaml.NewDecoder(file).Decode(&data); err != nil {
				c.pp.LogError("Can not decode yaml file %s: %v", path, err)
				return "null"
			}
		default:
			return "null"
		}

		res, err := json.Marshal(data)
		if err != nil {
			if err := json.NewDecoder(file).Decode(&data); err != nil {
				c.pp.LogError("Can not encode json file %s: %v", path, err)
				return "null"
			}
		}
		return string(res)
	})
	return code
}

func (c *Compiler) applyI18n(code string) (string, error) {
	lang := c.lang
	if lang == "" {
		lang = "en-EN"
	}

	algs := 0

	if c.modulesI18n != nil {
		algs++
		m := i18n.NewI18nMap(c.modulesI18n)
		code = m.Localize(code, lang)
	}

	if c.i18n != nil {
		algs++
		code = c.i18n.Localize(code, lang)
	}

	if algs < 2 {
		return clearI18n(code), nil
	} else {
		return code, nil
	}
}

func (c *Compiler) getAppStart(conf string) string {
	if !c.isApp {
		return ""
	}

	tail := ""
	if len(c.compiledModules) > 0 {
		mm := make([]string, 0, len(c.compiledModules))
		for _, m := range c.compiledModules {
			mm = append(mm, "'"+m+"'")
		}
		tail = fmt.Sprintf("lx.app.dependencies.noteModules([%s]);", strings.Join(mm, ","))
	}

	return fmt.Sprintf("lx.app.start(%s);", conf) + tail
}

func (c *Compiler) parseAppConfig() (res string) {
	if !c.isApp {
		return ""
	}

	res = "{}"
	conf := make(map[string]any)
	cPath := c.config.AppConfig
	defer func() {
		if mm, exists := conf["use"]; exists {
			delete(conf, "use")
			if use, ok := mm.([]any); ok {
				for _, m := range use {
					mStr, ok := m.(string)
					if !ok {
						c.pp.LogError("invalid format for 'use' element '%v' in JS-application config '%s': string require", m, cPath)
						continue
					}
					if !slices.Contains(c.useModules, mStr) {
						c.useModules = append(c.useModules, mStr)
					}
				}
			} else {
				c.pp.LogError("invalid format for 'use' in JS-application config '%s': []string require", cPath)
			}
		}

		if _, exists := conf["settings"]; !exists {
			conf["settings"] = make(map[string]any, 1)
		}
		settings := conf["settings"].(map[string]any)
		settings["csrs"] = c.config.CssScopeRenderSide

		bConf, err := json.Marshal(conf)
		if err != nil {
			c.pp.LogError("can not marshal to json JS-application config '%s': %v", cPath, err)
		} else {
			res = string(bConf)
			re := regexp.MustCompile(`@config\(([^)]+)\)`)
			res = re.ReplaceAllStringFunc(res, func(match string) string {
				matches := re.FindStringSubmatch(match)
				if len(matches) != 2 {
					return match
				}

				key := matches[1]
				val := c.app.ConfigParam(key)
				if val == nil {
					return "null"
				}

				sVal, ok := val.(string)
				if ok {
					return sVal
				}
				bVal, ok := val.(bool)
				if ok {
					if bVal {
						return "true"
					} else {
						return "false"
					}
				}
				iVal, ok := val.(int)
				if ok {
					return strconv.Itoa(iVal)
				}

				return match
			})
		}
	}()

	if cPath == "" {
		return
	}

	rawConf, err := os.ReadFile(cPath)
	if err != nil {
		if os.IsNotExist(err) {
			c.pp.LogError("JS-application config '%s' not found", cPath)
		} else {
			c.pp.LogError("can not read JS-application config '%s': %v", cPath, err)
		}
		return
	}

	if err := yaml.Unmarshal(rawConf, &conf); err != nil {
		c.pp.LogError("can not unmarshal yaml JS-application config '%s': %v", cPath, err)
		return
	}

	_, exists := conf["local"]
	if exists {
		localPath, ok := conf["local"].(string)
		if !ok {
			c.pp.LogError("invalid format for JS-application local config path: '%v'", conf["local"])
		} else {
			var path string
			switch localPath[0] {
			case '/':
				path = localPath
			case '@':
				path = c.pathfinder.GetAbsPath(localPath)
			default:
				dir := filepath.Dir(cPath)
				path = filepath.Join(dir, localPath)
			}
			rawConf, err := os.ReadFile(path)
			if err != nil {
				if os.IsNotExist(err) {
					c.pp.LogError("JS-application local config '%s' not found", path)
				} else {
					c.pp.LogError("can not read JS-application local config '%s': %v", path, err)
				}
				return
			}
			lConf := make(map[string]any)
			if err := yaml.Unmarshal(rawConf, &lConf); err != nil {
				c.pp.LogError("can not unmarshal yaml JS-application local config '%s': %v", cPath, err)
				return
			}
			mergeRecursive(conf, lConf)
		}
	}

	return
}

func clearI18n(code string) string {
	re := regexp.MustCompile(`lx\.i18n\(['"]?([\w\d_\-.]+)['"]?\)`)
	code = re.ReplaceAllStringFunc(code, func(match string) string {
		matches := re.FindStringSubmatch(match)
		if len(matches) < 2 {
			return match
		}
		key := matches[1]
		re := regexp.MustCompile(`module\-[^\-]+\-`)
		key = re.ReplaceAllString(key, "")
		return "'" + key + "'"
	})
	return code
}

func deepCopyMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		if sm, ok := v.(map[string]any); ok {
			out[k] = deepCopyMap(sm)
		} else {
			out[k] = v
		}
	}
	return out
}

func mergeRecursive(dst, src map[string]any) {
	if dst == nil || src == nil {
		return
	}

	for key, srcVal := range src {
		if dstVal, ok := dst[key]; ok {
			dstMap, okDst := dstVal.(map[string]any)
			srcMap, okSrc := srcVal.(map[string]any)
			if okDst && okSrc {
				if dstMap == nil {
					dstMap = make(map[string]any)
					dst[key] = dstMap
				}
				mergeRecursive(dstMap, srcMap)
				continue
			}
		}

		if sm, ok := srcVal.(map[string]any); ok {
			dst[key] = deepCopyMap(sm)
		} else {
			dst[key] = srcVal
		}
	}
}
