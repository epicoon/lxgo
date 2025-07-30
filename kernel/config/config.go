package config

import (
	"fmt"
	"os"
	"reflect"

	"github.com/epicoon/lxgo/kernel"
	"gopkg.in/yaml.v3"
)

func Load(filepath string) (*kernel.Config, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("can not open config file: %w", err)
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	config := make(kernel.Config)
	if err := decoder.Decode(config); err != nil {
		return nil, fmt.Errorf("cannot decode config file: %w", err)
	}

	return &config, nil
}

func SetParam(c *kernel.Config, param string, val any) {
	(*c)[param] = val
}

func HasParam(c *kernel.Config, param string) bool {
	_, exists := (*c)[param]
	return exists
}

func GetParam[T any](c *kernel.Config, param string) (T, error) {
	val, exists := (*c)[param]
	if !exists {
		return *new(T), fmt.Errorf("config does not contain parameter '%s'", param)
	}

	typedVal, ok := val.(T)
	if ok {
		return typedVal, nil
	}

	if reflect.TypeOf(val).Kind() == reflect.Slice && reflect.TypeOf((*new(T))).Kind() == reflect.Slice {
		input := reflect.ValueOf(val)
		output := reflect.MakeSlice(reflect.TypeOf(*new(T)), input.Len(), input.Len())

		for i := 0; i < input.Len(); i++ {
			item := input.Index(i).Interface()
			convertedItem, ok := item.(string)
			if !ok {
				return *new(T), fmt.Errorf("invalid type in slice for config param %q: expected string, got %T", param, item)
			}
			output.Index(i).Set(reflect.ValueOf(convertedItem))
		}

		return output.Interface().(T), nil
	}

	return *new(T), fmt.Errorf("wrong value type for config %q param: %v, type: %T", param, val, val)
}
