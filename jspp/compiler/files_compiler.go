package compiler

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
)

func (c *Compiler) plugRequire(requireName string, flags Flags, parentDir string, rootPath string) (string, error) {
	var filePaths []string

	if !strings.HasSuffix(requireName, "/") {
		if !strings.HasSuffix(requireName, ".js") {
			requireName += ".js"
		}
		filePaths = append(filePaths, filepath.Join(parentDir, requireName))
	} else {
		dirFiles, err := listFilesInDir(requireName, parentDir, flags.Recursive)
		if err != nil {
			return "", err
		}
		filePaths = append(filePaths, dirFiles...)
	}

	return c.compileFileGroup(filePaths, flags, rootPath)
}

// addModuleI18nPrefix rewrites lx.i18n('key') calls (also lx.i18n(key) with
// no quotes, or lx.i18n("key")) into lx.i18n('module-{{moduleName}}-key') -
// i.e. inserts the module-scoping prefix right after the opening quote,
// preserving whichever quote character (or absence of one) was actually
// used, so the call stays syntactically valid.
func addModuleI18nPrefix(code string, moduleName string) string {
	re := regexp.MustCompile(`lx\.i18n\((['"]?)`)
	return re.ReplaceAllString(code, `lx.i18n(${1}module-`+moduleName+`-`)
}

func (c *Compiler) compileFileGroup(fileNames []string, flags Flags, rootPath string) (string, error) {
	type fileInfo struct {
		Path         string
		Code         string
		DependsOf    []string
		Dependencies []string
		Counter      int
		ModuleName   string
	}

	list := make(map[string]*fileInfo)
	classesMap := make(map[string]string)
	reClass := regexp.MustCompile(`(?:@lx:namespace\s+([\w\d_.]+?)\s*;\s*)?class\s+(.+?)\b\s+(?:extends\s+([\w\d_.]+?))?\s*{`)
	reModule := regexp.MustCompile(`@lx:module +?([^;]+?) *?;`)

	// Collect files data
	for _, fileName := range fileNames {
		data, err := os.ReadFile(fileName)
		if err != nil {
			continue
		}
		originalCode := string(data)

		moduleName := ""
		if match := reModule.FindStringSubmatch(originalCode); match != nil {
			moduleName = match[1]
		}

		code, err := c.compileCodeInnerDirectives(originalCode, fileName)
		if err != nil {
			return "", err
		}

		matches := reClass.FindAllStringSubmatch(originalCode, -1)
		var dependsOf []string
		for _, match := range matches {
			namespace, class, parent := match[1], match[2], match[3]
			className := class
			if namespace != "" {
				className = namespace + "." + class
			}
			if _, exists := classesMap[className]; exists {
				if className != "_am_" {
					fmt.Printf("Class %s is already defined in %s, cannot redeclare in %s\n", className, classesMap[className], fileName)
				}
			} else {
				classesMap[className] = fileName
			}
			if parent != "" {
				dependsOf = append(dependsOf, parent)
			}
		}

		list[fileName] = &fileInfo{
			Path:         fileName,
			Code:         code,
			DependsOf:    dependsOf,
			Dependencies: []string{},
			Counter:      0,
			ModuleName:   moduleName,
		}
	}

	// Set dependencies
	for fileName, fileInfo := range list {
		for _, parentClass := range fileInfo.DependsOf {
			if parentPath, exists := classesMap[parentClass]; exists && parentPath != fileName {
				if !slices.Contains(list[parentPath].Dependencies, fileName) {
					list[parentPath].Dependencies = append(list[parentPath].Dependencies, fileName)
				}
				continue
			}

			if c.pp == nil {
				continue
			}
			parentData := c.pp.ModulesMap().Get(parentClass)
			if parentData == nil {
				continue
			}
			if slices.Contains(c.compiledFiles, parentData.Path()) {
				// Already written out somewhere earlier in c.modulesCode -
				// safe, nothing more to do.
				continue
			}
			var extraFilePaths, extraModulesForBuild []string
			c.checkModule(parentClass, &extraModulesForBuild, &extraFilePaths)
			forcedSingleFile := false
			if len(extraFilePaths) == 0 {
				if slices.Contains(c.compilingFiles, parentData.Path()) {
					continue
				}
				extraFilePaths = []string{parentData.Path()}
				forcedSingleFile = true
				c.compilingFiles = append(c.compilingFiles, parentData.Path())
			}
			extraCode, err := c.compileFileGroup(extraFilePaths, Flags{}, rootPath)
			if forcedSingleFile {
				c.compilingFiles = c.compilingFiles[:len(c.compilingFiles)-1]
			}
			if err != nil {
				return "", err
			}
			for _, m := range extraModulesForBuild {
				if !slices.Contains(c.compiledModules, m) {
					c.compiledModules = append(c.compiledModules, m)
				}
			}
			c.modulesCode += extraCode
		}
	}

	// Count dependencies recursievly
	var incrementCounter func(string)
	incrementCounter = func(index string) {
		list[index].Counter++
		for _, dep := range list[index].Dependencies {
			incrementCounter(dep)
		}
	}
	for key := range list {
		incrementCounter(key)
	}

	// Sort files according to dependencies. Built from fileNames (this
	// call's own argument, in a fixed order) rather than by ranging over
	// the list map - Go deliberately randomizes map iteration order on
	// every run, so two files with no dependency relationship between them
	// (equal Counter - e.g. two unrelated bare-module lx.import(...)
	// targets pulled into the same batch by checkModuleDependencies, with
	// nothing in this batch's own class-extends graph linking them) would
	// otherwise come out in a coin-flip order every time this batch is
	// compiled, even though sort.SliceStable is used below specifically to
	// keep equal-Counter files in their input order.
	sortedFiles := make([]*fileInfo, 0, len(list))
	seen := make(map[string]bool, len(list))
	for _, fileName := range fileNames {
		file, ok := list[fileName]
		if !ok || seen[fileName] {
			continue
		}
		seen[fileName] = true
		sortedFiles = append(sortedFiles, file)
	}
	sort.SliceStable(sortedFiles, func(i, j int) bool {
		return sortedFiles[i].Counter < sortedFiles[j].Counter
	})

	var result strings.Builder
	var err error
	for _, file := range sortedFiles {
		if !c.checkFileCompileAvailable(file.Path, flags.Force) {
			continue
		}
		c.noteFileCompiled(file.Path)

		code := file.Code
		code, err = c.compileCodeOuterDirectives(code, file.Path, !flags.Unwrapped)
		if err != nil {
			return "", err
		}

		// lx.i18n(  =>  lx.i18n(module-{{moduleName}}-  - applied here, to
		// the fully assembled code. A module's i18n data is meant to cover
		// its whole file tree.
		if file.ModuleName != "" {
			code = addModuleI18nPrefix(code, file.ModuleName)
		}

		code = c.markDevInterrupting(code, rootPath)

		result.WriteString(code)
	}

	return result.String(), nil
}

func (c *Compiler) checkFileCompileAvailable(path string, force bool) bool {
	if _, err := os.Stat(path); err != nil {
		c.logError("Can not compile file %s: %v", path, err)
		return false
	}

	if force {
		return true
	}

	return !slices.Contains(c.compiledFiles, path)
}

func (c *Compiler) noteFileCompiled(path string) {
	c.compiledFiles = append(c.compiledFiles, path)
}

func listFilesInDir(dirPath string, parentDir string, recursive bool) ([]string, error) {
	fullPath := filepath.Join(parentDir, dirPath)
	var files []string
	err := filepath.Walk(fullPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".js") {
			files = append(files, path)
		}
		if info.IsDir() && !recursive && path != fullPath {
			return filepath.SkipDir
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}
