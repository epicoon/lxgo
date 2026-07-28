package client

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/epicoon/lxgo/kernel"
	lxHttp "github.com/epicoon/lxgo/kernel/http"
)

func newTestContextWithHeader(t *testing.T, key, value string) kernel.IHandleContext {
	t.Helper()
	req := httptest.NewRequest("GET", "/", nil)
	if key != "" {
		req.Header.Set(key, value)
	}
	ctx := lxHttp.NewHandleContext(nil, "/", nil)
	ctx.Init(nil, "/", "GET", httptest.NewRecorder(), req)
	return ctx
}

func TestGetBearer_Valid(t *testing.T) {
	ctx := newTestContextWithHeader(t, "Authorization", "Bearer abc123")
	token, err := GetBearer(ctx)
	if err != nil {
		t.Fatalf("GetBearer: %v", err)
	}
	if token != "abc123" {
		t.Fatalf("token = %q, want 'abc123'", token)
	}
}

func TestGetBearer_MissingHeader(t *testing.T) {
	ctx := newTestContextWithHeader(t, "", "")
	_, err := GetBearer(ctx)
	if err != ErrAuthMissing {
		t.Fatalf("err = %v, want ErrAuthMissing", err)
	}
}

func TestGetBearer_WrongScheme(t *testing.T) {
	ctx := newTestContextWithHeader(t, "Authorization", "Basic abc123")
	_, err := GetBearer(ctx)
	if err != ErrAuthWrongScheme {
		t.Fatalf("err = %v, want ErrAuthWrongScheme", err)
	}
}

func TestTokens_Set(t *testing.T) {
	data := &tokensForm{
		AccessToken:         "access-val",
		RefreshToken:        "refresh-val",
		AccessTokenExpired:  1000,
		RefreshTokenExpired: 2000,
		Scope:               "profile",
	}

	tokens := new(Tokens)
	tokens.Set(data)

	if tokens.Access.Value() != "access-val" {
		t.Fatalf("Access.Value() = %q", tokens.Access.Value())
	}
	if !tokens.Access.ExpiresAt().Equal(time.Unix(1000, 0)) {
		t.Fatalf("Access.ExpiresAt() = %v, want %v", tokens.Access.ExpiresAt(), time.Unix(1000, 0))
	}
	if tokens.Refresh.Value() != "refresh-val" {
		t.Fatalf("Refresh.Value() = %q", tokens.Refresh.Value())
	}
	if !tokens.Refresh.ExpiresAt().Equal(time.Unix(2000, 0)) {
		t.Fatalf("Refresh.ExpiresAt() = %v, want %v", tokens.Refresh.ExpiresAt(), time.Unix(2000, 0))
	}
	if tokens.Scope != "profile" {
		t.Fatalf("Scope = %q", tokens.Scope)
	}
}
