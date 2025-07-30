package compiler

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/epicoon/lxgo/jspp"
	"github.com/epicoon/lxgo/jspp/internal/base"
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

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * PRIVATE
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

func (c *Compiler) buildFile() (string, error) {
	if err := c.getCode(); err != nil {
		return "", fmt.Errorf("can not get code from file '%s': %v", c.filePath, err)
	}
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
	code, err = c.compileExtensions(code, filePath)
	if err != nil {
		return "", err
	}

	return code, nil
}

func (c *Compiler) applyUsedModules(code string) string {
	mm := ""
	for _, m := range c.useModules {
		mm += "@lx:use " + m + ";"
	}
	return mm + code
}

func (c *Compiler) compileCodeInnerDirectives(code, path string) (string, error) {
	var err error
	re := regexp.MustCompile(`// *?@lx:`)
	code = re.ReplaceAllString(code, "@lx:")

	//TODO
	// $extensions = $this->getExtensions();
	// foreach ($extensions as $extension) {
	// 	$extension->setConductor($this->conductor);
	// 	$code = $extension->beforeCutComments($code, $path);
	// }

	code = c.processLxml(code)

	//TODO need leave comments for DEV ?
	code = c.cutComments(code)

	//TODO
	// foreach ($extensions as $extension) {
	// 	$code = $extension->afterCutComments($code, $path);
	// }

	code = c.cutCoordinationDirectives(code)

	//TODO
	// $code = $this->parseMd($code, $path);
	// $code = $this->processMacroses($code);

	code = c.applyContext(code)
	code = c.injectDatum(code)
	code, err = applyExtendedSyntax(c, code, path)
	if err != nil {
		return "", err
	}
	code = c.plugDependencies(code)
	code = c.applyMode(code)

	return code, nil
}

func (c *Compiler) compileCodeOuterDirectives(code, path string, wrapped bool) (string, error) {
	//TODO
	// $code = $this->markDev($code, $path);

	if wrapped {
		code = `(()=>{` + code + `})();`
	}

	code, err := c.plugAllRequires(code, path)
	if err != nil {
		return "", err
	}

	code, err = c.plugAllModules(code, path)
	if err != nil {
		return "", err
	}

	return code, nil
}

func (c *Compiler) compileExtensions(code, path string) (string, error) {
	//TODO
	_ = path

	return code, nil
}

func (c *Compiler) processLxml(code string) string {
	re := regexp.MustCompile(`(?:/\*\s*)?@lx:<ml(?::|\s+([\w\d_]+?)?:)([\w\W]+?)@lx:ml>(?:\s*\*/)?`)
	return re.ReplaceAllStringFunc(code, func(s string) string {
		match := re.FindStringSubmatch(s)
		if len(match) != 3 {
			return ""
		}

		out := match[1]
		ml := match[2]

		parser := lxml.NewParser(c.pp)
		parser.SetOutput(out)
		code, err := parser.ParseText(ml)
		if err != nil {
			c.pp.LogError(err.Error())
			return ""
		}

		return code
	})
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
	// val = lx(json, 'path');
	// val = lx(yaml, 'path');
	pattern := regexp.MustCompile(`lx\( *(json|yaml) *, *'([^']+?)'\)`)
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

		modulesMap, modulesOk := c.modulesI18n[lang]
		re := regexp.MustCompile(`lx\(i18n\)\.(module\-[^\-]+\-([\w\d_\-.]+))`)
		code = re.ReplaceAllStringFunc(code, func(match string) string {
			matches := re.FindStringSubmatch(match)
			if len(matches) < 3 {
				return match
			}

			key := matches[1]
			var tr string
			if modulesOk {
				tr = modulesMap[key]
			}
			if tr != "" {
				return "'" + tr + "'"
			}

			return "'" + matches[2] + "'"
		})
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
						c.pp.LogError("invaliv format for 'use' element '%v' in application config '%s': string require", m, cPath)
						continue
					}
					if !slices.Contains(c.useModules, mStr) {
						c.useModules = append(c.useModules, mStr)
					}
				}
			} else {
				c.pp.LogError("invaliv format for 'use' in application config '%s': []string require", cPath)
			}
		}

		if _, exists := conf["settings"]; !exists {
			conf["settings"] = make(map[string]any, 1)
		}
		settings := conf["settings"].(map[string]any)
		settings["csrs"] = c.config.CssScopeRenderSide

		bConf, err := json.Marshal(conf)
		if err != nil {
			c.pp.LogError("can not marshal to json application config '%s':", cPath, err)
		} else {
			res = string(bConf)
		}
	}()

	if cPath == "" {
		return
	}

	rawConf, err := os.ReadFile(cPath)
	if err != nil {
		if os.IsNotExist(err) {
			c.pp.LogError("application config '%s' not found", cPath)
		} else {
			c.pp.LogError("can not read application config '%s': %v", cPath, err)
		}
		return
	}

	if err := yaml.Unmarshal(rawConf, &conf); err != nil {
		c.pp.LogError("can not unmarshal yaml application config '%s':", cPath, err)
		return
	}

	bConf, err := json.Marshal(conf)
	if err != nil {
		c.pp.LogError("can not marshal to json application config '%s':", cPath, err)
		return
	}

	res = string(bConf)
	return
}

func clearI18n(code string) string {
	re := regexp.MustCompile(`lx\(i18n\)\.([\w\d_\-.]+)`)
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
