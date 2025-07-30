package handlers

import (
	"fmt"
	"net/http"

	"github.com/epicoon/lxgo/jspp"
	"github.com/epicoon/lxgo/kernel"
	lxHttp "github.com/epicoon/lxgo/kernel/http"
)

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * ServiceRequest
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

/** @interface kernel.IForm */
type ServiceRequest struct {
	*lxHttp.Form
	Action string         `json:"action"`
	Params map[string]any `json:"params"`
}

/** @constructor */
func NewServiceRequest() *ServiceRequest {
	return &ServiceRequest{Form: lxHttp.NewForm()}
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * ServiceHandler
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

/** @interface kernel.IHttpResource */
type ServiceHandler struct {
	*lxHttp.Resource
}

var _ kernel.IHttpResource = (*ServiceHandler)(nil)

/** @constructor kernel.CHttpResource */
func NewServiceHandler() kernel.IHttpResource {
	return &ServiceHandler{Resource: lxHttp.NewResource()}
}

func (h *ServiceHandler) Run() kernel.IHttpResponse {
	req := NewServiceRequest()

	lxHttp.FormFiller().SetContext(h.Context()).SetForm(req).Fill()
	if req.HasErrors() {
		return h.ErrorResponse(http.StatusBadRequest, req.GetFirstError().Error())
	}

	if req.Action == "get-modules" {
		return getModules(h, req.Params)
	}

	return errorResponse(h, "Unknown action")
}

func getModules(h *ServiceHandler, list map[string]any) kernel.IHttpResponse {
	haveAny, exists := list["have"]
	if !exists {
		return errorResponse(h, "'have' parameter required")
	}
	have, ok := haveAny.([]any)
	if !ok {
		return errorResponse(h, "Wrong type of 'have' parameter")
	}
	needAny, exists := list["need"]
	if !exists {
		return errorResponse(h, "'need' parameter required")
	}
	need, ok := needAny.([]any)
	if !ok {
		return errorResponse(h, "Wrong type of 'need' parameter")
	}

	param := h.Context().Get("jspp")
	pp, ok := param.(jspp.IPreprocessor)
	if !ok {
		pp.LogError("Context params work wrong: can not get 'jspp'")
		return serverErrorResponse(h)
	}

	needReset := false
	code := ""
	for _, val := range need {
		name, ok := val.(string)
		if !ok {
			return errorResponse(h, "Wrong 'need' module name type")
		}
		code += fmt.Sprintf("@lx:use %s;", name)
		if !pp.ModulesMap().Has(name) {
			needReset = true
		}
	}

	except := make([]string, len(have))
	for _, val := range have {
		name, ok := val.(string)
		if !ok {
			return errorResponse(h, "Wrong 'have' module name type")
		}
		except = append(except, name)
	}

	if needReset {
		pp.ModulesMap().Reset()
	}

	lang := h.Lang()
	compiler := pp.CompilerBuilder().
		SetLang(lang).
		SetClientContext().
		IgnoreModules(except).
		SetCode(code).
		Compiler()

	mCode, err := compiler.Run()
	if err != nil {
		pp.LogError("Can not compile modules '%s': %v", code, err)
	}

	//TODO
	// Internationalization here?

	return h.JsonResponse(kernel.JsonResponseConfig{
		Data: map[string]any{
			"success": true,
			"data": map[string]any{
				"code":            mCode,
				"compiledModules": compiler.CompiledModules(),
				"css":             []string{},
			},
		},
	})
}

func serverErrorResponse(h *ServiceHandler) kernel.IHttpResponse {
	return h.JsonResponse(kernel.JsonResponseConfig{
		Code: http.StatusInternalServerError,
		Data: map[string]any{
			"success": false,
			"data":    "Something went wrong",
		},
	})
}

func errorResponse(h *ServiceHandler, msg string) kernel.IHttpResponse {
	return h.JsonResponse(kernel.JsonResponseConfig{
		Code: http.StatusBadRequest,
		Data: map[string]any{
			"success": false,
			"data":    msg,
		},
	})
}
