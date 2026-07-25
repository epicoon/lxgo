package client

import (
	"time"

	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/http"
)

/** @interface kernel.IForm */

type tokensForm struct {
	*http.Form
	Success             bool   `json:"success"`
	ErrorCode           int    `json:"error_code,omitempty"`
	ErrorMessage        string `json:"error_message,omitempty"`
	AccessToken         string `dict:"access_token"`
	RefreshToken        string `dict:"refresh_token"`
	AccessTokenExpired  int64  `dict:"access_token_expired"`
	RefreshTokenExpired int64  `dict:"refresh_token_expired"`
	Scope               string `dict:"scope"`
}

var _ kernel.IForm = (*tokensForm)(nil)

// token is one access or refresh token's value and expiry - unexported, but
// reachable (and its methods usable) via Tokens.Access/Tokens.Refresh.
type token struct {
	value     string
	expiresAt time.Time
}

// Tokens is an access/refresh token pair, as returned by
// AuthClient.ExchangeCodeForTokens/RefreshTokens.
type Tokens struct {
	// Access is the access token - use Access.Value()/Access.ExpiresAt().
	Access *token
	// Refresh is the refresh token - use Refresh.Value()/Refresh.ExpiresAt().
	Refresh *token
	// Scope is the access level the server actually granted ("profile" or
	// "profile:data") - may be narrower than what was requested.
	Scope string
}

// Set populates ts from data - used internally right after a tokens/refresh
// API call succeeds.
func (ts *Tokens) Set(data *tokensForm) {
	ts.Access = new(token)
	ts.Access.value = data.AccessToken
	ts.Access.expiresAt = time.Unix(data.AccessTokenExpired, 0)
	ts.Refresh = new(token)
	ts.Refresh.value = data.RefreshToken
	ts.Refresh.expiresAt = time.Unix(data.RefreshTokenExpired, 0)
	ts.Scope = data.Scope
}

// Value returns the token's string value.
func (t *token) Value() string {
	return t.value
}

// ExpiresAt returns when the token expires.
func (t *token) ExpiresAt() time.Time {
	return t.expiresAt
}
