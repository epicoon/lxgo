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
type RefreshRequest struct {
	*lxHttp.Form
	RefreshToken string `json:"refresh_token"`
}

func (f *RefreshRequest) Config() kernel.FormConfig {
	return kernel.FormConfig{
		"refresh_token": kernel.FormFieldConfig{
			Description: "token for refresh of a pair of tokens",
			Required:    true,
		},
	}
}

/** @constructor */
func NewRefreshRequest() kernel.IForm {
	return lxHttp.PrepareForm(&RefreshRequest{Form: lxHttp.NewForm()})
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * RefreshHandler
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */
/** @interface kernel.IHttpResource */
type RefreshHandler struct {
	*lxHttp.Resource
}

/** @constructor */
func NewRefreshHandler() kernel.IHttpResource {
	return &RefreshHandler{Resource: lxHttp.NewResource(kernel.HttpResourceConfig{
		CRequestForm: NewRefreshRequest,
	})}
}

func (handler *RefreshHandler) ProcessRequestErrors() kernel.IHttpResponse {
	return handler.ErrorResponse(http.StatusBadRequest, "Wrong params")
}

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
