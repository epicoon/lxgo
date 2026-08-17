package model

import (
	"database/sql"
	"fmt"
)

// systemRelationsTableName is the service table's bare name (no schema
// qualifier) - used where information_schema.tables.table_name is
// compared directly (that column never includes the schema).
// systemRelationsTable is the schema-qualified form used everywhere the
// table is actually read from or written to - a bare FK constraint in
// Postgres carries no information about which declared Relation it
// implements (its type, which model/attribute is on the other side), so
// this table is what IntrospectModelSchema reads to restore that meaning.
// One row per physical FK: RelationTypeOneToOne/RelationTypeManyToOne get
// one (the FK column lives on the "home" side only); RelationTypeManyToMany
// gets two, one per join-table column, each written from that column's own
// side's perspective (see SetRelationFk).
const systemRelationsTableName = "model_relations"
const systemRelationsTable = lxSysSchema + "." + systemRelationsTableName

// RelationFk is one physical foreign key's recorded meaning - see
// SetRelationFk/DeleteRelationFk. HomeModel/HomeAttribute/RelatedModel/
// RelatedAttribute are logical (schema-file) names; HomeTable/RelatedTable
// are the physical Postgres tables they correspond to, schema-qualified
// ("schema.table", see pgQualifiedTable) - recorded separately because a
// RelationTypeManyToMany lookup needs to find "every FK belonging to
// physical table X" without first knowing X's logical model name (see
// loadManyToManyRelationFks). JoinTable is set only for
// RelationTypeManyToMany - the join table's own physical name, also
// schema-qualified, fixed once at creation time (see pgManyToManyTableName)
// and never recomputed afterward: unlike every other physical identifier
// here, it's built from both sides' attribute names, so a fresh computation
// from the current attribute names would point at the wrong table (or none
// at all) after a rename on either side (metadata-only, see
// RenameRelationAction's doc) - recording it once and always reading it
// back instead avoids that entirely. A caller that needs to quote it as a
// Postgres identifier (pgIdent) must split it back to its bare table name
// first (see splitQualifiedTable) - pgIdent has no notion of a two-part
// schema-qualified identifier.
type RelationFk struct {
	Type             RelationType
	HomeTable        string
	HomeModel        string
	HomeAttribute    string
	RelatedTable     string
	RelatedModel     string
	RelatedAttribute string
	JoinTable        string
}

// ensureSystemRelationsTable creates systemRelationsTable (and
// lxSysSchema, if needed) if it doesn't exist yet - called before every
// write.
func ensureSystemRelationsTable(db sqlExecutor) error {
	if err := ensureLxSysSchema(db); err != nil {
		return err
	}

	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS ` + systemRelationsTable + ` (
			fk_name           text NOT NULL PRIMARY KEY,
			type              text NOT NULL,
			home_table        text NOT NULL,
			home_model        text NOT NULL,
			home_attribute    text NOT NULL,
			related_table     text NOT NULL,
			related_model     text NOT NULL,
			related_attribute text NOT NULL DEFAULT '',
			join_table        text NOT NULL DEFAULT ''
		)
	`)
	if err != nil {
		return fmt.Errorf("creating %s: %w", systemRelationsTable, err)
	}
	return nil
}

// SetRelationFk records fkName's meaning - overwriting any previous record
// under the same name. Call this whenever a foreign key is created, in the
// same transaction/step as the DDL that does it.
func SetRelationFk(db sqlExecutor, fkName string, fk RelationFk) error {
	if err := ensureSystemRelationsTable(db); err != nil {
		return err
	}

	_, err := db.Exec(`
		INSERT INTO `+systemRelationsTable+`
			(fk_name, type, home_table, home_model, home_attribute, related_table, related_model, related_attribute, join_table)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (fk_name) DO UPDATE
		SET type = EXCLUDED.type, home_table = EXCLUDED.home_table, home_model = EXCLUDED.home_model,
		    home_attribute = EXCLUDED.home_attribute, related_table = EXCLUDED.related_table,
		    related_model = EXCLUDED.related_model, related_attribute = EXCLUDED.related_attribute,
		    join_table = EXCLUDED.join_table
	`, fkName, string(fk.Type), fk.HomeTable, fk.HomeModel, fk.HomeAttribute,
		fk.RelatedTable, fk.RelatedModel, fk.RelatedAttribute, fk.JoinTable)
	if err != nil {
		return fmt.Errorf("recording foreign key %q: %w", fkName, err)
	}
	return nil
}

// DeleteRelationFk removes fkName's recorded meaning, if any - call this
// when the physical foreign key itself is dropped. Not an error if
// nothing was recorded for it.
func DeleteRelationFk(db sqlExecutor, fkName string) error {
	if err := ensureSystemRelationsTable(db); err != nil {
		return err
	}

	_, err := db.Exec(`DELETE FROM `+systemRelationsTable+` WHERE fk_name = $1`, fkName)
	if err != nil {
		return fmt.Errorf("deleting recorded foreign key %q: %w", fkName, err)
	}
	return nil
}

// systemRelationsTableExists reports whether systemRelationsTable has
// already been created - loadRelationFk/loadIncomingRelationFks/
// loadManyToManyRelationFks all need this check first, the same reasoning
// as loadColumnOverrides: nothing has ever been recorded yet is not an
// error, just an empty result.
func systemRelationsTableExists(db sqlExecutor) (bool, error) {
	var exists bool
	err := db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = $1 AND table_name = $2
		)
	`, lxSysSchema, systemRelationsTableName).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("checking %s: %w", systemRelationsTable, err)
	}
	return exists, nil
}

// loadRelationFk reads fkName's recorded meaning - ok is false if nothing
// was recorded for it (including if systemRelationsTable doesn't exist
// yet at all).
func loadRelationFk(db sqlExecutor, fkName string) (fk RelationFk, ok bool, err error) {
	exists, err := systemRelationsTableExists(db)
	if err != nil || !exists {
		return RelationFk{}, false, err
	}

	var relType string
	err = db.QueryRow(`
		SELECT type, home_table, home_model, home_attribute, related_table, related_model, related_attribute, join_table
		FROM `+systemRelationsTable+` WHERE fk_name = $1
	`, fkName).Scan(&relType, &fk.HomeTable, &fk.HomeModel, &fk.HomeAttribute,
		&fk.RelatedTable, &fk.RelatedModel, &fk.RelatedAttribute, &fk.JoinTable)
	if err == sql.ErrNoRows {
		return RelationFk{}, false, nil
	}
	if err != nil {
		return RelationFk{}, false, fmt.Errorf("reading foreign key %q: %w", fkName, err)
	}
	fk.Type = RelationType(relType)
	return fk, true, nil
}

// loadRelationFkByHomeAttribute finds the recorded fk_name/RelationFk for
// the relation homeModel declares under homeAttribute - the reliable way
// to locate an existing RelationTypeManyToMany relation's own metadata row
// (and, via RelationFk.JoinTable, its physical join table) without
// recomputing anything from the relation's current attribute names, which
// a prior rename on either side could have made stale (see RelationFk's
// doc). Works the same for the other relation types too, just not needed
// there - their own fk_name is always cheaply recomputable instead (see
// pgRelationFkName).
func loadRelationFkByHomeAttribute(db sqlExecutor, homeModel, homeAttribute string) (fkName string, fk RelationFk, ok bool, err error) {
	exists, err := systemRelationsTableExists(db)
	if err != nil || !exists {
		return "", RelationFk{}, false, err
	}

	var relType string
	err = db.QueryRow(`
		SELECT fk_name, type, home_table, home_model, home_attribute, related_table, related_model, related_attribute, join_table
		FROM `+systemRelationsTable+` WHERE home_model = $1 AND home_attribute = $2
	`, homeModel, homeAttribute).Scan(&fkName, &relType, &fk.HomeTable, &fk.HomeModel, &fk.HomeAttribute,
		&fk.RelatedTable, &fk.RelatedModel, &fk.RelatedAttribute, &fk.JoinTable)
	if err == sql.ErrNoRows {
		return "", RelationFk{}, false, nil
	}
	if err != nil {
		return "", RelationFk{}, false, fmt.Errorf("reading foreign key for %q.%q: %w", homeModel, homeAttribute, err)
	}
	fk.Type = RelationType(relType)
	return fkName, fk, true, nil
}

// loadManyToManyRelationFks reads every RelationTypeManyToMany row
// recorded from physTable's own perspective (see RelationFk's doc - a
// many-to-many relation always has a row written from each side's own
// perspective, so this alone is enough to find every such relation
// physTable participates in, without needing to know physTable's logical
// model name or the join table's name in advance). A row whose fk_name no
// longer names a foreign key constraint that still references physTable
// (the join table itself, or just that one FK column, was dropped outside
// this package, leaving the row orphaned) is silently excluded rather than
// trusted as-is - unlike RelationTypeOneToOne/RelationTypeManyToOne, this
// lookup starts from the recorded row rather than from a physical
// information_schema scan, so it's the one place that needs to double-
// check the row still corresponds to something real before returning it.
// Checking fk_name's existence alone, without confirming it still
// references physTable specifically, wouldn't be enough - a Postgres
// constraint name is only unique per table, not database-wide, so an
// unrelated FK elsewhere in the schema could coincidentally reuse the same
// name after the real one was dropped. The check itself goes through
// pg_constraint directly rather than information_schema.table_constraints/
// constraint_column_usage - those two views can only be joined by
// constraint name, and a same-named but unrelated constraint elsewhere
// (even a non-foreign-key one - constraint_column_usage also lists a
// UNIQUE/PRIMARY KEY/CHECK constraint's own columns under the same name)
// could otherwise get matched in instead of the real foreign key.
// pg_constraint's confrelid resolves a specific constraint's referenced
// table directly from that same row, with no name-based join at all.
func loadManyToManyRelationFks(db sqlExecutor, physTable, pgSchema string) ([]RelationFk, error) {
	exists, err := systemRelationsTableExists(db)
	if err != nil || !exists {
		return nil, err
	}

	rows, err := db.Query(`
		SELECT r.home_table, r.home_model, r.home_attribute, r.related_table, r.related_model, r.related_attribute, r.join_table
		FROM `+systemRelationsTable+` r
		WHERE r.home_table = $1 AND r.type = $2
			AND EXISTS (
				SELECT 1
				FROM pg_constraint con
				JOIN pg_class refcl ON refcl.oid = con.confrelid
				JOIN pg_namespace refns ON refns.oid = refcl.relnamespace
				WHERE con.contype = 'f' AND con.conname = r.fk_name
					AND refns.nspname = $3 AND refcl.relname = $4
			)
	`, pgQualifiedTable(pgSchema, physTable), string(RelationTypeManyToMany), pgSchema, physTable)
	if err != nil {
		return nil, fmt.Errorf("querying %s: %w", systemRelationsTable, err)
	}
	defer rows.Close()

	var fks []RelationFk
	for rows.Next() {
		fk := RelationFk{Type: RelationTypeManyToMany}
		if err := rows.Scan(&fk.HomeTable, &fk.HomeModel, &fk.HomeAttribute,
			&fk.RelatedTable, &fk.RelatedModel, &fk.RelatedAttribute, &fk.JoinTable); err != nil {
			return nil, fmt.Errorf("reading %s: %w", systemRelationsTable, err)
		}
		fks = append(fks, fk)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading %s: %w", systemRelationsTable, err)
	}
	return fks, nil
}
