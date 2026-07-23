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
 * ElemRequest
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

/** @interface kernel.IForm */
type ElemRequest struct {
	*lxHttp.Form
	Elem   string         `json:"elem"`
	Path   string         `json:"path"`
	Params map[string]any `json:"params"`
}

/** @constructor */
func NewElemRequest() kernel.IForm {
	return &ElemRequest{Form: lxHttp.NewForm()}
}

func (f *ElemRequest) Config() kernel.FormConfig {
	return kernel.FormConfig{
		"elem": kernel.FormFieldConfig{
			Description: "element DI-key",
			Required:    true,
		},
		"path": kernel.FormFieldConfig{
			Description: "route to dispatch handler",
			Required:    true,
		},
		"params": kernel.FormFieldConfig{
			Description: "handling params",
			Required:    false,
		},
	}
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * ElemHandler
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

/** @interface kernel.IHttpResource */
type ElemHandler struct {
	*lxHttp.Resource
}

var _ kernel.IHttpResource = (*ElemHandler)(nil)

/** @constructor kernel.CHttpResource */
func NewElemHandler() kernel.IHttpResource {
	return &ElemHandler{Resource: lxHttp.NewResource(kernel.HttpResourceConfig{
		CRequestForm: NewElemRequest,
	})}
}

func (h *ElemHandler) Run() kernel.IHttpResponse {
	req := h.RequestForm().(*ElemRequest)
	if req.HasErrors() {
		return h.ErrorResponse(http.StatusBadRequest, req.GetFirstError().Error())
	}

	param := h.Context().Get("jspp")
	pp, ok := param.(jspp.IPreprocessor)
	if !ok {
		pp.LogError("Context params work wrong: can not get 'jspp'")
		return serverErrorResponse(h)
	}

	rawElem := h.App().DIContainer().Get(req.Elem)
	if rawElem == nil {
		return h.ErrorResponse(http.StatusNotFound, fmt.Sprintf("Element '%s' not found", req.Elem))
	}

	elem, ok := rawElem.(jspp.IElement)
	if !ok {
		return h.ErrorResponse(http.StatusNotFound, fmt.Sprintf("Element '%s' not found", req.Elem))
	}
	elem.Init(pp)

	m := elem.AjaxHandlers()
	cHandler, exists := m[req.Path]
	if !exists {
		return errorResponse(h, fmt.Sprintf("Element handler '%s' not found", req.Path))
	}

	orig := h.Request()
	newReq := orig.Clone(orig.Context())

	// Change request body
	bodyBytes, err := json.Marshal(req.Params)
	if err != nil {
		pp.LogError("Can not serialize request data: %v", err)
		return serverErrorResponse(h)
	}
	newReq.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	newReq.ContentLength = int64(len(bodyBytes))

	handler := cHandler()
	handler.Context().Set("elem", elem)
	handler.Init()
	return h.App().Router().Handle(handler, req.Path, h.ResponseWriter(), newReq)
}
