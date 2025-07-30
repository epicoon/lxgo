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
 * GetUserRequest
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */
/** @interface kernel.IForm */
type GetUserRequest struct {
	*lxHttp.Form
	ClientID uint `json:"client_id"`
}

var _ kernel.IForm = (*GetUserRequest)(nil)

func (f *GetUserRequest) Config() kernel.FormConfig {
	return kernel.FormConfig{
		"client_id": kernel.FormFieldConfig{
			Description: "client application identifier",
			Required:    true,
		},
	}
}

/** @constructor */
func NewGetUserRequest() kernel.IForm {
	return lxHttp.PrepareForm(&GetUserRequest{Form: lxHttp.NewForm()})
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * GetUserResponse
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */
/** @interface kernel.IForm */
type GetUserResponse struct {
	*lxHttp.Form
	Success bool   `json:"success"`
	Login   string `json:"login"`
	Data    string `json:"data"`
}

var _ kernel.IForm = (*GetUserResponse)(nil)

func (f *GetUserResponse) Config() kernel.FormConfig {
	return kernel.FormConfig{
		"success": kernel.FormFieldConfig{
			Description: "equals true",
			Required:    true,
		},
		"login": kernel.FormFieldConfig{
			Description: "unique user login in the system",
			Required:    true,
		},
		"data": kernel.FormFieldConfig{
			Description: "user data previously reported by the client application (! not implemented !)",
			Required:    true,
		},
	}
}

/** @constructor */
func NewGetUserResponse() kernel.IForm {
	return lxHttp.PrepareForm(&GetUserResponse{
		Form:    lxHttp.NewForm(),
		Success: true,
	})
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * GetUserHandler
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */
/** @interface kernel.IHttpResource */
type GetUserHandler struct {
	*BaseHandler
}

var _ kernel.IHttpResource = (*GetUserHandler)(nil)

/** constructor kernel.CHttpResource */
func NewGetUserHandler() kernel.IHttpResource {
	return &GetUserHandler{BaseHandler: NewBaseHandler(kernel.HttpResourceConfig{
		CRequestForm:  NewGetUserRequest,
		CResponseForm: NewGetUserResponse,
	})}
}

func (handler *GetUserHandler) Run() kernel.IHttpResponse {
	coreApp, ok := handler.App().(cvn.IApp)
	if !ok {
		panic("app does not implement core.IApp")
	}

	// Check Client
	req := handler.RequestForm().(*GetUserRequest)
	client, err := coreApp.ClientsRepo().FindByID(req.ClientID)
	if err != nil {
		if errors.Is(err, repos.ErrClientNotFound) {
			return errorResponse(handler, http.StatusNotFound, ERR_CLIENT_NOT_FOUND, "Client not found")
		}
		return serverErrorResponse(handler, fmt.Sprintf("Can not find client '%d'", req.ClientID))
	}

	// Check Access Token
	accessValue, err := authClient.GetBearer(handler.Context())
	if err != nil {
		return errorResponse(handler, http.StatusUnauthorized, ERR_INVAL_AUTH_HEADER, fmt.Sprintf("Request issue: %s", err))
	}
	accessToken, err := coreApp.TokensRepo().FindAccessToken(client, accessValue)
	if err != nil {
		return errorResponse(handler, http.StatusUnauthorized, ERR_TOKEN_NOT_FOUND, "Token not found")
	}
	if accessToken.IsExpired() {
		return errorResponse(handler, http.StatusUnauthorized, ERR_TOKEN_EXPIRED, "Token expired")
	}

	user, err := coreApp.UsersRepo().FindByToken(accessToken)
	if err != nil {
		if errors.Is(err, repos.ErrUserNotFound) {
			return errorResponse(handler, http.StatusNotFound, ERR_USER_NOT_FOUND, "User not found")
		}
		return serverErrorResponse(handler, fmt.Sprintf("Can not find user id='%d'", user.ID))
	}
	userData, err := coreApp.UsersRepo().FindData(user)
	if err != nil {
		return serverErrorResponse(handler, fmt.Sprintf("Can not find user data, user id='%d'", user.ID))
	}

	//TODO
	_ = userData

	return handler.JsonResponse(kernel.JsonResponseConfig{
		Dict: kernel.Dict{
			"login": user.Login,
			"data":  "{\"test\": 1}",
		},
	})
}
