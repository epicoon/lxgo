# Package HTTP sessions in lxgo/kernel web-applications

> Actual version: `v0.1.0-alpha.4`. [Details](https://github.com/epicoon/lxgo/tree/master/session/CHANGE_LOG.md)

> You can use it if your application is based on [lxgo/kernel](https://github.com/epicoon/lxgo/tree/master/kernel)

1. Add app component in your app config file:
```yaml
Components:
  # ...
  SessionStorage:
    CookieName: lxgosessid
    MaxLifeTime: 36000 # seconds
```

2. Plug application component:
```go
import (
	"github.com/epicoon/lxgo/session"
)

// app implements kernel.IApp
if err := session.SetAppComponent(app, "Components.SessionStorage"); err != nil {
    // process err
}
```

> Sessions will be associated with request contexts (`kernel.IHandleContext`)

3. Example how to use sessions:
```go
// ... somewhere in [[kernel.IHttpResource.Run()]]
// r is kernel.IHttpResource

// Get session
sess, err := session.ExtractSession(r.Context())
if err != nil {
    r.LogError("Server configuration is wrong: sessions are required", "App")
    return r.HtmlResponse(kernel.HtmlResponseConfig{
        Code: 500,
        Html: "internal server error",
    })
}

// Set session data - errors if the key is already set
if err := sess.Set("my_key", data); err != nil {
    // "my_key" was already set - use SetForce below if you mean to overwrite it
}
// SetForce always (over)writes the value, no error
sess.SetForce("my_key", data)

// Get and remove session data
if sess.Has("some_key") {
    val := sess.Get("some_key")
    sess.Remove("some_key")
}

// Drop session
sessStorage, err := session.AppComponent(recource.App())
if err != nil {
    recource.LogError("Server configuration is wrong: sessions are required", "App")
    return recource.HtmlResponse(kernel.HtmlResponseConfig{
        Code: 500,
        Html: "internal server error",
    })
}
sessStorage.DestroySession(sess)
// ...
```


## License

Apache License 2.0 — see [LICENSE](./LICENSE).
