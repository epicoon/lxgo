package inconf

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/cast"
)

func checkParams(app kernel.IApp, params map[string]any, report *[]string) {
	cfg := app.Config()

	for name, val := range params {
		existing, found := getNestedValue(cfg, name)
		if !found {
			*report = append(*report, fmt.Sprintf("%s: parameter not found, will be created", name))
			continue
		}
		if reflect.TypeOf(existing) != reflect.TypeOf(val) {
			*report = append(*report, fmt.Sprintf("%s: type mismatch (%T != %T)", name, existing, val))
		} else {
			*report = append(*report, fmt.Sprintf("%s: OK (%T)", name, val))
		}
	}
}

func checkArrAdd(app kernel.IApp, list map[string][]any, report *[]string) {
	cfg := app.Config()
	for name, arr := range list {
		existing, found := getNestedValue(cfg, name)
		if !found {
			*report = append(*report, fmt.Sprintf("%s: array not found, will be created", name))
			continue
		}
		existingVal := reflect.ValueOf(existing)
		if existingVal.Kind() != reflect.Slice {
			*report = append(*report, fmt.Sprintf("%s: is not an array", name))
			continue
		}
		for _, newElem := range arr {
			found := false
			for i := 0; i < existingVal.Len(); i++ {
				if reflect.DeepEqual(existingVal.Index(i).Interface(), newElem) {
					found = true
					break
				}
			}
			if found {
				*report = append(*report, fmt.Sprintf("%s[%v]: already exists", name, newElem))
			} else {
				*report = append(*report, fmt.Sprintf("%s[%v]: will be added", name, newElem))
			}
		}
	}
}

func checkArrRemove(app kernel.IApp, list map[string][]any, report *[]string) {
	cfg := app.Config()
	for name, arr := range list {
		existing, found := getNestedValue(cfg, name)
		if !found {
			*report = append(*report, fmt.Sprintf("%s: array not found, nothing to remove", name))
			continue
		}
		existingVal := reflect.ValueOf(existing)
		if existingVal.Kind() != reflect.Slice {
			*report = append(*report, fmt.Sprintf("%s: is not an array", name))
			continue
		}
		for _, remElem := range arr {
			found := false
			for i := 0; i < existingVal.Len(); i++ {
				if reflect.DeepEqual(existingVal.Index(i).Interface(), remElem) {
					found = true
					break
				}
			}
			if found {
				*report = append(*report, fmt.Sprintf("%s[%v]: will be removed", name, remElem))
			} else {
				*report = append(*report, fmt.Sprintf("%s[%v]: element not found", name, remElem))
			}
		}
	}
}

func getNestedValue(cfg any, path string) (any, bool) {
	if cfg == nil {
		return nil, false
	}

	cur := cfg
	parts := strings.Split(path, ".")

	for _, part := range parts {
		// Check indexes — example: "Servers[0]"
		key, idx := parseArrayAccess(part)

		m, err := cast.To[map[string]any](cur)
		if err != nil {
			return nil, false
		}

		val, exists := m[key]
		if !exists {
			return nil, false
		}

		// if has index — call to array element
		if idx != nil {
			arr, ok := val.([]any)
			if !ok {
				return nil, false
			}
			if *idx < 0 || *idx >= len(arr) {
				return nil, false
			}
			val = arr[*idx]
		}

		cur = val
	}

	return cur, true
}

// parseArrayAccess("Servers[3]") → ("Servers", 3)
// parseArrayAccess("Params") → ("Params", nil)
func parseArrayAccess(s string) (string, *int) {
	open := strings.Index(s, "[")
	close := strings.Index(s, "]")

	if open == -1 || close == -1 || close < open {
		return s, nil
	}

	key := s[:open]
	idxStr := s[open+1 : close]
	i, err := strconv.Atoi(idxStr)
	if err != nil {
		return key, nil
	}

	return key, &i
}
