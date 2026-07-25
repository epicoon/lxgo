package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"

	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/conv"
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

	if err := conv.DictToStruct(&dict, f); err != nil {
		f.CollectErrorf(err.Error())
	}
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
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	//TODO cache?
	m := buildJSONFieldMap(v)

	missingParams := []string{}
	for _, param := range f.Required() {
		field, exists := m[param]
		if exists && !isZeroValue(field) {
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

func buildJSONFieldMap(v reflect.Value) map[string]reflect.Value {
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	fieldMap := make(map[string]reflect.Value)
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)

		if field.Anonymous {
			for k, v2 := range buildJSONFieldMap(value) {
				fieldMap[k] = v2
			}
			continue
		}

		tag := field.Tag.Get("json")
		if tag == "-" {
			continue
		}

		jsonName := strings.Split(tag, ",")[0]
		if jsonName == "" {
			jsonName = field.Name
		}

		fieldMap[jsonName] = value
	}

	return fieldMap
}

func isZeroValue(field reflect.Value) bool {
	if !field.IsValid() {
		return false
	}

	switch field.Kind() {
	case reflect.Ptr, reflect.Interface:
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
