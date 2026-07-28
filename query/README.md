# The package helps to work with DB

> Actual version: `v0.1.0-alpha.5`. [Details](https://github.com/epicoon/lxgo/tree/master/query/CHANGE_LOG.md)

> You can use it if your application is based on [lxgo/kernel](https://github.com/epicoon/lxgo/tree/master/kernel)

A thin, generics-based layer over [GORM](https://gorm.io/) - a `BaseRepo[T]` with the common CRUD methods already
written, plus a small condition-builder (`QueryBuilder`/`query.And`/`query.Eq`/...) for the cases where GORM's own
chained API gets awkward (dynamic AND/OR trees, filtering by a related model's field, JSONB fields).

## Content:
* [BaseRepo](#link1)
* [QueryBuilder](#link2)
* [Remain](#link3)


### <a name="link1">BaseRepo:</a>

Structure `BaseRepo` implements `IBaseRepo` and `IRepoTx`

* Create repository:
    ```go
    import "github.com/epicoon/lxgo/query"

    // gormDB *gorm.DB
    // allowedFields []string - currently unused by BaseRepo itself; kept for repos built on top of it
    repo := query.NewBaseRepo[modelStruct](gormDB, allowedFields)
    ```

* Inherit repository example:
    ```go
    type UserRepo struct {
        *query.BaseRepo[models.User]
    }

    /** @constructor */
    func NewUserRepo(db *gorm.DB) *UserRepo {
        return &UserRepo{BaseRepo: query.NewBaseRepo[models.User](db, []string{})}
    }
    ```

* `IBaseRepo[T]` gives you `Create`/`CreateFromMap`, `ReadByID`/`ReadByIDs`/`ReadBy`/`ReadWhere`/`ReadAll`/`ExistsByID`,
  `Update`/`UpdateByID`/`UpdateFromMap`, `DeleteByID`/`ForceDeleteByID` (soft-delete vs. `Unscoped()`), `Count()`, and
  `QueryBuilder()` (see below) - all working on `*T`/`[]*T` and backed by `DB()` (the transaction if one is set via
  `SetTx`, the plain connection otherwise).

* Transaction example:
    ```go
    // db *gorm.DB
    tx = db.Begin()
    repo1.SetTx(tx)
    repo2.SetTx(tx)

    // ...

    // Something went wrong <=> problem == true:
    if problem {
        // Rollback transaction
        tx.Rollback()
    }

    // Commit transaction
    if err = tx.Commit().Error; err != nil {
        // Do something
    }
    ```


### <a name="link2">QueryBuilder:</a>

`repo.QueryBuilder()` returns an `IQueryBuilder[T]` for building a query GORM's own chained API doesn't cover as
cleanly - conditions as a tree (`query.And`/`query.Or` of nested `query.Eq`/`query.In`/...), filtering by a related
model's field (`"Role.Name"` - auto-joins `Role`, no manual `Joins(...)` needed), or a JSONB field (`"AuthData->role"`
- kept as-is, no dot-splitting).

* Example:
    ```go
    import "github.com/epicoon/lxgo/query"

    // WHERE status = 'inactive' AND email IN ('1@1.1', '2@2.2') AND role.name = 'admin'
    // repo IBaseRepo
    qb := repo.QueryBuilder().
        With("Role").
        Where(query.And(
            query.Eq("Status", "inactive"),
            query.In("Email", []string{"1@1.1", "2@2.2"}),
            query.Eq("Role.Name", "admin"),
        ))
    count, _ := qb.Count()
    page, _ := qb.
        PerPage(10).
        Page(1). // 1-based; Page() only has an effect once PerPage() set a limit, and must be called after it
        All()
    ```

* Conditions (`Node`s, combine with `query.And(...)`/`query.Or(...)`, pass to `Where`/`AndWhere`/`Or`/`Having`):

    | Function                    | SQL                | Note |
    | ---------------------------- | ------------------ | ---- |
    | `Eq(field, value)`           | `field = ?`         | |
    | `Gt(field, value)`           | `field > ?`         | |
    | `Lt(field, value)`           | `field < ?`         | |
    | `Gte(field, value)`          | `field >= ?`        | |
    | `Lte(field, value)`          | `field <= ?`        | |
    | `Like(field, value)`         | `field LIKE ?`      | no automatic `%` wrapping - include it in `value` yourself |
    | `In(field, value)`           | `field IN (?)`      | `value` is a slice, or a `*SubQuery` (see below) |
    | `IsNull(field)`               | `field IS NULL`     | no `value` argument |
    | `NotNull(field)`              | `field IS NOT NULL` | no `value` argument |
    | `Exists(sub *SubQuery)`      | `EXISTS (...)`      | `sub` wraps a `*gorm.DB` |
    | `And(nodes...)`/`Or(nodes...)` | `(n1 AND/OR n2 ...)` | groups any number of nested conditions |

* Besides `With`/`Where`/`Count`/`PerPage`/`Page`/`All` (above), `IQueryBuilder[T]` also has:
    * `AndWhere(n)`/`Or(n)` - add another condition to whatever `Where` already set, without rebuilding the tree
      yourself.
    * `Distinct()`, `GroupBy(fields...)`, `Having(n)`.
    * `OrderBy(field, desc)`, or the shorthands `OrderAsc(field)`/`OrderDesc(field)`.
    * `With(relation, scope...)` also accepts an optional GORM scope function to filter/modify the preloaded
      relation itself (`With("Role", func(db *gorm.DB) *gorm.DB { return db.Where(...) })`).


### <a name="link3">Remain:</a>

* `BaseModel` uses `ID uint64` instead of `gorm.Model`'s `ID uint` - this isn't cosmetic: every `BaseRepo[T]` method
  that takes an ID (`ReadByID`, `UpdateByID`, `DeleteByID`, ...) is typed `uint64`, so keeping the model's own `ID`
  the same type avoids casting between `uint`/`uint64` at every call site. `BaseModel` otherwise matches
  `gorm.Model` (`CreatedAt`/`UpdatedAt`/`DeletedAt`, soft-delete via `gorm.DeletedAt`).


## License

Apache License 2.0 — see [LICENSE](./LICENSE).
