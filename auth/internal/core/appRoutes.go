package core

import (
	"github.com/epicoon/lxgo/auth/internal/handlers"
	"github.com/epicoon/lxgo/kernel"
)

func InitRoutes(router kernel.IRouter) {
	router.RegisterResources(kernel.HttpResourcesList{
		"/":             handlers.NewFormHandler,
		"/auth[GET]":    handlers.NewGetAuthHandler,
		"/signup[POST]": handlers.NewSignupHandler,
		"/login[POST]":  handlers.NewLoginHandler,
		"/return[GET]":  handlers.NewReturnHandler,

		// API
		"/auth[POST]":     handlers.NewPostAuthHandler,
		"/tokens[POST]":   handlers.NewTokensHandler,
		"/refresh[POST]":  handlers.NewRefreshHandler,
		"/logout[POST]":   handlers.NewLogoutHandler,
		"/user-data[GET]": handlers.NewGetUserHandler,
	})
	router.RegisterFileAssets(map[string]string{
		"/js-form/":   "client/js/apps/form/dist",
		"/js-client/": "client/js/apps/client/dist",
		"/css/":       "client/css",
		"/img/":       "client/img",
	})
}
