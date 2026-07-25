package client

import (
	"fmt"
	"net/http"

	"github.com/epicoon/lxgo/kernel"
	lxHttp "github.com/epicoon/lxgo/kernel/http"
)

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * RefreshRequest
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

/** @interface kernel.IForm */

// RefreshRequest is NewRefreshHandler's request form.
type RefreshRequest struct {
	*lxHttp.Form
	RefreshToken string `json:"refresh_token"`
}

var _ kernel.IForm = (*RefreshRequest)(nil)

// Config describes RefreshRequest's fields - see kernel.IForm.
func (f *RefreshRequest) Config() kernel.FormConfig {
	return kernel.FormConfig{
		"refresh_token": kernel.FormFieldConfig{
			Description: "token for refresh of a pair of tokens",
			Required:    true,
		},
	}
}

/** @constructor kernel.CForm */

// NewRefreshRequest returns a RefreshRequest, ready to be used as an HTTP
// resource's CRequestForm.
func NewRefreshRequest() kernel.IForm {
	return lxHttp.PrepareForm(&RefreshRequest{Form: lxHttp.NewForm()})
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * RefreshHandler
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

/** @interface kernel.IHttpResource */

// RefreshHandler proxies a token-refresh request through to the
// authorization service, returning the new token pair as JSON. Register it
// at the path configured as AuthConfig.RefreshPath.
type RefreshHandler struct {
	*lxHttp.Resource
}

var _ kernel.IHttpResource = (*RefreshHandler)(nil)

/** @constructor */

// NewRefreshHandler constructs a RefreshHandler.
func NewRefreshHandler() kernel.IHttpResource {
	return &RefreshHandler{Resource: lxHttp.NewResource(kernel.HttpResourceConfig{
		CRequestForm: NewRefreshRequest,
	})}
}

// ProcessRequestErrors reports a malformed request - see kernel.IHttpResource.
func (handler *RefreshHandler) ProcessRequestErrors() kernel.IHttpResponse {
	return handler.ErrorResponse(http.StatusBadRequest, "Wrong params")
}

// Run exchanges the refresh token for a new token pair and returns it as JSON.
func (handler *RefreshHandler) Run() kernel.IHttpResponse {
	authClient, err := AppComponent(handler.App())
	if err != nil {
		handler.LogError("wrong application configuration: auth_client component required", "App")
		return handler.ErrorResponse(http.StatusInternalServerError, "Something went wrong")
	}

	req := handler.RequestForm().(*RefreshRequest)

	tokens, err := authClient.RefreshTokens(req.RefreshToken)
	if err != nil {
		handler.LogError(fmt.Sprintf("can not refresh tokens: %s", err), "App")
		return handler.ErrorResponse(http.StatusInternalServerError, "Something went wrong")
	}

	return handler.JsonResponse(kernel.JsonResponseConfig{
		Data: map[string]any{
			"access_token":          tokens.Access.Value(),
			"access_token_expired":  tokens.Access.ExpiresAt().Unix(),
			"refresh_token":         tokens.Refresh.Value(),
			"refresh_token_expired": tokens.Refresh.ExpiresAt().Unix(),
		},
	})
}
