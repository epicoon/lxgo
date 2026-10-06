package http

import (
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"

	"github.com/epicoon/lxgo/kernel"
)

/** @interface kernel.IRouter */

// Router is the default kernel.IRouter implementation - also implements
// http.Handler, so it can be registered directly with net/http (see Start).
type Router struct {
	app        kernel.IApp
	resources  map[string]kernel.HttpResourcesList
	templates  []*routeTemplate
	assetsMap  map[string]string
	middleware []kernel.FMiddleware
}

// routeTokenKind distinguishes a route template's segment kinds - see
// parseRouteTemplate.
type routeTokenKind int

const (
	routeTokenLiteral routeTokenKind = iota
	routeTokenParam
	routeTokenWildcard
)

// routeToken is one parsed segment of a route template.
type routeToken struct {
	kind    routeTokenKind
	literal string // routeTokenLiteral
	name    string // routeTokenParam or routeTokenWildcard
}

// routeTemplate is one registered route containing at least one "{name}" or
// "*{name}" segment - see parseRouteTemplate and Router's own doc comment.
type routeTemplate struct {
	raw      string // the route string as registered, to merge re-registrations
	tokens   []routeToken
	handlers kernel.HttpResourcesList
}

var _ kernel.IRouter = (*Router)(nil)

/** @constructor */

// NewRouter constructs an empty Router bound to app (may be nil for
// standalone use outside a kernel.IApp).
func NewRouter(app kernel.IApp) kernel.IRouter {
	return &Router{
		app:       app,
		resources: make(map[string]kernel.HttpResourcesList),
		assetsMap: make(map[string]string),
	}
}

// AddMiddleware registers a middleware, run before every request.
func (router *Router) AddMiddleware(mh kernel.FMiddleware) {
	if router.middleware == nil {
		router.middleware = make([]kernel.FMiddleware, 0, 1)
	}
	router.middleware = append(router.middleware, mh)
}

// Resources returns all registered resources, keyed by route then HTTP method.
func (router *Router) Resources() map[string]kernel.HttpResourcesList {
	return router.resources
}

// RegisterTemplates registers named templates, each served via GET at its
// own route with the template/params made available through the request context.
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

// RegisterResources registers a batch of routes - each key may embed its
// method as "path[METHOD]" (see parseRoute).
func (router *Router) RegisterResources(routes kernel.HttpResourcesList) {
	for route, cResource := range routes {
		path, method := parseRoute(route)
		router.RegisterResource(path, method, cResource)
	}
}

// RegisterResource registers a single route/method - an empty method
// matches any HTTP method not otherwise registered for the route. route may
// be a template (see Router's own doc comment) - panics if a "*{name}"
// segment isn't route's last one, if more than one appears, or if either
// placeholder form is given an empty name.
func (router *Router) RegisterResource(route string, method string, cResource kernel.CHttpResource) {
	method = strings.ToUpper(method)
	if method == "" {
		method = "ALL"
	}

	if tokens, isTemplate := parseRouteTemplate(route); isTemplate {
		for _, t := range router.templates {
			if t.raw == route {
				t.handlers[method] = cResource
				return
			}
		}
		router.templates = append(router.templates, &routeTemplate{
			raw:      route,
			tokens:   tokens,
			handlers: kernel.HttpResourcesList{method: cResource},
		})
		return
	}

	_, exists := router.resources[route]
	if !exists {
		router.resources[route] = make(kernel.HttpResourcesList)
	}

	router.resources[route][method] = cResource
}

// RegisterFileAssets registers static file routes: each key is a URL
// prefix, each value the directory it's served from (resolved via the
// app's pathfinder, if any). Served through the router's own pipeline
// (a "{urlPrefix}*{path}" template route, see RegisterResource) like any
// other resource - middleware runs for these requests too.
func (router *Router) RegisterFileAssets(assets map[string]string) {
	maps.Copy(router.assetsMap, assets)
	for urlPrefix, dir := range assets {
		route := strings.TrimSuffix(urlPrefix, "/") + "/*{" + assetPathParam + "}"
		router.RegisterResource(route, "", newAssetHandler(router.app, dir))
	}
}

// RegisterProxy registers conf.Routes/conf.Map's routes to be proxied
// through to conf.Server.
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

// GetAssetRoute returns the URL prefix registered for the directory path,
// or "" if none matches - the reverse of RegisterFileAssets.
func (router *Router) GetAssetRoute(path string) string {
	for urlPrefix, dir := range router.assetsMap {
		if dir == path {
			return urlPrefix
		}
	}
	return ""
}

// Handle initializes res's context for route/w/r, fires
// EVENT_APP_BEFORE_HANDLE_REQUEST, and runs it through the router's
// middleware/form-filling pipeline.
func (router *Router) Handle(res kernel.IHttpResource, route string, w http.ResponseWriter, r *http.Request) kernel.IHttpResponse {
	res.Context().Init(
		router.app,
		route,
		r.Method,
		w,
		r,
	)

	if router.app != nil {
		router.app.Events().Trigger(kernel.EVENT_APP_BEFORE_HANDLE_REQUEST, kernel.Dict{
			"context": res.Context(),
		})
	}

	return processResource(router, res)
}

// Start registers the router as the handler for "/" on the default net/http mux.
func (router *Router) Start() {
	http.Handle("/", router)
}

// ServeHTTP implements http.Handler: resolves the matching resource for the
// request, runs it, and sends its response.
func (router *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestedRoute := r.URL.Path
	if requestedRoute != "/" {
		requestedRoute, _ = strings.CutSuffix(requestedRoute, "/")
	}

	cResource, pathSegments, code := router.defineResource(requestedRoute, r.Method)
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
	res.Context().SetPathSegments(pathSegments)
	if response := router.Handle(res, requestedRoute, w, r); response != nil {
		ctx := res.Context()
		if router.app != nil {
			router.app.Events().Trigger(kernel.EVENT_APP_BEFORE_SEND_RESPONSE, kernel.Dict{
				"context":  ctx,
				"response": response,
			})
		}
		response.Send(ctx.ResponseWriter())
	}
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * PRIVATE
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

// defineResource resolves requestedRoute/method to a resource constructor -
// an exact (placeholder-free) route match always wins over every template
// (see Router's own doc comment); only once no exact route matches at all
// does it fall back to the best-matching template, if any. pathSegments is
// non-nil only for a template match.
func (router *Router) defineResource(requestedRoute, method string) (cResource kernel.CHttpResource, pathSegments map[string]string, code int) {
	if hList, ok := router.resources[requestedRoute]; ok {
		cResource, code = pickHandler(hList, method)
		return cResource, nil, code
	}

	segments := splitPath(requestedRoute)
	var best *templateMatch
	var bestHandlers kernel.HttpResourcesList
	for _, t := range router.templates {
		m, ok := matchTemplate(t, segments)
		if !ok {
			continue
		}
		if best == nil || m.moreSpecificThan(*best) {
			best = &m
			bestHandlers = t.handlers
		}
	}
	if best == nil {
		return nil, nil, http.StatusNotFound
	}

	cResource, code = pickHandler(bestHandlers, method)
	if code != 0 {
		return nil, nil, code
	}
	return cResource, best.params, 0
}

// pickHandler selects method's handler from hList, falling back to "ALL" -
// shared by both the exact-route and template-route resolution paths.
func pickHandler(hList kernel.HttpResourcesList, method string) (kernel.CHttpResource, int) {
	if try, exists := hList[method]; exists {
		return try, 0
	}
	if try, exists := hList["ALL"]; exists {
		return try, 0
	}
	return nil, http.StatusMethodNotAllowed
}

// splitPath splits route into its "/"-delimited segments, dropping the
// leading "/" - nil for the root route ("/") or an empty string.
func splitPath(route string) []string {
	trimmed := strings.TrimPrefix(route, "/")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "/")
}

// parseRouteTemplate splits route into tokens, recognizing "{name}" (exactly
// one path segment) and "*{name}" (every remaining segment, greedily, down
// to zero) placeholders. ok is false (tokens nil) for a route with no
// placeholder segment at all - the common case, registered as an exact
// route instead. Panics on a malformed template (see RegisterResource).
func parseRouteTemplate(route string) (tokens []routeToken, ok bool) {
	segments := splitPath(route)
	tokens = make([]routeToken, len(segments))
	hasPlaceholder := false

	for i, seg := range segments {
		switch {
		case strings.HasPrefix(seg, "*{") && strings.HasSuffix(seg, "}"):
			name := seg[2 : len(seg)-1]
			if name == "" {
				panic(fmt.Sprintf("lxgo-kernel: route %q has an unnamed wildcard segment", route))
			}
			if i != len(segments)-1 {
				panic(fmt.Sprintf("lxgo-kernel: route %q's wildcard segment %q must be its last one", route, seg))
			}
			tokens[i] = routeToken{kind: routeTokenWildcard, name: name}
			hasPlaceholder = true
		case strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}"):
			name := seg[1 : len(seg)-1]
			if name == "" {
				panic(fmt.Sprintf("lxgo-kernel: route %q has an unnamed path-parameter segment", route))
			}
			tokens[i] = routeToken{kind: routeTokenParam, name: name}
			hasPlaceholder = true
		default:
			tokens[i] = routeToken{kind: routeTokenLiteral, literal: seg}
		}
	}

	if !hasPlaceholder {
		return nil, false
	}
	return tokens, true
}

// templateMatch is one routeTemplate's successful match against a request's
// path segments - see matchTemplate.
type templateMatch struct {
	params      map[string]string
	hasWildcard bool
	paramCount  int
}

// moreSpecificThan reports whether m should be preferred over other when
// both match the same request (see Router's own doc comment): no wildcard
// beats having one; otherwise fewer named parameters wins.
func (m templateMatch) moreSpecificThan(other templateMatch) bool {
	if m.hasWildcard != other.hasWildcard {
		return !m.hasWildcard
	}
	return m.paramCount < other.paramCount
}

// matchTemplate reports whether segments (see splitPath) matches t's
// tokens, and if so, the resulting path-parameter values and the match's
// specificity.
func matchTemplate(t *routeTemplate, segments []string) (templateMatch, bool) {
	params := make(map[string]string, len(t.tokens))
	paramCount := 0

	for i, tok := range t.tokens {
		switch tok.kind {
		case routeTokenLiteral:
			if i >= len(segments) || segments[i] != tok.literal {
				return templateMatch{}, false
			}
		case routeTokenParam:
			if i >= len(segments) {
				return templateMatch{}, false
			}
			params[tok.name] = segments[i]
			paramCount++
		case routeTokenWildcard:
			// Always the template's last token (enforced by
			// parseRouteTemplate) - consumes every remaining segment,
			// including none (i == len(segments) is a valid, empty slice).
			params[tok.name] = strings.Join(segments[i:], "/")
			return templateMatch{params: params, hasWildcard: true, paramCount: paramCount}, true
		}
	}

	// No wildcard reached: every token must have consumed exactly one
	// segment, with nothing left over.
	if len(segments) != len(t.tokens) {
		return templateMatch{}, false
	}
	return templateMatch{params: params, paramCount: paramCount}, true
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
		if err := FormFiller().SetContext(ctx).SetForm(reqForm).Fill(); err != nil {
			msg := fmt.Sprintf("can not fill request form: %s", err)
			if router.app == nil {
				fmt.Println(msg)
			} else {
				router.app.LogError(msg, "HttpHandling")
			}
			http.Error(ctx.ResponseWriter(), "Internal server error", http.StatusInternalServerError)
			return nil
		}
		resource.SetRequestForm(reqForm)
		if reqForm.HasErrors() {
			if resp = resource.ProcessRequestErrors(); resp != nil {
				return resp
			}
		}
	}

	preHooks := resource.BeforeRunCallbacks()
	for _, f := range preHooks {
		f(resource)
	}

	resp = resource.Run()
	return resp
}
