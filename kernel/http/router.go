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

func (router *Router) RegisterProxy(conf kernel.HttpProxyConfig) {
	for _, path := range conf.Routes {
		router.RegisterResource(path, "", newProxyHandler)
	}
	for path := range conf.Map {
		router.RegisterResource(path, "", newProxyHandler)
	}

	router.AddMiddleware(func(ctx kernel.IHandleContext) error {
		r := ctx.Route()
		if slices.Contains(conf.Routes, r) {
			ctx.Set("Server", conf.Server)
			return nil
		}
		path, exists := conf.Map[r]
		if exists {
			ctx.Set("Server", conf.Server)
			ctx.Set("Path", path)
			return nil
		}
		return nil
	})
}

func (router *Router) GetAssetRoute(path string) string {
	for urlPrefix, dir := range router.assetsMap {
		if dir == path {
			return urlPrefix
		}
	}
	return ""
}

func (router *Router) Handle(res kernel.IHttpResource, route string, w http.ResponseWriter, r *http.Request) kernel.IHttpResponse {
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

	return processResource(router, res)
}

func (router *Router) Start() {
	http.Handle("/", router)
}

func (router *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestedRoute := r.URL.Path
	if requestedRoute != "/" {
		requestedRoute, _ = strings.CutSuffix(requestedRoute, "/")
	}

	cResource, code := router.defineResource(requestedRoute, r.Method)
	if code != 0 {
		switch code {
		case http.StatusNotFound:
			http.NotFound(w, r)
		case http.StatusMethodNotAllowed:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		default:
			if router.app != nil {
				router.app.LogError(fmt.Sprintf("Can not define resource, code: %d", code), "HttpHandling")
			}
			http.Error(w, "Something went wrong", http.StatusInternalServerError)
		}
		return
	}

	res := cResource()
	res.Init()
	if response := router.Handle(res, requestedRoute, w, r); response != nil {
		ctx := res.Context()
		if router.app != nil {
			router.app.Events().Trigger(kernel.EVENT_APP_BEFORE_SEND_RESPONSE, kernel.NewData(map[string]any{
				"context":  ctx,
				"response": response,
			}))
		}
		response.Send(ctx.ResponseWriter())
	}
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * PRIVATE
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

func (router *Router) defineResource(requestedRoute, method string) (kernel.CHttpResource, int) {
	hList, ok := router.resources[requestedRoute]
	if !ok {
		return nil, http.StatusNotFound
	}

	availRoutes := slices.Collect(maps.Keys(router.resources))
	if !slices.Contains(availRoutes, requestedRoute) {
		return nil, http.StatusNotFound
	}

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
		return nil, http.StatusMethodNotAllowed
	}

	return cHandler, 0
}

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
			if resp = resource.ProcessRequestErrors(); resp != nil {
				return resp
			}
		}
	}

	resource.PreRun()
	resp = resource.Run()
	return resp
}
