package handlers

import (
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/apptest"
	lxHttp "github.com/epicoon/lxgo/kernel/http"
)

func newTestApp(t *testing.T) kernel.IApp {
	t.Helper()
	app, err := apptest.New()
	if err != nil {
		t.Fatalf("apptest.New: %v", err)
	}
	return app
}

func newTestContext(t *testing.T, app kernel.IApp, res kernel.IHttpResource, route, query string) kernel.IHandleContext {
	t.Helper()
	req := httptest.NewRequest("GET", route+query, nil)
	rec := httptest.NewRecorder()
	ctx := lxHttp.NewHandleContext(app, route, res)
	ctx.Init(app, route, "GET", rec, req)
	return ctx
}

func newJSONTestContext(t *testing.T, app kernel.IApp, res kernel.IHttpResource, route, jsonBody string) kernel.IHandleContext {
	t.Helper()
	req := httptest.NewRequest("POST", route, strings.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	ctx := lxHttp.NewHandleContext(app, route, res)
	ctx.Init(app, route, "POST", rec, req)
	return ctx
}

// TestElemHandler_MissingJSPPInContext_DoesNotPanic is a regression test for
// task 0100: when the "jspp" context key isn't the expected type, the error
// branch used to call pp.LogError on a nil pp and panic instead of
// reporting a clean 500.
func TestElemHandler_MissingJSPPInContext_DoesNotPanic(t *testing.T) {
	app := newTestApp(t)
	res := NewElemHandler()
	ctx := newTestContext(t, app, res, "/lx/elem", "")
	res.SetContext(ctx)

	req := &ElemRequest{Form: lxHttp.NewForm(), Elem: "someElem", Path: "somePath"}
	res.SetRequestForm(req)

	resp := res.Run()

	if resp.Code() != 500 {
		t.Fatalf("expected a 500 response, got %d", resp.Code())
	}
}

// TestServiceHandler_MissingJSPPInContext_DoesNotPanic is the same
// regression as above, for ServiceHandler.
func TestServiceHandler_MissingJSPPInContext_DoesNotPanic(t *testing.T) {
	app := newTestApp(t)
	res := NewServiceHandler()
	body := `{"action":"get-modules","params":{"have":["a"],"need":["b"]}}`
	ctx := newJSONTestContext(t, app, res, "/lx/service", body)
	res.SetContext(ctx)

	resp := res.Run()

	if resp.Code() != 500 {
		t.Fatalf("expected a 500 response, got %d", resp.Code())
	}
}

// TestNamesFromAny is a regression test for task 0102: except used to be
// built via make([]string, len(have)) followed by append, doubling its
// length with a run of empty strings in front.
func TestNamesFromAny(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		got, err := namesFromAny(nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("expected an empty slice, got %v", got)
		}
	})

	t.Run("non_empty", func(t *testing.T) {
		got, err := namesFromAny([]any{"a", "b", "c"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := []string{"a", "b", "c"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("expected %v, got %v", want, got)
		}
	})

	t.Run("wrong_type", func(t *testing.T) {
		_, err := namesFromAny([]any{"a", 42})
		if err == nil {
			t.Fatal("expected an error for a non-string element")
		}
	})
}

// TestPluginHandler_MissingJSPPInContext_DoesNotPanic is the same
// regression as above, for PluginHandler.
func TestPluginHandler_MissingJSPPInContext_DoesNotPanic(t *testing.T) {
	app := newTestApp(t)
	res := NewPluginHandler()
	ctx := newTestContext(t, app, res, "/lx/plugin", "?plugin=x&path=y")
	res.SetContext(ctx)

	resp := res.Run()

	if resp.Code() != 500 {
		t.Fatalf("expected a 500 response, got %d", resp.Code())
	}
}
