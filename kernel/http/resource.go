package http

import (
	"fmt"
	"html"
	"net/http"
	"strings"

	"github.com/epicoon/lxgo/kernel"
)

/** @interface kernel.IHttpResource */
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
	return r
}

func (r *Resource) CRequestForm() kernel.CForm {
	return r.cRequestForm
}

func (r *Resource) CResponseForm() kernel.CForm {
	return r.cResponseForm
}

func (r *Resource) CFailForm() kernel.CForm {
	return r.cFailForm
}

func (r *Resource) Init() {
	// Pass
}

func (r *Resource) BeforeRun(callback func(res kernel.IHttpResource)) {
	if r.preCallbacks == nil {
		r.preCallbacks = make([]func(kernel.IHttpResource), 0, 1)
	}
	r.preCallbacks = append(r.preCallbacks, callback)
}

func (r *Resource) PreRun() {
	if r.preCallbacks == nil {
		return
	}

	for _, f := range r.preCallbacks {
		f(r)
	}
}

/** @abstract */
func (r *Resource) Run() kernel.IHttpResponse {
	// Pass
	return nil
}

/** @abstract */
func (r *Resource) ProcessRequestErrors() kernel.IHttpResponse {
	// Pass
	return nil
}

func (r *Resource) Lang() (res string) {
	res = "en-EN"

	req := r.Request()
	if req == nil {
		return
	}

	lang, err := Lang(req)
	if err != nil {
		r.LogError(fmt.Sprintf("Error while getting 'lxlang' cookie in service_handler: %v", err), "HttpHandling")
		return
	}

	return lang
}

func (r *Resource) SetContext(c kernel.IHandleContext) {
	r.context = c
}

func (r *Resource) Context() kernel.IHandleContext {
	return r.context
}

func (r *Resource) App() kernel.IApp {
	return r.context.App()
}

func (r *Resource) Route() string {
	return r.context.Route()
}

func (r *Resource) Method() string {
	return r.context.Method()
}

func (r *Resource) ResponseWriter() http.ResponseWriter {
	return r.context.ResponseWriter()
}

func (r *Resource) Request() *http.Request {
	return r.context.Request()
}

func (r *Resource) SetRequestForm(f kernel.IForm) {
	r.requestForm = f
}

func (r *Resource) RequestForm() kernel.IForm {
	return r.requestForm
}

func (r *Resource) Log(msg string, category string) {
	r.App().Log(fmt.Sprintf("Message from '%s' handling: %s", r.Route(), msg), category)
}

func (r *Resource) LogWarning(msg string, category string) {
	r.App().Log(fmt.Sprintf("Warning from '%s' handling: %s", r.Route(), msg), category)
}

func (r *Resource) LogError(msg string, category string) {
	r.App().Log(fmt.Sprintf("Error occured while '%s' handling: %s", r.Route(), msg), category)
}

func (r *Resource) HtmlResponse(conf kernel.HtmlResponseConfig) kernel.IHttpResponse {
	resp, err := HtmlResponse(r.App(), conf)
	if err != nil {
		r.LogError(fmt.Sprintf("Can not render template: %s", err), "HttpHandling")
		http.Error(r.ResponseWriter(), "Something went wrong", http.StatusInternalServerError)
		return nil
	}
	return resp
}

func (r *Resource) JsonResponse(conf kernel.JsonResponseConfig) kernel.IHttpResponse {
	return jsonResponse(r, conf, r.cResponseForm)
}

func (r *Resource) FailResponse(conf kernel.JsonResponseConfig) kernel.IHttpResponse {
	return jsonResponse(r, conf, r.cFailForm)
}

func (r *Resource) ErrorResponse(code int, msg string) kernel.IHttpResponse {
	response := new(Response)
	response.SetError(code, msg)
	return response
}

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
		FormFiller().SetForm(f).SetDict(conf.Dict).Fill()
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
