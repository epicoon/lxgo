package query

import "testing"

func TestCompiler_Column_PlainField(t *testing.T) {
	c := newCompiler("t")
	if got := c.column("Name"); got != "t.name" {
		t.Fatalf("column(Name) = %q, want 't.name'", got)
	}
}

func TestCompiler_Column_ID(t *testing.T) {
	c := newCompiler("t")
	if got := c.column("ID"); got != "t.id" {
		t.Fatalf("column(ID) = %q, want 't.id' (gorm's own naming, not 't.i_d')", got)
	}
}

func TestCompiler_Column_Acronym(t *testing.T) {
	c := newCompiler("t")
	if got := c.column("UserID"); got != "t.user_id" {
		t.Fatalf("column(UserID) = %q, want 't.user_id'", got)
	}
}

func TestCompiler_Column_RelationField(t *testing.T) {
	c := newCompiler("t")
	got := c.column("Role.Name")
	if got != "j1.name" {
		t.Fatalf("column(Role.Name) = %q, want 'j1.name'", got)
	}
	if c.joins["Role"] != "j1" {
		t.Fatalf("expected joins[Role] = j1, got %#v", c.joins)
	}
}

func TestCompiler_Column_RelationField_ReusesAlias(t *testing.T) {
	c := newCompiler("t")
	c.column("Role.Name")
	got := c.column("Role.ID")
	if got != "j1.id" {
		t.Fatalf("column(Role.ID) = %q, want the same alias 'j1.id' reused, not a second join", got)
	}
	if len(c.joins) != 1 {
		t.Fatalf("expected exactly 1 join, got %#v", c.joins)
	}
}

func TestCompiler_Column_MultipleRelations_GetDistinctAliases(t *testing.T) {
	c := newCompiler("t")
	role := c.column("Role.Name")
	dept := c.column("Department.Name")
	if role == dept {
		t.Fatalf("expected distinct relations to get distinct aliases, both got %q", role)
	}
}

func TestCompiler_Column_JSONBPath(t *testing.T) {
	c := newCompiler("t")
	if got := c.column("AuthData->role"); got != "AuthData->role" {
		t.Fatalf("column(JSONB path) = %q, want it left untouched", got)
	}
}
