# Package for manage migrations

> Actual version: `v0.1.0-alpha.5`. Details [here](https://github.com/epicoon/lxgo/tree/master/migrator/CHANGE_LOG.md)

> You can use it if your application is based on [lxgo/kernel](https://github.com/epicoon/lxgo/tree/master/kernel)

1. Make sure you have a DB connection set up. An example of DB connection configuration in file `path/to/app/config.yaml`:
```yaml
Database:
  Host: "localhost"
  Port: 5432
  User: "me"
  Password: "111111"
  DBName: "my_db"
  SSLMode: "disable"
```

2. Plug in console command, create command wrapper:
    > How to set up console commands in your application you can find [here](https://github.com/epicoon/lxgo/tree/master/cmd)

```go
package main

import (
	"fmt"

	"github.com/epicoon/lxgo/cmd"
	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/app"
	"github.com/epicoon/lxgo/kernel/config"
	"github.com/epicoon/lxgo/migrator"
)

func NewMigratorCommand(_ ...cmd.ICommandOptions) cmd.ICommand {
    // Load configuration from `path/to/app/config.yaml`
	conf, err := config.Load("config.yaml")
	if err != nil {
		fmt.Printf("Can not read application config. Cause: %q", err)
		return nil
	}

    // It is assumed that the DB-connection is configured by key "Database"
	if !config.HasParam(conf, "Database") {
		fmt.Println("The application config doesn't contain Database configuration")
		return nil
	}
	dbConf, err := config.GetParam[kernel.Config](conf, "Database")
	if err != nil {
		fmt.Printf("can not read Database config: %s", err)
		return nil
	}

    // Create connection
    connection := app.NewConnection()
	connection.SetConfig(&dbConf)
	err = connection.Connect()
	if err != nil {
		fmt.Printf("Can not connect to DB. Cause: %q", err)
		return nil
	}

    // Init migrator and return original command:
	migrator.Init(migrator.Config{
		DB: connection.DB(),

		// Path to directory with migrations
		MigrationsPath: "runtime/migrations",

		// If you use seeds - path to directory with seed files
		SeedsPath: "runtime/seeds",
	})
	return migrator.NewCommand()
}
```

3. Set command call in your `main.go` file:
```go
package main

import (
	"github.com/epicoon/lxgo/cmd"
)

func main() {
	cmd.Init(cmd.CommandsList{
		"":         NewMainCommand,
		"migrator": NewMigratorCommand,
	})
	cmd.Run()
}
```

4. Now you can use commands:
    - `go run . migrator:check`
    - `go run . migrator:show`
    - `go run . migrator:show --count=2`
    - `go run . migrator:create --name="my_migration"` (no `.yaml` — it's appended automatically, along with a timestamp prefix)
    - `go run . migrator:up`
    - `go run . migrator:down`
    - `go run . migrator:down --count=2`
	- `go run . migrator:up-seeds`

5. An example of migration (`migrator:create` generates a skeleton with this
   shape, named `<timestamp>_<name>.yaml`):
```yaml
name: create_tables
type: query

up:
  - CREATE TABLE
    my_table (
      id SERIAL PRIMARY KEY,
      name VARCHAR(255) NOT NULL
    )
  - CREATE INDEX my_table_name_idx ON my_table (name);

down:
  - DROP INDEX IF EXISTS my_table_name_idx;
  - DROP TABLE IF EXISTS my_table;
```
   `up`/`down` accept either a single SQL string or (as above) a list of
   statements, run in order inside one transaction. `name`/`type` aren't
   actually read by the migrator (only the filename matters for tracking
   which migrations ran) — `type: query` is currently just a placeholder the
   generated skeleton includes, reserved for possibly distinguishing other
   kinds of migrations later.

   Only PostgreSQL is supported for now (migrations run through
   `database/sql` with the `lib/pq` driver, same as [`lxgo/kernel`'s DB
   connection](https://github.com/epicoon/lxgo/tree/master/kernel#db)).

6. An example of seeds:
```yaml
# filename - table_name.yaml
- id: 1
  field: "value for 1"
- id: 2
  field: "value for 2"
```


## License

Apache License 2.0 — see [LICENSE](./LICENSE).
