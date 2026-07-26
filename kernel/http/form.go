package http

import (
	"reflect"

	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/errors"
)

/** @interface kernel.IForm */

// Form is the base kernel.IForm implementation - embed it in your own form
// struct and override at least Validate (and Config, to declare required
// fields). Fields are populated automatically (via cast.DictToStruct,
// matched by "dict"/"json" tag or field name).
type Form struct {
	*errors.ErrorsCollector
	required []string
}

var _ kernel.IForm = (*Form)(nil)

// PrepareForm sets f's required fields from its own Config - call this
// right after constructing a form (see the CForm pattern in
// kernel.HttpResourceConfig).
func PrepareForm(f kernel.IForm) kernel.IForm {
	configureForm(f, f.Config())
	return f
}

// FormToMap renders form f's fields as a map[string]any, keyed by their
// "json" tag (skipping fields with no tag or a "-" tag).
func FormToMap(f kernel.IForm) map[string]any {
	result := make(map[string]any)

	val := reflect.ValueOf(f)
	if val.Kind() == reflect.Pointer {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return result
	}

	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		value := val.Field(i)

		tag := field.Tag.Get("json")
		if tag == "" {
			continue
		}
		if tag == "-" {
			continue
		}

		jsonKey := tag
		if comma := len(tag); comma > 0 {
			for j, c := range tag {
				if c == ',' {
					jsonKey = tag[:j]
					break
				}
			}
		}

		if !value.CanInterface() {
			continue
		}

		result[jsonKey] = value.Interface()
	}

	return result
}

/** @constructor */

// NewForm constructs an empty Form with no required fields.
func NewForm() *Form {
	return &Form{
		ErrorsCollector: errors.NewErrorsCollector(),
		required:        []string{},
	}
}

// Config returns an empty FormConfig - override this to declare the form's fields.
func (f *Form) Config() kernel.FormConfig {
	return *new(kernel.FormConfig)
}

// SetRequired overrides which fields are required.
func (f *Form) SetRequired(required []string) {
	f.required = required
}

// Required returns the currently required fields.
func (f *Form) Required() []string {
	return f.required
}

/** @abstract */

// AfterFill is a no-op - override it for cross-field logic after the form's
// fields have been populated.
func (f *Form) AfterFill() {
	// Pass
}

/** @abstract */

// Validate returns true - override it to validate the filled form.
func (f *Form) Validate() bool {
	// Pass
	return true
}

func configureForm(f kernel.IForm, conf kernel.FormConfig) {
	required := make([]string, 0, len(conf))
	for fName, fConf := range conf {
		if fConf.Required {
			required = append(required, fName)
		}
	}
	if len(required) > 0 {
		f.SetRequired(required)
	}
}
