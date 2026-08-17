package model

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ErrTableNotFound is returned by IntrospectModelSchema when the named
// table doesn't exist in the database - a distinct, recognizable condition
// (the model just hasn't been created yet), not a query failure.
var ErrTableNotFound = errors.New("table not found")

// pgTypeToFieldType maps a Postgres column's information_schema.columns
// data_type to the FieldType it represents. Only the types the package's
// own migrations are expected to produce are listed - an unrecognized
// data_type is a hard error (see IntrospectModelSchema).
//
// "timestamp without time zone" is deliberately absent: FieldTypeDateTime
// always carries an explicit offset (see field.go), so only "timestamp
// with time zone" columns round-trip through it.
var pgTypeToFieldType = map[string]FieldType{
	"character varying":        FieldTypeString,
	"character":                FieldTypeString,
	"text":                     FieldTypeString,
	"integer":                  FieldTypeInt,
	"bigint":                   FieldTypeInt,
	"smallint":                 FieldTypeInt,
	"double precision":         FieldTypeFloat,
	"real":                     FieldTypeFloat,
	"numeric":                  FieldTypeDecimal,
	"boolean":                  FieldTypeBool,
	"date":                     FieldTypeDate,
	"time without time zone":   FieldTypeTime,
	"timestamp with time zone": FieldTypeDateTime,
	"interval":                 FieldTypeInterval,
	"jsonb":                    FieldTypeDict,
	"json":                     FieldTypeDict,
}

// IntrospectModelSchema builds a ModelSchema from the live structure of
// tableName in pgSchema (the Postgres schema the table itself lives in,
// unrelated to the ModelSchema being built) - the reverse of what a schema
// file declares (see LoadModelSchema). The primary key column ("id") is not
// represented in the result, the same way it's absent from a schema file
// (see ModelSchema.Fields) - only composite/non-"id" primary keys would
// need explicit handling, and models don't have those.
//
// withTimestamps additionally excludes created_at/updated_at/deleted_at
// (see isTimestampColumn) the same implicit way - pass the model's own
// EffectiveTimestamps() here. Declaring a Field under one of these three
// names is only forbidden when the model's own Timestamps is on (see
// ModelManager.LoadModelSchemas) - with it off, there's no such
// restriction, so passing false here lets a table that happens to have
// one of them (an ordinary field the model itself declares, or a column
// left over from Timestamps having been on before) come back as a
// regular Field instead - CompareFields then reconciles it the same way
// it would any other column a schema file no longer declares.
//
// Returns ErrTableNotFound if tableName doesn't exist in pgSchema.
//
// A column whose declared type was recorded via SetColumnType (the
// physical Postgres column type alone can't always tell it apart from
// something else - a dict stored in a text/character varying column looks
// the same to information_schema as a plain string one) uses the recorded
// type instead of the usual physical-type mapping - see columnOverride.
//
// Relations are restored from three sources, each read via
// systemRelationsTable (a bare Postgres FK constraint carries no
// information about which declared Relation it implements - see
// RelationFk): a column of tableName itself that's a foreign key (this
// table is the relation's RelationTypeManyToOne/FK-holding-
// RelationTypeOneToOne side - such a column is represented only as a
// Relation, excluded from Fields entirely, the same way "id" is);
// a foreign key on another table that references tableName (the
// RelationTypeOneToMany/non-holding-RelationTypeOneToOne side - skipped
// entirely for a "uni" relation, which by definition has nothing recorded
// for this side to restore); and RelationTypeManyToMany, which has no FK
// column on tableName's own table at all (its physical footprint is a
// separate join table) - found instead by querying
// systemRelationsTable directly for rows recorded from tableName's own
// side.
func IntrospectModelSchema(db *sql.DB, tableName, pgSchema string, withTimestamps bool) (*ModelSchema, error) {
	overrides, err := loadColumnOverrides(db, tableName, pgSchema)
	if err != nil {
		return nil, err
	}
	outgoingFKs, err := loadOutgoingForeignKeys(db, tableName, pgSchema)
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(`
		SELECT column_name, data_type, character_maximum_length,
		       numeric_precision, numeric_scale, is_nullable, column_default
		FROM information_schema.columns
		WHERE table_schema = $1 AND table_name = $2
		ORDER BY ordinal_position
	`, pgSchema, tableName)
	if err != nil {
		return nil, fmt.Errorf("querying columns of %q: %w", tableName, err)
	}
	defer rows.Close()

	result := &ModelSchema{Name: tableName}
	found := false
	for rows.Next() {
		found = true

		var (
			columnName    string
			dataType      string
			maxLength     sql.NullInt64
			precision     sql.NullInt64
			scale         sql.NullInt64
			isNullable    string
			columnDefault sql.NullString
		)
		if err := rows.Scan(&columnName, &dataType, &maxLength, &precision, &scale, &isNullable, &columnDefault); err != nil {
			return nil, fmt.Errorf("reading column of %q: %w", tableName, err)
		}
		if columnName == "id" || (withTimestamps && isTimestampColumn(columnName)) {
			continue
		}

		if fkName, ok := outgoingFKs[columnName]; ok {
			rel, err := outgoingRelation(db, fkName, tableName, columnName, pgSchema)
			if err != nil {
				return nil, fmt.Errorf("column %q of table %q: %w", columnName, tableName, err)
			}
			result.Relations = append(result.Relations, rel)
			continue
		}

		var override *columnOverride
		if o, ok := overrides[columnName]; ok {
			override = &o
		}

		field, err := columnToField(dataType, maxLength, precision, scale, isNullable, columnDefault, override)
		if err != nil {
			return nil, fmt.Errorf("column %q of table %q: %w", columnName, tableName, err)
		}
		result.Fields = append(result.Fields, NamedField{Name: columnName, Field: field})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading columns of %q: %w", tableName, err)
	}
	if !found {
		return nil, ErrTableNotFound
	}

	incoming, err := incomingRelations(db, tableName, pgSchema)
	if err != nil {
		return nil, fmt.Errorf("table %q: %w", tableName, err)
	}
	result.Relations = append(result.Relations, incoming...)

	manyToMany, err := manyToManyRelations(db, tableName, pgSchema)
	if err != nil {
		return nil, fmt.Errorf("table %q: %w", tableName, err)
	}
	result.Relations = append(result.Relations, manyToMany...)

	return result, nil
}

// isTimestampColumn reports whether columnName is one of the three columns
// execCreateTable/execAddTimestamps add when Timestamps is enabled -
// created_at/updated_at/deleted_at.
func isTimestampColumn(columnName string) bool {
	switch columnName {
	case "created_at", "updated_at", "deleted_at":
		return true
	default:
		return false
	}
}

// timestampColumnSpec describes one of created_at/updated_at/deleted_at's
// expected physical shape - required is true for created_at/updated_at
// (NOT NULL), false for deleted_at (nullable).
type timestampColumnSpec struct {
	name     string
	required bool
}

// timestampColumnSpecs is created_at/updated_at/deleted_at, in the same
// order execCreateTable/execAddTimestamps physically add them.
var timestampColumnSpecs = []timestampColumnSpec{
	{name: "created_at", required: true},
	{name: "updated_at", required: true},
	{name: "deleted_at", required: false},
}

// timestampColumnInfo is one existing physical column's actual shape, as
// read back by existingTimestampColumns - just enough to decide
// compatibility (see checkTimestampColumnCompatible), not a full Field.
type timestampColumnInfo struct {
	dataType string
	nullable bool
}

// existingTimestampColumns reads tableName's own actual created_at/
// updated_at/deleted_at columns (whichever of the three physically exist)
// in pgSchema, keyed by name - a model can reach Timestamps being on with
// some of these three already present as ordinary columns (declared by
// hand while Timestamps was off - see "Timestamps
// (created_at/updated_at/deleted_at)"), so execAddTimestamps needs to know
// exactly which of the three still need adding, and CompareModel needs
// the same information to decide whether an AddTimestamps action is
// needed at all (see missingTimestampColumns).
func existingTimestampColumns(db sqlExecutor, tableName, pgSchema string) (map[string]timestampColumnInfo, error) {
	rows, err := db.Query(`
		SELECT column_name, data_type, is_nullable
		FROM information_schema.columns
		WHERE table_schema = $1 AND table_name = $2
			AND column_name IN ('created_at', 'updated_at', 'deleted_at')
	`, pgSchema, tableName)
	if err != nil {
		return nil, fmt.Errorf("querying timestamp columns of %q: %w", tableName, err)
	}
	defer rows.Close()

	result := map[string]timestampColumnInfo{}
	for rows.Next() {
		var name, dataType, isNullable string
		if err := rows.Scan(&name, &dataType, &isNullable); err != nil {
			return nil, fmt.Errorf("reading timestamp columns of %q: %w", tableName, err)
		}
		result[name] = timestampColumnInfo{dataType: dataType, nullable: isNullable == "YES"}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading timestamp columns of %q: %w", tableName, err)
	}
	return result, nil
}

// checkTimestampColumnCompatible reports an error if info (an existing
// physical column already named created_at/updated_at/deleted_at) can't
// be adopted as-is for spec - a physical type other than "timestamp with
// time zone", or nullability not matching spec.required.
// execAddTimestamps adopts a compatible existing column completely
// unchanged (never recreates it, so no data is lost turning Timestamps on
// for a model that already had one of these as an ordinary field) - an
// incompatible one is left for the caller to resolve by hand (e.g. a
// hand-written type: query migration to change its type first), the same
// way this package never auto-synchronizes any other column type
// mismatch on its own.
func checkTimestampColumnCompatible(spec timestampColumnSpec, info timestampColumnInfo) error {
	if info.dataType != "timestamp with time zone" {
		return fmt.Errorf("existing column %q has type %q, want \"timestamp with time zone\"", spec.name, info.dataType)
	}
	wantNullable := !spec.required
	if info.nullable != wantNullable {
		if spec.required {
			return fmt.Errorf("existing column %q is nullable, want NOT NULL", spec.name)
		}
		return fmt.Errorf("existing column %q is NOT NULL, want nullable", spec.name)
	}
	return nil
}

// missingTimestampColumns reports which of created_at/updated_at/
// deleted_at tableName doesn't already have, in pgSchema - exactly the
// set an AddTimestamps action needs to physically add (see
// ModelDiff.AddTimestamps), baked into the action once at generation
// time rather than re-derived when it's later applied.
// Anything already there is left for execAddTimestamps to adopt
// unchanged, not re-added. Returns an error if an EXISTING column under
// one of these names isn't compatible with what Timestamps expects (see
// checkTimestampColumnCompatible) - surfaced here, at generation time,
// rather than only once the resulting migration is applied.
func missingTimestampColumns(db sqlExecutor, tableName, pgSchema string) ([]string, error) {
	existing, err := existingTimestampColumns(db, tableName, pgSchema)
	if err != nil {
		return nil, err
	}

	var missing []string
	for _, spec := range timestampColumnSpecs {
		info, ok := existing[spec.name]
		if !ok {
			missing = append(missing, spec.name)
			continue
		}
		if err := checkTimestampColumnCompatible(spec, info); err != nil {
			return nil, fmt.Errorf("table %q: %w", tableName, err)
		}
	}
	return missing, nil
}

// timestampsIndexExists reports whether tableName's deleted_at column is
// already covered by an index, by column rather than by name - reuses
// columnHasIndex directly (see its own doc: a plain single-column index,
// a multi-column index starting with it, or a UNIQUE constraint's own
// backing index all count), not just an index under this package's own
// computed name (timestampsIndexName). CompareModel needs this the same
// way it needs missingTimestampColumns: an AddTimestamps action must not
// claim credit for (and shouldn't create a redundant duplicate of) an
// index that was already there before Timestamps was ever turned on for
// this model, adopted the same way an existing compatible column is (see
// missingTimestampColumns's doc) - regardless of whether that
// pre-existing index happens to share this package's own naming
// convention or not. execDelTimestamps's own DROP INDEX still only ever
// targets the exact name this package computes (see its doc) - an
// adopted index under a different name is left alone on rollback too,
// the same as any column this package doesn't own.
func timestampsIndexExists(db *sql.DB, tableName, pgSchema string) (bool, error) {
	return columnHasIndex(db, tableName, "deleted_at", pgSchema)
}

// loadOutgoingForeignKeys maps each of tableName's own foreign-key
// columns (in pgSchema) to that foreign key's own constraint name.
func loadOutgoingForeignKeys(db *sql.DB, tableName, pgSchema string) (map[string]string, error) {
	rows, err := db.Query(`
		SELECT kcu.column_name, tc.constraint_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
			ON tc.constraint_name = kcu.constraint_name AND tc.constraint_schema = kcu.constraint_schema
		WHERE tc.constraint_type = 'FOREIGN KEY' AND tc.table_schema = $1 AND tc.table_name = $2
	`, pgSchema, tableName)
	if err != nil {
		return nil, fmt.Errorf("querying foreign keys of %q: %w", tableName, err)
	}
	defer rows.Close()

	result := map[string]string{}
	for rows.Next() {
		var columnName, constraintName string
		if err := rows.Scan(&columnName, &constraintName); err != nil {
			return nil, fmt.Errorf("reading foreign keys of %q: %w", tableName, err)
		}
		result[columnName] = constraintName
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading foreign keys of %q: %w", tableName, err)
	}
	return result, nil
}

// outgoingRelation restores the Relation an outgoing foreign key (a
// column on this table itself) implements - this side always physically
// holds the FK column, so it's always the RelationTypeOneToOne FkHolder
// when the recorded type is RelationTypeOneToOne (RelationTypeManyToOne
// is unconditionally the holder regardless of FkHolder, see Relation's
// doc, so nothing needs checking for it here).
//
// NoIndex is restored from physical reality (columnHasIndex).
// RelationTypeOneToOne's FK column is always left NoIndex=false without
// even checking - its UNIQUE constraint always carries its own backing
// index, so the answer can never be anything else (see Relation.NoIndex's
// doc), and this is also what validation guarantees a valid schema file
// declares for it.
func outgoingRelation(db *sql.DB, fkName, tableName, columnName, pgSchema string) (NamedRelation, error) {
	fk, ok, err := loadRelationFk(db, fkName)
	if err != nil {
		return NamedRelation{}, err
	}
	if !ok {
		return NamedRelation{}, fmt.Errorf("foreign key %q has no recorded meaning in %s", fkName, systemRelationsTable)
	}

	var noIndex bool
	if fk.Type != RelationTypeOneToOne {
		indexed, err := columnHasIndex(db, tableName, columnName, pgSchema)
		if err != nil {
			return NamedRelation{}, err
		}
		noIndex = !indexed
	}

	return NamedRelation{
		Name: fk.HomeAttribute,
		Relation: Relation{
			Type:             fk.Type,
			RelatedModel:     fk.RelatedModel,
			RelatedAttribute: fk.RelatedAttribute,
			FkHolder:         fk.Type == RelationTypeOneToOne,
			NoIndex:          noIndex,
		},
	}, nil
}

// columnHasIndex reports whether table.column is covered by an index
// whose first key column is exactly this one - a plain single-column
// index, a multi-column index starting with it, or a UNIQUE constraint's
// own backing index all count.
func columnHasIndex(db *sql.DB, table, column, pgSchema string) (bool, error) {
	var exists bool
	err := db.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM pg_index idx
			JOIN pg_class tbl ON tbl.oid = idx.indrelid
			JOIN pg_namespace ns ON ns.oid = tbl.relnamespace
			JOIN pg_attribute att ON att.attrelid = tbl.oid AND att.attnum = idx.indkey[0]
			WHERE ns.nspname = $3 AND tbl.relname = $1 AND att.attname = $2
		)
	`, table, column, pgSchema).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("checking index on %s.%s: %w", table, column, err)
	}
	return exists, nil
}

// incomingRelations restores the relations implemented by foreign keys on
// OTHER tables that reference tableName - the RelationTypeOneToMany/
// non-holding-RelationTypeOneToOne side. A "uni" relation (RelatedAttribute
// == "" on the recorded FK's home side) is skipped - by definition nothing
// was ever declared for this side to restore.
//
// The discovery query goes through pg_constraint directly rather than
// information_schema.table_constraints/constraint_column_usage - those two
// views can only be joined by constraint name (constraint names are only
// unique per table, not database-wide), so a same-named but unrelated
// constraint elsewhere (even a non-foreign-key one, since
// constraint_column_usage also lists a UNIQUE/PRIMARY KEY/CHECK
// constraint's own columns under the same name) could otherwise get
// matched into the result instead of the real foreign key.
// pg_constraint's confrelid resolves a specific constraint's referenced
// table directly from that same row, with no name-based join at all.
func incomingRelations(db *sql.DB, tableName, pgSchema string) ([]NamedRelation, error) {
	rows, err := db.Query(`
		SELECT con.conname
		FROM pg_constraint con
		JOIN pg_class refcl ON refcl.oid = con.confrelid
		JOIN pg_namespace refns ON refns.oid = refcl.relnamespace
		WHERE con.contype = 'f' AND refns.nspname = $2 AND refcl.relname = $1
	`, tableName, pgSchema)
	if err != nil {
		return nil, fmt.Errorf("querying incoming foreign keys: %w", err)
	}
	defer rows.Close()

	var fkNames []string
	for rows.Next() {
		var constraintName string
		if err := rows.Scan(&constraintName); err != nil {
			return nil, fmt.Errorf("reading incoming foreign keys: %w", err)
		}
		fkNames = append(fkNames, constraintName)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading incoming foreign keys: %w", err)
	}

	var relations []NamedRelation
	for _, fkName := range fkNames {
		fk, ok, err := loadRelationFk(db, fkName)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, fmt.Errorf("foreign key %q has no recorded meaning in %s", fkName, systemRelationsTable)
		}
		if fk.Type == RelationTypeManyToMany {
			// A join table's own two FK columns each point at one of the
			// two participating tables - each is picked up by this same
			// incoming-FK scan when run against either one, but neither
			// column belongs to that table's own Fields/Relations the way
			// a direct oneToMany/oneToOne-passive FK does. manyToMany is
			// restored exclusively via manyToManyRelations instead.
			continue
		}
		if fk.RelatedAttribute == "" {
			continue
		}
		relations = append(relations, NamedRelation{
			Name: fk.RelatedAttribute,
			Relation: Relation{
				Type:             contrRelationType(fk.Type),
				RelatedModel:     fk.HomeModel,
				RelatedAttribute: fk.HomeAttribute,
			},
		})
	}
	return relations, nil
}

// manyToManyRelations restores the RelationTypeManyToMany relations
// tableName participates in - these have no FK column on tableName's own
// table at all (the join table holds both FK columns instead), so they're
// found by querying systemRelationsTable directly (see
// loadManyToManyRelationFks) rather than via information_schema. NoIndex
// is restored per relation from the physical join-table column's own
// index (see manyToManyJoinColumn/columnHasIndex) - each side's index is
// independent (see Relation.NoIndex's doc), so this is checked separately
// for every relation, not just once for the whole join table.
func manyToManyRelations(db *sql.DB, tableName, pgSchema string) ([]NamedRelation, error) {
	fks, err := loadManyToManyRelationFks(db, tableName, pgSchema)
	if err != nil {
		return nil, err
	}

	relations := make([]NamedRelation, 0, len(fks))
	for _, fk := range fks {
		noIndex, err := manyToManyNoIndex(db, tableName, fk.HomeAttribute, pgSchema)
		if err != nil {
			return nil, err
		}
		relations = append(relations, NamedRelation{
			Name: fk.HomeAttribute,
			Relation: Relation{
				Type:             RelationTypeManyToMany,
				RelatedModel:     fk.RelatedModel,
				RelatedAttribute: fk.RelatedAttribute,
				NoIndex:          noIndex,
			},
		})
	}
	return relations, nil
}

// manyToManyNoIndex reports whether the many-to-many relation recorded
// under (homeTable, homeAttribute) is missing an index on its own
// physical join-table column - false (indexed) if the join column can't
// be resolved at all.
func manyToManyNoIndex(db *sql.DB, homeTable, homeAttribute, pgSchema string) (bool, error) {
	table, column, ok, err := manyToManyJoinColumn(db, homeTable, homeAttribute, pgSchema)
	if err != nil || !ok {
		return false, err
	}
	// The join table isn't recorded under its own schema separately - it
	// lives in the same schema as homeTable (see RelationFk's doc: there's
	// no per-side "join table schema" concept, only the two participating
	// models' own schemas).
	indexed, err := columnHasIndex(db, table, column, pgSchema)
	if err != nil {
		return false, err
	}
	return !indexed, nil
}

// manyToManyJoinColumn resolves the physical join-table name and column
// the many-to-many relation recorded under (homeTable, homeAttribute, in
// pgSchema) lives on - the metadata row itself (RelationFk/
// systemRelationsTable) doesn't carry the join table's name (see
// RelationFk's doc), so this goes through pg_constraint by the row's own
// fk_name instead, scoped to confrelid = homeTable the same way
// loadManyToManyRelationFks's own physical-existence check is, so an
// orphaned row (fk_name no longer naming a real constraint referencing
// homeTable) resolves to ok=false.
func manyToManyJoinColumn(db *sql.DB, homeTable, homeAttribute, pgSchema string) (table, column string, ok bool, err error) {
	err = db.QueryRow(`
		SELECT owncl.relname, att.attname
		FROM `+systemRelationsTable+` r
		JOIN pg_constraint con ON con.conname = r.fk_name AND con.contype = 'f'
		JOIN pg_class refcl ON refcl.oid = con.confrelid
		JOIN pg_namespace refns ON refns.oid = refcl.relnamespace
		JOIN pg_class owncl ON owncl.oid = con.conrelid
		JOIN pg_attribute att ON att.attrelid = con.conrelid AND att.attnum = con.conkey[1]
		WHERE r.home_table = $1 AND r.home_attribute = $2 AND r.type = $3
			AND refns.nspname = $4 AND refcl.relname = $5
	`, pgQualifiedTable(pgSchema, homeTable), homeAttribute, string(RelationTypeManyToMany), pgSchema, homeTable).Scan(&table, &column)
	if err == sql.ErrNoRows {
		return "", "", false, nil
	}
	if err != nil {
		return "", "", false, fmt.Errorf("resolving join column for %s.%s: %w", homeTable, homeAttribute, err)
	}
	return table, column, true, nil
}

// columnToField determines one column's Field - from override if the
// column has a recorded declared type (see loadColumnOverrides), otherwise
// from its physical Postgres type via pgTypeToFieldType.
func columnToField(
	dataType string,
	maxLength, precision, scale sql.NullInt64,
	isNullable string,
	columnDefault sql.NullString,
	override *columnOverride,
) (Field, error) {
	var fieldType FieldType
	var size, prec, scl int

	if override != nil {
		fieldType = override.Type
		if !knownFieldTypes[fieldType] {
			return Field{}, fmt.Errorf("recorded declared type %q is not a known field type", fieldType)
		}
		size, prec, scl = override.Size, override.Precision, override.Scale
	} else {
		var ok bool
		fieldType, ok = pgTypeToFieldType[dataType]
		if !ok {
			return Field{}, fmt.Errorf("unsupported column type %q", dataType)
		}
		if fieldType == FieldTypeString && maxLength.Valid {
			size = int(maxLength.Int64)
		}
		if fieldType == FieldTypeDecimal {
			if precision.Valid {
				prec = int(precision.Int64)
			}
			if scale.Valid {
				scl = int(scale.Int64)
			}
		}
	}

	f := Field{
		Type:      fieldType,
		Required:  isNullable == "NO",
		Size:      size,
		Precision: prec,
		Scale:     scl,
	}

	if !columnDefault.Valid {
		return f, nil
	}
	def, err := pgDefaultToField(fieldType, columnDefault.String, f.Size, f.Precision, f.Scale)
	if err != nil {
		return Field{}, fmt.Errorf("default %q: %w", columnDefault.String, err)
	}
	f.Default = def
	return f, nil
}

// pgDefaultToField converts a Postgres information_schema.columns.
// column_default expression into fieldType's typed Default. Returns nil,
// nil for a sequence default (nextval(...), on a non-"id" column) or an
// exact "now()" (the one function-call default this package's own DDL
// produces, for created_at/updated_at when Timestamps is on - see
// execCreateTable/execAddTimestamps) - the field just has no
// representable static default. Any OTHER function-call expression
// (upper('x'), gen_random_uuid(), round(19.99), etc.) is a hard error
// rather than silently accepted or dropped - there's no way to tell
// whether it's equivalent to whatever the field would otherwise declare,
// and a schema/database diff needs to know that rather than silently
// missing it.
func pgDefaultToField(fieldType FieldType, raw string, size, precision, scale int) (any, error) {
	if strings.HasPrefix(raw, "nextval(") || strings.TrimSpace(raw) == "now()" {
		return nil, nil
	}

	literal, quoted, ok := pgDefaultLiteral(raw)
	if !ok {
		return nil, fmt.Errorf("unsupported default expression %q", raw)
	}

	var typedRaw any
	switch fieldType {
	case FieldTypeString, FieldTypeDecimal, FieldTypeDate, FieldTypeTime:
		typedRaw = literal
	case FieldTypeInt:
		n, err := strconv.ParseInt(literal, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("not an integer: %w", err)
		}
		typedRaw = n
	case FieldTypeFloat:
		n, err := strconv.ParseFloat(literal, 64)
		if err != nil {
			return nil, fmt.Errorf("not a number: %w", err)
		}
		typedRaw = n
	case FieldTypeBool:
		b, err := strconv.ParseBool(literal)
		if err != nil {
			return nil, fmt.Errorf("not a bool: %w", err)
		}
		typedRaw = b
	case FieldTypeDateTime:
		s, err := normalizePgTimestamptz(literal)
		if err != nil {
			return nil, err
		}
		typedRaw = s
	case FieldTypeInterval:
		d, err := parsePgInterval(literal)
		if err != nil {
			return nil, err
		}
		typedRaw = d.String()
	case FieldTypeDict:
		if !quoted {
			return nil, fmt.Errorf("unsupported default expression %q", raw)
		}
		var v any
		if err := json.Unmarshal([]byte(literal), &v); err != nil {
			return nil, fmt.Errorf("not valid JSON: %w", err)
		}
		typedRaw = v
	default:
		return nil, fmt.Errorf("unsupported default expression %q", raw)
	}

	return parseDefault(fieldType, typedRaw, size, precision, scale)
}

// pgBareLiteralRe matches an unquoted default's remainder once a trailing
// "::type" cast is stripped - a bare number/bool token, never containing
// the parentheses, quotes or whitespace a function call or other
// expression (nextval(...) is handled separately, upper('x'), now()) would
// leave behind.
var pgBareLiteralRe = regexp.MustCompile(`^[^()'"\s,;]+$`)

// pgDefaultLiteral strips a trailing "::type" cast (if any) from a Postgres
// default expression and, if what's left is a quoted string, unquotes it -
// a literal quote inside is written doubled (two consecutive single
// quotes), the same convention this package's own compact field syntax
// uses (see field_compact.go). Fails (ok = false) for anything that isn't
// a plain quoted string or a bare literal token - a database-side function
// call or other expression, quoted or not, has no static value to extract.
func pgDefaultLiteral(raw string) (literal string, quoted bool, ok bool) {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, "'") {
		if idx := strings.Index(raw, "::"); idx >= 0 {
			raw = raw[:idx]
		}
		if raw == "" || !pgBareLiteralRe.MatchString(raw) {
			return "", false, false
		}
		return raw, false, true
	}

	i := 1
	for i < len(raw) {
		if raw[i] == '\'' {
			if i+1 < len(raw) && raw[i+1] == '\'' {
				i += 2
				continue
			}
			inner := raw[1:i]
			inner = strings.ReplaceAll(inner, "''", "'")
			return inner, true, true
		}
		i++
	}
	return "", false, false
}

// normalizePgTimestamptz turns a Postgres "timestamp with time zone"
// default literal (space-separated date/time, an offset with no minutes
// when it's a whole number of hours, e.g. "2026-01-02 15:04:05+00") into
// an RFC3339 string parseDefault accepts.
func normalizePgTimestamptz(s string) (string, error) {
	s = strings.Replace(s, " ", "T", 1)
	if pgShortOffsetRe.MatchString(s) {
		s += ":00"
	} else if strings.HasSuffix(s, "Z") {
		// already fine
	} else if !pgFullOffsetRe.MatchString(s) {
		return "", fmt.Errorf("unrecognized timestamptz literal %q", s)
	}
	return s, nil
}

var (
	pgShortOffsetRe = regexp.MustCompile(`[+-]\d{2}$`)
	pgFullOffsetRe  = regexp.MustCompile(`[+-]\d{2}:\d{2}$`)
)

// pgIntervalTimeRe matches interval's "HH:MM:SS[.fraction]" component,
// optionally signed.
var pgIntervalTimeRe = regexp.MustCompile(`^([+-]?)(\d+):(\d{2}):(\d{2}(?:\.\d+)?)$`)

// parsePgInterval converts a Postgres "postgres"-style interval default
// literal (e.g. "1 day", "01:30:00", "1 day 01:30:00", "-1 days
// +01:00:00") into a time.Duration. Year/month components are rejected -
// Go's Duration can't represent a calendar month (its length varies),
// unlike the day/hour/minute/second components handled here.
func parsePgInterval(s string) (time.Duration, error) {
	fields := strings.Fields(s)
	var total time.Duration
	i := 0
	for i < len(fields) {
		f := fields[i]
		if m := pgIntervalTimeRe.FindStringSubmatch(f); m != nil {
			d, err := parsePgIntervalTime(m)
			if err != nil {
				return 0, err
			}
			total += d
			i++
			continue
		}

		if i+1 >= len(fields) {
			return 0, fmt.Errorf("can't parse interval component %q", f)
		}
		n, err := strconv.Atoi(f)
		if err != nil {
			return 0, fmt.Errorf("can't parse interval component %q: %w", f, err)
		}
		unit := strings.TrimSuffix(fields[i+1], "s")
		switch unit {
		case "day":
			total += time.Duration(n) * 24 * time.Hour
		case "year", "mon":
			return 0, fmt.Errorf("interval default has a %s component, not representable as a fixed-length duration", unit)
		default:
			return 0, fmt.Errorf("unrecognized interval unit %q", fields[i+1])
		}
		i += 2
	}
	return total, nil
}

func parsePgIntervalTime(m []string) (time.Duration, error) {
	sign, hours, minutes, seconds := m[1], m[2], m[3], m[4]

	h, err := strconv.Atoi(hours)
	if err != nil {
		return 0, fmt.Errorf("invalid interval hours %q: %w", hours, err)
	}
	min, err := strconv.Atoi(minutes)
	if err != nil {
		return 0, fmt.Errorf("invalid interval minutes %q: %w", minutes, err)
	}
	sec, err := strconv.ParseFloat(seconds, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid interval seconds %q: %w", seconds, err)
	}

	d := time.Duration(h)*time.Hour + time.Duration(min)*time.Minute + time.Duration(sec*float64(time.Second))
	if sign == "-" {
		d = -d
	}
	return d, nil
}
