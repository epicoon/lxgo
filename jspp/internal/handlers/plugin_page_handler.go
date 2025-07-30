package handlers

import (
	"net/http"

	"github.com/epicoon/lxgo/jspp"
	"github.com/epicoon/lxgo/kernel"
	lxHttp "github.com/epicoon/lxgo/kernel/http"
)

/** @interface kernel.IHttpResource */
type PluginPageHandler struct {
	*lxHttp.Resource
}

/** @constructor kernel.CHttpResource */
func NewPluginPageHandler() kernel.IHttpResource {
	return &PluginPageHandler{Resource: lxHttp.NewResource()}
}

var _ kernel.IHttpResource = (*PluginPageHandler)(nil)

func (h *PluginPageHandler) Run() kernel.IHttpResponse {
	pp, ok := h.Context().Get("jspp").(jspp.IPreprocessor)
	if !ok {
		h.LogError("can not render plugin as page - can not get js-preprocessor", "JSPreprocessor")
		return h.ErrorResponse(http.StatusInternalServerError, "Something went wrong")
	}

	pluginName, ok := h.Context().Get("pluginName").(string)
	if !ok {
		pp.LogError("can not render plugin as page - can not get plugin name", "JSPreprocessor")
		return h.ErrorResponse(http.StatusInternalServerError, "Something went wrong")
	}

	//TODO custom error response
	html, err := pp.PluginManager().HtmlPage(pluginName, h.Lang())
	if err != nil {
		pp.LogError("can not render plugin '%s' as page: %v", pluginName, err)
		return h.ErrorResponse(http.StatusInternalServerError, "Something went wrong")
	}

	return h.HtmlResponse(kernel.HtmlResponseConfig{
		Html: html,
	})
}
