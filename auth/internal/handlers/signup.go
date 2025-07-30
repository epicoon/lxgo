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
	Login    string `dict:"login"`
	Password string `dict:"password"`
}

/** @constructor */
func NewSignupRequest() *SignupRequest {
	req := &SignupRequest{Form: lxHttp.NewForm()}
	req.SetRequired([]string{"login", "password"})
	return req
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * SignupHandler
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */
/** @interface kernel.IHttpResource */
type SignupHandler struct {
	*lxHttp.Resource
}

/** @type kernel.CHttpResource */
func NewSignupHandler() kernel.IHttpResource {
	return &SignupHandler{Resource: lxHttp.NewResource()}
}

func (handler *SignupHandler) Run() kernel.IHttpResponse {
	coreApp, ok := handler.App().(cvn.IApp)
	if !ok {
		panic("app does not implement core.IApp")
	}

	// Checking Login and Password
	req := NewSignupRequest()
	lxHttp.FormFiller().SetContext(handler.Context()).SetForm(req).Fill()
	if req.HasErrors() {
		return errorResponse(handler, http.StatusBadRequest, ERR_NO_LOGIN_PWD, "Missing login or password")
	}

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
