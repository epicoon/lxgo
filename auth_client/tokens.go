package client

import (
	"time"

	"github.com/epicoon/lxgo/kernel/http"
)

type tokensForm struct {
	*http.Form
	Success             bool   `json:"success"`
	ErrorCode           int    `json:"error_code,omitempty"`
	ErrorMessage        string `json:"error_message,omitempty"`
	AccessToken         string `dict:"access_token"`
	RefreshToken        string `dict:"refresh_token"`
	AccessTokenExpired  int64  `dict:"access_token_expired"`
	RefreshTokenExpired int64  `dict:"refresh_token_expired"`
}

type token struct {
	value     string
	expiresAt time.Time
}

type Tokens struct {
	Access  *token
	Refresh *token
}

func (ts *Tokens) Set(data *tokensForm) {
	ts.Access = new(token)
	ts.Access.value = data.AccessToken
	ts.Access.expiresAt = time.Unix(data.AccessTokenExpired, 0)
	ts.Refresh = new(token)
	ts.Refresh.value = data.RefreshToken
	ts.Refresh.expiresAt = time.Unix(data.RefreshTokenExpired, 0)
}

func (t *token) Value() string {
	return t.value
}

func (t *token) ExpiresAt() time.Time {
	return t.expiresAt
}
