------------------------------------------------------------------------------------------------------------------------
Date: 2026.10.05
Version: v0.1.0-alpha.1
Changes:
- add: initial release - `Cors`, an optional `kernel.IAppComponent` adding CORS response headers
  (`Access-Control-Allow-Origin`/`Access-Control-Expose-Headers`) to opted-in routes. `Config.Origins` is a whitelist
  of allowed `Origin` values, each with its own settings; `EnableFor`/`EnableForAll`/`DisableFor` opt routes in or out
  by exact route or prefix, with `DisableFor` always winning over either enable form on a matching route. Preflight
  (`OPTIONS`) handling isn't implemented yet - fine for a simple cross-origin request (e.g. an `<img>` src), not yet
  for one that needs an actual preflight exchange.
