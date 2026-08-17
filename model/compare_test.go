package model

import "testing"

func namedField(name string, f Field) NamedField {
	return NamedField{Name: name, Field: f}
}

func TestCompareFields_AddedAndDeleted(t *testing.T) {
	code := &ModelSchema{Fields: []NamedField{
		namedField("keep", Field{Type: FieldTypeString}),
		namedField("newOne", Field{Type: FieldTypeInt}),
	}}
	db := &ModelSchema{Fields: []NamedField{
		namedField("keep", Field{Type: FieldTypeString}),
		namedField("gone", Field{Type: FieldTypeBool}),
	}}

	diff := CompareFields(code, db)
	if len(diff.Added) != 1 || diff.Added[0] != "newOne" {
		t.Fatalf("Added = %v, want [newOne]", diff.Added)
	}
	if len(diff.Deleted) != 1 || diff.Deleted[0] != "gone" {
		t.Fatalf("Deleted = %v, want [gone]", diff.Deleted)
	}
	if len(diff.Changed) != 0 || len(diff.Renamed) != 0 {
		t.Fatalf("unexpected Changed/Renamed: %#v", diff)
	}
}

func TestCompareFields_Changed(t *testing.T) {
	code := &ModelSchema{Fields: []NamedField{
		namedField("size", Field{Type: FieldTypeString, Size: 255}),
	}}
	db := &ModelSchema{Fields: []NamedField{
		namedField("size", Field{Type: FieldTypeString, Size: 100}),
	}}

	diff := CompareFields(code, db)
	if len(diff.Changed) != 1 || diff.Changed[0] != "size" {
		t.Fatalf("Changed = %v, want [size]", diff.Changed)
	}
	if len(diff.Added) != 0 || len(diff.Deleted) != 0 || len(diff.Renamed) != 0 {
		t.Fatalf("unexpected Added/Deleted/Renamed: %#v", diff)
	}
}

func TestCompareFields_SameNameSameShapeIsNotChanged(t *testing.T) {
	code := &ModelSchema{Fields: []NamedField{
		namedField("name", Field{Type: FieldTypeString, Required: true}),
	}}
	db := &ModelSchema{Fields: []NamedField{
		namedField("name", Field{Type: FieldTypeString, Required: true}),
	}}

	diff := CompareFields(code, db)
	if !diff.IsEmpty() {
		t.Fatalf("expected an empty diff, got %#v", diff)
	}
}

func TestCompareFields_HeuristicRename(t *testing.T) {
	code := &ModelSchema{Fields: []NamedField{
		namedField("fullName", Field{Type: FieldTypeString, Size: 255}),
	}}
	db := &ModelSchema{Fields: []NamedField{
		namedField("name", Field{Type: FieldTypeString, Size: 255}),
	}}

	diff := CompareFields(code, db)
	if len(diff.Renamed) != 1 {
		t.Fatalf("Renamed = %#v, want exactly one entry", diff.Renamed)
	}
	r := diff.Renamed[0]
	if r.From != "name" || r.To != "fullName" || r.Explicit {
		t.Fatalf("got %#v, want {From: name, To: fullName, Explicit: false}", r)
	}
	if len(diff.Added) != 0 || len(diff.Deleted) != 0 || len(diff.Changed) != 0 {
		t.Fatalf("unexpected Added/Deleted/Changed: %#v", diff)
	}
}

// TestCompareFields_ExplicitRenameWinsEvenWhenShapeChanged is the case the
// heuristic alone can't handle: a field renamed *and* retyped in the same
// schema edit. Without RenamedFrom this would look like an unrelated
// delete+add (see CompareFields' explicit-first pass).
func TestCompareFields_ExplicitRenameWinsEvenWhenShapeChanged(t *testing.T) {
	code := &ModelSchema{Fields: []NamedField{
		namedField("fullName", Field{Type: FieldTypeString, Size: 500, RenamedFrom: "name"}),
	}}
	db := &ModelSchema{Fields: []NamedField{
		namedField("name", Field{Type: FieldTypeString, Size: 255}),
	}}

	diff := CompareFields(code, db)
	if len(diff.Renamed) != 1 {
		t.Fatalf("Renamed = %#v, want exactly one entry", diff.Renamed)
	}
	r := diff.Renamed[0]
	if r.From != "name" || r.To != "fullName" || !r.Explicit {
		t.Fatalf("got %#v, want {From: name, To: fullName, Explicit: true}", r)
	}
	if len(diff.Added) != 0 || len(diff.Deleted) != 0 || len(diff.Changed) != 0 {
		t.Fatalf("unexpected Added/Deleted/Changed: %#v", diff)
	}
}

// TestCompareFields_RenamedFromPointingNowhereFallsBackToAdded checks that
// a RenamedFrom naming a field that isn't actually db-only (typo, or the
// field is genuinely new) doesn't wrongly consume an unrelated db-only
// field - it just falls through as an ordinary Added (and whatever the
// stale db field was still shows up as Deleted, unless the heuristic pass
// happens to match it on shape).
func TestCompareFields_RenamedFromPointingNowhereFallsBackToAdded(t *testing.T) {
	code := &ModelSchema{Fields: []NamedField{
		namedField("fullName", Field{Type: FieldTypeString, RenamedFrom: "doesNotExist"}),
	}}
	db := &ModelSchema{Fields: []NamedField{
		namedField("unrelated", Field{Type: FieldTypeInt}),
	}}

	diff := CompareFields(code, db)
	if len(diff.Renamed) != 0 {
		t.Fatalf("Renamed = %#v, want none", diff.Renamed)
	}
	if len(diff.Added) != 1 || diff.Added[0] != "fullName" {
		t.Fatalf("Added = %v, want [fullName]", diff.Added)
	}
	if len(diff.Deleted) != 1 || diff.Deleted[0] != "unrelated" {
		t.Fatalf("Deleted = %v, want [unrelated]", diff.Deleted)
	}
}

// TestCompareFields_ExplicitRenamePreventsSpuriousHeuristicPairing checks
// that once an explicit rename consumes a db-only name, the heuristic pass
// can't also pair it with a different code-only field of the same shape.
func TestCompareFields_ExplicitRenamePreventsSpuriousHeuristicPairing(t *testing.T) {
	shape := Field{Type: FieldTypeString, Size: 255}
	explicitShape := shape
	explicitShape.RenamedFrom = "old"

	code := &ModelSchema{Fields: []NamedField{
		namedField("explicitNew", explicitShape),
		namedField("heuristicNew", shape),
	}}
	db := &ModelSchema{Fields: []NamedField{
		namedField("old", shape),
	}}

	diff := CompareFields(code, db)
	if len(diff.Renamed) != 1 || diff.Renamed[0].From != "old" || diff.Renamed[0].To != "explicitNew" || !diff.Renamed[0].Explicit {
		t.Fatalf("Renamed = %#v, want exactly the explicit old->explicitNew pair", diff.Renamed)
	}
	if len(diff.Added) != 1 || diff.Added[0] != "heuristicNew" {
		t.Fatalf("Added = %v, want [heuristicNew] (nothing left in db to pair it with)", diff.Added)
	}
}

func TestModelDiff_IsEmpty(t *testing.T) {
	if !(ModelDiff{}).IsEmpty() {
		t.Fatal("zero-value ModelDiff should be empty")
	}
	if (ModelDiff{NeedsTable: true}).IsEmpty() {
		t.Fatal("NeedsTable=true should not be empty")
	}
	if (ModelDiff{Fields: FieldsDiff{Added: []string{"x"}}}).IsEmpty() {
		t.Fatal("a non-empty Fields diff should not be empty")
	}
	if (ModelDiff{Relations: RelationsDiff{Added: []string{"x"}}}).IsEmpty() {
		t.Fatal("a non-empty Relations diff should not be empty")
	}
	if (ModelDiff{AddTimestamps: []string{"created_at"}}).IsEmpty() {
		t.Fatal("a non-empty AddTimestamps should not be empty")
	}
	if (ModelDiff{AddTimestampsIndexMissing: true}).IsEmpty() {
		t.Fatal("AddTimestampsIndexMissing=true should not be empty, even with AddTimestamps empty")
	}
}

func namedRelation(name string, r Relation) NamedRelation {
	return NamedRelation{Name: name, Relation: r}
}

func TestCompareRelations_AddedAndDeleted(t *testing.T) {
	code := []NamedRelation{
		namedRelation("keep", Relation{Type: RelationTypeManyToOne, RelatedModel: "A"}),
		namedRelation("newOne", Relation{Type: RelationTypeManyToOne, RelatedModel: "B"}),
	}
	db := []NamedRelation{
		namedRelation("keep", Relation{Type: RelationTypeManyToOne, RelatedModel: "A"}),
		namedRelation("gone", Relation{Type: RelationTypeManyToOne, RelatedModel: "C"}),
	}

	diff := CompareRelations(code, db)
	if len(diff.Added) != 1 || diff.Added[0] != "newOne" {
		t.Fatalf("Added = %v, want [newOne]", diff.Added)
	}
	if len(diff.Deleted) != 1 || diff.Deleted[0] != "gone" {
		t.Fatalf("Deleted = %v, want [gone]", diff.Deleted)
	}
	if len(diff.Changed) != 0 || len(diff.Renamed) != 0 {
		t.Fatalf("unexpected Changed/Renamed: %#v", diff)
	}
}

func TestCompareRelations_Changed(t *testing.T) {
	code := []NamedRelation{namedRelation("client", Relation{Type: RelationTypeManyToOne, RelatedModel: "Client"})}
	db := []NamedRelation{namedRelation("client", Relation{Type: RelationTypeManyToOne, RelatedModel: "Customer"})}

	diff := CompareRelations(code, db)
	if len(diff.Changed) != 1 || diff.Changed[0] != "client" {
		t.Fatalf("Changed = %v, want [client]", diff.Changed)
	}
	if len(diff.Added) != 0 || len(diff.Deleted) != 0 || len(diff.Renamed) != 0 {
		t.Fatalf("unexpected Added/Deleted/Renamed: %#v", diff)
	}
}

func TestCompareRelations_SameNameSameShapeIsNotChanged(t *testing.T) {
	code := []NamedRelation{namedRelation("client", Relation{Type: RelationTypeManyToOne, RelatedModel: "Client", RelatedAttribute: "orders"})}
	db := []NamedRelation{namedRelation("client", Relation{Type: RelationTypeManyToOne, RelatedModel: "Client", RelatedAttribute: "orders"})}

	diff := CompareRelations(code, db)
	if !diff.IsEmpty() {
		t.Fatalf("expected an empty diff, got %#v", diff)
	}
}

func TestCompareRelations_HeuristicRename(t *testing.T) {
	code := []NamedRelation{namedRelation("owner", Relation{Type: RelationTypeManyToOne, RelatedModel: "User"})}
	db := []NamedRelation{namedRelation("creator", Relation{Type: RelationTypeManyToOne, RelatedModel: "User"})}

	diff := CompareRelations(code, db)
	if len(diff.Renamed) != 1 || diff.Renamed[0].From != "creator" || diff.Renamed[0].To != "owner" {
		t.Fatalf("Renamed = %#v, want [{From: creator, To: owner}]", diff.Renamed)
	}
	if len(diff.Added) != 0 || len(diff.Deleted) != 0 || len(diff.Changed) != 0 {
		t.Fatalf("unexpected Added/Deleted/Changed: %#v", diff)
	}
}

func TestCanIgnoreRelation(t *testing.T) {
	cases := []struct {
		name          string
		modelName     string
		attributeName string
		r             Relation
		want          bool
	}{
		{"oneToMany always ignored", "A", "manys", Relation{Type: RelationTypeOneToMany}, true},
		{"manyToOne never ignored", "A", "one", Relation{Type: RelationTypeManyToOne}, false},
		{"oneToOne fkHolder acts", "A", "b", Relation{Type: RelationTypeOneToOne, FkHolder: true}, false},
		{"oneToOne non-holder ignored", "A", "b", Relation{Type: RelationTypeOneToOne, FkHolder: false}, true},
		{"manyToMany alphabetically-first acts", "A", "bs", Relation{Type: RelationTypeManyToMany, RelatedModel: "B"}, false},
		{"manyToMany alphabetically-later ignored", "B", "as", Relation{Type: RelationTypeManyToMany, RelatedModel: "A"}, true},
		// Self-referential (same model on both sides) - the model name
		// can't break the tie, the attribute name pair does instead.
		{"manyToMany self-referential lexicographically-first acts", "User", "friendOf",
			Relation{Type: RelationTypeManyToMany, RelatedModel: "User", RelatedAttribute: "friends"}, false},
		{"manyToMany self-referential lexicographically-later ignored", "User", "friends",
			Relation{Type: RelationTypeManyToMany, RelatedModel: "User", RelatedAttribute: "friendOf"}, true},
		// A relation that mirrors itself under one single attribute name
		// (e.g. a symmetric "siblings" link) always acts - there's only one
		// such declaration to begin with, so it must never be filtered out.
		{"manyToMany self-referential same attribute name always acts", "User", "siblings",
			Relation{Type: RelationTypeManyToMany, RelatedModel: "User", RelatedAttribute: "siblings"}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := canIgnoreRelation(c.modelName, c.attributeName, c.r); got != c.want {
				t.Errorf("canIgnoreRelation(%q, %q, %#v) = %v, want %v", c.modelName, c.attributeName, c.r, got, c.want)
			}
		})
	}
}

func TestFilterIgnorableRelations(t *testing.T) {
	schema := &ModelSchema{
		Name: "A",
		Relations: []NamedRelation{
			namedRelation("acting", Relation{Type: RelationTypeManyToOne, RelatedModel: "B"}),
			namedRelation("passive", Relation{Type: RelationTypeOneToMany, RelatedModel: "B"}),
		},
	}
	got := filterIgnorableRelations([]string{"acting", "passive"}, schema, "A")
	if len(got) != 1 || got[0] != "acting" {
		t.Fatalf("got %v, want [acting]", got)
	}
}

// TestPgSchemaProjection_TranslatesNamesAndRecoversRenamedFrom checks that
// pgSchemaProjection translates a schema's Name/Fields (and RenamedFrom, if
// set) to their physical equivalents, and that the returned map resolves
// both a current field's own physical name and - for a field declaring
// RenamedFrom - its old logical name, keyed by that name's own physical
// translation (see CompareModel's use of it for a rename's old side).
func TestPgSchemaProjection_TranslatesNamesAndRecoversRenamedFrom(t *testing.T) {
	code := &ModelSchema{
		Name: "WidgetCopy",
		Fields: []NamedField{
			namedField("NameCopy", Field{Type: FieldTypeString}),
			namedField("NewName", Field{Type: FieldTypeString, RenamedFrom: "OldName"}),
		},
	}

	proj, physToLogical := pgSchemaProjection(code)

	if proj.Name != "widget_copies" {
		t.Fatalf("proj.Name = %q, want %q", proj.Name, "widget_copies")
	}
	if len(proj.Fields) != 2 || proj.Fields[0].Name != "name_copy" || proj.Fields[1].Name != "new_name" {
		t.Fatalf("proj.Fields = %#v", proj.Fields)
	}
	if proj.Fields[1].RenamedFrom != "old_name" {
		t.Fatalf("proj.Fields[1].RenamedFrom = %q, want %q", proj.Fields[1].RenamedFrom, "old_name")
	}
	if physToLogical["name_copy"] != "NameCopy" || physToLogical["new_name"] != "NewName" {
		t.Fatalf("physToLogical missing current fields: %#v", physToLogical)
	}
	if physToLogical["old_name"] != "OldName" {
		t.Fatalf("physToLogical[%q] = %q, want %q (recovered from RenamedFrom)", "old_name", physToLogical["old_name"], "OldName")
	}
}

// TestCompareModelPipeline_MixedAddedChangedRenamedDeleted exercises the
// same sequence CompareModel runs (pgSchemaProjection -> CompareFields ->
// logicalizeFieldsDiff/logicalizeDBSchema) without a database, covering one
// of each outcome in a single schema: a field present in both sides
// (unchanged, matched at the physical level), a new code-only field
// (Added), an explicit rename (old side recovered to its logical name) and
// a column with no code-side declaration at all (Deleted, no logical name
// to recover - physical is the only name it ever had).
func TestCompareModelPipeline_MixedAddedChangedRenamedDeleted(t *testing.T) {
	code := &ModelSchema{
		Name: "WidgetCopy",
		Fields: []NamedField{
			namedField("NameCopy", Field{Type: FieldTypeString}),
			namedField("Description", Field{Type: FieldTypeString, Required: true}),
			namedField("NewName", Field{Type: FieldTypeString, RenamedFrom: "OldName"}),
		},
	}
	// dbSchema simulates IntrospectModelSchema's result - physical names,
	// as they'd actually be read from information_schema.
	db := &ModelSchema{
		Name: "widget_copies",
		Fields: []NamedField{
			namedField("name_copy", Field{Type: FieldTypeString}),
			namedField("old_name", Field{Type: FieldTypeString}),
			namedField("junk_column", Field{Type: FieldTypeString}),
		},
	}

	physSchema, physToLogical := pgSchemaProjection(code)
	diff := logicalizeFieldsDiff(CompareFields(physSchema, db), physToLogical)
	logicalDB := logicalizeDBSchema(db, code.Name, physToLogical)

	if len(diff.Added) != 1 || diff.Added[0] != "Description" {
		t.Fatalf("Added = %v, want [Description]", diff.Added)
	}
	if len(diff.Changed) != 0 {
		t.Fatalf("Changed = %v, want none (name_copy/NameCopy match and are IsEqual)", diff.Changed)
	}
	if len(diff.Deleted) != 1 || diff.Deleted[0] != "junk_column" {
		t.Fatalf("Deleted = %v, want [junk_column] (no code declaration ever existed for it)", diff.Deleted)
	}
	if len(diff.Renamed) != 1 || diff.Renamed[0].From != "OldName" || diff.Renamed[0].To != "NewName" || !diff.Renamed[0].Explicit {
		t.Fatalf("Renamed = %#v, want [{From: OldName, To: NewName, Explicit: true}]", diff.Renamed)
	}

	if logicalDB.Name != "WidgetCopy" {
		t.Fatalf("logicalDB.Name = %q, want %q", logicalDB.Name, "WidgetCopy")
	}
	if _, ok := logicalDB.FieldByName("NameCopy"); !ok {
		t.Fatal("logicalDB should expose the unchanged field under its logical name NameCopy")
	}
	if _, ok := logicalDB.FieldByName("OldName"); !ok {
		t.Fatal("logicalDB should expose the rename's old side under its recovered logical name OldName")
	}
	if _, ok := logicalDB.FieldByName("junk_column"); !ok {
		t.Fatal("logicalDB should keep junk_column under its physical name - no logical name exists for it")
	}
}
