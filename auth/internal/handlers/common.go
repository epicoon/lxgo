package handlers

import (
	"errors"
	"fmt"
	"net/http"

	cvn "github.com/epicoon/lxgo/auth/internal/conventions"
	"github.com/epicoon/lxgo/auth/internal/models"
	"github.com/epicoon/lxgo/kernel"
	lxHttp "github.com/epicoon/lxgo/kernel/http"
	"github.com/epicoon/lxgo/session"
)

const (
	ERR_INVAL_CODE = 1001 + iota
	ERR_NO_REFRESH_TOKEN
	ERR_NO_CLIENT_ID
	ERR_TOKEN_NOT_FOUND
	ERR_TOKEN_EXPIRED
	ERR_INVAL_TOKEN
	ERR_INVAL_AUTH_HEADER
	ERR_CLIENT_NOT_FOUND
	ERR_USER_NOT_FOUND
	ERR_SERVER_ERROR
	ERR_COMMON

	// 1012
	ERR_NO_LOGIN_PWD
	ERR_WRONG_LOGIN_PWD
	ERR_INVAL_LOGIN
	ERR_INVAL_PWD
	ERR_LOGIN_EXISTS
)

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * BaseHandler
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

/** @interface kernel.IHttpResource */
type BaseHandler struct {
	*lxHttp.Resource
}

var _ kernel.IHttpResource = (*BaseHandler)(nil)

/** @type kernel.CHttpResource */
func NewBaseHandler(c ...kernel.HttpResourceConfig) *BaseHandler {
	var conf kernel.HttpResourceConfig
	if len(c) == 0 {
		conf = kernel.HttpResourceConfig{}
	} else {
		conf = c[0]
	}
	conf.CFailForm = NewFailResponse
	return &BaseHandler{Resource: lxHttp.NewResource(conf)}
}

func (handler *TokensHandler) ProcessRequestErrors() kernel.IHttpResponse {
	return handler.FailResponse(kernel.JsonResponseConfig{
		Code: http.StatusBadRequest,
		Dict: kernel.Dict{
			"error_code":    ERR_COMMON,
			"error_message": fmt.Sprintf("Invalid request: %v", handler.RequestForm().GetFirstError()),
		},
	})
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * FailResponse
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

/** @interface kernel.IForm */
type FailResponse struct {
	*lxHttp.Form
	Success      bool   `json:"success"`
	ErrorCode    int    `json:"error_code"`
	ErrorMessage string `json:"error_message"`
}

var _ kernel.IForm = (*FailResponse)(nil)

func (f *FailResponse) Config() kernel.FormConfig {
	return kernel.FormConfig{
		"success": kernel.FormFieldConfig{
			Description: "equals false",
			Required:    true,
		},
		"error_code": kernel.FormFieldConfig{
			Description: "number code of the issue occured",
			Required:    true,
		},
		"error_message": kernel.FormFieldConfig{
			Description: "explanation of the issue occured",
			Required:    true,
		},
	}
}

func NewFailResponse() kernel.IForm {
	return &FailResponse{
		Form:    lxHttp.NewForm(),
		Success: false,
	}
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * SuccessResponse
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

/** @interface kernel.IForm */
type SuccessResponse struct {
	*lxHttp.Form
	Success bool `json:"success"`
}

var _ kernel.IForm = (*SuccessResponse)(nil)

func NewSuccessResponse() *SuccessResponse {
	return &SuccessResponse{
		Form:    lxHttp.NewForm(),
		Success: true,
	}
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * TokensResponse
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

/** @interface kernel.IForm */
type TokensResponse struct {
	*lxHttp.Form
	Success             bool   `json:"success"`
	AccessToken         string `json:"access_token"`
	RefreshToken        string `json:"refresh_token"`
	AccessTokenExpired  int64  `json:"access_token_expired"`
	RefreshTokenExpired int64  `json:"refresh_token_expired"`
}

var _ kernel.IForm = (*TokensResponse)(nil)

func (f *TokensResponse) Config() kernel.FormConfig {
	return kernel.FormConfig{
		"success": kernel.FormFieldConfig{
			Description: "equals true",
			Required:    true,
		},
		"access_token": kernel.FormFieldConfig{
			Description: "access token",
			Required:    true,
		},
		"access_token_expired": kernel.FormFieldConfig{
			Description: "UNIX timestamp in seconds when the token will expire",
			Required:    true,
		},
		"refresh_token": kernel.FormFieldConfig{
			Description: "token for refresh of a pair of tokens",
			Required:    true,
		},
		"refresh_token_expired": kernel.FormFieldConfig{
			Description: "UNIX timestamp in seconds when the token will expire",
			Required:    true,
		},
	}
}

func NewTokensResponse() kernel.IForm {
	return lxHttp.PrepareForm(&TokensResponse{
		Form:    lxHttp.NewForm(),
		Success: true,
	})
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * PRIVATE
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

func genAuthCode(app cvn.IApp, ctx kernel.IHandleContext, user *models.User) error {
	// Get Authentication Params from Session for Authentication Code generating
	sess, err := session.ExtractSession(ctx)
	if err != nil {
		return errors.New("server configuration is wrong: sessions support required")
	}
	p := sess.Get("lxgo_auth_params")
	params, ok := p.(*AuthParams)
	if !ok {
		return errors.New("param 'lxgo_auth_params' must be in the session")
	}

	// Generating Authentication Code
	authCode, err := app.CodesRepo().Create(params.ClientID, user.ID)
	if err != nil {
		return fmt.Errorf("can not create code: %s", err)
	}
	sess.SetForce("lxgo_auth_code", authCode.Value)

	return nil
}

func errorResponse(handler kernel.IHttpResource, code, errCode int, msg string) kernel.IHttpResponse {
	resp := NewFailResponse().(*FailResponse)
	resp.ErrorCode = errCode
	resp.ErrorMessage = msg
	return handler.JsonResponse(kernel.JsonResponseConfig{
		Code: code,
		Form: resp,
	})
}

func serverErrorResponse(handler kernel.IHttpResource, logMsg string) kernel.IHttpResponse {
	handler.LogError(logMsg, "App")
	return errorResponse(handler, http.StatusInternalServerError, ERR_SERVER_ERROR, "Something went wrong")
}

type AuthParams struct {
	ResponseType string
	ClientID     uint
	RedirectUri  string
	State        string
}
