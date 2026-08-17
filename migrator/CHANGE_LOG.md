------------------------------------------------------------------------------------------------------------------------
Date: 2026.08.11
Version: v0.1.0-alpha.10
Changes:
- add: `RegisterMigrationType(name string, apply, invert FMigrationApply) error` - plugs a custom
  migration type into `Up`/`Down` dispatch, alongside the built-in `query` type (a plain SQL list
  under `Up`/`Down`). A migration file declaring `Type: <name>` is dispatched to `apply`/`invert`
  instead of being read as SQL; `name` must not already be registered. `FMigrationApply func(tx
  *sql.Tx, raw []byte) error` receives the migration file's entire, untouched content - `migrator`
  doesn't interpret it beyond reading the top-level `Type` field to pick the handler
- add: `CreateWithContent(name string, content []byte) error` - writes a migration file with
  caller-supplied content (its own `Type` field included), the same timestamped-filename
  convention `Create` uses - for a caller building whole migration files programmatically (e.g.
  [`lxgo/model`](https://github.com/epicoon/lxgo/tree/master/model), the first registered custom
  migration type) rather than filling in the `query` template `Create` writes
- change (breaking): the applied-migrations table moved from `public._lxgo_migrator` to
  `lx_sys.migrator` - its own Postgres schema, created automatically (`CREATE SCHEMA IF NOT EXISTS
  lx_sys`) the first time it's needed, grouping it with other `lx`-family packages' service tables
  instead of standing out via a long name prefix in `public`. The DB role migrations run under
  needs `CREATE SCHEMA` (or the schema must already exist, created ahead of time by a role that
  does)
- upgrade: an existing deployment's `public._lxgo_migrator` isn't migrated automatically - run this
  once, by hand, against each database before deploying this version:
  ```sql
  CREATE SCHEMA IF NOT EXISTS lx_sys;
  ALTER TABLE public._lxgo_migrator SET SCHEMA lx_sys;
  ALTER TABLE lx_sys._lxgo_migrator RENAME TO migrator;
  ```

------------------------------------------------------------------------------------------------------------------------
Date: 2026.08.07
Version: v0.1.0-alpha.9
Changes:
- change (breaking): a migration file's own protocol keys are now capitalized - top-level `Name`/`Type`, and the
  built-in `query` type's `Up`/`Down` SQL sections; `Type`'s value (`query`, or a registered custom type's name) is
  unaffected, only the key. No backward compatibility with the old lowercase spelling

------------------------------------------------------------------------------------------------------------------------
Date: 2026.08.05
Version: v0.1.0-alpha.8
Changes:
- internal: added missing `golang.org/x/sys`/`x/term` indirect requires to `go.mod` (Go's module-graph pruning needs
  them explicit even under a local `go.work` replace) - fixes red/unresolved imports in some IDEs

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
