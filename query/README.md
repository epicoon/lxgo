# The package helps to work with DB

> Actual version: `v0.1.0-alpha.2`. [Details](https://github.com/epicoon/lxgo/tree/master/query/CHANGE_LOG.md)

> You can use it if your application is based on [lxgo/kernel](https://github.com/epicoon/lxgo/tree/master/kernel)

## Content:
* [BaseRepo](#link1)
* [QueryBuilder](#link2)
* [Remain](#link3)


### <a name="link1">BaseRepo:</a>

Structure `BaseRepo` implements `IBaseRepo` and `IRepoTx`

* Create repository:
    ```go
    // gormDB *gorm.DB
    // allowedFields []string
    repo := lxModels.NewBaseRepo[modelStruct](gormDB, allowedFields)
    ```

* Inherit repository example:
    ```go
    type UserRepo struct {
        *lxModels.BaseRepo[models.User]
    }

    /** @constructor */
    func NewUserRepo(db *gorm.DB) *UserRepo {
        return &UserRepo{BaseRepo: lxModels.NewBaseRepo[models.User](db, []string{})}
    }
    ```

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

* QueryBuilder example:
    ```go
    // WHERE status = 'inactive' AND email IN ('1@1.1', '2@2.2') AND role.name = 'admin'
    // repo IBaseRepo
    query := repo.QueryBuilder().
        With("Role").
        Where(query.And(
            query.Eq("Status", "inactive"),
            query.In("Email", []string{"1@1.1", "2@2.2"}),
            query.Eq("Role.Name", "admin"),
        ))
    count, _ := query.Count()
    page, _ := query.
        PerPage(10).
        Page(1).
        All()
    ```

* Operators:
    | Operator | SQL         |
    | -------- | ----------- |
    | And      | AND         |
    | Or       | OR          |
    | Like     | LIKE        |
    | IsNull   | IS NULL     |
    | NotNull  | IS NOT NULL |
    | In       | IN          |
    | Exists   | EXISTS      |
    | Eq       | =           |
    | Gt       | >           |
    | Lt       | <           |
    | Gte      | >=          |
    | Lte      | <=          |


### <a name="link3">Remain:</a>

* There is `BaseModel` with `ID uint64` instead of `gorm.Model` with `ID uint`


## License

Apache License 2.0 — see [LICENSE](./LICENSE).
