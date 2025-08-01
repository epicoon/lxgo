package http

import (
	"fmt"
	"maps"
	"net/http"
	"path/filepath"
	"slices"
	"strings"

	"github.com/epicoon/lxgo/kernel"
)

/** @interface kernel.IRouter */
type Router struct {
	app        kernel.IApp
	resources  map[string]kernel.HttpResourcesList
	assetsMap  map[string]string
	middleware []kernel.FMiddleware
}

var _ kernel.IRouter = (*Router)(nil)

/** @constructor */
func NewRouter(app kernel.IApp) kernel.IRouter {
	return &Router{
		app:       app,
		resources: make(map[string]kernel.HttpResourcesList),
		assetsMap: make(map[string]string),
	}
}

func (router *Router) AddMiddleware(mh kernel.FMiddleware) {
	if router.middleware == nil {
		router.middleware = make([]kernel.FMiddleware, 0, 1)
	}
	router.middleware = append(router.middleware, mh)
}

func (router *Router) Resources() map[string]kernel.HttpResourcesList {
	return router.resources
}

func (router *Router) RegisterTemplates(tpls kernel.HttpTemplatesList) {
	for url := range tpls {
		router.RegisterResource(url, "GET", newStdHandler)
	}

	router.AddMiddleware(func(ctx kernel.IHandleContext) error {
		r := ctx.Route()
		options, exists := tpls[r]
		if !exists {
			return nil
		}

		ctx.Set("Template", options.Template)
		ctx.Set("Params", options.Params)
		return nil
	})
}

func (router *Router) RegisterResources(routes kernel.HttpResourcesList) {
	for route, cResource := range routes {
		path, method := parseRoute(route)
		router.RegisterResource(path, method, cResource)
	}
}

func (router *Router) RegisterResource(route string, method string, cResource kernel.CHttpResource) {
	method = strings.ToUpper(method)

	_, exists := router.resources[route]
	if !exists {
		router.resources[route] = make(kernel.HttpResourcesList)
	}

	if method == "" {
		method = "ALL"
	}

	router.resources[route][method] = cResource
}

func (router *Router) RegisterFileAssets(assets map[string]string) {
	maps.Copy(router.assetsMap, assets)
	for urlPrefix, dir := range assets {
		http.Handle(urlPrefix, http.StripPrefix(urlPrefix, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var filePath string
			if router.app == nil {
				filePath = filepath.Join(dir, r.URL.Path)
			} else {
				filePath = filepath.Join(router.app.Pathfinder().GetAbsPath(dir), r.URL.Path)
				router.app.Events().Trigger(kernel.EVENT_APP_BEFORE_SEND_ASSET, kernel.NewData(map[string]any{
					"request": r,
					"file":    filePath,
				}))
			}
			http.ServeFile(w, r, filePath)
		})))
	}
}

func (router *Router) GetAssetRoute(path string) string {
	for urlPrefix, dir := range router.assetsMap {
		if dir == path {
			return urlPrefix
		}
	}
	return ""
}

func (router *Router) Handle(res kernel.IHttpResource, route string, w http.ResponseWriter, r *http.Request) {
	ctx := &HandleContext{
		app:      router.app,
		route:    route,
		method:   r.Method,
		writer:   w,
		request:  r,
		resource: res,
	}
	res.SetContext(ctx)

	if router.app != nil {
		router.app.Events().Trigger(kernel.EVENT_APP_BEFORE_HANDLE_REQUEST, kernel.NewData(map[string]any{
			"context": ctx,
		}))
	}

	if response := processResource(router, res); response != nil {
		if router.app != nil {
			router.app.Events().Trigger(kernel.EVENT_APP_BEFORE_SEND_RESPONSE, kernel.NewData(map[string]any{
				"context":  ctx,
				"response": response,
			}))
		}
		response.Send(ctx.ResponseWriter())
	}
}

func (router *Router) Start() {
	availRoutes := slices.Collect(maps.Keys(router.resources))

	for route, hList := range router.resources {
		http.Handle(route, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cResource := defineResource(w, r, hList, availRoutes)
			if cResource == nil {
				return
			}

			res := cResource()
			res.Init()
			router.Handle(res, route, w, r)
		}))
	}
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * PRIVATE
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

func parseRoute(route string) (string, string) {
	if strings.Contains(route, "[") && strings.Contains(route, "]") {
		start := strings.Index(route, "[")
		end := strings.Index(route, "]")
		method := route[start+1 : end]
		path := route[:start]
		return path, method
	}
	return route, ""
}

func defineResource(w http.ResponseWriter, r *http.Request, hList kernel.HttpResourcesList, availRoutes []string) kernel.CHttpResource {
	requestedRoute := r.URL.Path
	if !slices.Contains(availRoutes, requestedRoute) {
		http.NotFound(w, r)
		return nil
	}

	method := r.Method
	var cHandler kernel.CHttpResource
	try, exists := hList[method]
	if exists {
		cHandler = try
	} else {
		try, exists := hList["ALL"]
		if exists {
			cHandler = try
		}
	}

	if cHandler == nil {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return nil
	}

	return cHandler
}

func processResource(router *Router, resource kernel.IHttpResource) kernel.IHttpResponse {
	ctx := resource.Context()
	for _, mw := range router.middleware {
		err := mw(ctx)
		if err != nil {
			msg := fmt.Sprintf("can not process middleware: %s", err)
			if router.app == nil {
				fmt.Println(msg)
			} else {
				router.app.LogError(msg, "HttpHandling")
			}
			http.Error(ctx.ResponseWriter(), "Internal server error", http.StatusInternalServerError)
			return nil
		}
	}

	var resp kernel.IHttpResponse
	cReq := resource.CRequestForm()
	if cReq != nil {
		reqForm := cReq()
		FormFiller().SetContext(ctx).SetForm(reqForm).Fill()
		resource.SetRequestForm(reqForm)
		if reqForm.HasErrors() {
			resp = resource.ProcessRequestErrors()
		}
	}

	if resp == nil {
		resource.PreRun()
		resp = resource.Run()
	}

	return resp
}
