package http

import (
	"encoding/json"
	"net/http"
	"reflect"
	"strings"

	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/conv"
)

type formFiller struct {
	form kernel.IForm
	ctx  kernel.IHandleContext
	dict kernel.Dict
}

func FormFiller() *formFiller {
	return &formFiller{}
}

func (ff *formFiller) SetForm(f kernel.IForm) *formFiller {
	ff.form = f
	return ff
}

func (ff *formFiller) SetContext(ctx kernel.IHandleContext) *formFiller {
	ff.ctx = ctx
	return ff
}

func (ff *formFiller) SetDict(d kernel.Dict) *formFiller {
	ff.dict = d
	return ff
}

func (ff *formFiller) Fill() {
	if ff.form == nil {
		panic("nothing to fiil")
	}
	if ff.ctx == nil && ff.dict == nil {
		panic("no data to fill form")
	}

	if ff.ctx != nil {
		fillFormByHandleContext(ff.form, ff.ctx)
	} else if ff.dict != nil {
		fillFormByDict(ff.form, ff.dict)
	}
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

	// if useFill(f) {
	// 	if err := f.Fill(&dict); err != nil {
	// 		f.CollectErrorf(err.Error())
	// 	}
	// } else {
	// 	if err := conv.DictToStruct(&dict, f); err != nil {
	// 		f.CollectErrorf(err.Error())
	// 	}
	// }
}

// var fillMethodCache sync.Map

// func useFill(f kernel.IForm) bool {
// 	t := reflect.TypeOf(f)
// 	if t.Kind() == reflect.Ptr {
// 		t = t.Elem()
// 	}
// 	if cached, ok := fillMethodCache.Load(t); ok {
// 		return cached.(bool)
// 	}
// 	method, exists := t.MethodByName("Fill")
// 	orig, _ := reflect.TypeOf((*Form)(nil)).MethodByName("Fill")
// 	isOverridden := exists && method.Type != orig.Type
// 	fillMethodCache.Store(t, isOverridden)
// 	return isOverridden
// }

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

	missingParams := []string{}
	for _, param := range f.Required() {
		field := v.FieldByName(param)
		if !isZeroValue(field) {
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
