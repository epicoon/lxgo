package http

import (
	"reflect"

	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/errors"
)

/** @interface kernel.IForm */
type Form struct {
	*errors.ErrorsCollector
	required []string
}

var _ kernel.IForm = (*Form)(nil)

func PrepareForm(f kernel.IForm) kernel.IForm {
	configureForm(f, f.Config())
	return f
}

func FormToMap(f kernel.IForm) map[string]any {
	result := make(map[string]any)

	val := reflect.ValueOf(f)
	if val.Kind() == reflect.Ptr {
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
func NewForm() *Form {
	return &Form{
		ErrorsCollector: errors.NewErrorsCollector(),
		required:        []string{},
	}
}

func (f *Form) Config() kernel.FormConfig {
	return *new(kernel.FormConfig)
}

func (f *Form) SetRequired(required []string) {
	f.required = required
}

func (f *Form) Required() []string {
	return f.required
}

/** @abstract */
func (f *Form) Fill(d *kernel.Dict) error {
	// Pass
	return nil
}

/** @abstract */
func (f *Form) AfterFill() {
	// Pass
}

/** @abstract */
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
