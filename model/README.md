# Package for describing DB models as yaml schemas + generating migrations from schema diffs

> Actual version: `v0.1.0-alpha.1`. [Details](https://github.com/epicoon/lxgo/tree/master/model/CHANGE_LOG.md)

> Requires an application based on [lxgo/kernel](https://github.com/epicoon/lxgo/tree/master/kernel); depends on
> [lxgo/migrator](https://github.com/epicoon/lxgo/tree/master/migrator) to actually run generated migrations.

Not to be confused with `lxgo/jspp`'s `lx.Model`/`lx.ModelSchema` - those are a client-side JS binding concept with
no relation to the database, despite the similar name.

## Contents

- [Application component](#application-component)
- [Console commands](#console-commands)
- [Model schema files](#model-schema-files)
  - [Field types](#field-types)
  - [Relations](#relations)
  - [Namespace, BaseModel, BaseRepo, Timestamps](#namespace-basemodel-baserepo-timestamps)
  - [Models and Repos output directories](#models-and-repos-output-directories)
- [Generated Go code](#generated-go-code)
  - [Model structs](#model-structs)
  - [Repositories](#repositories)
- [Using the Go API directly](#using-the-go-api-directly)
- [Service tables and auditing](#service-tables-and-auditing)
- [License](#license)

## Application component

`ModelManager` is a [`kernel.IAppComponent`](https://github.com/epicoon/lxgo/tree/master/kernel) - its `AfterInit`
registers this package's migration type (`Apply`/`Invert`) with `migrator`, so a generated migration is recognized
once the component is set up.

1. Add it to your app config - `Targets` accepts more than one schemas directory, each optionally overriding the
   component-wide defaults:
```yaml
Components:
  # ...
  ModelManager:
    Namespace: part1                                    # optional, component-wide default schema
    BaseModel: github.com/epicoon/lxgo/query.BaseModel  # optional, component-wide default base type
    BaseRepo: github.com/epicoon/lxgo/query.BaseRepo    # optional, component-wide default repo base type
    Timestamps: true                                    # optional, component-wide default (false if omitted)
    Targets:
      - Schemas: path/to/model/schemas
        Models: path/to/generated/models   # optional, see "Models and Repos output directories"
        Repos: path/to/generated/models    # optional, see "Models and Repos output directories"
        # Namespace: part2                      # optional, overrides the component default for this directory
        # BaseModel: gorm.io/gorm.Model         # optional, overrides the component default for this directory
        # BaseRepo: github.com/some/pkg.MyRepo  # optional, overrides the component default for this directory
        # Timestamps: false                     # optional, overrides the component default for this directory
```

2. Plug it in:
```go
import "github.com/epicoon/lxgo/model"

// app implements kernel.IApp
if err := model.SetAppComponent(app, "Components.ModelManager"); err != nil {
    // process err
}
```

3. Use it:
```go
mm, err := model.AppComponent(app)
if err != nil {
    // process err
}

schemas, err := mm.LoadModelSchemas() // every configured schema, Namespace/BaseModel/BaseRepo/Timestamps resolved
diffs, err := model.CompareSchemas(mm.DB(), schemas)
// or: actions, err := model.GenerateMigration(mm.DB(), schemas, "migration_name")
```

## Console commands

`NewCommand` builds the `model:db-status`/`model:db-migrate`/`model:db-audit`/`model:codegen-status`/
`model:codegen-generate`/`model:codegen-repos`/`model:actualize` [console command](https://github.com/epicoon/lxgo/tree/master/cmd)
- the `db-` actions manage the schema/database/migration side, the `codegen-` actions manage the generated Go
model/repository code side, and `actualize` runs both together interactively.

1. Make a command wrapper:
```go
import (
	"github.com/epicoon/lxgo/cmd"
	"github.com/epicoon/lxgo/model"
	"github.com/epicoon/lxgo/migrator"
)

func NewModelCommand(_ ...cmd.ICommandOptions) cmd.ICommand {
	// Create your app ...

  // Init the migrator
	migrator.Init(migrator.Config{
		DB:             app.Connection().DB(),
		MigrationsPath: "runtime/migrations",
		SeedsPath:      "runtime/seeds",
	})

  return model.NewCommand(model.CommandOptions{
		App: app,
	})
}
```

2. Plug in the command constructor:
```go
func main() {
	cmd.Init(cmd.CommandsList{
		// ...
		"model": NewModelCommand,
	})
	cmd.Run()
}
```

3. Use it:
    - `go run . model:db-status` - prints the diff between schema files and the database (added/deleted/changed
      fields per model, renames marked `explicit` or `heuristic`), or reports unapplied migrations instead if
      there are any.
    - `go run . model:db-migrate --name=add_widgets` - generates a migration file from the current diff.
    - `go run . model:db-migrate --name=add_widgets --apply` - generates the migration and immediately runs
      `migrator.Up()`.
    - `go run . model:db-audit` - reports service-table records that no longer match anything in the database (see
      [Service tables and auditing](#service-tables-and-auditing)). Prints what it finds; doesn't delete anything.
    - `go run . model:codegen-status` - for every `Target` with `Models` set, reports each model as
      `not generated`/`stale`/`up to date`, and (if `Repos` is also set) each repository as
      `not scaffolded`/`scaffolded`.
    - `go run . model:codegen-generate` - (re)generates every model under a `Target` with `Models` set (see
      [Model structs](#model-structs)).
    - `go run . model:codegen-repos` - scaffolds `<model>_repo.go` for every model under a `Target` with `Repos`
      set, **only if the file doesn't already exist** (see [Repositories](#repositories)).
    - `go run . model:actualize` - checks the schema diff and the generated code status together and prints a plan
      (schema diff, models to (re)generate, repositories to scaffold), or `Nothing to actualize`. Any already-
      pending unapplied migrations are applied first. Unless `--yes` is passed, asks to confirm (`Apply`/`Cancel`)
      before doing anything; on confirmation it regenerates every configured model, then - if the diff was
      non-empty - generates and applies a migration (name from `--name`, or prompted for interactively; `--yes`
      without `--name` when a migration is actually needed is an error), then scaffolds any missing repositories.
      `model:db-audit` isn't part of this check - it's a separate diagnostic.

## Model schema files

A model is one yaml file per model, named after the model (`GameSave.yaml` → model `GameSave` - the filename is
authoritative, an optional `Name:` key inside the file is purely informational):

```yaml
Name: GameSave
Fields:
  GameType: string required
  Data: string(4000) required
Relations:
  Gamers: (-<) GamerInGame.GameSave
```

`LoadModelSchema` parses a single file. Loading a whole directory (or several) goes through
`ModelManager.LoadModelSchemas()` instead (see [Application component](#application-component)) - every `*.yaml`
file across all configured `Targets`, sorted by name, into one combined result. `ModelSchema.Save` writes a schema
back out, preserving field order and each field/relation's original compact-vs-map form.

### Field types

A model attribute is declared as a `Field`: a type, whether it's required, an optional default, and type-specific
details (a size for `string`, precision/scale for `decimal`). Two equivalent forms are accepted - the structured
map form (keys capitalized - `Type`/`Required`/`Default`/`Size`/`Precision`/`Scale`/`RenamedFrom`):

```yaml
Name:
  Type: string
  Required: true
  Size: 255
  Default: "unnamed"
```

and a compact single-line form:

```yaml
Name: string(255) required default('unnamed')
```

`<type>[(<details>)]`, then `required` and/or `default(<literal>)` in either order. `<literal>` is a bareword (no
whitespace/quotes/parens) or a single-quoted string for anything that needs one: `default('hello world')`,
`default('{"count":0}')` - a literal quote inside is written doubled, `default('it''s ok')` → `it's ok`. Neither
form is a shorthand for the other - both validate identically, and a `Field` marshaled back to yaml keeps whichever
form it was read in.

`default` is validated against `type` at parse time - an invalid default is a hard parse error. Every type has its
own literal format:

| Type | Compact example | Literal format |
| --- | --- | --- |
| `string` | `name: string(255) required default('unnamed')` | any string; quote it if it contains whitespace |
| `int` | `sort: int default(0)` | a whole number |
| `float` | `ratio: float default(0.5)` | any number |
| `decimal` | `price: decimal(10, 2) default(19.99)` | checked against `precision`/`scale` - backed by [`shopspring/decimal`](https://github.com/shopspring/decimal), not a binary float |
| `bool` | `isActive: bool default(false)` | `true`/`false` (and yaml's other recognized bool spellings) |
| `date` | `birthDate: date default(2000-01-01)` | `"2006-01-02"` |
| `time` | `startTime: time default(09:00:00)` | 24-hour `"15:04:05"` |
| `datetime` | `publishedAt: datetime default(2026-01-02T15:04:05Z)` | RFC3339 **with an explicit timezone offset** - an offset-less literal is rejected, not assumed UTC |
| `interval` | `sessionTimeout: interval default(1h30m)` | a Go `time.ParseDuration` string |
| `dict` | `settings: dict default('{"theme":"dark"}')` | a JSON object/array - quoting is mandatory |

The same defaults, map form:

```yaml
Price:
  Type: decimal
  Precision: 10
  Scale: 2
  Default: "19.99"
Settings:
  Type: dict
  Default:
    theme: dark
```

A field can optionally declare `RenamedFrom: oldName` for the case where a field is renamed *and* its definition
changes in the same schema edit - the diff's rename detection can't tell that apart from a delete+add on its own
(see [Using the Go API directly](#using-the-go-api-directly)). Once a migration captures the rename, `RenamedFrom`
is cleared from the schema file **on disk** - `GenerateMigration` rewrites it automatically, no separate step
needed.

### Relations

`Relations:` declares a model's relations to other models. Two equivalent forms, same as `Field` - a compact
string:

```yaml
Relations:
  Gamers: (-<) GamerInGame.GameSave
```

or a structured map:

```yaml
Relations:
  Gamers:
    Type: OneToMany
    Model: GamerInGame
    Field: GameSave
```

`TYPE`/`Type:` is `--`/`OneToOne`, `-<`/`OneToMany`, `>-`/`ManyToOne` or `><`/`ManyToMany`. `RelatedModel`/`Model:`
and, if given, `.attribute`/`Field:` name the matching relation on the related model's own schema - required for
`OneToMany`/`ManyToMany` (otherwise there's no way to tell which of possibly several relations to that model this
one pairs with), optional for `OneToOne`/`ManyToOne`, in which case this side stands alone (a "uni" relation - the
other model doesn't need to declare anything back).

A non-uni relation is declared **symmetrically on both models** - `GameSave.Gamers: (-<) GamerInGame.GameSave`
above pairs with `GamerInGame.GameSave: (>-) GameSave.Gamers`. Loading a whole directory (or several, via
`ModelManager.LoadModelSchemas()`) cross-validates every non-uni relation against the whole batch: the related
model must exist, must declare a relation under the named attribute, that relation's type must be the correct
counterpart, and it must point back to this exact model/relation.

`RelationTypeOneToOne` additionally requires exactly one side to hold the foreign key, written
`(FK--)`/`OneToOne(FK)` instead of `(--)`/`OneToOne`. `RelationTypeManyToOne` is always the FK holder;
`RelationTypeOneToMany`/`RelationTypeManyToMany` never hold a single FK column at all (the former's FK lives on its
`RelationTypeManyToOne` counterpart, the latter uses a join table).

Postgres doesn't index a foreign key column automatically, but a generated migration creates one by default anyway
(the typical access pattern for a relation wants one). `NoIndex` (`no-index` in the compact form, `Index: false` in
the map form - inverted relative to the field, `true`/omitted means indexed) opts a relation's own column out of
it - valid only for `RelationTypeManyToOne`/`RelationTypeManyToMany` (the only two with an ordinary indexable FK
column of their own):

```yaml
Relations:
  Owner: (>-) User no-index
```

### Namespace, BaseModel, BaseRepo, Timestamps

Four independent per-model settings, each resolved through the same 3-level cascade by
`ModelManager.LoadModelSchemas()` - a model's own declared value, else its `Target`'s, else the component-wide
`ModelManager.Config` default - exposed as `ModelSchema.EffectiveNamespace()`/`EffectiveBaseModel()`/
`EffectiveBaseRepo()`/`EffectiveTimestamps()` (read these, not the raw fields, unless you specifically need to know
whether a value was declared locally):

```yaml
Name: Widget
Namespace: part1                                    # Postgres schema this model lives in
BaseModel: github.com/epicoon/lxgo/query.BaseModel  # Go type the generated struct embeds
BaseRepo: github.com/epicoon/lxgo/query.BaseRepo    # Go type a scaffolded repository embeds
Timestamps: true                                    # created_at/updated_at/deleted_at columns
Fields:
  Name: string(255) required
```

- **`Namespace`** - which Postgres schema the model's table lives in. Two models sharing a `Name` in different
  `Target`s is only an error if they also resolve to the same namespace.
- **`BaseModel`** - a bare `package/path.Type` string (always the full import path, e.g. `gorm.io/gorm.Model`, not
  a short alias), not parsed or interpreted beyond what [Model structs](#model-structs) needs it for. Not
  synchronized with this package's own DDL - the primary key is always `id serial`, regardless of what `BaseModel`
  declares; a model needing a wider primary key gets one through a hand-written `type: query` migration. Unset
  falls back to a bare `ID uint` field in generated code.
- **`BaseRepo`** - same shape as `BaseModel`, used by [Repositories](#repositories). Defaults to
  `github.com/epicoon/lxgo/query.BaseRepo` when unset anywhere in the cascade (unlike `BaseModel`, a repository
  always needs some base type to embed).
- **`Timestamps`** - a `*bool` (so an explicit `false` is distinguishable from "not declared, inherit"). When
  effective, `created_at`/`updated_at`/`deleted_at` become implicit columns the same way `id` is - a model can't
  also declare an ordinary field under one of those three names while it's on. Toggling it on an existing model is
  an ordinary part of the generated diff (an existing compatible column/index under one of these names is adopted,
  not recreated).

### Models and Repos output directories

```yaml
Targets:
  - Schemas: path/to/model/schemas
    Models: path/to/generated/models
    Repos: path/to/generated/models   # same package as Models
    # Repos: path/to/generated/repos  # a separate package instead
```

`Target.Models` names the directory generated Go model files are written into (one `<model>_gen.go` per schema);
`Target.Repos` names the directory scaffolded repository files are written into (one `<model>_repo.go` per schema
that doesn't already have one). Neither has a cascade - every model in a directory shares the same output
directory/Go package by construction. Empty means that target's models get no code generation/repository
scaffolding at all. Whether `Repos` resolves to the same directory as `Models` decides whether a scaffolded
repository references its model as a bare identifier (same package) or through an explicit import (see
[Repositories](#repositories)).

## Generated Go code

### Model structs

`BuildModelCode(pkgName string, schema *ModelSchema) ([]byte, error)` generates the Go source of a
[GORM](https://gorm.io)-mapped struct for `schema`, gofmt-formatted, always starting `// Code generated by
lxgo-model; DO NOT EDIT.` - meant to be overwritten whole on every regeneration
(`go run . model:codegen-generate`), never hand-edited. `ModelCodeFileName(modelName string) string` returns the
conventional file name (`Widget` → `widget_gen.go`).

| `FieldType` | Go type |
| --- | --- |
| `string` | `string` |
| `int` | `int64` |
| `float` | `float64` |
| `decimal` | `decimal.Decimal` (`github.com/shopspring/decimal`) |
| `bool` | `bool` |
| `date`, `time` | `string` |
| `datetime` | `time.Time` |
| `interval` | `time.Duration` |
| `dict` | `datatypes.JSON` (`gorm.io/datatypes`) |

Every field carries an explicit `column:...` tag (never left to GORM's naming guess), `not null` when `Required`,
and `default:...` when `Default` is set - GORM's `Create` reads it to decide whether a Go-zero-valued field should
be omitted from the `INSERT` column list. There's no `type:...` tag - `AutoMigrate` is never used against generated
code, this package builds all DDL itself.

`Timestamps` on, with a `BaseModel` that isn't `gorm.io/gorm.Model` or `github.com/epicoon/lxgo/query.BaseModel`
(both already carry timestamp fields of their own), generates explicit `CreatedAt`/`UpdatedAt`/`DeletedAt` fields
(`gorm.DeletedAt`, so GORM's own soft-delete filtering applies).

Relations: `RelationTypeOneToOne`/`RelationTypeManyToOne`/`RelationTypeManyToMany` get generated fields (tagged for
GORM's association loading, join table/column names computed the same way this package's own DDL does, not GORM's
naming guess); `RelationTypeOneToMany`'s reverse side has no codegen of its own - its physical shape is entirely
the `RelationTypeManyToOne` side's. A related model is always referenced as a bare identifier - `BuildModelCode`
assumes it's generated into the same package; `model:codegen-generate` rejects a relation pointing at a different
`Target`'s `Models` directory before ever calling `BuildModelCode`, rather than emitting code that fails to
compile.

### Repositories

`BuildRepoCode(pkgName string, schema *ModelSchema, modelPkg string) ([]byte, error)` generates a named repository
type wrapping the model's resolved `BaseRepo` - `<Model>Repo`, embedding `<BaseRepoType>[<Model>]`, plus a
`New<Model>Repo(db *gorm.DB) *<Model>Repo` constructor. `RepoCodeFileName(modelName string) string` returns
`<snake_case>_repo.go` (no `_gen` suffix, no `DO NOT EDIT` banner).

Unlike a model file, a repository file is meant to be hand-edited right after scaffolding (custom finder methods,
etc.) - `model:codegen-repos` writes it **at most once**, skipping (never overwriting, never erroring on) a file
that already exists. `modelPkg` only needs to be non-empty when `Repos` resolves to a different directory than
`Models` - resolved automatically from the nearest `go.mod` above `Models`.

## Using the Go API directly

Most usage goes through the [console commands](#console-commands). The underlying functions are exported for a
caller that wants to drive the same workflow programmatically - each is documented in full via its own Go-doc
comment; this is a map of what exists, not a restatement of how each one works.

- **`IntrospectModelSchema(db *sql.DB, tableName, pgSchema string, withTimestamps bool) (*ModelSchema, error)`** -
  builds a `ModelSchema` from a table's actual structure in Postgres, the reverse of loading one from a yaml file.
  Returns `ErrTableNotFound` (a distinct, checkable condition) if the table doesn't exist yet.
- **`SetColumnType`/`DeleteColumnType`** and **`SetRelationFk`/`DeleteRelationFk`** - record/remove metadata
  introspection can't recover from Postgres alone (see [Service tables and auditing](#service-tables-and-auditing)).
  Both pairs accept either a `*sql.DB` or a `*sql.Tx`.
- **`CompareFields`/`CompareRelations`/`CompareModel`/`CompareSchemas`** - diff schema files against the database
  (added/deleted/changed/renamed). `CompareSchemas(db, schemas)` returns `ErrUnappliedMigrations` without comparing
  anything if [`migrator.Check`](https://github.com/epicoon/lxgo/tree/master/migrator) reports any migration that
  hasn't been applied yet - requires `migrator.Init` already called by the caller.
- **`GenerateMigration(db *sql.DB, schemas []*ModelSchema, name string) ([]Action, error)`** - runs `CompareSchemas`
  and, if it found a diff, writes a migration file via
  [`migrator.CreateWithContent`](https://github.com/epicoon/lxgo/tree/master/migrator). Returns `nil` (no file
  written) if there was nothing to migrate. Same `migrator.Init` precondition as `CompareSchemas`.
- **`Apply`/`Invert`** - the pair registered with
  [`migrator.RegisterMigrationType`](https://github.com/epicoon/lxgo/tree/master/migrator) under `MigrationType`;
  run a generated migration's actions as real DDL, or undo them in reverse order. Called by `migrator`, not
  normally by application code directly.

A model/field's schema-file name is a logical name - what actually reaches the database is its physical name,
translated through the same zero-configuration `gorm.io/gorm/schema.NamingStrategy{}`
[`lxgo/query`](https://github.com/epicoon/lxgo/tree/master/query) itself uses, so a table this package creates and
a GORM model of the same name are queryable through either one interchangeably. Everywhere in this package's own
output (schema files, `model:db-status`, a migration's own content) the logical name is what's shown.

## Service tables and auditing

Some information can't be recovered from Postgres's own catalogs alone - which field type a `text` column was
declared as (`dict` and `string` look identical to Postgres), and which declared relation a bare foreign key
constraint implements. This package records that metadata in two tables of its own, `lx_sys.model_types` and
`lx_sys.model_relations` (created automatically on first use, in their own `lx_sys` Postgres schema), consulted by
`IntrospectModelSchema` and kept in sync by `Apply`/`Invert` as migrations run.

A row can become stale if a foreign key or column is dropped by hand (raw SQL, a manual schema change) rather than
through a generated migration - harmless to correctness (introspection only ever looks up a record for something
it already found physically present), but worth knowing about. `AuditRelationFks`/`AuditColumnTypes` (or
`go run . model:db-audit`) report exactly those stale rows; neither deletes anything, that's a deliberate follow-up
via `DeleteRelationFk`/`DeleteColumnType`.

## License

Apache License 2.0 — see [LICENSE](./LICENSE).
