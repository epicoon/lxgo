package http

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/epicoon/lxgo/kernel"
)

type testResource struct {
	*Resource
	ran               bool
	processErrorsResp kernel.IHttpResponse
	cReqForm          kernel.CForm
	label             string
}

// labeledResource returns a kernel.CHttpResource constructing a
// *testResource tagged with label, so a test can tell which of several
// registered routes/templates actually got matched.
func labeledResource(label string) kernel.CHttpResource {
	return func() kernel.IHttpResource {
		r := newTestResource()
		r.label = label
		return r
	}
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

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * Template routes
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

func TestRouter_Template_MatchesAndExtractsParam(t *testing.T) {
	router := NewRouter(nil).(*Router)
	router.RegisterResource("/game/deps/{nodeKey}", "", labeledResource("deps"))

	cResource, params, code := router.defineResource("/game/deps/abc123", "GET")

	if code != 0 {
		t.Fatalf("expected a match, got code %d", code)
	}
	res := cResource().(*testResource)
	if res.label != "deps" {
		t.Fatalf("expected the template route's handler, got label %q", res.label)
	}
	if params["nodeKey"] != "abc123" {
		t.Fatalf("expected nodeKey=abc123, got %#v", params)
	}
}

func TestRouter_ExactRouteAlwaysWinsOverTemplate(t *testing.T) {
	router := NewRouter(nil).(*Router)
	router.RegisterResource("/game/deps/{nodeKey}", "", labeledResource("template"))
	router.RegisterResource("/game/deps/abc123", "", labeledResource("exact"))

	cResource, params, code := router.defineResource("/game/deps/abc123", "GET")

	if code != 0 {
		t.Fatalf("expected a match, got code %d", code)
	}
	res := cResource().(*testResource)
	if res.label != "exact" {
		t.Fatalf("expected the exact route to win, got label %q", res.label)
	}
	if params != nil {
		t.Fatalf("expected no path segments for an exact-route match, got %#v", params)
	}
}

func TestRouter_Template_SegmentCountMismatch_NotFound(t *testing.T) {
	router := NewRouter(nil).(*Router)
	router.RegisterResource("/game/deps/{nodeKey}", "", labeledResource("deps"))

	if _, _, code := router.defineResource("/game/deps", "GET"); code != http.StatusNotFound {
		t.Fatalf("expected 404 for too few segments, got code %d", code)
	}
	if _, _, code := router.defineResource("/game/deps/abc/extra", "GET"); code != http.StatusNotFound {
		t.Fatalf("expected 404 for too many segments, got code %d", code)
	}
}

func TestRouter_Template_MultipleNamedParams(t *testing.T) {
	router := NewRouter(nil).(*Router)
	router.RegisterResource("/cartridges/{cartridge}/games/{game}", "", labeledResource("multi"))

	_, params, code := router.defineResource("/cartridges/lxGames/games/ootv", "GET")

	if code != 0 {
		t.Fatalf("expected a match, got code %d", code)
	}
	if params["cartridge"] != "lxGames" || params["game"] != "ootv" {
		t.Fatalf("expected both params extracted, got %#v", params)
	}
}

func TestRouter_WildcardTemplate_CapturesRestIncludingEmpty(t *testing.T) {
	router := NewRouter(nil).(*Router)
	router.RegisterResource("/assets/*{rest}", "", labeledResource("assets"))

	_, params, code := router.defineResource("/assets/css/main.css", "GET")
	if code != 0 {
		t.Fatalf("expected a match, got code %d", code)
	}
	if params["rest"] != "css/main.css" {
		t.Fatalf("expected rest=\"css/main.css\", got %#v", params)
	}

	_, params, code = router.defineResource("/assets", "GET")
	if code != 0 {
		t.Fatalf("expected a match for the empty tail, got code %d", code)
	}
	if params["rest"] != "" {
		t.Fatalf("expected rest=\"\" for the empty tail, got %#v", params)
	}
}

func TestRouter_TemplateVsTemplate_FewerWildcardsWins(t *testing.T) {
	router := NewRouter(nil).(*Router)
	router.RegisterResource("/game/deps/*{rest}", "", labeledResource("wildcard"))
	router.RegisterResource("/game/deps/{nodeKey}", "", labeledResource("param"))

	cResource, params, code := router.defineResource("/game/deps/abc123", "GET")

	if code != 0 {
		t.Fatalf("expected a match, got code %d", code)
	}
	res := cResource().(*testResource)
	if res.label != "param" {
		t.Fatalf("expected the single-segment param template to win over the wildcard, got label %q", res.label)
	}
	if params["nodeKey"] != "abc123" {
		t.Fatalf("expected nodeKey=abc123, got %#v", params)
	}
}

func TestRouter_Template_MethodNotRegistered_MethodNotAllowed(t *testing.T) {
	router := NewRouter(nil).(*Router)
	router.RegisterResource("/game/deps/{nodeKey}", "POST", labeledResource("deps"))

	if _, _, code := router.defineResource("/game/deps/abc123", "GET"); code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got code %d", code)
	}
}

func TestRouter_RegisterResource_SameTemplateTwice_MergesMethods(t *testing.T) {
	router := NewRouter(nil).(*Router)
	router.RegisterResource("/game/deps/{nodeKey}", "GET", labeledResource("get"))
	router.RegisterResource("/game/deps/{nodeKey}", "POST", labeledResource("post"))

	if len(router.templates) != 1 {
		t.Fatalf("expected the second registration to merge into the same template, got %d templates", len(router.templates))
	}

	cResource, _, code := router.defineResource("/game/deps/abc123", "POST")
	if code != 0 {
		t.Fatalf("expected a match, got code %d", code)
	}
	if res := cResource().(*testResource); res.label != "post" {
		t.Fatalf("expected the POST handler, got label %q", res.label)
	}
}

func TestRouter_WildcardNotLastSegment_Panics(t *testing.T) {
	router := NewRouter(nil).(*Router)
	defer func() {
		if recover() == nil {
			t.Fatal("expected a panic for a wildcard segment that isn't last")
		}
	}()
	router.RegisterResource("/game/*{rest}/deps", "", labeledResource("bad"))
}

func TestRouter_UnnamedParamSegment_Panics(t *testing.T) {
	router := NewRouter(nil).(*Router)
	defer func() {
		if recover() == nil {
			t.Fatal("expected a panic for an unnamed \"{}\" segment")
		}
	}()
	router.RegisterResource("/game/{}", "", labeledResource("bad"))
}

func TestRouter_UnnamedWildcardSegment_Panics(t *testing.T) {
	router := NewRouter(nil).(*Router)
	defer func() {
		if recover() == nil {
			t.Fatal("expected a panic for an unnamed \"*{}\" segment")
		}
	}()
	router.RegisterResource("/game/*{}", "", labeledResource("bad"))
}
