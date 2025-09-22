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
