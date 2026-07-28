------------------------------------------------------------------------------------------------------------------------
Date: 2026.07.28
Version: v0.1.0-alpha.7
Changes:
- fix: `Down(steps)` only clamped `steps` to 1 when it was exactly `0` - a negative `steps` fell through unclamped
  and could roll back far more migrations than intended
- refactor: `getMigrations`'s applied/unapplied filtering split out into a standalone `filterMigrations`, testable
  without a DB connection
- test: added unit tests for `filterMigrations`; added integration tests against a real Postgres

------------------------------------------------------------------------------------------------------------------------
Date: 2026.07.25
Version: v0.1.0-alpha.6
Changes:
- docs: Go-doc comments for every exported declaration in the package (`Config`, `Init`/`SetDB`/`SetMigrationsPath`,
  `Create`/`Check`/`Show`/`Up`/`Down`/`UpSeeds`, `MigratorCommand`/`NewCommand`) - previously undocumented

------------------------------------------------------------------------------------------------------------------------
Date: 2026.07.24
Version: v0.1.0-alpha.5
Changes:
- docs: README clarified - migrator:create's actual filename shape, that name/type in a migration file aren't read by the migrator (only the filename matters for tracking), up/down accepting a single statement or an ordered list, PostgreSQL-only for now; assorted typo fixes

------------------------------------------------------------------------------------------------------------------------
Date: 2026.07.13
Version: v0.1.0-alpha.4
Changes:
- fix: Up() no longer continues after a Check()/createTable() error (missing return)

------------------------------------------------------------------------------------------------------------------------
Date: 2026.03.19
Version: v0.1.0-alpha.3
Changes:
- add seeds maintenance
- add command :up-seeds

------------------------------------------------------------------------------------------------------------------------
Date: 2025.08.13
Version: v0.1.0-alpha.2
Changes:
- add console command configuration

------------------------------------------------------------------------------------------------------------------------
Date: 2025.07.24
Version: v0.1.0-alpha.1
