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

func (c *Compiler) plugAllRequires(code, path string) (string, error) {
	parentDir := ""
	if path != "" {
		parentDir = filepath.Dir(path) + "/"
	}

	pattern := `@lx:require(\s+-[\S]+)?\s+[\'"]?([^;]+?)[\'"]?;`
	re := regexp.MustCompile(pattern)

	processedCode := re.ReplaceAllStringFunc(code, func(match string) string {
		matches := re.FindStringSubmatch(match)
		if len(matches) < 3 {
			return match
		}

		ff := matches[1]
		requireName := matches[2]

		flags := Flags{
			Recursive: strings.Contains(ff, "R"),
			Force:     strings.Contains(ff, "F"),
			Unwrapped: strings.Contains(ff, "U"),
		}

		includedCode, err := c.plugRequire(requireName, flags, parentDir, path)
		if err != nil {
			fmt.Printf("Warning: could not process require %s: %v\n", requireName, err)
			return match
		}

		return includedCode
	})

	return processedCode, nil
}

func (c *Compiler) plugRequire(requireName string, flags Flags, parentDir string, rootPath string) (string, error) {
	var filePaths []string

	// Files list
	if strings.HasPrefix(requireName, "{") && strings.HasSuffix(requireName, "}") {
		entries := strings.Split(strings.Trim(requireName, "{}"), ",")
		for _, entry := range entries {
			entry = strings.TrimSpace(entry)
			filePaths = append(filePaths, filepath.Join(parentDir, entry))
		}
	} else {
		// Single file or directory
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
	}

	return c.compileFileGroup(filePaths, flags, rootPath)
}

func (c *Compiler) compileFileGroup(fileNames []string, flags Flags, rootPath string) (string, error) {
	//TODO
	_ = rootPath

	type fileInfo struct {
		Path         string
		Code         string
		DependsOf    []string
		Dependencies []string
		Counter      int
	}

	list := make(map[string]*fileInfo)
	classesMap := make(map[string]string)
	reClass := regexp.MustCompile(`(?:@lx:namespace\s+([\w\d_.]+?)\s*;\s*)?class\s+(.+?)\b\s+(?:extends\s+([\w\d_.]+?))?\s*{`)

	// Collect files data
	for _, fileName := range fileNames {
		data, err := os.ReadFile(fileName)
		if err != nil {
			continue
		}
		originalCode := string(data)

		// lx(i18n).  =>  lx(i18n).module-{{moduleName}}-
		re := regexp.MustCompile(`@lx:module +?([^;]+?) *?;`)
		match := re.FindStringSubmatch(originalCode)
		if match != nil {
			moduleName := match[1]
			re := regexp.MustCompile(`lx\(i18n\)\.`)
			originalCode = re.ReplaceAllString(originalCode, "lx(i18n).module-"+moduleName+"-")
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
		}
	}

	// Set dependencies
	for fileName, fileInfo := range list {
		for _, parentClass := range fileInfo.DependsOf {
			if parentPath, exists := classesMap[parentClass]; exists && parentPath != fileName {
				if !slices.Contains(list[parentPath].Dependencies, fileName) {
					list[parentPath].Dependencies = append(list[parentPath].Dependencies, fileName)
				}
			}
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

	// Sort files according to dependencies
	sortedFiles := make([]*fileInfo, 0, len(list))
	for _, file := range list {
		sortedFiles = append(sortedFiles, file)
	}
	sort.Slice(sortedFiles, func(i, j int) bool {
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

		//TODO
		// $code = $this->markDevInterrupting($code, $rootPath);

		result.WriteString(code)
	}

	return result.String(), nil
}

func (c *Compiler) checkFileCompileAvailable(path string, force bool) bool {
	if _, err := os.Stat(path); err != nil {
		c.pp.LogError("Can not compile file %s: %v", path, err)
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
