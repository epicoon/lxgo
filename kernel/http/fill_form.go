package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"

	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/cast"
)

/** @interface kernel.IFormFiller */

type formFiller struct {
	form kernel.IForm
	ctx  kernel.IHandleContext
	dict kernel.Dict
}

var _ kernel.IFormFiller = (*formFiller)(nil)

// FormFiller starts a fluent form-filling call: chain SetForm with either
// SetContext (fill from an HTTP request, GET query or JSON/urlencoded body)
// or SetDict (fill from an already-parsed kernel.Dict), then call Fill.
func FormFiller() kernel.IFormFiller {
	return &formFiller{}
}

// SetForm sets the form to fill.
func (ff *formFiller) SetForm(f kernel.IForm) kernel.IFormFiller {
	ff.form = f
	return ff
}

// SetContext sets the request context to fill the form from (an HTTP
// request's GET query or JSON/urlencoded body) - mutually exclusive with SetDict.
func (ff *formFiller) SetContext(ctx kernel.IHandleContext) kernel.IFormFiller {
	ff.ctx = ctx
	return ff
}

// SetDict sets an already-parsed kernel.Dict to fill the form from -
// mutually exclusive with SetContext.
func (ff *formFiller) SetDict(d kernel.Dict) kernel.IFormFiller {
	ff.dict = d
	return ff
}

// Fill fills the form from whichever of SetContext/SetDict was called,
// returning an error (without touching the form) if SetForm wasn't called,
// if neither SetContext nor SetDict was, or was called both.
func (ff *formFiller) Fill() error {
	if ff.form == nil {
		return errors.New("form filler: no form set, call SetForm first")
	}
	if ff.ctx == nil && ff.dict == nil {
		return errors.New("form filler: no data source set, call SetContext or SetDict first")
	}
	if ff.ctx != nil && ff.dict != nil {
		return errors.New("form filler: context and dict are both presented, it's not clear what to choose")
	}

	if ff.ctx != nil {
		fillFormByHandleContext(ff.form, ff.ctx)
	} else if ff.dict != nil {
		fillFormByDict(ff.form, ff.dict)
	}
	return nil
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * PRIVATE
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

func fillFormByHandleContext(f kernel.IForm, ctx kernel.IHandleContext) {
	r := ctx.Request()

	// GET-requests
	if r.Method == http.MethodGet {
		fillGetParams(f, r)
		if !f.HasErrors() {
			f.AfterFill()
		}
		return
	}

	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "plane/text"
	}

	if strings.HasPrefix(contentType, "application/json") {
		parseJSON(f, r)
	} else if strings.HasPrefix(contentType, "application/x-www-form-urlencoded") {
		parseForm(f, r)
	}
	//TODO more variants?

	if !f.HasErrors() {
		f.AfterFill()
	}
}

func fillFormByDict(f kernel.IForm, dict kernel.Dict) {
	checkMissingParams(f, dict)
	if f.HasErrors() {
		return
	}

	if err := cast.DictToStruct(&dict, f); err != nil {
		f.CollectErrorf(err.Error())
		return
	}

	fillNestedForms(f, dict)
}

var formInterfaceType = reflect.TypeOf((*kernel.IForm)(nil)).Elem()

// fillNestedForms finds every named (non-anonymous) field of f that is
// itself a kernel.IForm - a form nested inside another form, as opposed to
// an anonymous/embedded one (buildFieldMap already flattens those into f's
// own namespace). cast.DictToStruct deliberately skips kernel.IForm-typed
// fields (see its own doc comment), so this fills each one in place (onto
// the already-constructed instance, preserving its embedded
// ErrorsCollector), then runs its own required-fields check, AfterFill/
// Validate, and folds any errors it collects into f's own collection -
// recursively, for however many levels deep the nesting goes.
func fillNestedForms(f kernel.IForm, dict kernel.Dict) {
	v := reflect.ValueOf(f)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	collectNestedForms(v, dict, f, "")
}

func collectNestedForms(v reflect.Value, dict kernel.Dict, into kernel.IErrorsCollector, pathPrefix string) {
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)

		if field.Anonymous {
			ev := value
			if ev.Kind() == reflect.Pointer {
				if ev.IsNil() {
					continue
				}
				ev = ev.Elem()
			}
			if ev.Kind() == reflect.Struct {
				collectNestedForms(ev, dict, into, pathPrefix)
			}
			continue
		}

		nested, ok := asIForm(value)
		if !ok {
			continue
		}

		fName := cast.FieldName(field)
		fullName := fName
		if pathPrefix != "" {
			fullName = pathPrefix + "." + fName
		}

		// Required, for a nested-form field, means the block itself must
		// be present - checkMissingParams (called on f before this ever
		// runs) already enforces that if fName is in f.Required(). If the
		// block simply wasn't given and isn't required, there's nothing
		// to fill or validate here.
		if !dict.Has(fName) {
			continue
		}
		subDict, ok := asDict(dict.Get(fName))
		if !ok {
			continue
		}

		checkMissingParams(nested, subDict)
		if nested.HasErrors() {
			into.CollectErrorf("%s: %s", fullName, nested.GetFirstError().Error())
			continue
		}

		if err := cast.DictToStruct(&subDict, nested); err != nil {
			into.CollectErrorf("%s: %s", fullName, err.Error())
			continue
		}

		nested.AfterFill()
		if !nested.Validate() {
			if nested.HasErrors() {
				into.CollectErrorf("%s: %s", fullName, nested.GetFirstError().Error())
			} else {
				into.CollectErrorf("%s: invalid", fullName)
			}
			continue
		}
		if nested.HasErrors() {
			into.CollectErrorf("%s: %s", fullName, nested.GetFirstError().Error())
			continue
		}

		nv := reflect.ValueOf(nested)
		if nv.Kind() == reflect.Pointer {
			nv = nv.Elem()
		}
		collectNestedForms(nv, subDict, into, fullName)
	}
}

// asIForm reports whether value's address (or value itself, if it's
// already a pointer) implements kernel.IForm - kernel.IForm's methods are
// all pointer-receiver in the base Form, so a value-typed field needs
// Addr() to reach them.
func asIForm(value reflect.Value) (kernel.IForm, bool) {
	if value.Kind() == reflect.Pointer {
		if value.IsNil() || !value.Type().Implements(formInterfaceType) {
			return nil, false
		}
		f, ok := value.Interface().(kernel.IForm)
		return f, ok
	}
	if !value.CanAddr() || !reflect.PointerTo(value.Type()).Implements(formInterfaceType) {
		return nil, false
	}
	f, ok := value.Addr().Interface().(kernel.IForm)
	return f, ok
}

// asDict reads v as a kernel.Dict - directly, or converted from a plain
// map[string]any (what a parsed JSON body's nested objects decode as).
func asDict(v any) (kernel.Dict, bool) {
	switch m := v.(type) {
	case kernel.Dict:
		return m, true
	case map[string]any:
		return kernel.Dict(m), true
	}
	return nil, false
}

func fillGetParams(f kernel.IForm, r *http.Request) {
	queryParams := r.URL.Query()
	data := make(kernel.Dict)
	for key, values := range queryParams {
		if len(values) > 0 {
			if len(values) == 1 {
				data[key] = values[0]
			} else {
				data[key] = values
			}
		}
	}
	fillFormByDict(f, data)
}

func parseJSON(f kernel.IForm, r *http.Request) {
	data := make(kernel.Dict)
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		f.CollectErrorf("invalid request params")
		return
	}
	fillFormByDict(f, data)
}

func parseForm(f kernel.IForm, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		f.CollectErrorf("invalid request params")
		return
	}
	data := make(kernel.Dict)
	for key, values := range r.Form {
		if len(values) > 0 {
			if len(values) == 1 {
				data[key] = values[0]
			} else {
				data[key] = values
			}
		}
	}
	fillFormByDict(f, data)
}

func checkMissingParams(f kernel.IForm, data kernel.Dict) {
	if len(f.Required()) == 0 {
		return
	}

	v := reflect.ValueOf(f)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}

	m := buildFieldMap(v)

	missingParams := []string{}
	for _, param := range f.Required() {
		field, exists := m[param]
		// A nested form's own bookkeeping (its embedded *Form) is never
		// nil once properly constructed, so isZeroValue's generic
		// "already has a value" shortcut can never fire for it - which is
		// exactly backwards for a required nested-form field: what matters
		// is whether the caller's data included a block for it, not
		// whether the (always-non-zero) struct happens to look unset.
		if _, isNested := asIForm(field); !isNested && exists && !isZeroValue(field) {
			continue
		}
		if _, ok := data[param]; !ok {
			missingParams = append(missingParams, param)
		}
	}
	if len(missingParams) > 0 {
		f.CollectErrorf("missing required parameters: %s", strings.Join(missingParams, ","))
	}
}

func buildFieldMap(v reflect.Value) map[string]reflect.Value {
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}

	fieldMap := make(map[string]reflect.Value)
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)

		if field.Anonymous {
			for k, v2 := range buildFieldMap(value) {
				fieldMap[k] = v2
			}
			continue
		}

		if field.Tag.Get("json") == "-" {
			continue
		}

		fieldMap[cast.FieldName(field)] = value
	}

	return fieldMap
}

func isZeroValue(field reflect.Value) bool {
	if !field.IsValid() {
		return false
	}

	switch field.Kind() {
	case reflect.Pointer, reflect.Interface:
		return field.IsNil()
	case reflect.String:
		return field.Len() == 0
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return field.Interface() == reflect.Zero(field.Type()).Interface()
	case reflect.Slice, reflect.Map, reflect.Chan:
		return field.Len() == 0
	default:
		return false
	}
}
