package http

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/epicoon/lxgo/kernel/conv"
)

// RequestBuilder starts a fluent outgoing HTTP request: chain Set*/AddHeader
// calls (each returns the Request itself), then call Send.
func RequestBuilder() *Request {
	return &Request{}
}

// Request is a fluent builder for outgoing HTTP requests - see RequestBuilder.
type Request struct {
	method   string
	url      string
	params   map[string]any
	headers  map[string]string
	respForm any
}

// SetMethod sets the HTTP method.
func (b *Request) SetMethod(method string) *Request {
	b.method = method
	return b
}

// SetURL sets the target URL.
func (b *Request) SetURL(url string) *Request {
	b.url = url
	return b
}

// SetJson adds a "Content-Type: application/json" header.
func (b *Request) SetJson() *Request {
	b.AddHeader("Content-Type", "application/json")
	return b
}

// SetParams sets the request parameters - sent as a query string for GET, as a JSON body otherwise.
func (b *Request) SetParams(params map[string]any) *Request {
	b.params = params
	return b
}

// AddHeader adds a request header.
func (b *Request) AddHeader(key, val string) *Request {
	if b.headers == nil {
		b.headers = make(map[string]string)
	}
	b.headers[key] = val
	return b
}

// SetResponseForm sets a struct (or *struct) to unmarshal the JSON response body into.
func (b *Request) SetResponseForm(f any) *Request {
	b.respForm = f
	return b
}

// Send performs the request and returns the raw *http.Response alongside
// the response form (if SetResponseForm was called) populated from its JSON body.
func (b *Request) Send() (*http.Response, any, error) {
	// Create request
	var req *http.Request
	if b.method == http.MethodGet {
		// Prepare query parameters
		urlWithParams := b.url
		if len(b.params) > 0 {
			query := make([]string, 0, len(b.params))
			for key, value := range b.params {
				query = append(query, key+"="+url.QueryEscape(conv.ToString(value)))
			}
			urlWithParams += "?" + strings.Join(query, "&")
		}
		// Create GET request
		r, err := http.NewRequest(b.method, urlWithParams, nil)
		if err != nil {
			return nil, nil, err
		}
		req = r
	} else {
		// Prepare JSON body for other methods
		jsonBody, err := json.Marshal(b.params)
		if err != nil {
			return nil, nil, err
		}
		req, err = http.NewRequest(b.method, b.url, bytes.NewReader(jsonBody))
		if err != nil {
			return nil, nil, err
		}
	}

	// Set headers
	for key, val := range b.headers {
		req.Header.Add(key, val)
	}

	// Do request and get response body
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	if b.respForm != nil {
		// Parse JSON-response
		if err = conv.JsonToStruct(body, b.respForm); err != nil {
			return nil, nil, err
		}
	}

	return resp, b.respForm, nil
}
