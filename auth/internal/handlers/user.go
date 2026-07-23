package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	cvn "github.com/epicoon/lxgo/auth/internal/conventions"
	"github.com/epicoon/lxgo/auth/internal/models"
	"github.com/epicoon/lxgo/auth/internal/repos"
	authClient "github.com/epicoon/lxgo/auth_client"
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
			Description: "user data previously reported by the client application (see POST /user-data); empty unless the access token's scope is 'profile:data' or no data was reported yet",
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
	// "data" is only granted with the wider scope - a "profile"-scoped token
	// gets identification (login) only.
	data := ""
	if accessToken.Scope == models.SCOPE_PROFILE_DATA {
		userData, err := coreApp.UsersRepo().FindData(user, client)
		if err != nil {
			return serverErrorResponse(handler, fmt.Sprintf("Can not find user data, user id='%d'", user.ID))
		}
		if userData != nil {
			encoded, err := json.Marshal(userData.Data)
			if err != nil {
				return serverErrorResponse(handler, fmt.Sprintf("Can not encode user data, user id='%d': %s", user.ID, err))
			}
			data = string(encoded)
		}
	}

	return handler.JsonResponse(kernel.JsonResponseConfig{
		Dict: kernel.Dict{
			"login": user.Login,
			"data":  data,
		},
	})
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * SetUserDataRequest
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */
/** @interface kernel.IForm */
type SetUserDataRequest struct {
	*lxHttp.Form
	ClientID uint   `json:"client_id"`
	Data     string `json:"data"`
}

var _ kernel.IForm = (*SetUserDataRequest)(nil)

func (f *SetUserDataRequest) Config() kernel.FormConfig {
	return kernel.FormConfig{
		"client_id": kernel.FormFieldConfig{
			Description: "client application identifier",
			Required:    true,
		},
		"data": kernel.FormFieldConfig{
			Description: "arbitrary JSON-encoded data the client application wants the authorization service to store for the current user, retrievable later via GET /user-data",
			Required:    true,
		},
	}
}

/** @constructor */
func NewSetUserDataRequest() kernel.IForm {
	return lxHttp.PrepareForm(&SetUserDataRequest{Form: lxHttp.NewForm()})
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * SetUserDataHandler
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */
/** @interface kernel.IHttpResource */
type SetUserDataHandler struct {
	*BaseHandler
}

var _ kernel.IHttpResource = (*SetUserDataHandler)(nil)

/** @constructor kernel.CHttpResource */
func NewSetUserDataHandler() kernel.IHttpResource {
	return &SetUserDataHandler{BaseHandler: NewBaseHandler(kernel.HttpResourceConfig{
		CRequestForm: NewSetUserDataRequest,
	})}
}

func (handler *SetUserDataHandler) Run() kernel.IHttpResponse {
	coreApp, ok := handler.App().(cvn.IApp)
	if !ok {
		panic("app does not implement core.IApp")
	}

	req := handler.RequestForm().(*SetUserDataRequest)

	// Check Client
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

	// Writing is gated the same way reading is - a token that can't read
	// "data" (see GetUserHandler.Run() above) shouldn't be able to write it
	// either.
	if accessToken.Scope != models.SCOPE_PROFILE_DATA {
		return errorResponse(handler, http.StatusForbidden, ERR_INSUFFICIENT_SCOPE, "Token scope does not allow storing user data")
	}

	var parsed models.JSONB
	if err := json.Unmarshal([]byte(req.Data), &parsed); err != nil {
		return errorResponse(handler, http.StatusBadRequest, ERR_COMMON, fmt.Sprintf("Invalid request: 'data' is not valid JSON: %s", err))
	}

	user, err := coreApp.UsersRepo().FindByToken(accessToken)
	if err != nil {
		if errors.Is(err, repos.ErrUserNotFound) {
			return errorResponse(handler, http.StatusNotFound, ERR_USER_NOT_FOUND, "User not found")
		}
		return serverErrorResponse(handler, fmt.Sprintf("Can not find user id='%d'", user.ID))
	}

	if _, err := coreApp.UsersRepo().SetData(user, client, parsed); err != nil {
		return serverErrorResponse(handler, fmt.Sprintf("Can not store user data, user id='%d': %s", user.ID, err))
	}

	return handler.JsonResponse(kernel.JsonResponseConfig{
		Form: NewSuccessResponse(),
	})
}
