package http

import (
	"reflect"
	"testing"

	"github.com/epicoon/lxgo/kernel"
)

type testForm struct {
	*Form
	Name string `json:"name"`
	Age  int    `dict:"age" json:"ignored_in_favor_of_dict_tag"`
}

func newTestForm() *testForm {
	return &testForm{Form: NewForm()}
}

func TestFormFiller_Fill_Errors(t *testing.T) {
	t.Run("no_form_set", func(t *testing.T) {
		err := FormFiller().SetDict(kernel.Dict{}).Fill()
		if err == nil {
			t.Fatal("expected error, got none")
		}
	})

	t.Run("no_data_source_set", func(t *testing.T) {
		err := FormFiller().SetForm(newTestForm()).Fill()
		if err == nil {
			t.Fatal("expected error, got none")
		}
	})

	t.Run("both_context_and_dict_set", func(t *testing.T) {
		ctx := NewHandleContext(nil, "/", nil)
		err := FormFiller().SetForm(newTestForm()).SetContext(ctx).SetDict(kernel.Dict{}).Fill()
		if err == nil {
			t.Fatal("expected error, got none")
		}
	})
}

func TestFormFiller_Fill_Dict(t *testing.T) {
	f := newTestForm()
	err := FormFiller().SetForm(f).SetDict(kernel.Dict{"name": "Alice", "age": "30"}).Fill()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.HasErrors() {
		t.Fatalf("unexpected form errors: %v", f.GetFirstError())
	}
	if f.Name != "Alice" || f.Age != 30 {
		t.Fatalf("got %+v, want Name=Alice Age=30", f)
	}
}

func TestCheckMissingParams(t *testing.T) {
	t.Run("required_field_present_in_data", func(t *testing.T) {
		f := newTestForm()
		f.SetRequired([]string{"name"})
		checkMissingParams(f, kernel.Dict{"name": "Alice"})
		if f.HasErrors() {
			t.Fatalf("unexpected errors: %v", f.GetFirstError())
		}
	})

	t.Run("required_field_missing_and_zero", func(t *testing.T) {
		f := newTestForm()
		f.SetRequired([]string{"name"})
		checkMissingParams(f, kernel.Dict{})
		if !f.HasErrors() {
			t.Fatal("expected a missing-required-parameter error, got none")
		}
	})

	t.Run("required_field_missing_but_already_nonzero", func(t *testing.T) {
		f := newTestForm()
		f.Name = "preset"
		f.SetRequired([]string{"name"})
		checkMissingParams(f, kernel.Dict{})
		if f.HasErrors() {
			t.Fatalf("expected no error (field already has a non-zero value), got: %v", f.GetFirstError())
		}
	})

	t.Run("no_required_fields_is_a_no_op", func(t *testing.T) {
		f := newTestForm()
		checkMissingParams(f, kernel.Dict{})
		if f.HasErrors() {
			t.Fatalf("unexpected errors: %v", f.GetFirstError())
		}
	})
}

func TestBuildFieldMap(t *testing.T) {
	type embedded struct {
		Nested string `json:"nested"`
	}
	type withEmbed struct {
		embedded
		DictTagged string `dict:"dict_name" json:"json_name"`
		JSONOnly   string `json:"json_only"`
		PlainName  string
		Skipped    string `json:"-"`
	}

	m := buildFieldMap(reflect.ValueOf(&withEmbed{}))

	cases := map[string]bool{
		"nested":    true, // flattened from the anonymous embedded field
		"dict_name": true, // dict tag takes precedence over json tag
		"json_only": true,
		"PlainName": true, // no tag at all - falls back to field name
		"json_name": false,
		"Skipped":   false,
	}
	for key, wantPresent := range cases {
		_, present := m[key]
		if present != wantPresent {
			t.Errorf("key %q: present=%v, want %v (map: %v)", key, present, wantPresent, keysOf(m))
		}
	}
}

func keysOf(m map[string]reflect.Value) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func TestIsZeroValue(t *testing.T) {
	type sample struct {
		Str   string
		Num   int
		Slice []string
		Ptr   *int
		Iface any
	}
	var s sample
	v := reflect.ValueOf(&s).Elem()

	cases := []struct {
		name string
		want bool
	}{
		{"Str", true},
		{"Num", true},
		{"Slice", true},
		{"Ptr", true},
		{"Iface", true},
	}
	for _, tc := range cases {
		if got := isZeroValue(v.FieldByName(tc.name)); got != tc.want {
			t.Errorf("isZeroValue(%s) = %v, want %v", tc.name, got, tc.want)
		}
	}

	s.Str = "x"
	s.Num = 1
	s.Slice = []string{"a"}
	n := 5
	s.Ptr = &n
	s.Iface = "set"
	for _, tc := range cases {
		if got := isZeroValue(v.FieldByName(tc.name)); got {
			t.Errorf("isZeroValue(%s) after setting = true, want false", tc.name)
		}
	}

	t.Run("invalid_value", func(t *testing.T) {
		if isZeroValue(reflect.Value{}) {
			t.Fatal("expected false for an invalid reflect.Value")
		}
	})
}
