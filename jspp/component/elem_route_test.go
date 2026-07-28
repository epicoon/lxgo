package component_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/epicoon/lxgo/jspp"
	"github.com/epicoon/lxgo/jspp/component"
	"github.com/epicoon/lxgo/jspp/elems"
	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/apptest"
	lxHttp "github.com/epicoon/lxgo/kernel/http"
)

// testElement is a minimal jspp.IElement with one ajax route, registered
// under the DI key "testElem" below.
type testElement struct {
	*elems.Element
}

func (e *testElement) AjaxHandlers() kernel.HttpResourcesList {
	return kernel.HttpResourcesList{
		"ping": func() kernel.IHttpResource {
			return &pingHandler{Resource: lxHttp.NewResource()}
		},
	}
}

type pingHandler struct {
	*lxHttp.Resource
}

func (h *pingHandler) Run() kernel.IHttpResponse {
	return h.JsonResponse(kernel.JsonResponseConfig{
		Data: map[string]any{"pong": true},
	})
}

// TestElemRoute_RealHTTPRoundTrip registers a real jspp.IElement in the
// app's DI container and dispatches a real HTTP request to /lx/elem, which
// exercises the "jspp" context middleware, ElemHandler, and the element's
// own AjaxHandlers routing end to end.
func TestElemRoute_RealHTTPRoundTrip(t *testing.T) {
	sysPath := t.TempDir()
	app, err := apptest.New(kernel.Dict{
		"Components": kernel.Dict{
			"JSPreprocessor": kernel.Dict{
				"SysPath":  sysPath,
				"MapsPath": sysPath,
			},
		},
	})
	if err != nil {
		t.Fatalf("apptest.New: %v", err)
	}
	if err := component.SetAppComponent(app, "Components.JSPreprocessor"); err != nil {
		t.Fatalf("SetAppComponent: %v", err)
	}
	if err := app.DIContainer().Register(kernel.CAnyList{
		"testElem": func(...any) any {
			return &testElement{Element: elems.NewElement()}
		},
	}); err != nil {
		t.Fatalf("DIContainer().Register: %v", err)
	}

	srv := apptest.Server(app)
	defer srv.Close()

	body, _ := json.Marshal(map[string]any{
		"elem":   "testElem",
		"path":   "ping",
		"params": map[string]any{},
	})
	resp, err := http.Post(srv.URL+"/lx/elem", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /lx/elem: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var decoded struct {
		Pong bool `json:"pong"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !decoded.Pong {
		t.Fatalf("expected pong:true in the response")
	}
}

// TestElemRoute_UnknownElement_Returns404 checks the "element not found"
// path over the same real HTTP round-trip.
func TestElemRoute_UnknownElement_Returns404(t *testing.T) {
	sysPath := t.TempDir()
	app, err := apptest.New(kernel.Dict{
		"Components": kernel.Dict{
			"JSPreprocessor": kernel.Dict{
				"SysPath":  sysPath,
				"MapsPath": sysPath,
			},
		},
	})
	if err != nil {
		t.Fatalf("apptest.New: %v", err)
	}
	if err := component.SetAppComponent(app, "Components.JSPreprocessor"); err != nil {
		t.Fatalf("SetAppComponent: %v", err)
	}

	srv := apptest.Server(app)
	defer srv.Close()

	body, _ := json.Marshal(map[string]any{
		"elem":   "neverRegistered",
		"path":   "ping",
		"params": map[string]any{},
	})
	resp, err := http.Post(srv.URL+"/lx/elem", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /lx/elem: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

var _ jspp.IElement = (*testElement)(nil)
