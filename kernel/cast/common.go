package cast

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/epicoon/lxgo/kernel"
)

// JsonToStruct unmarshals JSON data into a kernel.Dict and populates struct s from it - see DictToStruct.
func JsonToStruct(data []byte, s any) error {
	dict := make(kernel.Dict)
	if err := json.Unmarshal(data, &dict); err != nil {
		return fmt.Errorf("failed to parse JSON '%v': %s", string(data), err)
	}
	return DictToStruct(&dict, s)
}

// MapToStruct populates struct s from map m - see DictToStruct.
func MapToStruct(m map[string]any, s any) error {
	dict := kernel.Dict(m)
	return DictToStruct(&dict, s)
}

// DictToStruct populates struct s's fields from dict, matching each field
// by its "dict" tag, then "json" tag, then field name; recurses into
// nested structs/slices/maps, coercing mismatched-but-compatible value
// types along the way (numeric strings, etc.). s must be a struct or a
// pointer to one.
func DictToStruct(dict kernel.IDict, s any) error {
	val := reflect.ValueOf(s)
	typ := reflect.TypeOf(s)

	// For pointer
	if val.Kind() == reflect.Pointer {
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

		// Define field name
		fName := FieldName(field)

		if !dict.Has(fName) {
			continue
		}

		v, err := Value(dict.Get(fName), field.Type)
		if err != nil {
			return err
		}

		fieldValue.Set(reflect.ValueOf(v))
	}

	return nil
}

// FieldName resolves the dict/JSON key a struct field is populated from and
// read back into - its "dict" tag if set, else its "json" tag (stripped of
// options), else the field's own name.
func FieldName(field reflect.StructField) string {
	if tag := field.Tag.Get("dict"); tag != "" {
		return tag
	}

	if tag := field.Tag.Get("json"); tag != "" {
		if name := strings.Split(tag, ",")[0]; name != "" {
			return name
		}
	}

	return field.Name
}
