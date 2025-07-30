package client

import (
	"fmt"
	"net/http"

	"github.com/epicoon/lxgo/kernel"
	lxHttp "github.com/epicoon/lxgo/kernel/http"
)

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * LogoutHandler
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */
/** @interface kernel.IHttpResource */
type LogoutHandler struct {
	*lxHttp.Resource
}

/** @constructor kernel.CHttpResource */
func NewLogoutHandler() kernel.IHttpResource {
	return &LogoutHandler{Resource: &lxHttp.Resource{}}
}

func (handler *LogoutHandler) Run() kernel.IHttpResponse {
	authClient, err := AppComponent(handler.App())
	if err != nil {
		handler.LogError("wrong application configuration: auth_client component required", "App")
		return handler.ErrorResponse(http.StatusInternalServerError, "Something went wrong")
	}

	accessToken, err := GetBearer(handler.Context())
	if err != nil {
		return handler.JsonResponse(kernel.JsonResponseConfig{
			Code: http.StatusUnauthorized,
			Data: err,
		})
	}

	if err := authClient.LogOut(accessToken); err != nil {
		handler.LogError(fmt.Sprintf("can not logout: %s", err), "App")
		return handler.ErrorResponse(http.StatusInternalServerError, "Something went wrong")
	}

	return handler.JsonResponse(kernel.JsonResponseConfig{
		Data: map[string]any{
			"success": "ok",
		},
	})
}
