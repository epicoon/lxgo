package client

import (
	"net/http"

	"github.com/epicoon/lxgo/kernel"
	lxHttp "github.com/epicoon/lxgo/kernel/http"
	"github.com/epicoon/lxgo/kernel/utils"
	"github.com/epicoon/lxgo/session"
)

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * StateRequest
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */
/** @interface kernel.IForm */
type StateRequest struct {
	*lxHttp.Form
	URI string `json:"uri"`
}

func (f *StateRequest) Config() kernel.FormConfig {
	return kernel.FormConfig{
		"uri": kernel.FormFieldConfig{
			Description: "the page to return to once authentication completes; defaults to '/' if not given",
			Required:    false,
		},
	}
}

/** @constructor */
func NewStateRequest() kernel.IForm {
	return lxHttp.PrepareForm(&StateRequest{Form: lxHttp.NewForm()})
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * StateHandler
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */
/** @interface kernel.IHttpResource */
type StateHandler struct {
	*lxHttp.Resource
}

/** @kernel.CHttpResource */
func NewStateHandler() kernel.IHttpResource {
	return &StateHandler{Resource: lxHttp.NewResource(kernel.HttpResourceConfig{
		CRequestForm: NewStateRequest,
	})}
}

func (handler *StateHandler) Run() kernel.IHttpResponse {
	// Get session
	sess, err := session.ExtractSession(handler.Context())
	if err != nil {
		handler.LogError("Server configuration is wrong: sessions are required", "App")
		return handler.ErrorResponse(http.StatusInternalServerError, "Something went wrong")
	}

	req := handler.RequestForm().(*StateRequest)
	var URI string
	if req.URI == "" {
		URI = "/"
	} else {
		URI = req.URI
	}

	// Gen state and keep it in session
	state := utils.GenRandomHash(16)
	sess.SetForce("lxgo_auth_state", state)
	sess.SetForce("lxgo_auth_holder", URI)

	// Success
	return handler.JsonResponse(kernel.JsonResponseConfig{
		Data: struct {
			State string `json:"state"`
		}{
			State: state,
		},
	})
}
