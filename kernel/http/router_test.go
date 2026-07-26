package http

import (
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/epicoon/lxgo/kernel"
)

type testResource struct {
	*Resource
	ran               bool
	processErrorsResp kernel.IHttpResponse
	cReqForm          kernel.CForm
}

func newTestResource() *testResource {
	return &testResource{Resource: NewResource()}
}

func (r *testResource) Run() kernel.IHttpResponse {
	r.ran = true
	return &Response{}
}

func (r *testResource) ProcessRequestErrors() kernel.IHttpResponse {
	return r.processErrorsResp
}

func (r *testResource) CRequestForm() kernel.CForm {
	return r.cReqForm
}

func newGetContext(query string) (kernel.IHandleContext, *httptest.ResponseRecorder) {
	req := httptest.NewRequest("GET", "/resource"+query, nil)
	rec := httptest.NewRecorder()
	ctx := &HandleContext{}
	ctx.Init(nil, "/resource", "GET", rec, req)
	return ctx, rec
}

func TestProcessResource_MiddlewareError(t *testing.T) {
	router := &Router{}
	router.AddMiddleware(func(ctx kernel.IHandleContext) error {
		return errors.New("boom")
	})

	res := newTestResource()
	ctx, rec := newGetContext("")
	res.SetContext(ctx)

	resp := processResource(router, res)

	if resp != nil {
		t.Fatalf("expected nil response, got %#v", resp)
	}
	if res.ran {
		t.Fatal("Run() must not be called when middleware errors")
	}
	if rec.Code != 500 {
		t.Fatalf("expected 500 written to the response, got %d", rec.Code)
	}
}

func TestProcessResource_NoRequestForm_RunsDirectly(t *testing.T) {
	router := &Router{}
	res := newTestResource()
	ctx, _ := newGetContext("")
	res.SetContext(ctx)

	resp := processResource(router, res)

	if !res.ran {
		t.Fatal("expected Run() to be called")
	}
	if resp == nil {
		t.Fatal("expected a non-nil response from Run()")
	}
}

func TestProcessResource_ValidRequestForm_Fills_AndRuns(t *testing.T) {
	router := &Router{}
	res := newTestResource()
	res.cReqForm = func() kernel.IForm { return newTestForm() }
	ctx, _ := newGetContext("?name=Alice&age=30")
	res.SetContext(ctx)

	resp := processResource(router, res)

	if !res.ran {
		t.Fatal("expected Run() to be called")
	}
	if resp == nil {
		t.Fatal("expected a non-nil response")
	}
	filled, ok := res.RequestForm().(*testForm)
	if !ok {
		t.Fatalf("expected RequestForm to be a *testForm, got %T", res.RequestForm())
	}
	if filled.Name != "Alice" || filled.Age != 30 {
		t.Fatalf("form not filled as expected: %+v", filled)
	}
}

func TestProcessResource_RequestFormErrors_ProcessRequestErrorsShortCircuits(t *testing.T) {
	router := &Router{}
	res := newTestResource()
	res.cReqForm = func() kernel.IForm {
		f := newTestForm()
		f.SetRequired([]string{"name"})
		return f
	}
	custom := &Response{}
	res.processErrorsResp = custom
	ctx, _ := newGetContext("") // no "name" query param -> missing required field
	res.SetContext(ctx)

	resp := processResource(router, res)

	if resp != custom {
		t.Fatalf("expected ProcessRequestErrors' response to be returned, got %#v", resp)
	}
	if res.ran {
		t.Fatal("Run() must not be called when ProcessRequestErrors returns a response")
	}
}

func TestProcessResource_RequestFormErrors_NilProcessRequestErrorsFallsThroughToRun(t *testing.T) {
	// This is intentional, not a gap: if ProcessRequestErrors isn't
	// overridden, Run is expected to check RequestForm().HasErrors() itself
	// when it cares - see the doc comments on both.
	router := &Router{}
	res := newTestResource()
	res.cReqForm = func() kernel.IForm {
		f := newTestForm()
		f.SetRequired([]string{"name"})
		return f
	}
	res.processErrorsResp = nil
	ctx, _ := newGetContext("")
	res.SetContext(ctx)

	resp := processResource(router, res)

	if !res.ran {
		t.Fatal("expected Run() to still be called when ProcessRequestErrors returns nil")
	}
	if resp == nil {
		t.Fatal("expected a non-nil response from Run()")
	}
	if !res.RequestForm().HasErrors() {
		t.Fatal("expected the request form to still carry its validation error")
	}
}

func TestProcessResource_BeforeRunCallbacks_RunInOrder(t *testing.T) {
	router := &Router{}
	res := newTestResource()
	ctx, _ := newGetContext("")
	res.SetContext(ctx)

	var order []string
	res.BeforeRun(func(kernel.IHttpResource) { order = append(order, "first") })
	res.BeforeRun(func(kernel.IHttpResource) { order = append(order, "second") })

	processResource(router, res)

	if !res.ran {
		t.Fatal("expected Run() to be called")
	}
	want := []string{"first", "second"}
	if len(order) != len(want) || order[0] != want[0] || order[1] != want[1] {
		t.Fatalf("got %v, want %v", order, want)
	}
}
