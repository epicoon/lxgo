package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/epicoon/lxgo/jspp"
	"github.com/epicoon/lxgo/kernel"
	lxHttp "github.com/epicoon/lxgo/kernel/http"
)

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * PluginRequest
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

/** @interface kernel.IForm */
type PluginRequest struct {
	*lxHttp.Form
	Plugin       string         `json:"plugin"`
	Path         string         `json:"path"`
	PluginParams map[string]any `json:"pluginParams"`
	Data         map[string]any `json:"data"`
}

/** @constructor */
func NewPluginRequest() *PluginRequest {
	return &PluginRequest{Form: lxHttp.NewForm()}
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * ServiceHandler
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

/** @interface kernel.IHttpResource */
type PluginHandler struct {
	*lxHttp.Resource
}

var _ kernel.IHttpResource = (*PluginHandler)(nil)

/** @constructor kernel.CHttpResource */
func NewPluginHandler() kernel.IHttpResource {
	return &PluginHandler{Resource: lxHttp.NewResource()}
}

func (h *PluginHandler) Run() kernel.IHttpResponse {
	reqForm := NewPluginRequest()

	lxHttp.FormFiller().SetContext(h.Context()).SetForm(reqForm).Fill()
	if reqForm.HasErrors() {
		return h.ErrorResponse(http.StatusBadRequest, reqForm.GetFirstError().Error())
	}

	param := h.Context().Get("jspp")
	pp, ok := param.(jspp.IPreprocessor)
	if !ok {
		pp.LogError("Context params work wrong: can not get 'jspp'")
		return serverErrorResponse(h)
	}

	plugin := pp.PluginManager().Get(reqForm.Plugin)
	if plugin == nil {
		return errorResponse(h, fmt.Sprintf("Plugin '%s' not found", reqForm.Plugin))
	}

	m := plugin.AjaxHandlers()
	cHandler, exists := m[reqForm.Path]
	if !exists {
		return errorResponse(h, fmt.Sprintf("Handler '%s' not found", reqForm.Path))
	}

	// Change request body
	bodyBytes, err := json.Marshal(reqForm.Data)
	if err != nil {
		pp.LogError("Can not serialize request data: %v", err)
		return serverErrorResponse(h)
	}
	orig := h.Request()
	newReq := orig.Clone(orig.Context())
	newReq.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	newReq.ContentLength = int64(len(bodyBytes))

	handler := cHandler()
	handler.Init()
	return h.App().Router().Handle(handler, reqForm.Path, h.ResponseWriter(), newReq)
}
