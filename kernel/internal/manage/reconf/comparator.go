package reconf

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/utils"
)

type diff struct {
	Path    string
	Old     any
	New     any
	OldType string
	NewType string
}

type report struct {
	errs    []diff
	added   []diff
	removed []diff
	changed []diff
}

func compareConfigs(orig, new *kernel.Config) *report {
	rep := &report{}
	compareRecursive(orig.ToMap(), new.ToMap(), "", rep)
	return rep
}

func compareRecursive(orig, new map[string]any, prefix string, diffs *report) {
	visited := make(map[string]bool)

	// Check old keys
	for k, oldVal := range orig {
		fullKey := joinKey(prefix, k)
		visited[k] = true

		newVal, exists := new[k]
		if !exists {
			diffs.removed = append(diffs.removed, diff{Path: fullKey, Old: oldVal})
			continue
		}

		// If maps
		oldMap, okOld := oldVal.(map[string]any)
		newMap, okNew := newVal.(map[string]any)
		if okOld && okNew {
			compareRecursive(oldMap, newMap, fullKey, diffs)
			continue
		}
		// If configs
		oldConf, okOld := oldVal.(kernel.Config)
		newConf, okNew := newVal.(kernel.Config)
		if okOld && okNew {
			compareRecursive(oldConf.ToMap(), newConf.ToMap(), fullKey, diffs)
			continue
		}

		// If arrays
		if isSlice(oldVal) && isSlice(newVal) {
			compareSlices(oldVal, newVal, fullKey, diffs)
			continue
		}

		// Check values
		if reflect.TypeOf(oldVal) != reflect.TypeOf(newVal) {
			diffs.errs = append(diffs.errs, diff{
				Path:    fullKey,
				Old:     oldVal,
				New:     newVal,
				OldType: fmt.Sprintf("%T", oldVal),
				NewType: fmt.Sprintf("%T", newVal),
			})
		} else if !reflect.DeepEqual(oldVal, newVal) {
			diffs.changed = append(diffs.changed, diff{
				Path: fullKey,
				Old:  oldVal,
				New:  newVal,
			})
		}
	}

	// Check new keys
	for k, newVal := range new {
		if visited[k] {
			continue
		}
		fullKey := joinKey(prefix, k)
		diffs.added = append(diffs.added, diff{Path: fullKey, New: newVal})
	}
}

func compareSlices(oldVal, newVal any, prefix string, diffs *report) {
	oldSlice := reflect.ValueOf(oldVal)
	newSlice := reflect.ValueOf(newVal)

	lOld := oldSlice.Len()
	lNew := newSlice.Len()
	hashesOld := make([]string, lOld)
	hashesNew := make([]string, lNew)
	inxMapOld := make(map[string]int, lOld)
	inxMapNew := make(map[string]int, lNew)

	for i := range lOld {
		v := oldSlice.Index(i).Interface()
		h := hashValue(v)
		hashesOld[i] = h
		inxMapOld[h] = i
	}
	for i := range lNew {
		v := newSlice.Index(i).Interface()
		h := hashValue(v)
		hashesNew[i] = h
		inxMapNew[h] = i
	}

	uniqOld := utils.SliceDiff(hashesOld, hashesNew)
	uniqNew := utils.SliceDiff(hashesNew, hashesOld)

	for _, v := range uniqOld {
		i := inxMapOld[v]
		key := fmt.Sprintf("%s[]", prefix)
		diffs.removed = append(diffs.removed, diff{Path: key, Old: oldSlice.Index(i).Interface()})
	}

	for _, v := range uniqNew {
		i := inxMapNew[v]
		key := fmt.Sprintf("%s[]", prefix)

		expl := oldSlice.Index(0).Interface()
		newEl := newSlice.Index(i).Interface()

		if reflect.TypeOf(expl) != reflect.TypeOf(newEl) {
			diffs.errs = append(diffs.errs, diff{
				Path:    key,
				New:     newEl,
				OldType: fmt.Sprintf("%T", expl),
				NewType: fmt.Sprintf("%T", newEl),
			})
		} else {
			diffs.added = append(diffs.added, diff{Path: key, New: newEl})
		}
	}
}

func hashValue(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	h := sha1.Sum(b)
	return fmt.Sprintf("%x", h)
}

func isSlice(v any) bool {
	if v == nil {
		return false
	}
	return reflect.TypeOf(v).Kind() == reflect.Slice
}

func joinKey(prefix, key string) string {
	if prefix == "" {
		return key
	}
	return prefix + "." + key
}
