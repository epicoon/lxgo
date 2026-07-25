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
