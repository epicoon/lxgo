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

var formInterfaceType = reflect.TypeOf((*kernel.IForm)(nil)).Elem()

// DictToStruct populates struct s's fields from dict, matching each field
// by its "dict" tag, then "json" tag, then field name; recurses into
// nested structs/slices/maps, coercing mismatched-but-compatible value
// types along the way (numeric strings, etc.). s must be a struct or a
// pointer to one.
//
// An anonymous (embedded) field's own fields are populated from the SAME
// dict, at the same level, not from a nested value under the embedded
// type's name - mirroring Go's own field promotion for embedding (and
// matching lxgo-kernel/http's buildFieldMap, which already flattened
// anonymous fields this way; this used to be the one place in the form-
// filling pipeline that didn't, so a required field declared inside an
// embedded (not just the top-level *Form) struct would validate as
// present but never actually get filled).
//
// A NAMED field that is itself a kernel.IForm (a form nested inside
// another form, as opposed to embedded) is left untouched here -
// populating it via plain field coercion would replace an
// already-constructed nested form with a fresh, unconstructed zero value,
// discarding its embedded ErrorsCollector. Filling and validating a nested
// form is more than field coercion (its own required-fields check,
// AfterFill, Validate, error aggregation) and is handled by whoever owns
// that lifecycle (see lxgo-kernel/http's fillNestedForms).
func DictToStruct(dict kernel.IDict, s any) error {
	val := reflect.ValueOf(s)

	// For pointer
	if val.Kind() == reflect.Pointer {
		val = val.Elem()
	}

	// Check struct
	if val.Kind() != reflect.Struct {
		return errors.New("provided value is not a struct")
	}

	return dictToStructValue(dict, val)
}

func dictToStructValue(dict kernel.IDict, val reflect.Value) error {
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		fieldValue := val.Field(i)

		if !fieldValue.CanSet() {
			continue
		}

		if field.Anonymous {
			ev := fieldValue
			if ev.Kind() == reflect.Pointer {
				if ev.IsNil() {
					continue
				}
				ev = ev.Elem()
			}
			if ev.Kind() == reflect.Struct {
				if err := dictToStructValue(dict, ev); err != nil {
					return err
				}
			}
			continue
		}

		if isFormField(fieldValue) {
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

// isFormField reports whether fieldValue's address (or fieldValue itself,
// if it's already a pointer) implements kernel.IForm - kernel.IForm's
// methods are all pointer-receiver in the base Form, so a value-typed
// field needs Addr() to reach them.
func isFormField(fieldValue reflect.Value) bool {
	if fieldValue.Kind() == reflect.Pointer {
		return !fieldValue.IsNil() && fieldValue.Type().Implements(formInterfaceType)
	}
	return fieldValue.CanAddr() && reflect.PointerTo(fieldValue.Type()).Implements(formInterfaceType)
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
