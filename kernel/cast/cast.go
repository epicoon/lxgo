// Package cast coerces loosely-typed values (from YAML/JSON config, form
// data, etc.) to a target type - Value is the single reflect-based entry
// point, handling the common cross-type conversions (numeric strings into
// numbers, "true"/"false" strings into bool, anything into its string form,
// and recursively for slices and string-keyed maps); To is a generic-typed
// wrapper around it.
package cast

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/epicoon/lxgo/kernel"
)

// To coerces v to T - a generic-typed wrapper around Value.
func To[T any](v any) (T, error) {
	var zero T
	target := reflect.TypeOf(&zero).Elem()

	result, err := Value(v, target)
	if err != nil {
		return zero, err
	}

	typed, ok := result.(T)
	if !ok {
		return zero, fmt.Errorf("cast: %T does not fit target type %T", result, zero)
	}
	return typed, nil
}

// Value coerces v to target. Direct assignability is tried first; failing
// that, target's kind decides the conversion: numeric kinds accept other
// numeric kinds and decimal strings, bool accepts "true"/"false" strings,
// string accepts anything via its natural formatting, slices/arrays coerce
// element-wise, and string-keyed maps coerce element-wise from another
// string-keyed map or from anything with a `ToMap() map[string]any` method.
func Value(v any, target reflect.Type) (any, error) {
	if v == nil {
		return reflect.Zero(target).Interface(), nil
	}

	rv := reflect.ValueOf(v)

	if target.Kind() == reflect.Interface {
		if target.NumMethod() == 0 || rv.Type().Implements(target) {
			return v, nil
		}
		return nil, fmt.Errorf("cast: %T does not implement %s", v, target)
	}

	if rv.Type().AssignableTo(target) {
		// Convert, not Interface(): AssignableTo only says the assignment is
		// legal, but Interface() would keep v's own dynamic type (e.g. a
		// kernel.Dict staying kernel.Dict instead of becoming the requested
		// map[string]any) - Convert normalizes the dynamic type to target.
		return rv.Convert(target).Interface(), nil
	}

	switch target.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return toInt(v, target)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return toUint(v, target)
	case reflect.Float32, reflect.Float64:
		return toFloat(v, target)
	case reflect.Bool:
		return toBool(v)
	case reflect.String:
		return stringify(v), nil
	case reflect.Slice:
		return toSlice(v, target)
	case reflect.Map:
		return toMap(v, target)
	case reflect.Struct:
		return toStruct(v, target)
	}

	return nil, fmt.Errorf("cast: cannot assign %T to %s", v, target)
}

func toInt(v any, target reflect.Type) (any, error) {
	var i64 int64
	switch n := v.(type) {
	case int:
		i64 = int64(n)
	case int8:
		i64 = int64(n)
	case int16:
		i64 = int64(n)
	case int32:
		i64 = int64(n)
	case int64:
		i64 = n
	case uint:
		i64 = int64(n)
	case uint8:
		i64 = int64(n)
	case uint16:
		i64 = int64(n)
	case uint32:
		i64 = int64(n)
	case uint64:
		i64 = int64(n)
	case float32:
		i64 = int64(n)
	case float64:
		i64 = int64(n)
	case string:
		parsed, err := strconv.ParseInt(n, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("cast: cannot convert %q to %s: %w", n, target, err)
		}
		i64 = parsed
	default:
		return nil, fmt.Errorf("cast: cannot convert %T to %s", v, target)
	}

	result := reflect.New(target).Elem()
	result.SetInt(i64)
	return result.Interface(), nil
}

func toUint(v any, target reflect.Type) (any, error) {
	var u64 uint64
	switch n := v.(type) {
	case int:
		u64 = uint64(n)
	case int8:
		u64 = uint64(n)
	case int16:
		u64 = uint64(n)
	case int32:
		u64 = uint64(n)
	case int64:
		u64 = uint64(n)
	case uint:
		u64 = uint64(n)
	case uint8:
		u64 = uint64(n)
	case uint16:
		u64 = uint64(n)
	case uint32:
		u64 = uint64(n)
	case uint64:
		u64 = n
	case float32:
		u64 = uint64(n)
	case float64:
		u64 = uint64(n)
	case string:
		parsed, err := strconv.ParseUint(n, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("cast: cannot convert %q to %s: %w", n, target, err)
		}
		u64 = parsed
	default:
		return nil, fmt.Errorf("cast: cannot convert %T to %s", v, target)
	}

	result := reflect.New(target).Elem()
	result.SetUint(u64)
	return result.Interface(), nil
}

func toFloat(v any, target reflect.Type) (any, error) {
	var f64 float64
	switch n := v.(type) {
	case int:
		f64 = float64(n)
	case int8:
		f64 = float64(n)
	case int16:
		f64 = float64(n)
	case int32:
		f64 = float64(n)
	case int64:
		f64 = float64(n)
	case uint:
		f64 = float64(n)
	case uint8:
		f64 = float64(n)
	case uint16:
		f64 = float64(n)
	case uint32:
		f64 = float64(n)
	case uint64:
		f64 = float64(n)
	case float32:
		f64 = float64(n)
	case float64:
		f64 = n
	case string:
		parsed, err := strconv.ParseFloat(n, 64)
		if err != nil {
			return nil, fmt.Errorf("cast: cannot convert %q to %s: %w", n, target, err)
		}
		f64 = parsed
	default:
		return nil, fmt.Errorf("cast: cannot convert %T to %s", v, target)
	}

	result := reflect.New(target).Elem()
	result.SetFloat(f64)
	return result.Interface(), nil
}

func toBool(v any) (any, error) {
	switch b := v.(type) {
	case bool:
		return b, nil
	case string:
		parsed, err := strconv.ParseBool(b)
		if err != nil {
			return nil, fmt.Errorf("cast: cannot convert %q to bool: %w", b, err)
		}
		return parsed, nil
	default:
		return nil, fmt.Errorf("cast: cannot convert %T to bool", v)
	}
}

func stringify(v any) string {
	switch s := v.(type) {
	case string:
		return s
	case int, int8, int16, int32, int64:
		return strconv.FormatInt(reflect.ValueOf(s).Int(), 10)
	case uint, uint8, uint16, uint32, uint64:
		return strconv.FormatUint(reflect.ValueOf(s).Uint(), 10)
	case float32, float64:
		return strconv.FormatFloat(reflect.ValueOf(s).Float(), 'f', -1, 64)
	case bool:
		return strconv.FormatBool(s)
	case fmt.Stringer:
		return s.String()
	default:
		return fmt.Sprintf("%v", s)
	}
}

func toSlice(v any, target reflect.Type) (any, error) {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return nil, fmt.Errorf("cast: cannot convert %T to %s", v, target)
	}

	elemType := target.Elem()
	n := rv.Len()
	result := reflect.MakeSlice(target, 0, n)
	for i := range n {
		coerced, err := Value(rv.Index(i).Interface(), elemType)
		if err != nil {
			return nil, fmt.Errorf("cast: element %d: %w", i, err)
		}
		result = reflect.Append(result, reflect.ValueOf(coerced))
	}
	return result.Interface(), nil
}

func toMap(v any, target reflect.Type) (any, error) {
	if target.Key().Kind() != reflect.String {
		return nil, fmt.Errorf("cast: unsupported map key type %s", target.Key())
	}

	dict, ok := toStringMap(v)
	if !ok {
		return nil, fmt.Errorf("cast: cannot convert %T to %s", v, target)
	}

	elemType := target.Elem()
	result := reflect.MakeMapWithSize(target, len(dict))
	for key, val := range dict {
		coerced, err := Value(val, elemType)
		if err != nil {
			return nil, fmt.Errorf("cast: key %q: %w", key, err)
		}
		result.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(coerced))
	}
	return result.Interface(), nil
}

// toStruct populates a new value of target (a struct type) from v via
// DictToStruct - v must be a map[string]any/kernel.Dict or anything else
// toStringMap accepts.
func toStruct(v any, target reflect.Type) (any, error) {
	dict, ok := toStringMap(v)
	if !ok {
		return nil, fmt.Errorf("cast: cannot assign %T to %s", v, target)
	}

	ptr := reflect.New(target)
	if err := DictToStruct(kernel.Dict(dict), ptr.Interface()); err != nil {
		return nil, err
	}
	return ptr.Elem().Interface(), nil
}

// toStringMap reads v as a map[string]any - directly for map[string]any/
// kernel.Dict/*map[string]any, via reflection for any other string-keyed
// map, via a ToMap() method for anything else that has one, and via plain
// struct-field reflection (see structToMap) as a last resort.
func toStringMap(v any) (map[string]any, bool) {
	switch m := v.(type) {
	case map[string]any:
		return m, true
	case kernel.Dict:
		return m.ToMap(), true
	case *map[string]any:
		if m == nil {
			return nil, false
		}
		return *m, true
	}

	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Map && rv.Type().Key().Kind() == reflect.String {
		out := make(map[string]any, rv.Len())
		iter := rv.MapRange()
		for iter.Next() {
			out[iter.Key().String()] = iter.Value().Interface()
		}
		return out, true
	}

	method := rv.MethodByName("ToMap")
	if method.IsValid() && method.Type().NumIn() == 0 && method.Type().NumOut() == 1 {
		if m, ok := method.Call(nil)[0].Interface().(map[string]any); ok {
			return m, true
		}
	}

	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil, false
		}
		rv = rv.Elem()
	}
	if rv.Kind() == reflect.Struct {
		return structToMap(rv), true
	}

	return nil, false
}

// structToMap reads a struct's fields into a map[string]any, keyed by the
// same field-name resolution DictToStruct uses (dict tag, then json tag,
// then field name) - the mirror image of toStruct.
func structToMap(rv reflect.Value) map[string]any {
	typ := rv.Type()
	out := make(map[string]any, rv.NumField())
	for i := 0; i < rv.NumField(); i++ {
		fieldVal := rv.Field(i)
		if !fieldVal.CanInterface() {
			continue
		}
		out[FieldName(typ.Field(i))] = fieldVal.Interface()
	}
	return out
}
