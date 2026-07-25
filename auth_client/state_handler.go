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

// StateRequest is NewStateHandler's request form.
type StateRequest struct {
	*lxHttp.Form
	URI string `json:"uri"`
}

var _ kernel.IForm = (*StateRequest)(nil)

// Config describes StateRequest's fields - see kernel.IForm.
func (f *StateRequest) Config() kernel.FormConfig {
	return kernel.FormConfig{
		"uri": kernel.FormFieldConfig{
			Description: "the page to return to once authentication completes; defaults to '/' if not given",
			Required:    false,
		},
	}
}

/** @constructor kernel.CForm */

// NewStateRequest returns a StateRequest, ready to be used as an HTTP
// resource's CRequestForm.
func NewStateRequest() kernel.IForm {
	return lxHttp.PrepareForm(&StateRequest{Form: lxHttp.NewForm()})
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * StateHandler
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

/** @interface kernel.IHttpResource */

// StateHandler generates a fresh CSRF state, stores it (and the page to
// return to, URI) in the session, and hands the state back to the caller -
// call this before redirecting the browser to the authorization service, so
// the state can be echoed back through it and validated by
// AuthCallbackHandler. Register it at the path configured as
// AuthConfig.StatePath.
type StateHandler struct {
	*lxHttp.Resource
}

var _ kernel.IHttpResource = (*StateHandler)(nil)

/** @constructor kernel.CHttpResource */

// NewStateHandler constructs a StateHandler.
func NewStateHandler() kernel.IHttpResource {
	return &StateHandler{Resource: lxHttp.NewResource(kernel.HttpResourceConfig{
		CRequestForm: NewStateRequest,
	})}
}

// Run generates the CSRF state and stores it (with the return URI) in the session.
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
