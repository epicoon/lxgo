package model

import (
	"database/sql"
	"fmt"
	"strings"
)

// lxSysSchema is the Postgres schema this package's service tables live in.
const lxSysSchema = "lx_sys"

// pgDefaultSchema is the Postgres schema a model lives in when nothing in
// its namespace cascade overrides it anywhere (see
// ModelSchema.EffectiveNamespace) - Postgres's own default schema, and
// what every model lived in exclusively before schema resolution was wired
// into DDL/introspection.
const pgDefaultSchema = "public"

// pgResolveSchema returns namespace if it's set, else pgDefaultSchema -
// namespace is typically a ModelSchema.EffectiveNamespace()/Action.Namespace
// value, empty meaning "no override anywhere in the cascade", which every
// physical Postgres call site needs turned into an actual schema name
// rather than passing an empty string through.
func pgResolveSchema(namespace string) string {
	if namespace == "" {
		return pgDefaultSchema
	}
	return namespace
}

// pgQualifiedTable joins pgSchema and table into the "schema.table" form
// RelationFk.HomeTable/RelatedTable/JoinTable and system_types.go's own
// table_name column are stored as - an opaque identifier for those service
// tables' own bookkeeping only, never itself a Postgres identifier to quote
// (see pgIdent) - a physical DDL/catalog lookup always takes pgSchema and
// table as separate values instead (see IntrospectModelSchema and its
// internal helpers).
func pgQualifiedTable(pgSchema, table string) string {
	return pgSchema + "." + table
}

// splitQualifiedTable splits a "schema.table" string (as stored via
// pgQualifiedTable) back into its parts - only needed where a stored value
// is the sole source of truth for a physical lookup (auditing a recorded
// row against reality, or feeding a table name recorded before a schema
// wasn't recomputable back into DDL - see RelationFk.JoinTable's doc).
// Ordinary introspection reads never need this: they already have pgSchema
// and table as separate parameters throughout.
func splitQualifiedTable(qualified string) (pgSchema, table string) {
	pgSchema, table, _ = strings.Cut(qualified, ".")
	return pgSchema, table
}

// ensureLxSysSchema creates lxSysSchema if it doesn't exist yet - called
// before creating any service table in it.
func ensureLxSysSchema(db sqlExecutor) error {
	_, err := db.Exec("CREATE SCHEMA IF NOT EXISTS " + lxSysSchema)
	if err != nil {
		return fmt.Errorf("creating schema %s: %w", lxSysSchema, err)
	}
	return nil
}

// systemTypesTableName is the service table's bare name (no schema
// qualifier) - used where information_schema.tables.table_name is
// compared directly (that column never includes the schema).
// systemTypesTable is the schema-qualified form used everywhere the table
// is actually read from or written to - lxgo-model records a column's
// declared field type in it, for the columns where the physical Postgres
// type alone doesn't disambiguate it (see columnOverride/
// loadColumnOverrides) - today that's FieldTypeDict stored in a text/
// character varying column, which otherwise looks exactly like a
// FieldTypeString one (a jsonb/json column is unambiguous on its own -
// see pgTypeToFieldType).
const systemTypesTableName = "model_types"
const systemTypesTable = lxSysSchema + "." + systemTypesTableName

// sqlExecutor is satisfied by both *sql.DB and *sql.Tx - lets
// SetColumnType/DeleteColumnType/loadColumnOverrides run against either a
// plain connection or an in-progress transaction, so a migration's apply/
// invert (which only ever have a *sql.Tx) can keep the system types table
// consistent in the same transaction as the DDL that changes a column.
type sqlExecutor interface {
	Exec(query string, args ...any) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}

// ensureSystemTypesTable creates systemTypesTable (and lxSysSchema, if
// needed) if it doesn't exist yet - called before every write.
func ensureSystemTypesTable(db sqlExecutor) error {
	if err := ensureLxSysSchema(db); err != nil {
		return err
	}

	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS ` + systemTypesTable + ` (
			table_name  text NOT NULL,
			column_name text NOT NULL,
			type        text NOT NULL,
			size        integer NOT NULL DEFAULT 0,
			precision   integer NOT NULL DEFAULT 0,
			scale       integer NOT NULL DEFAULT 0,
			PRIMARY KEY (table_name, column_name)
		)
	`)
	if err != nil {
		return fmt.Errorf("creating %s: %w", systemTypesTable, err)
	}
	return nil
}

// SetColumnType records tableName.columnName's declared field type (f.Type,
// and f.Size/f.Precision/f.Scale where meaningful for that type) -
// overwriting any previous record for the same column. Call this whenever
// a column is created or its type changes, in the same transaction/step as
// the DDL that does it. IntrospectModelSchema prefers a recorded type over
// its own physical-column-type mapping (see columnOverride). tableName is
// stored and matched back exactly as given - pass it already schema-
// qualified ("schema.table", see pgQualifiedTable) so it round-trips
// through loadColumnOverrides, which reads it back qualified the same way.
//
// f.Required/f.Default/f.RenamedFrom aren't recorded here - Required and
// Default are always recoverable from the column itself (NOT NULL,
// column_default), and RenamedFrom is schema-file-only metadata with no DB
// representation at all.
func SetColumnType(db sqlExecutor, tableName, columnName string, f Field) error {
	if err := validateFieldShape(f.Type, f.Size, f.Precision, f.Scale); err != nil {
		return err
	}
	if err := ensureSystemTypesTable(db); err != nil {
		return err
	}

	_, err := db.Exec(`
		INSERT INTO `+systemTypesTable+` (table_name, column_name, type, size, precision, scale)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (table_name, column_name) DO UPDATE
		SET type = EXCLUDED.type, size = EXCLUDED.size, precision = EXCLUDED.precision, scale = EXCLUDED.scale
	`, tableName, columnName, string(f.Type), f.Size, f.Precision, f.Scale)
	if err != nil {
		return fmt.Errorf("recording declared type of %s.%s: %w", tableName, columnName, err)
	}
	return nil
}

// DeleteColumnType removes tableName.columnName's recorded declared type,
// if any - call this when the column itself is dropped. Not an error if
// nothing was recorded for it. tableName must match exactly what SetColumnType
// stored it as (schema-qualified, see its own doc).
func DeleteColumnType(db sqlExecutor, tableName, columnName string) error {
	if err := ensureSystemTypesTable(db); err != nil {
		return err
	}

	_, err := db.Exec(`
		DELETE FROM `+systemTypesTable+` WHERE table_name = $1 AND column_name = $2
	`, tableName, columnName)
	if err != nil {
		return fmt.Errorf("deleting recorded type of %s.%s: %w", tableName, columnName, err)
	}
	return nil
}

// columnOverride is one column's recorded declared type, read from
// systemTypesTable - see loadColumnOverrides.
type columnOverride struct {
	Type      FieldType
	Size      int
	Precision int
	Scale     int
}

// loadColumnOverrides reads every recorded column override for tableName in
// pgSchema, keyed by column name. Returns an empty map, not an error, if
// systemTypesTable doesn't exist yet - nothing has ever been recorded for
// any table.
func loadColumnOverrides(db sqlExecutor, tableName, pgSchema string) (map[string]columnOverride, error) {
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
		return map[string]columnOverride{}, nil
	}

	rows, err := db.Query(`
		SELECT column_name, type, size, precision, scale
		FROM `+systemTypesTable+`
		WHERE table_name = $1
	`, pgQualifiedTable(pgSchema, tableName))
	if err != nil {
		return nil, fmt.Errorf("querying %s: %w", systemTypesTable, err)
	}
	defer rows.Close()

	overrides := map[string]columnOverride{}
	for rows.Next() {
		var columnName, fieldType string
		var size, precision, scale int
		if err := rows.Scan(&columnName, &fieldType, &size, &precision, &scale); err != nil {
			return nil, fmt.Errorf("reading %s: %w", systemTypesTable, err)
		}
		overrides[columnName] = columnOverride{
			Type:      FieldType(fieldType),
			Size:      size,
			Precision: precision,
			Scale:     scale,
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading %s: %w", systemTypesTable, err)
	}
	return overrides, nil
}
