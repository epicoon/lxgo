package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/epicoon/lxgo/jspp"
	"gopkg.in/yaml.v3"
)

type MapBuilderOptions struct {
	Modules bool
	Plugins bool
}

type goModule struct {
	Path string
	Dir  string
}

func BuildMaps(pp jspp.IPreprocessor, op MapBuilderOptions) error {
	modulesMap, pluginsMap := getMaps(pp, op)

	if op.Modules {
		if err := pp.ModulesMap().Save(modulesMap); err != nil {
			return err
		}
	}
	if op.Plugins {
		if err := pp.PluginManager().Save(pluginsMap); err != nil {
			return err
		}
	}

	return nil
}

func GetModulesSrcList(pp jspp.IPreprocessor) []string {
	mmPaths := make([]string, 0, 1)

	for _, path := range pp.Config().Modules {
		mmPaths = append(mmPaths, pp.App().Pathfinder().GetAbsPath(path))
	}

	goMm := getGoModules()
	for _, goMod := range goMm {
		if goMod.Dir != "" {
			mmPaths = append(mmPaths, goMod.Dir)
		}
	}

	return mmPaths
}

func GetPluginsSrcList(pp jspp.IPreprocessor) []string {
	ppPaths := make([]string, 0, 1)

	for _, path := range pp.Config().Plugins {
		ppPaths = append(ppPaths, pp.App().Pathfinder().GetAbsPath(path))
	}

	goMm := getGoModules()
	for _, goMod := range goMm {
		if goMod.Dir != "" {
			ppPaths = append(ppPaths, goMod.Dir)
		}
	}

	return ppPaths
}

func getGoModules() []goModule {
	cmd := exec.Command("go", "list", "-m", "-json", "all")
	output, err := cmd.Output()
	if err != nil {
		panic(err)
	}

	var modules []goModule
	decoder := json.NewDecoder(strings.NewReader(string(output)))

	for {
		var mod goModule
		if err := decoder.Decode(&mod); err != nil {
			break
		}
		modules = append(modules, mod)
	}

	return modules
}

func getMaps(pp jspp.IPreprocessor, op MapBuilderOptions) ([]jspp.IJSModuleData, []jspp.IPluginData) {
	var mmMap []jspp.IJSModuleData
	var ppMap []jspp.IPluginData

	for _, p := range pp.Config().Modules {
		dir := pp.App().Pathfinder().GetAbsPath(p)
		filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if err := checkPath(pp, path, info, MapBuilderOptions{Modules: true}, &mmMap, &ppMap); err != nil {
				return err
			}
			return nil
		})
	}

	for _, p := range pp.Config().Plugins {
		dir := pp.App().Pathfinder().GetAbsPath(p)
		filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if err := checkPath(pp, path, info, MapBuilderOptions{Plugins: true}, &mmMap, &ppMap); err != nil {
				return err
			}
			return nil
		})
	}

	goModules := getGoModules()
	for _, goModule := range goModules {
		filepath.Walk(goModule.Dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if err := checkPath(pp, path, info, op, &mmMap, &ppMap); err != nil {
				return err
			}
			return nil
		})
	}

	return mmMap, ppMap
}

func checkPath(
	pp jspp.IPreprocessor,
	path string,
	info os.FileInfo,
	op MapBuilderOptions,
	mmMap *[]jspp.IJSModuleData,
	ppMap *[]jspp.IPluginData,
) error {
	if op.Modules {
		if err := checkModulePath(pp, path, info, mmMap); err != nil {
			return err
		}
	}
	if op.Plugins {
		if err := checkPluginPath(pp, path, info, ppMap); err != nil {
			return err
		}
	}
	return nil
}

func checkModulePath(pp jspp.IPreprocessor, path string, info os.FileInfo, mmMap *[]jspp.IJSModuleData) error {
	if info.IsDir() || !strings.HasSuffix(path, ".js") {
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	code := string(data)
	re := regexp.MustCompile(`@lx:module +?([^;]+?) *?;`)
	match := re.FindStringSubmatch(code)
	if match == nil {
		return nil
	}

	var entryPath string
	modsPath := pp.Config().ModsPath
	if modsPath != "" {
		re = regexp.MustCompile(`@lx:namespace +?([^;]+?) *?;`)
		nmspMatch := re.FindStringSubmatch(code)
		if nmspMatch == nil {
			entryPath = modsPath
		} else {
			nmsp := strings.ReplaceAll(nmspMatch[1], ".", "/")
			entryPath = filepath.Join(modsPath, nmsp)
		}

		destPath := entryPath

		dir, file := filepath.Split(path)
		dirName := filepath.Base(dir)
		ext := filepath.Ext(file)
		fileName := strings.TrimSuffix(file, ext)
		if dirName == fileName {
			entryPath = filepath.Join(destPath, dirName, file)
			if err := copyDir(dir, filepath.Join(destPath, dirName)); err != nil {
				return fmt.Errorf("JS-Module copying error: %v", err)
			}
		} else {
			entryPath = filepath.Join(destPath, file)
			if err := copyFile(path, entryPath); err != nil {
				return fmt.Errorf("JS-Module copying error: %v", err)
			}
		}
	} else {
		entryPath = path
	}

	jsData := pp.ModulesMap().NewData(match[1], entryPath)

	re = regexp.MustCompile(`@lx:module-data: *(.+?) *= *([^;]+?) *;`)
	matches := re.FindAllStringSubmatch(code, -1)
	for _, val := range matches {
		jsData.AddData(val[1], val[2])
	}
	re = regexp.MustCompile(`@widget\s+([\w\d_.]+)`)
	match = re.FindStringSubmatch(code)
	if len(match) == 2 {
		jsData.AddData("widget", match[1])
	}

	*mmMap = append(*mmMap, jsData)
	return nil
}

func checkPluginPath(pp jspp.IPreprocessor, path string, info os.FileInfo, ppMap *[]jspp.IPluginData) error {
	if !info.IsDir() {
		return nil
	}

	configPath := filepath.Join(path, "lx-plugin.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}
	var config struct {
		Name   string         `yaml:"name"`
		Server map[string]any `yaml:"server"`
	}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return err
	}

	rawKey, exists := config.Server["key"]
	var key string
	if exists {
		val, ok := rawKey.(string)
		if ok {
			key = val
		} else {
			pp.LogError("invalid server key value '%v' for plugin '%s'", rawKey, config.Name)
			key = ""
		}
	} else {
		key = ""
	}
	plugin := pp.PluginManager().NewData(config.Name, path, key)
	*ppMap = append(*ppMap, plugin)

	return nil
}

func copyFile(src, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), os.ModePerm); err != nil {
		return err
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	destFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	return err
}

func copyDir(srcDir, destDir string) error {
	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(destDir, relPath)

		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		} else {
			return copyFile(path, destPath)
		}
	})
}
