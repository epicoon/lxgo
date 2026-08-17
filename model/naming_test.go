package model

import "testing"

func TestPgTableName(t *testing.T) {
	cases := map[string]string{
		"Widget":        "widgets",
		"WidgetCopy":    "widget_copies",
		"apply_widgets": "apply_widgets",
	}
	for in, want := range cases {
		if got := pgTableName(in); got != want {
			t.Errorf("pgTableName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPgColumnName(t *testing.T) {
	cases := map[string]string{
		"NameCopy":  "name_copy",
		"GameType":  "game_type",
		"old_count": "old_count",
	}
	for in, want := range cases {
		if got := pgColumnName(in); got != want {
			t.Errorf("pgColumnName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPgRelationColumnName(t *testing.T) {
	cases := map[string]string{
		"oneSide": "one_side_id",
		"owner":   "owner_id",
	}
	for in, want := range cases {
		if got := pgRelationColumnName(in); got != want {
			t.Errorf("pgRelationColumnName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPgRelationFkName(t *testing.T) {
	got := pgRelationFkName("widgets", "owner_id")
	if want := "fk__widgets__owner_id"; got != want {
		t.Errorf("pgRelationFkName(...) = %q, want %q", got, want)
	}
}

// TestPgRelationFkName_NoSingleUnderscoreCollision demonstrates why the
// separator has to be a double underscore, not one - two different
// (table, column) pairs that would collide under a single-underscore
// scheme must stay distinguishable here.
func TestPgRelationFkName_NoSingleUnderscoreCollision(t *testing.T) {
	a := pgRelationFkName("foo_bar", "baz")
	b := pgRelationFkName("foo", "bar_baz")
	if a == b {
		t.Fatalf("pgRelationFkName(%q, %q) and pgRelationFkName(%q, %q) both produced %q", "foo_bar", "baz", "foo", "bar_baz", a)
	}
}

func TestPgManyToManyTableName_SortsByModelName(t *testing.T) {
	forward := pgManyToManyTableName("Tag", "words", "Word", "tags")
	backward := pgManyToManyTableName("Word", "tags", "Tag", "words")
	if forward != backward {
		t.Fatalf("pgManyToManyTableName is not order-independent: %q != %q", forward, backward)
	}
	if want := "rel__tags__words__words__tags"; forward != want {
		t.Fatalf("pgManyToManyTableName(...) = %q, want %q", forward, want)
	}
}

// TestPgManyToManyTableName_SelfReferentialSortsByAttribute checks the
// same order-independence TestPgManyToManyTableName_SortsByModelName
// checks for two distinct models, but for a self-referential relation
// (both sides declared on the same model) - there the model-name sort
// alone can't tell the two sides apart (they're equal), so the attribute
// names need their own tiebreak for either side to compute the same
// physical join table name as the other.
func TestPgManyToManyTableName_SelfReferentialSortsByAttribute(t *testing.T) {
	forward := pgManyToManyTableName("User", "friends", "User", "friendOf")
	backward := pgManyToManyTableName("User", "friendOf", "User", "friends")
	if forward != backward {
		t.Fatalf("pgManyToManyTableName is not order-independent for a self-referential relation: %q != %q", forward, backward)
	}
}

func TestPgManyToManyTableName_DistinctAttributesDontCollide(t *testing.T) {
	friends := pgManyToManyTableName("User", "friends", "User", "friendOf")
	blocked := pgManyToManyTableName("User", "blocked", "User", "blockedBy")
	if friends == blocked {
		t.Fatalf("two distinct many-to-many relations between the same models produced the same table name: %q", friends)
	}
}

func TestPgTruncateIdent_ShortNameUnchanged(t *testing.T) {
	if got := pgTruncateIdent("short_name"); got != "short_name" {
		t.Errorf("pgTruncateIdent(short) = %q, want unchanged", got)
	}
}

func TestPgTruncateIdent_LongNameShortenedWithinLimit(t *testing.T) {
	long := "a_very_long_identifier_that_is_definitely_over_the_sixty_three_byte_postgres_limit"
	got := pgTruncateIdent(long)
	if len(got) > pgMaxIdentifierLength {
		t.Fatalf("pgTruncateIdent(...) length = %d, want <= %d", len(got), pgMaxIdentifierLength)
	}
	if got == long[:pgMaxIdentifierLength] {
		t.Fatal("expected a hash suffix, not a bare truncation")
	}
}

// TestPgTruncateIdent_DivergingTailsDontCollide is exactly the scenario
// that broke a many-to-many relation's two FK constraint names in
// practice: pgRelationFkName's inputs differ only in a trailing,
// model-specific column name once the shared join-table-name prefix
// exceeds the Postgres identifier limit - a bare truncation would cut
// that difference off entirely and collide.
func TestPgTruncateIdent_DivergingTailsDontCollide(t *testing.T) {
	prefix := "fk__rel__some_quite_long_table_name__with_a_long_attribute_name__another_quite_long_table_name__"
	a := pgTruncateIdent(prefix + "column_a")
	b := pgTruncateIdent(prefix + "column_b")
	if a == b {
		t.Fatalf("pgTruncateIdent produced the same result for two different long names: %q", a)
	}
	if len(a) > pgMaxIdentifierLength || len(b) > pgMaxIdentifierLength {
		t.Fatalf("a = %d bytes, b = %d bytes, want both <= %d", len(a), len(b), pgMaxIdentifierLength)
	}
}

func TestPgRelationIndexName(t *testing.T) {
	got := pgRelationIndexName("widgets", "owner_id")
	if want := "idx__widgets__owner_id"; got != want {
		t.Errorf("pgRelationIndexName(...) = %q, want %q", got, want)
	}
}

func TestPgJoinColumnName(t *testing.T) {
	if got := pgJoinColumnName("Widget", "tags"); got != "widgets_tags_id" {
		t.Errorf("pgJoinColumnName(%q, %q) = %q, want %q", "Widget", "tags", got, "widgets_tags_id")
	}
}

// TestPgJoinColumnName_SelfReferentialDoesNotCollide is the exact scenario
// that broke a self-referential many-to-many relation's CREATE TABLE in
// practice: both of a join table's columns reference the same model
// ("User" linked to itself via "friends"/"friendOf"), so the column name
// can't be derived from the referenced model alone - it has to fold in the
// side's own attribute too, or both columns end up with the same name.
func TestPgJoinColumnName_SelfReferentialDoesNotCollide(t *testing.T) {
	own := pgJoinColumnName("User", "friends")
	related := pgJoinColumnName("User", "friendOf")
	if own == related {
		t.Fatalf("pgJoinColumnName produced the same column name for both sides of a self-referential relation: %q", own)
	}
}
