package modules

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/epicoon/lxgo/jspp"
	"github.com/epicoon/lxgo/jspp/internal/utils"
)

type Map struct {
	pp   jspp.IPreprocessor
	data map[string]*data
}

/** @interface */
var _ jspp.IModulesMap = (*Map)(nil)

/** @constructor */
func NewMap(pp jspp.IPreprocessor) jspp.IModulesMap {
	return &Map{pp: pp}
}

func (m *Map) Path() string {
	app := m.pp.App()
	dir := app.Pathfinder().GetAbsPath(m.pp.Config().MapsPath)
	return filepath.Join(dir, "_modules.json")
}

func (m *Map) NewData(name, path string) jspp.IJSModuleData {
	return NewJSModuleData(name, path)
}

func (m *Map) Has(moduleName string) bool {
	_, ok := m.data[moduleName]
	return ok
}

func (m *Map) Get(moduleName string) jspp.IJSModuleData {
	if m.data == nil {
		if err := m.Load(); err != nil {
			m.pp.LogError("can not load modules map: %v", err)
			return nil
		}
	}

	result, ok := m.data[moduleName]
	if !ok {
		return nil
	}
	return result
}

func (m *Map) Load() error {
	path := m.Path()

	d, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("can not read file %s: %w", path, err)
	}

	var mData []*data
	if err := json.Unmarshal(d, &mData); err != nil {
		return fmt.Errorf("can not parse JSON from %s: %w", path, err)
	}

	m.data = mapping(mData)

	return nil
}

func (m *Map) Save(d []jspp.IJSModuleData) error {
	filePath := m.Path()
	dir := filepath.Dir(filePath)

	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return fmt.Errorf("can not make directory: %w", err)
	}

	jsonData, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return fmt.Errorf("can not serialize JSON: %w", err)
	}

	if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
		return fmt.Errorf("can not write file: %w", err)
	}

	dataSlice := make([]*data, len(d))
	for i, v := range d {
		if d, ok := v.(*data); ok {
			dataSlice[i] = d
		} else {
			return fmt.Errorf("invalid data type: expected *data, got %T", v)
		}
	}
	m.data = mapping(dataSlice)

	return nil
}

func (m *Map) Each(f func(data jspp.IJSModuleData)) {
	if m.data == nil {
		m.Load()
	}
	for _, d := range m.data {
		f(d)
	}
}

func (m *Map) Reset() {
	utils.BuildMaps(m.pp, utils.MapBuilderOptions{Modules: true})
}

func mapping(d []*data) map[string]*data {
	result := make(map[string]*data, len(d))
	for _, val := range d {
		result[val.Name()] = val
	}
	return result
}
