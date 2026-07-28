package client_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"testing"

	client "github.com/epicoon/lxgo/auth_client"
	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/apptest"
	"github.com/epicoon/lxgo/session"
)

// newTestApp builds a real kernel.IApp with both the session component (the
// handlers under test extract the session from the request) and the
// AuthClient component (pointed at stubServerURL, a stand-in for the real
// lxgo-auth service - see the testing skill's guidance on stubbing rather
// than standing up the real, heavier service for this package's
// integration tests), then registers every handler at its usual route.
func newTestApp(t *testing.T, stubServerURL string) kernel.IApp {
	t.Helper()
	app, err := apptest.New(kernel.Dict{
		"Components": kernel.Dict{
			"SessionsStorage": kernel.Dict{
				"CookieName":  "lxgosessid",
				"MaxLifeTime": 3600,
			},
			"AuthClient": kernel.Dict{
				"ID":          1,
				"Secret":      "s",
				"Server":      stubServerURL,
				"RedirectUri": "/auth/callback",
			},
		},
	})
	if err != nil {
		t.Fatalf("apptest.New: %v", err)
	}
	if err := session.SetAppComponent(app, "Components.SessionsStorage"); err != nil {
		t.Fatalf("session.SetAppComponent: %v", err)
	}
	if err := client.SetAppComponent(app, "Components.AuthClient"); err != nil {
		t.Fatalf("client.SetAppComponent: %v", err)
	}

	app.Router().RegisterResource("/auth/state", "GET", client.NewStateHandler)
	app.Router().RegisterResource("/auth/callback", "GET", client.NewAuthCallbackHandler)
	app.Router().RegisterResource("/auth/refresh", "POST", client.NewRefreshHandler)
	app.Router().RegisterResource("/auth/logout", "POST", client.NewLogoutHandler)

	return app
}

func newClientWithCookies(t *testing.T) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar.New: %v", err)
	}
	return &http.Client{Jar: jar}
}

func TestStateHandler_RealHTTPRoundTrip(t *testing.T) {
	app := newTestApp(t, "http://unused.invalid")
	srv := apptest.Server(app)
	defer srv.Close()

	c := newClientWithCookies(t)
	resp, err := c.Get(srv.URL + "/auth/state")
	if err != nil {
		t.Fatalf("GET /auth/state: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}

	var decoded struct {
		State string `json:"state"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if decoded.State == "" {
		t.Fatal("expected a non-empty state")
	}
}

// TestAuthCallbackHandler_FullCycle drives the whole StateHandler -> (client
// echoes the state back, as the real authorization service's redirect
// would) -> AuthCallbackHandler flow, with a stub server standing in for
// lxgo-auth's /tokens endpoint.
func TestAuthCallbackHandler_FullCycle(t *testing.T) {
	stub := newJSONStub(t, "/tokens", http.StatusOK, map[string]any{
		"success":              true,
		"access_token":         "acc-value",
		"refresh_token":        "ref-value",
		"access_token_expired": 9999999999,
	})
	app := newTestApp(t, stub.URL)
	srv := apptest.Server(app)
	defer srv.Close()

	c := newClientWithCookies(t)

	resp, err := c.Get(srv.URL + "/auth/state")
	if err != nil {
		t.Fatalf("GET /auth/state: %v", err)
	}
	var stateResp struct {
		State string `json:"state"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&stateResp); err != nil {
		t.Fatalf("decode state response: %v", err)
	}
	resp.Body.Close()

	resp, err = c.Get(srv.URL + "/auth/callback?code=somecode&state=" + stateResp.State)
	if err != nil {
		t.Fatalf("GET /auth/callback: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}

	body := make([]byte, 4096)
	n, _ := resp.Body.Read(body)
	html := string(body[:n])
	if !strings.Contains(html, "acc-value") {
		t.Fatalf("expected the access token in the response HTML, got: %s", html)
	}
}

func TestAuthCallbackHandler_InvalidState_Returns400(t *testing.T) {
	app := newTestApp(t, "http://unused.invalid")
	srv := apptest.Server(app)
	defer srv.Close()

	c := newClientWithCookies(t)

	resp, err := c.Get(srv.URL + "/auth/state")
	if err != nil {
		t.Fatalf("GET /auth/state: %v", err)
	}
	resp.Body.Close()

	resp, err = c.Get(srv.URL + "/auth/callback?code=somecode&state=wrong-state")
	if err != nil {
		t.Fatalf("GET /auth/callback: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for a mismatched CSRF state", resp.StatusCode)
	}
}

// TestAuthCallbackHandler_ExchangeRejected_Returns500 is a real-HTTP
// regression test for the ExchangeCodeForTokens Success-check fix: a
// rejected code must surface as a clean 500 through the handler, not a 200
// with an HTML page carrying empty tokens.
func TestAuthCallbackHandler_ExchangeRejected_Returns500(t *testing.T) {
	stub := newJSONStub(t, "/tokens", http.StatusOK, map[string]any{
		"success":       false,
		"error_message": "code is invalid or expired",
	})
	app := newTestApp(t, stub.URL)
	srv := apptest.Server(app)
	defer srv.Close()

	c := newClientWithCookies(t)

	resp, err := c.Get(srv.URL + "/auth/state")
	if err != nil {
		t.Fatalf("GET /auth/state: %v", err)
	}
	var stateResp struct {
		State string `json:"state"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&stateResp); err != nil {
		t.Fatalf("decode state response: %v", err)
	}
	resp.Body.Close()

	resp, err = c.Get(srv.URL + "/auth/callback?code=badcode&state=" + stateResp.State)
	if err != nil {
		t.Fatalf("GET /auth/callback: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 for a rejected code exchange", resp.StatusCode)
	}
}

func TestRefreshHandler_RealHTTPRoundTrip(t *testing.T) {
	stub := newJSONStub(t, "/refresh", http.StatusOK, map[string]any{
		"success":       true,
		"access_token":  "newacc",
		"refresh_token": "newref",
	})
	app := newTestApp(t, stub.URL)
	srv := apptest.Server(app)
	defer srv.Close()

	body, _ := json.Marshal(map[string]any{"refresh_token": "oldref"})
	resp, err := http.Post(srv.URL+"/auth/refresh", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /auth/refresh: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}

	var decoded struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if decoded.AccessToken != "newacc" {
		t.Fatalf("decoded = %#v", decoded)
	}
}

// TestLogoutHandler_RealHTTPRoundTrip is a regression test: NewLogoutHandler
// used to build its *lxHttp.Resource via a bare struct literal instead of
// lxHttp.NewResource(), leaving its context nil - the router panicked on
// every real request. This drives a real HTTP request through the actual
// router (apptest.Server), not just a direct Go call, so it would have hit
// exactly that panic.
func TestLogoutHandler_RealHTTPRoundTrip(t *testing.T) {
	stub := newJSONStub(t, "/logout", http.StatusOK, map[string]any{"success": true})
	app := newTestApp(t, stub.URL)
	srv := apptest.Server(app)
	defer srv.Close()

	req, err := http.NewRequest("POST", srv.URL+"/auth/logout", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Authorization", "Bearer sometoken")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /auth/logout: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
}
