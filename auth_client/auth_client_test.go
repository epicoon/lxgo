package client_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	client "github.com/epicoon/lxgo/auth_client"
	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/apptest"
)

func newTestAuthClient(t *testing.T, cfg kernel.Dict) *client.AuthClient {
	t.Helper()
	app, err := apptest.New(kernel.Dict{
		"Components": kernel.Dict{"AuthClient": cfg},
	})
	if err != nil {
		t.Fatalf("apptest.New: %v", err)
	}
	if err := client.SetAppComponent(app, "Components.AuthClient"); err != nil {
		t.Fatalf("SetAppComponent: %v", err)
	}
	ac, err := client.AppComponent(app)
	if err != nil {
		t.Fatalf("AppComponent: %v", err)
	}
	return ac
}

func TestAuthConfig_ConfiguredValues(t *testing.T) {
	ac := newTestAuthClient(t, kernel.Dict{
		"ID":          7,
		"Secret":      "shh",
		"RedirectUri": "/auth/callback",
		"Server":      "http://auth.example",
		"StatePath":   "/auth/state",
		"LogoutPath":  "/auth/logout",
		"RefreshPath": "/auth/refresh",
	})
	conf := ac.Config()

	if conf.ID != 7 || conf.Secret != "shh" || conf.RedirectUri != "/auth/callback" ||
		conf.Server != "http://auth.example" || conf.StatePath != "/auth/state" ||
		conf.LogoutPath != "/auth/logout" || conf.RefreshPath != "/auth/refresh" {
		t.Fatalf("conf = %#v", conf)
	}
}

func TestAuthConfig_UnconfiguredFieldsAreZeroValued(t *testing.T) {
	ac := newTestAuthClient(t, kernel.Dict{"ID": 1, "Secret": "s", "Server": "http://x"})
	conf := ac.Config()

	if conf.UserDataPath != "" {
		t.Fatalf("UserDataPath = %q, want the zero value", conf.UserDataPath)
	}
}

// jsonStub is a minimal httptest.Server that responds to a fixed path with
// a fixed JSON body/status, recording the last request it received.
type jsonStub struct {
	*httptest.Server
	lastReq  *http.Request
	lastBody map[string]any
}

func newJSONStub(t *testing.T, path string, status int, body map[string]any) *jsonStub {
	t.Helper()
	s := &jsonStub{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.lastReq = r
		var b map[string]any
		_ = json.NewDecoder(r.Body).Decode(&b)
		s.lastBody = b
		if r.URL.Path != path {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(body)
	}))
	t.Cleanup(s.Close)
	return s
}

func TestExchangeCodeForTokens_Success(t *testing.T) {
	stub := newJSONStub(t, "/tokens", http.StatusOK, map[string]any{
		"success":              true,
		"access_token":         "acc",
		"refresh_token":        "ref",
		"access_token_expired": 111,
		"scope":                "profile",
	})
	ac := newTestAuthClient(t, kernel.Dict{"ID": 1, "Secret": "s", "Server": stub.URL})

	tokens, err := ac.ExchangeCodeForTokens("somecode")
	if err != nil {
		t.Fatalf("ExchangeCodeForTokens: %v", err)
	}
	if tokens.Access.Value() != "acc" || tokens.Refresh.Value() != "ref" {
		t.Fatalf("tokens = %#v", tokens)
	}
}

// TestExchangeCodeForTokens_ServerRejectsCode is a regression test: a
// {"success": false, ...} response used to be built into a Tokens value
// with empty fields and returned as if it had succeeded, instead of the
// error the server actually reported.
func TestExchangeCodeForTokens_ServerRejectsCode(t *testing.T) {
	stub := newJSONStub(t, "/tokens", http.StatusOK, map[string]any{
		"success":       false,
		"error_message": "code is invalid or expired",
	})
	ac := newTestAuthClient(t, kernel.Dict{"ID": 1, "Secret": "s", "Server": stub.URL})

	tokens, err := ac.ExchangeCodeForTokens("badcode")
	if err == nil {
		t.Fatalf("expected an error, got tokens = %#v", tokens)
	}
	if err.Error() != "code is invalid or expired" {
		t.Fatalf("err = %q", err.Error())
	}
}

func TestRefreshTokens_Success(t *testing.T) {
	stub := newJSONStub(t, "/refresh", http.StatusOK, map[string]any{
		"success":       true,
		"access_token":  "newacc",
		"refresh_token": "newref",
	})
	ac := newTestAuthClient(t, kernel.Dict{"ID": 1, "Secret": "s", "Server": stub.URL})

	tokens, err := ac.RefreshTokens("oldref")
	if err != nil {
		t.Fatalf("RefreshTokens: %v", err)
	}
	if tokens.Access.Value() != "newacc" {
		t.Fatalf("tokens = %#v", tokens)
	}
}

func TestRefreshTokens_ServerRejects(t *testing.T) {
	stub := newJSONStub(t, "/refresh", http.StatusOK, map[string]any{
		"success":       false,
		"error_message": "refresh token revoked",
	})
	ac := newTestAuthClient(t, kernel.Dict{"ID": 1, "Secret": "s", "Server": stub.URL})

	if _, err := ac.RefreshTokens("revoked"); err == nil {
		t.Fatal("expected an error for a rejected refresh token")
	}
}

// TestRefreshTokens_ServerRejects_CarriesHTTPStatus checks that a
// {success:false} rejection surfaces as a *client.StatusError carrying the
// server's actual HTTP status (400 for invalid_scope, in this case) - not
// a plain error a caller can't distinguish from a transport failure.
func TestRefreshTokens_ServerRejects_CarriesHTTPStatus(t *testing.T) {
	stub := newJSONStub(t, "/refresh", http.StatusBadRequest, map[string]any{
		"success":       false,
		"error_code":    1017,
		"error_message": "Requested scope exceeds the scope already granted",
	})
	ac := newTestAuthClient(t, kernel.Dict{"ID": 1, "Secret": "s", "Server": stub.URL})

	_, err := ac.RefreshTokens("oldref", "profile:data")
	if err == nil {
		t.Fatal("expected an error for a rejected scope")
	}
	var statusErr *client.StatusError
	if !errors.As(err, &statusErr) {
		t.Fatalf("expected a *client.StatusError, got %T: %v", err, err)
	}
	if statusErr.Status != http.StatusBadRequest {
		t.Fatalf("StatusError.Status = %d, want %d", statusErr.Status, http.StatusBadRequest)
	}
	if statusErr.Message != "Requested scope exceeds the scope already granted" {
		t.Fatalf("StatusError.Message = %q", statusErr.Message)
	}
}

func TestRefreshTokens_WithScope_SendsScope(t *testing.T) {
	stub := newJSONStub(t, "/refresh", http.StatusOK, map[string]any{
		"success":       true,
		"access_token":  "newacc",
		"refresh_token": "newref",
		"scope":         "profile",
	})
	ac := newTestAuthClient(t, kernel.Dict{"ID": 1, "Secret": "s", "Server": stub.URL})

	tokens, err := ac.RefreshTokens("oldref", "profile")
	if err != nil {
		t.Fatalf("RefreshTokens: %v", err)
	}
	if stub.lastBody["scope"] != "profile" {
		t.Fatalf("request body scope = %v, want %q", stub.lastBody["scope"], "profile")
	}
	if tokens.Scope != "profile" {
		t.Fatalf("tokens.Scope = %q, want %q", tokens.Scope, "profile")
	}
}

// TestRefreshTokens_WithoutScope_OmitsScopeFromRequest checks the backward
// compatible path: calling RefreshTokens the old way (no scope argument at
// all) must not add a "scope" key to the request body.
func TestRefreshTokens_WithoutScope_OmitsScopeFromRequest(t *testing.T) {
	stub := newJSONStub(t, "/refresh", http.StatusOK, map[string]any{
		"success":       true,
		"access_token":  "newacc",
		"refresh_token": "newref",
	})
	ac := newTestAuthClient(t, kernel.Dict{"ID": 1, "Secret": "s", "Server": stub.URL})

	if _, err := ac.RefreshTokens("oldref"); err != nil {
		t.Fatalf("RefreshTokens: %v", err)
	}
	if _, ok := stub.lastBody["scope"]; ok {
		t.Fatalf("request body unexpectedly carries scope: %#v", stub.lastBody)
	}
}

// TestRefreshTokens_EmptyScopeArg_OmitsScopeFromRequest checks that an
// explicit empty string behaves the same as not passing scope at all.
func TestRefreshTokens_EmptyScopeArg_OmitsScopeFromRequest(t *testing.T) {
	stub := newJSONStub(t, "/refresh", http.StatusOK, map[string]any{
		"success":       true,
		"access_token":  "newacc",
		"refresh_token": "newref",
	})
	ac := newTestAuthClient(t, kernel.Dict{"ID": 1, "Secret": "s", "Server": stub.URL})

	if _, err := ac.RefreshTokens("oldref", ""); err != nil {
		t.Fatalf("RefreshTokens: %v", err)
	}
	if _, ok := stub.lastBody["scope"]; ok {
		t.Fatalf("request body unexpectedly carries scope: %#v", stub.lastBody)
	}
}

func TestLogOut_Success(t *testing.T) {
	stub := newJSONStub(t, "/logout", http.StatusOK, map[string]any{"success": true})
	ac := newTestAuthClient(t, kernel.Dict{"ID": 1, "Secret": "s", "Server": stub.URL})

	if err := ac.LogOut("sometoken"); err != nil {
		t.Fatalf("LogOut: %v", err)
	}
	if got := stub.lastReq.Header.Get("Authorization"); got != "Bearer sometoken" {
		t.Fatalf("Authorization header = %q", got)
	}
}

func TestLogOut_ServerRejects(t *testing.T) {
	stub := newJSONStub(t, "/logout", http.StatusOK, map[string]any{
		"success":       false,
		"error_message": "invalid token",
	})
	ac := newTestAuthClient(t, kernel.Dict{"ID": 1, "Secret": "s", "Server": stub.URL})

	if err := ac.LogOut("badtoken"); err == nil {
		t.Fatal("expected an error for a rejected access token")
	}
}
