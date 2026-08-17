package model

import "testing"

// TestSetColumnType_RejectsInvalidShape checks that SetColumnType validates
// its Field argument before touching the database at all (a nil *sql.DB
// would panic on any actual query, so reaching that point without a panic
// here would itself be a failure).
func TestSetColumnType_RejectsInvalidShape(t *testing.T) {
	err := SetColumnType(nil, "widgets", "sort", Field{Type: FieldTypeInt, Size: 10})
	if err == nil {
		t.Fatal("expected an error for size on a non-string type")
	}
}

func TestSetColumnType_RejectsUnknownType(t *testing.T) {
	err := SetColumnType(nil, "widgets", "sort", Field{Type: FieldType("bogus")})
	if err == nil {
		t.Fatal("expected an error for an unknown field type")
	}
}
