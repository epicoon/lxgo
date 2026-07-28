package query

import (
	"reflect"

	"gorm.io/gorm/schema"
)

// namingStrategy is gorm's own default naming strategy - tableName and
// compiler.column/query_builder.go's JOIN-building call it directly
// instead of hand-rolling snake_case/pluralization, so raw SQL fragments
// built here (QueryBuilder's table alias, compiler.column's column
// references) can never diverge from the table/column names gorm itself
// uses for BaseRepo's Model(new(T))-based queries and migrations.
var namingStrategy = schema.NamingStrategy{}

func tableName[T any]() string {
	t := reflect.TypeOf((*T)(nil)).Elem()
	return namingStrategy.TableName(t.Name())
}
