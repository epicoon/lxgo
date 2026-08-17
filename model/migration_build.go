package model

import (
	"database/sql"
	"fmt"
	"path/filepath"

	"github.com/epicoon/lxgo/migrator"
)

// MigrationType is the migration file "type" a generated migration
// declares - whatever applies/inverts these migrations registers this
// value via migrator.RegisterMigrationType.
const MigrationType = "model"

// MigrationActionType identifies what one Action does.
type MigrationActionType string

const (
	ActionCreateTable    MigrationActionType = "createTable"
	ActionDropTable      MigrationActionType = "dropTable"
	ActionAddField       MigrationActionType = "addField"
	ActionChangeField    MigrationActionType = "changeField"
	ActionRenameField    MigrationActionType = "renameField"
	ActionDelField       MigrationActionType = "delField"
	ActionAddRelation    MigrationActionType = "addRelation"
	ActionChangeRelation MigrationActionType = "changeRelation"
	ActionRenameRelation MigrationActionType = "renameRelation"
	ActionDelRelation    MigrationActionType = "delRelation"
	ActionAddTimestamps  MigrationActionType = "addTimestamps"
	ActionDelTimestamps  MigrationActionType = "delTimestamps"
)

// AddFieldAction adds Definition under FieldName - its inverse is a
// DelFieldAction with the same data.
type AddFieldAction struct {
	FieldName  string `yaml:"FieldName"`
	Definition Field  `yaml:"Definition"`
}

// DelFieldAction removes the field FieldName, previously declared as
// Definition - its inverse is an AddFieldAction with the same data.
type DelFieldAction struct {
	FieldName  string `yaml:"FieldName"`
	Definition Field  `yaml:"Definition"`
}

// ChangeFieldAction changes FieldName's definition from OldDefinition to
// NewDefinition - its inverse swaps the two.
type ChangeFieldAction struct {
	FieldName     string `yaml:"FieldName"`
	OldDefinition Field  `yaml:"OldDefinition"`
	NewDefinition Field  `yaml:"NewDefinition"`
}

// RenameFieldAction renames a field from OldFieldName to NewFieldName,
// definition unchanged - its inverse swaps the two names.
type RenameFieldAction struct {
	OldFieldName string `yaml:"OldFieldName"`
	NewFieldName string `yaml:"NewFieldName"`
}

// AddRelationAction adds Definition under RelationName - its inverse is a
// DelRelationAction with the same data. RelatedNoIndex only matters for
// RelationTypeManyToMany (each side's own join-table column is indexed
// independently, see Relation.NoIndex's doc) - it's the OTHER side's own
// declared NoIndex, which BuildModelActions can't see on its own (it only
// processes one model's diff at a time), filled in separately by
// GenerateMigration (see attachRelatedRelationInfo) from that other
// model's own schema file. Always false for every other RelationType.
// RelatedNamespace is the related model's own resolved Postgres schema
// (see ModelSchema.EffectiveNamespace), filled in the same way and at the
// same time as RelatedNoIndex, but for every relation type that reaches
// here (RelationTypeOneToOne/RelationTypeManyToOne/RelationTypeManyToMany
// all physically reference the related table) - empty means the related
// model resolves to the default schema, the same meaning Action.Namespace
// gives an empty value.
type AddRelationAction struct {
	RelationName     string   `yaml:"RelationName"`
	Definition       Relation `yaml:"Definition"`
	RelatedNoIndex   bool     `yaml:"RelatedNoIndex,omitempty"`
	RelatedNamespace string   `yaml:"RelatedNamespace,omitempty"`
}

// DelRelationAction removes the relation RelationName, previously declared
// as Definition - its inverse is an AddRelationAction with the same data.
// RelatedNoIndex/RelatedNamespace are AddRelationAction's own fields,
// reused here for the same reason - needed so Inverse() recreates the
// related side's index choice and schema exactly as they were.
type DelRelationAction struct {
	RelationName     string   `yaml:"RelationName"`
	Definition       Relation `yaml:"Definition"`
	RelatedNoIndex   bool     `yaml:"RelatedNoIndex,omitempty"`
	RelatedNamespace string   `yaml:"RelatedNamespace,omitempty"`
}

// ChangeRelationAction changes RelationName's definition from
// OldDefinition to NewDefinition - its inverse swaps the two (and
// RelatedNoIndex/RelatedNamespace, unchanged - neither is old/new-versioned,
// see below). Applying this is one of two shapes depending on what
// actually differs: if only NoIndex changed, it's a plain index toggle on
// the relation's existing physical column(s); anything else
// (Type/RelatedModel/RelatedAttribute/FkHolder) has no in-place SQL
// equivalent - the relation's old physical shape is torn down and the new
// one built, the same DDL AddRelation/DelRelation already know how to do
// (see Apply). RelatedNoIndex/RelatedNamespace are AddRelationAction's own
// fields, reused here for the same reason (a NewDefinition needs the
// related model's own NoIndex/namespace the same way a fresh AddRelation
// would).
type ChangeRelationAction struct {
	RelationName     string   `yaml:"RelationName"`
	OldDefinition    Relation `yaml:"OldDefinition"`
	NewDefinition    Relation `yaml:"NewDefinition"`
	RelatedNoIndex   bool     `yaml:"RelatedNoIndex,omitempty"`
	RelatedNamespace string   `yaml:"RelatedNamespace,omitempty"`
}

// RenameRelationAction renames a relation from OldRelationName to
// NewRelationName - Definition describes the relation's shape, carried
// here unlike RenameFieldAction because applying a relation rename needs
// to know Definition.Type/FkHolder to decide what DDL (if any) it takes: a
// field rename is always the same ALTER TABLE RENAME COLUMN regardless of
// the field's type, but a relation's rename might touch a physical column
// (RelationTypeManyToOne/the FK-holding side of RelationTypeOneToOne) or
// might be metadata-only (every other case) - see Apply.
//
// Unlike an explicit field rename (RenamedFrom), which can also carry a
// type change in the same edit and so sometimes needs a following
// ChangeFieldAction too, a relation rename never does: CompareRelations
// only classifies something as a rename when the old and new sides are
// already Relation.IsEqual apart from the name (see CompareRelations's
// doc) - so OldRelationName and NewRelationName always share the exact
// same Definition, and this one field describes both unambiguously.
type RenameRelationAction struct {
	OldRelationName string   `yaml:"OldRelationName"`
	NewRelationName string   `yaml:"NewRelationName"`
	Definition      Relation `yaml:"Definition"`
}

// AddTimestampsAction adds Columns - a subset of created_at/updated_at/
// deleted_at, exactly the ones ModelDiff.AddTimestamps found missing at
// generation time (baked in here the same way every other action bakes
// in its own decision, rather than being re-derived when the migration is
// later applied - see missingTimestampColumns). A name not listed already
// existed physically when the migration was generated and is left alone
// by execAddTimestamps (adopted, not recreated - no data lost). IndexExisted
// is the same kind of generation-time decision for deleted_at's own index
// (see timestampsIndexExists/ModelDiff.AddTimestampsIndexMissing) - true if
// it already existed and is left alone, false if execAddTimestamps creates
// it. Its inverse is a DelTimestampsAction with the same Columns and
// IndexExisted.
type AddTimestampsAction struct {
	Columns      []string `yaml:"Columns"`
	IndexExisted bool     `yaml:"IndexExisted,omitempty"`
}

// DelTimestampsAction removes Columns - its inverse is an
// AddTimestampsAction with the same Columns. BuildModelActions never
// builds one directly (turning Timestamps off needs no dedicated action,
// see ModelDiff.AddTimestamps's doc) - the only way one ever appears in a
// migration is as an AddTimestampsAction's own Inverse(), the same way
// ActionDropTable only ever appears as ActionCreateTable's. Dropping
// exactly Columns (not unconditionally all three) matters here for the
// same reason it does on the way in: a column AddTimestamps adopted
// rather than added isn't this action's to remove either. IndexExisted
// carries the same meaning across the inversion: true means the index
// already existed before the AddTimestamps this is undoing, so
// execDelTimestamps leaves it alone rather than dropping something that
// predates the migration.
type DelTimestampsAction struct {
	Columns      []string `yaml:"Columns"`
	IndexExisted bool     `yaml:"IndexExisted,omitempty"`
}

// Action is one step of a generated migration - exactly one of
// CreateTable/DropTable/AddField/ChangeField/RenameField/DelField/
// AddTimestamps/DelTimestamps is set, matching Type. ModelName is the model
// the action applies to (redundant with CreateTable/DropTable's own
// Schema.Name, but always present so a reader doesn't need to branch on
// Type just to find out which model an action belongs to). Namespace is the
// model's resolved Postgres schema (see ModelSchema.EffectiveNamespace) at
// the time the migration was generated - empty means no override anywhere
// in the cascade, the same meaning EffectiveNamespace itself gives it.
// Redundant with CreateTable/DropTable's own embedded ModelSchema for the
// same reason ModelName is: every exec* handler reads it from here
// uniformly, instead of only the table-level actions being able to answer
// "which schema". Timestamps is CreateTable's own resolved Timestamps
// switch (see ModelSchema.EffectiveTimestamps) at generation time, needed
// the same way Namespace is - CreateTable's embedded ModelSchema never
// marshals its own Resolved* fields (see ModelSchema.MarshalYAML), so this
// is the only value execCreateTable can read it back from once the
// migration file is unmarshaled. Unlike Namespace, it's only ever set (and
// meaningful) on an ActionCreateTable - every other action type leaves it
// at its zero value.
type Action struct {
	Type       MigrationActionType `yaml:"Type"`
	ModelName  string              `yaml:"ModelName"`
	Namespace  string              `yaml:"Namespace,omitempty"`
	Timestamps bool                `yaml:"Timestamps,omitempty"`

	CreateTable *ModelSchema       `yaml:"CreateTable,omitempty"`
	DropTable   *ModelSchema       `yaml:"DropTable,omitempty"`
	AddField    *AddFieldAction    `yaml:"AddField,omitempty"`
	ChangeField *ChangeFieldAction `yaml:"ChangeField,omitempty"`
	RenameField *RenameFieldAction `yaml:"RenameField,omitempty"`
	DelField    *DelFieldAction    `yaml:"DelField,omitempty"`

	AddRelation    *AddRelationAction    `yaml:"AddRelation,omitempty"`
	ChangeRelation *ChangeRelationAction `yaml:"ChangeRelation,omitempty"`
	RenameRelation *RenameRelationAction `yaml:"RenameRelation,omitempty"`
	DelRelation    *DelRelationAction    `yaml:"DelRelation,omitempty"`

	AddTimestamps *AddTimestampsAction `yaml:"AddTimestamps,omitempty"`
	DelTimestamps *DelTimestampsAction `yaml:"DelTimestamps,omitempty"`
}

// Inverse returns the action that undoes a. Returns an error if a's Type
// doesn't match a known action type, or if the payload field Type calls
// for is nil (a malformed Action, e.g. built by hand or unmarshaled from a
// corrupted migration file). The result always carries a's own Namespace
// unchanged - inverting an action never moves a model to a different
// schema, only undoes what it did within its own.
func (a Action) Inverse() (Action, error) {
	inv, err := a.inverseWithoutNamespace()
	if err != nil {
		return Action{}, err
	}
	inv.Namespace = a.Namespace
	return inv, nil
}

func (a Action) inverseWithoutNamespace() (Action, error) {
	switch a.Type {
	case ActionCreateTable:
		if a.CreateTable == nil {
			return Action{}, fmt.Errorf("%s action has no createTable payload", a.Type)
		}
		return Action{Type: ActionDropTable, ModelName: a.ModelName, DropTable: a.CreateTable}, nil

	case ActionDropTable:
		if a.DropTable == nil {
			return Action{}, fmt.Errorf("%s action has no dropTable payload", a.Type)
		}
		return Action{Type: ActionCreateTable, ModelName: a.ModelName, CreateTable: a.DropTable}, nil

	case ActionAddField:
		if a.AddField == nil {
			return Action{}, fmt.Errorf("%s action has no addField payload", a.Type)
		}
		return Action{Type: ActionDelField, ModelName: a.ModelName, DelField: &DelFieldAction{
			FieldName: a.AddField.FieldName, Definition: a.AddField.Definition,
		}}, nil

	case ActionDelField:
		if a.DelField == nil {
			return Action{}, fmt.Errorf("%s action has no delField payload", a.Type)
		}
		return Action{Type: ActionAddField, ModelName: a.ModelName, AddField: &AddFieldAction{
			FieldName: a.DelField.FieldName, Definition: a.DelField.Definition,
		}}, nil

	case ActionChangeField:
		if a.ChangeField == nil {
			return Action{}, fmt.Errorf("%s action has no changeField payload", a.Type)
		}
		return Action{Type: ActionChangeField, ModelName: a.ModelName, ChangeField: &ChangeFieldAction{
			FieldName:     a.ChangeField.FieldName,
			OldDefinition: a.ChangeField.NewDefinition,
			NewDefinition: a.ChangeField.OldDefinition,
		}}, nil

	case ActionRenameField:
		if a.RenameField == nil {
			return Action{}, fmt.Errorf("%s action has no renameField payload", a.Type)
		}
		return Action{Type: ActionRenameField, ModelName: a.ModelName, RenameField: &RenameFieldAction{
			OldFieldName: a.RenameField.NewFieldName, NewFieldName: a.RenameField.OldFieldName,
		}}, nil

	case ActionAddRelation:
		if a.AddRelation == nil {
			return Action{}, fmt.Errorf("%s action has no addRelation payload", a.Type)
		}
		return Action{Type: ActionDelRelation, ModelName: a.ModelName, DelRelation: &DelRelationAction{
			RelationName: a.AddRelation.RelationName, Definition: a.AddRelation.Definition,
			RelatedNoIndex: a.AddRelation.RelatedNoIndex, RelatedNamespace: a.AddRelation.RelatedNamespace,
		}}, nil

	case ActionDelRelation:
		if a.DelRelation == nil {
			return Action{}, fmt.Errorf("%s action has no delRelation payload", a.Type)
		}
		return Action{Type: ActionAddRelation, ModelName: a.ModelName, AddRelation: &AddRelationAction{
			RelationName: a.DelRelation.RelationName, Definition: a.DelRelation.Definition,
			RelatedNoIndex: a.DelRelation.RelatedNoIndex, RelatedNamespace: a.DelRelation.RelatedNamespace,
		}}, nil

	case ActionChangeRelation:
		if a.ChangeRelation == nil {
			return Action{}, fmt.Errorf("%s action has no changeRelation payload", a.Type)
		}
		return Action{Type: ActionChangeRelation, ModelName: a.ModelName, ChangeRelation: &ChangeRelationAction{
			RelationName:     a.ChangeRelation.RelationName,
			OldDefinition:    a.ChangeRelation.NewDefinition,
			NewDefinition:    a.ChangeRelation.OldDefinition,
			RelatedNoIndex:   a.ChangeRelation.RelatedNoIndex,
			RelatedNamespace: a.ChangeRelation.RelatedNamespace,
		}}, nil

	case ActionRenameRelation:
		if a.RenameRelation == nil {
			return Action{}, fmt.Errorf("%s action has no renameRelation payload", a.Type)
		}
		return Action{Type: ActionRenameRelation, ModelName: a.ModelName, RenameRelation: &RenameRelationAction{
			OldRelationName: a.RenameRelation.NewRelationName, NewRelationName: a.RenameRelation.OldRelationName,
			Definition: a.RenameRelation.Definition,
		}}, nil

	case ActionAddTimestamps:
		if a.AddTimestamps == nil {
			return Action{}, fmt.Errorf("%s action has no addTimestamps payload", a.Type)
		}
		return Action{Type: ActionDelTimestamps, ModelName: a.ModelName, DelTimestamps: &DelTimestampsAction{
			Columns: a.AddTimestamps.Columns, IndexExisted: a.AddTimestamps.IndexExisted,
		}}, nil

	case ActionDelTimestamps:
		if a.DelTimestamps == nil {
			return Action{}, fmt.Errorf("%s action has no delTimestamps payload", a.Type)
		}
		return Action{Type: ActionAddTimestamps, ModelName: a.ModelName, AddTimestamps: &AddTimestampsAction{
			Columns: a.DelTimestamps.Columns, IndexExisted: a.DelTimestamps.IndexExisted,
		}}, nil

	default:
		return Action{}, fmt.Errorf("unknown action type %q", a.Type)
	}
}

// BuildModelActions turns diff (see CompareModel) into the ordered list of
// actions needed to bring the database in line with diff.CodeSchema - a
// CreateTable if the table doesn't exist yet (followed by an AddRelation
// per diff.Relations.Added - a brand new model can already declare
// relations, and those need their own physical action just like an
// existing model's would, see CompareModel), otherwise one action per
// changed/renamed/added/deleted field and relation, plus an AddTimestamps
// if diff.AddTimestamps or diff.AddTimestampsIndexMissing (an
// already-existing table whose resolved Timestamps switch just turned on,
// but is still missing a column and/or deleted_at's own index - a brand
// new table's own Timestamps is instead carried directly on its
// CreateTable action, see Action's doc; turning Timestamps back off needs
// no equivalent action here at all, see ModelDiff.AddTimestamps's doc).
// Emitted even when diff.AddTimestamps is empty, as long as the index is
// still missing (diff.AddTimestampsIndexMissing) - a table can have
// adopted all three columns by hand and still never have had deleted_at
// indexed.
//
// A field CompareFields paired as an explicit rename (RenamedField.
// Explicit) can also have a changed definition in the same schema edit -
// CompareFields itself doesn't record whether that's the case (its
// RenamedField only carries the two names), so this is where that's
// checked: a RenameField action is always emitted for the pair, and a
// ChangeField action follows it if the two fields aren't Field.IsEqual.
// CompareRelations has no such case (see RenameRelationAction's doc) - a
// RenameRelation action never needs a following ChangeRelation.
//
// Returns an error if diff.Fields/diff.Relations names a field/relation
// that isn't actually in diff.CodeSchema/diff.DBSchema - can't happen for
// a diff produced by CompareModel/CompareSchemas, but diff is exported and
// callers can build one by hand, so this is checked rather than silently
// building an action around a zero-value Field/Relation.
//
// A caller combining several models' actions into one migration (see
// GenerateMigration) still needs to move every CreateTable ahead of every
// relation action across the WHOLE batch, not just within one model's own
// list - a many-to-many AddRelation can reference a table that itself
// needs creating in this same migration, possibly by a different model's
// diff. BuildModelActions only orders its own single model's actions
// correctly; see reorderCreateTableFirst for the cross-model fix.
func BuildModelActions(diff ModelDiff) ([]Action, error) {
	// Every action produced here belongs to diff.Name, so they all share
	// its one resolved namespace - stamped onto the whole batch right
	// before each return rather than repeated at every Action{} literal.
	namespace := diff.CodeSchema.EffectiveNamespace()

	if diff.NeedsTable {
		actions := []Action{{
			Type: ActionCreateTable, ModelName: diff.Name, CreateTable: diff.CodeSchema,
			Timestamps: diff.CodeSchema.EffectiveTimestamps(),
		}}
		for _, name := range diff.Relations.Added {
			r, ok := diff.CodeSchema.RelationByName(name)
			if !ok {
				return nil, fmt.Errorf("model %q: added relation %q not found in code schema", diff.Name, name)
			}
			actions = append(actions, Action{
				Type: ActionAddRelation, ModelName: diff.Name,
				AddRelation: &AddRelationAction{RelationName: name, Definition: r},
			})
		}
		for i := range actions {
			actions[i].Namespace = namespace
		}
		return actions, nil
	}

	var actions []Action

	for _, r := range diff.Fields.Renamed {
		newField, ok := diff.CodeSchema.FieldByName(r.To)
		if !ok {
			return nil, fmt.Errorf("model %q: renamed field %q not found in code schema", diff.Name, r.To)
		}
		oldField, ok := diff.DBSchema.FieldByName(r.From)
		if !ok {
			return nil, fmt.Errorf("model %q: renamed field %q not found in database schema", diff.Name, r.From)
		}
		actions = append(actions, Action{
			Type: ActionRenameField, ModelName: diff.Name,
			RenameField: &RenameFieldAction{OldFieldName: r.From, NewFieldName: r.To},
		})
		if !newField.IsEqual(oldField) {
			actions = append(actions, Action{
				Type: ActionChangeField, ModelName: diff.Name,
				ChangeField: &ChangeFieldAction{FieldName: r.To, OldDefinition: oldField, NewDefinition: newField},
			})
		}
	}

	for _, name := range diff.Fields.Changed {
		newField, ok := diff.CodeSchema.FieldByName(name)
		if !ok {
			return nil, fmt.Errorf("model %q: changed field %q not found in code schema", diff.Name, name)
		}
		oldField, ok := diff.DBSchema.FieldByName(name)
		if !ok {
			return nil, fmt.Errorf("model %q: changed field %q not found in database schema", diff.Name, name)
		}
		actions = append(actions, Action{
			Type: ActionChangeField, ModelName: diff.Name,
			ChangeField: &ChangeFieldAction{FieldName: name, OldDefinition: oldField, NewDefinition: newField},
		})
	}

	for _, name := range diff.Fields.Added {
		f, ok := diff.CodeSchema.FieldByName(name)
		if !ok {
			return nil, fmt.Errorf("model %q: added field %q not found in code schema", diff.Name, name)
		}
		actions = append(actions, Action{
			Type: ActionAddField, ModelName: diff.Name,
			AddField: &AddFieldAction{FieldName: name, Definition: f},
		})
	}

	for _, name := range diff.Fields.Deleted {
		f, ok := diff.DBSchema.FieldByName(name)
		if !ok {
			return nil, fmt.Errorf("model %q: deleted field %q not found in database schema", diff.Name, name)
		}
		actions = append(actions, Action{
			Type: ActionDelField, ModelName: diff.Name,
			DelField: &DelFieldAction{FieldName: name, Definition: f},
		})
	}

	for _, r := range diff.Relations.Renamed {
		// r.From/r.To are always Relation.IsEqual (see
		// RenameRelationAction's doc) - either side's own definition
		// describes both, so the code schema's (the "new" side's) is used.
		rel, ok := diff.CodeSchema.RelationByName(r.To)
		if !ok {
			return nil, fmt.Errorf("model %q: renamed relation %q not found in code schema", diff.Name, r.To)
		}
		actions = append(actions, Action{
			Type: ActionRenameRelation, ModelName: diff.Name,
			RenameRelation: &RenameRelationAction{OldRelationName: r.From, NewRelationName: r.To, Definition: rel},
		})
	}

	for _, name := range diff.Relations.Changed {
		newRel, ok := diff.CodeSchema.RelationByName(name)
		if !ok {
			return nil, fmt.Errorf("model %q: changed relation %q not found in code schema", diff.Name, name)
		}
		if canIgnoreRelation(diff.Name, name, newRel) {
			// Unlike Added/Deleted (filtered by CompareModel itself, see
			// canIgnoreRelation), Changed is deliberately left unfiltered
			// there - model:status and other diagnostic consumers want to
			// see a mismatch reported from both sides. But only one
			// physical action should ever come out of it: the acting
			// side's own ChangeRelation already rebuilds (or re-indexes)
			// the whole relation, including this side's own recorded
			// meaning where one exists at all (see execChangeRelation) -
			// building a second action from this side would either be
			// redundant or, for RelationTypeOneToMany/the non-holding
			// RelationTypeOneToOne side, have no physical column of its
			// own to act on in the first place.
			continue
		}
		oldRel, ok := diff.DBSchema.RelationByName(name)
		if !ok {
			return nil, fmt.Errorf("model %q: changed relation %q not found in database schema", diff.Name, name)
		}
		actions = append(actions, Action{
			Type: ActionChangeRelation, ModelName: diff.Name,
			ChangeRelation: &ChangeRelationAction{RelationName: name, OldDefinition: oldRel, NewDefinition: newRel},
		})
	}

	for _, name := range diff.Relations.Added {
		r, ok := diff.CodeSchema.RelationByName(name)
		if !ok {
			return nil, fmt.Errorf("model %q: added relation %q not found in code schema", diff.Name, name)
		}
		actions = append(actions, Action{
			Type: ActionAddRelation, ModelName: diff.Name,
			AddRelation: &AddRelationAction{RelationName: name, Definition: r},
		})
	}

	for _, name := range diff.Relations.Deleted {
		r, ok := diff.DBSchema.RelationByName(name)
		if !ok {
			return nil, fmt.Errorf("model %q: deleted relation %q not found in database schema", diff.Name, name)
		}
		actions = append(actions, Action{
			Type: ActionDelRelation, ModelName: diff.Name,
			DelRelation: &DelRelationAction{RelationName: name, Definition: r},
		})
	}

	if len(diff.AddTimestamps) > 0 || diff.AddTimestampsIndexMissing {
		actions = append(actions, Action{
			Type: ActionAddTimestamps, ModelName: diff.Name,
			AddTimestamps: &AddTimestampsAction{Columns: diff.AddTimestamps, IndexExisted: !diff.AddTimestampsIndexMissing},
		})
	}

	for i := range actions {
		actions[i].Namespace = namespace
	}
	return actions, nil
}

// attachRelatedRelationInfo fills in RelatedNamespace for every
// AddRelation/ChangeRelation/DelRelation action in actions, and
// RelatedNoIndex for the RelationTypeManyToMany ones among them (mutating
// them in place, all three carry a pointer) - see AddRelationAction's doc
// for why this can't be done inside BuildModelActions itself.
// RelatedNamespace applies to every relation type that reaches here
// (RelationTypeOneToOne/RelationTypeManyToOne/RelationTypeManyToMany all
// physically reference the related table); RelatedNoIndex only matters for
// RelationTypeManyToMany (each side's own join-table column is indexed
// independently, see Relation.NoIndex's doc).
//
// AddRelation/ChangeRelation read the related model's own CURRENT schema
// file (schemaByName - every model's own file, not just the ones with a
// diff, see GenerateMigration) rather than through diffs, since the
// related side might have no diff of its own at all if this relation is
// the only thing that changed on it (filtered out of its own Added by
// canIgnoreRelation). A related model/relation that can't be found there
// (shouldn't happen for a diff produced by CompareSchemas, whose
// cross-schema validation already requires the pairing to exist) is left
// at the zero value (indexed, default schema).
//
// DelRelation's own RelatedNamespace still comes from schemaByName - the
// related MODEL's schema file is still there even though the relation
// itself has already been removed from it (the only way to remove a
// symmetrically-declared relation), since a model's own Namespace doesn't
// depend on which relations it currently declares. RelatedNoIndex is the
// one part DelRelation can't get from schemaByName at all: by the time a
// relation is Deleted, nothing there still describes the deleted
// relation's own NoIndex, so its only surviving source is the related
// model's own live table structure (still physically there, about to be
// dropped by this very DelRelation) - read via db instead. Needed so
// Inverse() (undoing the delete) recreates the related side's index choice
// exactly as it was, not just defaulted back to indexed (see
// DelRelationAction's doc) - an unreadable/missing related table is the
// same best-effort case as a schemaByName miss above.
func attachRelatedRelationInfo(db *sql.DB, actions []Action, schemaByName map[string]*ModelSchema) {
	relatedNamespace := func(def Relation) string {
		related, ok := schemaByName[def.RelatedModel]
		if !ok {
			return ""
		}
		return related.EffectiveNamespace()
	}

	relatedNoIndex := func(def Relation) (bool, bool) {
		related, ok := schemaByName[def.RelatedModel]
		if !ok {
			return false, false
		}
		rel, ok := related.RelationByName(def.RelatedAttribute)
		if !ok {
			return false, false
		}
		return rel.NoIndex, true
	}

	relatedNoIndexFromDB := func(def Relation) bool {
		var relatedTimestamps bool
		if related, ok := schemaByName[def.RelatedModel]; ok {
			relatedTimestamps = related.EffectiveTimestamps()
		}
		relatedSchema, err := IntrospectModelSchema(db, pgTableName(def.RelatedModel), pgResolveSchema(relatedNamespace(def)), relatedTimestamps)
		if err != nil {
			return false
		}
		rel, ok := relatedSchema.RelationByName(def.RelatedAttribute)
		if !ok {
			return false
		}
		return rel.NoIndex
	}

	for i := range actions {
		switch actions[i].Type {
		case ActionAddRelation:
			if actions[i].AddRelation == nil {
				continue
			}
			actions[i].AddRelation.RelatedNamespace = relatedNamespace(actions[i].AddRelation.Definition)
			if actions[i].AddRelation.Definition.Type != RelationTypeManyToMany {
				continue
			}
			if noIndex, ok := relatedNoIndex(actions[i].AddRelation.Definition); ok {
				actions[i].AddRelation.RelatedNoIndex = noIndex
			}
		case ActionChangeRelation:
			if actions[i].ChangeRelation == nil {
				continue
			}
			actions[i].ChangeRelation.RelatedNamespace = relatedNamespace(actions[i].ChangeRelation.NewDefinition)
			if actions[i].ChangeRelation.NewDefinition.Type != RelationTypeManyToMany {
				continue
			}
			if noIndex, ok := relatedNoIndex(actions[i].ChangeRelation.NewDefinition); ok {
				actions[i].ChangeRelation.RelatedNoIndex = noIndex
			}
		case ActionDelRelation:
			if actions[i].DelRelation == nil {
				continue
			}
			actions[i].DelRelation.RelatedNamespace = relatedNamespace(actions[i].DelRelation.Definition)
			if actions[i].DelRelation.Definition.Type != RelationTypeManyToMany {
				continue
			}
			actions[i].DelRelation.RelatedNoIndex = relatedNoIndexFromDB(actions[i].DelRelation.Definition)
		}
	}
}

// reorderCreateTableFirst moves every ActionCreateTable in actions to the
// front (stable - everything else keeps its relative order) - see
// BuildModelActions's doc for why this matters across a whole migration
// batch: a many-to-many AddRelation can reference a table that itself
// needs creating in the same migration, possibly from a different model's
// own diff, so every table must exist before any relation action runs
// regardless of which model's action list it came from.
func reorderCreateTableFirst(actions []Action) []Action {
	ordered := make([]Action, 0, len(actions))
	var rest []Action
	for _, a := range actions {
		if a.Type == ActionCreateTable {
			ordered = append(ordered, a)
		} else {
			rest = append(rest, a)
		}
	}
	return append(ordered, rest...)
}

// migrationFile is a generated migration's on-disk shape - Type is always
// MigrationType, and is the same top-level "Type" field
// migrator.RegisterMigrationType's dispatcher reads generically (as a map
// key, not through this struct) to decide a migration file is this
// package's, before handing the raw content to Apply/Invert.
type migrationFile struct {
	Name    string   `yaml:"Name"`
	Type    string   `yaml:"Type"`
	Actions []Action `yaml:"Actions"`
}

// GenerateMigration compares every schema file across schemaDirs against
// db (see CompareSchemas) and, if there's any diff, writes a new migration
// file via migrator.CreateWithContent (name is the migration's descriptive
// name, migrator.CreateWithContent's own timestamping applies as usual).
// Returns the actions written, or nil if there was nothing to migrate.
// Returns ErrUnappliedMigrations (via CompareSchemas), without writing
// anything, if the database has unapplied migrations.
//
// For every field CompareFields paired as an explicit rename
// (RenamedField.Explicit, from the schema file's own renamedFrom), once
// the migration file is written, renamedFrom is cleared from that field in
// its schema file - the migration action itself keeps the old name (needed
// for Inverse()), but the schema file's job for that hint is done: an
// unapplied migration can't accumulate stale renamedFrom hints from a
// previous run. If that cleanup step fails, the actions already written to
// the migration file are still returned alongside the error - the
// migration itself exists on disk at that point, only the schema file
// cleanup didn't complete.
func GenerateMigration(db *sql.DB, schemas []*ModelSchema, name string) ([]Action, error) {
	diffs, err := CompareSchemas(db, schemas)
	if err != nil {
		return nil, err
	}
	if len(diffs) == 0 {
		return nil, nil
	}

	var actions []Action
	for _, d := range diffs {
		modelActions, err := BuildModelActions(d)
		if err != nil {
			return nil, err
		}
		actions = append(actions, modelActions...)
	}

	// attachRelatedRelationInfo needs every model's own schema file.
	schemaByName := make(map[string]*ModelSchema, len(schemas))
	for _, s := range schemas {
		schemaByName[s.Name] = s
	}
	attachRelatedRelationInfo(db, actions, schemaByName)

	actions = reorderCreateTableFirst(actions)

	content, err := marshalYAML(migrationFile{Name: name, Type: MigrationType, Actions: actions})
	if err != nil {
		return nil, fmt.Errorf("encoding migration content: %w", err)
	}
	if err := migrator.CreateWithContent(name, content); err != nil {
		return nil, err
	}

	if err := clearExplicitRenames(diffs); err != nil {
		return actions, err
	}

	return actions, nil
}

// clearExplicitRenames clears renamedFrom on every field diffs paired as
// an explicit rename, saving each affected schema file back to its own
// CodeSchema.SourceDir - see GenerateMigration.
func clearExplicitRenames(diffs []ModelDiff) error {
	for _, d := range diffs {
		if d.NeedsTable || len(d.Fields.Renamed) == 0 {
			continue
		}

		changed := false
		for i, f := range d.CodeSchema.Fields {
			if f.RenamedFrom == "" {
				continue
			}
			for _, r := range d.Fields.Renamed {
				if r.Explicit && r.To == f.Name && r.From == f.RenamedFrom {
					d.CodeSchema.Fields[i].RenamedFrom = ""
					changed = true
					break
				}
			}
		}

		if !changed {
			continue
		}
		path := filepath.Join(d.CodeSchema.SourceDir, d.CodeSchema.Name+".yaml")
		if err := d.CodeSchema.Save(path); err != nil {
			return fmt.Errorf("clearing renamedFrom in %q: %w", path, err)
		}
	}
	return nil
}
