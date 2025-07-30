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

func RequestBuilder() *Request {
	return &Request{}
}

type Request struct {
	method   string
	url      string
	params   map[string]any
	headers  map[string]string
	respForm any
}

func (b *Request) SetMethod(method string) *Request {
	b.method = method
	return b
}

func (b *Request) SetURL(url string) *Request {
	b.url = url
	return b
}

func (b *Request) SetJson() *Request {
	b.AddHeader("Content-Type", "application/json")
	return b
}

func (b *Request) SetParams(params map[string]any) *Request {
	b.params = params
	return b
}

func (b *Request) AddHeader(key, val string) *Request {
	if b.headers == nil {
		b.headers = make(map[string]string)
	}
	b.headers[key] = val
	return b
}

func (b *Request) SetResponseForm(f any) *Request {
	b.respForm = f
	return b
}

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
