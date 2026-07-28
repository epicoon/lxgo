package session_test

import (
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"testing"

	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/apptest"
	lxHttp "github.com/epicoon/lxgo/kernel/http"
	"github.com/epicoon/lxgo/session"
)

func newTestStorage(t *testing.T) (kernel.IApp, session.IStorage) {
	t.Helper()
	app, err := apptest.New(kernel.Dict{
		"Components": kernel.Dict{
			"SessionsStorage": kernel.Dict{
				"CookieName":  "lxgosessid",
				"MaxLifeTime": 3600,
			},
		},
	})
	if err != nil {
		t.Fatalf("apptest.New: %v", err)
	}
	if err := session.SetAppComponent(app, "Components.SessionsStorage"); err != nil {
		t.Fatalf("SetAppComponent: %v", err)
	}
	storage, err := session.AppComponent(app)
	if err != nil {
		t.Fatalf("AppComponent: %v", err)
	}
	return app, storage
}

// TestStorage_Scanner_BeforeFirstSession_DoesNotPanic is a regression
// test: Scanner() used to read the lazily-initialized provider field
// directly instead of through getProvider(), so calling it before any
// session had ever been started left it nil and any Scanner method call
// (Len/IsEmpty/PrintContent) would panic on a nil provider.
func TestStorage_Scanner_BeforeFirstSession_DoesNotPanic(t *testing.T) {
	_, storage := newTestStorage(t)

	scanner := storage.Scanner()
	if !scanner.IsEmpty() {
		t.Fatal("expected IsEmpty to be true before any session was started")
	}
	if scanner.Len() != 0 {
		t.Fatalf("Len() = %d, want 0", scanner.Len())
	}
	_ = scanner.PrintContent()
}

// TestStorage_Scanner_ReflectsStoredSessions checks Scanner against a
// non-empty store: Len/IsEmpty/PrintContent must reflect a session actually
// started, and its data must show up in PrintContent's rendering.
func TestStorage_Scanner_ReflectsStoredSessions(t *testing.T) {
	_, storage := newTestStorage(t)

	ctx := lxHttp.NewHandleContext(nil, "/whoami", nil)
	ctx.Init(nil, "/whoami", "GET", newRecorder(), newRequestWithoutCookie(t))
	sess, err := storage.StartSession(ctx)
	if err != nil {
		t.Fatalf("StartSession: %v", err)
	}
	sess.SetForce("greeting", "hello")

	scanner := storage.Scanner()
	if scanner.IsEmpty() {
		t.Fatal("expected IsEmpty to be false after starting a session")
	}
	if scanner.Len() != 1 {
		t.Fatalf("Len() = %d, want 1", scanner.Len())
	}
	content := scanner.PrintContent()
	if !strings.Contains(content, sess.ID()) || !strings.Contains(content, "hello") {
		t.Fatalf("PrintContent() = %q, want it to mention the session ID and its data", content)
	}
}

// TestStorage_Scanner_PrintContextContent checks the current-request
// variant: it must render the session attached to ctx's own cookie.
func TestStorage_Scanner_PrintContextContent(t *testing.T) {
	_, storage := newTestStorage(t)

	rec := newRecorder()
	ctx := lxHttp.NewHandleContext(nil, "/whoami", nil)
	ctx.Init(nil, "/whoami", "GET", rec, newRequestWithoutCookie(t))
	sess, err := storage.StartSession(ctx)
	if err != nil {
		t.Fatalf("StartSession: %v", err)
	}
	sess.SetForce("greeting", "hello")

	var cookie *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == "lxgosessid" {
			cookie = c
		}
	}
	if cookie == nil {
		t.Fatal("expected StartSession to set a session cookie")
	}

	req2 := newRequestWithoutCookie(t)
	req2.AddCookie(cookie)
	ctx2 := lxHttp.NewHandleContext(nil, "/whoami", nil)
	ctx2.Init(nil, "/whoami", "GET", newRecorder(), req2)

	got := storage.Scanner().PrintContextContent(ctx2)
	if !strings.Contains(got, "hello") {
		t.Fatalf("PrintContextContent() = %q, want it to mention the session's data", got)
	}
}

type whoamiResource struct {
	*lxHttp.Resource
}

func (r *whoamiResource) Run() kernel.IHttpResponse {
	sess, err := session.ExtractSession(r.Context())
	if err != nil {
		return r.ErrorResponse(http.StatusInternalServerError, err.Error())
	}
	return r.JsonResponse(kernel.JsonResponseConfig{Data: kernel.Dict{"id": sess.ID()}})
}

// TestStorage_CookieRoundTrip_RealHTTP is an integration test: a real
// kernel.IApp with the session component wired up exactly as a real
// application would (session.SetAppComponent), serving real HTTP requests
// through apptest.Server - the first request gets a fresh session and a
// Set-Cookie, the second (via an http.Client with a cookie jar, carrying
// that cookie back) must be recognized as the same session.
func TestStorage_CookieRoundTrip_RealHTTP(t *testing.T) {
	app, _ := newTestStorage(t)
	app.Router().RegisterResource("/whoami", "GET", func() kernel.IHttpResource {
		return &whoamiResource{Resource: lxHttp.NewResource()}
	})

	srv := apptest.Server(app)
	defer srv.Close()

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar.New: %v", err)
	}
	client := &http.Client{Jar: jar}

	getID := func() string {
		t.Helper()
		resp, err := client.Get(srv.URL + "/whoami")
		if err != nil {
			t.Fatalf("GET /whoami: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		var decoded struct {
			ID string `json:"id"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		return decoded.ID
	}

	firstID := getID()
	if firstID == "" {
		t.Fatal("expected a non-empty session ID on the first request")
	}

	secondID := getID()
	if secondID != firstID {
		t.Fatalf("expected the second request (same cookie jar) to see the same session, got %q then %q", firstID, secondID)
	}
}

// TestStorage_DestroySession_ClearsCookie is an integration test: after
// DestroySession, the session's cookie must actually be cleared (Set-Cookie
// with MaxAge<0), and the session itself must be gone from the provider.
//
// This is a regression test: DestroySession originally read the clearing
// cookie off sess.Context(), fixed at session-creation time and never
// updated - so it only ever saw the very first request that created the
// session, which never carries the cookie yet. The current design doesn't
// need to inspect any request at all (the cookie's name is always known
// from Config()) - it just needs the current response's
// http.ResponseWriter to write the clearing Set-Cookie to, passed in
// directly by the caller (e.g. the resource handling a logout request).
func TestStorage_DestroySession_ClearsCookie(t *testing.T) {
	_, storage := newTestStorage(t)

	ctx := lxHttp.NewHandleContext(nil, "/whoami", nil)
	rec := newRecorder()
	ctx.Init(nil, "/whoami", "GET", rec, newRequestWithoutCookie(t))
	sess, err := storage.StartSession(ctx)
	if err != nil {
		t.Fatalf("StartSession: %v", err)
	}

	storage.DestroySession(rec, sess)

	again, err := storage.SessionByID(sess.ID())
	if err != nil {
		t.Fatalf("SessionByID: %v", err)
	}
	if again != nil {
		t.Fatal("expected the session to be gone from storage after DestroySession")
	}

	cleared := false
	for _, c := range rec.Result().Cookies() {
		if c.Name == "lxgosessid" && c.MaxAge < 0 {
			cleared = true
		}
	}
	if !cleared {
		t.Fatal("expected DestroySession to set an expiring cookie clearing the session")
	}
}

// TestStorage_SetSessionID_ReKeysWithoutLeavingStaleEntry is an integration
// test of the real re-keying sequence used in production (SetSessionID),
// which relies on the caller clearing the old key before AddSession stores
// the new one (see BaseProvider.AddSession's own doc comment).
func TestStorage_SetSessionID_ReKeysWithoutLeavingStaleEntry(t *testing.T) {
	_, storage := newTestStorage(t)

	req := newRequestWithoutCookie(t)
	rec := newRecorder()
	ctx := lxHttp.NewHandleContext(nil, "/whoami", nil)
	ctx.Init(nil, "/whoami", "GET", rec, req)

	sess, err := storage.StartSession(ctx)
	if err != nil {
		t.Fatalf("StartSession: %v", err)
	}
	oldID := sess.ID()

	storage.SetSessionID(sess, "new-id")

	if sess.ID() != "new-id" {
		t.Fatalf("ID() = %q, want 'new-id'", sess.ID())
	}
	stale, err := storage.SessionByID(oldID)
	if err != nil {
		t.Fatalf("SessionByID(old): %v", err)
	}
	if stale != nil {
		t.Fatal("expected the old session ID to be gone after SetSessionID")
	}
	fresh, err := storage.SessionByID("new-id")
	if err != nil {
		t.Fatalf("SessionByID(new): %v", err)
	}
	if fresh == nil {
		t.Fatal("expected the new session ID to be present after SetSessionID")
	}
}
