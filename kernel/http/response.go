package http

import (
	"encoding/json"
	"net/http"

	"github.com/epicoon/lxgo/kernel"
)

const (
	typeError = "error"
	typeHtml  = "html"
	typeJson  = "json"
)

/** @interface kernel.IHttpResponse */
type Response struct {
	code     int
	headers  map[string]string
	dataType string
	data     string
}

var _ kernel.IHttpResponse = (*Response)(nil)

func (r *Response) SetCode(code int) {
	r.code = code
}

func (r *Response) Code() int {
	if r.code == 0 {
		return http.StatusOK
	}
	return r.code
}

func (r *Response) AddHeader(key, val string) {
	r.headers[key] = val
}

func (r *Response) SetError(code int, msg string) {
	r.code = code
	r.data = msg
	r.dataType = typeError
}

func (r *Response) SetHtmlData(data string) {
	r.data = data
	r.dataType = typeHtml
}

func (r *Response) SetJsonData(data any) error {
	jsonBody, err := json.Marshal(data)
	if err != nil {
		return err
	}

	r.data = string(jsonBody)
	r.dataType = typeJson
	return nil
}

func (r *Response) Send(w http.ResponseWriter) {
	if r.dataType == typeError {
		http.Error(w, r.data, r.code)
		return
	}

	for key, val := range r.headers {
		w.Header().Set(key, val)
	}

	switch r.dataType {
	case typeHtml:
		w.Header().Set("Content-Type", "text/html")
	case typeJson:
		w.Header().Set("Content-Type", "application/json")
	}

	w.WriteHeader(r.Code())

	w.Write([]byte(r.data))
}
