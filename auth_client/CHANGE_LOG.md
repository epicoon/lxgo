------------------------------------------------------------------------------------------------------------------------
Date: 2026.08.06
Version: v0.1.0-alpha.5
Changes:
- add: `AuthClient.RefreshTokens(refreshToken string, scope ...string)` - an optional trailing `scope` narrows the
  reissued tokens' access (RFC 6749 §6); omit it to keep the current scope unchanged, same as before
- add: `/auth-refresh` (`RefreshHandler`/`RefreshRequest`) accepts an optional `scope` field in the request body,
  forwarded straight through to `RefreshTokens`
- test: unit tests for `RefreshTokens`'s scope handling (passed/omitted/empty), plus an integration test that a
  submitted `scope` actually reaches the auth server through the HTTP proxy handler

------------------------------------------------------------------------------------------------------------------------
Date: 2026.07.28
Version: v0.1.0-alpha.4
Changes:
- fix: `NewLogoutHandler()` built its `Resource` via a bare `&lxHttp.Resource{}` instead of `lxHttp.NewResource()`,
  skipping its constructor-side initialization and panicking as soon as the handler ran
- fix: `ExchangeCodeForTokens` ignored a failed exchange's `Success: false` response and built `Tokens` out of its
  empty fields anyway instead of returning the server's actual error message
- test: added unit tests for `GetBearer`, `Tokens.Set`, `AuthConfig`, and `ExchangeCodeForTokens`/`RefreshTokens`/
  `LogOut`; added integration tests for `StateHandler`/`AuthCallbackHandler`/`RefreshHandler`/`LogoutHandler` via full
  HTTP round trips

------------------------------------------------------------------------------------------------------------------------
Date: 2026.07.25
Version: v0.1.0-alpha.3
Changes:
- docs: Go-doc comments for every exported declaration in the package (`AuthClient`, `AuthConfig`, `Tokens`,
  `GetBearer`, and all 4 ready-made HTTP handlers with their request forms) - previously undocumented

------------------------------------------------------------------------------------------------------------------------
Date: 2026.07.24
Version: v0.1.0-alpha.2
Changes:
- add: Tokens now carries the granted Scope, matching lxgo-auth's new OAuth scope support
- refactor: request handlers migrated to CRequestForm/ProcessRequestErrors instead of manual FormFiller/SetRequired
- docs: README rewritten (was a TODO stub) - describes the package and full setup steps

------------------------------------------------------------------------------------------------------------------------
Date: 2025.07.24
Version: v0.1.0-alpha.1
