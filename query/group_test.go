package query

import (
	"reflect"
	"testing"
)

func TestGroup_Compile_And(t *testing.T) {
	c := newCompiler("t")
	sql, args := And(Eq("A", 1), Eq("B", 2)).compile(c)
	if sql != "(t.a = ? AND t.b = ?)" {
		t.Fatalf("sql = %q", sql)
	}
	if !reflect.DeepEqual(args, []any{1, 2}) {
		t.Fatalf("args = %#v", args)
	}
}

func TestGroup_Compile_Or(t *testing.T) {
	c := newCompiler("t")
	sql, _ := Or(Eq("A", 1), Eq("B", 2)).compile(c)
	if sql != "(t.a = ? OR t.b = ?)" {
		t.Fatalf("sql = %q", sql)
	}
}

func TestGroup_Compile_Nested(t *testing.T) {
	c := newCompiler("t")
	sql, args := And(Eq("A", 1), Or(Eq("B", 2), Eq("C", 3))).compile(c)
	if sql != "(t.a = ? AND (t.b = ? OR t.c = ?))" {
		t.Fatalf("sql = %q", sql)
	}
	if !reflect.DeepEqual(args, []any{1, 2, 3}) {
		t.Fatalf("args = %#v", args)
	}
}

// TestGroup_Compile_SingleNode documents (not a bug) that And/Or with a
// single node still wraps it in parens - harmless, just not collapsed.
func TestGroup_Compile_SingleNode(t *testing.T) {
	c := newCompiler("t")
	sql, _ := And(Eq("A", 1)).compile(c)
	if sql != "(t.a = ?)" {
		t.Fatalf("sql = %q", sql)
	}
}
