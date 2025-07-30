package http

import "github.com/epicoon/lxgo/kernel"

/** @interface kernel.IHttpResource */
type stdHandler struct {
	*Resource
}

/** @constructor kernel.CHttpResource */
func newStdHandler() kernel.IHttpResource {
	return &stdHandler{Resource: NewResource()}
}

func (handler *stdHandler) Run() kernel.IHttpResponse {
	template := handler.Context().Get("Template").(string)
	params := handler.Context().Get("Params")

	return handler.HtmlResponse(kernel.HtmlResponseConfig{
		Template: template,
		Params:   params,
	})
}
