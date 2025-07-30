package handlers

import (
	"github.com/epicoon/lxgo/kernel"
	lxHttp "github.com/epicoon/lxgo/kernel/http"
	"github.com/epicoon/lxgo/session"
)

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * ReturnHandler
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */
/** @interface kernel.IHandler */
type ReturnHandler struct {
	*lxHttp.Resource
}

/** @type kernel.CHttpResource */
func NewReturnHandler() kernel.IHttpResource {
	return &ReturnHandler{Resource: lxHttp.NewResource()}
}

func (handler *ReturnHandler) Run() kernel.IHttpResponse {
	// Try to get Session
	sess, err := session.ExtractSession(handler.Context())
	if err != nil {
		return serverErrorResponse(handler, "server configuration is wrong: sessions support required")
	}

	// Read Authorization Params
	p := sess.Get("lxgo_auth_params")
	params, ok := p.(*AuthParams)
	if !ok {
		return serverErrorResponse(handler, "param 'lxgo_auth_params' must be in the session")
	}

	// Read Authorization Code
	if !sess.Has("lxgo_auth_code") {
		return serverErrorResponse(handler, "param 'lxgo_auth_code' must be in the session")
	}
	authCode := sess.Get("lxgo_auth_code")

	// Time to drop the session
	sessStorage, err := session.AppComponent(handler.App())
	if err != nil {
		return serverErrorResponse(handler, "server configuration is wrong: sessions support required")
	}
	sessStorage.DestroySession(sess)

	// Redirect
	return handler.PostRedirect(params.RedirectUri, map[string]any{
		"state": params.State,
		"code":  authCode,
	})
}
