package http

import (
	"reflect"
	"strings"
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

// EmbeddedFields is a stand-in for a real embedded (not nested-as-a-form)
// struct with its own data field - exported, since Go reflection taints an
// entire subtree read-only once it passes through an unexported
// field/type (real forms only ever embed exported types like *Form
// anyway).
type EmbeddedFields struct {
	Nested string `json:"nested"`
}

type formWithEmbeddedFields struct {
	*Form
	EmbeddedFields
	Name string `json:"name"`
}

func newFormWithEmbeddedFields() *formWithEmbeddedFields {
	return &formWithEmbeddedFields{Form: NewForm()}
}

// nestedTestForm is a small form meant to be nested (as a named field, not
// embedded) inside another form - see the outer*TestForm types below.
type nestedTestForm struct {
	*Form
	Value string `json:"value"`
}

func newNestedTestForm() *nestedTestForm {
	f := &nestedTestForm{Form: NewForm()}
	f.SetRequired([]string{"value"})
	return f
}

func (f *nestedTestForm) Validate() bool {
	if f.Value == "forbidden" {
		f.CollectErrorf("value must not be 'forbidden'")
		return false
	}
	return true
}

type outerNestedTestForm struct {
	*Form
	Name   string         `json:"name"`
	Nested nestedTestForm `json:"nested"`
}

func newOuterNestedTestForm() *outerNestedTestForm {
	return &outerNestedTestForm{
		Form:   NewForm(),
		Nested: *newNestedTestForm(),
	}
}

// deepOuterTestForm nests outerNestedTestForm itself (two levels of named
// nested-form fields), to check that error messages carry the full dotted
// path down to whichever level actually failed.
type deepOuterTestForm struct {
	*Form
	Mid outerNestedTestForm `json:"mid"`
}

func newDeepOuterTestForm() *deepOuterTestForm {
	return &deepOuterTestForm{
		Form: NewForm(),
		Mid:  *newOuterNestedTestForm(),
	}
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

// TestFormFiller_Fill_Dict_EmbeddedField is a regression test for the
// buildFieldMap/cast.DictToStruct divergence: buildFieldMap already
// flattened anonymous (embedded) fields into the form's own namespace, so
// checkMissingParams was perfectly happy accepting a required field
// declared on an embedded struct as "present" - but cast.DictToStruct
// looked for a dict key matching the embedded field's own type name
// (which real request data never sends), so the field was validated as
// fine yet never actually got filled. Both now agree.
func TestFormFiller_Fill_Dict_EmbeddedField(t *testing.T) {
	f := newFormWithEmbeddedFields()
	f.SetRequired([]string{"nested"})
	err := FormFiller().SetForm(f).SetDict(kernel.Dict{"name": "Alice", "nested": "value"}).Fill()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.HasErrors() {
		t.Fatalf("unexpected form errors: %v", f.GetFirstError())
	}
	if f.Name != "Alice" || f.Nested != "value" {
		t.Fatalf("got %+v, want Name=Alice Nested=value", f)
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

func TestFillNestedForms_ValidNestedForm(t *testing.T) {
	f := newOuterNestedTestForm()
	err := FormFiller().SetForm(f).SetDict(kernel.Dict{
		"name":   "Alice",
		"nested": kernel.Dict{"value": "ok"},
	}).Fill()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.HasErrors() {
		t.Fatalf("unexpected form errors: %v", f.GetFirstError())
	}
	if f.Name != "Alice" || f.Nested.Value != "ok" {
		t.Fatalf("got %+v, want Name=Alice Nested.Value=ok", f)
	}
}

// TestFillNestedForms_NestedRequiredFieldMissing is a regression test: a
// nested form's own required fields used to never be consulted at all -
// cast.DictToStruct only fills fields, it doesn't know about
// kernel.IForm's required-fields contract, and nothing else walked nested
// form fields either.
func TestFillNestedForms_NestedRequiredFieldMissing(t *testing.T) {
	f := newOuterNestedTestForm()
	err := FormFiller().SetForm(f).SetDict(kernel.Dict{
		"name":   "Alice",
		"nested": kernel.Dict{},
	}).Fill()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !f.HasErrors() {
		t.Fatal("expected an error for the nested form's own missing required field")
	}
	msg := f.GetFirstError().Error()
	if !strings.Contains(msg, "nested") || !strings.Contains(msg, "value") {
		t.Fatalf("expected the error to mention the nested field's path and its own missing param, got %q", msg)
	}
}

// TestFillNestedForms_NestedValidateFails is a regression test: a nested
// form's own Validate() used to never be called, so it could never reject
// the parent form.
func TestFillNestedForms_NestedValidateFails(t *testing.T) {
	f := newOuterNestedTestForm()
	err := FormFiller().SetForm(f).SetDict(kernel.Dict{
		"name":   "Alice",
		"nested": kernel.Dict{"value": "forbidden"},
	}).Fill()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !f.HasErrors() {
		t.Fatal("expected the nested form's Validate() failure to propagate to the parent")
	}
	msg := f.GetFirstError().Error()
	if !strings.Contains(msg, "nested") || !strings.Contains(msg, "forbidden") {
		t.Fatalf("expected the error to mention the nested field's path and its own message, got %q", msg)
	}
}

// TestFillNestedForms_BlockAbsentAndNotRequired checks that a nested-form
// field with no corresponding data, and not itself required at the outer
// level, is silently left alone - its own required fields aren't enforced
// against data that was never given at all.
func TestFillNestedForms_BlockAbsentAndNotRequired(t *testing.T) {
	f := newOuterNestedTestForm()
	err := FormFiller().SetForm(f).SetDict(kernel.Dict{"name": "Alice"}).Fill()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.HasErrors() {
		t.Fatalf("unexpected form errors: %v", f.GetFirstError())
	}
	if f.Nested.Value != "" {
		t.Fatalf("expected the untouched nested form to stay zero-valued, got %q", f.Nested.Value)
	}
}

// TestFillNestedForms_BlockRequiredButAbsent checks that requiring the
// nested-form field itself (as a whole block) is enforced by the existing,
// generic checkMissingParams on the outer form - before fillNestedForms
// ever runs.
func TestFillNestedForms_BlockRequiredButAbsent(t *testing.T) {
	f := newOuterNestedTestForm()
	f.SetRequired([]string{"nested"})
	err := FormFiller().SetForm(f).SetDict(kernel.Dict{"name": "Alice"}).Fill()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !f.HasErrors() {
		t.Fatal("expected an error for the missing required nested block")
	}
}

// TestFillNestedForms_DeepNesting_ErrorPathIsFullyDotted checks that an
// error from a form nested two levels deep carries the full dotted path
// (not just the innermost field name), so it's clear which nested block
// actually failed.
func TestFillNestedForms_DeepNesting_ErrorPathIsFullyDotted(t *testing.T) {
	f := newDeepOuterTestForm()
	err := FormFiller().SetForm(f).SetDict(kernel.Dict{
		"mid": kernel.Dict{
			"name":   "Alice",
			"nested": kernel.Dict{"value": "forbidden"},
		},
	}).Fill()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !f.HasErrors() {
		t.Fatal("expected the doubly-nested form's Validate() failure to propagate all the way up")
	}
	msg := f.GetFirstError().Error()
	if !strings.Contains(msg, "mid.nested") {
		t.Fatalf("expected the full dotted path 'mid.nested' in the error, got %q", msg)
	}
}
