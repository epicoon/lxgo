package src

import (
	"net/http"
	"testing"

	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/apptest"
	lxHttp "github.com/epicoon/lxgo/kernel/http"
)

// wsTestForm is a minimal kernel.IForm for Router.Handle tests - mirrors
// lxgo-kernel/http's own testForm, just local to this package to avoid
// depending on an unexported test type from another module.
type wsTestForm struct {
	*lxHttp.Form
	Name string `json:"name"`
	Age  int    `dict:"age"`
}

func newWsTestForm() *wsTestForm {
	return &wsTestForm{Form: lxHttp.NewForm()}
}

type wsTestResource struct {
	*lxHttp.Resource
	ran               bool
	cReqForm          kernel.CForm
	processErrorsResp kernel.IHttpResponse
	beforeRunOrder    *[]string
}

func newWsTestResource() *wsTestResource {
	return &wsTestResource{Resource: lxHttp.NewResource()}
}

func (r *wsTestResource) Run() kernel.IHttpResponse {
	r.ran = true
	resp := &lxHttp.Response{}
	if err := resp.SetJsonData(r.Context().Params()); err != nil {
		panic(err)
	}
	return resp
}

func (r *wsTestResource) ProcessRequestErrors() kernel.IHttpResponse {
	return r.processErrorsResp
}

func (r *wsTestResource) CRequestForm() kernel.CForm {
	return r.cReqForm
}

func newTestApp(t *testing.T) kernel.IApp {
	t.Helper()
	app, err := apptest.New()
	if err != nil {
		t.Fatalf("apptest.New: %v", err)
	}
	return app
}

// TestConnection_LxwsRequest_RoundTripsThroughRealMessageLoop is a
// regression test: processRequest built its target struct via
// `msgStruct := new(struct{...})` (already a pointer) and then passed
// `&msgStruct` to cast.MapToStruct - a pointer to that pointer.
// DictToStruct only unwraps one level of pointer before checking for a
// struct, so every real __lxws_request__ message failed with "invalid
// __lxws_request__ struct format: provided value is not a struct",
// regardless of route or payload - Router.Handle itself (tested directly
// above, bypassing processRequest) never exercised this and so never
// caught it.
func TestConnection_LxwsRequest_RoundTripsThroughRealMessageLoop(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)
	s.app = newTestApp(t)

	s.Router().RegisterResource("/echo", func() kernel.IHttpResource {
		return newWsTestResource()
	})

	c := newWSTestClient(t, s, "")
	c.readHandshakeResponse()
	c.readJSON() // handshake ack

	c.sendJSON(map[string]any{
		"__lxws_request__": map[string]any{"route": "/echo", "key": "k1"},
		"__data__":         map[string]any{"a": 1},
	})

	resp := c.readJSON()
	if errMsg, hasErr := resp["error"]; hasErr {
		t.Fatalf("expected a successful __lxws_response__, got an error instead: %v", errMsg)
	}
	if resp["__lxws_response__"] != true {
		t.Fatalf("expected a __lxws_response__ message, got %#v", resp)
	}
	if resp["key"] != "k1" {
		t.Fatalf("expected the response to echo the request key, got %#v", resp)
	}
	code, _ := resp["code"].(float64)
	if int(code) != http.StatusOK {
		t.Fatalf("expected code 200, got %#v", resp["code"])
	}
}

func TestRouter_Handle_UnknownRoute404(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)
	s.app = newTestApp(t)

	resp := s.Router().Handle("nope", map[string]any{})
	if resp.Code() != http.StatusNotFound {
		t.Fatalf("expected 404 for an unregistered route, got %d", resp.Code())
	}
}

func TestRouter_Handle_DispatchesAndPassesParams(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)
	s.app = newTestApp(t)

	var gotResource *wsTestResource
	s.Router().RegisterResource("echo", func() kernel.IHttpResource {
		r := newWsTestResource()
		gotResource = r
		return r
	})

	resp := s.Router().Handle("echo", map[string]any{"a": float64(1)})
	if !gotResource.ran {
		t.Fatalf("expected Run() to be called on the registered resource")
	}
	if resp.Code() != http.StatusOK {
		t.Fatalf("expected the resource's own 200 response untouched, got %d", resp.Code())
	}
	if resp.Data() != `{"a":1}` {
		t.Fatalf("expected the dispatched params echoed back, got %q", resp.Data())
	}
}

func TestRouter_RegisterResource_DoesNotOverwriteExisting(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)
	s.app = newTestApp(t)

	firstRan, secondRan := false, false
	s.Router().RegisterResource("r", func() kernel.IHttpResource {
		firstRan = true
		return newWsTestResource()
	})
	s.Router().RegisterResource("r", func() kernel.IHttpResource {
		secondRan = true
		return newWsTestResource()
	})

	s.Router().Handle("r", map[string]any{})
	if !firstRan || secondRan {
		t.Fatalf("expected the first registration to win, got firstRan=%v secondRan=%v", firstRan, secondRan)
	}
}

// TestRouter_Handle_UnCoercibleParam_SurfacesAsFormErrorNotFillError documents
// the actual, reachable contract at this call site: a per-field cast
// failure (e.g. a map value where an int is expected) is collected onto the
// form as a validation error (HasErrors()) by cast.DictToStruct/CollectErrorf
// - it does NOT make FormFiller().Fill() itself return a Go error. Handle's
// own "can not fill request form" 500 branch guards FormFiller() misuse
// (no form/no dict set) - which can't happen here, since Handle always
// calls SetDict+SetForm together - so that branch is effectively
// unreachable through this call site, not a gap worth chasing.
func TestRouter_Handle_UnCoercibleParam_SurfacesAsFormErrorNotFillError(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)
	s.app = newTestApp(t)

	var gotResource *wsTestResource
	s.Router().RegisterResource("r", func() kernel.IHttpResource {
		r := newWsTestResource()
		r.cReqForm = func() kernel.IForm { return newWsTestForm() }
		gotResource = r
		return r
	})

	resp := s.Router().Handle("r", map[string]any{"age": map[string]any{"nested": true}})
	if resp.Code() == http.StatusInternalServerError {
		t.Fatalf("an un-coercible field must not produce a hard 500 from this call site, got %d: %s", resp.Code(), resp.Data())
	}
	if !gotResource.RequestForm().HasErrors() {
		t.Fatalf("expected the un-coercible field to land as a form validation error")
	}
}

func TestRouter_Handle_RequestFormErrors_ProcessRequestErrorsShortCircuits(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)
	s.app = newTestApp(t)

	custom := &lxHttp.Response{}
	custom.SetCode(http.StatusTeapot)
	var gotResource *wsTestResource
	s.Router().RegisterResource("r", func() kernel.IHttpResource {
		r := newWsTestResource()
		f := newWsTestForm()
		f.SetRequired([]string{"name"})
		r.cReqForm = func() kernel.IForm { return f }
		r.processErrorsResp = custom
		gotResource = r
		return r
	})

	resp := s.Router().Handle("r", map[string]any{}) // missing required "name"
	if resp != custom {
		t.Fatalf("expected ProcessRequestErrors' own response to be returned, got %#v", resp)
	}
	if gotResource.ran {
		t.Fatalf("Run() must not be called when ProcessRequestErrors short-circuits")
	}
}

func TestRouter_Handle_RequestFormErrors_NilProcessRequestErrorsFallsThroughToRun(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)
	s.app = newTestApp(t)

	var gotResource *wsTestResource
	s.Router().RegisterResource("r", func() kernel.IHttpResource {
		r := newWsTestResource()
		f := newWsTestForm()
		f.SetRequired([]string{"name"})
		r.cReqForm = func() kernel.IForm { return f }
		r.processErrorsResp = nil
		gotResource = r
		return r
	})

	resp := s.Router().Handle("r", map[string]any{})
	if !gotResource.ran {
		t.Fatalf("expected Run() to still be called when ProcessRequestErrors returns nil")
	}
	if resp == nil {
		t.Fatalf("expected a non-nil response from Run()")
	}
	if !gotResource.RequestForm().HasErrors() {
		t.Fatalf("expected the request form to still carry its validation error")
	}
}

func TestRouter_Handle_BeforeRunCallbacksRunBeforeRun(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)
	s.app = newTestApp(t)

	var order []string
	s.Router().RegisterResource("r", func() kernel.IHttpResource {
		r := newWsTestResource()
		r.BeforeRun(func(res kernel.IHttpResource) { order = append(order, "before") })
		return r
	})

	orderBefore := len(order)
	s.Router().Handle("r", map[string]any{})
	if orderBefore != 0 || len(order) != 1 || order[0] != "before" {
		t.Fatalf("expected exactly one BeforeRun callback to have run before Run, got %v", order)
	}
}

func TestRouter_Handle_TriggersLifecycleEvents(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)
	app := newTestApp(t)
	s.app = app

	var fired []string
	app.Events().Subscribe(kernel.EVENT_APP_BEFORE_HANDLE_REQUEST, func(e kernel.IEvent) {
		fired = append(fired, "beforeHandle")
	})
	app.Events().Subscribe(kernel.EVENT_APP_BEFORE_SEND_RESPONSE, func(e kernel.IEvent) {
		fired = append(fired, "beforeSend")
	})

	s.Router().RegisterResource("r", func() kernel.IHttpResource { return newWsTestResource() })
	s.Router().Handle("r", map[string]any{})

	if len(fired) != 2 || fired[0] != "beforeHandle" || fired[1] != "beforeSend" {
		t.Fatalf("expected both lifecycle events to fire in order, got %v", fired)
	}
}
