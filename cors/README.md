# CORS component for lxgo/kernel web-applications

> Actual version: `v0.1.0-alpha.1`. [Details](https://github.com/epicoon/lxgo/tree/master/cors/CHANGE_LOG.md)

> You can use it if your application is based on [lxgo/kernel](https://github.com/epicoon/lxgo/tree/master/kernel)

Nothing is CORS-enabled just because the component exists - a route gets CORS
response headers only once it's explicitly opted in (`EnableFor`/`EnableForAll`),
and only for an `Origin` listed in `Config.Origins`.

1. Add the app component in your app config file, with the origins allowed to
   make cross-origin requests:
```yaml
Components:
  # ...
  Cors:
    Origins:
      "http://domain.com:8000": {}
      "https://domain.com":
        ExposeHeaders: "X-Custom-Header"
```

2. Plug the application component:
```go
import (
	"github.com/epicoon/lxgo/cors"
)

// app implements kernel.IApp
if err := cors.SetAppComponent(app, "Components.Cors"); err != nil {
    // process err
}
```

3. Opt specific routes in - exact route or prefix (e.g. `"/img/"` matches
   every path under it):
```go
corsComponent, err := cors.AppComponent(app)
if err != nil {
    // process err
}
corsComponent.EnableFor([]string{"/img/"})
```
Or opt every route in at once:
```go
corsComponent.EnableForAll()
```
A route can also be explicitly excluded, regardless of `EnableFor`/
`EnableForAll` - an excluded route never gets CORS headers, even if it also
matches an enabled one:
```go
corsComponent.DisableFor([]string{"/lx/service"})
```

## Preflight requests

Only simple-request headers are added (`Access-Control-Allow-Origin`/
`Access-Control-Expose-Headers`) - a preflight `OPTIONS` exchange
(`Access-Control-Request-Method`/`-Headers`, `Access-Control-Allow-Methods`/
`-Allow-Headers`/`-Max-Age`) isn't handled yet. Fine for a simple cross-origin
request a browser never preflights (e.g. an `<img>` loaded from another
origin); not yet for one that needs an actual preflight round trip.

## License

Apache License 2.0 — see [LICENSE](./LICENSE).
