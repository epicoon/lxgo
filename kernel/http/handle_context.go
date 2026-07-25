// Package http provides the default implementations of kernel's HTTP
// pieces - Router (kernel.IRouter), Resource (kernel.IHttpResource),
// HandleContext (kernel.IHandleContext), Form (kernel.IForm), Response
// (kernel.IHttpResponse) - for building an application's request handling.
package http

import (
	"net/http"

	"github.com/epicoon/lxgo/kernel"
)

/** @interface kernel.IHandleContext */

// HandleContext is the default kernel.IHandleContext implementation.
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

// NewHandleContext constructs a HandleContext for res at route - call Init
// to fill in the per-request method/writer/request before use.
func NewHandleContext(app kernel.IApp, route string, res kernel.IHttpResource) kernel.IHandleContext {
	return &HandleContext{
		app:      app,
		route:    route,
		resource: res,
	}
}

// Init sets up the context for a single request.
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

// App returns the owning application.
func (h *HandleContext) App() kernel.IApp {
	return h.app
}

// Route returns the matched route.
func (h *HandleContext) Route() string {
	return h.route
}

// Method returns the HTTP method, or "ANY" if none was set.
func (h *HandleContext) Method() string {
	if h.method == "" {
		return "ANY"
	}
	return h.method
}

// Resource returns the resource handling this request.
func (c *HandleContext) Resource() kernel.IHttpResource {
	return c.resource
}

// ResponseWriter returns the underlying http.ResponseWriter.
func (c *HandleContext) ResponseWriter() http.ResponseWriter {
	return c.writer
}

// Request returns the underlying *http.Request.
func (c *HandleContext) Request() *http.Request {
	return c.request
}

// Has reports whether key is set in the context.
func (c *HandleContext) Has(key any) bool {
	_, exists := c.metaData[key]
	return exists
}

// SetParams replaces all of the context's parameters.
func (c *HandleContext) SetParams(params map[string]any) {
	c.params = params
}

// Params returns all of the context's parameters.
func (c *HandleContext) Params() map[string]any {
	return c.params
}

// Set stores a value under key, for use across middleware/the resource.
func (c *HandleContext) Set(key any, value any) {
	if c.metaData == nil {
		c.metaData = make(map[any]any)
	}
	c.metaData[key] = value
}

// Get returns the value stored under key, or nil.
func (c *HandleContext) Get(key any) any {
	val, ok := c.metaData[key]
	if !ok {
		return nil
	}
	return val
}
