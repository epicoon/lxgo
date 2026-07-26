package cast_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/cast"
)

// toMapper is a stand-in for any type with a `ToMap() map[string]any` method
// that isn't itself a map (exercises toStringMap's reflective method
// fallback, not its generic map-kind branch).
type toMapper struct {
	data map[string]any
}

func (m toMapper) ToMap() map[string]any {
	return m.data
}

func TestValue_Numeric(t *testing.T) {
	cases := []struct {
		name   string
		v      any
		target reflect.Type
		want   any
	}{
		{"int<-string", "42", reflect.TypeOf(0), 42},
		{"int<-int64", int64(7), reflect.TypeOf(0), 7},
		{"int<-float64", 3.9, reflect.TypeOf(0), 3},
		{"int<-uint", uint(5), reflect.TypeOf(0), 5},
		{"int64<-int", 42, reflect.TypeOf(int64(0)), int64(42)},
		{"uint<-string", "42", reflect.TypeOf(uint(0)), uint(42)},
		{"uint<-int", 7, reflect.TypeOf(uint(0)), uint(7)},
		{"uint8<-int", 200, reflect.TypeOf(uint8(0)), uint8(200)},
		// Regression: GetDictItem used to lack this case entirely and fail.
		{"float64<-string", "3.14", reflect.TypeOf(float64(0)), 3.14},
		{"float64<-int", 3, reflect.TypeOf(float64(0)), float64(3)},
		{"float64<-float64_passthrough", 1.5, reflect.TypeOf(float64(0)), 1.5},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := cast.Value(tc.v, tc.target)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %#v (%T), want %#v (%T)", got, got, tc.want, tc.want)
			}
		})
	}
}

func TestValue_NumericErrors(t *testing.T) {
	cases := []struct {
		name   string
		v      any
		target reflect.Type
	}{
		{"int<-non_numeric_string", "abc", reflect.TypeOf(0)},
		{"int<-bool", true, reflect.TypeOf(0)},
		{"uint<-negative_string", "-1", reflect.TypeOf(uint(0))},
		{"float64<-non_numeric_string", "abc", reflect.TypeOf(float64(0))},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := cast.Value(tc.v, tc.target); err == nil {
				t.Fatalf("expected error, got none")
			}
		})
	}
}

func TestValue_Bool(t *testing.T) {
	t.Run("bool<-bool", func(t *testing.T) {
		got, err := cast.Value(true, reflect.TypeOf(false))
		if err != nil || got != true {
			t.Fatalf("got %#v, err %v", got, err)
		}
	})

	t.Run("bool<-string_true", func(t *testing.T) {
		got, err := cast.Value("true", reflect.TypeOf(false))
		if err != nil || got != true {
			t.Fatalf("got %#v, err %v", got, err)
		}
	})

	t.Run("bool<-string_false", func(t *testing.T) {
		got, err := cast.Value("false", reflect.TypeOf(false))
		if err != nil || got != false {
			t.Fatalf("got %#v, err %v", got, err)
		}
	})

	t.Run("bool<-invalid_string", func(t *testing.T) {
		if _, err := cast.Value("nope", reflect.TypeOf(false)); err == nil {
			t.Fatalf("expected error, got none")
		}
	})

	t.Run("bool<-int", func(t *testing.T) {
		if _, err := cast.Value(1, reflect.TypeOf(false)); err == nil {
			t.Fatalf("expected error, got none")
		}
	})
}

func TestValue_String(t *testing.T) {
	cases := []struct {
		name string
		v    any
		want string
	}{
		{"string<-string", "hi", "hi"},
		// Regression: setFieldValue used to hit Go's legal-but-wrong
		// int->string rune conversion (42 -> "*") via reflect.ConvertibleTo
		// before ever reaching a string-specific branch.
		{"string<-int", 42, "42"},
		{"string<-int64", int64(-7), "-7"},
		{"string<-float64", 3.5, "3.5"},
		{"string<-bool", true, "true"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := cast.Value(tc.v, reflect.TypeOf(""))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestValue_Slice(t *testing.T) {
	t.Run("[]int<-[]any", func(t *testing.T) {
		got, err := cast.Value([]any{"1", 2, 3.0}, reflect.TypeOf([]int{}))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := []int{1, 2, 3}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %#v, want %#v", got, want)
		}
	})

	t.Run("[]string<-[]any_numeric", func(t *testing.T) {
		got, err := cast.Value([]any{1, 2}, reflect.TypeOf([]string{}))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := []string{"1", "2"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %#v, want %#v", got, want)
		}
	})

	t.Run("element_error_propagates", func(t *testing.T) {
		if _, err := cast.Value([]any{"1", "not-a-number"}, reflect.TypeOf([]int{})); err == nil {
			t.Fatalf("expected error, got none")
		}
	})

	t.Run("non_slice_source", func(t *testing.T) {
		if _, err := cast.Value(42, reflect.TypeOf([]int{})); err == nil {
			t.Fatalf("expected error, got none")
		}
	})
}

func TestValue_Map(t *testing.T) {
	t.Run("map[string]int<-map[string]any", func(t *testing.T) {
		got, err := cast.Value(map[string]any{"a": "1", "b": 2}, reflect.TypeOf(map[string]int{}))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := map[string]int{"a": 1, "b": 2}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %#v, want %#v", got, want)
		}
	})

	t.Run("map[string]any<-kernel.Dict", func(t *testing.T) {
		d := kernel.Dict{"a": 1}
		got, err := cast.Value(d, reflect.TypeOf(map[string]any{}))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := map[string]any{"a": 1}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %#v, want %#v", got, want)
		}
	})

	t.Run("map[string]any<-custom_ToMap", func(t *testing.T) {
		got, err := cast.Value(toMapper{data: map[string]any{"a": 1}}, reflect.TypeOf(map[string]any{}))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := map[string]any{"a": 1}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %#v, want %#v", got, want)
		}
	})

	t.Run("unsupported_key_type", func(t *testing.T) {
		if _, err := cast.Value(map[string]any{}, reflect.TypeOf(map[int]any{})); err == nil {
			t.Fatalf("expected error, got none")
		}
	})

	t.Run("element_error_propagates", func(t *testing.T) {
		if _, err := cast.Value(map[string]any{"a": "not-a-number"}, reflect.TypeOf(map[string]int{})); err == nil {
			t.Fatalf("expected error, got none")
		}
	})
}

func TestValue_Interface(t *testing.T) {
	t.Run("any_passthrough", func(t *testing.T) {
		var target any
		got, err := cast.Value(42, reflect.TypeOf(&target).Elem())
		if err != nil || got != 42 {
			t.Fatalf("got %#v, err %v", got, err)
		}
	})

	t.Run("implements_interface", func(t *testing.T) {
		got, err := cast.Value(errors.New("boom"), reflect.TypeOf((*error)(nil)).Elem())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, ok := got.(error); !ok {
			t.Fatalf("got %#v, want error", got)
		}
	})

	t.Run("does_not_implement_interface", func(t *testing.T) {
		if _, err := cast.Value(42, reflect.TypeOf((*error)(nil)).Elem()); err == nil {
			t.Fatalf("expected error, got none")
		}
	})
}

func TestValue_Nil(t *testing.T) {
	t.Run("nil<-int_target", func(t *testing.T) {
		got, err := cast.Value(nil, reflect.TypeOf(0))
		if err != nil || got != 0 {
			t.Fatalf("got %#v, err %v", got, err)
		}
	})

	t.Run("nil<-slice_target", func(t *testing.T) {
		got, err := cast.Value(nil, reflect.TypeOf([]int{}))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.([]int) != nil {
			t.Fatalf("got %#v, want nil slice", got)
		}
	})
}

func TestValue_UnsupportedTarget(t *testing.T) {
	if _, err := cast.Value(42, reflect.TypeOf(make(chan int))); err == nil {
		t.Fatalf("expected error, got none")
	}
}

func TestValue_AssignablePassthrough(t *testing.T) {
	got, err := cast.Value(5, reflect.TypeOf(0))
	if err != nil || got != 5 {
		t.Fatalf("got %#v, err %v", got, err)
	}
}

func TestTo(t *testing.T) {
	t.Run("float64<-string", func(t *testing.T) {
		got, err := cast.To[float64]("3.14")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != 3.14 {
			t.Fatalf("got %v, want 3.14", got)
		}
	})

	t.Run("string<-int", func(t *testing.T) {
		got, err := cast.To[string](42)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "42" {
			t.Fatalf("got %q, want %q", got, "42")
		}
	})

	t.Run("error_propagates", func(t *testing.T) {
		if _, err := cast.To[int]("abc"); err == nil {
			t.Fatalf("expected error, got none")
		}
	})

	t.Run("any_target", func(t *testing.T) {
		got, err := cast.To[any]("hi")
		if err != nil || got != "hi" {
			t.Fatalf("got %#v, err %v", got, err)
		}
	})
}

type connConfig struct {
	Host string
	Port int
}

func TestValue_Struct(t *testing.T) {
	t.Run("struct<-kernel.Dict", func(t *testing.T) {
		got, err := cast.Value(kernel.Dict{"Host": "localhost", "Port": "5432"}, reflect.TypeOf(connConfig{}))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := connConfig{Host: "localhost", Port: 5432}
		if got != want {
			t.Fatalf("got %#v, want %#v", got, want)
		}
	})

	t.Run("[]struct<-[]any_of_dict", func(t *testing.T) {
		got, err := cast.Value(
			[]any{kernel.Dict{"Host": "a", "Port": 1}, kernel.Dict{"Host": "b", "Port": 2}},
			reflect.TypeOf([]connConfig{}),
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := []connConfig{{Host: "a", Port: 1}, {Host: "b", Port: 2}}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %#v, want %#v", got, want)
		}
	})

	t.Run("non_dict_source", func(t *testing.T) {
		if _, err := cast.Value(42, reflect.TypeOf(connConfig{})); err == nil {
			t.Fatalf("expected error, got none")
		}
	})
}

func TestValue_MapFromStruct(t *testing.T) {
	t.Run("map[string]any<-struct", func(t *testing.T) {
		got, err := cast.Value(connConfig{Host: "localhost", Port: 5432}, reflect.TypeOf(map[string]any{}))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := map[string]any{"Host": "localhost", "Port": 5432}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %#v, want %#v", got, want)
		}
	})

	t.Run("map[string]any<-pointer_to_struct", func(t *testing.T) {
		got, err := cast.Value(&connConfig{Host: "a", Port: 1}, reflect.TypeOf(map[string]any{}))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := map[string]any{"Host": "a", "Port": 1}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %#v, want %#v", got, want)
		}
	})

	t.Run("map[string]string<-struct_coerces_values", func(t *testing.T) {
		got, err := cast.Value(connConfig{Host: "a", Port: 1}, reflect.TypeOf(map[string]string{}))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := map[string]string{"Host": "a", "Port": "1"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %#v, want %#v", got, want)
		}
	})

	t.Run("json_tag_used_as_key", func(t *testing.T) {
		type withJSONTag struct {
			Active bool `json:"Active"`
		}
		got, err := cast.Value(withJSONTag{Active: true}, reflect.TypeOf(map[string]any{}))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := map[string]any{"Active": true}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %#v, want %#v", got, want)
		}
	})

	t.Run("round_trip_struct_to_map_to_struct", func(t *testing.T) {
		orig := connConfig{Host: "roundtrip", Port: 9}
		m, err := cast.To[map[string]any](orig)
		if err != nil {
			t.Fatalf("unexpected error converting to map: %v", err)
		}
		var back connConfig
		if err := cast.DictToStruct(kernel.Dict(m), &back); err != nil {
			t.Fatalf("unexpected error converting back to struct: %v", err)
		}
		if back != orig {
			t.Fatalf("got %#v, want %#v", back, orig)
		}
	})
}

func TestDictToStruct(t *testing.T) {
	t.Run("plain_field_name_no_tag", func(t *testing.T) {
		// Regression: FieldName used to return "" whenever a field had
		// no explicit `dict:"..."` tag, so plain-field-name structs (the
		// common case) silently never got populated at all.
		var cc connConfig
		err := cast.DictToStruct(kernel.Dict{"Host": "localhost", "Port": "5432"}, &cc)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := connConfig{Host: "localhost", Port: 5432}
		if cc != want {
			t.Fatalf("got %#v, want %#v", cc, want)
		}
	})

	t.Run("json_tag_fallback", func(t *testing.T) {
		type withJSONTag struct {
			Active bool `json:"Active"`
		}
		var v withJSONTag
		err := cast.DictToStruct(kernel.Dict{"Active": "true"}, &v)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !v.Active {
			t.Fatalf("got %#v, want Active=true", v)
		}
	})

	t.Run("nested_struct_and_slice_of_struct", func(t *testing.T) {
		// Regression: Value used to have no case for reflect.Struct at all,
		// so nested structs and slices/maps of structs came back empty.
		type nested struct {
			Inner connConfig
			List  []connConfig
		}
		var n nested
		err := cast.DictToStruct(kernel.Dict{
			"Inner": kernel.Dict{"Host": "a", "Port": 1},
			"List":  []any{kernel.Dict{"Host": "b", "Port": 2}},
		}, &n)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := nested{Inner: connConfig{Host: "a", Port: 1}, List: []connConfig{{Host: "b", Port: 2}}}
		if !reflect.DeepEqual(n, want) {
			t.Fatalf("got %#v, want %#v", n, want)
		}
	})

	t.Run("missing_key_leaves_field_unset", func(t *testing.T) {
		v := connConfig{Host: "kept"}
		if err := cast.DictToStruct(kernel.Dict{}, &v); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v.Host != "kept" {
			t.Fatalf("got %#v, want Host to stay 'kept'", v)
		}
	})
}
