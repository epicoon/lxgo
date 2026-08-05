# Authentication client for lxgo/kernel applications

> Actual version: `v0.1.0-alpha.5`. [Details](https://github.com/epicoon/lxgo/tree/master/auth_client/CHANGE_LOG.md)

This package is the client-side counterpart of the
[lxgo/auth](https://github.com/epicoon/lxgo/tree/master/auth) authentication microservice — it wires an
[lxgo/kernel](https://github.com/epicoon/lxgo/tree/master/kernel)-based application into the OAuth2-like flow that
microservice implements (see its README, ["Full ready-to-use solution for lxgo/kernel
applications"](https://github.com/epicoon/lxgo/tree/master/auth/README.md#full-sol)), so you don't have to write the
integration by hand.

Concretely, it gives you:
* an app component (`AuthClient`) that talks to the authorization service's API (exchanging a code for tokens,
  refreshing them, logging out, fetching user data) — see `auth_client.go`. The `Tokens` it returns
  (`ExchangeCodeForTokens`/`RefreshTokens`, `tokens.go`) carry a `Scope` field with the access level the server
  actually granted (`profile` or `profile:data`) — read it if your app needs to know which one it got.
  `RefreshTokens(refreshToken, scope...)` takes an optional `scope` to narrow the reissued tokens' access
  (RFC 6749 §6) — omit it to keep the current scope unchanged; requesting anything broader than what was already
  granted is rejected by the server (`400`/`invalid_scope`) — surfaced from `RefreshTokens` as a `*client.StatusError`
  (`Status`/`Message`), so callers can tell a rejection like this apart from a transport-level failure;
* ready-made HTTP handlers for four of the five endpoints your application needs to expose to the browser (the
  auth-callback redirect target, CSRF state generation, and the `/logout`/`/refresh` proxies) — see
  `auth_callback_handler.go`/`state_handler.go`/`logout_handler.go`/`refresh_handler.go`;
* a small helper (`GetBearer`) for reading the `Authorization: Bearer <token>` header off an incoming request.

Use it if your application is based on `lxgo/kernel` and you want to authenticate its users against an `lxgo/auth`
instance; if it isn't, see `lxgo/auth`'s README sections on reimplementing the client/server-side integration
yourself instead.

1. Add the app component to your app config file:
```yaml
Components:
  # ...
  Auth:
    ID: 1
    Secret: rand_string
    RedirectUri: http://client_app_url/auth-callback
    Server: http://auth_server_url
    StatePath: /gen-state
    LogoutPath: /logout
    RefreshPath: /auth-refresh
    UserDataPath: /get-user
```

2. Plug the application component:
```go
import (
	github.com/epicoon/lxgo/auth_client
)

// app implements kernel.IApp
if err := auth_client.SetAppComponent(app, "Components.Auth"); err != nil {
    // process err
}
```

3. Register the ready-made handlers under the same paths you configured above:
```go
import (
	github.com/epicoon/lxgo/auth_client
    // ...
)

app.Router().RegisterResources(kernel.HttpResourcesList{
    "/auth-callback": auth_client.NewAuthCallbackHandler,
    "/gen-state":     auth_client.NewStateHandler,
    "/logout":        auth_client.NewLogoutHandler,
    "/auth-refresh":  auth_client.NewRefreshHandler,
    "/get-user":      NewGetUserHandler,
    // ...
})
```

`/auth-refresh`'s `RefreshRequest` accepts an optional `scope` field in the POST body alongside `refresh_token` — pass
it to narrow the reissued tokens' access (RFC 6749 §6); the request is rejected (`400`, with the authorization
service's own message) if it asks for anything broader than what the tokens already had:
```json
{"refresh_token": "<token>", "scope": "profile"}
```
Calling `AuthClient.RefreshTokens` directly works the same way — the trailing `scope` argument is optional, so
existing calls that don't pass one keep working unchanged:
```go
// keeps the current scope
tokens, err := authClient.RefreshTokens(refreshToken)

// narrows to "profile"
tokens, err := authClient.RefreshTokens(refreshToken, "profile")
```

4. The fifth endpoint (`/get-user`, proxying the authorization service's `/user-data`) doesn't have a ready-made
   handler — write a thin one yourself on top of `AuthClient.GetUserData`:
```go
import (
	"fmt"
	"net/http"

	client "github.com/epicoon/lxgo/auth_client"
	"github.com/epicoon/lxgo/kernel"
	lxHttp "github.com/epicoon/lxgo/kernel/http"
)

type GetUserHandler struct {
	*lxHttp.Resource
}

func NewGetUserHandler() kernel.IHttpResource {
	return &GetUserHandler{Resource: lxHttp.NewResource()}
}

func (handler *GetUserHandler) Run() kernel.IHttpResponse {
	accessToken, err := client.GetBearer(handler.Context())
	if err != nil {
		return handler.JsonResponse(kernel.JsonResponseConfig{
			Code: http.StatusUnauthorized,
			Data: err,
		})
	}

	authClient, err := client.AppComponent(handler.App())
	if err != nil {
		handler.LogError("wrong application configuration: auth_client component required", "App")
		return handler.ErrorResponse(http.StatusInternalServerError, "Something went wrong")
	}

	data, err := authClient.GetUserData(accessToken)
	if err != nil {
		handler.LogError(fmt.Sprintf("can not get user data: %s", err), "App")
		return handler.ErrorResponse(http.StatusInternalServerError, "Something went wrong")
	}

	return handler.JsonResponse(kernel.JsonResponseConfig{
		Data: map[string]any{
			"login": data.Login,
			"data":  data.Data,
		},
	})
}
```


## License

Apache License 2.0 — see [LICENSE](./LICENSE).
