------------------------------------------------------------------------------------------------------------------------
Date: 2026.09.02
Version: v0.1.0-alpha.31
Changes:
- add: `IApp` gains `SelfApp()`, mirroring the self-binding `AppComponent` already had - `InitApp` now hands an
  embedding app struct's own outward-facing instance back to the base `*app.App`, so code holding only a `kernel.IApp`
  reaches the outer struct's own method overrides instead of always resolving to `*app.App`'s
- fix: `http.Resource.App()` returned the base app straight from its context, losing any embedding struct's
  overrides - now returns `SelfApp()` when one is bound, falling back to the base app otherwise

------------------------------------------------------------------------------------------------------------------------
Date: 2026.08.17
Version: v0.1.0-alpha.30
Changes:
- add: `cast.Value` now coerces into a pointer target (of any element type, not just a specific one) - allocates and
  fills a fresh pointer via the same coercion rules as the pointed-to type, so a `*bool`/etc struct field (e.g. a
  3-level cascade's "not declared, inherit" vs. "explicitly set" distinction, which a plain `bool` field can't
  represent) can actually be populated from a `kernel.Dict`/`DictToStruct` call. Previously a hard "cannot assign"
  error for any pointer-typed target

------------------------------------------------------------------------------------------------------------------------
Date: 2026.08.06
Version: v0.1.0-alpha.29
Changes:
- add: `App.Run()` now handles `SIGINT`/`SIGTERM` - stops accepting new HTTP connections and waits (up to a
  `ShutdownTimeout`, default 5s, configurable via `config.yaml`) for in-flight requests to finish before returning,
  instead of blocking forever on `ListenAndServe`
- fix: registered components' `Run()` sat unreachable after the old blocking `ListenAndServe` call and never actually
  executed - fixed as part of the graceful-shutdown rework above
- fix: an embedding component's `LogCategory()` override (e.g. `session.Storage`) was never honored by
  `Log`/`LogWarning`/`LogError` - they always logged under the generic `"AppComponent"` category, since Go embedding
  has no virtual dispatch; `InitComponent` now binds the component back to itself so the override actually takes
  effect

------------------------------------------------------------------------------------------------------------------------
Date: 2026.08.05
Version: v0.1.0-alpha.28
Changes:
- add: `manage:trigger --event=NAME [--params=...]` - fires a custom app event (`app.Events().Trigger`) from outside
  the running process over the manage socket, as if it happened internally (was a stub before, always replying "Not
  implemented yet")
- add: nested forms - a named (non-embedded) `kernel.IForm`-typed field inside another form is now filled and
  validated in place (its own required-fields check, `AfterFill`/`Validate`, errors folded into the parent's),
  recursively for however many levels deep the nesting goes
- fix: an anonymous (embedded) struct field's own fields weren't populated by `cast.DictToStruct` - only
  `lxgo-kernel/http`'s separate field-mapping did that flattening, so a required field declared on an embedded (not
  top-level) form struct validated as present but was never actually filled
- fix: a required nested-form field was always treated as "present" by `checkMissingParams`, since its embedded
  `*Form` is never nil once constructed - a request missing that block entirely passed validation instead of being
  reported as missing
- fix: `applyEnv` skipped `${VAR}` placeholder resolution entirely when no `.env` file existed (and none was
  required) - placeholders were left unresolved literally instead of falling back to the process environment or
  their own default
- fix: `Router.defineResource` did a second, redundant "is this route registered" check (rebuilding the full list of
  registered routes on every single request) after the map lookup just above it already covered the same case
- remove: `IApp.SetRouter` dropped from the public interface - the router is now built directly in `NewApp()`
  instead of being set later during `InitApp`
- i18n: `manage:inconf`'s report messages translated from Russian to English, for consistency with the rest of the
  package's user-facing text

------------------------------------------------------------------------------------------------------------------------
Date: 2026.07.28
Version: v0.1.0-alpha.27
Changes:
- fix: `app.appPathfinder.GetAbsPath("")` panicked (indexed into an empty string before the `@alias`-prefix check)
  - found while testing `lxgo-jspp`, which passes an unset config path straight through to it
- test: added a regression test for the above

------------------------------------------------------------------------------------------------------------------------
Date: 2026.07.27
Version: v0.1.0-alpha.26
Changes:
- add: `cast` package - a single reflect-based coercion API (`cast.Value`/`cast.To[T]`/`cast.DictToStruct`/
  `cast.JsonToStruct`/`cast.MapToStruct`/`cast.FieldName`), replacing four independent, partially-inconsistent
  coercion implementations previously scattered across `app.SetConfigParam`, `config.GetParam`, `conv.GetDictItem`
  and `conv.setFieldValue` - fixes real inconsistencies found while consolidating them (e.g. `float64` from a numeric
  string used to fail via `GetDictItem` while working via `GetParam`/`SetConfigParam`; an `int` coerced into a
  `string` field could silently produce a garbage single-character string via Go's rune-conversion rule instead of
  the decimal value)
- remove: `conv` package - fully superseded by `cast`
- add: `kernel.IDict` interface (`Set`/`Get`/`Has`) - `kernel.Dict` now implements it
- remove: `kernel.Config` type - replaced by `kernel.IDict` across the public API (`IApp.SetConfig`/`Config()`,
  `IConnection.SetConfig`, `config.Load`/`GetParam`/`HasParam`/`SetParam`) and by `kernel.Dict` where a concrete map
  is actually needed; `Config.ToMap()`/`ToDict()` removed with it
- remove: `kernel.IData`/`kernel.Data`/`kernel.NewData`/`kernel.NewEmptyData` - replaced by `kernel.IDict`/
  `kernel.Dict` (`IEventManager.Trigger`, `IEvent.SetPayload`/`Payload` now take/return `kernel.IDict`)
- remove: `kernel.IForm.Fill(d *Dict) error` - dead code that was never called (form filling has always gone through
  `cast.DictToStruct`/tag matching) and never overridden anywhere in this workspace
- add: `apptest` package - `apptest.New`/`apptest.Server` build a minimal `kernel.IApp` (and, optionally, a real
  HTTP test server around its router) for other lxgo-* packages' integration tests, without needing a `config.yaml`
  file on disk
- fix: `internal/manage/reconf` config diffing (`manage:refresh-config`) panicked when comparing a newly-added array
  against a previously-empty one (e.g. `Servers: []` followed by `manage:inject-config --add Servers=[...]`)
- fix: `internal/manage/inconf` (`manage:inject-config --test`) no longer prints raw request params to stdout
- docs: clarified that `Run` still executes even when the request form failed validation and
  `ProcessRequestErrors` wasn't overridden - previously undocumented, easy to misread as a gap
- test: added unit tests for `cast`, `http` (form filling, request handling), `internal/manage/reconf`,
  `internal/manage/inconf` and `apptest` - previously the package had zero tests

------------------------------------------------------------------------------------------------------------------------
Date: 2026.07.25
Version: v0.1.0-alpha.25
Changes:
- rename: `app.Configurate` → `app.Configure`; `app.NewDIConteiner` → `app.NewDIContainer` (typo fixes)
- refactor: `app.NewAppPathfinder`/`template.NewTemplateHolder` now return `kernel.IPathfinder`/`kernel.ITemplateHolder`
  directly instead of an unexported concrete type
- refactor: `IHttpResource.PreRun()` replaced by `Base() IHttpResource` (mirrors `IApp.BaseApp()`) and
  `BeforeRunCallbacks() []func(res IHttpResource)` - the router now runs the registered hooks itself instead of the
  resource running them internally
- add: `IFormFiller` interface (`http.FormFiller()`'s return type) - `Fill()` now returns an `error` for
  misuse (no form set, no data source set, or both `SetContext`/`SetDict` set) instead of panicking; the router and
  `Resource.JsonResponse`/`FailResponse` log the error and return a generic 500 instead of crashing the request
- docs: Go-doc comments for the whole public API - package root plus every public subpackage (`app`, `cmd`, `config`,
  `conv`, `errors`, `events`, `http`, `template`, `utils`) - previously completely undocumented

------------------------------------------------------------------------------------------------------------------------
Date: 2026.07.24
Version: v0.1.0-alpha.24
Changes:
- add: IDIContainer.Register(list) - register additional DI entries after Init(), erroring on a duplicate key instead of silently overwriting
- docs: documented the existing database connection setup (config.yaml's Database section, Connection.Connect()/DB()) and the manage:inject-config command, both previously undocumented; assorted typo fixes

------------------------------------------------------------------------------------------------------------------------
Date: 2026.07.13
Version: v0.1.0-alpha.23
Changes:
- fix: utils.GenRandomHash() now panics instead of silently returning a hash of zero bytes when crypto/rand fails
- refactor: http.Lang(app, req) now logs cookie-parsing failures internally and no longer returns an error
- fix: App.Final() now logs a connection.Close() error instead of ignoring it

------------------------------------------------------------------------------------------------------------------------
Date: 2026.07.12
Version: v0.1.0-alpha.22
Changes:
- add: IHandleContext.Init()

------------------------------------------------------------------------------------------------------------------------
Date: 2026.03.19
Version: v0.1.0-alpha.21
Changes:
- fix map parsing for conv.DictToStruct()
- add IApp.ConfigParam(key string) any
- add EVENT_APP_BEFORE_FAIL
- add EVENT_APP_BEFORE_FINAL
- add IHttpResource.Redirect(URL string, code int, params map[string]any)
- add FormToMap(f IForm) map[string]any

------------------------------------------------------------------------------------------------------------------------
Date: 2025.12.21
Version: v0.1.0-alpha.20
Changes:
- refactor http

------------------------------------------------------------------------------------------------------------------------
Date: 2025.11.28
Version: v0.1.0-alpha.19
Changes:
- fix numbers parsing for conv.DictToStruct()

------------------------------------------------------------------------------------------------------------------------
Date: 2025.11.27
Version: v0.1.0-alpha.18
Changes:
- fix slices parsing for conv.DictToStruct()

------------------------------------------------------------------------------------------------------------------------
Date: 2025.10.29
Version: v0.1.0-alpha.17
Changes:
- fix manage socket test mode

------------------------------------------------------------------------------------------------------------------------
Date: 2025.10.29
Version: v0.1.0-alpha.16
Changes:
- refactor default DBconnection config values

------------------------------------------------------------------------------------------------------------------------
Date: 2025.10.28
Version: v0.1.0-alpha.15
Changes:
- add manage socket
- add runtime manage console command
- refactor app component
- add func utils.SliceDiff[T comparable](slice1, slice2 []T) []T

------------------------------------------------------------------------------------------------------------------------
Date: 2025.09.30
Version: v0.1.0-alpha.14
Changes:
- refactor routing

------------------------------------------------------------------------------------------------------------------------
Date: 2025.09.24
Version: v0.1.0-alpha.13
Changes:
- fix int cast from .env

------------------------------------------------------------------------------------------------------------------------
Date: 2025.09.22
Version: v0.1.0-alpha.12
Changes:
- refactor application pathfinder

------------------------------------------------------------------------------------------------------------------------
Date: 2025.09.22
Version: v0.1.0-alpha.11
Changes:
- add retry DB connecting, DB config params - ConnectAttempts, ConnectAttemptDelay (seconds)

------------------------------------------------------------------------------------------------------------------------
Date: 2025.09.21
Version: v0.1.0-alpha.10
Changes:
- config can use variables, in particular from .env file

------------------------------------------------------------------------------------------------------------------------
Date: 2025.09.18
Version: v0.1.0-alpha.8
Changes:
- if lxlang cookie is not found it's not the error anymore

------------------------------------------------------------------------------------------------------------------------
Date: 2025.09.03
Version: v0.1.0-alpha.7
Changes:
- fix recursive merge for local config

------------------------------------------------------------------------------------------------------------------------
Date: 2025.09.03
Version: v0.1.0-alpha.6
Changes:
- added local config

------------------------------------------------------------------------------------------------------------------------
Date: 2025.09.02
Version: v0.1.0-alpha.5
Changes:
- added proxy reauests handling
- HttpTemplateOptions renamed to HttpTemplateConfig

------------------------------------------------------------------------------------------------------------------------
Date: 2025.08.15
Version: v0.1.0-alpha.4
Changes:
- changed requests handling
- fix form filling with empty params

------------------------------------------------------------------------------------------------------------------------
Date: 2025.08.01
Version: v0.1.0-alpha.3
Changes:
- fixed empty params processing while rendering
- removed request handling dev message

------------------------------------------------------------------------------------------------------------------------
Date: 2025.07.31
Version: v0.1.0-alpha.2
Changes:
- removed event: EVENT_APP_BEFORE_RUN
- added function: app.RegisterComponent()
- refactored ITemplateRenderer
- changed event payload: EVENT_APP_BEFORE_HANDLE_REQUEST "resource" IHttpResource -> "context" IHandleContext

------------------------------------------------------------------------------------------------------------------------
Date: 2025.07.24
Version: v0.1.0-alpha.1
