package cors_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/epicoon/lxgo/cors"
	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/apptest"
	lxHttp "github.com/epicoon/lxgo/kernel/http"
)

// pingHandler is a minimal kernel.IHttpResource, just enough to exercise
// the router's middleware pipeline.
type pingHandler struct {
	*lxHttp.Resource
}

func newPingHandler() kernel.IHttpResource {
	return &pingHandler{Resource: lxHttp.NewResource()}
}

func (h *pingHandler) Run() kernel.IHttpResponse {
	return h.JsonResponse(kernel.JsonResponseConfig{Data: map[string]any{"ok": true}})
}

// newTestApp builds an app with Cors configured (origins as given) and a
// "/ping" route registered, returning the app's Cors component and a
// running test server.
func newTestApp(t *testing.T, origins kernel.Dict) (cors.ICors, *httptest.Server) {
	t.Helper()
	app, err := apptest.New(kernel.Dict{
		"Components": kernel.Dict{
			"Cors": kernel.Dict{
				"Origins": origins,
			},
		},
	})
	if err != nil {
		t.Fatalf("apptest.New: %v", err)
	}
	if err := cors.SetAppComponent(app, "Components.Cors"); err != nil {
		t.Fatalf("SetAppComponent: %v", err)
	}
	c, err := cors.AppComponent(app)
	if err != nil {
		t.Fatalf("AppComponent: %v", err)
	}

	app.Router().RegisterResources(kernel.HttpResourcesList{
		"/ping":    newPingHandler,
		"/img/foo": newPingHandler,
		"/other":   newPingHandler,
	})

	handler, ok := app.Router().(http.Handler)
	if !ok {
		t.Fatalf("app.Router() does not implement http.Handler")
	}
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	return c, srv
}

func get(t *testing.T, srv *httptest.Server, path, origin string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, srv.URL+path, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	t.Cleanup(func() { resp.Body.Close() })
	return resp
}

func TestCors_EnableFor_AddsHeadersForConfiguredOrigin(t *testing.T) {
	c, srv := newTestApp(t, kernel.Dict{"http://example.com": kernel.Dict{}})
	c.EnableFor([]string{"/ping"})

	resp := get(t, srv, "/ping", "http://example.com")

	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "http://example.com" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want %q", got, "http://example.com")
	}
}

func TestCors_RouteNotEnabled_NoHeaders(t *testing.T) {
	c, srv := newTestApp(t, kernel.Dict{"http://example.com": kernel.Dict{}})
	c.EnableFor([]string{"/other"}) // "/ping" itself is never enabled

	resp := get(t, srv, "/ping", "http://example.com")

	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want empty (route not enabled)", got)
	}
}

func TestCors_OriginNotConfigured_NoHeaders(t *testing.T) {
	c, srv := newTestApp(t, kernel.Dict{"http://example.com": kernel.Dict{}})
	c.EnableFor([]string{"/ping"})

	resp := get(t, srv, "/ping", "http://not-allowed.com")

	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want empty (origin not in Origins)", got)
	}
}

func TestCors_NoOriginHeader_NoHeadersAdded(t *testing.T) {
	c, srv := newTestApp(t, kernel.Dict{"http://example.com": kernel.Dict{}})
	c.EnableFor([]string{"/ping"})

	resp := get(t, srv, "/ping", "")

	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want empty (no Origin header sent)", got)
	}
}

func TestCors_EnableForAll_AppliesToEveryRoute(t *testing.T) {
	c, srv := newTestApp(t, kernel.Dict{"http://example.com": kernel.Dict{}})
	c.EnableForAll()

	for _, route := range []string{"/ping", "/other", "/img/foo"} {
		resp := get(t, srv, route, "http://example.com")
		if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "http://example.com" {
			t.Fatalf("route %s: Access-Control-Allow-Origin = %q, want %q", route, got, "http://example.com")
		}
	}
}

func TestCors_DisableFor_WinsOverEnableForAll(t *testing.T) {
	c, srv := newTestApp(t, kernel.Dict{"http://example.com": kernel.Dict{}})
	c.EnableForAll()
	c.DisableFor([]string{"/ping"})

	resp := get(t, srv, "/ping", "http://example.com")
	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want empty (route excluded, wins over EnableForAll)", got)
	}

	// An unrelated route must be unaffected.
	resp2 := get(t, srv, "/other", "http://example.com")
	if got := resp2.Header.Get("Access-Control-Allow-Origin"); got != "http://example.com" {
		t.Fatalf("route /other: Access-Control-Allow-Origin = %q, want %q", got, "http://example.com")
	}
}

func TestCors_DisableFor_WinsOverEnableFor(t *testing.T) {
	c, srv := newTestApp(t, kernel.Dict{"http://example.com": kernel.Dict{}})
	c.EnableFor([]string{"/ping"})
	c.DisableFor([]string{"/ping"})

	resp := get(t, srv, "/ping", "http://example.com")
	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want empty (same route also excluded)", got)
	}
}

func TestCors_EnableFor_MatchesByPrefix(t *testing.T) {
	c, srv := newTestApp(t, kernel.Dict{"http://example.com": kernel.Dict{}})
	c.EnableFor([]string{"/img/"})

	resp := get(t, srv, "/img/foo", "http://example.com")
	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "http://example.com" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want %q (prefix match)", got, "http://example.com")
	}
}

func TestCors_ExposeHeaders_SetWhenConfigured(t *testing.T) {
	c, srv := newTestApp(t, kernel.Dict{
		"http://example.com": kernel.Dict{"ExposeHeaders": "X-Custom-Header"},
	})
	c.EnableFor([]string{"/ping"})

	resp := get(t, srv, "/ping", "http://example.com")
	if got := resp.Header.Get("Access-Control-Expose-Headers"); got != "X-Custom-Header" {
		t.Fatalf("Access-Control-Expose-Headers = %q, want %q", got, "X-Custom-Header")
	}
}

func TestCors_ExposeHeaders_AbsentWhenNotConfigured(t *testing.T) {
	c, srv := newTestApp(t, kernel.Dict{"http://example.com": kernel.Dict{}})
	c.EnableFor([]string{"/ping"})

	resp := get(t, srv, "/ping", "http://example.com")
	if got := resp.Header.Get("Access-Control-Expose-Headers"); got != "" {
		t.Fatalf("Access-Control-Expose-Headers = %q, want empty", got)
	}
}
