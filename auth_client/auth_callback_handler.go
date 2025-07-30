package client

import (
	"fmt"
	"net/http"

	"github.com/epicoon/lxgo/kernel"
	lxHttp "github.com/epicoon/lxgo/kernel/http"
	"github.com/epicoon/lxgo/session"
)

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * AuthCallbackRequest
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */
/** kernel.IForm */
type AuthCallbackRequest struct {
	*lxHttp.Form
	Code  string `dict:"code"`
	State string `dict:"state"`
}

/** @constructor */
func NewAuthCallbackRequest() *AuthCallbackRequest {
	f := &AuthCallbackRequest{Form: lxHttp.NewForm()}
	f.SetRequired([]string{"code", "state"})
	return f
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * AuthCallbackHandler
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */
/** @interface kernel.IHttpResource */
type AuthCallbackHandler struct {
	*lxHttp.Resource
}

/** @type kernel.CHttpResource */
func NewAuthCallbackHandler() kernel.IHttpResource {
	return &AuthCallbackHandler{Resource: lxHttp.NewResource()}
}

func (handler *AuthCallbackHandler) Run() kernel.IHttpResponse {
	// Check session
	sess, err := session.ExtractSession(handler.Context())
	if err != nil {
		handler.LogError("Server configuration is wrong: sessions are required", "App")
		return handler.ErrorResponse(http.StatusInternalServerError, "Something went wrong")
	}
	if !sess.Has("lxgo_auth_state") {
		handler.LogError("Session must keep 'lxgo_auth_state'", "App")
		return handler.ErrorResponse(http.StatusInternalServerError, "Something went wrong")
	}

	// Extract request parameters
	reqForm := NewAuthCallbackRequest()
	lxHttp.FormFiller().SetContext(handler.Context()).SetForm(reqForm).Fill()
	if reqForm.HasErrors() {
		return handler.ErrorResponse(http.StatusBadRequest, fmt.Sprintf("Invalid request: %v", reqForm.GetFirstError()))
	}

	// Validate received state
	origState, ok := sess.Get("lxgo_auth_state").(string)
	if !ok {
		handler.LogError("Can not read 'lxgo_auth_state' from sesion", "App")
		return handler.ErrorResponse(http.StatusInternalServerError, "Something went wrong")
	}
	if origState != reqForm.State {
		return handler.ErrorResponse(http.StatusBadRequest, "Request is illegal")
	}

	// Try to exchange code for tokens
	authClient, err := AppComponent(handler.App())
	if err != nil {
		handler.LogError("wrong application configuration: auth_client component required", "App")
		return handler.ErrorResponse(http.StatusInternalServerError, "Something went wrong")
	}
	tokens, err := authClient.ExchangeCodeForTokens(reqForm.Code)
	if err != nil {
		handler.LogError(fmt.Sprintf("tokens exchange failed: %s", err), "App")
		return handler.ErrorResponse(http.StatusInternalServerError, "Something went wrong")
	}

	// Hold tokens
	sess.SetForce("lxgo_auth_tokens", tokens)

	// Define destination to redirect after getting tokens
	var origURL string
	if sess.Has("lxgo_auth_holder") {
		origURL, ok = sess.Get("lxgo_auth_holder").(string)
		if !ok {
			origURL = "/"
		}
	} else {
		origURL = "/"
	}

	// Return html with tokens, put tokens to LocalStorage, redirect to original page
	formHTML := fmt.Sprintf(`
    <html>
    <body>
        <script>
		localStorage.setItem('lxAuthAccessToken', '["%s", %d]');
		localStorage.setItem('lxAuthRefreshToken', '["%s", %d]');
		window.location.href = '%s';
		</script>
    </body>
    </html>
	`, tokens.Access.Value(), tokens.Access.expiresAt.Unix(),
		tokens.Refresh.Value(), tokens.Refresh.expiresAt.Unix(),
		origURL)
	return handler.HtmlResponse(kernel.HtmlResponseConfig{
		Html: formHTML,
	})
}
