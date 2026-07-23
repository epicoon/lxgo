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
