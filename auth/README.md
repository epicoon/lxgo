# Authentication microservice

> Actual version: `v0.1.0-alpha.3`. [Details](https://github.com/epicoon/lxgo/tree/master/auth/CHANGE_LOG.md)

There are ways to use the service:
* [Full ready-to-use solution for lxgo/kernel applications](#full-sol)
* [Ready-to-use solution for browser](#browser-sol)
* [Implementation of integration for server on the client side](#server-imp)
* [Implementation of integration for browser on the client side](#browser-imp)

Microservice API:
* [API](https://github.com/epicoon/lxgo/tree/master/auth/README_API.md)

Running the service itself:
* [Deploying the service](#deploy)
* [Console commands](#cmds)
* [Admin API](#admin-api)

<a name="full-sol"><h3>Full ready-to-use solution for lxgo/kernel applications</h3></a>

There is a [package](https://github.com/epicoon/lxgo/tree/master/auth_client) with already done solutions for integration. You can use it if your application based on `lxgo/kernel`.


<a name="browser-sol"><h3>Ready-to-use solution for browser</h3></a>

> Tokens should be kept in local storage with keys: `lxAuthAccessToken`, `lxAuthRefreshToken`

1. Plug in ready-to-use JS-bundle:
```html (twig)
<script defer src="{{ auth_server_url }}/js-client/bundle.js"></script>
```
You'll get an object `lxAuth`:
* Events:
    - `lxAuth.TOKENS_FOUND` - if lxAuth can get tokens from the local storage
    - `lxAuth.TOKENS_NOT_FOUND` - if tokens are not in the local storage
    - `lxAuth.TOKENS_REMOVED` - after drop tokens from the local storage
* Methods:
    - `lxAuth.goToAuth()` - redirect to the auth server form
    - `lxAuth.logOut()` - log client out
    - `lxAuth.run()` - launching the Authentication Manager
    - `lxAuth.on(event, handler)` - subscribe to an lxAuth event
    - `lxAuth.async fetch(url, params = {})` - function-wrappet for `fetch` to ensure access token will be sent in auth header
    - `lxAuth.async getUserData()` - request `{success, login, data}` for the currently authenticated user (via `user_data_path`, see "Data Structure" below)

2. The way to use `lxAuth` object:
```js
loginBut.addEventListener('mousedown', () => lxAuth.goToAuth());
logoutBut.addEventListener('mousedown', () => lxAuth.logOut());

lxAuth.on(lxAuth.TOKENS_NOT_FOUND, () => {
    // Display a button to allow login or an immediate redirect to the authorization service form depending on the needs of the application
});

lxAuth.on(lxAuth.TOKENS_FOUND, async () => {
    // Request to obtain user data
    const userData = await lxAuth.getUserData();
    if (userData.success) {
        // Further data using
    }
});

lxAuth.on(lxAuth.TOKENS_REMOVED, () => {
    // Show a button to allow login
});

// Launching the Authentication Manager
lxAuth.run();
```

3. To provide the Authentication Manager with access to the data it needs to operate, you need to assign a specific data structure (see "Data Structure" below) in JSON string format to the global variable `_lxauth_settings` (see the example below "Example of data preparation"). The Authentication Manager will read this data when it starts and delete the global variable.
Data Structure:
```yaml
authSettings:
    # Client application identifier
    id: 1
    # Ajax method to generate unique string for CSRF protection
    state_path: /gen-state
    # Endpoint for proxying request to /logout of authorization service
    logout_path: /logout
    # Endpoint for proxying request to /refresh of authorization service
    refresh_path: /auth-refresh
    # Endpoint for proxying request to /user-data of authorization service
    user_data_path: /get-user
    # URL of the authorization service
    server: http://auth_server_url
    # Endpoint for redirection from the authorization service form back to the client application
    redirect_uri: http://client_url/auth-callback
```
Example of data preparation
```html (twig)
<script>
    window._lxauth_settings = {{ authSettingsJSON }}
</script>
```


<a name="server-imp"><h3>Implementation of integration for server on the client side</h3></a>

If you're not using [lxgo/auth_client](https://github.com/epicoon/lxgo/tree/master/auth_client) directly (see
["Full ready-to-use solution"](#full-sol) above for that), here's what its handlers actually do — a
reimplementation on another stack needs to replicate the same five endpoints. All of them are simple
proxies/wrappers around the [microservice API](https://github.com/epicoon/lxgo/tree/master/auth/README_API.md); none
of them talk to the authorization service's database directly.

* **Endpoint for redirection from the authorization service form back to the client application**
  (reference: `auth_callback_handler.go`). Request params from the Authentication Server: `code` (string, unique
  string to exchange for tokens), `state` (string, CSRF protection value).
    1. Check the returned `state` against the value saved earlier (the reference implementation keeps it in the
       session, under the same key the state-generation endpoint below wrote it to).
    2. Exchange `code` for a token pair — `POST {server}/tokens` with `grant_type: authorization_code`, `code`,
       `client_id`, `client_secret` (see [/tokens](https://github.com/epicoon/lxgo/tree/master/auth/README_API.md#r3)).
       The response also carries `scope` — the access level actually granted, which was requested (or defaulted) back
       at `/auth`, not something this exchange step controls itself.
    3. Store the tokens on the client. The reference implementation returns a small HTML page whose only content is
       a `<script>` that writes both tokens into `localStorage` (keys `lxAuthAccessToken`/`lxAuthRefreshToken`, each
       value a `["<token>", <unix_expires_at>]` JSON-encoded pair — this exact format is what the [ready-to-use
       browser solution](#browser-sol) and the [browser-side reimplementation notes](#browser-imp) below expect).
    4. Redirect the user to the page they originally started from (the reference implementation saved that URL
       alongside `state` in the previous step, defaulting to `/` if none was recorded).
* **Endpoint for generating the unique string for CSRF protection** (reference: `state_handler.go`). Accepts an
  optional `uri` (the page to return to once authentication completes, defaulting to `/`), generates a random
  string, saves both the string and `uri` server-side (session, in the reference implementation) so the callback
  endpoint above can validate/use them, and responds with `{"state": "<generated string>"}`.
* **Endpoint for proxying a request to `/logout` of the authorization service** (reference: `logout_handler.go`).
  Extracts the access token from the incoming request's `Authorization: Bearer <token>` header, calls
  [/logout](https://github.com/epicoon/lxgo/tree/master/auth/README_API.md#r5) with that token and the client's
  `client_id`, and relays `{"success": "ok"}` (or an error) back.
* **Endpoint for proxying a request to `/refresh` of the authorization service** (reference: `refresh_handler.go`).
  Accepts `refresh_token`, calls
  [/refresh](https://github.com/epicoon/lxgo/tree/master/auth/README_API.md#r4) with `grant_type: refresh_token`,
  `client_id`, `client_secret`, `refresh_token`, and relays the new token pair
  (`access_token`/`access_token_expired`/`refresh_token`/`refresh_token_expired`/`scope`) back as JSON. The reference
  implementation doesn't expose scope narrowing to the browser (it never forwards an optional `scope` of its own to
  `/refresh`) — it just keeps whatever scope the tokens already had.
* **Endpoint for proxying a request to `/user-data` of the authorization service.** Unlike the four above,
  `lxgo/auth_client` doesn't ship a ready-made handler for this one — build it the same way: extract the bearer
  token, call [/user-data](https://github.com/epicoon/lxgo/tree/master/auth/README_API.md#r2) with it and the
  client's `client_id`, and relay `{"success", "login", "data"}` back. `lxgo/auth_client`'s `AuthClient.GetUserData(accessToken)`
  (`auth_client.go`) already does the actual HTTP call and response parsing for you — a handler just needs to wrap
  it (get the bearer token via `auth_client.GetBearer(ctx)`, call `AppComponent(app)` to get the `*AuthClient`, call
  `.GetUserData(token)`, and serialize the result), same shape as `logout_handler.go`/`refresh_handler.go` above.
  `data` comes back empty unless the token's scope is `profile:data` (see [/user-data](https://github.com/epicoon/lxgo/tree/master/auth/README_API.md#r2)) -
  a `profile`-scoped token only gets `login`. There's also a write side -
  [`POST /user-data`](https://github.com/epicoon/lxgo/tree/master/auth/README_API.md#r7) - for the client application
  to report that data in the first place; `lxgo/auth_client` doesn't have a ready wrapper for it either, so it needs
  the same manual treatment (same headers/scope requirement as the `GET` above).


<a name="browser-imp"><h3>Implementation of integration for browser on the client side</h3></a>

If you're not using the [ready-to-use JS bundle](#browser-sol) above, here's what it actually does internally
(reference: `client/js/apps/client/src/App.js` and `AuthManager.js`) — a reimplementation needs to replicate this
behavior:

* On load, read and parse `window._lxauth_settings` (see "Data Structure" in [Ready-to-use solution for
  browser](#browser-sol) above), then delete the global variable. If it's missing/invalid, the manager stays
  inactive (every method becomes a no-op rather than throwing).
* `run()` checks whether a still-valid access token, or failing that a still-valid refresh token, is present in
  `localStorage` — triggering `TOKENS_FOUND` or `TOKENS_NOT_FOUND` accordingly. A token is "valid"/"active" simply
  by not being past its stored expiry timestamp.
* `goToAuth()` first requests a CSRF `state` from `state_path` (`POST`, body `{uri: window.location.href}` — the
  current page, so the server-side callback above can redirect back here), then **redirects via a same-origin
  `POST`** built from a dynamically created, auto-submitted `<form>` (not a `fetch`/`XHR`, since this needs to be a
  real browser navigation) to `{server}/auth` with `response_type: 'code'`, `client_id`, `redirect_uri`, `state`. It
  doesn't send `scope`, so the server defaults to the narrowest one (`profile`) — a reimplementation that needs
  `profile:data` has to add the field itself.
* `logOut()` sends `GET {logout_path}` with the current access token as a bearer header; on success it clears both
  tokens from `localStorage` and triggers `TOKENS_REMOVED`. If there's no valid refresh token to begin with, it
  just clears storage and triggers the event without a network request.
* `refreshTokens()` sends `POST {refresh_path}` with `{refresh_token}`, and on success overwrites both tokens in
  `localStorage` from the response (`access_token`/`access_token_expired`/`refresh_token`/`refresh_token_expired`).
* `getUserData()` requests `user_data_path` through the same wrapped-fetch mechanism the public `fetch()` uses (see
  next point), returning `{success, login, data}` (or `{success: false}` on failure).
* The wrapped `fetch(url, params)` (exposed as `lxAuth.fetch`) is what actually attaches the `Authorization: Bearer`
  header automatically before every request: if the current access token isn't active, it first tries
  `refreshTokens()` (bailing out — returning `null` without making the request — if that fails or there's no valid
  refresh token either), then adds the header and calls the real `fetch`.


<a name="deploy"><h3>Deploying the service</h3></a>

`lxgo/auth` is a [lxgo/kernel](https://github.com/epicoon/lxgo/tree/master/kernel) application, so the general setup
(config file, local config, database connection) follows the same rules described there — this section only covers
what's specific to `lxgo/auth`.

1. **Configuration.** `config.yaml` holds the shared/committed part (templates, session cookie name); anything
   machine-specific (`Port`, `Database`, `Settings`) belongs in a git-ignored `config-local.yaml` referenced via
   `Local: config-local.yaml` — see kernel's [local config](https://github.com/epicoon/lxgo/tree/master/kernel#lconfig).
2. **Database.** Add a `Database` section (in the local config, per above) — see kernel's [database
   connection](https://github.com/epicoon/lxgo/tree/master/kernel#db) for the exact keys.
3. **Run migrations** to create the schema: `go run . migrator:up` (see [Console commands](#cmds) below and
   [lxgo/migrator](https://github.com/epicoon/lxgo/tree/master/migrator) for the command in general).
4. **Create at least one OAuth client** — the service has nothing to authenticate against without one:
   `go run . client:new --redirect-uri=<your callback URL>`. The secret is printed once; save it.
5. **Bootstrap the service's own admin client and first superadmin** (only needed if you'll use the [admin
   API](#admin-api) below - skip this if you don't need it yet):
   1. Register a client for the service to authenticate its own operators against:
      `go run . client:new --redirect-uri=<your admin tool's callback URL>` - same command as step 4, nothing
      admin-specific about this client itself (see [Admin API](#admin-api) for why it still has to be a real,
      dedicated `Client`).
   2. Put its id into `config-local.yaml`: `Settings.AdminClientID: <that id>`. Every admin-gated endpoint only
      accepts tokens issued through this one client - see [Admin API](#admin-api).
   3. Create the first superadmin (this also creates the underlying `User`): `go run . admin:new --login=<login>
      --password=<password>`.
   4. From here on, an admin authenticates exactly like any other end user - `/auth` → `/tokens` through the client
      from step 1, with that login/password.
6. **Run the server**: `go run .` (the default, unnamed command — see [Console commands](#cmds)).


<a name="cmds"><h3>Console commands</h3></a>

* `go run .` (no command name) — runs the server itself.
* `go run . client:<action>` — manage OAuth2 clients (the `Client` entity these endpoints authenticate against, not
  end-user accounts — those are the `User` table, created via [`/signup`](#server-imp)/`UsersRepo.Create`, with no
  console command of their own):
    - `client:new --redirect-uri=<uri> [--secret=<value>]` — create a client (same thing [`POST
      /clients`](https://github.com/epicoon/lxgo/tree/master/auth/README_API.md#r8) does over HTTP - use whichever is
      convenient). If `secret` isn't given, one is generated and printed once.
    - `client:new-secret --id=<id>` — regenerate a client's secret (printed once).
    - `client:show --id=<id> --secret=<secret>` — print a client's data (redirect_uri, timestamps).
    - `client:del --id=<id>` — delete a client.
  Run `go run . client --help` for the full auto-generated reference (descriptions/types/required/defaults for every
  action and parameter).
* `go run . admin:new --login=<login> --password=<password> [--role=superadmin|admin]` — bootstrap a new admin
  (`role` defaults to `superadmin`); creates the underlying `User` too. See [Admin API](#admin-api) - there's no
  console command to manage admins beyond this one bootstrap action, further admin management is meant to go through
  that API once it grows past the current single `DELETE /admin/clients` operation.
* `go run . migrator:<action>` — manage the DB schema; see
  [lxgo/migrator](https://github.com/epicoon/lxgo/tree/master/migrator) for the available actions
  (`up`/`down`/`create`/...).
* `go run . apidoc:gen` — regenerate `ApiDoc.md` from the actually registered routes/forms.


<a name="admin-api"><h3>Admin API</h3></a>

`Admin` - a `User` marked as an operator of the service itself. An admin authenticates exactly like any other end user
(`/auth` → `/tokens`), but only through the one `Client` configured as `Settings.AdminClientID` (see
["Deploying"](#deploy) above) - a token issued through any other client is rejected outright, even for the same
underlying admin `User`, so a compromised/malicious third-party client can never borrow an admin's identity to reach
these endpoints.

* [`DELETE /admin/clients`](https://github.com/epicoon/lxgo/tree/master/auth/README_API.md#r9) - delete a client.
  Currently the only admin operation; any admin role (`admin`/`superadmin`) is sufficient for it. `Right`/role-level
  distinctions beyond "is this user an admin at all" aren't enforced yet - not needed until there's more than one
  admin operation to tell apart.


## OAuth2 grant types

Only the **Authorization Code Grant** (RFC 6749 §4.1) is implemented — the flow documented above (`/auth` →
`/tokens`), plus reissuing tokens via `/refresh`. Neither of these is currently wired up, and there's no ETA for
either — noted here so it isn't quietly assumed to exist later:
* **Client Credentials Grant** (RFC 6749 §4.4) — machine-to-machine authentication with no end user. `Client`/token
  creation is currently modeled around always having a `User` attached (see `TokensRepo.CreateAccessToken(client,
  user)`), so this isn't just a missing branch in a handler — it needs its own token-issuing path with no user.
* **PKCE** (RFC 7636) — `Client.Secret` is mandatory and every token exchange sends `client_secret`, i.e. every
  client is currently assumed to be a confidential client capable of holding a secret server-side. There's no path
  for a public client (an SPA/mobile app with no backend of its own) to authenticate safely without one.

(Implicit Grant and Resource Owner Password Credentials are deliberately not on this list — both are deprecated in
OAuth 2.1, so not having them is intentional, not a gap.)


## License

Apache License 2.0 — see [LICENSE](./LICENSE).
