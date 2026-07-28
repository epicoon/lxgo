package query

import (
	"reflect"
	"testing"
)

func TestCondition_Compile_Operators(t *testing.T) {
	cases := []struct {
		name     string
		node     Node
		wantSQL  string
		wantArgs []any
	}{
		{"Eq", Eq("Name", "bob"), "t.name = ?", []any{"bob"}},
		{"Gt", Gt("Age", 18), "t.age > ?", []any{18}},
		{"Lt", Lt("Age", 18), "t.age < ?", []any{18}},
		{"Gte", Gte("Age", 18), "t.age >= ?", []any{18}},
		{"Lte", Lte("Age", 18), "t.age <= ?", []any{18}},
		{"Like", Like("Name", "%bob%"), "t.name LIKE ?", []any{"%bob%"}},
		{"IsNull", IsNull("DeletedAt"), "t.deleted_at IS NULL", nil},
		{"NotNull", NotNull("DeletedAt"), "t.deleted_at IS NOT NULL", nil},
		{"In_Slice", In("ID", []uint64{1, 2, 3}), "t.id IN ?", []any{[]uint64{1, 2, 3}}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := newCompiler("t")
			sql, args := tc.node.compile(c)
			if sql != tc.wantSQL {
				t.Fatalf("sql = %q, want %q", sql, tc.wantSQL)
			}
			if !reflect.DeepEqual(args, tc.wantArgs) {
				t.Fatalf("args = %#v, want %#v", args, tc.wantArgs)
			}
		})
	}
}

func TestCondition_Compile_InSubQuery(t *testing.T) {
	sub := &SubQuery{}
	c := newCompiler("t")
	sql, args := In("ID", sub).compile(c)
	if sql != "t.id IN (?)" {
		t.Fatalf("sql = %q, want 't.id IN (?)'", sql)
	}
	if len(args) != 1 || args[0] != sub.Query {
		t.Fatalf("args = %#v, want the subquery's own Query", args)
	}
}

func TestCondition_Compile_Exists(t *testing.T) {
	sub := &SubQuery{}
	c := newCompiler("t")
	sql, args := Exists(sub).compile(c)
	if sql != "EXISTS (?)" {
		t.Fatalf("sql = %q, want 'EXISTS (?)'", sql)
	}
	if len(args) != 1 || args[0] != sub.Query {
		t.Fatalf("args = %#v, want the subquery's own Query", args)
	}
}

func TestCondition_Compile_JSONBPath(t *testing.T) {
	c := newCompiler("t")
	sql, _ := Eq("AuthData->role", "admin").compile(c)
	if sql != "AuthData->role = ?" {
		t.Fatalf("sql = %q, want the JSONB path left untouched (no table alias/snake_case)", sql)
	}
}
