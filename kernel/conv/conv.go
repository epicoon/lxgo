package conv

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/epicoon/lxgo/kernel"
)

func ToString(value any) string {
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	case string:
		return v
	case int, int8, int16, int32, int64:
		return strconv.FormatInt(reflect.ValueOf(v).Int(), 10)
	case uint, uint8, uint16, uint32, uint64:
		return strconv.FormatUint(reflect.ValueOf(v).Uint(), 10)
	case float32, float64:
		return strconv.FormatFloat(reflect.ValueOf(v).Float(), 'f', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	case fmt.Stringer:
		return v.String()
	default:
		// Use fmt.Sprintf for unsupported types
		return fmt.Sprintf("%v", v)
	}
}

func ToMap(s any) map[string]any {
	result := make(map[string]any)

	val := reflect.ValueOf(s)
	typ := reflect.TypeOf(s)

	if val.Kind() == reflect.Ptr {
		val = val.Elem()
		typ = typ.Elem()
	}

	if val.Kind() != reflect.Struct {
		fmt.Println("ToMap: not a struct")
		return result
	}

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		if !fieldVal.CanInterface() {
			continue
		}

		result[field.Name] = fmt.Sprintf("%v", fieldVal.Interface())
	}

	return result
}

func GetDictItem[T any](d *kernel.Dict, item string) (T, error) {
	val, exists := (*d)[item]
	if !exists {
		return *new(T), fmt.Errorf("does not contain item '%s'", item)
	}

	var result T
	switch any(result).(type) {
	case int:
		switch v := val.(type) {
		case int:
			result = any(v).(T)
		case int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			result = any(int(reflect.ValueOf(v).Int())).(T)
		case float64:
			result = any(int(v)).(T)
		case string:
			parsed, err := strconv.Atoi(v)
			if err != nil {
				return *new(T), fmt.Errorf("cannot convert %q to int: %v", v, err)
			}
			result = any(parsed).(T)
		default:
			return *new(T), fmt.Errorf("wrong value type for %q: expected int, got %T", item, val)
		}
	case uint:
		switch v := val.(type) {
		case uint:
			result = any(v).(T)
		case int, int8, int16, int32, int64, uint8, uint16, uint32, uint64:
			result = any(uint(reflect.ValueOf(v).Int())).(T)
		case float64:
			result = any(uint(v)).(T)
		case string:
			parsed, err := strconv.ParseUint(v, 0, 0)
			if err != nil {
				return *new(T), fmt.Errorf("cannot convert %q to uint: %v", v, err)
			}
			result = any(uint(parsed)).(T)
		default:
			return *new(T), fmt.Errorf("wrong value type for %q: expected uint, got %T", item, val)
		}
	case string:
		if str, ok := val.(string); ok {
			result = any(str).(T)
		} else {
			return *new(T), fmt.Errorf("wrong value type for '%v': expected string, got %T", item, val)
		}
	default:
		typedVal, ok := val.(T)
		if !ok {
			return *new(T), fmt.Errorf("wrong value type for '%v': real %T, called %T", item, val, new(T))
		}
		result = typedVal
	}
	return result, nil
}

func JsonToStruct(data []byte, s any) error {
	dict := make(kernel.Dict)
	if err := json.Unmarshal(data, &dict); err != nil {
		return fmt.Errorf("failed to parse JSON '%v': %s", string(data), err)
	}
	return DictToStruct(&dict, s)
}

func MapToStruct(m map[string]any, s any) error {
	dict := kernel.Dict(m)
	return DictToStruct(&dict, s)
}

func DictToStruct(dict *kernel.Dict, s any) error {
	val := reflect.ValueOf(s)
	typ := reflect.TypeOf(s)

	// For pointer
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
		typ = typ.Elem()
	}

	// Check struct
	if val.Kind() != reflect.Struct {
		return errors.New("provided value is not a struct")
	}

	// Parse struct
	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		fieldValue := val.Field(i)

		if !fieldValue.CanSet() {
			continue
		}

		// Use tag
		tag := field.Tag.Get("dict")
		if tag == "" {
			tag = field.Tag.Get("json")
			if tag != "" {
				tag = parseTag(tag)
			}
			if tag == "" {
				tag = field.Name
			}
		}

		_, exists := (*dict)[tag]
		if !exists {
			continue
		}

		v, err := getFieldValue(dict, tag, field.Type)
		if err != nil {
			return err
		}

		fieldValue.Set(reflect.ValueOf(v))
	}

	return nil
}

func parseTag(tag string) string {
	parts := strings.Split(tag, ",")
	return parts[0]
}

func getFieldValue(config *kernel.Dict, fieldName string, fieldType reflect.Type) (any, error) {
	switch fieldType.Kind() {
	case reflect.String:
		return GetDictItem[string](config, fieldName)

	case reflect.Int:
		return GetDictItem[int](config, fieldName)

	case reflect.Int64:
		v, err := GetDictItem[int](config, fieldName)
		if err != nil {
			return nil, err
		}
		return int64(v), nil

	case reflect.Uint:
		return GetDictItem[uint](config, fieldName)

	case reflect.Bool:
		return GetDictItem[bool](config, fieldName)

	case reflect.Float64:
		return GetDictItem[float64](config, fieldName)

	case reflect.Slice:
		rawSlice, err := GetDictItem[[]any](config, fieldName)
		if err != nil {
			return nil, err
		}

		sliceType := fieldType.Elem()
		resultSlice := reflect.MakeSlice(reflect.SliceOf(sliceType), 0, len(rawSlice))

		for _, item := range rawSlice {
			elemValue := reflect.New(sliceType).Elem()
			if sliceType.Kind() == reflect.Struct {
				dict, ok := convertToDict(item)
				if !ok {
					return nil, fmt.Errorf("expected Dict for struct slice element, got %T", item)
				}
				err := DictToStruct(&dict, elemValue.Addr().Interface())
				if err != nil {
					return nil, err
				}
			} else {
				elemValue.Set(reflect.ValueOf(item))
			}
			resultSlice = reflect.Append(resultSlice, elemValue)
		}
		return resultSlice.Interface(), nil

	case reflect.Map:
		if fieldType.Key().Kind() != reflect.String {
			return nil, fmt.Errorf("unsupported map key type: %s", fieldType.Key())
		}

		rawMap, err := GetDictItem[map[string]any](config, fieldName)
		if err != nil {
			return nil, err
		}

		mapType := reflect.MapOf(fieldType.Key(), fieldType.Elem())
		resultMap := reflect.MakeMap(mapType)

		for key, value := range rawMap {
			elemValue := reflect.New(fieldType.Elem()).Elem()

			if fieldType.Elem().Kind() == reflect.Struct {
				dict, ok := convertToDict(value)
				if !ok {
					return nil, fmt.Errorf("expected Dict for struct map value, got %T", value)
				}
				err := DictToStruct(&dict, elemValue.Addr().Interface())
				if err != nil {
					return nil, err
				}
			} else {
				elemValue.Set(reflect.ValueOf(value))
			}

			resultMap.SetMapIndex(reflect.ValueOf(key), elemValue)
		}
		return resultMap.Interface(), nil

	case reflect.Struct:
		dict, err := GetDictItem[kernel.Dict](config, fieldName)
		if err != nil {
			d, err := GetDictItem[kernel.Config](config, fieldName)
			if err != nil {
				return nil, err
			}
			dict = d.ToDict()
		}
		ptr := reflect.New(fieldType)
		err = DictToStruct(&dict, ptr.Interface())
		if err != nil {
			return nil, err
		}
		return ptr.Elem().Interface(), nil

	default:
		return nil, fmt.Errorf("unsupported field type: %s", fieldType)
	}
}

func convertToDict(item any) (kernel.Dict, bool) {
	switch v := item.(type) {
	case kernel.Dict:
		return v, true
	case map[string]any:
		dict := kernel.Dict(v)
		return dict, true
	default:
		val := reflect.ValueOf(item)
		method := val.MethodByName("ToDict")
		if method.IsValid() && method.Type().NumOut() == 1 {
			result := method.Call(nil)
			if converted, ok := result[0].Interface().(kernel.Dict); ok {
				return converted, true
			}
		}
		method = val.MethodByName("ToMap")
		if method.IsValid() && method.Type().NumOut() == 1 {
			result := method.Call(nil)
			if converted, ok := result[0].Interface().(map[string]any); ok {
				dict := kernel.Dict(converted)
				return dict, true
			}
		}
		return nil, false
	}
}
