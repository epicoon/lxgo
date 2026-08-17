//go:build integration

package model_test

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/epicoon/lxgo/migrator"
	"github.com/epicoon/lxgo/model"
	"gopkg.in/yaml.v3"
)

// testMigrationFile mirrors this package's own unexported migrationFile -
// can't reference that directly from this external model_test package, so
// tests that need to hand-build a migration's raw content for model.Apply
// (rather than going through GenerateMigration) use this instead.
type testMigrationFile struct {
	Name    string         `yaml:"Name"`
	Type    string         `yaml:"Type"`
	Actions []model.Action `yaml:"Actions"`
}

// TestApply_ChangeRelation_ActingSideFlipRejected checks execChangeRelation's
// guard against an edit that would move which side holds the physical
// foreign key (see ChangeRelationAction's doc) - needs no real tables or
// data at all, the check is a pure function of the two Relation values
// (canIgnoreRelation), evaluated before any DDL runs.
//
// OldDefinition/NewDefinition are picked so that both, on their own, are a
// type execDelRelation/execAddRelation would otherwise accept
// (RelationTypeOneToOne and RelationTypeManyToOne both dispatch to the
// to-one path) - a passive RelationTypeOneToMany old side would instead be
// rejected earlier by execDelRelation's own dispatch ("can not be deleted
// directly"), which would make this test pass even without the guard.
func TestApply_ChangeRelation_ActingSideFlipRejected(t *testing.T) {
	db := setupDB(t)
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	defer tx.Rollback()

	action := model.Action{
		Type: model.ActionChangeRelation, ModelName: "B",
		ChangeRelation: &model.ChangeRelationAction{
			RelationName:  "a",
			OldDefinition: model.Relation{Type: model.RelationTypeOneToOne, RelatedModel: "A", RelatedAttribute: "b", FkHolder: false},
			NewDefinition: model.Relation{Type: model.RelationTypeManyToOne, RelatedModel: "A", RelatedAttribute: "b"},
		},
	}
	raw, err := yaml.Marshal(testMigrationFile{Name: "test", Type: model.MigrationType, Actions: []model.Action{action}})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	err = model.Apply(tx, raw)
	if err == nil {
		t.Fatal("expected an error for a change that moves which side holds the physical foreign key")
	}
	if !strings.Contains(err.Error(), "moves which side holds the physical foreign key") {
		t.Fatalf("Apply err = %v, want the acting-side-flip guard's own error, not some other failure (e.g. a DROP COLUMN against a column/table that was never created)", err)
	}
}

func TestApplyInvert_AddDelRelation_ManyToOne(t *testing.T) {
	db := setupDB(t)
	physOne := pgTableName("ApplyRelOne")
	physMany := pgTableName("ApplyRelMany")
	for _, tbl := range []string{physMany, physOne} {
		db.Exec("DROP TABLE IF EXISTS " + tbl)
	}
	t.Cleanup(func() {
		db.Exec("DROP TABLE IF EXISTS " + physMany)
		db.Exec("DROP TABLE IF EXISTS " + physOne)
		db.Exec(`DELETE FROM lx_sys.model_relations WHERE home_table IN ($1, $2)`, "public."+physMany, "public."+physOne)
	})

	migrationsDir := t.TempDir()
	db.Exec("DROP TABLE IF EXISTS lx_sys.migrator")
	migrator.Init(migrator.Config{DB: db, MigrationsPath: migrationsDir})

	schemaDir := t.TempDir()
	one := &model.ModelSchema{Name: "ApplyRelOne", Relations: []model.NamedRelation{
		{Name: "manys", Relation: model.Relation{Type: model.RelationTypeOneToMany, RelatedModel: "ApplyRelMany", RelatedAttribute: "one"}},
	}}
	many := &model.ModelSchema{Name: "ApplyRelMany", Relations: []model.NamedRelation{
		{Name: "one", Relation: model.Relation{Type: model.RelationTypeManyToOne, RelatedModel: "ApplyRelOne", RelatedAttribute: "manys"}},
	}}
	if err := one.Save(filepath.Join(schemaDir, "ApplyRelOne.yaml")); err != nil {
		t.Fatalf("save ApplyRelOne: %v", err)
	}
	if err := many.Save(filepath.Join(schemaDir, "ApplyRelMany.yaml")); err != nil {
		t.Fatalf("save ApplyRelMany: %v", err)
	}

	schemas := loadTestSchemas(t, schemaDir)
	if _, err := model.GenerateMigration(db, schemas, "create_apply_rel_many_to_one"); err != nil {
		t.Fatalf("GenerateMigration: %v", err)
	}
	migrator.Up()

	schema, err := model.IntrospectModelSchema(db, physMany, "public", false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema after Up: %v", err)
	}
	if _, ok := schema.FieldByName("one_id"); ok {
		t.Fatal("the FK column should not appear in Fields")
	}
	rel, ok := schema.RelationByName("one")
	if !ok || rel.Type != model.RelationTypeManyToOne || rel.RelatedModel != "ApplyRelOne" || rel.RelatedAttribute != "manys" || rel.NoIndex {
		t.Fatalf("one after Up = %#v, %v", rel, ok)
	}

	oneSchema, err := model.IntrospectModelSchema(db, physOne, "public", false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema(one) after Up: %v", err)
	}
	manysRel, ok := oneSchema.RelationByName("manys")
	if !ok || manysRel.Type != model.RelationTypeOneToMany || manysRel.RelatedModel != "ApplyRelMany" || manysRel.RelatedAttribute != "one" {
		t.Fatalf("manys after Up = %#v, %v", manysRel, ok)
	}

	migrator.Down(0)

	_, err = model.IntrospectModelSchema(db, physMany, "public", false)
	if err != model.ErrTableNotFound {
		t.Fatalf("ApplyRelMany err after Down = %v, want ErrTableNotFound", err)
	}
	_, err = model.IntrospectModelSchema(db, physOne, "public", false)
	if err != model.ErrTableNotFound {
		t.Fatalf("ApplyRelOne err after Down = %v, want ErrTableNotFound", err)
	}
}

// TestApplyInvert_AddDelRelation_OneToOne also checks that the UNIQUE
// constraint AddRelation adds for RelationTypeOneToOne is physically
// enforced, not just reported by introspection.
func TestApplyInvert_AddDelRelation_OneToOne(t *testing.T) {
	db := setupDB(t)
	physA := pgTableName("ApplyRelOneToOneA")
	physB := pgTableName("ApplyRelOneToOneB")
	for _, tbl := range []string{physA, physB} {
		db.Exec("DROP TABLE IF EXISTS " + tbl)
	}
	t.Cleanup(func() {
		db.Exec("DROP TABLE IF EXISTS " + physA)
		db.Exec("DROP TABLE IF EXISTS " + physB)
		db.Exec(`DELETE FROM lx_sys.model_relations WHERE home_table IN ($1, $2)`, "public."+physA, "public."+physB)
	})

	migrationsDir := t.TempDir()
	db.Exec("DROP TABLE IF EXISTS lx_sys.migrator")
	migrator.Init(migrator.Config{DB: db, MigrationsPath: migrationsDir})

	schemaDir := t.TempDir()
	a := &model.ModelSchema{Name: "ApplyRelOneToOneA", Relations: []model.NamedRelation{
		{Name: "b", Relation: model.Relation{Type: model.RelationTypeOneToOne, RelatedModel: "ApplyRelOneToOneB", RelatedAttribute: "a", FkHolder: true}},
	}}
	b := &model.ModelSchema{Name: "ApplyRelOneToOneB", Relations: []model.NamedRelation{
		{Name: "a", Relation: model.Relation{Type: model.RelationTypeOneToOne, RelatedModel: "ApplyRelOneToOneA", RelatedAttribute: "b", FkHolder: false}},
	}}
	if err := a.Save(filepath.Join(schemaDir, "ApplyRelOneToOneA.yaml")); err != nil {
		t.Fatalf("save A: %v", err)
	}
	if err := b.Save(filepath.Join(schemaDir, "ApplyRelOneToOneB.yaml")); err != nil {
		t.Fatalf("save B: %v", err)
	}

	schemas := loadTestSchemas(t, schemaDir)
	if _, err := model.GenerateMigration(db, schemas, "create_apply_rel_1to1"); err != nil {
		t.Fatalf("GenerateMigration: %v", err)
	}
	migrator.Up()

	schemaA, err := model.IntrospectModelSchema(db, physA, "public", false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema(A) after Up: %v", err)
	}
	relA, ok := schemaA.RelationByName("b")
	if !ok || relA.Type != model.RelationTypeOneToOne || !relA.FkHolder || relA.NoIndex {
		t.Fatalf("A.b after Up = %#v, %v", relA, ok)
	}

	var bID1 int
	if err := db.QueryRow(`INSERT INTO ` + physB + ` DEFAULT VALUES RETURNING id`).Scan(&bID1); err != nil {
		t.Fatalf("insert b1: %v", err)
	}
	if err := db.QueryRow(`INSERT INTO ` + physB + ` DEFAULT VALUES RETURNING id`).Scan(new(int)); err != nil {
		t.Fatalf("insert b2: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO `+physA+` (b_id) VALUES ($1)`, bID1); err != nil {
		t.Fatalf("insert a pointing at b1: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO `+physA+` (b_id) VALUES ($1)`, bID1); err == nil {
		t.Fatal("expected a UNIQUE violation inserting a second row pointing at the same b")
	}

	migrator.Down(0)

	_, err = model.IntrospectModelSchema(db, physA, "public", false)
	if err != model.ErrTableNotFound {
		t.Fatalf("A err after Down = %v, want ErrTableNotFound", err)
	}
}

func TestApplyInvert_AddDelRelation_ManyToMany(t *testing.T) {
	db := setupDB(t)
	physA := pgTableName("ApplyRelM2mA")
	physB := pgTableName("ApplyRelM2mB")
	for _, tbl := range []string{physA, physB} {
		db.Exec("DROP TABLE IF EXISTS " + tbl)
	}
	t.Cleanup(func() {
		db.Exec("DROP TABLE IF EXISTS " + physA)
		db.Exec("DROP TABLE IF EXISTS " + physB)
		db.Exec(`DELETE FROM lx_sys.model_relations WHERE home_table IN ($1, $2)`, "public."+physA, "public."+physB)
	})

	migrationsDir := t.TempDir()
	db.Exec("DROP TABLE IF EXISTS lx_sys.migrator")
	migrator.Init(migrator.Config{DB: db, MigrationsPath: migrationsDir})

	schemaDir := t.TempDir()
	a := &model.ModelSchema{Name: "ApplyRelM2mA", Relations: []model.NamedRelation{
		{Name: "bs", Relation: model.Relation{Type: model.RelationTypeManyToMany, RelatedModel: "ApplyRelM2mB", RelatedAttribute: "as"}},
	}}
	b := &model.ModelSchema{Name: "ApplyRelM2mB", Relations: []model.NamedRelation{
		{Name: "as", Relation: model.Relation{Type: model.RelationTypeManyToMany, RelatedModel: "ApplyRelM2mA", RelatedAttribute: "bs"}},
	}}
	if err := a.Save(filepath.Join(schemaDir, "ApplyRelM2mA.yaml")); err != nil {
		t.Fatalf("save A: %v", err)
	}
	if err := b.Save(filepath.Join(schemaDir, "ApplyRelM2mB.yaml")); err != nil {
		t.Fatalf("save B: %v", err)
	}

	schemas := loadTestSchemas(t, schemaDir)
	if _, err := model.GenerateMigration(db, schemas, "create_apply_rel_m2m"); err != nil {
		t.Fatalf("GenerateMigration: %v", err)
	}
	migrator.Up()

	schemaA, err := model.IntrospectModelSchema(db, physA, "public", false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema(A) after Up: %v", err)
	}
	relA, ok := schemaA.RelationByName("bs")
	if !ok || relA.Type != model.RelationTypeManyToMany || relA.RelatedModel != "ApplyRelM2mB" || relA.RelatedAttribute != "as" || relA.NoIndex {
		t.Fatalf("A.bs after Up = %#v, %v", relA, ok)
	}
	schemaB, err := model.IntrospectModelSchema(db, physB, "public", false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema(B) after Up: %v", err)
	}
	relB, ok := schemaB.RelationByName("as")
	if !ok || relB.Type != model.RelationTypeManyToMany || relB.RelatedModel != "ApplyRelM2mA" || relB.RelatedAttribute != "bs" || relB.NoIndex {
		t.Fatalf("B.as after Up = %#v, %v", relB, ok)
	}

	migrator.Down(0)

	_, err = model.IntrospectModelSchema(db, physA, "public", false)
	if err != model.ErrTableNotFound {
		t.Fatalf("A err after Down = %v, want ErrTableNotFound", err)
	}
	_, err = model.IntrospectModelSchema(db, physB, "public", false)
	if err != model.ErrTableNotFound {
		t.Fatalf("B err after Down = %v, want ErrTableNotFound", err)
	}
}

// TestApplyInvert_AddDelRelation_ManyToMany_SelfReferential checks a
// many-to-many relation declared between a model and itself under two
// distinct attribute names (e.g. "friends"/"friendOf") - both
// pgJoinColumnName (the join table's two columns must not collide just
// because they reference the same table) and canIgnoreRelation (exactly
// one of the two attribute names must act, or the relation would get two
// separate physical join tables instead of one) have to handle this
// correctly for the migration to produce a single, working join table.
func TestApplyInvert_AddDelRelation_ManyToMany_SelfReferential(t *testing.T) {
	db := setupDB(t)
	phys := pgTableName("ApplyRelM2mSelfUser")
	db.Exec("DROP TABLE IF EXISTS " + phys)
	t.Cleanup(func() {
		db.Exec("DROP TABLE IF EXISTS " + phys)
		db.Exec(`DELETE FROM lx_sys.model_relations WHERE home_table = $1`, "public."+phys)
	})

	migrationsDir := t.TempDir()
	db.Exec("DROP TABLE IF EXISTS lx_sys.migrator")
	migrator.Init(migrator.Config{DB: db, MigrationsPath: migrationsDir})

	schemaDir := t.TempDir()
	u := &model.ModelSchema{Name: "ApplyRelM2mSelfUser", Relations: []model.NamedRelation{
		{Name: "friends", Relation: model.Relation{Type: model.RelationTypeManyToMany, RelatedModel: "ApplyRelM2mSelfUser", RelatedAttribute: "friendOf"}},
		{Name: "friendOf", Relation: model.Relation{Type: model.RelationTypeManyToMany, RelatedModel: "ApplyRelM2mSelfUser", RelatedAttribute: "friends"}},
	}}
	if err := u.Save(filepath.Join(schemaDir, "ApplyRelM2mSelfUser.yaml")); err != nil {
		t.Fatalf("save: %v", err)
	}

	schemas := loadTestSchemas(t, schemaDir)
	if _, err := model.GenerateMigration(db, schemas, "create_apply_rel_m2m_self"); err != nil {
		t.Fatalf("GenerateMigration: %v", err)
	}
	migrator.Up()

	schema, err := model.IntrospectModelSchema(db, phys, "public", false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema: %v", err)
	}
	friends, ok := schema.RelationByName("friends")
	if !ok || friends.Type != model.RelationTypeManyToMany || friends.RelatedModel != "ApplyRelM2mSelfUser" || friends.RelatedAttribute != "friendOf" {
		t.Fatalf("friends = %#v, %v", friends, ok)
	}
	friendOf, ok := schema.RelationByName("friendOf")
	if !ok || friendOf.Type != model.RelationTypeManyToMany || friendOf.RelatedModel != "ApplyRelM2mSelfUser" || friendOf.RelatedAttribute != "friends" {
		t.Fatalf("friendOf = %#v, %v", friendOf, ok)
	}

	var aID, bID int
	if err := db.QueryRow(`INSERT INTO ` + phys + ` DEFAULT VALUES RETURNING id`).Scan(&aID); err != nil {
		t.Fatalf("insert a: %v", err)
	}
	if err := db.QueryRow(`INSERT INTO ` + phys + ` DEFAULT VALUES RETURNING id`).Scan(&bID); err != nil {
		t.Fatalf("insert b: %v", err)
	}

	// There must be exactly one physical join table for this relation, not
	// one per attribute name - and its two FK columns, found here by their
	// own physical name rather than assumed, must be distinct.
	joinTableRows, err := db.Query(`SELECT table_name FROM information_schema.tables WHERE table_name LIKE 'rel__apply_rel_m2m_self%'`)
	if err != nil {
		t.Fatalf("querying join tables: %v", err)
	}
	var joinTables []string
	for joinTableRows.Next() {
		var n string
		if err := joinTableRows.Scan(&n); err != nil {
			t.Fatalf("scanning join table name: %v", err)
		}
		joinTables = append(joinTables, n)
	}
	joinTableRows.Close()
	if len(joinTables) != 1 {
		t.Fatalf("join tables = %v, want exactly 1", joinTables)
	}
	joinTable := joinTables[0]

	colRows, err := db.Query(`SELECT column_name FROM information_schema.columns WHERE table_name = $1 ORDER BY ordinal_position`, joinTable)
	if err != nil {
		t.Fatalf("querying join table columns: %v", err)
	}
	var cols []string
	for colRows.Next() {
		var c string
		if err := colRows.Scan(&c); err != nil {
			t.Fatalf("scanning column name: %v", err)
		}
		cols = append(cols, c)
	}
	colRows.Close()
	if len(cols) != 2 || cols[0] == cols[1] {
		t.Fatalf("join table %q columns = %v, want exactly 2 distinct columns", joinTable, cols)
	}

	insertQuery := fmt.Sprintf(`INSERT INTO %q (%q, %q) VALUES ($1, $2)`, joinTable, cols[0], cols[1])
	if _, err := db.Exec(insertQuery, aID, bID); err != nil {
		t.Fatalf("insert into join table: %v", err)
	}

	migrator.Down(0)

	_, err = model.IntrospectModelSchema(db, phys, "public", false)
	if err != model.ErrTableNotFound {
		t.Fatalf("err after Down = %v, want ErrTableNotFound", err)
	}
	var joinTableCount int
	if err := db.QueryRow(`SELECT count(*) FROM information_schema.tables WHERE table_name LIKE 'rel__apply_rel_m2m_self_users%'`).Scan(&joinTableCount); err != nil {
		t.Fatalf("counting leftover join tables: %v", err)
	}
	if joinTableCount != 0 {
		t.Fatalf("join tables left after Down = %d, want 0", joinTableCount)
	}
}

// TestApplyInvert_AddRelation_ManyToMany_PerSideNoIndex checks that each
// side's own no-index declaration ends up on that side's own join-table
// column specifically - exercising GenerateMigration's cross-model
// attachRelatedRelationInfo plumbing (only one side's diff actually
// builds the AddRelation action, see canIgnoreRelation, so the other
// side's own NoIndex has to be found separately).
func TestApplyInvert_AddRelation_ManyToMany_PerSideNoIndex(t *testing.T) {
	db := setupDB(t)
	physA := pgTableName("ApplyRelM2mIdxA")
	physB := pgTableName("ApplyRelM2mIdxB")
	for _, tbl := range []string{physA, physB} {
		db.Exec("DROP TABLE IF EXISTS " + tbl)
	}
	t.Cleanup(func() {
		db.Exec("DROP TABLE IF EXISTS " + physA)
		db.Exec("DROP TABLE IF EXISTS " + physB)
		db.Exec(`DELETE FROM lx_sys.model_relations WHERE home_table IN ($1, $2)`, "public."+physA, "public."+physB)
	})

	migrationsDir := t.TempDir()
	db.Exec("DROP TABLE IF EXISTS lx_sys.migrator")
	migrator.Init(migrator.Config{DB: db, MigrationsPath: migrationsDir})

	schemaDir := t.TempDir()
	// "ApplyRelM2mIdxA" < "ApplyRelM2mIdxB" - A is the acting side (see
	// canIgnoreRelation), so this also exercises that the OTHER
	// (non-acting) side's own no-index declaration still takes effect.
	a := &model.ModelSchema{Name: "ApplyRelM2mIdxA", Relations: []model.NamedRelation{
		{Name: "bs", Relation: model.Relation{Type: model.RelationTypeManyToMany, RelatedModel: "ApplyRelM2mIdxB", RelatedAttribute: "as", NoIndex: true}},
	}}
	b := &model.ModelSchema{Name: "ApplyRelM2mIdxB", Relations: []model.NamedRelation{
		{Name: "as", Relation: model.Relation{Type: model.RelationTypeManyToMany, RelatedModel: "ApplyRelM2mIdxA", RelatedAttribute: "bs"}},
	}}
	if err := a.Save(filepath.Join(schemaDir, "ApplyRelM2mIdxA.yaml")); err != nil {
		t.Fatalf("save A: %v", err)
	}
	if err := b.Save(filepath.Join(schemaDir, "ApplyRelM2mIdxB.yaml")); err != nil {
		t.Fatalf("save B: %v", err)
	}

	schemas := loadTestSchemas(t, schemaDir)
	if _, err := model.GenerateMigration(db, schemas, "create_apply_rel_m2m_idx"); err != nil {
		t.Fatalf("GenerateMigration: %v", err)
	}
	migrator.Up()

	schemaA, err := model.IntrospectModelSchema(db, physA, "public", false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema(A): %v", err)
	}
	relA, ok := schemaA.RelationByName("bs")
	if !ok || !relA.NoIndex {
		t.Fatalf("A.bs = %#v, %v, want NoIndex true", relA, ok)
	}

	schemaB, err := model.IntrospectModelSchema(db, physB, "public", false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema(B): %v", err)
	}
	relB, ok := schemaB.RelationByName("as")
	if !ok || relB.NoIndex {
		t.Fatalf("B.as = %#v, %v, want NoIndex false", relB, ok)
	}

	// Down needs to go through DelRelation (not a bare DROP TABLE IF
	// EXISTS on physA/physB in t.Cleanup) - the join table's own FK
	// columns reference both, so dropping either one first without also
	// dropping the join table would fail (or, without CASCADE, silently
	// no-op and leak both the join table and physA/physB).
	migrator.Down(0)

	_, err = model.IntrospectModelSchema(db, physA, "public", false)
	if err != model.ErrTableNotFound {
		t.Fatalf("A err after Down = %v, want ErrTableNotFound", err)
	}
}

func TestApplyInvert_RenameRelation_ManyToOne(t *testing.T) {
	db := setupDB(t)
	physOne := pgTableName("ApplyRenRelOne")
	physMany := pgTableName("ApplyRenRelMany")
	for _, tbl := range []string{physMany, physOne} {
		db.Exec("DROP TABLE IF EXISTS " + tbl)
	}
	t.Cleanup(func() {
		db.Exec("DROP TABLE IF EXISTS " + physMany)
		db.Exec("DROP TABLE IF EXISTS " + physOne)
		db.Exec(`DELETE FROM lx_sys.model_relations WHERE home_table IN ($1, $2)`, "public."+physMany, "public."+physOne)
	})

	migrationsDir := t.TempDir()
	db.Exec("DROP TABLE IF EXISTS lx_sys.migrator")
	migrator.Init(migrator.Config{DB: db, MigrationsPath: migrationsDir})

	schemaDir := t.TempDir()
	onePath := filepath.Join(schemaDir, "ApplyRenRelOne.yaml")
	manyPath := filepath.Join(schemaDir, "ApplyRenRelMany.yaml")

	one := &model.ModelSchema{Name: "ApplyRenRelOne", Relations: []model.NamedRelation{
		{Name: "manys", Relation: model.Relation{Type: model.RelationTypeOneToMany, RelatedModel: "ApplyRenRelMany", RelatedAttribute: "one"}},
	}}
	many := &model.ModelSchema{Name: "ApplyRenRelMany", Relations: []model.NamedRelation{
		{Name: "one", Relation: model.Relation{Type: model.RelationTypeManyToOne, RelatedModel: "ApplyRenRelOne", RelatedAttribute: "manys"}},
	}}
	if err := one.Save(onePath); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := many.Save(manyPath); err != nil {
		t.Fatalf("save: %v", err)
	}
	schemas := loadTestSchemas(t, schemaDir)
	if _, err := model.GenerateMigration(db, schemas, "create_apply_ren_rel"); err != nil {
		t.Fatalf("GenerateMigration (create): %v", err)
	}
	migrator.Up()

	// Both sides declare a relation symmetrically (cross-validated, see
	// validateRelations) - renaming one side's attribute means the OTHER
	// side's own RelatedAttribute has to be updated to match in the same
	// edit, same as any other symmetric rename.
	many.Relations[0].Name = "theOne"
	if err := many.Save(manyPath); err != nil {
		t.Fatalf("save renamed: %v", err)
	}
	one.Relations[0].RelatedAttribute = "theOne"
	if err := one.Save(onePath); err != nil {
		t.Fatalf("save updated back-reference: %v", err)
	}
	schemas = loadTestSchemas(t, schemaDir)
	if _, err := model.GenerateMigration(db, schemas, "rename_apply_ren_rel"); err != nil {
		t.Fatalf("GenerateMigration (rename): %v", err)
	}
	migrator.Up()

	schema, err := model.IntrospectModelSchema(db, physMany, "public", false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema after rename Up: %v", err)
	}
	if _, ok := schema.RelationByName("one"); ok {
		t.Fatal("old relation name should be gone after rename")
	}
	rel, ok := schema.RelationByName("theOne")
	if !ok || rel.RelatedModel != "ApplyRenRelOne" {
		t.Fatalf("theOne after rename Up = %#v, %v", rel, ok)
	}

	oneSchema, err := model.IntrospectModelSchema(db, physOne, "public", false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema(one) after rename Up: %v", err)
	}
	manysRel, ok := oneSchema.RelationByName("manys")
	if !ok || manysRel.RelatedAttribute != "theOne" {
		t.Fatalf("manys after rename Up = %#v, %v, want RelatedAttribute theOne", manysRel, ok)
	}

	migrator.Down(0)

	schema, err = model.IntrospectModelSchema(db, physMany, "public", false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema after rename Down: %v", err)
	}
	if _, ok := schema.RelationByName("theOne"); ok {
		t.Fatal("renamed name should be gone after Down")
	}
	rel, ok = schema.RelationByName("one")
	if !ok {
		t.Fatal("expected the original relation name back after Down")
	}
}

// TestApplyInvert_RenameRelation_PassiveSideDoesNotRebuildActingSide renames
// the passive RelationTypeOneToMany side's own attribute name (which has no
// physical column of its own, see execRenamePassiveRelation) and checks
// that this doesn't disturb the acting RelationTypeManyToOne side's own
// physical FK column at all - a rename on the passive side only changes
// the acting side's own recorded RelatedAttribute (metadata, see
// execUpdateRelationRelatedAttribute), never DDL, so existing rows on the
// acting side's table must survive untouched.
func TestApplyInvert_RenameRelation_PassiveSideDoesNotRebuildActingSide(t *testing.T) {
	db := setupDB(t)
	physOne := pgTableName("ApplyRenPassiveOne")
	physMany := pgTableName("ApplyRenPassiveMany")
	for _, tbl := range []string{physMany, physOne} {
		db.Exec("DROP TABLE IF EXISTS " + tbl)
	}
	t.Cleanup(func() {
		db.Exec("DROP TABLE IF EXISTS " + physMany)
		db.Exec("DROP TABLE IF EXISTS " + physOne)
		db.Exec(`DELETE FROM lx_sys.model_relations WHERE home_table IN ($1, $2)`, "public."+physMany, "public."+physOne)
	})

	migrationsDir := t.TempDir()
	db.Exec("DROP TABLE IF EXISTS lx_sys.migrator")
	migrator.Init(migrator.Config{DB: db, MigrationsPath: migrationsDir})

	schemaDir := t.TempDir()
	onePath := filepath.Join(schemaDir, "ApplyRenPassiveOne.yaml")
	manyPath := filepath.Join(schemaDir, "ApplyRenPassiveMany.yaml")

	one := &model.ModelSchema{Name: "ApplyRenPassiveOne", Relations: []model.NamedRelation{
		{Name: "manys", Relation: model.Relation{Type: model.RelationTypeOneToMany, RelatedModel: "ApplyRenPassiveMany", RelatedAttribute: "one"}},
	}}
	many := &model.ModelSchema{Name: "ApplyRenPassiveMany", Relations: []model.NamedRelation{
		{Name: "one", Relation: model.Relation{Type: model.RelationTypeManyToOne, RelatedModel: "ApplyRenPassiveOne", RelatedAttribute: "manys"}},
	}}
	if err := one.Save(onePath); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := many.Save(manyPath); err != nil {
		t.Fatalf("save: %v", err)
	}
	schemas := loadTestSchemas(t, schemaDir)
	if _, err := model.GenerateMigration(db, schemas, "create"); err != nil {
		t.Fatalf("GenerateMigration (create): %v", err)
	}
	migrator.Up()

	var oneID int
	if err := db.QueryRow(`INSERT INTO ` + physOne + ` DEFAULT VALUES RETURNING id`).Scan(&oneID); err != nil {
		t.Fatalf("insert one: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO `+physMany+` (one_id) VALUES ($1)`, oneID); err != nil {
		t.Fatalf("insert many: %v", err)
	}

	// Rename the passive (RelationTypeOneToMany) side's own attribute name -
	// the acting side's schema is updated to match, same symmetric
	// requirement as any other rename.
	one.Relations[0].Name = "items"
	if err := one.Save(onePath); err != nil {
		t.Fatalf("save renamed: %v", err)
	}
	many.Relations[0].RelatedAttribute = "items"
	if err := many.Save(manyPath); err != nil {
		t.Fatalf("save updated back-reference: %v", err)
	}
	schemas = loadTestSchemas(t, schemaDir)
	if _, err := model.GenerateMigration(db, schemas, "rename"); err != nil {
		t.Fatalf("GenerateMigration (rename): %v", err)
	}
	migrator.Up()

	var count int
	if err := db.QueryRow(`SELECT count(*) FROM ` + physMany).Scan(&count); err != nil {
		t.Fatalf("counting rows after rename Up: %v", err)
	}
	if count != 1 {
		t.Fatalf("existing row on the acting side was lost by the passive-side rename: count = %d, want 1", count)
	}
	manySchema, err := model.IntrospectModelSchema(db, physMany, "public", false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema(Many) after rename Up: %v", err)
	}
	rel, ok := manySchema.RelationByName("one")
	if !ok || rel.RelatedAttribute != "items" {
		t.Fatalf("Many.one after rename Up = %#v, %v, want RelatedAttribute items", rel, ok)
	}

	migrator.Down(0)

	if err := db.QueryRow(`SELECT count(*) FROM ` + physMany).Scan(&count); err != nil {
		t.Fatalf("counting rows after Down: %v", err)
	}
	if count != 1 {
		t.Fatalf("existing row on the acting side was lost by inverting the passive-side rename: count = %d, want 1", count)
	}
	manySchema, err = model.IntrospectModelSchema(db, physMany, "public", false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema(Many) after Down: %v", err)
	}
	rel, ok = manySchema.RelationByName("one")
	if !ok || rel.RelatedAttribute != "manys" {
		t.Fatalf("Many.one after Down = %#v, %v, want RelatedAttribute manys", rel, ok)
	}
}

// TestApplyInvert_RenameRelation_ManyToMany_SingleSide renames only one
// side's own attribute name on an existing many-to-many relation, checking
// two things at once: execRenameManyToManyRelation's own metadata-only
// update (the renaming side), and execUpdateRelationRelatedAttribute's
// metadata-only update on the OTHER side's own ChangeRelation action
// (CompareRelations reports the other side's relation as Changed too,
// since its RelatedAttribute now mismatches - see execChangeRelation's
// doc) - neither should touch the join table's data or its physical name.
func TestApplyInvert_RenameRelation_ManyToMany_SingleSide(t *testing.T) {
	db := setupDB(t)
	physA := pgTableName("ApplyRenM2mA")
	physB := pgTableName("ApplyRenM2mB")
	dropJoinTables := func() {
		rows, err := db.Query(`SELECT table_name FROM information_schema.tables WHERE table_name LIKE 'rel__apply_ren_m2m%'`)
		if err != nil {
			return
		}
		var names []string
		for rows.Next() {
			var n string
			if rows.Scan(&n) == nil {
				names = append(names, n)
			}
		}
		rows.Close()
		for _, n := range names {
			db.Exec(fmt.Sprintf(`DROP TABLE IF EXISTS %q`, n))
		}
	}
	dropJoinTables()
	for _, tbl := range []string{physA, physB} {
		db.Exec("DROP TABLE IF EXISTS " + tbl + " CASCADE")
	}
	t.Cleanup(func() {
		// migrator.Down(0) below only undoes the rename migration, not the
		// original create - the join table is still there, and CASCADE on
		// physA/physB only drops ITS foreign key constraints into them (a
		// table merely holding a now-dangling integer column doesn't get
		// dropped by CASCADE on the table it used to reference), so the
		// join table itself has to be dropped explicitly by name, before
		// physA/physB.
		dropJoinTables()
		db.Exec("DROP TABLE IF EXISTS " + physA + " CASCADE")
		db.Exec("DROP TABLE IF EXISTS " + physB + " CASCADE")
		db.Exec(`DELETE FROM lx_sys.model_relations WHERE home_table IN ($1, $2)`, "public."+physA, "public."+physB)
	})

	migrationsDir := t.TempDir()
	db.Exec("DROP TABLE IF EXISTS lx_sys.migrator")
	migrator.Init(migrator.Config{DB: db, MigrationsPath: migrationsDir})

	schemaDir := t.TempDir()
	aPath := filepath.Join(schemaDir, "ApplyRenM2mA.yaml")
	bPath := filepath.Join(schemaDir, "ApplyRenM2mB.yaml")
	a := &model.ModelSchema{Name: "ApplyRenM2mA", Relations: []model.NamedRelation{
		{Name: "bs", Relation: model.Relation{Type: model.RelationTypeManyToMany, RelatedModel: "ApplyRenM2mB", RelatedAttribute: "as"}},
	}}
	b := &model.ModelSchema{Name: "ApplyRenM2mB", Relations: []model.NamedRelation{
		{Name: "as", Relation: model.Relation{Type: model.RelationTypeManyToMany, RelatedModel: "ApplyRenM2mA", RelatedAttribute: "bs"}},
	}}
	if err := a.Save(aPath); err != nil {
		t.Fatalf("save a: %v", err)
	}
	if err := b.Save(bPath); err != nil {
		t.Fatalf("save b: %v", err)
	}
	schemas := loadTestSchemas(t, schemaDir)
	if _, err := model.GenerateMigration(db, schemas, "create"); err != nil {
		t.Fatalf("GenerateMigration (create): %v", err)
	}
	migrator.Up()

	var aID, bID int
	if err := db.QueryRow(`INSERT INTO ` + physA + ` DEFAULT VALUES RETURNING id`).Scan(&aID); err != nil {
		t.Fatalf("insert a: %v", err)
	}
	if err := db.QueryRow(`INSERT INTO ` + physB + ` DEFAULT VALUES RETURNING id`).Scan(&bID); err != nil {
		t.Fatalf("insert b: %v", err)
	}

	joinTableRows, err := db.Query(`SELECT table_name FROM information_schema.tables WHERE table_name LIKE 'rel__apply_ren_m2m%'`)
	if err != nil {
		t.Fatalf("querying join table: %v", err)
	}
	var joinTable string
	for joinTableRows.Next() {
		if err := joinTableRows.Scan(&joinTable); err != nil {
			t.Fatalf("scanning join table name: %v", err)
		}
	}
	joinTableRows.Close()
	if joinTable == "" {
		t.Fatal("no join table found")
	}
	colRows, err := db.Query(`SELECT column_name FROM information_schema.columns WHERE table_name = $1 ORDER BY ordinal_position`, joinTable)
	if err != nil {
		t.Fatalf("querying join table columns: %v", err)
	}
	var cols []string
	for colRows.Next() {
		var c string
		if err := colRows.Scan(&c); err != nil {
			t.Fatalf("scanning column name: %v", err)
		}
		cols = append(cols, c)
	}
	colRows.Close()
	if len(cols) != 2 {
		t.Fatalf("join table %q columns = %v, want exactly 2", joinTable, cols)
	}
	if _, err := db.Exec(fmt.Sprintf(`INSERT INTO %q (%q, %q) VALUES ($1, $2)`, joinTable, cols[0], cols[1]), aID, bID); err != nil {
		t.Fatalf("insert into join table: %v", err)
	}

	// Rename B's own side only - "as" -> "cs".
	b.Relations[0].Name = "cs"
	if err := b.Save(bPath); err != nil {
		t.Fatalf("save renamed b: %v", err)
	}
	a.Relations[0].RelatedAttribute = "cs"
	if err := a.Save(aPath); err != nil {
		t.Fatalf("save updated a back-reference: %v", err)
	}
	schemas = loadTestSchemas(t, schemaDir)
	if _, err := model.GenerateMigration(db, schemas, "rename"); err != nil {
		t.Fatalf("GenerateMigration (rename): %v", err)
	}
	migrator.Up()

	var joinCount int
	if err := db.QueryRow(fmt.Sprintf(`SELECT count(*) FROM %q`, joinTable)).Scan(&joinCount); err != nil {
		t.Fatalf("counting join rows after rename: %v", err)
	}
	if joinCount != 1 {
		t.Fatalf("existing join row was lost by the rename: count = %d, want 1", joinCount)
	}

	schemaA, err := model.IntrospectModelSchema(db, physA, "public", false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema(A) after rename: %v", err)
	}
	relA, ok := schemaA.RelationByName("bs")
	if !ok || relA.RelatedAttribute != "cs" {
		t.Fatalf("A.bs after rename = %#v, %v, want RelatedAttribute cs", relA, ok)
	}
	schemaB, err := model.IntrospectModelSchema(db, physB, "public", false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema(B) after rename: %v", err)
	}
	if _, ok := schemaB.RelationByName("as"); ok {
		t.Fatal("old relation name should be gone after rename")
	}
	relB, ok := schemaB.RelationByName("cs")
	if !ok || relB.RelatedAttribute != "bs" {
		t.Fatalf("B.cs after rename = %#v, %v, want RelatedAttribute bs", relB, ok)
	}

	migrator.Down(0)

	if err := db.QueryRow(fmt.Sprintf(`SELECT count(*) FROM %q`, joinTable)).Scan(&joinCount); err != nil {
		t.Fatalf("counting join rows after Down: %v", err)
	}
	if joinCount != 1 {
		t.Fatalf("existing join row was lost by inverting the rename: count = %d, want 1", joinCount)
	}
}

// TestApplyInvert_ChangeRelation_HeavyPath_RelatedModelChanges checks
// execChangeRelation's teardown-and-rebuild path for a genuine structural
// change (RelatedModel itself changes, not just RelatedAttribute/NoIndex) -
// the relation's old physical column/constraint/index must be gone and a
// new one pointing at the new related table built in their place.
func TestApplyInvert_ChangeRelation_HeavyPath_RelatedModelChanges(t *testing.T) {
	db := setupDB(t)
	physMany := pgTableName("ApplyChgRelMany")
	physOldOne := pgTableName("ApplyChgRelOldOne")
	physNewOne := pgTableName("ApplyChgRelNewOne")
	for _, tbl := range []string{physMany, physOldOne, physNewOne} {
		db.Exec("DROP TABLE IF EXISTS " + tbl)
	}
	t.Cleanup(func() {
		db.Exec("DROP TABLE IF EXISTS " + physMany)
		db.Exec("DROP TABLE IF EXISTS " + physOldOne)
		db.Exec("DROP TABLE IF EXISTS " + physNewOne)
		db.Exec(`DELETE FROM lx_sys.model_relations WHERE home_table IN ($1, $2, $3)`, "public."+physMany, "public."+physOldOne, "public."+physNewOne)
	})

	migrationsDir := t.TempDir()
	db.Exec("DROP TABLE IF EXISTS lx_sys.migrator")
	migrator.Init(migrator.Config{DB: db, MigrationsPath: migrationsDir})

	schemaDir := t.TempDir()
	manyPath := filepath.Join(schemaDir, "ApplyChgRelMany.yaml")
	oldOnePath := filepath.Join(schemaDir, "ApplyChgRelOldOne.yaml")
	newOnePath := filepath.Join(schemaDir, "ApplyChgRelNewOne.yaml")

	many := &model.ModelSchema{Name: "ApplyChgRelMany", Relations: []model.NamedRelation{
		{Name: "one", Relation: model.Relation{Type: model.RelationTypeManyToOne, RelatedModel: "ApplyChgRelOldOne", RelatedAttribute: "manys"}},
	}}
	oldOne := &model.ModelSchema{Name: "ApplyChgRelOldOne", Relations: []model.NamedRelation{
		{Name: "manys", Relation: model.Relation{Type: model.RelationTypeOneToMany, RelatedModel: "ApplyChgRelMany", RelatedAttribute: "one"}},
	}}
	newOne := &model.ModelSchema{Name: "ApplyChgRelNewOne"}
	if err := many.Save(manyPath); err != nil {
		t.Fatalf("save many: %v", err)
	}
	if err := oldOne.Save(oldOnePath); err != nil {
		t.Fatalf("save oldOne: %v", err)
	}
	if err := newOne.Save(newOnePath); err != nil {
		t.Fatalf("save newOne: %v", err)
	}
	schemas := loadTestSchemas(t, schemaDir)
	if _, err := model.GenerateMigration(db, schemas, "create"); err != nil {
		t.Fatalf("GenerateMigration (create): %v", err)
	}
	migrator.Up()

	// Point ApplyChgRelMany.one at ApplyChgRelNewOne instead - a genuine
	// structural change, not renamable or index-only.
	many.Relations[0].RelatedModel = "ApplyChgRelNewOne"
	if err := many.Save(manyPath); err != nil {
		t.Fatalf("save changed: %v", err)
	}
	oldOne.Relations = nil
	if err := oldOne.Save(oldOnePath); err != nil {
		t.Fatalf("save oldOne without relation: %v", err)
	}
	newOne.Relations = []model.NamedRelation{
		{Name: "manys", Relation: model.Relation{Type: model.RelationTypeOneToMany, RelatedModel: "ApplyChgRelMany", RelatedAttribute: "one"}},
	}
	if err := newOne.Save(newOnePath); err != nil {
		t.Fatalf("save newOne with relation: %v", err)
	}
	schemas = loadTestSchemas(t, schemaDir)
	if _, err := model.GenerateMigration(db, schemas, "change"); err != nil {
		t.Fatalf("GenerateMigration (change): %v", err)
	}
	migrator.Up()

	schema, err := model.IntrospectModelSchema(db, physMany, "public", false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema after change Up: %v", err)
	}
	rel, ok := schema.RelationByName("one")
	if !ok || rel.RelatedModel != "ApplyChgRelNewOne" {
		t.Fatalf("one after change Up = %#v, %v, want RelatedModel ApplyChgRelNewOne", rel, ok)
	}

	var newOneID int
	if err := db.QueryRow(`INSERT INTO ` + physNewOne + ` DEFAULT VALUES RETURNING id`).Scan(&newOneID); err != nil {
		t.Fatalf("insert into new related table: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO `+physMany+` (one_id) VALUES ($1)`, newOneID); err != nil {
		t.Fatalf("insert row referencing the new related table: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO `+physMany+` (one_id) VALUES ($1)`, 999999); err == nil {
		t.Fatal("expected a foreign key violation against a non-existent ApplyChgRelNewOne row")
	}

	// Inverting a genuine structural change (RelatedModel itself changing)
	// really does drop and recreate the "one_id" column, the same as any
	// other RelationTypeManyToOne teardown-and-rebuild - unlike the
	// RelatedAttribute-only case (see execChangeRelation's doc), there's no
	// way to keep existing rows' values meaningful across it, so the row
	// inserted above has to go before Down can succeed (exactly as a plain
	// ALTER TABLE ADD COLUMN ... NOT NULL against a non-empty table would
	// also require).
	if _, err := db.Exec(`DELETE FROM ` + physMany); err != nil {
		t.Fatalf("clearing rows before Down: %v", err)
	}

	migrator.Down(0)

	schema, err = model.IntrospectModelSchema(db, physMany, "public", false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema after change Down: %v", err)
	}
	rel, ok = schema.RelationByName("one")
	if !ok || rel.RelatedModel != "ApplyChgRelOldOne" {
		t.Fatalf("one after change Down = %#v, %v, want RelatedModel ApplyChgRelOldOne", rel, ok)
	}
}

// TestApplyInvert_DelRelation_ManyToMany_RestoresRelatedNoIndex checks that
// inverting a many-to-many DelRelation (undoing a deletion, i.e. re-adding
// the relation) restores the related side's own no-index choice exactly,
// rather than defaulting it back to indexed - DelRelationAction.
// RelatedNoIndex only exists to make this possible, since by the time a
// relation is deleted it's already gone from both sides' schema files (see
// DelRelationAction's doc).
func TestApplyInvert_DelRelation_ManyToMany_RestoresRelatedNoIndex(t *testing.T) {
	db := setupDB(t)
	physA := pgTableName("ApplyDelRelM2mIdxA")
	physB := pgTableName("ApplyDelRelM2mIdxB")
	dropJoinTables := func() {
		rows, err := db.Query(`SELECT table_name FROM information_schema.tables WHERE table_name LIKE 'rel__apply_del_rel_m2m_idx%'`)
		if err != nil {
			return
		}
		var names []string
		for rows.Next() {
			var n string
			if rows.Scan(&n) == nil {
				names = append(names, n)
			}
		}
		rows.Close()
		for _, n := range names {
			db.Exec(fmt.Sprintf(`DROP TABLE IF EXISTS %q`, n))
		}
	}
	dropJoinTables()
	for _, tbl := range []string{physA, physB} {
		db.Exec("DROP TABLE IF EXISTS " + tbl + " CASCADE")
	}
	t.Cleanup(func() {
		// migrator.Down(0) below only undoes the delete migration (i.e. it
		// re-adds the relation) - the join table is there again, and
		// CASCADE on physA/physB only drops ITS foreign key constraints
		// into them, not the table itself, so it has to be dropped
		// explicitly by name, before physA/physB (see the sibling comment
		// on TestApplyInvert_RenameRelation_ManyToMany_SingleSide).
		dropJoinTables()
		db.Exec("DROP TABLE IF EXISTS " + physA + " CASCADE")
		db.Exec("DROP TABLE IF EXISTS " + physB + " CASCADE")
		db.Exec(`DELETE FROM lx_sys.model_relations WHERE home_table IN ($1, $2)`, "public."+physA, "public."+physB)
	})

	migrationsDir := t.TempDir()
	db.Exec("DROP TABLE IF EXISTS lx_sys.migrator")
	migrator.Init(migrator.Config{DB: db, MigrationsPath: migrationsDir})

	schemaDir := t.TempDir()
	aPath := filepath.Join(schemaDir, "ApplyDelRelM2mIdxA.yaml")
	bPath := filepath.Join(schemaDir, "ApplyDelRelM2mIdxB.yaml")
	// "ApplyDelRelM2mIdxA" < "ApplyDelRelM2mIdxB" - A is the acting side
	// (see canIgnoreRelation), so DelRelation is built on A's own diff, and
	// B's own no-index declaration is the "related" one this test cares
	// about restoring.
	a := &model.ModelSchema{Name: "ApplyDelRelM2mIdxA", Relations: []model.NamedRelation{
		{Name: "bs", Relation: model.Relation{Type: model.RelationTypeManyToMany, RelatedModel: "ApplyDelRelM2mIdxB", RelatedAttribute: "as"}},
	}}
	b := &model.ModelSchema{Name: "ApplyDelRelM2mIdxB", Relations: []model.NamedRelation{
		{Name: "as", Relation: model.Relation{Type: model.RelationTypeManyToMany, RelatedModel: "ApplyDelRelM2mIdxA", RelatedAttribute: "bs", NoIndex: true}},
	}}
	if err := a.Save(aPath); err != nil {
		t.Fatalf("save a: %v", err)
	}
	if err := b.Save(bPath); err != nil {
		t.Fatalf("save b: %v", err)
	}
	schemas := loadTestSchemas(t, schemaDir)
	if _, err := model.GenerateMigration(db, schemas, "create"); err != nil {
		t.Fatalf("GenerateMigration (create): %v", err)
	}
	migrator.Up()

	schemaB, err := model.IntrospectModelSchema(db, physB, "public", false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema(B) after create: %v", err)
	}
	relB, ok := schemaB.RelationByName("as")
	if !ok || !relB.NoIndex {
		t.Fatalf("B.as after create = %#v, %v, want NoIndex true", relB, ok)
	}

	// Delete the relation from both sides - the only way to remove a
	// symmetrically-declared relation.
	a.Relations = nil
	if err := a.Save(aPath); err != nil {
		t.Fatalf("save a without relation: %v", err)
	}
	b.Relations = nil
	if err := b.Save(bPath); err != nil {
		t.Fatalf("save b without relation: %v", err)
	}
	schemas = loadTestSchemas(t, schemaDir)
	if _, err := model.GenerateMigration(db, schemas, "delete"); err != nil {
		t.Fatalf("GenerateMigration (delete): %v", err)
	}
	migrator.Up()

	if _, err := model.IntrospectModelSchema(db, physA, "public", false); err != nil {
		t.Fatalf("IntrospectModelSchema(A) after delete Up: %v", err)
	}

	migrator.Down(0)

	schemaB, err = model.IntrospectModelSchema(db, physB, "public", false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema(B) after delete Down: %v", err)
	}
	relB, ok = schemaB.RelationByName("as")
	if !ok || !relB.NoIndex {
		t.Fatalf("B.as after undoing the delete = %#v, %v, want NoIndex true (restored, not defaulted to indexed)", relB, ok)
	}
}

func TestApplyInvert_ChangeRelation_IndexToggle(t *testing.T) {
	db := setupDB(t)
	physOne := pgTableName("ApplyIdxRelOne")
	physMany := pgTableName("ApplyIdxRelMany")
	for _, tbl := range []string{physMany, physOne} {
		db.Exec("DROP TABLE IF EXISTS " + tbl)
	}
	t.Cleanup(func() {
		db.Exec("DROP TABLE IF EXISTS " + physMany)
		db.Exec("DROP TABLE IF EXISTS " + physOne)
		db.Exec(`DELETE FROM lx_sys.model_relations WHERE home_table IN ($1, $2)`, "public."+physMany, "public."+physOne)
	})

	migrationsDir := t.TempDir()
	db.Exec("DROP TABLE IF EXISTS lx_sys.migrator")
	migrator.Init(migrator.Config{DB: db, MigrationsPath: migrationsDir})

	schemaDir := t.TempDir()
	onePath := filepath.Join(schemaDir, "ApplyIdxRelOne.yaml")
	manyPath := filepath.Join(schemaDir, "ApplyIdxRelMany.yaml")

	one := &model.ModelSchema{Name: "ApplyIdxRelOne", Relations: []model.NamedRelation{
		{Name: "manys", Relation: model.Relation{Type: model.RelationTypeOneToMany, RelatedModel: "ApplyIdxRelMany", RelatedAttribute: "one"}},
	}}
	many := &model.ModelSchema{Name: "ApplyIdxRelMany", Relations: []model.NamedRelation{
		{Name: "one", Relation: model.Relation{Type: model.RelationTypeManyToOne, RelatedModel: "ApplyIdxRelOne", RelatedAttribute: "manys"}},
	}}
	if err := one.Save(onePath); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := many.Save(manyPath); err != nil {
		t.Fatalf("save: %v", err)
	}
	schemas := loadTestSchemas(t, schemaDir)
	if _, err := model.GenerateMigration(db, schemas, "create_apply_idx_rel"); err != nil {
		t.Fatalf("GenerateMigration (create): %v", err)
	}
	migrator.Up()

	schema, err := model.IntrospectModelSchema(db, physMany, "public", false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema before toggle: %v", err)
	}
	rel, ok := schema.RelationByName("one")
	if !ok || rel.NoIndex {
		t.Fatalf("one before toggle = %#v, %v, want indexed by default", rel, ok)
	}

	many.Relations[0].Relation.NoIndex = true
	if err := many.Save(manyPath); err != nil {
		t.Fatalf("save no-index: %v", err)
	}
	schemas = loadTestSchemas(t, schemaDir)
	if _, err := model.GenerateMigration(db, schemas, "toggle_apply_idx_rel"); err != nil {
		t.Fatalf("GenerateMigration (toggle): %v", err)
	}
	migrator.Up()

	schema, err = model.IntrospectModelSchema(db, physMany, "public", false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema after toggle Up: %v", err)
	}
	rel, ok = schema.RelationByName("one")
	if !ok || !rel.NoIndex {
		t.Fatalf("one after toggle Up = %#v, %v, want NoIndex true", rel, ok)
	}
	// The column/constraint themselves must survive untouched - a toggle
	// is index-only, not a rebuild.
	if _, ok := schema.FieldByName("one_id"); ok {
		t.Fatal("the FK column should still not appear in Fields")
	}

	migrator.Down(0)

	schema, err = model.IntrospectModelSchema(db, physMany, "public", false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema after toggle Down: %v", err)
	}
	rel, ok = schema.RelationByName("one")
	if !ok || rel.NoIndex {
		t.Fatalf("one after toggle Down = %#v, %v, want NoIndex false again", rel, ok)
	}
}
