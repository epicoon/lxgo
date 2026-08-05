------------------------------------------------------------------------------------------------------------------------
Date: 2026.08.06
Version: v0.1.0-alpha.8
Changes:
- refactor: `assets:build` now compiles through `lxgo-jspp`'s new standalone compiler entry point
  (`jspp/compiler.Builder()`) directly, instead of spinning up a throwaway `kernel.IApp` and registering a
  `JSPreprocessor` component just to reach `CompilerBuilder()`
- docs: `README.md`'s `refresh_handler.go` reference description now mentions the optional `scope` param (see
  `lxgo-auth_client`'s scope-narrowing feature)

------------------------------------------------------------------------------------------------------------------------
Date: 2026.08.05
Version: v0.1.0-alpha.7
Changes:
- add: `assets:build` command - compiles `client/js/{client,form}`'s bundles through `lxgo-jspp`'s own compiler
  instead of a separate webpack/babel/npm toolchain
- refactor: `client/js/apps/{client,form}` flattened to `client/js/{client,form}`; the committed `node_modules`,
  `package-lock.json`, and webpack/babel config removed (superseded by `assets:build` above)
- internal: added missing `golang.org/x/sys`/`x/term` indirect requires to `go.mod` (Go's module-graph pruning needs
  them explicit even under a local `go.work` replace) - fixes red/unresolved imports in some IDEs

------------------------------------------------------------------------------------------------------------------------
Date: 2026.07.28
Version: v0.1.0-alpha.6
Changes:
- fix: `RefreshHandler.Run()`'s `SaveTokens` call went through a fresh `TokensRepo()` instance that was never
  attached to the surrounding transaction (`SetTx` was never called) - if the second of its two sequential `Save()`
  calls failed, the first stayed committed with nothing to roll it back
- internal: adapted to `lxgo-session`'s `DestroySession` signature change (now takes the response writer explicitly)
- test: the entire pre-existing `internal/handlers/tests/` suite (10 files) never actually validated a handler's
  response - `Router.Handle`'s returned `IHttpResponse` was discarded instead of being sent to the test recorder, so
  assertions ran against a recorder stuck at its default 200/empty body regardless of what the handler returned; all
  call sites fixed
- test: `testutils` now spins up a throwaway dockerized Postgres per run (`//go:build integration`) instead of
  assuming an already-running local database, and gained an admin-client fixture (`CreateAdminUser`) so admin-gated
  endpoints are actually testable
- test: added integration tests for `CreateClientHandler`, `DeleteClientHandler` (including the admin-gate's
  security properties), `RefreshHandler`'s scope-narrowing, and `AuthHandler`'s `redirect_uri` mismatch check; added
  unit tests for `validateLogin`/`validatePassword` and `models.ScopeIncludes`/`ValidateScope`

------------------------------------------------------------------------------------------------------------------------
Date: 2026.07.27
Version: v0.1.0-alpha.5
Changes:
- internal: adapted to `lxgo-kernel`'s `Config` removal - `cvn.IApp.Settings()` now returns `kernel.IDict` instead of
  `*kernel.Config`; no visible change to the CLI (everything touched lives under `internal/`)

------------------------------------------------------------------------------------------------------------------------
Date: 2026.07.25
Version: v0.1.0-alpha.4
Changes:
- docs: Go-doc comments for the `cmd` package's exported declarations - the module's only real public Go API (root is
  `package main`, everything else lives under `internal/`)
- docs: removed leftover references to internal task-tracking files from code comments (`AdminCommand`,
  `internal/models/role.go`, `internal/models/admin.go`) - no behavior change

------------------------------------------------------------------------------------------------------------------------
Date: 2026.07.24
Version: v0.1.0-alpha.3
Changes:
- add: OAuth scope support (`profile`/`profile:data`) - requested via `scope` on `/auth`, carried through the
  authorization code and issued tokens; `/refresh` can narrow the granted scope but never broaden it (RFC 6749 §6)
- add: `POST /user-data` lets a client application store arbitrary JSON data for the current user (gated by the
  `profile:data` scope); `GET /user-data` now actually returns it instead of a hardcoded stub
- add: self-service client registration - `POST /clients` (open, no auth) lets any application register itself as
  an OAuth client
- add: service administrators - `Admin` model/repo, `admin new` CLI command to bootstrap the first superadmin, and
  an admin-gated `DELETE /admin/clients` endpoint (requires a token issued by the configured `Settings.AdminClientID`)
- fix: `redirect_uri` is now validated on `/auth` against the client's registered URI - previously accepted
  unchecked (a `//TODO`)
- refactor: `Role`/`ROLE_*` now apply only to `Admin`, not to `Client` - a `Client` no longer has a `RoleID`
- rename: `ClietnsRepo` → `ClientsRepo` (typo)
- refactor: request forms migrated to the `CRequestForm`/`ProcessRequestErrors` convention instead of manual
  `FormFiller`/`SetRequired` calls

------------------------------------------------------------------------------------------------------------------------
Date: 2026.07.12
Version: v0.1.0-alpha.2
Changes:
- fix typos

------------------------------------------------------------------------------------------------------------------------
Date: 2025.07.24
Version: v0.1.0-alpha.1
