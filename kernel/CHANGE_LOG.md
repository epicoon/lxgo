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
