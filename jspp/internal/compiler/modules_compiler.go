package compiler

import (
	"os"
	"path/filepath"
	"slices"

	"github.com/epicoon/lxgo/jspp"
	"gopkg.in/yaml.v3"
)

func (c *Compiler) checkModule(moduleName string, modulesForBuild *[]string, filePaths *[]string) {
	if resolved, ok := c.pp.Config().ModuleInjector[moduleName]; ok {
		moduleName = resolved
	}

	if slices.Contains(c.ignoredModules, moduleName) ||
		slices.Contains(c.compiledModules, moduleName) ||
		slices.Contains(*modulesForBuild, moduleName) {
		return
	}

	mData := c.pp.ModulesMap().Get(moduleName)

	if mData == nil {
		c.pp.LogError("Module '%s' does not exist", moduleName)
		return
	}
	path := mData.Path()
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			c.pp.LogError("File for module '%s' does not exist", moduleName)
		} else {
			c.pp.LogError("Error while module '%s' file checking: %v", moduleName, err)
		}
		return
	}

	if slices.Contains(*filePaths, path) {
		return
	}

	*filePaths = append(*filePaths, path)
	if mData.HasData() {
		c.applyModuleMetaData(mData)
	}
	*modulesForBuild = append(*modulesForBuild, moduleName)

	c.checkModuleDependencies(path, modulesForBuild, filePaths)
}

func (c *Compiler) checkModuleDependencies(modulePath string, modulesForBuild *[]string, filePaths *[]string) {
	// lx.import(ModuleName)  =>  add used modules (path arguments in the
	// module's own lx.import(...) calls are resolved later, in place, when
	// this module's file itself goes through compileFileGroup/
	// compileCodeOuterDirectives - here we only need to pre-discover its
	// bare module-name arguments, to fold them into the same dependency scan)
	code, err := os.ReadFile(modulePath)
	if err != nil {
		c.pp.LogError("Failed to read module file '%s': %v\n", modulePath, err)
		return
	}

	calls := findImportCalls(string(code))
	if len(calls) == 0 {
		return
	}

	for _, call := range calls {
		for _, moduleName := range call.modules {
			c.checkModule(moduleName, modulesForBuild, filePaths)
		}
	}
}

func (c *Compiler) applyModuleMetaData(mData jspp.IJSModuleData) {
	data := mData.Data()
	i18n, ok := data["i18n"].(string)
	if ok {
		c.applyModuleI18n(mData, i18n)
	}
}

func (c *Compiler) applyModuleI18n(mData jspp.IJSModuleData, i18n string) {
	var path string
	if filepath.IsAbs(i18n) {
		path = i18n
	} else {
		dir := filepath.Dir(mData.Path())
		path = filepath.Join(dir, i18n)
	}

	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			c.pp.LogError("File i18n '%s' for module '%s' does not exist", path, mData.Name())
		} else {
			c.pp.LogError("Error while module '%s' i18n '%s' file checking", mData.Name(), path)
		}
		return
	}

	file, err := os.Open(path)
	if err != nil {
		c.pp.LogError("Can not open module '%s' i18n '%s' file", mData.Name(), path)
		return
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	i18nMap := make(map[string]map[string]string)
	if err := decoder.Decode(i18nMap); err != nil {
		c.pp.LogError("Can not decode module '%s' i18n '%s' file", mData.Name(), path)
		return
	}

	prefix := "module-" + mData.Name() + "-"
	for lang, trMap := range i18nMap {
		if c.modulesI18n == nil {
			c.modulesI18n = make(map[string]map[string]string)
		}
		_, exists := c.modulesI18n[lang]
		if !exists {
			c.modulesI18n[lang] = make(map[string]string, len(trMap))
		}
		for trKey, trVal := range trMap {
			key := prefix + trKey
			c.modulesI18n[lang][key] = trVal
		}
	}
}

func extractModuleNames(matches [][]string) []string {
	var names []string
	for _, match := range matches {
		if len(match) > 1 {
			names = append(names, match[1])
		}
	}
	return names
}
