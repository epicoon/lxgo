package http

import (
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strings"

	"github.com/epicoon/lxgo/kernel"
)

/** @interface kernel.IHttpResource */

// Resource is the base kernel.IHttpResource implementation - embed it in
// your own resource struct and override at least Run.
type Resource struct {
	context       kernel.IHandleContext
	preCallbacks  []func(kernel.IHttpResource)
	cRequestForm  kernel.CForm
	cResponseForm kernel.CForm
	cFailForm     kernel.CForm
	requestForm   kernel.IForm
}

var _ kernel.IHttpResource = (*Resource)(nil)

/** @constructor */

// NewResource constructs a Resource, optionally configured with request/
// response/fail form constructors via c.
func NewResource(c ...kernel.HttpResourceConfig) *Resource {
	var conf *kernel.HttpResourceConfig
	if len(c) == 1 {
		conf = &c[0]
	}

	r := &Resource{}
	if conf != nil {
		if conf.CRequestForm != nil {
			r.cRequestForm = conf.CRequestForm
		}
		if conf.CResponseForm != nil {
			r.cResponseForm = conf.CResponseForm
		}
		if conf.CFailForm != nil {
			r.cFailForm = conf.CFailForm
		}
	}

	r.context = &HandleContext{resource: r}

	return r
}

// Base returns the underlying *http.Resource - useful when IHttpResource is wrapped.
func (r *Resource) Base() kernel.IHttpResource {
	return r
}

/** @abstract */

// Init is a no-op - override it for per-request setup.
func (r *Resource) Init() {
	// Pass
}

/** @abstract */

// Run returns nil - override it to handle the request and return a response.
// Run executes even if the request form failed validation and
// ProcessRequestErrors wasn't overridden (returned nil) - check
// RequestForm().HasErrors() yourself here if Run needs to react to that
// instead of relying on ProcessRequestErrors' short-circuit.
func (r *Resource) Run() kernel.IHttpResponse {
	// Pass
	return nil
}

/** @abstract */

// ProcessRequestErrors returns nil - override it to return a custom
// response when the request form fails validation, short-circuiting Run.
// Returning nil (the default) does NOT block Run - it still executes, with
// a request form that may carry errors.
func (r *Resource) ProcessRequestErrors() kernel.IHttpResponse {
	// Pass
	return nil
}

// BeforeRun registers a hook to run right before Run.
func (r *Resource) BeforeRun(callback func(res kernel.IHttpResource)) {
	if r.preCallbacks == nil {
		r.preCallbacks = make([]func(kernel.IHttpResource), 0, 1)
	}
	r.preCallbacks = append(r.preCallbacks, callback)
}

// BeforeRunCallbacks returns hooks to run right before Run.
func (r *Resource) BeforeRunCallbacks() []func(res kernel.IHttpResource) {
	return r.preCallbacks
}

// Lang returns the request's language via Lang, or "en-EN" if there's no request yet.
func (r *Resource) Lang() string {
	req := r.Request()
	if req == nil {
		return "en-EN"
	}

	return Lang(r.App(), req)
}

// SetContext binds the resource to its IHandleContext.
func (r *Resource) SetContext(c kernel.IHandleContext) {
	r.context = c
}

// Context returns the resource's IHandleContext.
func (r *Resource) Context() kernel.IHandleContext {
	return r.context
}

// App returns the owning application.
func (r *Resource) App() kernel.IApp {
	return r.context.App()
}

// Route returns the matched route.
func (r *Resource) Route() string {
	return r.context.Route()
}

// Method returns the HTTP method.
func (r *Resource) Method() string {
	return r.context.Method()
}

// ResponseWriter returns the underlying http.ResponseWriter.
func (r *Resource) ResponseWriter() http.ResponseWriter {
	return r.context.ResponseWriter()
}

// Request returns the underlying *http.Request.
func (r *Resource) Request() *http.Request {
	return r.context.Request()
}

// SetRequestForm sets the parsed request form.
func (r *Resource) SetRequestForm(f kernel.IForm) {
	r.requestForm = f
}

// RequestForm returns the parsed request form.
func (r *Resource) RequestForm() kernel.IForm {
	return r.requestForm
}

// Log writes an informational message under category, prefixed with the resource's route.
func (r *Resource) Log(msg string, category string) {
	r.App().Log(fmt.Sprintf("Message from '%s' handling: %s", r.Route(), msg), category)
}

// LogWarning writes a warning message under category, prefixed with the resource's route.
func (r *Resource) LogWarning(msg string, category string) {
	r.App().Log(fmt.Sprintf("Warning from '%s' handling: %s", r.Route(), msg), category)
}

// LogError writes an error message under category, prefixed with the resource's route.
func (r *Resource) LogError(msg string, category string) {
	r.App().Log(fmt.Sprintf("Error occurred while '%s' handling: %s", r.Route(), msg), category)
}

// HtmlResponse builds an HTML response - see the package-level HtmlResponse.
func (r *Resource) HtmlResponse(conf kernel.HtmlResponseConfig) kernel.IHttpResponse {
	resp, err := HtmlResponse(r.App(), conf)
	if err != nil {
		r.LogError(fmt.Sprintf("Can not render template: %s", err), "HttpHandling")
		http.Error(r.ResponseWriter(), "Something went wrong", http.StatusInternalServerError)
		return nil
	}
	return resp
}

// JsonResponse builds a successful JSON response, filling/validating it
// through the configured response form (see CResponseForm) if conf.Dict or
// conf.Form is used.
func (r *Resource) JsonResponse(conf kernel.JsonResponseConfig) kernel.IHttpResponse {
	return jsonResponse(r, conf, r.cResponseForm)
}

// FailResponse builds a failed JSON response, filling/validating it through
// the configured fail form (see CFailForm) if conf.Dict or conf.Form is used.
func (r *Resource) FailResponse(conf kernel.JsonResponseConfig) kernel.IHttpResponse {
	return jsonResponse(r, conf, r.cFailForm)
}

// ErrorResponse builds a JSON error response with the given HTTP code and message.
func (r *Resource) ErrorResponse(code int, msg string) kernel.IHttpResponse {
	response := new(Response)
	response.SetError(code, msg)
	return response
}

// Redirect builds an HTTP redirect response to URL with params appended as query string.
func (r *Resource) Redirect(URL string, code int, params map[string]any) kernel.IHttpResponse {
	u, err := url.Parse(URL)
	if err != nil {
		r.LogError(fmt.Sprintf("Can not redirect to %s: %v", URL, err), "HttpHandling")
		return nil
	}
	q := u.Query()
	for k, v := range params {
		q.Set(k, fmt.Sprint(v))
	}
	u.RawQuery = q.Encode()
	URL = u.String()

	http.Redirect(r.context.ResponseWriter(), r.Request(), URL, code)
	return nil
}

// PostRedirect builds a response with a self-submitting HTML form that
// POSTs params to url - use this instead of Redirect when the target needs
// a POST rather than a GET.
func (r *Resource) PostRedirect(url string, params map[string]any) kernel.IHttpResponse {
	escapedActionURL := html.EscapeString(url)
	var inputs strings.Builder
	for key, value := range params {
		escapedKey := html.EscapeString(key)
		escapedValue := html.EscapeString(fmt.Sprintf("%v", value))
		inputs.WriteString(fmt.Sprintf(`<input type="hidden" name="%s" value="%s">`, escapedKey, escapedValue))
	}

	html := fmt.Sprintf(`
	<html>
	<body>
		<div style="position:absolute;top:0;left:0;width:100%%;height:100%%;background-color:#272822"></div>
		<form id="postForm" action="%s" method="POST">
			%s
		</form>
		<script>
			document.getElementById('postForm').submit();
		</script>
	</body>
	</html>
	`, escapedActionURL, inputs.String())

	return r.HtmlResponse(kernel.HtmlResponseConfig{
		Html: html,
	})
}

// CRequestForm returns the request form constructor, if configured.
func (r *Resource) CRequestForm() kernel.CForm {
	return r.cRequestForm
}

// CResponseForm returns the response form constructor, if configured.
func (r *Resource) CResponseForm() kernel.CForm {
	return r.cResponseForm
}

// CFailForm returns the fail-response form constructor, if configured.
func (r *Resource) CFailForm() kernel.CForm {
	return r.cFailForm
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * PRIVATE
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

func jsonResponse(r kernel.IHttpResource, conf kernel.JsonResponseConfig, cForm kernel.CForm) kernel.IHttpResponse {
	response := new(Response)
	if conf.Code != 0 {
		response.SetCode(conf.Code)
	}
	if conf.Headers != nil {
		for key, val := range conf.Headers {
			response.AddHeader(key, val)
		}
	}

	if conf.Data != nil {
		response.SetJsonData(conf.Data)
		return response
	}

	if conf.Dict != nil {
		if cForm == nil {
			response.SetJsonData(conf.Dict)
			return response
		}
		f := cForm()
		if err := FormFiller().SetForm(f).SetDict(conf.Dict).Fill(); err != nil {
			r.LogError(fmt.Sprintf("Can not fill response form: %s", err), "HttpHandling")
			http.Error(r.ResponseWriter(), "Something went wrong", http.StatusInternalServerError)
			return nil
		}
		if f.HasErrors() {
			r.LogError(fmt.Sprintf("Can not fill response form: %s", f.GetFirstError().Error()), "HttpHandling")
			http.Error(r.ResponseWriter(), "Something went wrong", http.StatusInternalServerError)
			return nil
		}
		conf.Form = f
	}

	if conf.Form != nil {
		f := conf.Form
		if !f.Validate() {
			r.LogError(fmt.Sprintf("Response form validation failed: %s", f.GetFirstError().Error()), "HttpHandling")
			http.Error(r.ResponseWriter(), "Something went wrong", http.StatusInternalServerError)
			return nil
		}
		response.SetJsonData(f)
		return response
	}

	response.SetJsonData("")
	return response
}
