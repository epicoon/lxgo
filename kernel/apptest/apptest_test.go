package apptest_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/app"
	"github.com/epicoon/lxgo/kernel/apptest"
	lxHttp "github.com/epicoon/lxgo/kernel/http"
)

func TestNew_Defaults(t *testing.T) {
	a, err := apptest.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a == nil {
		t.Fatal("expected a non-nil app")
	}
	if a.Router() == nil {
		t.Fatal("expected the app to have a router")
	}
	if !a.Config().Has("Port") {
		t.Fatal("expected a default 'Port' key in config")
	}
}

func TestNew_CustomConfig(t *testing.T) {
	a, err := apptest.New(kernel.Dict{"Port": 9090, "Custom": "value"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val := a.ConfigParam("Custom"); val != "value" {
		t.Fatalf("got Custom=%v, want 'value'", val)
	}
	if val := a.ConfigParam("Port"); val != 9090 {
		t.Fatalf("got Port=%v, want 9090", val)
	}
}

func TestNew_InvalidConfig(t *testing.T) {
	// Port must coerce to int - a value that can't (e.g. a nested dict)
	// should surface as an error from InitApp, not panic or silently pass.
	_, err := apptest.New(kernel.Dict{"Port": kernel.Dict{}})
	if err == nil {
		t.Fatal("expected an error for a non-int Port")
	}
}

type echoResource struct {
	*lxHttp.Resource
}

func (r *echoResource) Run() kernel.IHttpResponse {
	return r.JsonResponse(kernel.JsonResponseConfig{Data: kernel.Dict{"ok": true}})
}

func TestServer_RoundTrip(t *testing.T) {
	a, err := apptest.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	a.Router().RegisterResource("/echo", "GET", func() kernel.IHttpResource {
		return &echoResource{Resource: lxHttp.NewResource()}
	})

	srv := apptest.Server(a)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/echo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("got status %d, want 200", resp.StatusCode)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("unexpected error decoding body: %v", err)
	}
	if body["ok"] != true {
		t.Fatalf("got body %v, want {\"ok\":true}", body)
	}
}

type testComponent struct {
	*app.AppComponent
	afterInitCalled bool
}

func (c *testComponent) Name() string { return "test" }

func (c *testComponent) AfterInit() {
	c.afterInitCalled = true
}

func TestRegisterComponent_OnBuiltApp(t *testing.T) {
	a, err := apptest.New(kernel.Dict{"MyComponent": kernel.Dict{}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	c := &testComponent{AppComponent: app.NewAppComponent()}
	if err := app.RegisterComponent(a, c, "test", "MyComponent"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !a.HasComponent("test") {
		t.Fatal("expected the component to be registered")
	}
	if !c.afterInitCalled {
		t.Fatal("expected AfterInit to have been called")
	}
	if c.App() != a {
		t.Fatal("expected the component to be bound to the built app")
	}
}
