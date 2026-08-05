package client

import (
	"errors"
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
	Scope        string `json:"scope"`
}

var _ kernel.IForm = (*RefreshRequest)(nil)

// Config describes RefreshRequest's fields - see kernel.IForm.
func (f *RefreshRequest) Config() kernel.FormConfig {
	return kernel.FormConfig{
		"refresh_token": kernel.FormFieldConfig{
			Description: "token for refresh of a pair of tokens",
			Required:    true,
		},
		"scope": kernel.FormFieldConfig{
			Description: "optionally narrow the scope of the reissued tokens; omit to keep the current scope unchanged",
			Required:    false,
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

	tokens, err := authClient.RefreshTokens(req.RefreshToken, req.Scope)
	if err != nil {
		// A *StatusError means the authorization service rejected the
		// request itself (e.g. 400 for a wider scope than already
		// granted, 401 for a bad/expired token) - relay that status
		// instead of reporting it as a 500, but only for 4xx: a 5xx (or
		// anything else, like a network failure) is still our own
		// generic "Something went wrong" below, not a client mistake to
		// pass through as-is.
		var statusErr *StatusError
		if errors.As(err, &statusErr) && statusErr.Status >= 400 && statusErr.Status < 500 {
			return handler.ErrorResponse(statusErr.Status, statusErr.Message)
		}
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
