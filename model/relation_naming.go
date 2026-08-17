package model

import (
	"fmt"
	"hash/fnv"
)

// pgMaxIdentifierLength is Postgres's own maximum identifier length
// (NAMEDATALEN - 1) - a longer identifier is silently truncated to this
// length by Postgres itself when parsed, which could turn two distinct
// computed names into the exact same physical identifier once both are
// cut the same way. pgTruncateIdent guards against that below.
const pgMaxIdentifierLength = 63

// pgTruncateIdent shortens name to fit pgMaxIdentifierLength if it
// doesn't already - by appending a short content hash, so two different
// over-long names that happen to share the same first ~63 characters still
// end up different after truncation too.
func pgTruncateIdent(name string) string {
	if len(name) <= pgMaxIdentifierLength {
		return name
	}
	sum := fnv.New32a()
	sum.Write([]byte(name))
	suffix := fmt.Sprintf("_%08x", sum.Sum32())
	keep := pgMaxIdentifierLength - len(suffix)
	return name[:keep] + suffix
}

// pgRelationColumnName returns attribute's physical foreign-key column
// name for the RelationTypeOneToOne/RelationTypeManyToOne side that holds
// it, e.g. "oneSide" -> "one_side_id" - the same GORM-compatible
// conversion regular fields go through (pgColumnName), applied with an
// "Id" suffix appended to the logical name first.
func pgRelationColumnName(attribute string) string {
	return pgTruncateIdent(pgColumnName(attribute + "Id"))
}

// pgRelationFkName returns the FK constraint name for column on table -
// both already physical names. The double underscore is deliberate:
// table/column names are snake_case and can themselves contain a single
// underscore, so a single-underscore separator could collide between
// different (table, column) pairs - e.g. table "foo_bar" with column "baz"
// and table "foo" with column "bar_baz" would both produce "fk_foo_bar_baz"
// under a single-underscore scheme. A double underscore essentially never
// occurs inside a snake_case component on its own, keeping the two halves
// unambiguous.
func pgRelationFkName(table, column string) string {
	return pgTruncateIdent("fk__" + table + "__" + column)
}

// pgManyToManyTableName returns the physical join table name for the
// many-to-many relation between model1.attr1 and model2.attr2 -
// deterministic regardless of which side calls it: the two models are
// sorted by their logical name first (the same ordering canIgnoreRelation
// uses to decide the "acting" side), so either side computes the exact
// same string. If the relation is within one table, the attribute names
// are sorted.
func pgManyToManyTableName(model1, attr1, model2, attr2 string) string {
	if model1 > model2 || (model1 == model2 && attr1 > attr2) {
		model1, attr1, model2, attr2 = model2, attr2, model1, attr1
	}
	return pgTruncateIdent("rel__" + pgTableName(model1) + "__" + pgColumnName(attr1) + "__" + pgTableName(model2) + "__" + pgColumnName(attr2))
}

// pgRelationIndexName returns the name a relation's own FK column index is
// created under - an explicit name, not Postgres's own auto-generated
// default (which a plain unnamed `CREATE INDEX ON table (column)` would
// otherwise get, `<table>_<column>_idx`), so a later index toggle can
// reliably reconstruct the exact same name to drop it again without
// needing to duplicate Postgres's own default-naming algorithm (which
// also isn't guaranteed collision-free the way pgRelationFkName's double
// underscore already is - see its own doc).
func pgRelationIndexName(table, column string) string {
	return pgTruncateIdent("idx__" + table + "__" + column)
}

// pgJoinColumnName returns a many-to-many join table's column name for the
// side that references model's own table, referenced via its own attribute
// on that side - both the table and the attribute are part of the name,
// not just the table, because a self-referential relation (model equal on
// both sides, e.g. "User" linked to itself via "friends"/"friendOf") would
// otherwise produce the exact same column name for both of the join
// table's two columns.
func pgJoinColumnName(model, attribute string) string {
	return pgTruncateIdent(pgTableName(model) + "_" + pgColumnName(attribute) + "_id")
}
