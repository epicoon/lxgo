package model

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/epicoon/lxgo/migrator"
)

// ErrUnappliedMigrations is returned by CompareSchemas when the database
// has migrations that haven't been applied yet - the database isn't in
// the schema files' target state, so a diff against it would be
// misleading.
var ErrUnappliedMigrations = errors.New("database has unapplied migrations")

// RenamedField is one field CompareFields decided was renamed rather than
// deleted and separately added - see CompareFields.
type RenamedField struct {
	// From is the field's name in the database, To its name in the code
	// schema.
	From, To string
	// Explicit is true if To's Field declared RenamedFrom == From (an
	// author-asserted rename), false if it was inferred by matching
	// shape (Field.IsEqual) between a code-only and a db-only field -
	// see CompareFields.
	Explicit bool
}

// FieldsDiff is the result of comparing one model's fields as declared in
// its schema file against the fields of its table in the database - see
// CompareFields.
type FieldsDiff struct {
	Added   []string
	Deleted []string
	Changed []string
	Renamed []RenamedField
}

// IsEmpty reports whether the diff found no difference at all.
func (d FieldsDiff) IsEmpty() bool {
	return len(d.Added) == 0 && len(d.Deleted) == 0 && len(d.Changed) == 0 && len(d.Renamed) == 0
}

// CompareFields diffs codeSchema's fields (as declared in a schema file)
// against dbSchema's (as restored from the database, see
// IntrospectModelSchema) - a pure function, independent of how either
// ModelSchema was obtained.
//
// A field present under the same name in both, but not Field.IsEqual, is
// Changed. A field name present only on one side is either Added/Deleted,
// or Renamed if it's paired with a same-shaped field on the other side -
// paired two ways: first, a code-only field whose Field.RenamedFrom names
// an actual db-only field is an explicit rename, regardless of whether the
// two fields are otherwise IsEqual (an explicit rename can also change the
// field's definition in the same edit); everything left after that is
// paired by the same shape-matching heuristic (Field.IsEqual, ignoring
// name) - a coincidental match this way still ends up Renamed rather than
// Deleted+Added.
func CompareFields(codeSchema, dbSchema *ModelSchema) FieldsDiff {
	codeFields := make(map[string]Field, len(codeSchema.Fields))
	codeNames := make([]string, 0, len(codeSchema.Fields))
	for _, f := range codeSchema.Fields {
		codeFields[f.Name] = f.Field
		codeNames = append(codeNames, f.Name)
	}
	dbFields := make(map[string]Field, len(dbSchema.Fields))
	dbNames := make([]string, 0, len(dbSchema.Fields))
	for _, f := range dbSchema.Fields {
		dbFields[f.Name] = f.Field
		dbNames = append(dbNames, f.Name)
	}

	var changed []string
	codeOnly := make(map[string]bool, len(codeNames))
	for _, name := range codeNames {
		df, ok := dbFields[name]
		if !ok {
			codeOnly[name] = true
			continue
		}
		if !codeFields[name].IsEqual(df) {
			changed = append(changed, name)
		}
	}
	dbOnly := make(map[string]bool, len(dbNames))
	for _, name := range dbNames {
		if _, ok := codeFields[name]; !ok {
			dbOnly[name] = true
		}
	}

	var renamed []RenamedField

	for _, name := range codeNames {
		if !codeOnly[name] {
			continue
		}
		from := codeFields[name].RenamedFrom
		if from == "" || !dbOnly[from] {
			continue
		}
		renamed = append(renamed, RenamedField{From: from, To: name, Explicit: true})
		delete(codeOnly, name)
		delete(dbOnly, from)
	}

	for _, name := range codeNames {
		if !codeOnly[name] {
			continue
		}
		cf := codeFields[name]
		for _, oldName := range dbNames {
			if !dbOnly[oldName] {
				continue
			}
			if cf.IsEqual(dbFields[oldName]) {
				renamed = append(renamed, RenamedField{From: oldName, To: name, Explicit: false})
				delete(codeOnly, name)
				delete(dbOnly, oldName)
				break
			}
		}
	}

	var added, deleted []string
	for _, name := range codeNames {
		if codeOnly[name] {
			added = append(added, name)
		}
	}
	for _, name := range dbNames {
		if dbOnly[name] {
			deleted = append(deleted, name)
		}
	}

	return FieldsDiff{Added: added, Deleted: deleted, Changed: changed, Renamed: renamed}
}

// RenamedRelation is one relation CompareRelations decided was renamed
// rather than deleted and separately added - see CompareRelations.
type RenamedRelation struct {
	// From is the relation's name in the database, To its name in the
	// code schema.
	From, To string
}

// RelationsDiff is the result of comparing one model's relations as
// declared in its schema file against the relations restored from its
// table in the database - see CompareRelations.
type RelationsDiff struct {
	Added   []string
	Deleted []string
	Changed []string
	Renamed []RenamedRelation
}

// IsEmpty reports whether the diff found no difference at all.
func (d RelationsDiff) IsEmpty() bool {
	return len(d.Added) == 0 && len(d.Deleted) == 0 && len(d.Changed) == 0 && len(d.Renamed) == 0
}

// CompareRelations diffs codeRelations (as declared in a schema file)
// against dbRelations (as restored from the database, see
// IntrospectModelSchema) - a pure function, the relation counterpart of
// CompareFields, following the exact same shape: a relation present under
// the same name on both sides but not Relation.IsEqual is Changed; a name
// present on only one side is Added/Deleted, unless it pairs with a
// same-shaped relation on the other side (Relation.IsEqual, ignoring
// name), in which case it's Renamed instead. Unlike CompareFields, there's
// no explicit-rename equivalent of Field.RenamedFrom - only the
// shape-matching heuristic pairs a rename.
func CompareRelations(codeRelations, dbRelations []NamedRelation) RelationsDiff {
	codeByName := make(map[string]Relation, len(codeRelations))
	codeNames := make([]string, 0, len(codeRelations))
	for _, r := range codeRelations {
		codeByName[r.Name] = r.Relation
		codeNames = append(codeNames, r.Name)
	}
	dbByName := make(map[string]Relation, len(dbRelations))
	dbNames := make([]string, 0, len(dbRelations))
	for _, r := range dbRelations {
		dbByName[r.Name] = r.Relation
		dbNames = append(dbNames, r.Name)
	}

	var changed []string
	codeOnly := make(map[string]bool, len(codeNames))
	for _, name := range codeNames {
		dr, ok := dbByName[name]
		if !ok {
			codeOnly[name] = true
			continue
		}
		if !codeByName[name].IsEqual(dr) {
			changed = append(changed, name)
		}
	}
	dbOnly := make(map[string]bool, len(dbNames))
	for _, name := range dbNames {
		if _, ok := codeByName[name]; !ok {
			dbOnly[name] = true
		}
	}

	var renamed []RenamedRelation
	for _, name := range codeNames {
		if !codeOnly[name] {
			continue
		}
		cr := codeByName[name]
		for _, oldName := range dbNames {
			if !dbOnly[oldName] {
				continue
			}
			if cr.IsEqual(dbByName[oldName]) {
				renamed = append(renamed, RenamedRelation{From: oldName, To: name})
				delete(codeOnly, name)
				delete(dbOnly, oldName)
				break
			}
		}
	}

	var added, deleted []string
	for _, name := range codeNames {
		if codeOnly[name] {
			added = append(added, name)
		}
	}
	for _, name := range dbNames {
		if dbOnly[name] {
			deleted = append(deleted, name)
		}
	}

	return RelationsDiff{Added: added, Deleted: deleted, Changed: changed, Renamed: renamed}
}

// canIgnoreRelation reports whether modelName's own declaration of
// attributeName (r) should be skipped when deciding which side of a
// relation actually acts (generates an Added/Deleted entry) - a relation
// is declared symmetrically on both models (see the Relations package
// doc), but only one side may act, or the same relation would be added/
// deleted twice. RelationTypeOneToMany never acts (its FK lives on the
// RelationTypeManyToOne side); RelationTypeOneToOne acts only when
// FkHolder; RelationTypeManyToOne always acts (it's unconditionally the FK
// holder). RelationTypeManyToMany acts on the alphabetically-first of the
// two model names - if the model is the same on both sides, the attribute
// names are compared instead, except when they're equal too (a relation
// mirroring itself under one name), which always acts.
func canIgnoreRelation(modelName, attributeName string, r Relation) bool {
	switch r.Type {
	case RelationTypeOneToMany:
		return true
	case RelationTypeOneToOne:
		return !r.FkHolder
	case RelationTypeManyToMany:
		if modelName != r.RelatedModel {
			return modelName > r.RelatedModel
		}
		return attributeName != r.RelatedAttribute && attributeName > r.RelatedAttribute
	default:
		return false
	}
}

// filterIgnorableRelations drops every name from names whose Relation
// (looked up in schema) canIgnoreRelation - used to remove the
// non-acting side's Added/Deleted entries from a RelationsDiff before
// it's returned from CompareModel (see canIgnoreRelation).
func filterIgnorableRelations(names []string, schema *ModelSchema, modelName string) []string {
	if len(names) == 0 {
		return names
	}

	kept := make([]string, 0, len(names))
	for _, name := range names {
		r, ok := schema.RelationByName(name)
		if ok && canIgnoreRelation(modelName, name, r) {
			continue
		}
		kept = append(kept, name)
	}
	return kept
}

// ModelDiff is one model's comparison result against the database - see
// CompareModel.
type ModelDiff struct {
	// Name is the model's name (ModelSchema.Name).
	Name string
	// CodeSchema is the schema as declared in the schema file.
	CodeSchema *ModelSchema
	// DBSchema is the schema as introspected from the database - nil if
	// NeedsTable (there's nothing to introspect yet).
	DBSchema *ModelSchema
	// NeedsTable is true if the model's table doesn't exist yet - Fields/
	// Relations are the zero value in that case (there's nothing to diff
	// against).
	NeedsTable bool
	Fields     FieldsDiff
	Relations  RelationsDiff
	// AddTimestamps lists which of created_at/updated_at/deleted_at
	// codeSchema's resolved Timestamps (see ModelSchema.EffectiveTimestamps)
	// wants but the table doesn't have yet - nil if Timestamps is off, or
	// on but the table already has all three (an existing column under
	// one of these names is left alone rather than recreated, see
	// missingTimestampColumns - only a genuinely absent one is listed).
	// Always nil when NeedsTable (a brand new table's Timestamps is
	// instead carried directly on its CreateTable action, see Action's
	// doc). There's no mirror-image DelTimestamps: turning Timestamps off
	// needs no dedicated detection - IntrospectModelSchema stops excluding
	// the three columns the moment codeSchema.EffectiveTimestamps() is
	// false (see its own doc), so Fields already reports them as Deleted
	// through the ordinary path, the same as any other column a schema
	// file no longer declares.
	AddTimestamps []string
	// AddTimestampsIndexMissing is true if codeSchema's resolved
	// Timestamps wants deleted_at's own dedicated index (see
	// timestampsIndexName) but the table doesn't have it yet (see
	// timestampsIndexExists) - always false when Timestamps is off or
	// NeedsTable, the same conditions AddTimestamps is nil under. Tracked
	// separately from AddTimestamps because the index can be missing even
	// when every column is already present (a table adopted all three
	// columns by hand but never indexed deleted_at) - BuildModelActions
	// still needs an AddTimestamps action in that case even though
	// AddTimestamps itself is empty.
	AddTimestampsIndexMissing bool
}

// IsEmpty reports whether the model needs no change at all (no table to
// create, no field/relation diff, no Timestamps switched on).
func (d ModelDiff) IsEmpty() bool {
	return !d.NeedsTable && d.Fields.IsEmpty() && d.Relations.IsEmpty() &&
		len(d.AddTimestamps) == 0 && !d.AddTimestampsIndexMissing
}

// pgSchemaProjection returns a copy of schema with every field's Name (and
// RenamedFrom, if set) translated to its physical Postgres name via
// pgColumnName - so it can be compared directly against dbSchema
// (IntrospectModelSchema's result, which is physical by construction,
// straight from information_schema) without CompareFields itself needing
// to know about the logical/physical distinction at all.
//
// The returned map recovers a logical name from a physical one wherever
// that's possible: every current field's own physical name (guaranteed,
// it's how the map is built), plus - for a field declaring RenamedFrom -
// its old logical name, keyed by that name's own physical translation, so
// an explicit rename's old side can be reported logically too (see
// logicalizeFieldsDiff). A physical name with no code-side declaration at
// all (a column dropped from the schema file, or the old side of a rename
// CompareFields paired by shape rather than an explicit RenamedFrom) has no
// entry - callers fall back to the physical name for those.
func pgSchemaProjection(schema *ModelSchema) (*ModelSchema, map[string]string) {
	proj := &ModelSchema{Name: pgTableName(schema.Name)}
	physToLogical := make(map[string]string, len(schema.Fields))
	for _, f := range schema.Fields {
		physName := pgColumnName(f.Name)
		physField := f.Field
		if physField.RenamedFrom != "" {
			physOldName := pgColumnName(physField.RenamedFrom)
			if _, exists := physToLogical[physOldName]; !exists {
				physToLogical[physOldName] = physField.RenamedFrom
			}
			physField.RenamedFrom = physOldName
		}
		proj.Fields = append(proj.Fields, NamedField{Name: physName, Field: physField})
		physToLogical[physName] = f.Name
	}
	return proj, physToLogical
}

// logicalizeDBSchema returns a copy of dbSchema (physical names, as
// IntrospectModelSchema returns them) with every field whose physical name
// resolves through physToLogical renamed to that logical name - a field
// with no such mapping (see pgSchemaProjection) keeps its physical name.
// logicalName replaces dbSchema's own Name (the physical table name
// IntrospectModelSchema was called with) - the model's logical name, same
// reasoning as the fields. Relations need no such translation - unlike
// Fields, they're restored from systemRelationsTable's own logical
// payload (see RelationFk), never from a physical Postgres identifier, so
// dbSchema.Relations is already logical and carried over as-is.
func logicalizeDBSchema(dbSchema *ModelSchema, logicalName string, physToLogical map[string]string) *ModelSchema {
	logical := &ModelSchema{Name: logicalName, Relations: dbSchema.Relations}
	for _, f := range dbSchema.Fields {
		name := f.Name
		if logicalName, ok := physToLogical[name]; ok {
			name = logicalName
		}
		logical.Fields = append(logical.Fields, NamedField{Name: name, Field: f.Field})
	}
	return logical
}

// logicalizeFieldsDiff translates a FieldsDiff computed against
// pgSchemaProjection's physical-named schemas back to logical names,
// wherever physToLogical resolves one - see pgSchemaProjection for exactly
// which names that covers.
func logicalizeFieldsDiff(diff FieldsDiff, physToLogical map[string]string) FieldsDiff {
	logicalName := func(name string) string {
		if l, ok := physToLogical[name]; ok {
			return l
		}
		return name
	}

	out := FieldsDiff{Deleted: diff.Deleted}
	for _, name := range diff.Added {
		out.Added = append(out.Added, logicalName(name))
	}
	for _, name := range diff.Changed {
		out.Changed = append(out.Changed, logicalName(name))
	}
	for _, r := range diff.Renamed {
		out.Renamed = append(out.Renamed, RenamedField{
			From: logicalName(r.From), To: logicalName(r.To), Explicit: r.Explicit,
		})
	}
	return out
}

// CompareModel compares codeSchema against its table in db - NeedsTable is
// set (Fields/Relations and DBSchema left zero) if the table doesn't exist
// yet, otherwise DBSchema is the table's introspected schema (with every
// field still declared in codeSchema shown under its logical name - see
// logicalizeDBSchema), Fields is CompareFields(codeSchema, DBSchema),
// matched at the physical level (see pgSchemaProjection) and translated
// back to logical names (see logicalizeFieldsDiff), and Relations is
// CompareRelations(codeSchema.Relations, DBSchema.Relations) (no physical
// translation needed, see logicalizeDBSchema) with the non-acting side of
// every symmetrically-declared relation filtered out of Added/Deleted (see
// canIgnoreRelation) - Changed/Renamed aren't filtered, both sides
// independently report a mismatch the same way CompareFields's own
// consumers already expect duplicated-but-anchored errors elsewhere in
// this package (see the relation cross-validation in LoadModelSchemas).
// AddTimestamps/AddTimestampsIndexMissing compare codeSchema's resolved
// Timestamps switch against the table's actual columns and index (see
// missingTimestampColumns/timestampsIndexExists) - checked separately from
// Fields/DBSchema, since IntrospectModelSchema is called with codeSchema's
// OWN resolved Timestamps (so if it's already enabled, the three columns
// are excluded from DBSchema/Fields the same way "id" is, and can't be
// seen missing through CompareFields at all). Only the columns actually
// missing are listed - one already there (declared by hand while
// Timestamps was off) is left for execAddTimestamps to adopt, not
// recreated; the index is treated the same way (see
// AddTimestampsIndexMissing's doc). When Timestamps is disabled,
// IntrospectModelSchema does NOT exclude the three columns - a table that
// still has them (Timestamps was just turned off, or a migration author
// added one by hand) reports them as ordinary Fields, which CompareFields
// then reconciles on its own if codeSchema doesn't declare a matching
// field - no separate "DelTimestamps" detection needed for that
// direction.
func CompareModel(db *sql.DB, codeSchema *ModelSchema) (ModelDiff, error) {
	physSchema, physToLogical := pgSchemaProjection(codeSchema)
	pgSchema := pgResolveSchema(codeSchema.EffectiveNamespace())
	wantTimestamps := codeSchema.EffectiveTimestamps()

	dbSchema, err := IntrospectModelSchema(db, physSchema.Name, pgSchema, wantTimestamps)
	if errors.Is(err, ErrTableNotFound) {
		// The table doesn't exist yet, so every field is implicitly
		// "added" via CreateTable's own schema payload (BuildModelActions
		// doesn't need a FieldsDiff for that) - but relations are a
		// separate physical action (AddRelation) even when the table is
		// brand new, so this side still needs its own Added list, filtered
		// to the acting side the same way the normal (table-exists) path
		// already is (see filterIgnorableRelations).
		var names []string
		for _, r := range codeSchema.Relations {
			names = append(names, r.Name)
		}
		return ModelDiff{
			Name: codeSchema.Name, CodeSchema: codeSchema, NeedsTable: true,
			Relations: RelationsDiff{Added: filterIgnorableRelations(names, codeSchema, codeSchema.Name)},
		}, nil
	}
	if err != nil {
		return ModelDiff{}, fmt.Errorf("model %q: %w", codeSchema.Name, err)
	}

	logicalDB := logicalizeDBSchema(dbSchema, codeSchema.Name, physToLogical)

	relationsDiff := CompareRelations(codeSchema.Relations, logicalDB.Relations)
	relationsDiff.Added = filterIgnorableRelations(relationsDiff.Added, codeSchema, codeSchema.Name)
	relationsDiff.Deleted = filterIgnorableRelations(relationsDiff.Deleted, logicalDB, codeSchema.Name)

	var addTimestamps []string
	var addTimestampsIndexMissing bool
	if wantTimestamps {
		addTimestamps, err = missingTimestampColumns(db, physSchema.Name, pgSchema)
		if err != nil {
			return ModelDiff{}, fmt.Errorf("model %q: %w", codeSchema.Name, err)
		}
		indexExists, err := timestampsIndexExists(db, physSchema.Name, pgSchema)
		if err != nil {
			return ModelDiff{}, fmt.Errorf("model %q: %w", codeSchema.Name, err)
		}
		addTimestampsIndexMissing = !indexExists
	}

	return ModelDiff{
		Name:                      codeSchema.Name,
		CodeSchema:                codeSchema,
		DBSchema:                  logicalDB,
		Fields:                    logicalizeFieldsDiff(CompareFields(physSchema, dbSchema), physToLogical),
		Relations:                 relationsDiff,
		AddTimestamps:             addTimestamps,
		AddTimestampsIndexMissing: addTimestampsIndexMissing,
	}, nil
}

// CompareSchemas loads every schema file across schemaDirs (see
// LoadModelSchemasFromDirs) and compares each against its table in db, in
// name order. Returns ErrUnappliedMigrations without comparing anything if
// migrator.Check() (migrator must already be initialized, see
// migrator.Init) reports any migration that hasn't been applied yet - the
// database isn't in the schema files' target state, so a diff against it
// would be misleading. Models with no diff at all (ModelDiff.IsEmpty) are
// omitted from the result.
func CompareSchemas(db *sql.DB, schemas []*ModelSchema) ([]ModelDiff, error) {
	pending, err := migrator.Check()
	if err != nil {
		return nil, fmt.Errorf("checking for unapplied migrations: %w", err)
	}
	if len(pending) > 0 {
		return nil, ErrUnappliedMigrations
	}

	var diffs []ModelDiff
	for _, s := range schemas {
		d, err := CompareModel(db, s)
		if err != nil {
			return nil, err
		}
		if !d.IsEmpty() {
			diffs = append(diffs, d)
		}
	}
	return diffs, nil
}
