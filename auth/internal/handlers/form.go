package handlers

import (
	"github.com/epicoon/lxgo/kernel"
	lxHttp "github.com/epicoon/lxgo/kernel/http"
)

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * FormHandler
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */
/** @interface kernel.IHttpResource */
type FormHandler struct {
	*lxHttp.Resource
}

/** @constructor kernel.CHttpResource */
func NewFormHandler() kernel.IHttpResource {
	return &FormHandler{Resource: lxHttp.NewResource()}
}

func (handler *FormHandler) Run() kernel.IHttpResponse {
	return handler.HtmlResponse(kernel.HtmlResponseConfig{
		Template: "form",
		Params: struct {
			Title string
		}{
			Title: "Auth",
		},
	})
}
