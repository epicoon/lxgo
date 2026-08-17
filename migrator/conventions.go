// Package migrator manages DB schema migrations and data seeds for
// lxgo/kernel applications - migrations are YAML files with `Up`/`Down` SQL,
// tracked in a dedicated table; see NewCommand for the ready-made console
// command wrapping Create/Show/Check/Up/Down/UpSeeds. Other migration file
// types (generated rather than hand-written SQL) can plug in via
// RegisterMigrationType/CreateWithContent.
package migrator

import "database/sql"

// Config configures the package-level migrator state - see Init.
type Config struct {
	// DB is the database connection migrations/seeds run against.
	DB *sql.DB
	// MigrationsPath is the directory migration YAML files are read from/written to.
	MigrationsPath string
	// SeedsPath is the directory seed YAML files are read from.
	SeedsPath string
}

// FMigrationApply runs one direction (up or down) of a custom-typed
// migration file against tx - see RegisterMigrationType. raw is the
// migration file's entire, unparsed content; migrator does not interpret it
// beyond reading the top-level "Type" field to pick the handler.
type FMigrationApply func(tx *sql.Tx, raw []byte) error
