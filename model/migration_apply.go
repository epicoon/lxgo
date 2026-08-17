package model

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"gopkg.in/yaml.v3"
)

// Apply is the "apply" (up) side of a generated migration - registered
// with lxgo-migrator's migration type registry (see
// migrator.RegisterMigrationType) under MigrationType. Executes every
// action in raw (a migration file written by GenerateMigration) against
// tx, in file order.
func Apply(tx *sql.Tx, raw []byte) error {
	var f migrationFile
	if err := yaml.Unmarshal(raw, &f); err != nil {
		return fmt.Errorf("parsing migration content: %w", err)
	}

	for _, a := range f.Actions {
		if err := applyAction(tx, a); err != nil {
			return fmt.Errorf("action %q on model %q: %w", a.Type, a.ModelName, err)
		}
	}
	return nil
}

// Invert is the "invert" (down) side - runs Action.Inverse() of every
// action in raw against tx, in reverse file order (undoing the last
// action first - the same order any sequence of dependent operations
// unwinds in, e.g. a RenameField followed by a ChangeField on the field's
// new name must have the ChangeField undone before the RenameField is).
func Invert(tx *sql.Tx, raw []byte) error {
	var f migrationFile
	if err := yaml.Unmarshal(raw, &f); err != nil {
		return fmt.Errorf("parsing migration content: %w", err)
	}

	for i := len(f.Actions) - 1; i >= 0; i-- {
		inv, err := f.Actions[i].Inverse()
		if err != nil {
			return fmt.Errorf("inverting action %q on model %q: %w", f.Actions[i].Type, f.Actions[i].ModelName, err)
		}
		if err := applyAction(tx, inv); err != nil {
			return fmt.Errorf("action %q on model %q: %w", inv.Type, inv.ModelName, err)
		}
	}
	return nil
}

func applyAction(tx *sql.Tx, a Action) error {
	pgSchema := pgResolveSchema(a.Namespace)
	switch a.Type {
	case ActionCreateTable:
		return execCreateTable(tx, a.CreateTable, pgSchema, a.Timestamps)
	case ActionDropTable:
		return execDropTable(tx, a.DropTable, pgSchema)
	case ActionAddField:
		return execAddField(tx, a.ModelName, a.AddField, pgSchema)
	case ActionDelField:
		return execDelField(tx, a.ModelName, a.DelField, pgSchema)
	case ActionChangeField:
		return execChangeField(tx, a.ModelName, a.ChangeField, pgSchema)
	case ActionRenameField:
		return execRenameField(tx, a.ModelName, a.RenameField, pgSchema)
	case ActionAddRelation:
		return execAddRelation(tx, a.ModelName, a.AddRelation, pgSchema)
	case ActionDelRelation:
		return execDelRelation(tx, a.ModelName, a.DelRelation, pgSchema)
	case ActionRenameRelation:
		return execRenameRelation(tx, a.ModelName, a.RenameRelation, pgSchema)
	case ActionChangeRelation:
		return execChangeRelation(tx, a.ModelName, a.ChangeRelation, pgSchema)
	case ActionAddTimestamps:
		return execAddTimestamps(tx, a.ModelName, a.AddTimestamps, pgSchema)
	case ActionDelTimestamps:
		return execDelTimestamps(tx, a.ModelName, a.DelTimestamps, pgSchema)
	default:
		return fmt.Errorf("unknown action type %q", a.Type)
	}
}

// ensureSchemaExists creates pgSchema if it doesn't exist yet - called
// before the first CREATE TABLE into it. Skipped for pgDefaultSchema
// ("public") - it always exists already, and this avoids requiring CREATE
// privilege on the database for the common case where no namespace
// override is in play at all.
func ensureSchemaExists(tx *sql.Tx, pgSchema string) error {
	if pgSchema == pgDefaultSchema {
		return nil
	}
	if _, err := tx.Exec("CREATE SCHEMA IF NOT EXISTS " + pgIdent(pgSchema)); err != nil {
		return fmt.Errorf("creating schema %s: %w", pgSchema, err)
	}
	return nil
}

// pgQualifiedIdent quotes pgSchema and name separately and joins them with
// "." - the schema-qualified Postgres identifier form ("schema"."name"),
// used everywhere DDL targets a table/index/etc. that needs to resolve
// regardless of the connection's search_path. Unlike pgQualifiedTable
// (which builds the plain "schema.table" string RelationFk/system_types.go
// store), this always produces a valid quoted SQL identifier.
func pgQualifiedIdent(pgSchema, name string) string {
	return pgIdent(pgSchema) + "." + pgIdent(name)
}

func execCreateTable(tx *sql.Tx, schema *ModelSchema, pgSchema string, withTimestamps bool) error {
	if schema == nil {
		return fmt.Errorf("createTable action has no schema")
	}
	if err := ensureSchemaExists(tx, pgSchema); err != nil {
		return err
	}

	cols := []string{"id serial PRIMARY KEY"}
	for _, f := range schema.Fields {
		def, err := pgColumnDefinition(pgColumnName(f.Name), f.Field)
		if err != nil {
			return fmt.Errorf("field %q: %w", f.Name, err)
		}
		cols = append(cols, def)
	}
	if withTimestamps {
		cols = append(cols, timestampColumnDefs()...)
	}

	physTable := pgTableName(schema.Name)
	query := fmt.Sprintf("CREATE TABLE %s (%s)", pgQualifiedIdent(pgSchema, physTable), strings.Join(cols, ", "))
	if _, err := tx.Exec(query); err != nil {
		return fmt.Errorf("creating table %q: %w", schema.Name, err)
	}

	if withTimestamps {
		if err := createTimestampsIndex(tx, physTable, pgSchema); err != nil {
			return fmt.Errorf("creating table %q: %w", schema.Name, err)
		}
	}
	return nil
}

func execDropTable(tx *sql.Tx, schema *ModelSchema, pgSchema string) error {
	if schema == nil {
		return fmt.Errorf("dropTable action has no schema")
	}

	physTable := pgTableName(schema.Name)
	if _, err := tx.Exec(fmt.Sprintf("DROP TABLE %s", pgQualifiedIdent(pgSchema, physTable))); err != nil {
		return fmt.Errorf("dropping table %q: %w", schema.Name, err)
	}

	if err := deleteAllColumnTypes(tx, pgQualifiedTable(pgSchema, physTable)); err != nil {
		return err
	}
	return nil
}

// timestampColumnDef returns the "<name> <type> [NOT NULL DEFAULT now()]"
// column definition execCreateTable/execAddTimestamps add for name (one of
// created_at/updated_at/deleted_at, see timestampColumnSpecs) when
// Timestamps is enabled - created_at/updated_at are always NOT NULL
// DEFAULT now(), the same physical type FieldTypeDateTime itself renders
// as (see pgColumnType); deleted_at is nullable, indexed separately (see
// createTimestampsIndex).
func timestampColumnDef(name string) string {
	if name == "deleted_at" {
		return pgIdent(name) + " timestamp with time zone"
	}
	return pgIdent(name) + " timestamp with time zone NOT NULL DEFAULT now()"
}

// timestampColumnDefs returns every column execCreateTable adds when
// Timestamps is enabled from the start, in created_at/updated_at/
// deleted_at order - unlike execAddTimestamps (which only adds whatever's
// actually missing on an already-existing table, see AddTimestampsAction's
// doc), a brand new table has nothing to adopt, so all three are always
// wanted.
func timestampColumnDefs() []string {
	defs := make([]string, len(timestampColumnSpecs))
	for i, spec := range timestampColumnSpecs {
		defs[i] = timestampColumnDef(spec.name)
	}
	return defs
}

// timestampsIndexName is the name deleted_at's own index is created under -
// the same naming convention a relation's own FK column index uses (see
// pgRelationIndexName), reused directly rather than duplicated.
func timestampsIndexName(physTable string) string {
	return pgRelationIndexName(physTable, "deleted_at")
}

// createTimestampsIndex indexes physTable's deleted_at column - soft-delete
// lookups filter on it, called right after the column itself is created
// (or, for execAddTimestamps, even when deleted_at was adopted rather than
// added - it may have existed without this index, see AddTimestampsAction's
// doc) by both execCreateTable and execAddTimestamps.
func createTimestampsIndex(tx *sql.Tx, physTable, pgSchema string) error {
	q := fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s (%s)",
		pgIdent(timestampsIndexName(physTable)), pgQualifiedIdent(pgSchema, physTable), pgIdent("deleted_at"))
	if _, err := tx.Exec(q); err != nil {
		return fmt.Errorf("indexing deleted_at: %w", err)
	}
	return nil
}

// execAddTimestamps adds a.Columns (a subset of created_at/updated_at/
// deleted_at decided once at generation time, see AddTimestampsAction's
// doc) to modelName's own table, then creates deleted_at's index unless
// a.IndexExisted (it was already there, adopted the same way an existing
// compatible column is - see AddTimestampsAction's doc) - its inverse is
// execDelTimestamps.
func execAddTimestamps(tx *sql.Tx, modelName string, a *AddTimestampsAction, pgSchema string) error {
	if a == nil {
		return fmt.Errorf("addTimestamps action has no payload")
	}
	physTable := pgTableName(modelName)
	table := pgQualifiedIdent(pgSchema, physTable)

	if len(a.Columns) > 0 {
		clauses := make([]string, len(a.Columns))
		for i, name := range a.Columns {
			clauses[i] = "ADD COLUMN " + timestampColumnDef(name)
		}
		query := fmt.Sprintf("ALTER TABLE %s %s", table, strings.Join(clauses, ", "))
		if _, err := tx.Exec(query); err != nil {
			return fmt.Errorf("adding timestamps to %q: %w", modelName, err)
		}
	}

	if !a.IndexExisted {
		if err := createTimestampsIndex(tx, physTable, pgSchema); err != nil {
			return fmt.Errorf("adding timestamps to %q: %w", modelName, err)
		}
	}
	return nil
}

// execDelTimestamps drops a.Columns (never more - a column AddTimestamps
// adopted rather than added isn't this action's to remove, see
// DelTimestampsAction's doc) from modelName's own table - its inverse is
// execAddTimestamps. deleted_at's own index is dropped unless
// a.IndexExisted (it already existed before the AddTimestamps this is
// undoing, so it isn't this action's to remove either - the same
// adopted-vs-added distinction Columns already makes, applied to the
// index) - dropped explicitly first, the same explicit style
// execToggleRelationIndex already uses for its own DROP INDEX, rather than
// relying on DROP COLUMN to take a dependent index along implicitly. IF
// EXISTS regardless, since this can run as the inverse of an AddTimestamps
// that was itself preceded, in the same rollback, by an unrelated
// DelField-based removal/AddField-based restoration of these same columns
// (turning Timestamps off is ordinary field deletion, see
// ModelDiff.AddTimestamps's doc) - that path never recreates the index on
// its own way back in, so by the time this runs an index this action does
// own may already be gone too.
func execDelTimestamps(tx *sql.Tx, modelName string, a *DelTimestampsAction, pgSchema string) error {
	if a == nil {
		return fmt.Errorf("delTimestamps action has no payload")
	}
	physTable := pgTableName(modelName)
	table := pgQualifiedIdent(pgSchema, physTable)

	if !a.IndexExisted {
		idxQuery := fmt.Sprintf("DROP INDEX IF EXISTS %s", pgQualifiedIdent(pgSchema, timestampsIndexName(physTable)))
		if _, err := tx.Exec(idxQuery); err != nil {
			return fmt.Errorf("dropping timestamps from %q: %w", modelName, err)
		}
	}

	if len(a.Columns) == 0 {
		return nil
	}
	clauses := make([]string, len(a.Columns))
	for i, name := range a.Columns {
		clauses[i] = "DROP COLUMN " + pgIdent(name)
	}
	query := fmt.Sprintf("ALTER TABLE %s %s", table, strings.Join(clauses, ", "))
	if _, err := tx.Exec(query); err != nil {
		return fmt.Errorf("dropping timestamps from %q: %w", modelName, err)
	}
	return nil
}

func execAddField(tx *sql.Tx, modelName string, a *AddFieldAction, pgSchema string) error {
	if a == nil {
		return fmt.Errorf("addField action has no payload")
	}

	def, err := pgColumnDefinition(pgColumnName(a.FieldName), a.Definition)
	if err != nil {
		return fmt.Errorf("field %q: %w", a.FieldName, err)
	}

	query := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", pgQualifiedIdent(pgSchema, pgTableName(modelName)), def)
	if _, err := tx.Exec(query); err != nil {
		return fmt.Errorf("adding column %q to %q: %w", a.FieldName, modelName, err)
	}
	return nil
}

func execDelField(tx *sql.Tx, modelName string, a *DelFieldAction, pgSchema string) error {
	if a == nil {
		return fmt.Errorf("delField action has no payload")
	}

	physTable, physField := pgTableName(modelName), pgColumnName(a.FieldName)
	query := fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", pgQualifiedIdent(pgSchema, physTable), pgIdent(physField))
	if _, err := tx.Exec(query); err != nil {
		return fmt.Errorf("dropping column %q from %q: %w", a.FieldName, modelName, err)
	}

	if err := DeleteColumnType(tx, pgQualifiedTable(pgSchema, physTable), physField); err != nil {
		return err
	}
	return nil
}

func execRenameField(tx *sql.Tx, modelName string, a *RenameFieldAction, pgSchema string) error {
	if a == nil {
		return fmt.Errorf("renameField action has no payload")
	}

	physTable, physOld, physNew := pgTableName(modelName), pgColumnName(a.OldFieldName), pgColumnName(a.NewFieldName)
	query := fmt.Sprintf("ALTER TABLE %s RENAME COLUMN %s TO %s",
		pgQualifiedIdent(pgSchema, physTable), pgIdent(physOld), pgIdent(physNew))
	if _, err := tx.Exec(query); err != nil {
		return fmt.Errorf("renaming column %q to %q on %q: %w", a.OldFieldName, a.NewFieldName, modelName, err)
	}

	if err := DeleteColumnType(tx, pgQualifiedTable(pgSchema, physTable), physOld); err != nil {
		return err
	}
	return nil
}

func execChangeField(tx *sql.Tx, modelName string, a *ChangeFieldAction, pgSchema string) error {
	if a == nil {
		return fmt.Errorf("changeField action has no payload")
	}

	colType, err := pgColumnType(a.NewDefinition)
	if err != nil {
		return fmt.Errorf("field %q: %w", a.FieldName, err)
	}
	table, col := pgQualifiedIdent(pgSchema, pgTableName(modelName)), pgIdent(pgColumnName(a.FieldName))

	typeQuery := fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s TYPE %s USING %s::%s", table, col, colType, col, colType)
	if _, err := tx.Exec(typeQuery); err != nil {
		return fmt.Errorf("changing type of %q on %q: %w", a.FieldName, modelName, err)
	}

	nullQuery := fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET NOT NULL", table, col)
	if !a.NewDefinition.Required {
		nullQuery = fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP NOT NULL", table, col)
	}
	if _, err := tx.Exec(nullQuery); err != nil {
		return fmt.Errorf("setting nullability of %q on %q: %w", a.FieldName, modelName, err)
	}

	if a.NewDefinition.Default != nil {
		lit, err := pgDefaultSQL(a.NewDefinition)
		if err != nil {
			return fmt.Errorf("field %q: %w", a.FieldName, err)
		}
		query := fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET DEFAULT %s", table, col, lit)
		if _, err := tx.Exec(query); err != nil {
			return fmt.Errorf("setting default of %q on %q: %w", a.FieldName, modelName, err)
		}
	} else {
		query := fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP DEFAULT", table, col)
		if _, err := tx.Exec(query); err != nil {
			return fmt.Errorf("dropping default of %q on %q: %w", a.FieldName, modelName, err)
		}
	}

	return nil
}

// execAddRelation creates the physical shape AddRelationAction's
// Definition describes - a FK column(+UNIQUE for RelationTypeOneToOne) on
// modelName's own table for RelationTypeOneToOne/RelationTypeManyToOne,
// or a join table for RelationTypeManyToMany. RelationTypeOneToMany/the
// non-holding RelationTypeOneToOne side never reach here - filterIgnorableRelations
// (via canIgnoreRelation) already keeps them out of Added, the only
// source BuildModelActions builds AddRelation actions from.
func execAddRelation(tx *sql.Tx, modelName string, a *AddRelationAction, pgSchema string) error {
	if a == nil {
		return fmt.Errorf("addRelation action has no payload")
	}
	relatedPgSchema := pgResolveSchema(a.RelatedNamespace)
	switch a.Definition.Type {
	case RelationTypeOneToOne, RelationTypeManyToOne:
		return execAddToOneRelation(tx, modelName, a.RelationName, a.Definition, pgSchema, relatedPgSchema)
	case RelationTypeManyToMany:
		return execAddManyToManyRelation(tx, modelName, a, pgSchema, relatedPgSchema)
	default:
		return fmt.Errorf("relation %q: type %q can not be added directly", a.RelationName, a.Definition.Type)
	}
}

// execAddToOneRelation adds relationName's FK column to modelName's own
// table - RelationTypeOneToOne additionally gets UNIQUE (see Relation's
// doc: this side always holds the FK column for either type). The related
// table is qualified with relatedPgSchema, its own resolved schema
// (AddRelationAction.RelatedNamespace) - not assumed to equal pgSchema, so
// a relation between models resolved to different schemas works the same
// as one between models in the same schema.
func execAddToOneRelation(tx *sql.Tx, modelName, relationName string, r Relation, pgSchema, relatedPgSchema string) error {
	physTable := pgTableName(modelName)
	physRelatedTable := pgTableName(r.RelatedModel)
	physColumn := pgRelationColumnName(relationName)
	fkName := pgRelationFkName(physTable, physColumn)

	query := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s integer NOT NULL CONSTRAINT %s REFERENCES %s(id)",
		pgQualifiedIdent(pgSchema, physTable), pgIdent(physColumn), pgIdent(fkName), pgQualifiedIdent(relatedPgSchema, physRelatedTable))
	if r.Type == RelationTypeOneToOne {
		query += " UNIQUE"
	}
	if _, err := tx.Exec(query); err != nil {
		return fmt.Errorf("adding relation %q to %q: %w", relationName, modelName, err)
	}

	// RelationTypeOneToOne's UNIQUE constraint above already carries its
	// own backing index (see Relation.NoIndex's doc) - r.NoIndex is always
	// false for it (validated at schema-load time), so this condition
	// alone would already skip the block below for it, but the explicit
	// type check avoids ever creating a redundant second index on the
	// same column regardless of that invariant holding elsewhere.
	if r.Type != RelationTypeOneToOne && !r.NoIndex {
		idxQuery := fmt.Sprintf("CREATE INDEX %s ON %s (%s)",
			pgIdent(pgRelationIndexName(physTable, physColumn)), pgQualifiedIdent(pgSchema, physTable), pgIdent(physColumn))
		if _, err := tx.Exec(idxQuery); err != nil {
			return fmt.Errorf("indexing relation %q on %q: %w", relationName, modelName, err)
		}
	}

	fk := RelationFk{
		Type: r.Type, HomeTable: pgQualifiedTable(pgSchema, physTable), HomeModel: modelName, HomeAttribute: relationName,
		RelatedTable: pgQualifiedTable(relatedPgSchema, physRelatedTable), RelatedModel: r.RelatedModel, RelatedAttribute: r.RelatedAttribute,
	}
	return SetRelationFk(tx, fkName, fk)
}

// execAddManyToManyRelation creates the join table for a.Definition -
// deterministically named (see pgManyToManyTableName), with one FK column
// per side, each optionally indexed independently (a.Definition.NoIndex
// for modelName's own column, a.RelatedNoIndex for the other side's - see
// AddRelationAction's doc for where that comes from), and one metadata
// row per side (see RelationFk's doc - a many-to-many relation is always
// recorded from both sides' own perspective). The related table is
// qualified with relatedPgSchema, its own resolved schema, the same as
// execAddToOneRelation - the join table itself always lives in pgSchema
// (the acting model's own schema), regardless of which schema the related
// table resolves to; Postgres has no requirement either way, and this
// keeps the join table's location deterministic without a third schema
// concept of its own.
func execAddManyToManyRelation(tx *sql.Tx, modelName string, a *AddRelationAction, pgSchema, relatedPgSchema string) error {
	if err := ensureSchemaExists(tx, pgSchema); err != nil {
		return err
	}

	r := a.Definition
	physTable := pgTableName(modelName)
	physRelatedTable := pgTableName(r.RelatedModel)
	joinTable := pgManyToManyTableName(modelName, a.RelationName, r.RelatedModel, r.RelatedAttribute)
	ownColumn := pgJoinColumnName(modelName, a.RelationName)
	relColumn := pgJoinColumnName(r.RelatedModel, r.RelatedAttribute)
	ownFkName := pgRelationFkName(joinTable, ownColumn)
	relFkName := pgRelationFkName(joinTable, relColumn)

	query := fmt.Sprintf(
		"CREATE TABLE %s (%s integer NOT NULL CONSTRAINT %s REFERENCES %s(id), %s integer NOT NULL CONSTRAINT %s REFERENCES %s(id))",
		pgQualifiedIdent(pgSchema, joinTable),
		pgIdent(ownColumn), pgIdent(ownFkName), pgQualifiedIdent(pgSchema, physTable),
		pgIdent(relColumn), pgIdent(relFkName), pgQualifiedIdent(relatedPgSchema, physRelatedTable),
	)
	if _, err := tx.Exec(query); err != nil {
		return fmt.Errorf("adding relation %q to %q: %w", a.RelationName, modelName, err)
	}

	if !r.NoIndex {
		q := fmt.Sprintf("CREATE INDEX %s ON %s (%s)",
			pgIdent(pgRelationIndexName(joinTable, ownColumn)), pgQualifiedIdent(pgSchema, joinTable), pgIdent(ownColumn))
		if _, err := tx.Exec(q); err != nil {
			return fmt.Errorf("indexing relation %q on %q: %w", a.RelationName, modelName, err)
		}
	}
	if !a.RelatedNoIndex {
		q := fmt.Sprintf("CREATE INDEX %s ON %s (%s)",
			pgIdent(pgRelationIndexName(joinTable, relColumn)), pgQualifiedIdent(pgSchema, joinTable), pgIdent(relColumn))
		if _, err := tx.Exec(q); err != nil {
			return fmt.Errorf("indexing relation %q on %q: %w", r.RelatedAttribute, r.RelatedModel, err)
		}
	}

	qualifiedJoinTable := pgQualifiedTable(pgSchema, joinTable)
	fk := RelationFk{
		Type:      RelationTypeManyToMany,
		HomeTable: pgQualifiedTable(pgSchema, physTable), HomeModel: modelName, HomeAttribute: a.RelationName,
		RelatedTable: pgQualifiedTable(relatedPgSchema, physRelatedTable), RelatedModel: r.RelatedModel, RelatedAttribute: r.RelatedAttribute,
		JoinTable: qualifiedJoinTable,
	}
	if err := SetRelationFk(tx, ownFkName, fk); err != nil {
		return err
	}

	relFk := RelationFk{
		Type:      RelationTypeManyToMany,
		HomeTable: pgQualifiedTable(relatedPgSchema, physRelatedTable), HomeModel: r.RelatedModel, HomeAttribute: r.RelatedAttribute,
		RelatedTable: pgQualifiedTable(pgSchema, physTable), RelatedModel: modelName, RelatedAttribute: a.RelationName,
		JoinTable: qualifiedJoinTable,
	}
	return SetRelationFk(tx, relFkName, relFk)
}

// execDelRelation removes the physical shape execAddRelation created for
// DelRelationAction's Definition - the exact inverse, same type dispatch.
func execDelRelation(tx *sql.Tx, modelName string, a *DelRelationAction, pgSchema string) error {
	if a == nil {
		return fmt.Errorf("delRelation action has no payload")
	}
	switch a.Definition.Type {
	case RelationTypeOneToOne, RelationTypeManyToOne:
		return execDelToOneRelation(tx, modelName, a.RelationName, a.Definition, pgSchema)
	case RelationTypeManyToMany:
		return execDelManyToManyRelation(tx, modelName, a.RelationName, a.Definition)
	default:
		return fmt.Errorf("relation %q: type %q can not be deleted directly", a.RelationName, a.Definition.Type)
	}
}

// execDelToOneRelation drops relationName's FK column from modelName's
// own table - DROP COLUMN takes the column's own UNIQUE/FOREIGN KEY
// constraints and their backing index(es) with it automatically, so only
// the recorded metadata needs its own explicit cleanup.
func execDelToOneRelation(tx *sql.Tx, modelName, relationName string, r Relation, pgSchema string) error {
	physTable := pgTableName(modelName)
	physColumn := pgRelationColumnName(relationName)
	fkName := pgRelationFkName(physTable, physColumn)

	query := fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", pgQualifiedIdent(pgSchema, physTable), pgIdent(physColumn))
	if _, err := tx.Exec(query); err != nil {
		return fmt.Errorf("dropping relation %q from %q: %w", relationName, modelName, err)
	}
	return DeleteRelationFk(tx, fkName)
}

// execDelManyToManyRelation drops the join table execAddManyToManyRelation
// created, and both sides' recorded metadata rows along with it. The join
// table's own physical name and schema are read back from the recorded
// metadata (see RelationFk.JoinTable's doc), not passed in - the same
// reasoning as reading it back rather than recomputing it (a prior rename
// couldn't have moved it to a different schema either, there being no such
// operation, but reading it back keeps this and execToggleRelationIndex's
// pattern for JoinTable consistent).
func execDelManyToManyRelation(tx *sql.Tx, modelName, relationName string, r Relation) error {
	ownFkName, ownFk, ok, err := loadRelationFkByHomeAttribute(tx, modelName, relationName)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("relation %q on %q has no recorded meaning", relationName, modelName)
	}
	relFkName, _, ok, err := loadRelationFkByHomeAttribute(tx, r.RelatedModel, r.RelatedAttribute)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("relation %q on %q has no recorded meaning", r.RelatedAttribute, r.RelatedModel)
	}

	joinSchema, physJoinTable := splitQualifiedTable(ownFk.JoinTable)
	if _, err := tx.Exec(fmt.Sprintf("DROP TABLE %s", pgQualifiedIdent(joinSchema, physJoinTable))); err != nil {
		return fmt.Errorf("dropping relation %q from %q: %w", relationName, modelName, err)
	}
	if err := DeleteRelationFk(tx, ownFkName); err != nil {
		return err
	}
	return DeleteRelationFk(tx, relFkName)
}

// execRenameRelation renames a relation from OldRelationName to
// NewRelationName - RelationTypeManyToOne and the FK-holding side of
// RelationTypeOneToOne rename their own physical column/constraint/index;
// everything else has no physical column of its own tied to the
// attribute name at all, so only the recorded metadata is updated (see
// RenameRelationAction's doc).
func execRenameRelation(tx *sql.Tx, modelName string, a *RenameRelationAction, pgSchema string) error {
	if a == nil {
		return fmt.Errorf("renameRelation action has no payload")
	}

	switch {
	case a.Definition.Type == RelationTypeManyToOne:
		return execRenamePhysicalRelation(tx, modelName, a, pgSchema)
	case a.Definition.Type == RelationTypeOneToOne && a.Definition.FkHolder:
		return execRenamePhysicalRelation(tx, modelName, a, pgSchema)
	case a.Definition.Type == RelationTypeManyToMany:
		return execRenameManyToManyRelation(tx, modelName, a)
	case a.Definition.Type == RelationTypeOneToMany, a.Definition.Type == RelationTypeOneToOne:
		return execRenamePassiveRelation(tx, modelName, a)
	default:
		return fmt.Errorf("relation %q: type %q can not be renamed", a.OldRelationName, a.Definition.Type)
	}
}

// execRenamePhysicalRelation renames the FK column/constraint (and its
// index, if any) a RelationTypeManyToOne/FK-holding-RelationTypeOneToOne
// relation owns, and updates its recorded meaning to match - the fk_name
// itself changes along with the column (see pgRelationFkName), so the old
// row is moved to the new name rather than updated in place.
func execRenamePhysicalRelation(tx *sql.Tx, modelName string, a *RenameRelationAction, pgSchema string) error {
	physTable := pgTableName(modelName)
	oldColumn := pgRelationColumnName(a.OldRelationName)
	newColumn := pgRelationColumnName(a.NewRelationName)
	oldFkName := pgRelationFkName(physTable, oldColumn)
	newFkName := pgRelationFkName(physTable, newColumn)

	renameColQuery := fmt.Sprintf("ALTER TABLE %s RENAME COLUMN %s TO %s", pgQualifiedIdent(pgSchema, physTable), pgIdent(oldColumn), pgIdent(newColumn))
	if _, err := tx.Exec(renameColQuery); err != nil {
		return fmt.Errorf("renaming relation %q to %q on %q: %w", a.OldRelationName, a.NewRelationName, modelName, err)
	}

	renameFkQuery := fmt.Sprintf("ALTER TABLE %s RENAME CONSTRAINT %s TO %s", pgQualifiedIdent(pgSchema, physTable), pgIdent(oldFkName), pgIdent(newFkName))
	if _, err := tx.Exec(renameFkQuery); err != nil {
		return fmt.Errorf("renaming foreign key %q to %q on %q: %w", oldFkName, newFkName, modelName, err)
	}

	if a.Definition.Type != RelationTypeOneToOne && !a.Definition.NoIndex {
		oldIdx, newIdx := pgRelationIndexName(physTable, oldColumn), pgRelationIndexName(physTable, newColumn)
		// The new name is never schema-qualified - ALTER INDEX ... RENAME TO
		// keeps the index in the same schema it's already in, only the old
		// name (being looked up) needs qualifying to resolve regardless of
		// search_path.
		renameIdxQuery := fmt.Sprintf("ALTER INDEX %s RENAME TO %s", pgQualifiedIdent(pgSchema, oldIdx), pgIdent(newIdx))
		if _, err := tx.Exec(renameIdxQuery); err != nil {
			return fmt.Errorf("renaming index %q to %q on %q: %w", oldIdx, newIdx, modelName, err)
		}
	}

	fk, ok, err := loadRelationFk(tx, oldFkName)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("foreign key %q has no recorded meaning", oldFkName)
	}
	fk.HomeAttribute = a.NewRelationName
	if err := SetRelationFk(tx, newFkName, fk); err != nil {
		return err
	}
	return DeleteRelationFk(tx, oldFkName)
}

// execRenamePassiveRelation updates the recorded RelatedAttribute on the
// OTHER side's own metadata row for the passive RelationTypeOneToMany
// side, or the non-holding RelationTypeOneToOne side - neither has a
// physical column or a metadata row of its own at all (see RelationFk's
// doc), so this is the only place their new name is recorded.
func execRenamePassiveRelation(tx *sql.Tx, modelName string, a *RenameRelationAction) error {
	physRelatedTable := pgTableName(a.Definition.RelatedModel)
	relatedColumn := pgRelationColumnName(a.Definition.RelatedAttribute)
	fkName := pgRelationFkName(physRelatedTable, relatedColumn)

	fk, ok, err := loadRelationFk(tx, fkName)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("foreign key %q has no recorded meaning", fkName)
	}
	fk.RelatedAttribute = a.NewRelationName
	return SetRelationFk(tx, fkName, fk)
}

// execRenameManyToManyRelation updates modelName's own attribute name in
// both recorded metadata rows a many-to-many relation always has (see
// RelationFk's doc) - its own row's HomeAttribute, and the other side's
// row's RelatedAttribute. No physical DDL at all: both rows are found by
// their own recorded home_model/home_attribute (a.OldRelationName is still
// this side's current name at this point - a rename in progress hasn't
// updated it yet), not by recomputing the join table's name from the
// relation's current attribute names, which a prior rename on either side
// could have made stale (see RelationFk's doc).
func execRenameManyToManyRelation(tx *sql.Tx, modelName string, a *RenameRelationAction) error {
	ownFkName, ownFk, ok, err := loadRelationFkByHomeAttribute(tx, modelName, a.OldRelationName)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("relation %q on %q has no recorded meaning", a.OldRelationName, modelName)
	}
	ownFk.HomeAttribute = a.NewRelationName
	if err := SetRelationFk(tx, ownFkName, ownFk); err != nil {
		return err
	}

	relFkName, relFk, ok, err := loadRelationFkByHomeAttribute(tx, a.Definition.RelatedModel, a.Definition.RelatedAttribute)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("relation %q on %q has no recorded meaning", a.Definition.RelatedAttribute, a.Definition.RelatedModel)
	}
	relFk.RelatedAttribute = a.NewRelationName
	return SetRelationFk(tx, relFkName, relFk)
}

// execChangeRelation applies ChangeRelationAction - Type/RelatedModel/
// FkHolder are the only parts of a relation's definition any DDL is ever
// built from (see execAddToOneRelation/execAddManyToManyRelation); as long
// as none of those changed, this is a metadata-only update (see
// execUpdateRelationRelatedAttribute for RelatedAttribute, execToggle
// RelationIndex for NoIndex) - no DDL, regardless of which of the two
// changed. This matters beyond a direct edit of RelatedAttribute itself:
// renaming the OTHER side of a relation (RenameRelationAction, on that
// side's own model) already updates this side's own recorded
// RelatedAttribute directly, but CompareRelations still reports this
// side's own relation as Changed too (its code/DB shapes now differ by
// RelatedAttribute) - treating that as a DDL-worthy change would tear down
// and rebuild a physical column/join table (losing its data, or simply
// failing against a NOT NULL column with existing rows) for an edit that
// never should have touched DDL in the first place.
//
// Anything else (Type/RelatedModel/FkHolder) has no in-place SQL
// equivalent, so the relation's old physical shape is torn down and the
// new one built (see execDelRelation/execAddRelation) - except when the
// edit would also move which side holds the physical foreign key
// (canIgnoreRelation's answer for modelName differs between old and new),
// which isn't supported as a single change: OldDefinition might describe a
// side with no physical shape to tear down at all (RelationTypeOneToMany/
// the non-holding RelationTypeOneToOne side), or NewDefinition might
// describe a side that no longer owns the relation's physical
// representation going forward - either way there's no well-defined single
// action here, so this is rejected rather than attempted.
func execChangeRelation(tx *sql.Tx, modelName string, a *ChangeRelationAction, pgSchema string) error {
	if a == nil {
		return fmt.Errorf("changeRelation action has no payload")
	}

	old, new := a.OldDefinition, a.NewDefinition
	if canIgnoreRelation(modelName, a.RelationName, old) != canIgnoreRelation(modelName, a.RelationName, new) {
		return fmt.Errorf(
			"relation %q on %q: this edit moves which side holds the physical foreign key - not supported as a single change; delete the relation and re-add it instead",
			a.RelationName, modelName,
		)
	}

	physicallyEqual := old.Type == new.Type && old.RelatedModel == new.RelatedModel && old.FkHolder == new.FkHolder
	if !physicallyEqual {
		if err := execDelRelation(tx, modelName, &DelRelationAction{RelationName: a.RelationName, Definition: old}, pgSchema); err != nil {
			return err
		}
		return execAddRelation(tx, modelName, &AddRelationAction{
			RelationName: a.RelationName, Definition: new, RelatedNoIndex: a.RelatedNoIndex, RelatedNamespace: a.RelatedNamespace,
		}, pgSchema)
	}

	if old.RelatedAttribute != new.RelatedAttribute {
		if err := execUpdateRelationRelatedAttribute(tx, modelName, a.RelationName, old, new.RelatedAttribute); err != nil {
			return err
		}
	}
	if old.NoIndex != new.NoIndex {
		return execToggleRelationIndex(tx, modelName, a.RelationName, old, new.NoIndex, pgSchema)
	}
	return nil
}

// execUpdateRelationRelatedAttribute updates the recorded RelatedAttribute
// on relationName's own metadata row(s) to newRelatedAttribute - no DDL,
// the physical column/constraint/join table never encodes RelatedAttribute
// at all (see execChangeRelation's doc). RelationTypeManyToMany has two
// rows (see RelationFk's doc); only this side's own row needs updating
// here, since the other side's own row is either unrelated to this edit or
// already updated by that side's own RenameRelationAction - found by its
// recorded home_model/home_attribute (see loadRelationFkByHomeAttribute).
// RelationTypeManyToOne/RelationTypeOneToOne - their own fk_name only ever
// depends on this side's own current table/column, which a rename on the OTHER
// side never touches - so it's cheaper to just recompute it directly.
func execUpdateRelationRelatedAttribute(tx *sql.Tx, modelName, relationName string, r Relation, newRelatedAttribute string) error {
	if r.Type == RelationTypeManyToMany {
		fkName, fk, ok, err := loadRelationFkByHomeAttribute(tx, modelName, relationName)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("relation %q on %q has no recorded meaning", relationName, modelName)
		}
		fk.RelatedAttribute = newRelatedAttribute
		return SetRelationFk(tx, fkName, fk)
	}
	if r.Type != RelationTypeManyToOne && r.Type != RelationTypeOneToOne {
		return fmt.Errorf("relation %q: type %q has no recorded meaning to update", relationName, r.Type)
	}

	fkName := pgRelationFkName(pgTableName(modelName), pgRelationColumnName(relationName))
	fk, ok, err := loadRelationFk(tx, fkName)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("foreign key %q has no recorded meaning", fkName)
	}
	fk.RelatedAttribute = newRelatedAttribute
	return SetRelationFk(tx, fkName, fk)
}

// execToggleRelationIndex creates or drops the index on relationName's own
// physical column - the only in-place change a relation can have without
// touching its FK column/constraint at all.
func execToggleRelationIndex(tx *sql.Tx, modelName, relationName string, r Relation, newNoIndex bool, pgSchema string) error {
	if r.NoIndex == newNoIndex {
		return nil
	}

	var table, column, schema string
	switch r.Type {
	case RelationTypeManyToOne:
		table, column, schema = pgTableName(modelName), pgRelationColumnName(relationName), pgSchema
	case RelationTypeManyToMany:
		// The join table's own physical name and schema are read back from
		// the recorded metadata (see RelationFk.JoinTable's doc) rather than
		// assumed to be pgSchema - the same reasoning execDelManyToManyRelation
		// already follows for the same field.
		_, fk, ok, err := loadRelationFkByHomeAttribute(tx, modelName, relationName)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("relation %q on %q has no recorded meaning", relationName, modelName)
		}
		schema, table = splitQualifiedTable(fk.JoinTable)
		column = pgJoinColumnName(modelName, relationName)
	default:
		// RelationTypeOneToOne can never actually reach here (r.NoIndex ==
		// newNoIndex above already returns early - NoIndex is always false
		// for it, validated at schema-load time); the passive
		// RelationTypeOneToMany side has no physical column of its own to
		// index at all.
		return fmt.Errorf("relation %q: type %q has no index to toggle", relationName, r.Type)
	}

	idxName := pgRelationIndexName(table, column)
	if newNoIndex {
		// DROP INDEX needs the schema-qualified form to resolve regardless
		// of search_path - unlike CREATE INDEX below, there's no ON clause
		// to infer the schema from.
		if _, err := tx.Exec(fmt.Sprintf("DROP INDEX %s", pgQualifiedIdent(schema, idxName))); err != nil {
			return fmt.Errorf("dropping index on relation %q of %q: %w", relationName, modelName, err)
		}
		return nil
	}
	q := fmt.Sprintf("CREATE INDEX %s ON %s (%s)", pgIdent(idxName), pgQualifiedIdent(schema, table), pgIdent(column))
	if _, err := tx.Exec(q); err != nil {
		return fmt.Errorf("indexing relation %q of %q: %w", relationName, modelName, err)
	}
	return nil
}

func deleteAllColumnTypes(tx *sql.Tx, tableName string) error {
	if err := ensureSystemTypesTable(tx); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM `+systemTypesTable+` WHERE table_name = $1`, tableName); err != nil {
		return fmt.Errorf("deleting recorded types of %q: %w", tableName, err)
	}
	return nil
}

// pgIdent quotes name as a Postgres identifier, doubling an embedded
// double quote - safe against reserved-word collisions and characters
// that would otherwise need escaping in unquoted form.
func pgIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

// pgQuoteLiteral quotes s as a Postgres string literal, doubling an
// embedded single quote (standard SQL escaping - the same convention this
// package already reads back in field_compact.go/db_introspect.go).
func pgQuoteLiteral(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// pgColumnType renders f's Postgres column type, including size/
// precision/scale where meaningful - the write-side counterpart of
// pgTypeToFieldType (db_introspect.go). FieldTypeDateTime always renders
// as "timestamp with time zone", never the naive "timestamp" (see
// pgTypeToFieldType's own doc comment - a naive timestamp column can't
// round-trip through FieldTypeDateTime's mandatory offset).
func pgColumnType(f Field) (string, error) {
	switch f.Type {
	case FieldTypeString:
		if f.Size > 0 {
			return fmt.Sprintf("character varying(%d)", f.Size), nil
		}
		return "text", nil
	case FieldTypeInt:
		return "integer", nil
	case FieldTypeFloat:
		return "double precision", nil
	case FieldTypeDecimal:
		if f.Precision > 0 {
			return fmt.Sprintf("numeric(%d,%d)", f.Precision, f.Scale), nil
		}
		return "numeric", nil
	case FieldTypeBool:
		return "boolean", nil
	case FieldTypeDate:
		return "date", nil
	case FieldTypeTime:
		return "time without time zone", nil
	case FieldTypeDateTime:
		return "timestamp with time zone", nil
	case FieldTypeInterval:
		return "interval", nil
	case FieldTypeDict:
		return "jsonb", nil
	default:
		return "", fmt.Errorf("unknown field type %q", f.Type)
	}
}

// pgDefaultSQL renders f.Default (already typed per Field's documented
// Default representation) as a Postgres literal expression suitable for a
// DEFAULT clause - quoted/cast as needed for f.Type, never naively
// string-interpolated. Callers only call this when f.Default != nil.
func pgDefaultSQL(f Field) (string, error) {
	switch f.Type {
	case FieldTypeString:
		s, ok := f.Default.(string)
		if !ok {
			return "", fmt.Errorf("default is %T, want string", f.Default)
		}
		return pgQuoteLiteral(s), nil

	case FieldTypeInt:
		n, ok := f.Default.(int64)
		if !ok {
			return "", fmt.Errorf("default is %T, want int64", f.Default)
		}
		return strconv.FormatInt(n, 10), nil

	case FieldTypeFloat:
		n, ok := f.Default.(float64)
		if !ok {
			return "", fmt.Errorf("default is %T, want float64", f.Default)
		}
		return strconv.FormatFloat(n, 'f', -1, 64), nil

	case FieldTypeDecimal:
		d, ok := f.Default.(decimal.Decimal)
		if !ok {
			return "", fmt.Errorf("default is %T, want decimal.Decimal", f.Default)
		}
		return pgQuoteLiteral(d.String()) + "::numeric", nil

	case FieldTypeBool:
		b, ok := f.Default.(bool)
		if !ok {
			return "", fmt.Errorf("default is %T, want bool", f.Default)
		}
		if b {
			return "TRUE", nil
		}
		return "FALSE", nil

	case FieldTypeDate:
		s, ok := f.Default.(string)
		if !ok {
			return "", fmt.Errorf("default is %T, want string", f.Default)
		}
		return pgQuoteLiteral(s) + "::date", nil

	case FieldTypeTime:
		s, ok := f.Default.(string)
		if !ok {
			return "", fmt.Errorf("default is %T, want string", f.Default)
		}
		return pgQuoteLiteral(s) + "::time", nil

	case FieldTypeDateTime:
		t, ok := f.Default.(time.Time)
		if !ok {
			return "", fmt.Errorf("default is %T, want time.Time", f.Default)
		}
		return pgQuoteLiteral(t.UTC().Format(time.RFC3339)) + "::timestamptz", nil

	case FieldTypeInterval:
		d, ok := f.Default.(time.Duration)
		if !ok {
			return "", fmt.Errorf("default is %T, want time.Duration", f.Default)
		}
		// A plain "<seconds> seconds" literal is unambiguous for Postgres
		// to parse, sidestepping any day/hour/minute breakdown - Postgres
		// itself normalizes it however it normally displays interval
		// values (see parsePgInterval in db_introspect.go, which already
		// handles reading that back).
		return pgQuoteLiteral(fmt.Sprintf("%.6f seconds", d.Seconds())) + "::interval", nil

	case FieldTypeDict:
		b, err := json.Marshal(f.Default)
		if err != nil {
			return "", fmt.Errorf("default is not JSON-serializable: %w", err)
		}
		return pgQuoteLiteral(string(b)) + "::jsonb", nil

	default:
		return "", fmt.Errorf("unknown field type %q", f.Type)
	}
}

// pgColumnDefinition renders "<name> <type> [NOT NULL] [DEFAULT ...]" for
// use in CREATE TABLE/ADD COLUMN.
func pgColumnDefinition(name string, f Field) (string, error) {
	colType, err := pgColumnType(f)
	if err != nil {
		return "", err
	}

	def := pgIdent(name) + " " + colType
	if f.Required {
		def += " NOT NULL"
	}
	if f.Default != nil {
		lit, err := pgDefaultSQL(f)
		if err != nil {
			return "", err
		}
		def += " DEFAULT " + lit
	}
	return def, nil
}
