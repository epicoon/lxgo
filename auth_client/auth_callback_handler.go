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

/** @interface kernel.IForm */

// AuthCallbackRequest is NewAuthCallbackHandler's request form - the
// authorization service redirects here with a code to exchange and the
// CSRF state to validate.
type AuthCallbackRequest struct {
	*lxHttp.Form
	Code  string `json:"code"`
	State string `json:"state"`
}

var _ kernel.IForm = (*AuthCallbackRequest)(nil)

// Config describes AuthCallbackRequest's fields - see kernel.IForm.
func (f *AuthCallbackRequest) Config() kernel.FormConfig {
	return kernel.FormConfig{
		"code": kernel.FormFieldConfig{
			Description: "unique string to exchange for tokens",
			Required:    true,
		},
		"state": kernel.FormFieldConfig{
			Description: "unique string for CSRF protection",
			Required:    true,
		},
	}
}

/** @constructor kernel.CForm */

// NewAuthCallbackRequest returns an AuthCallbackRequest, ready to be used as
// an HTTP resource's CRequestForm.
func NewAuthCallbackRequest() kernel.IForm {
	return lxHttp.PrepareForm(&AuthCallbackRequest{Form: lxHttp.NewForm()})
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * AuthCallbackHandler
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

/** @interface kernel.IHttpResource */

// AuthCallbackHandler is the redirect target the authorization service
// sends the browser back to after authenticating - validates the CSRF
// state, exchanges the code for tokens, stores them in the session, and
// redirects to wherever the user originally came from (see NewStateHandler).
// Register it at the path configured as AuthConfig.RedirectUri's route.
type AuthCallbackHandler struct {
	*lxHttp.Resource
}

var _ kernel.IHttpResource = (*AuthCallbackHandler)(nil)

/** @constructor kernel.CHttpResource */

// NewAuthCallbackHandler constructs an AuthCallbackHandler.
func NewAuthCallbackHandler() kernel.IHttpResource {
	return &AuthCallbackHandler{Resource: lxHttp.NewResource(kernel.HttpResourceConfig{
		CRequestForm: NewAuthCallbackRequest,
	})}
}

// ProcessRequestErrors reports a malformed request - see kernel.IHttpResource.
func (handler *AuthCallbackHandler) ProcessRequestErrors() kernel.IHttpResponse {
	return handler.ErrorResponse(
		http.StatusBadRequest,
		fmt.Sprintf("Invalid request: %v", handler.RequestForm().GetFirstError()),
	)
}

// Run validates the CSRF state, exchanges the code for tokens, stores them
// in the session, and redirects the browser to the original page.
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

	reqForm := handler.RequestForm().(*AuthCallbackRequest)

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
