# Package HTTP sessions in lxgo/kernel web-applications

> You can use it if your application is based on [lxgo/kernel](https://github.com/epicoon/lxgo/tree/master/kernel)

1. Add app component in your app config file:
```yaml
Components:
  # ...
  SessionStorage:
    CookieName: lxgosessid
    MaxLifeTime: 36000
```

> Sessions will be associated with request contexts (`kernel.IHandleContext`)

2. Example how to use sessions:
```go
// ... somewhere in [[kernel.IHttpResource.Run()]]
// r is kernel.IHttpResource

// Get session
sess, err := session.ExtractSession(r.Context())
if err != nil {
    r.LogError("Server configuration is wrong: sessions are required", "App")
    return r.HtmlResponse(kernel.HtmlResponseConfig{
        Code: 500,
        Html: "internal server error"
    })
}

// Set session data
sess.Set("my_key", data)
// or sess.SetForce - to rewrite value

// Get and delete session data
if sess.Has("some_key") {
    val := sess.Get("some_key")
    sess.Delete("some_key")
}

// Drop session
sessStorage, err := session.AppComponent(r.App())
if err != nil {
    r.LogError("Server configuration is wrong: sessions are required", "App")
    return r.HtmlResponse(kernel.HtmlResponseConfig{
        Code: 500,
        Html: "internal server error"
    })
}
sessStorage.DestroySession(sess)
// ...
```
