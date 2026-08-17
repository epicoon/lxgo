//go:build integration

package model_test

import (
	"path/filepath"
	"testing"

	"github.com/epicoon/lxgo/migrator"
	"github.com/epicoon/lxgo/model"
)

// TestApplyInvert_CreateAndDropTable_NonPublicSchema checks the DDL path's
// most basic non-"public" case end to end: the model's own declared
// Namespace resolves through loadTestSchemas/GenerateMigration into
// Action.Namespace, and execCreateTable creates the schema itself (it
// doesn't exist yet at the start of the test) before the table - the same
// "CREATE SCHEMA IF NOT EXISTS" precedent lx_sys already follows.
func TestApplyInvert_CreateAndDropTable_NonPublicSchema(t *testing.T) {
	db := setupDB(t)
	const ns = "model_test_ns_ddl_create"
	if _, err := db.Exec("DROP SCHEMA IF EXISTS " + ns + " CASCADE"); err != nil {
		t.Fatalf("drop schema %s: %v", ns, err)
	}
	t.Cleanup(func() { db.Exec("DROP SCHEMA IF EXISTS " + ns + " CASCADE") })

	migrationsDir := t.TempDir()
	db.Exec("DROP TABLE IF EXISTS lx_sys.migrator")
	migrator.Init(migrator.Config{DB: db, MigrationsPath: migrationsDir})

	schemaDir := t.TempDir()
	sch := &model.ModelSchema{
		Name:      "ns_ddl_widgets",
		Namespace: ns,
		Fields: []model.NamedField{
			{Name: "name", Field: model.Field{Type: model.FieldTypeString, Size: 100, Required: true}},
		},
	}
	if err := sch.Save(filepath.Join(schemaDir, "ns_ddl_widgets.yaml")); err != nil {
		t.Fatalf("save schema: %v", err)
	}

	schemas := loadTestSchemas(t, schemaDir)
	if _, err := model.GenerateMigration(db, schemas, "create_ns_ddl_widgets"); err != nil {
		t.Fatalf("GenerateMigration: %v", err)
	}
	migrator.Up()

	introspected, err := model.IntrospectModelSchema(db, "ns_ddl_widgets", ns, false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema(ns) after Up: %v", err)
	}
	name, ok := introspected.FieldByName("name")
	if !ok || name.Type != model.FieldTypeString || name.Size != 100 {
		t.Fatalf("name after Up = %#v, %v", name, ok)
	}

	// The table must not also have landed in "public" - proves the
	// resolved namespace actually drove the physical CREATE TABLE, not
	// just accepted and ignored.
	if _, err := model.IntrospectModelSchema(db, "ns_ddl_widgets", "public", false); err != model.ErrTableNotFound {
		t.Fatalf("IntrospectModelSchema(public) err = %v, want ErrTableNotFound", err)
	}

	migrator.Down(0)

	if _, err := model.IntrospectModelSchema(db, "ns_ddl_widgets", ns, false); err != model.ErrTableNotFound {
		t.Fatalf("err after Down = %v, want ErrTableNotFound", err)
	}
}

// TestApplyInvert_AddChangeRenameDelField_NonPublicSchema exercises the
// field-level DDL family (ALTER TABLE ADD/ALTER/RENAME/DROP COLUMN) against
// a table that already exists in a non-"public" schema - unlike the create
// case above, ensureSchemaExists is never called on this path at all
// (nothing here creates a table), so this also checks that field DDL
// doesn't depend on it having run.
func TestApplyInvert_AddChangeRenameDelField_NonPublicSchema(t *testing.T) {
	db := setupDB(t)
	const ns = "model_test_ns_ddl_fields"
	if _, err := db.Exec("DROP SCHEMA IF EXISTS " + ns + " CASCADE"); err != nil {
		t.Fatalf("drop schema %s: %v", ns, err)
	}
	t.Cleanup(func() { db.Exec("DROP SCHEMA IF EXISTS " + ns + " CASCADE") })
	if _, err := db.Exec("CREATE SCHEMA " + ns); err != nil {
		t.Fatalf("create schema %s: %v", ns, err)
	}
	if _, err := db.Exec(`
		CREATE TABLE ` + ns + `.ns_ddl_gadgets (
			id serial PRIMARY KEY,
			name character varying(50) NOT NULL,
			old_count integer
		)
	`); err != nil {
		t.Fatalf("create %s.ns_ddl_gadgets: %v", ns, err)
	}

	migrationsDir := t.TempDir()
	db.Exec("DROP TABLE IF EXISTS lx_sys.migrator")
	migrator.Init(migrator.Config{DB: db, MigrationsPath: migrationsDir})

	schemaDir := t.TempDir()
	sch := &model.ModelSchema{
		Name:      "ns_ddl_gadgets",
		Namespace: ns,
		Fields: []model.NamedField{
			{Name: "name", Field: model.Field{Type: model.FieldTypeString, Size: 200, Required: true}},  // changed: 50 -> 200
			{Name: "new_count", Field: model.Field{Type: model.FieldTypeInt, RenamedFrom: "old_count"}}, // renamed
			{Name: "brand_new", Field: model.Field{Type: model.FieldTypeBool, Default: false}},          // added
		},
	}
	schemaPath := filepath.Join(schemaDir, "ns_ddl_gadgets.yaml")
	if err := sch.Save(schemaPath); err != nil {
		t.Fatalf("save schema: %v", err)
	}

	schemas := loadTestSchemas(t, schemaDir)
	if _, err := model.GenerateMigration(db, schemas, "change_ns_ddl_gadgets"); err != nil {
		t.Fatalf("GenerateMigration: %v", err)
	}
	migrator.Up()

	after, err := model.IntrospectModelSchema(db, "ns_ddl_gadgets", ns, false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema after Up: %v", err)
	}
	if name, ok := after.FieldByName("name"); !ok || name.Size != 200 {
		t.Fatalf("name after Up = %#v, %v, want Size 200", name, ok)
	}
	if _, ok := after.FieldByName("old_count"); ok {
		t.Fatal("old_count should be gone after rename")
	}
	if newCount, ok := after.FieldByName("new_count"); !ok || newCount.Type != model.FieldTypeInt {
		t.Fatalf("new_count after Up = %#v, %v", newCount, ok)
	}
	if _, ok := after.FieldByName("brand_new"); !ok {
		t.Fatal("expected brand_new after Up")
	}

	migrator.Down(0)

	before, err := model.IntrospectModelSchema(db, "ns_ddl_gadgets", ns, false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema after Down: %v", err)
	}
	if name, ok := before.FieldByName("name"); !ok || name.Size != 50 {
		t.Fatalf("name after Down = %#v, %v, want Size 50 (reverted)", name, ok)
	}
	if _, ok := before.FieldByName("old_count"); !ok {
		t.Fatal("old_count should be back after Down")
	}
	if _, ok := before.FieldByName("brand_new"); ok {
		t.Fatal("brand_new should have been removed by Down")
	}
}

// TestApplyInvert_AddRenameToggleDelRelation_ManyToOne_NonPublicSchema
// covers the manyToOne relation DDL family in one non-"public" schema:
// add (FK column + REFERENCES + index, all schema-qualified), rename (the
// riskiest part - ALTER INDEX RENAME needs its OLD name schema-qualified
// to resolve under a non-default search_path, but its NEW name must stay
// bare), index toggle (DROP INDEX/CREATE INDEX, same asymmetry), and
// delete - each step's Up is checked, and the whole chain is unwound with
// a single Down(0) at the end.
func TestApplyInvert_AddRenameToggleDelRelation_ManyToOne_NonPublicSchema(t *testing.T) {
	db := setupDB(t)
	const ns = "model_test_ns_ddl_rel"
	physOne := pgTableName("ns_ddl_rel_one")
	physMany := pgTableName("ns_ddl_rel_many")
	if _, err := db.Exec("DROP SCHEMA IF EXISTS " + ns + " CASCADE"); err != nil {
		t.Fatalf("drop schema %s: %v", ns, err)
	}
	t.Cleanup(func() {
		db.Exec("DROP SCHEMA IF EXISTS " + ns + " CASCADE")
		db.Exec(`DELETE FROM lx_sys.model_relations WHERE home_table IN ($1, $2)`, ns+"."+physMany, ns+"."+physOne)
	})

	migrationsDir := t.TempDir()
	db.Exec("DROP TABLE IF EXISTS lx_sys.migrator")
	migrator.Init(migrator.Config{DB: db, MigrationsPath: migrationsDir})

	schemaDir := t.TempDir()
	onePath := filepath.Join(schemaDir, "ns_ddl_rel_one.yaml")
	manyPath := filepath.Join(schemaDir, "ns_ddl_rel_many.yaml")

	one := &model.ModelSchema{Name: "ns_ddl_rel_one", Namespace: ns, Relations: []model.NamedRelation{
		{Name: "manys", Relation: model.Relation{Type: model.RelationTypeOneToMany, RelatedModel: "ns_ddl_rel_many", RelatedAttribute: "one"}},
	}}
	many := &model.ModelSchema{Name: "ns_ddl_rel_many", Namespace: ns, Relations: []model.NamedRelation{
		{Name: "one", Relation: model.Relation{Type: model.RelationTypeManyToOne, RelatedModel: "ns_ddl_rel_one", RelatedAttribute: "manys"}},
	}}
	if err := one.Save(onePath); err != nil {
		t.Fatalf("save one: %v", err)
	}
	if err := many.Save(manyPath); err != nil {
		t.Fatalf("save many: %v", err)
	}

	schemas := loadTestSchemas(t, schemaDir)
	if _, err := model.GenerateMigration(db, schemas, "create"); err != nil {
		t.Fatalf("GenerateMigration (create): %v", err)
	}
	migrator.Up()

	schema, err := model.IntrospectModelSchema(db, physMany, ns, false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema after create: %v", err)
	}
	rel, ok := schema.RelationByName("one")
	if !ok || rel.Type != model.RelationTypeManyToOne || rel.NoIndex {
		t.Fatalf("one after create = %#v, %v, want indexed manyToOne", rel, ok)
	}

	// Rename - exercises ALTER TABLE RENAME COLUMN/CONSTRAINT and ALTER
	// INDEX RENAME, all against the qualified table/old-index name.
	many.Relations[0].Name = "theOne"
	if err := many.Save(manyPath); err != nil {
		t.Fatalf("save renamed: %v", err)
	}
	one.Relations[0].RelatedAttribute = "theOne"
	if err := one.Save(onePath); err != nil {
		t.Fatalf("save back-reference: %v", err)
	}
	schemas = loadTestSchemas(t, schemaDir)
	if _, err := model.GenerateMigration(db, schemas, "rename"); err != nil {
		t.Fatalf("GenerateMigration (rename): %v", err)
	}
	migrator.Up()

	schema, err = model.IntrospectModelSchema(db, physMany, ns, false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema after rename: %v", err)
	}
	if _, ok := schema.RelationByName("one"); ok {
		t.Fatal("old relation name should be gone after rename")
	}
	rel, ok = schema.RelationByName("theOne")
	if !ok || rel.NoIndex {
		t.Fatalf("theOne after rename = %#v, %v, want still indexed", rel, ok)
	}

	// Index toggle - exercises DROP INDEX (qualified old index) followed
	// later by CREATE INDEX (qualified ON table, bare index name).
	many.Relations[0].Relation.NoIndex = true
	if err := many.Save(manyPath); err != nil {
		t.Fatalf("save no-index: %v", err)
	}
	schemas = loadTestSchemas(t, schemaDir)
	if _, err := model.GenerateMigration(db, schemas, "toggle"); err != nil {
		t.Fatalf("GenerateMigration (toggle): %v", err)
	}
	migrator.Up()

	schema, err = model.IntrospectModelSchema(db, physMany, ns, false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema after toggle: %v", err)
	}
	rel, ok = schema.RelationByName("theOne")
	if !ok || !rel.NoIndex {
		t.Fatalf("theOne after toggle = %#v, %v, want NoIndex true", rel, ok)
	}

	// Delete both sides.
	one.Relations = nil
	if err := one.Save(onePath); err != nil {
		t.Fatalf("save one without relation: %v", err)
	}
	many.Relations = nil
	if err := many.Save(manyPath); err != nil {
		t.Fatalf("save many without relation: %v", err)
	}
	schemas = loadTestSchemas(t, schemaDir)
	if _, err := model.GenerateMigration(db, schemas, "delete"); err != nil {
		t.Fatalf("GenerateMigration (delete): %v", err)
	}
	migrator.Up()

	schema, err = model.IntrospectModelSchema(db, physMany, ns, false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema after delete: %v", err)
	}
	if _, ok := schema.RelationByName("theOne"); ok {
		t.Fatal("relation should be gone after delete")
	}

	// Unwind all 4 migrations (create/rename/toggle/delete) in one shot -
	// Down's steps <= 0 only rolls back the single last migration, so this
	// needs the real count. Undoing "create" (a NeedsTable action, so its
	// own Inverse is a DropTable) removes the table entirely - unless every
	// step's Inverse() carried pgSchema/Namespace correctly, this instead
	// fails outright trying to resolve one of the qualified identifiers
	// along the way.
	migrator.Down(4)

	if _, err := model.IntrospectModelSchema(db, physMany, ns, false); err != model.ErrTableNotFound {
		t.Fatalf("err after full Down = %v, want ErrTableNotFound", err)
	}
}

// TestApplyInvert_AddDelRelation_ManyToMany_NonPublicSchema checks the
// many-to-many DDL family in a non-"public" schema: the join table itself
// is created via ensureSchemaExists + qualified idents (both FK columns
// referencing each side's own qualified table), and dropped again via the
// schema recorded in RelationFk.JoinTable (see execDelManyToManyRelation).
func TestApplyInvert_AddDelRelation_ManyToMany_NonPublicSchema(t *testing.T) {
	db := setupDB(t)
	const ns = "model_test_ns_ddl_m2m"
	physA := pgTableName("ns_ddl_m2m_a")
	physB := pgTableName("ns_ddl_m2m_b")
	if _, err := db.Exec("DROP SCHEMA IF EXISTS " + ns + " CASCADE"); err != nil {
		t.Fatalf("drop schema %s: %v", ns, err)
	}
	t.Cleanup(func() {
		db.Exec("DROP SCHEMA IF EXISTS " + ns + " CASCADE")
		db.Exec(`DELETE FROM lx_sys.model_relations WHERE home_table IN ($1, $2)`, ns+"."+physA, ns+"."+physB)
	})

	migrationsDir := t.TempDir()
	db.Exec("DROP TABLE IF EXISTS lx_sys.migrator")
	migrator.Init(migrator.Config{DB: db, MigrationsPath: migrationsDir})

	schemaDir := t.TempDir()
	a := &model.ModelSchema{Name: "ns_ddl_m2m_a", Namespace: ns, Relations: []model.NamedRelation{
		{Name: "bs", Relation: model.Relation{Type: model.RelationTypeManyToMany, RelatedModel: "ns_ddl_m2m_b", RelatedAttribute: "as"}},
	}}
	b := &model.ModelSchema{Name: "ns_ddl_m2m_b", Namespace: ns, Relations: []model.NamedRelation{
		{Name: "as", Relation: model.Relation{Type: model.RelationTypeManyToMany, RelatedModel: "ns_ddl_m2m_a", RelatedAttribute: "bs"}},
	}}
	if err := a.Save(filepath.Join(schemaDir, "ns_ddl_m2m_a.yaml")); err != nil {
		t.Fatalf("save A: %v", err)
	}
	if err := b.Save(filepath.Join(schemaDir, "ns_ddl_m2m_b.yaml")); err != nil {
		t.Fatalf("save B: %v", err)
	}

	schemas := loadTestSchemas(t, schemaDir)
	if _, err := model.GenerateMigration(db, schemas, "create"); err != nil {
		t.Fatalf("GenerateMigration: %v", err)
	}
	migrator.Up()

	schemaA, err := model.IntrospectModelSchema(db, physA, ns, false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema(A) after Up: %v", err)
	}
	relA, ok := schemaA.RelationByName("bs")
	if !ok || relA.Type != model.RelationTypeManyToMany || relA.RelatedModel != "ns_ddl_m2m_b" || relA.NoIndex {
		t.Fatalf("A.bs after Up = %#v, %v", relA, ok)
	}
	schemaB, err := model.IntrospectModelSchema(db, physB, ns, false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema(B) after Up: %v", err)
	}
	relB, ok := schemaB.RelationByName("as")
	if !ok || relB.Type != model.RelationTypeManyToMany || relB.NoIndex {
		t.Fatalf("B.as after Up = %#v, %v", relB, ok)
	}

	migrator.Down(0)

	if _, err := model.IntrospectModelSchema(db, physA, ns, false); err != model.ErrTableNotFound {
		t.Fatalf("A err after Down = %v, want ErrTableNotFound", err)
	}
	if _, err := model.IntrospectModelSchema(db, physB, ns, false); err != model.ErrTableNotFound {
		t.Fatalf("B err after Down = %v, want ErrTableNotFound", err)
	}
}
