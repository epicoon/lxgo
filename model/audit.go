package model

import (
	"database/sql"
	"fmt"
)

// OrphanedRelationFk is a systemRelationsTable row whose fk_name no
// longer names a real Postgres foreign key matching what the row records
// - see AuditRelationFks.
type OrphanedRelationFk struct {
	FkName string
	RelationFk
}

// AuditRelationFks scans every row recorded in the relations service
// table and returns the ones that no longer correspond to a real foreign
// key - a constraint dropped outside this package's own migration path
// (raw SQL, a manual schema change) leaves its row behind, and ordinary
// use (IntrospectModelSchema) never surfaces it since it's simply never
// looked up again (see RelationFk's doc) - this is the only way to notice
// such a row exists at all. Not itself a correctness problem, but a
// growing count of orphaned rows is a sign something is changing the
// database schema outside this package's control.
func AuditRelationFks(db *sql.DB) ([]OrphanedRelationFk, error) {
	exists, err := systemRelationsTableExists(db)
	if err != nil || !exists {
		return nil, err
	}

	rows, err := db.Query(`
		SELECT fk_name, type, home_table, home_model, home_attribute, related_table, related_model, related_attribute
		FROM ` + systemRelationsTable)
	if err != nil {
		return nil, fmt.Errorf("querying %s: %w", systemRelationsTable, err)
	}
	defer rows.Close()

	var recorded []OrphanedRelationFk
	for rows.Next() {
		var o OrphanedRelationFk
		var relType string
		if err := rows.Scan(&o.FkName, &relType, &o.HomeTable, &o.HomeModel, &o.HomeAttribute,
			&o.RelatedTable, &o.RelatedModel, &o.RelatedAttribute); err != nil {
			return nil, fmt.Errorf("reading %s: %w", systemRelationsTable, err)
		}
		o.Type = RelationType(relType)
		recorded = append(recorded, o)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading %s: %w", systemRelationsTable, err)
	}

	var orphaned []OrphanedRelationFk
	for _, o := range recorded {
		ok, err := relationFkExists(db, o.Type, o.FkName, o.HomeTable)
		if err != nil {
			return nil, err
		}
		if !ok {
			orphaned = append(orphaned, o)
		}
	}
	return orphaned, nil
}

// relationFkExists reports whether fkName still names a real Postgres
// foreign key constraint matching how relType records it - see
// RelationFk's doc: RelationTypeOneToOne/RelationTypeManyToOne record
// homeTable as the constraint's own table (the FK column lives there),
// while RelationTypeManyToMany records homeTable as the table the
// constraint references (its own table is an unnamed join table this
// package doesn't otherwise track). homeTable is schema-qualified
// ("schema.table", see pgQualifiedTable) exactly as recorded - split back
// into its parts before matching against information_schema/pg_catalog,
// which only ever expose the bare table name and schema separately.
//
// The RelationTypeManyToMany case goes through pg_constraint directly
// rather than information_schema.table_constraints/constraint_column_usage
// - those two views can only be joined by constraint name (constraint
// names are only unique per table, not database-wide), so a same-named
// but unrelated constraint elsewhere (even a non-foreign-key one, since
// constraint_column_usage also lists a UNIQUE/PRIMARY KEY/CHECK
// constraint's own columns under the same name) could otherwise get
// matched in instead of the real foreign key. pg_constraint's confrelid
// resolves a specific constraint's referenced table directly from that
// same row, with no name-based join at all. The
// RelationTypeOneToOne/RelationTypeManyToOne case doesn't need this - it
// checks a single table_constraints row directly by (name, table), no
// join involved.
func relationFkExists(db *sql.DB, relType RelationType, fkName, homeTable string) (bool, error) {
	pgSchema, table := splitQualifiedTable(homeTable)

	query := `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.table_constraints tc
			WHERE tc.constraint_type = 'FOREIGN KEY' AND tc.table_schema = $3
				AND tc.constraint_name = $1 AND tc.table_name = $2
		)`
	if relType == RelationTypeManyToMany {
		query = `
			SELECT EXISTS (
				SELECT 1
				FROM pg_constraint con
				JOIN pg_class refcl ON refcl.oid = con.confrelid
				JOIN pg_namespace refns ON refns.oid = refcl.relnamespace
				WHERE con.contype = 'f' AND con.conname = $1
					AND refns.nspname = $3 AND refcl.relname = $2
			)`
	}

	var found bool
	if err := db.QueryRow(query, fkName, table, pgSchema).Scan(&found); err != nil {
		return false, fmt.Errorf("checking foreign key %q: %w", fkName, err)
	}
	return found, nil
}

// OrphanedColumnType is a systemTypesTable row whose column no longer
// physically exists - see AuditColumnTypes. TableName is schema-qualified
// ("schema.table", see pgQualifiedTable), exactly as stored.
type OrphanedColumnType struct {
	TableName  string
	ColumnName string
	Type       FieldType
	Size       int
	Precision  int
	Scale      int
}

// AuditColumnTypes scans every row recorded in the types service table
// and returns the ones whose column no longer exists - the same
// reasoning as AuditRelationFks: a column dropped outside this package's
// own migration path leaves its row behind, invisible to ordinary use
// (IntrospectModelSchema only ever consults the table per physical
// column it already found, so an orphaned row for a gone column is
// simply never matched).
func AuditColumnTypes(db *sql.DB) ([]OrphanedColumnType, error) {
	var exists bool
	err := db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = $1 AND table_name = $2
		)
	`, lxSysSchema, systemTypesTableName).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("checking %s: %w", systemTypesTable, err)
	}
	if !exists {
		return nil, nil
	}

	// t.table_name is schema-qualified ("schema.table", see
	// pgQualifiedTable) - split_part pulls the two parts back apart for the
	// comparison against information_schema.columns, which only ever
	// exposes them separately.
	rows, err := db.Query(`
		SELECT t.table_name, t.column_name, t.type, t.size, t.precision, t.scale
		FROM ` + systemTypesTable + ` t
		WHERE NOT EXISTS (
			SELECT 1 FROM information_schema.columns c
			WHERE c.table_schema = split_part(t.table_name, '.', 1)
				AND c.table_name = split_part(t.table_name, '.', 2)
				AND c.column_name = t.column_name
		)
	`)
	if err != nil {
		return nil, fmt.Errorf("querying %s: %w", systemTypesTable, err)
	}
	defer rows.Close()

	var orphaned []OrphanedColumnType
	for rows.Next() {
		var o OrphanedColumnType
		var fieldType string
		if err := rows.Scan(&o.TableName, &o.ColumnName, &fieldType, &o.Size, &o.Precision, &o.Scale); err != nil {
			return nil, fmt.Errorf("reading %s: %w", systemTypesTable, err)
		}
		o.Type = FieldType(fieldType)
		orphaned = append(orphaned, o)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading %s: %w", systemTypesTable, err)
	}
	return orphaned, nil
}
