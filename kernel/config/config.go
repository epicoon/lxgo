package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/epicoon/lxgo/kernel"
	"gopkg.in/yaml.v3"
)

func Load(path string) (*kernel.Config, error) {
	conf, err := load(path)
	if err != nil {
		return nil, err
	}

	dir := filepath.Dir(path)

	if HasParam(conf, "Local") {
		lPath, err := GetParam[string](conf, "Local")
		if err != nil {
			return conf, fmt.Errorf("wrong type for local config path: %v", err)
		}
		lPath = filepath.Join(dir, lPath)
		lConf, err := load(lPath)
		if err != nil {
			return conf, fmt.Errorf("can not read local config: %v", err)
		}
		mergeRecursive(*conf, *lConf)
	}

	envPath := filepath.Join(dir, ".env")
	if HasParam(conf, "Env") {
		env, err := GetParam[string](conf, "Env")
		if err != nil {
			return conf, fmt.Errorf("wrong type for env path: %v", err)
		}
		if strings.HasPrefix(env, "/") {
			envPath = env
		} else {
			envPath = filepath.Join(dir, env)
		}
	}

	if err := applyEnv(conf, envPath); err != nil {
		return conf, fmt.Errorf("error while applying evnironment variables: %v", err)
	}

	return conf, nil
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

func load(path string) (*kernel.Config, error) {
	file, err := os.Open(path)
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

func mergeRecursive(dst, src kernel.Config) {
	for key, srcVal := range src {
		if dstVal, ok := dst[key]; ok {
			dstMap, okDst := dstVal.(kernel.Config)
			srcMap, okSrc := srcVal.(kernel.Config)
			if okDst && okSrc {
				mergeRecursive(dstMap, srcMap)
				continue
			}
		}
		dst[key] = srcVal
	}
}

func applyEnv(conf *kernel.Config, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	env := make(map[string]any, 0)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip emplty and comment lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Split buy "="
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		val = strings.Trim(val, `"'`)

		// Set environment variables
		if os.Getenv(key) == "" {
			_ = os.Setenv(key, val)
		}

		env[key] = val
	}

	if err := envToConfig(conf, env); err != nil {
		return err
	}

	return scanner.Err()
}

func envToConfig(conf *kernel.Config, env map[string]any) error {
	for k, v := range *conf {
		str, ok := v.(string)
		if ok {
			if !strings.HasPrefix(str, "${") {
				continue
			}

			val, err := defineEnvVal(str, env)
			if err != nil {
				return err
			}
			(*conf)[k] = val

			continue
		}

		if err := envToSet(v, env); err != nil {
			return err
		}
	}
	return nil
}

func envToSet(set any, env map[string]any) error {
	subConf, ok := set.(kernel.Config)
	if ok {
		return envToConfig(&subConf, env)
	}

	arr, ok := set.([]any)
	if ok {
		for i, el := range arr {
			str, ok := el.(string)
			if ok {
				if !strings.HasPrefix(str, "${") {
					continue
				}

				val, err := defineEnvVal(str, env)
				if err != nil {
					return err
				}
				arr[i] = val

				continue
			}

			if err := envToSet(el, env); err != nil {
				return err
			}
		}
	}
	return nil
}

func defineEnvVal(str string, env map[string]any) (any, error) {
	str = strings.Trim(str, "${}")

	parts := strings.SplitN(str, ":-", 2)
	var name string
	var defaultVal any
	if len(parts) == 1 {
		name = parts[0]
		defaultVal = nil
	} else if len(parts) == 2 {
		name = parts[0]
		defaultVal = parts[1]
	} else {
		return nil, fmt.Errorf("wrong config syntax for env variable: %s", str)
	}

	if val, exists := env[name]; exists {
		return val, nil
	}
	if osVal := os.Getenv(name); osVal != "" {
		return osVal, nil
	}
	if defaultVal != nil {
		return defaultVal, nil
	}

	return nil, fmt.Errorf("env variable '%s' not found", name)
}
