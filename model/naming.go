package model

import "gorm.io/gorm/schema"

// pgNaming is the same zero-configuration gorm.io/gorm/schema.NamingStrategy
// lxgo-query uses (see its own namingStrategy) - table/column names this
// package creates physically coincide with what GORM derives from a Go
// struct of the same name, so a table can be queried through either one
// interchangeably.
var pgNaming = schema.NamingStrategy{}

// pgTableName returns name's physical Postgres table name (snake_case,
// pluralized) - computed on every call, never cached on a ModelSchema, so
// it can never drift from the name it was derived from.
func pgTableName(name string) string {
	return pgNaming.TableName(name)
}

// pgColumnName returns name's physical Postgres column name (snake_case) -
// computed on every call, same reasoning as pgTableName. The table
// parameter NamingStrategy.ColumnName accepts is unused by the zero-value
// strategy itself (see gorm's own source), so it's not threaded through
// here either.
func pgColumnName(name string) string {
	return pgNaming.ColumnName("", name)
}
