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
	params   map[string]any
	metaData map[any]any
}

var _ kernel.IHandleContext = (*HandleContext)(nil)

func NewHandleContext(app kernel.IApp, route string, res kernel.IHttpResource) kernel.IHandleContext {
	return &HandleContext{
		app:      app,
		route:    route,
		resource: res,
	}
}

func (h *HandleContext) Init(
	app kernel.IApp,
	route string,
	method string,
	writer http.ResponseWriter,
	request *http.Request,
) {
	h.app = app
	h.route = route
	h.method = method
	h.writer = writer
	h.request = request
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

func (c *HandleContext) Has(key any) bool {
	_, exists := c.metaData[key]
	return exists
}

func (c *HandleContext) SetParams(params map[string]any) {
	c.params = params
}

func (c *HandleContext) Params() map[string]any {
	return c.params
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
