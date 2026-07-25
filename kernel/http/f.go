package http

import (
	"fmt"
	"net/http"

	"github.com/epicoon/lxgo/kernel"
)

// Lang returns req's language from the "lxlang" cookie, defaulting to
// "en-EN" if it's missing.
func Lang(app kernel.IApp, req *http.Request) string {
	lc, err := req.Cookie("lxlang")
	if err != nil {
		if err != http.ErrNoCookie {
			app.LogError(fmt.Sprintf("Error occurred while parsing cookie for path '%s': %v", req.URL.Path, err), "HttpHandling")
		}
		return "en-EN"
	}
	return lc.Value
}

// HtmlResponse builds an HTML IHttpResponse from conf - conf.Html is used
// verbatim if set, otherwise conf.Template is rendered with conf.Params via
// app's template renderer.
func HtmlResponse(app kernel.IApp, conf kernel.HtmlResponseConfig) (kernel.IHttpResponse, error) {
	var html string
	if conf.Html != "" {
		html = conf.Html
	} else if conf.Template != "" {
		rendered, err := app.TemplateRenderer().
			SetTemplateName(conf.Template).
			SetParams(conf.Params).Render()
		if err != nil {
			return nil, err
		}
		html = rendered
	}

	response := new(Response)
	if conf.Code != 0 {
		response.SetCode(conf.Code)
	}
	if conf.Headers != nil {
		for key, val := range conf.Headers {
			response.AddHeader(key, val)
		}
	}
	response.SetHtmlData(html)
	return response, nil
}
