package src

import (
	"fmt"
	"net/http"

	"github.com/epicoon/lxgo/kernel"
	lxHttp "github.com/epicoon/lxgo/kernel/http"
	"github.com/epicoon/lxgo/ws"
)

type Router struct {
	server    ws.IWSServer
	resources kernel.HttpResourcesList
}

func NewRouter(s ws.IWSServer) ws.IRouter {
	return &Router{
		server:    s,
		resources: make(kernel.HttpResourcesList),
	}
}

func (router *Router) RegisterResources(routes kernel.HttpResourcesList) {
	for route, cResource := range routes {
		router.RegisterResource(route, cResource)
	}
}

func (router *Router) RegisterResource(route string, cResource kernel.CHttpResource) {
	_, exists := router.resources[route]
	if !exists {
		router.resources[route] = cResource
	}
}

func (router *Router) Handle(route string, params map[string]any) kernel.IHttpResponse {
	cResource, exists := router.resources[route]
	if !exists {
		resp := &lxHttp.Response{}
		resp.SetCode(http.StatusNotFound)
		return resp
	}

	res := cResource()
	res.Init()

	ctx := lxHttp.NewHandleContext(router.server.App(), route, res)
	ctx.SetParams(params)
	res.SetContext(ctx)

	router.server.App().Events().Trigger(kernel.EVENT_APP_BEFORE_HANDLE_REQUEST, kernel.NewData(map[string]any{
		"context": ctx,
	}))

	var resp kernel.IHttpResponse
	cReq := res.CRequestForm()
	if cReq != nil {
		reqForm := cReq()
		if err := lxHttp.FormFiller().SetDict(kernel.Dict(params)).SetForm(reqForm).Fill(); err != nil {
			router.server.App().LogError(fmt.Sprintf("can not fill request form: %s", err), "Handling")
			resp := &lxHttp.Response{}
			resp.SetError(http.StatusInternalServerError, "Something went wrong")
			return resp
		}
		res.SetRequestForm(reqForm)
		if reqForm.HasErrors() {
			if resp = res.ProcessRequestErrors(); resp != nil {
				return resp
			}
		}
	}

	preHooks := res.BeforeRunCallbacks()
	for _, f := range preHooks {
		f(res)
	}

	resp = res.Run()

	if resp != nil {
		router.server.App().Events().Trigger(kernel.EVENT_APP_BEFORE_SEND_RESPONSE, kernel.NewData(map[string]any{
			"context":  ctx,
			"response": resp,
		}))
	}

	return resp
}
