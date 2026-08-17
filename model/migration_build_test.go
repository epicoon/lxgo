package model

import (
	"reflect"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestBuildModelActions_NeedsTable(t *testing.T) {
	code := &ModelSchema{Name: "Widget", Fields: []NamedField{
		namedField("name", Field{Type: FieldTypeString}),
	}}
	diff := ModelDiff{Name: "Widget", CodeSchema: code, NeedsTable: true}

	actions, err := BuildModelActions(diff)
	if err != nil {
		t.Fatalf("BuildModelActions: %v", err)
	}
	if len(actions) != 1 {
		t.Fatalf("actions = %#v, want exactly one", actions)
	}
	a := actions[0]
	if a.Type != ActionCreateTable || a.ModelName != "Widget" || a.CreateTable != code {
		t.Fatalf("got %#v, want a CreateTable action wrapping code", a)
	}
}

func TestBuildModelActions_AddedChangedDeleted(t *testing.T) {
	code := &ModelSchema{Name: "Widget", Fields: []NamedField{
		namedField("kept", Field{Type: FieldTypeString, Size: 100}),
		namedField("brandNew", Field{Type: FieldTypeInt}),
	}}
	db := &ModelSchema{Name: "Widget", Fields: []NamedField{
		namedField("kept", Field{Type: FieldTypeString, Size: 50}),
		namedField("gone", Field{Type: FieldTypeBool}),
	}}
	fieldsDiff := CompareFields(code, db)
	d := ModelDiff{Name: "Widget", CodeSchema: code, DBSchema: db, Fields: fieldsDiff}

	actions, err := BuildModelActions(d)
	if err != nil {
		t.Fatalf("BuildModelActions: %v", err)
	}
	byType := map[MigrationActionType][]Action{}
	for _, a := range actions {
		byType[a.Type] = append(byType[a.Type], a)
	}

	if len(byType[ActionChangeField]) != 1 {
		t.Fatalf("expected exactly one changeField action, got %#v", byType[ActionChangeField])
	}
	ch := byType[ActionChangeField][0]
	if ch.ChangeField.FieldName != "kept" || ch.ChangeField.OldDefinition.Size != 50 || ch.ChangeField.NewDefinition.Size != 100 {
		t.Fatalf("changeField = %#v", ch.ChangeField)
	}

	if len(byType[ActionAddField]) != 1 {
		t.Fatalf("expected exactly one addField action, got %#v", byType[ActionAddField])
	}
	add := byType[ActionAddField][0]
	if add.AddField.FieldName != "brandNew" || add.AddField.Definition.Type != FieldTypeInt {
		t.Fatalf("addField = %#v", add.AddField)
	}

	if len(byType[ActionDelField]) != 1 {
		t.Fatalf("expected exactly one delField action, got %#v", byType[ActionDelField])
	}
	del := byType[ActionDelField][0]
	if del.DelField.FieldName != "gone" || del.DelField.Definition.Type != FieldTypeBool {
		t.Fatalf("delField = %#v", del.DelField)
	}
}

func TestBuildModelActions_RenamedWithoutRetype(t *testing.T) {
	shape := Field{Type: FieldTypeString, Size: 255}
	code := &ModelSchema{Name: "Widget", Fields: []NamedField{
		namedField("fullName", shape),
	}}
	db := &ModelSchema{Name: "Widget", Fields: []NamedField{
		namedField("name", shape),
	}}
	fieldsDiff := CompareFields(code, db)
	d := ModelDiff{Name: "Widget", CodeSchema: code, DBSchema: db, Fields: fieldsDiff}

	actions, err := BuildModelActions(d)
	if err != nil {
		t.Fatalf("BuildModelActions: %v", err)
	}
	if len(actions) != 1 {
		t.Fatalf("actions = %#v, want exactly a rename, no accompanying change", actions)
	}
	a := actions[0]
	if a.Type != ActionRenameField || a.RenameField.OldFieldName != "name" || a.RenameField.NewFieldName != "fullName" {
		t.Fatalf("got %#v", a)
	}
}

func TestBuildModelActions_RenamedWithRetype(t *testing.T) {
	code := &ModelSchema{Name: "Widget", Fields: []NamedField{
		namedField("fullName", Field{Type: FieldTypeString, Size: 500, RenamedFrom: "name"}),
	}}
	db := &ModelSchema{Name: "Widget", Fields: []NamedField{
		namedField("name", Field{Type: FieldTypeString, Size: 255}),
	}}
	fieldsDiff := CompareFields(code, db)
	d := ModelDiff{Name: "Widget", CodeSchema: code, DBSchema: db, Fields: fieldsDiff}

	actions, err := BuildModelActions(d)
	if err != nil {
		t.Fatalf("BuildModelActions: %v", err)
	}
	if len(actions) != 2 {
		t.Fatalf("actions = %#v, want a rename followed by a change", actions)
	}
	if actions[0].Type != ActionRenameField || actions[0].RenameField.OldFieldName != "name" || actions[0].RenameField.NewFieldName != "fullName" {
		t.Fatalf("actions[0] = %#v", actions[0])
	}
	if actions[1].Type != ActionChangeField || actions[1].ChangeField.FieldName != "fullName" ||
		actions[1].ChangeField.OldDefinition.Size != 255 || actions[1].ChangeField.NewDefinition.Size != 500 {
		t.Fatalf("actions[1] = %#v", actions[1])
	}
}

// TestBuildModelActions_NeedsTable_WithRelations checks that a brand new
// model already declaring relations gets an AddRelation for each one right
// after its CreateTable - not just the bare CreateTable a model with no
// relations gets (see TestBuildModelActions_NeedsTable).
func TestBuildModelActions_NeedsTable_WithRelations(t *testing.T) {
	code := &ModelSchema{
		Name:      "Widget",
		Fields:    []NamedField{namedField("name", Field{Type: FieldTypeString})},
		Relations: []NamedRelation{namedRelation("owner", Relation{Type: RelationTypeManyToOne, RelatedModel: "Gadget", RelatedAttribute: "widgets"})},
	}
	diff := ModelDiff{
		Name: "Widget", CodeSchema: code, NeedsTable: true,
		Relations: RelationsDiff{Added: []string{"owner"}},
	}

	actions, err := BuildModelActions(diff)
	if err != nil {
		t.Fatalf("BuildModelActions: %v", err)
	}
	if len(actions) != 2 {
		t.Fatalf("actions = %#v, want CreateTable followed by AddRelation", actions)
	}
	if actions[0].Type != ActionCreateTable || actions[0].CreateTable != code {
		t.Fatalf("actions[0] = %#v", actions[0])
	}
	if actions[1].Type != ActionAddRelation || actions[1].AddRelation.RelationName != "owner" ||
		actions[1].AddRelation.Definition.RelatedModel != "Gadget" {
		t.Fatalf("actions[1] = %#v", actions[1])
	}
}

func TestBuildModelActions_RelationsAddedChangedDeleted(t *testing.T) {
	code := &ModelSchema{Name: "Widget", Relations: []NamedRelation{
		namedRelation("kept", Relation{Type: RelationTypeManyToOne, RelatedModel: "Gadget", RelatedAttribute: "widgets"}),
		namedRelation("brandNew", Relation{Type: RelationTypeManyToOne, RelatedModel: "Sprocket", RelatedAttribute: "widgets"}),
	}}
	db := &ModelSchema{Name: "Widget", Relations: []NamedRelation{
		namedRelation("kept", Relation{Type: RelationTypeManyToOne, RelatedModel: "Gadget", RelatedAttribute: "widgets", NoIndex: true}),
		namedRelation("gone", Relation{Type: RelationTypeManyToOne, RelatedModel: "Doohickey", RelatedAttribute: "widgets"}),
	}}
	d := ModelDiff{Name: "Widget", CodeSchema: code, DBSchema: db, Relations: CompareRelations(code.Relations, db.Relations)}

	actions, err := BuildModelActions(d)
	if err != nil {
		t.Fatalf("BuildModelActions: %v", err)
	}
	byType := map[MigrationActionType][]Action{}
	for _, a := range actions {
		byType[a.Type] = append(byType[a.Type], a)
	}

	if len(byType[ActionChangeRelation]) != 1 {
		t.Fatalf("expected exactly one changeRelation action, got %#v", byType[ActionChangeRelation])
	}
	ch := byType[ActionChangeRelation][0]
	if ch.ChangeRelation.RelationName != "kept" || !ch.ChangeRelation.OldDefinition.NoIndex || ch.ChangeRelation.NewDefinition.NoIndex {
		t.Fatalf("changeRelation = %#v", ch.ChangeRelation)
	}

	if len(byType[ActionAddRelation]) != 1 {
		t.Fatalf("expected exactly one addRelation action, got %#v", byType[ActionAddRelation])
	}
	add := byType[ActionAddRelation][0]
	if add.AddRelation.RelationName != "brandNew" || add.AddRelation.Definition.RelatedModel != "Sprocket" {
		t.Fatalf("addRelation = %#v", add.AddRelation)
	}

	if len(byType[ActionDelRelation]) != 1 {
		t.Fatalf("expected exactly one delRelation action, got %#v", byType[ActionDelRelation])
	}
	del := byType[ActionDelRelation][0]
	if del.DelRelation.RelationName != "gone" || del.DelRelation.Definition.RelatedModel != "Doohickey" {
		t.Fatalf("delRelation = %#v", del.DelRelation)
	}
}

func TestBuildModelActions_RelationRenamed(t *testing.T) {
	shape := Relation{Type: RelationTypeManyToOne, RelatedModel: "Gadget", RelatedAttribute: "widgets"}
	code := &ModelSchema{Name: "Widget", Relations: []NamedRelation{namedRelation("newOwner", shape)}}
	db := &ModelSchema{Name: "Widget", Relations: []NamedRelation{namedRelation("owner", shape)}}
	d := ModelDiff{Name: "Widget", CodeSchema: code, DBSchema: db, Relations: CompareRelations(code.Relations, db.Relations)}

	actions, err := BuildModelActions(d)
	if err != nil {
		t.Fatalf("BuildModelActions: %v", err)
	}
	if len(actions) != 1 {
		t.Fatalf("actions = %#v, want exactly a rename, no accompanying change", actions)
	}
	a := actions[0]
	if a.Type != ActionRenameRelation || a.RenameRelation.OldRelationName != "owner" || a.RenameRelation.NewRelationName != "newOwner" {
		t.Fatalf("got %#v", a)
	}
}

// TestBuildModelActions_NeedsTable_Timestamps checks that a brand new
// model's resolved Timestamps switch is carried directly on its
// CreateTable action (Action.Timestamps), not through a separate
// AddTimestamps action - see Action's own doc for why CreateTable can't
// just read it back off its own embedded ModelSchema instead.
func TestBuildModelActions_NeedsTable_Timestamps(t *testing.T) {
	resolved := true
	code := &ModelSchema{Name: "Widget", ResolvedTimestamps: &resolved}
	diff := ModelDiff{Name: "Widget", CodeSchema: code, NeedsTable: true}

	actions, err := BuildModelActions(diff)
	if err != nil {
		t.Fatalf("BuildModelActions: %v", err)
	}
	if len(actions) != 1 {
		t.Fatalf("actions = %#v, want exactly one", actions)
	}
	if a := actions[0]; a.Type != ActionCreateTable || !a.Timestamps {
		t.Fatalf("got %#v, want a CreateTable action with Timestamps = true", a)
	}
}

// TestBuildModelActions_AddTimestamps checks that diff.AddTimestamps's own
// column list is carried through to the action unchanged - BuildModelActions
// doesn't recompute or filter it (see AddTimestampsAction's doc: the exact
// set was already decided at diff time, by missingTimestampColumns).
func TestBuildModelActions_AddTimestamps(t *testing.T) {
	code := &ModelSchema{Name: "Widget"}
	db := &ModelSchema{Name: "Widget"}
	d := ModelDiff{Name: "Widget", CodeSchema: code, DBSchema: db, AddTimestamps: []string{"updated_at", "deleted_at"}}

	actions, err := BuildModelActions(d)
	if err != nil {
		t.Fatalf("BuildModelActions: %v", err)
	}
	if len(actions) != 1 {
		t.Fatalf("actions = %#v, want exactly one", actions)
	}
	a := actions[0]
	if a.Type != ActionAddTimestamps || a.ModelName != "Widget" || a.AddTimestamps == nil {
		t.Fatalf("got %#v, want an addTimestamps action", a)
	}
	if got := a.AddTimestamps.Columns; len(got) != 2 || got[0] != "updated_at" || got[1] != "deleted_at" {
		t.Fatalf("Columns = %v, want [updated_at deleted_at]", got)
	}
	if !a.AddTimestamps.IndexExisted {
		t.Fatal("IndexExisted = false, want true (diff.AddTimestampsIndexMissing was false)")
	}
}

// TestBuildModelActions_AddTimestamps_IndexOnly checks that an
// AddTimestamps action is still emitted when every column already exists
// (diff.AddTimestamps is empty) but deleted_at's own index is still
// missing - a table can have adopted all three columns by hand and never
// had that index (see ModelDiff.AddTimestampsIndexMissing's doc).
func TestBuildModelActions_AddTimestamps_IndexOnly(t *testing.T) {
	code := &ModelSchema{Name: "Widget"}
	db := &ModelSchema{Name: "Widget"}
	d := ModelDiff{Name: "Widget", CodeSchema: code, DBSchema: db, AddTimestampsIndexMissing: true}

	actions, err := BuildModelActions(d)
	if err != nil {
		t.Fatalf("BuildModelActions: %v", err)
	}
	if len(actions) != 1 {
		t.Fatalf("actions = %#v, want exactly one", actions)
	}
	a := actions[0]
	if a.Type != ActionAddTimestamps || a.AddTimestamps == nil {
		t.Fatalf("got %#v, want an addTimestamps action", a)
	}
	if len(a.AddTimestamps.Columns) != 0 {
		t.Fatalf("Columns = %v, want none", a.AddTimestamps.Columns)
	}
	if a.AddTimestamps.IndexExisted {
		t.Fatal("IndexExisted = true, want false")
	}
}

// TestBuildModelActions_NoAddTimestamps_WhenNothingMissing checks that no
// action is emitted at all when Timestamps is on but both the columns and
// the index are already fully in place - the fully-adopted case.
func TestBuildModelActions_NoAddTimestamps_WhenNothingMissing(t *testing.T) {
	code := &ModelSchema{Name: "Widget"}
	db := &ModelSchema{Name: "Widget"}
	d := ModelDiff{Name: "Widget", CodeSchema: code, DBSchema: db}

	actions, err := BuildModelActions(d)
	if err != nil {
		t.Fatalf("BuildModelActions: %v", err)
	}
	if len(actions) != 0 {
		t.Fatalf("actions = %#v, want none", actions)
	}
}

// TestReorderCreateTableFirst_CrossModel checks the scenario that actually
// motivates it: two brand new models with a many-to-many relation between
// them - the AddRelation actions (which create a join table referencing
// both) must come after BOTH models' CreateTable actions, even though
// BuildModelActions only knows about one model's diff at a time and would,
// on its own, interleave CreateTable(A)+AddRelation(A) before
// CreateTable(B) ever runs.
func TestReorderCreateTableFirst_CrossModel(t *testing.T) {
	actions := []Action{
		{Type: ActionCreateTable, ModelName: "A", CreateTable: &ModelSchema{Name: "A"}},
		{Type: ActionAddRelation, ModelName: "A", AddRelation: &AddRelationAction{RelationName: "bs"}},
		{Type: ActionCreateTable, ModelName: "B", CreateTable: &ModelSchema{Name: "B"}},
	}
	got := reorderCreateTableFirst(actions)
	if len(got) != 3 {
		t.Fatalf("got %#v", got)
	}
	if got[0].Type != ActionCreateTable || got[0].ModelName != "A" {
		t.Fatalf("got[0] = %#v", got[0])
	}
	if got[1].Type != ActionCreateTable || got[1].ModelName != "B" {
		t.Fatalf("got[1] = %#v, want B's CreateTable moved ahead of A's AddRelation", got[1])
	}
	if got[2].Type != ActionAddRelation || got[2].ModelName != "A" {
		t.Fatalf("got[2] = %#v", got[2])
	}
}

// TestBuildModelActions_InconsistentDiffIsError checks that a hand-built
// ModelDiff whose Fields names a field CodeSchema/DBSchema doesn't actually
// have is rejected, rather than silently producing an action around a
// zero-value Field - ModelDiff is exported, so nothing stops a caller from
// building one that isn't self-consistent the way CompareModel's always is.
func TestBuildModelActions_InconsistentDiffIsError(t *testing.T) {
	code := &ModelSchema{Name: "Widget"}
	db := &ModelSchema{Name: "Widget"}
	d := ModelDiff{
		Name: "Widget", CodeSchema: code, DBSchema: db,
		Fields: FieldsDiff{Added: []string{"missing"}},
	}

	if _, err := BuildModelActions(d); err == nil {
		t.Fatal("expected an error for a field named in the diff but absent from CodeSchema")
	}
}

func TestAction_Inverse(t *testing.T) {
	tableSchema := &ModelSchema{Name: "Widget"}

	tests := []struct {
		name string
		a    Action
		want Action
	}{
		{
			"createTable",
			Action{Type: ActionCreateTable, ModelName: "Widget", CreateTable: tableSchema},
			Action{Type: ActionDropTable, ModelName: "Widget", DropTable: tableSchema},
		},
		{
			"dropTable",
			Action{Type: ActionDropTable, ModelName: "Widget", DropTable: tableSchema},
			Action{Type: ActionCreateTable, ModelName: "Widget", CreateTable: tableSchema},
		},
		{
			"addField",
			Action{Type: ActionAddField, ModelName: "Widget", AddField: &AddFieldAction{FieldName: "x", Definition: Field{Type: FieldTypeInt}}},
			Action{Type: ActionDelField, ModelName: "Widget", DelField: &DelFieldAction{FieldName: "x", Definition: Field{Type: FieldTypeInt}}},
		},
		{
			"delField",
			Action{Type: ActionDelField, ModelName: "Widget", DelField: &DelFieldAction{FieldName: "x", Definition: Field{Type: FieldTypeInt}}},
			Action{Type: ActionAddField, ModelName: "Widget", AddField: &AddFieldAction{FieldName: "x", Definition: Field{Type: FieldTypeInt}}},
		},
		{
			"renameField",
			Action{Type: ActionRenameField, ModelName: "Widget", RenameField: &RenameFieldAction{OldFieldName: "a", NewFieldName: "b"}},
			Action{Type: ActionRenameField, ModelName: "Widget", RenameField: &RenameFieldAction{OldFieldName: "b", NewFieldName: "a"}},
		},
		{
			"addRelation",
			Action{Type: ActionAddRelation, ModelName: "Widget", AddRelation: &AddRelationAction{RelationName: "x", Definition: Relation{Type: RelationTypeManyToOne, RelatedModel: "Gadget"}}},
			Action{Type: ActionDelRelation, ModelName: "Widget", DelRelation: &DelRelationAction{RelationName: "x", Definition: Relation{Type: RelationTypeManyToOne, RelatedModel: "Gadget"}}},
		},
		{
			"delRelation",
			Action{Type: ActionDelRelation, ModelName: "Widget", DelRelation: &DelRelationAction{RelationName: "x", Definition: Relation{Type: RelationTypeManyToOne, RelatedModel: "Gadget"}}},
			Action{Type: ActionAddRelation, ModelName: "Widget", AddRelation: &AddRelationAction{RelationName: "x", Definition: Relation{Type: RelationTypeManyToOne, RelatedModel: "Gadget"}}},
		},
		{
			"renameRelation",
			Action{Type: ActionRenameRelation, ModelName: "Widget", RenameRelation: &RenameRelationAction{OldRelationName: "a", NewRelationName: "b"}},
			Action{Type: ActionRenameRelation, ModelName: "Widget", RenameRelation: &RenameRelationAction{OldRelationName: "b", NewRelationName: "a"}},
		},
		{
			"addTimestamps",
			Action{Type: ActionAddTimestamps, ModelName: "Widget", AddTimestamps: &AddTimestampsAction{Columns: []string{"updated_at", "deleted_at"}, IndexExisted: true}},
			Action{Type: ActionDelTimestamps, ModelName: "Widget", DelTimestamps: &DelTimestampsAction{Columns: []string{"updated_at", "deleted_at"}, IndexExisted: true}},
		},
		{
			"delTimestamps",
			Action{Type: ActionDelTimestamps, ModelName: "Widget", DelTimestamps: &DelTimestampsAction{Columns: []string{"updated_at", "deleted_at"}, IndexExisted: true}},
			Action{Type: ActionAddTimestamps, ModelName: "Widget", AddTimestamps: &AddTimestampsAction{Columns: []string{"updated_at", "deleted_at"}, IndexExisted: true}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.a.Inverse()
			if err != nil {
				t.Fatalf("Inverse: %v", err)
			}
			if got.Type != tt.want.Type || got.ModelName != tt.want.ModelName {
				t.Fatalf("got %#v, want %#v", got, tt.want)
			}
			switch tt.want.Type {
			case ActionDropTable:
				if got.DropTable != tt.want.DropTable {
					t.Fatalf("DropTable = %v, want %v", got.DropTable, tt.want.DropTable)
				}
			case ActionCreateTable:
				if got.CreateTable != tt.want.CreateTable {
					t.Fatalf("CreateTable = %v, want %v", got.CreateTable, tt.want.CreateTable)
				}
			case ActionDelField:
				if *got.DelField != *tt.want.DelField {
					t.Fatalf("DelField = %#v, want %#v", got.DelField, tt.want.DelField)
				}
			case ActionAddField:
				if *got.AddField != *tt.want.AddField {
					t.Fatalf("AddField = %#v, want %#v", got.AddField, tt.want.AddField)
				}
			case ActionRenameField:
				if *got.RenameField != *tt.want.RenameField {
					t.Fatalf("RenameField = %#v, want %#v", got.RenameField, tt.want.RenameField)
				}
			case ActionDelRelation:
				if *got.DelRelation != *tt.want.DelRelation {
					t.Fatalf("DelRelation = %#v, want %#v", got.DelRelation, tt.want.DelRelation)
				}
			case ActionAddRelation:
				if *got.AddRelation != *tt.want.AddRelation {
					t.Fatalf("AddRelation = %#v, want %#v", got.AddRelation, tt.want.AddRelation)
				}
			case ActionRenameRelation:
				if *got.RenameRelation != *tt.want.RenameRelation {
					t.Fatalf("RenameRelation = %#v, want %#v", got.RenameRelation, tt.want.RenameRelation)
				}
			case ActionAddTimestamps:
				if got.AddTimestamps == nil || !reflect.DeepEqual(got.AddTimestamps, tt.want.AddTimestamps) {
					t.Fatalf("AddTimestamps = %#v, want %#v", got.AddTimestamps, tt.want.AddTimestamps)
				}
			case ActionDelTimestamps:
				if got.DelTimestamps == nil || !reflect.DeepEqual(got.DelTimestamps, tt.want.DelTimestamps) {
					t.Fatalf("DelTimestamps = %#v, want %#v", got.DelTimestamps, tt.want.DelTimestamps)
				}
			}
		})
	}
}

func TestAction_Inverse_ChangeFieldSwapsOldAndNew(t *testing.T) {
	a := Action{Type: ActionChangeField, ModelName: "Widget", ChangeField: &ChangeFieldAction{
		FieldName: "x", OldDefinition: Field{Type: FieldTypeInt}, NewDefinition: Field{Type: FieldTypeFloat},
	}}
	got, err := a.Inverse()
	if err != nil {
		t.Fatalf("Inverse: %v", err)
	}
	if got.ChangeField.FieldName != "x" || got.ChangeField.OldDefinition.Type != FieldTypeFloat || got.ChangeField.NewDefinition.Type != FieldTypeInt {
		t.Fatalf("got %#v", got.ChangeField)
	}
}

func TestAction_Inverse_ChangeRelationSwapsOldAndNew(t *testing.T) {
	a := Action{Type: ActionChangeRelation, ModelName: "Widget", ChangeRelation: &ChangeRelationAction{
		RelationName:  "x",
		OldDefinition: Relation{Type: RelationTypeManyToOne, RelatedModel: "Gadget", NoIndex: true},
		NewDefinition: Relation{Type: RelationTypeManyToOne, RelatedModel: "Gadget", NoIndex: false},
	}}
	got, err := a.Inverse()
	if err != nil {
		t.Fatalf("Inverse: %v", err)
	}
	if got.ChangeRelation.RelationName != "x" || got.ChangeRelation.OldDefinition.NoIndex || !got.ChangeRelation.NewDefinition.NoIndex {
		t.Fatalf("got %#v", got.ChangeRelation)
	}
}

func TestAction_Inverse_UnknownTypeIsError(t *testing.T) {
	if _, err := (Action{Type: "bogus"}).Inverse(); err == nil {
		t.Fatal("expected an error for an unknown action type")
	}
}

func TestAction_Inverse_NilPayloadIsError(t *testing.T) {
	if _, err := (Action{Type: ActionAddField}).Inverse(); err == nil {
		t.Fatal("expected an error for a nil addField payload")
	}
}

// TestAction_YAMLRoundTrip checks that the tagged-union shape (a Type
// discriminator plus one populated pointer field) actually round-trips
// through yaml as expected - each Action in the list keeps only its own
// action type's payload, the rest stay absent (omitempty), not present as
// null.
func TestAction_YAMLRoundTrip(t *testing.T) {
	original := migrationFile{
		Name: "add_and_rename",
		Type: MigrationType,
		Actions: []Action{
			{Type: ActionCreateTable, ModelName: "Widget", CreateTable: &ModelSchema{Name: "Widget"}},
			{Type: ActionAddField, ModelName: "Gadget", AddField: &AddFieldAction{FieldName: "x", Definition: Field{Type: FieldTypeInt}}},
			{Type: ActionRenameField, ModelName: "Gadget", RenameField: &RenameFieldAction{OldFieldName: "a", NewFieldName: "b"}},
		},
	}

	out, err := yaml.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded migrationFile
	if err := yaml.Unmarshal(out, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v\n%s", err, out)
	}

	if decoded.Type != MigrationType || decoded.Name != "add_and_rename" || len(decoded.Actions) != 3 {
		t.Fatalf("decoded = %#v", decoded)
	}
	if decoded.Actions[0].CreateTable == nil || decoded.Actions[0].AddField != nil || decoded.Actions[0].RenameField != nil {
		t.Fatalf("actions[0] = %#v, want only createTable set", decoded.Actions[0])
	}
	if decoded.Actions[1].AddField == nil || decoded.Actions[1].AddField.FieldName != "x" || decoded.Actions[1].CreateTable != nil {
		t.Fatalf("actions[1] = %#v", decoded.Actions[1])
	}
	if decoded.Actions[2].RenameField == nil || decoded.Actions[2].RenameField.OldFieldName != "a" || decoded.Actions[2].RenameField.NewFieldName != "b" {
		t.Fatalf("actions[2] = %#v", decoded.Actions[2])
	}
}
