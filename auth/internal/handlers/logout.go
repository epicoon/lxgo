package handlers

import (
	"errors"
	"fmt"
	"net/http"

	authClient "github.com/epicoon/lxgo/auth_client"
	cvn "github.com/epicoon/lxgo/auth/internal/conventions"
	"github.com/epicoon/lxgo/auth/internal/repos"
	"github.com/epicoon/lxgo/kernel"
	lxHttp "github.com/epicoon/lxgo/kernel/http"
)

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * LogoutRequest
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */
/** @interface kernel.IForm */
type LogoutRequest struct {
	*lxHttp.Form
	ClientID uint `json:"client_id"`
}

func (f *LogoutRequest) Config() kernel.FormConfig {
	return kernel.FormConfig{
		"client_id": kernel.FormFieldConfig{
			Description: "client application identifier",
			Required:    true,
		},
	}
}

/** @constructor */
func NewLogoutRequest() kernel.IForm {
	return lxHttp.PrepareForm(&LogoutRequest{Form: lxHttp.NewForm()})
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * LogoutHandler
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */
/** @interface kernel.IHttpResource */
type LogoutHandler struct {
	*BaseHandler
}

/** @type kernel.CHttpResource */
func NewLogoutHandler() kernel.IHttpResource {
	return &LogoutHandler{BaseHandler: NewBaseHandler(kernel.HttpResourceConfig{
		CRequestForm: NewLogoutRequest,
	})}
}

func (handler *LogoutHandler) ProcessRequestErrors() kernel.IHttpResponse {
	return errorResponse(
	    handler,
	    http.StatusBadRequest,
	    ERR_NO_CLIENT_ID,
	    fmt.Sprintf("Invalid request: %v", handler.RequestForm().GetFirstError()),
    )
}

func (handler *LogoutHandler) Run() kernel.IHttpResponse {
	coreApp, ok := handler.App().(cvn.IApp)
	if !ok {
		panic("app does not implement core.IApp")
	}

	reqForm := handler.RequestForm().(*LogoutRequest)

	// Check Client
	client, err := coreApp.ClientsRepo().FindByID(reqForm.ClientID)
	if err != nil {
		if errors.Is(err, repos.ErrClientNotFound) {
			return errorResponse(handler, http.StatusNotFound, ERR_CLIENT_NOT_FOUND, "Client not found")
		}
		return serverErrorResponse(handler, fmt.Sprintf("Can not find client '%d'", reqForm.ClientID))
	}

	// Get Access Token Value
	bearer, err := authClient.GetBearer(handler.Context())
	if err != nil {
		return errorResponse(handler, http.StatusUnauthorized, ERR_INVAL_AUTH_HEADER, fmt.Sprintf("Request issue: %s", err))
	}

	// Start Transaction
	tx := coreApp.Gorm().Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	// Check Tokens
	tokensRepo := coreApp.TokensRepo()
	tokensRepo.SetTx(tx)
	accessToken, refreshToken, err := tokensRepo.FindTokens(client, bearer)
	if err != nil {
		tx.Rollback()
		if errors.Is(err, repos.ErrTokensNotFound) {
			return errorResponse(handler, http.StatusUnauthorized, ERR_TOKEN_NOT_FOUND, "Token not found")
		}
		return serverErrorResponse(handler, fmt.Sprintf("Error while tokens searching: %v", err))
	}

	// Drop Tokens
	if err := tokensRepo.DropTokens(accessToken, refreshToken); err != nil {
		tx.Rollback()
		return serverErrorResponse(handler, fmt.Sprintf("Can not delete tokens: %v", err))
	}

	// Commit Transaction
	if err = tx.Commit().Error; err != nil {
		return serverErrorResponse(handler, fmt.Sprintf("Failed to commit transaction: %s", err))
	}

	// Successful Request
	return handler.JsonResponse(kernel.JsonResponseConfig{
		Form: NewSuccessResponse(),
	})
}
