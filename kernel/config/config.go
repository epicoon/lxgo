// Package config loads and reads a kernel.Dict from YAML - Load reads the
// file at path, merges in any files named by its own "Import" key (a list
// of paths, each relative to the file that names it, each resolved the
// same cascading way itself - see Load's own doc comment), and substitutes
// "${VAR}"/"${VAR:-default}" placeholders from a .env file and the process
// environment (the "Env" key). GetParam/HasParam/SetParam then read/write
// individual parameters, with GetParam coercing between common types (e.g.
// a YAML string into an int).
package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/cast"
	"gopkg.in/yaml.v3"
)

// Load reads and parses the YAML config file at path, merging in its own
// "Import" cascade (see loadCascade) and applying environment-variable
// substitution - see the package doc comment for the full behavior.
func Load(path string) (kernel.IDict, error) {
	conf, err := loadCascade(path)
	if err != nil {
		return nil, err
	}

	dir := filepath.Dir(path)

	envPath := filepath.Join(dir, ".env")
	required := false
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
		required = true
	}

	if err := applyEnv(conf, envPath, required); err != nil {
		return conf, fmt.Errorf("error while applying evnironment variables: %v", err)
	}

	return conf, nil
}

// SetParam sets a single top-level config parameter.
func SetParam(c kernel.IDict, param string, val any) {
	c.Set(param, val)
}

// HasParam reports whether param is set at the top level of c.
func HasParam(c kernel.IDict, param string) bool {
	return c.Has(param)
}

// GetParam returns param's value from c, coerced to T - see cast.Value for
// the supported conversions.
func GetParam[T any](c kernel.IDict, param string) (T, error) {
	if !c.Has(param) {
		return *new(T), fmt.Errorf("config does not contain parameter '%s'", param)
	}

	result, err := cast.To[T](c.Get(param))
	if err != nil {
		return *new(T), fmt.Errorf("wrong value type for config %q param: %w", param, err)
	}
	return result, nil
}

func load(path string) (*kernel.Dict, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("can not open config file: %w", err)
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	config := make(kernel.Dict)
	if err := decoder.Decode(config); err != nil {
		return nil, fmt.Errorf("cannot decode config file: %w", err)
	}

	return &config, nil
}

// loadCascade reads the YAML config file at path, then merges in each file
// named by its own top-level "Import" key, in array order - a later entry
// overwrites fields set by an earlier one or by path's own content (see
// mergeRecursive). Each import path is relative to path's own directory,
// not necessarily the original, top-level config's - and is itself fully
// resolved the same way first (its own "Import", if any, included), so
// import chains cascade to any depth.
func loadCascade(path string) (*kernel.Dict, error) {
	conf, err := load(path)
	if err != nil {
		return nil, err
	}

	rawImports, ok := (*conf)["Import"]
	if !ok {
		return conf, nil
	}
	imports, ok := rawImports.([]any)
	if !ok {
		return conf, fmt.Errorf("wrong type for \"Import\": expected a list of paths")
	}

	dir := filepath.Dir(path)
	for i, raw := range imports {
		iPath, ok := raw.(string)
		if !ok {
			return conf, fmt.Errorf("wrong type for \"Import\"[%d]: expected a string path", i)
		}
		iConf, err := loadCascade(filepath.Join(dir, iPath))
		if err != nil {
			return conf, fmt.Errorf("can not read imported config %q: %w", iPath, err)
		}
		mergeRecursive(*conf, *iConf)
	}

	return conf, nil
}

func mergeRecursive(dst, src kernel.Dict) {
	for key, srcVal := range src {
		if dstVal, ok := dst[key]; ok {
			dstMap, okDst := dstVal.(kernel.Dict)
			srcMap, okSrc := srcVal.(kernel.Dict)
			if okDst && okSrc {
				mergeRecursive(dstMap, srcMap)
				continue
			}
		}
		dst[key] = srcVal
	}
}

func applyEnv(conf *kernel.Dict, filename string, required bool) error {
	env := make(map[string]any, 0)

	file, err := os.Open(filename)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		if required {
			return fmt.Errorf("env file '%s' required but not found", filename)
		}
		// No .env file, and none required - "${VAR}" placeholders still
		// need resolving (from the process environment, or their own
		// ":-default"), there's just nothing to pre-load from a file.
		return envToConfig(conf, env)
	}
	defer file.Close()

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

func envToConfig(conf *kernel.Dict, env map[string]any) error {
	for k, v := range *conf {
		str, ok := v.(string)
		if ok {
			if !strings.Contains(str, "${") {
				continue
			}

			val, err := substituteEnvVars(str, env)
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
	subConf, ok := set.(kernel.Dict)
	if ok {
		return envToConfig(&subConf, env)
	}

	arr, ok := set.([]any)
	if ok {
		for i, el := range arr {
			str, ok := el.(string)
			if ok {
				if !strings.Contains(str, "${") {
					continue
				}

				val, err := substituteEnvVars(str, env)
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

// envVarPattern matches one "${VAR}"/"${VAR:-default}" placeholder.
var envVarPattern = regexp.MustCompile(`\$\{([^}]*)\}`)

// substituteEnvVars replaces every "${VAR}"/"${VAR:-default}" placeholder
// in str with its resolved value.
func substituteEnvVars(str string, env map[string]any) (string, error) {
	var firstErr error
	result := envVarPattern.ReplaceAllStringFunc(str, func(match string) string {
		if firstErr != nil {
			return match
		}
		val, err := defineEnvVal(match, env)
		if err != nil {
			firstErr = err
			return match
		}
		return fmt.Sprint(val)
	})
	if firstErr != nil {
		return "", firstErr
	}
	return result, nil
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
