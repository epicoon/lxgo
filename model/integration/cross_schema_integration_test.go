//go:build integration

package model_test

import (
	"path/filepath"
	"testing"

	"github.com/epicoon/lxgo/migrator"
	"github.com/epicoon/lxgo/model"
)

// TestApplyInvert_CrossSchema_ManyToOne checks a manyToOne relation between
// two models resolved to two different Postgres schemas - the acting
// model's own table lands in its own schema, the FK genuinely references
// the related table in ITS OWN (different) schema, not assumed to be the
// acting model's schema (see execAddToOneRelation's doc).
func TestApplyInvert_CrossSchema_ManyToOne(t *testing.T) {
	db := setupDB(t)
	const nsA = "model_test_ns_cross_a"
	const nsB = "model_test_ns_cross_b"
	physOne := pgTableName("CrossRelOne")
	physMany := pgTableName("CrossRelMany")
	for _, ns := range []string{nsA, nsB} {
		if _, err := db.Exec("DROP SCHEMA IF EXISTS " + ns + " CASCADE"); err != nil {
			t.Fatalf("drop schema %s: %v", ns, err)
		}
	}
	t.Cleanup(func() {
		db.Exec("DROP SCHEMA IF EXISTS " + nsA + " CASCADE")
		db.Exec("DROP SCHEMA IF EXISTS " + nsB + " CASCADE")
		db.Exec(`DELETE FROM lx_sys.model_relations WHERE home_table IN ($1, $2)`, nsB+"."+physMany, nsA+"."+physOne)
	})

	migrationsDir := t.TempDir()
	db.Exec("DROP TABLE IF EXISTS lx_sys.migrator")
	migrator.Init(migrator.Config{DB: db, MigrationsPath: migrationsDir})

	schemaDir := t.TempDir()
	one := &model.ModelSchema{Name: "CrossRelOne", Namespace: nsA, Relations: []model.NamedRelation{
		{Name: "manys", Relation: model.Relation{Type: model.RelationTypeOneToMany, RelatedModel: "CrossRelMany", RelatedAttribute: "one"}},
	}}
	many := &model.ModelSchema{Name: "CrossRelMany", Namespace: nsB, Relations: []model.NamedRelation{
		{Name: "one", Relation: model.Relation{Type: model.RelationTypeManyToOne, RelatedModel: "CrossRelOne", RelatedAttribute: "manys"}},
	}}
	if err := one.Save(filepath.Join(schemaDir, "CrossRelOne.yaml")); err != nil {
		t.Fatalf("save one: %v", err)
	}
	if err := many.Save(filepath.Join(schemaDir, "CrossRelMany.yaml")); err != nil {
		t.Fatalf("save many: %v", err)
	}

	schemas := loadTestSchemas(t, schemaDir)
	if _, err := model.GenerateMigration(db, schemas, "create"); err != nil {
		t.Fatalf("GenerateMigration: %v", err)
	}
	migrator.Up()

	oneSchema, err := model.IntrospectModelSchema(db, physOne, nsA, false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema(one, %s): %v", nsA, err)
	}
	if _, ok := oneSchema.RelationByName("manys"); !ok {
		t.Fatal("expected a manys relation on CrossRelOne")
	}

	manySchema, err := model.IntrospectModelSchema(db, physMany, nsB, false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema(many, %s): %v", nsB, err)
	}
	rel, ok := manySchema.RelationByName("one")
	if !ok || rel.Type != model.RelationTypeManyToOne || rel.RelatedModel != "CrossRelOne" {
		t.Fatalf("one = %#v, %v", rel, ok)
	}

	// Prove the foreign key genuinely spans the two schemas, not just that
	// the column/relation metadata claims to - insert into the "one" side
	// (nsA) and reference it from the "many" side (nsB), then confirm a
	// non-existent id is rejected by the cross-schema FK constraint.
	var oneID int
	if err := db.QueryRow(`INSERT INTO ` + nsA + `.` + physOne + ` DEFAULT VALUES RETURNING id`).Scan(&oneID); err != nil {
		t.Fatalf("insert into %s.%s: %v", nsA, physOne, err)
	}
	if _, err := db.Exec(`INSERT INTO `+nsB+`.`+physMany+` (one_id) VALUES ($1)`, oneID); err != nil {
		t.Fatalf("insert into %s.%s referencing %s.%s: %v", nsB, physMany, nsA, physOne, err)
	}
	if _, err := db.Exec(`INSERT INTO `+nsB+`.`+physMany+` (one_id) VALUES ($1)`, oneID+999999); err == nil {
		t.Fatal("expected a foreign key violation against a non-existent cross-schema row")
	}

	migrator.Down(0)

	if _, err := model.IntrospectModelSchema(db, physMany, nsB, false); err != model.ErrTableNotFound {
		t.Fatalf("many err after Down = %v, want ErrTableNotFound", err)
	}
}

// TestApplyInvert_CrossSchema_ManyToMany checks a manyToMany relation
// between two models resolved to two different schemas - the join table
// always lives in the acting side's own schema (see
// execAddManyToManyRelation's doc), with one FK column referencing each
// side's own (different) schema.
func TestApplyInvert_CrossSchema_ManyToMany(t *testing.T) {
	db := setupDB(t)
	const nsA = "model_test_ns_cross_m2m_a"
	const nsB = "model_test_ns_cross_m2m_b"
	physA := pgTableName("CrossM2mA")
	physB := pgTableName("CrossM2mB")
	for _, ns := range []string{nsA, nsB} {
		if _, err := db.Exec("DROP SCHEMA IF EXISTS " + ns + " CASCADE"); err != nil {
			t.Fatalf("drop schema %s: %v", ns, err)
		}
	}
	t.Cleanup(func() {
		db.Exec("DROP SCHEMA IF EXISTS " + nsA + " CASCADE")
		db.Exec("DROP SCHEMA IF EXISTS " + nsB + " CASCADE")
		db.Exec(`DELETE FROM lx_sys.model_relations WHERE home_table IN ($1, $2)`, nsA+"."+physA, nsB+"."+physB)
	})

	migrationsDir := t.TempDir()
	db.Exec("DROP TABLE IF EXISTS lx_sys.migrator")
	migrator.Init(migrator.Config{DB: db, MigrationsPath: migrationsDir})

	schemaDir := t.TempDir()
	a := &model.ModelSchema{Name: "CrossM2mA", Namespace: nsA, Relations: []model.NamedRelation{
		{Name: "bs", Relation: model.Relation{Type: model.RelationTypeManyToMany, RelatedModel: "CrossM2mB", RelatedAttribute: "as"}},
	}}
	b := &model.ModelSchema{Name: "CrossM2mB", Namespace: nsB, Relations: []model.NamedRelation{
		{Name: "as", Relation: model.Relation{Type: model.RelationTypeManyToMany, RelatedModel: "CrossM2mA", RelatedAttribute: "bs"}},
	}}
	if err := a.Save(filepath.Join(schemaDir, "CrossM2mA.yaml")); err != nil {
		t.Fatalf("save A: %v", err)
	}
	if err := b.Save(filepath.Join(schemaDir, "CrossM2mB.yaml")); err != nil {
		t.Fatalf("save B: %v", err)
	}

	schemas := loadTestSchemas(t, schemaDir)
	if _, err := model.GenerateMigration(db, schemas, "create"); err != nil {
		t.Fatalf("GenerateMigration: %v", err)
	}
	migrator.Up()

	schemaA, err := model.IntrospectModelSchema(db, physA, nsA, false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema(A, %s): %v", nsA, err)
	}
	relA, ok := schemaA.RelationByName("bs")
	if !ok || relA.Type != model.RelationTypeManyToMany || relA.RelatedModel != "CrossM2mB" {
		t.Fatalf("A.bs = %#v, %v", relA, ok)
	}

	schemaB, err := model.IntrospectModelSchema(db, physB, nsB, false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema(B, %s): %v", nsB, err)
	}
	relB, ok := schemaB.RelationByName("as")
	if !ok || relB.Type != model.RelationTypeManyToMany || relB.RelatedModel != "CrossM2mA" {
		t.Fatalf("B.as = %#v, %v", relB, ok)
	}

	// The join table lives in A's own schema (nsA, the acting side, since
	// "CrossM2mA" < "CrossM2mB" alphabetically) - if either FK column were
	// qualified with the wrong schema, migrator.Up() above would already
	// have failed (the REFERENCES target wouldn't exist), so this only
	// needs to confirm the join table itself landed where expected.
	var joinTableCount int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM information_schema.tables
		WHERE table_schema = $1 AND table_name LIKE 'rel__%'
	`, nsA).Scan(&joinTableCount); err != nil {
		t.Fatalf("checking join table in %s: %v", nsA, err)
	}
	if joinTableCount != 1 {
		t.Fatalf("join tables in %s = %d, want 1", nsA, joinTableCount)
	}

	migrator.Down(0)

	if _, err := model.IntrospectModelSchema(db, physA, nsA, false); err != model.ErrTableNotFound {
		t.Fatalf("A err after Down = %v, want ErrTableNotFound", err)
	}
	if _, err := model.IntrospectModelSchema(db, physB, nsB, false); err != model.ErrTableNotFound {
		t.Fatalf("B err after Down = %v, want ErrTableNotFound", err)
	}
}
