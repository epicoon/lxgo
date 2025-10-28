package inconf

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/epicoon/lxgo/kernel"
)

func checkParams(app kernel.IApp, params map[string]any, report *[]string) {

	fmt.Printf("Params: %v\n", params)

	cfg := app.Config()
	mp := cfg.ToMap()

	for name, val := range params {
		existing, found := getNestedValue(mp, name)
		if !found {
			*report = append(*report, fmt.Sprintf("%s: параметр не найден, будет создан", name))
			continue
		}
		if reflect.TypeOf(existing) != reflect.TypeOf(val) {
			*report = append(*report, fmt.Sprintf("%s: тип не совпадает (%T != %T)", name, existing, val))
		} else {
			*report = append(*report, fmt.Sprintf("%s: OK (%T)", name, val))
		}
	}
}

func checkArrAdd(app kernel.IApp, list map[string][]any, report *[]string) {

	fmt.Printf("Add: %v\n", list)

	cfg := app.Config()
	for name, arr := range list {
		existing, found := getNestedValue(cfg, name)
		if !found {
			*report = append(*report, fmt.Sprintf("%s: массив не найден, будет создан", name))
			continue
		}
		existingVal := reflect.ValueOf(existing)
		if existingVal.Kind() != reflect.Slice {
			*report = append(*report, fmt.Sprintf("%s: не является массивом", name))
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
				*report = append(*report, fmt.Sprintf("%s[%v]: уже существует", name, newElem))
			} else {
				*report = append(*report, fmt.Sprintf("%s[%v]: будет добавлен", name, newElem))
			}
		}
	}
}

func checkArrRemove(app kernel.IApp, list map[string][]any, report *[]string) {

	fmt.Printf("Remove: %v\n", list)

	cfg := app.Config()
	for name, arr := range list {
		existing, found := getNestedValue(cfg, name)
		if !found {
			*report = append(*report, fmt.Sprintf("%s: массив не найден, удалять нечего", name))
			continue
		}
		existingVal := reflect.ValueOf(existing)
		if existingVal.Kind() != reflect.Slice {
			*report = append(*report, fmt.Sprintf("%s: не является массивом", name))
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
				*report = append(*report, fmt.Sprintf("%s[%v]: будет удалён", name, remElem))
			} else {
				*report = append(*report, fmt.Sprintf("%s[%v]: элемента нет", name, remElem))
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

		// is current value is map
		var ok bool
		var m map[string]any
		m, ok = cur.(map[string]any)
		if !ok {
			var c kernel.Config
			c, ok = cur.(kernel.Config)
			if !ok {
				return nil, false
			}
			m = c.ToMap()
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
