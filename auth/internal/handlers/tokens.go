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
 * TokensRequest
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */
/** @interface kernel.IForm */
type TokensRequest struct {
	*lxHttp.Form
	Code         string `json:"code"`
	ClientID     uint   `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

var _ kernel.IForm = (*TokensRequest)(nil)

func (f *TokensRequest) Config() kernel.FormConfig {
	return kernel.FormConfig{
		"code": kernel.FormFieldConfig{
			Description: "unique string for exchange for tokens (redirect endpoint link)",
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
	}
}

/** @constructor */
func NewTokensRequest() kernel.IForm {
	return lxHttp.PrepareForm(&TokensRequest{Form: lxHttp.NewForm()})
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * TokensHandler
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */
/** @interface kernel.IHttpResource */
type TokensHandler struct {
	*BaseHandler
}

var _ kernel.IHttpResource = (*TokensHandler)(nil)

/** @type kernel.CHttpResource */
func NewTokensHandler() kernel.IHttpResource {
	return &TokensHandler{BaseHandler: NewBaseHandler(kernel.HttpResourceConfig{
		CRequestForm:  NewTokensRequest,
		CResponseForm: NewTokensResponse,
	})}
}

func (handler *TokensHandler) Run() kernel.IHttpResponse {
	coreApp, ok := handler.App().(cvn.IApp)
	if !ok {
		panic("app does not implement core.IApp")
	}

	reqForm := handler.RequestForm().(*TokensRequest)

	// Validate received code
	authCode, err := coreApp.CodesRepo().FindByValue(reqForm.Code)
	if err != nil {
		if errors.Is(err, repos.ErrorCodeNotFound) {
			return errorResponse(handler, http.StatusBadRequest, ERR_INVAL_CODE, "Verification code is not valid")
		}
		return serverErrorResponse(handler, fmt.Sprintf("Can not find code '%s': %s", reqForm.Code, err))
	}
	if authCode.ClientID != reqForm.ClientID {
		return errorResponse(handler, http.StatusBadRequest, ERR_INVAL_CODE, "Verification code is not valid")
	}

	// Find Client
	client, err := coreApp.ClientsRepo().FindOne(reqForm.ClientID, reqForm.ClientSecret)
	if err != nil {
		if errors.Is(err, repos.ErrClientNotFound) {
			return errorResponse(handler, http.StatusNotFound, ERR_CLIENT_NOT_FOUND, "Client not found")
		}
		return serverErrorResponse(handler, fmt.Sprintf("Can not find client '%d': %s", reqForm.ClientID, reqForm.ClientSecret))
	}

	// Find User
	user, err := coreApp.UsersRepo().FindByID(authCode.UserID)
	if err != nil {
		if errors.Is(err, repos.ErrUserNotFound) {
			return errorResponse(handler, http.StatusNotFound, ERR_USER_NOT_FOUND, "User not found")
		}
		return serverErrorResponse(handler, fmt.Sprintf("Can not find user with id='%d'", authCode.UserID))
	}

	// Start transaction
	tx := coreApp.Gorm().Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	// Delete code
	codesRepo := coreApp.CodesRepo()
	codesRepo.SetTx(tx)
	if err = codesRepo.Delete(authCode); err != nil {
		tx.Rollback()
		return serverErrorResponse(handler, fmt.Sprintf("Can not delete code '%s': %s", reqForm.Code, err))
	}

	// Create tokens
	tokensRepo := coreApp.TokensRepo()
	tokensRepo.SetTx(tx)
	if err := tokensRepo.DropTokensByUser(client, user); err != nil {
		tx.Rollback()
		return serverErrorResponse(handler, fmt.Sprintf("Can not delete existing tokens: %s", err))
	}
	accessToken, err := tokensRepo.CreateAccessToken(client, user)
	if err != nil {
		tx.Rollback()
		return serverErrorResponse(handler, fmt.Sprintf("Can not create access token: %s", err))
	}
	refreshToken, err := tokensRepo.CreateRefreshToken(client, user)
	if err != nil {
		tx.Rollback()
		return serverErrorResponse(handler, fmt.Sprintf("Can not create refresh token: %s", err))
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
