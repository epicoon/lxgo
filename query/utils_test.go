package query

import "testing"

type category struct {
	ID   uint64
	Name string
}

// TestTableName_IrregularPlural is a regression test: the package's own
// naive pluralization (name + "s") used to turn "Category" into
// "categorys" instead of gorm's real "categories" - any model with an
// irregular English plural would get a table name that doesn't exist.
func TestTableName_IrregularPlural(t *testing.T) {
	if got := tableName[category](); got != "categories" {
		t.Fatalf("tableName[category]() = %q, want 'categories' (gorm's own pluralization)", got)
	}
}

type user struct {
	ID   uint64
	Name string
}

func TestTableName_RegularPlural(t *testing.T) {
	if got := tableName[user](); got != "users" {
		t.Fatalf("tableName[user]() = %q, want 'users'", got)
	}
}
