
//TODO write descriptions

App config:
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

Init routes:
```go
import (
	client "github.com/epicoon/lxgo/auth_client"
    // ...
)

app.Router().RegisterResources(kernel.HttpResourcesList{
    "/auth-callback": client.NewAuthCallbackHandler,
    "/gen-state":     client.NewStateHandler,
    "/logout":        client.NewLogoutHandler,
    "/auth-refresh":  client.NewRefreshHandler,
    "/get-user":      NewGetUserHandler,
    // ...
})
```

"Get-user" handler:
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
