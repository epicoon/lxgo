package http

import (
	"net/http"

	"github.com/epicoon/lxgo/kernel"
)

/** @interface kernel.IHandleContext */
type HandleContext struct {
	app      kernel.IApp
	route    string
	method   string
	writer   http.ResponseWriter
	request  *http.Request
	resource kernel.IHttpResource
	metaData map[any]any
}

func (h *HandleContext) App() kernel.IApp {
	return h.app
}

func (h *HandleContext) Route() string {
	return h.route
}

func (h *HandleContext) Method() string {
	if h.method == "" {
		return "ANY"
	}
	return h.method
}

func (c *HandleContext) Resource() kernel.IHttpResource {
	return c.resource
}

func (c *HandleContext) ResponseWriter() http.ResponseWriter {
	return c.writer
}

func (c *HandleContext) Request() *http.Request {
	return c.request
}

func (c *HandleContext) Set(key any, value any) {
	if c.metaData == nil {
		c.metaData = make(map[any]any)
	}
	c.metaData[key] = value
}

func (c *HandleContext) Get(key any) any {
	val, ok := c.metaData[key]
	if !ok {
		return nil
	}
	return val
}
