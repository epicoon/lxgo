package handlers

import (
	"net/http"

	cvn "github.com/epicoon/lxgo/auth/internal/conventions"
	"github.com/epicoon/lxgo/kernel"
	lxHttp "github.com/epicoon/lxgo/kernel/http"
)

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * LoginRequest
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */
/** @interface kernel.IForm */
type LoginRequest struct {
	*lxHttp.Form
	Login    string `json:"login"`
	Password string `json:"password"`
}

/** @constructor */
func NewLoginRequest() *LoginRequest {
	req := &LoginRequest{Form: lxHttp.NewForm()}
	req.SetRequired([]string{"login", "password"})
	return req
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * LoginHandler
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */
/** @interface kernel.IHttpResource */
type LoginHandler struct {
	*lxHttp.Resource
}

/** @constructor kernel.CHttpResource */
func NewLoginHandler() kernel.IHttpResource {
	return &LoginHandler{Resource: lxHttp.NewResource()}
}

func (handler *LoginHandler) Run() kernel.IHttpResponse {
	coreApp, ok := handler.App().(cvn.IApp)
	if !ok {
		panic("app does not implement core.IApp")
	}

	// Checking Login and Password
	req := NewLoginRequest()
	lxHttp.FormFiller().SetContext(handler.Context()).SetForm(req).Fill()
	if req.HasErrors() {
		return errorResponse(handler, http.StatusBadRequest, ERR_NO_LOGIN_PWD, "Missing login or password")
	}

	// Find User
	user, err := coreApp.UsersRepo().FindByLP(req.Login, req.Password)
	if err != nil {
		return errorResponse(handler, http.StatusUnauthorized, ERR_WRONG_LOGIN_PWD, "Wrong login or password")
	}

	// Generating Authentication Code
	err = genAuthCode(coreApp, handler.Context(), user)
	if err != nil {
		return serverErrorResponse(handler, err.Error())
	}

	return handler.JsonResponse(kernel.JsonResponseConfig{
		Form: NewSuccessResponse(),
	})
}
