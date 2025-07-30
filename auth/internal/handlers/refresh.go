package handlers

import (
	"errors"
	"fmt"
	"net/http"

	cvn "github.com/epicoon/lxgo/auth/internal/conventions"
	"github.com/epicoon/lxgo/auth/internal/repos"
	"github.com/epicoon/lxgo/kernel"
	lxHttp "github.com/epicoon/lxgo/kernel/http"
)

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * RefreshRequest
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */
/** @interface kernel.IForm */
type RefreshRequest struct {
	*lxHttp.Form
	GrantType    string `json:"grant_type"`
	ClientID     uint   `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RefreshToken string `json:"refresh_token"`
}

var _ kernel.IForm = (*RefreshRequest)(nil)

func (f *RefreshRequest) Config() kernel.FormConfig {
	return kernel.FormConfig{
		"grant_type": kernel.FormFieldConfig{
			Description: "method for reissuing tokens, currently implemented 'refresh_token'",
			Required:    true,
		},
		"client_id": kernel.FormFieldConfig{
			Description: "client application identifier",
			Required:    true,
		},
		"client_secret": kernel.FormFieldConfig{
			Description: "unique string for client application verification",
			Required:    true,
		},
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
	*BaseHandler
}

var _ kernel.IHttpResource = (*RefreshHandler)(nil)

/** @type kernel.CHttpResource */
func NewRefreshHandler() kernel.IHttpResource {
	return &RefreshHandler{BaseHandler: NewBaseHandler(kernel.HttpResourceConfig{
		CRequestForm:  NewRefreshRequest,
		CResponseForm: NewTokensResponse,
	})}
}

func (handler *RefreshHandler) Run() kernel.IHttpResponse {
	coreApp, ok := handler.App().(cvn.IApp)
	if !ok {
		panic("app does not implement core.IApp")
	}

	req := handler.RequestForm().(*RefreshRequest)

	// Check client credentials
	client, err := coreApp.ClientsRepo().FindOne(req.ClientID, req.ClientSecret)
	if err != nil {
		if errors.Is(err, repos.ErrClientNotFound) {
			return errorResponse(handler, http.StatusNotFound, ERR_CLIENT_NOT_FOUND, "Client not found")
		}
		return serverErrorResponse(handler, fmt.Sprintf("Can not find client '%d': %s", req.ClientID, req.ClientSecret))
	}

	accessToken, refreshToken, err := coreApp.TokensRepo().FindTokensByRefresh(client, req.RefreshToken)
	if err != nil {
		if errors.Is(err, repos.ErrRefreshTokenExpired) {
			return errorResponse(handler, http.StatusUnauthorized, ERR_TOKEN_EXPIRED, "Token expired")
		}
		return errorResponse(handler, http.StatusUnauthorized, ERR_TOKEN_NOT_FOUND, "Token not found")
	}

	// Start transaction
	tx := coreApp.Gorm().Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	// Refresh tokens
	accessToken.Refresh(client)
	refreshToken.Refresh(client)
	if err := coreApp.TokensRepo().SaveTokens(accessToken, refreshToken); err != nil {
		tx.Rollback()
		return serverErrorResponse(handler, fmt.Sprintf("Can not save tokens %s, %s for client '%d': %s", accessToken.Value, refreshToken.Value, req.ClientID, req.ClientSecret))
	}

	// Commit transaction
	if err = tx.Commit().Error; err != nil {
		return serverErrorResponse(handler, fmt.Sprintf("Failed to commit transaction: %s", err))
	}

	// Successful request
	return handler.JsonResponse(kernel.JsonResponseConfig{
		Dict: kernel.Dict{
			"access_token":          accessToken.Value,
			"refresh_token":         refreshToken.Value,
			"access_token_expired":  accessToken.ExpiredAt.Unix(),
			"refresh_token_expired": refreshToken.ExpiredAt.Unix(),
		},
	})
}
