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

// Response is the default kernel.IHttpResponse implementation - built via
// one of SetError/SetHtmlData/SetJsonData, usually through
// kernel.IHttpResource's response-building methods rather than directly.
type Response struct {
	code     int
	headers  map[string]string
	dataType string
	data     string
}

var _ kernel.IHttpResponse = (*Response)(nil)

// SetCode sets the HTTP status code.
func (r *Response) SetCode(code int) {
	r.code = code
}

// AddHeader adds a response header.
func (r *Response) AddHeader(key, val string) {
	r.headers[key] = val
}

// SetError sets the response to an error with the given code and message.
func (r *Response) SetError(code int, msg string) {
	r.code = code
	r.data = msg
	r.dataType = typeError
}

// SetHtmlData sets the response body to raw HTML.
func (r *Response) SetHtmlData(data string) {
	r.data = data
	r.dataType = typeHtml
}

// SetJsonData sets the response body by marshaling data to JSON.
func (r *Response) SetJsonData(data any) error {
	jsonBody, err := json.Marshal(data)
	if err != nil {
		return err
	}

	r.data = string(jsonBody)
	r.dataType = typeJson
	return nil
}

// Code returns the HTTP status code, defaulting to 200 OK if none was set.
func (r *Response) Code() int {
	if r.code == 0 {
		return http.StatusOK
	}
	return r.code
}

// Headers returns the response headers.
func (r *Response) Headers() map[string]string {
	return r.headers
}

// Data returns the raw response body.
func (r *Response) Data() string {
	return r.data
}

// Send writes the response (status code, headers, Content-Type, body) to w.
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
