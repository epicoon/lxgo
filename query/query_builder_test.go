package query

import "testing"

func TestQueryBuilder_Where_ReplacesPrevious(t *testing.T) {
	qb := &QueryBuilder[user]{alias: "t"}
	qb.Where(Eq("A", 1))
	qb.Where(Eq("B", 2))

	cond, ok := qb.root.(Node)
	if !ok || cond == nil {
		t.Fatalf("root = %#v, want a Node", qb.root)
	}
	c := newCompiler("t")
	sql, _ := qb.root.compile(c)
	if sql != "t.b = ?" {
		t.Fatalf("sql = %q, want the second Where to have replaced the first", sql)
	}
}

func TestQueryBuilder_AndWhere_FirstCallJustSets(t *testing.T) {
	qb := &QueryBuilder[user]{alias: "t"}
	qb.AndWhere(Eq("A", 1))

	c := newCompiler("t")
	sql, _ := qb.root.compile(c)
	if sql != "t.a = ?" {
		t.Fatalf("sql = %q, want a plain condition (no AND wrapper) on the first call", sql)
	}
}

func TestQueryBuilder_AndWhere_AccumulatesWithAnd(t *testing.T) {
	qb := &QueryBuilder[user]{alias: "t"}
	qb.Where(Eq("A", 1))
	qb.AndWhere(Eq("B", 2))

	if _, ok := qb.root.(*Group); !ok {
		t.Fatalf("root = %#v (%T), want a *Group after a second AndWhere", qb.root, qb.root)
	}
	c := newCompiler("t")
	sql, _ := qb.root.compile(c)
	if sql != "(t.a = ? AND t.b = ?)" {
		t.Fatalf("sql = %q", sql)
	}
}

func TestQueryBuilder_Or_AccumulatesWithOr(t *testing.T) {
	qb := &QueryBuilder[user]{alias: "t"}
	qb.Where(Eq("A", 1))
	qb.Or(Eq("B", 2))

	c := newCompiler("t")
	sql, _ := qb.root.compile(c)
	if sql != "(t.a = ? OR t.b = ?)" {
		t.Fatalf("sql = %q", sql)
	}
}

func TestQueryBuilder_Page_NoEffectWithoutPerPage(t *testing.T) {
	qb := &QueryBuilder[user]{alias: "t"}
	qb.Page(3)
	if qb.offset != 0 {
		t.Fatalf("offset = %d, want 0 - Page should have no effect before PerPage", qb.offset)
	}
}

func TestQueryBuilder_Page_NoEffectAtPageOne(t *testing.T) {
	qb := &QueryBuilder[user]{alias: "t"}
	qb.PerPage(10)
	qb.Page(1)
	if qb.offset != 0 {
		t.Fatalf("offset = %d, want 0 at page 1", qb.offset)
	}
}

func TestQueryBuilder_Page_ComputesOffset(t *testing.T) {
	qb := &QueryBuilder[user]{alias: "t"}
	qb.PerPage(10)
	qb.Page(3)
	if qb.offset != 20 {
		t.Fatalf("offset = %d, want 20 (page 3, size 10)", qb.offset)
	}
}

// TestQueryBuilder_GroupBy_ResolvesColumnNames is a regression test:
// GroupBy's fields used to be joined into the SQL as raw strings, unlike
// OrderBy/Eq which resolve field names through compiler.column - so
// .GroupBy("UserID") produced a literal "UserID" instead of "user_id".
func TestQueryBuilder_GroupBy_ResolvesColumnNames(t *testing.T) {
	qb := &QueryBuilder[user]{alias: "t"}
	qb.GroupBy("UserID", "Name")

	got := qb.groupByClause(newCompiler(qb.alias))
	if got != "t.user_id,t.name" {
		t.Fatalf("groupByClause() = %q, want 't.user_id,t.name'", got)
	}
}

func TestQueryBuilder_GroupByClause_EmptyWhenUnset(t *testing.T) {
	qb := &QueryBuilder[user]{alias: "t"}
	if got := qb.groupByClause(newCompiler(qb.alias)); got != "" {
		t.Fatalf("groupByClause() = %q, want empty", got)
	}
}
