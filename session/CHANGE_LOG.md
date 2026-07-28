------------------------------------------------------------------------------------------------------------------------
Date: 2026.07.28
Version: v0.1.0-alpha.6
Changes:
- fix: `Session` cached the `kernel.IHandleContext` it was created with, but a session is shared by ID across
  concurrent requests, so that cached context went stale as soon as the request that created it returned;
  `ISession.Context()` is gone, and `IStorage.DestroySession` now takes the response writer explicitly instead of
  reading it back off the stale context
- fix: `Session`'s methods are now safe for concurrent use (a `sync.RWMutex` guards its id/data/last-accessed state)
  - matches sessions being shared by ID across concurrent requests
- fix: `Storage.Scanner()` built its `Scanner` from the raw `s.provider` field instead of `s.getProvider()`, so
  calling it before the provider was lazily initialized elsewhere returned a `Scanner` with a nil provider
- test: added unit tests for `BaseProvider`, `Session`, and `Storage` - previously the package had zero tests

------------------------------------------------------------------------------------------------------------------------
Date: 2026.07.25
Version: v0.1.0-alpha.5
Changes:
- rename: `IScaner`/`Scaner` → `IScanner`/`Scanner` (typo fix); `Storage.Scaner()` → `Storage.Scanner()`
- docs: Go-doc comments for every exported declaration in the package (`IStorage`/`Storage`, `ISession`/`Session`,
  `IProvider`/`BaseProvider`, `IScanner`/`Scanner`) - previously undocumented

------------------------------------------------------------------------------------------------------------------------
Date: 2026.07.24
Version: v0.1.0-alpha.4
Changes:
- fix: Storage.GC()'s self-rescheduling treated MaxLifeTime as nanoseconds instead of seconds, so garbage collection ran in a tight loop instead of every MaxLifeTime seconds
- docs: README caught up to the actual Set()/SetForce()/Remove() API (Set errors on an already-set key, "Delete" is now "Remove") - previously described a stale API

------------------------------------------------------------------------------------------------------------------------
Date: 2026.07.13
Version: v0.1.0-alpha.3
Changes:
- fix: IStorage.StartSession()/SessionByID() now return an error and propagate session-provider failures instead of silently returning a nil session

------------------------------------------------------------------------------------------------------------------------
Date: 2026.03.19
Version: v0.1.0-alpha.2
Changes:
- add IProvider.AddSession(sess ISession, sid string)
- add IScaner.PrintContextContent(ctx kernel.IHandleContext) string

------------------------------------------------------------------------------------------------------------------------
Date: 2025.07.24
Version: v0.1.0-alpha.1
