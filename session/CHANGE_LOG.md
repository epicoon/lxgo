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
