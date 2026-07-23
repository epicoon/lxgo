package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"

	cvn "github.com/epicoon/lxgo/auth/internal/conventions"
	"github.com/epicoon/lxgo/auth/internal/repos"
	"github.com/epicoon/lxgo/kernel"
	lxHttp "github.com/epicoon/lxgo/kernel/http"
)

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * SignupRequest
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */
/** @interface kernel.IForm */
type SignupRequest struct {
	*lxHttp.Form
	Login    string `json:"login"`
	Password string `json:"password"`
}

func (f *SignupRequest) Config() kernel.FormConfig {
	return kernel.FormConfig{
		"login": kernel.FormFieldConfig{
			Description: "new user's login",
			Required:    true,
		},
		"password": kernel.FormFieldConfig{
			Description: "new user's password",
			Required:    true,
		},
	}
}

/** @constructor */
func NewSignupRequest() kernel.IForm {
	return lxHttp.PrepareForm(&SignupRequest{Form: lxHttp.NewForm()})
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * SignupHandler
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */
/** @interface kernel.IHttpResource */
type SignupHandler struct {
	*BaseHandler
}

/** @type kernel.CHttpResource */
func NewSignupHandler() kernel.IHttpResource {
	return &SignupHandler{BaseHandler: NewBaseHandler(kernel.HttpResourceConfig{
		CRequestForm: NewSignupRequest,
	})}
}

func (handler *SignupHandler) ProcessRequestErrors() kernel.IHttpResponse {
	return errorResponse(handler, http.StatusBadRequest, ERR_NO_LOGIN_PWD, "Missing login or password")
}

func (handler *SignupHandler) Run() kernel.IHttpResponse {
	coreApp, ok := handler.App().(cvn.IApp)
	if !ok {
		panic("app does not implement core.IApp")
	}

	req := handler.RequestForm().(*SignupRequest)

	if !validateLogin(req.Login) {
		return errorResponse(handler, http.StatusBadRequest, ERR_INVAL_LOGIN, "Invalid login format")
	}
	if !validatePassword(req.Password) {
		return errorResponse(handler, http.StatusBadRequest, ERR_INVAL_PWD, "Invalid password format")
	}

	// Try to create new User
	users := coreApp.UsersRepo()
	user, err := users.Create(req.Login, req.Password)
	if errors.Is(err, repos.ErrUserAlreadyExists) {
		return errorResponse(handler, http.StatusConflict, ERR_LOGIN_EXISTS, "Login already exists")
	}
	if err != nil {
		return serverErrorResponse(handler, fmt.Sprintf("can not create new User: %s", err))
	}

	// Generating Authentication Code
	if err := genAuthCode(coreApp, handler.Context(), user); err != nil {
		return serverErrorResponse(handler, err.Error())
	}

	return handler.JsonResponse(kernel.JsonResponseConfig{
		Form: NewSuccessResponse(),
	})
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * PRIVATE
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

func validateLogin(login string) bool {
	l := len(login)
	if l < 3 || l > 20 {
		return false
	}
	doubleRegex := regexp.MustCompile(`(\.\.|__)`)
	loginRegex := regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_.]+$`)
	return !doubleRegex.MatchString(login) && loginRegex.MatchString(login)
}

func validatePassword(password string) bool {
	if len(password) < 8 {
		return false
	}
	numbers := regexp.MustCompile(`\d`)
	lowercase := regexp.MustCompile(`[a-z]`)
	uppercase := regexp.MustCompile(`[A-Z]`)
	specialChar := regexp.MustCompile(`[!@#$%^&*(),.?":{}|<>]`)
	return numbers.MatchString(password) && lowercase.MatchString(password) && uppercase.MatchString(password) && specialChar.MatchString(password)
}
