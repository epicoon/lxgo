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

func (f *LoginRequest) Config() kernel.FormConfig {
	return kernel.FormConfig{
		"login": kernel.FormFieldConfig{
			Description: "user's login",
			Required:    true,
		},
		"password": kernel.FormFieldConfig{
			Description: "user's password",
			Required:    true,
		},
	}
}

/** @constructor */
func NewLoginRequest() kernel.IForm {
	return lxHttp.PrepareForm(&LoginRequest{Form: lxHttp.NewForm()})
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * LoginHandler
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */
/** @interface kernel.IHttpResource */
type LoginHandler struct {
	*BaseHandler
}

/** @constructor kernel.CHttpResource */
func NewLoginHandler() kernel.IHttpResource {
	return &LoginHandler{BaseHandler: NewBaseHandler(kernel.HttpResourceConfig{
		CRequestForm: NewLoginRequest,
	})}
}

func (handler *LoginHandler) ProcessRequestErrors() kernel.IHttpResponse {
	return errorResponse(handler, http.StatusBadRequest, ERR_NO_LOGIN_PWD, "Missing login or password")
}

func (handler *LoginHandler) Run() kernel.IHttpResponse {
	coreApp, ok := handler.App().(cvn.IApp)
	if !ok {
		panic("app does not implement core.IApp")
	}

	req := handler.RequestForm().(*LoginRequest)

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
