//go:build integration

package model_test

import (
	"testing"

	"github.com/epicoon/lxgo/model"
)

func TestIntrospectModelSchema_ManyToOne_NoIndexWhenNoIndexExists(t *testing.T) {
	db := setupDB(t)
	physOne := pgTableName("RelIdxNoIdxOne")
	physMany := pgTableName("RelIdxNoIdxMany")
	for _, tbl := range []string{physMany, physOne} {
		if _, err := db.Exec("DROP TABLE IF EXISTS " + tbl); err != nil {
			t.Fatalf("drop %s: %v", tbl, err)
		}
	}
	t.Cleanup(func() {
		db.Exec("DROP TABLE IF EXISTS " + physMany)
		db.Exec("DROP TABLE IF EXISTS " + physOne)
		db.Exec(`DELETE FROM lx_sys.model_relations WHERE home_table = $1`, "public."+physMany)
	})

	if _, err := db.Exec(`CREATE TABLE ` + physOne + ` (id serial PRIMARY KEY)`); err != nil {
		t.Fatalf("create %s: %v", physOne, err)
	}
	fkName := "fk_rel_idx_no_idx_many_one"
	if _, err := db.Exec(`
		CREATE TABLE ` + physMany + ` (
			id serial PRIMARY KEY,
			one_id integer NOT NULL CONSTRAINT ` + fkName + ` REFERENCES ` + physOne + `(id)
		)
	`); err != nil {
		t.Fatalf("create %s: %v", physMany, err)
	}
	fk := model.RelationFk{
		Type: model.RelationTypeManyToOne, HomeTable: "public." + physMany, HomeModel: "RelIdxNoIdxMany", HomeAttribute: "one",
		RelatedTable: "public." + physOne, RelatedModel: "RelIdxNoIdxOne", RelatedAttribute: "manys",
	}
	if err := model.SetRelationFk(db, fkName, fk); err != nil {
		t.Fatalf("SetRelationFk: %v", err)
	}

	schema, err := model.IntrospectModelSchema(db, physMany, "public", false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema: %v", err)
	}
	rel, ok := schema.RelationByName("one")
	if !ok || !rel.NoIndex {
		t.Fatalf("one = %#v, %v, want NoIndex true (no index was ever created)", rel, ok)
	}
}

func TestIntrospectModelSchema_ManyToOne_IndexedWhenIndexExists(t *testing.T) {
	db := setupDB(t)
	physOne := pgTableName("RelIdxYesIdxOne")
	physMany := pgTableName("RelIdxYesIdxMany")
	for _, tbl := range []string{physMany, physOne} {
		if _, err := db.Exec("DROP TABLE IF EXISTS " + tbl); err != nil {
			t.Fatalf("drop %s: %v", tbl, err)
		}
	}
	t.Cleanup(func() {
		db.Exec("DROP TABLE IF EXISTS " + physMany)
		db.Exec("DROP TABLE IF EXISTS " + physOne)
		db.Exec(`DELETE FROM lx_sys.model_relations WHERE home_table = $1`, "public."+physMany)
	})

	if _, err := db.Exec(`CREATE TABLE ` + physOne + ` (id serial PRIMARY KEY)`); err != nil {
		t.Fatalf("create %s: %v", physOne, err)
	}
	fkName := "fk_rel_idx_yes_idx_many_one"
	if _, err := db.Exec(`
		CREATE TABLE ` + physMany + ` (
			id serial PRIMARY KEY,
			one_id integer NOT NULL CONSTRAINT ` + fkName + ` REFERENCES ` + physOne + `(id)
		)
	`); err != nil {
		t.Fatalf("create %s: %v", physMany, err)
	}
	if _, err := db.Exec(`CREATE INDEX ON ` + physMany + ` (one_id)`); err != nil {
		t.Fatalf("create index: %v", err)
	}
	fk := model.RelationFk{
		Type: model.RelationTypeManyToOne, HomeTable: "public." + physMany, HomeModel: "RelIdxYesIdxMany", HomeAttribute: "one",
		RelatedTable: "public." + physOne, RelatedModel: "RelIdxYesIdxOne", RelatedAttribute: "manys",
	}
	if err := model.SetRelationFk(db, fkName, fk); err != nil {
		t.Fatalf("SetRelationFk: %v", err)
	}

	schema, err := model.IntrospectModelSchema(db, physMany, "public", false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema: %v", err)
	}
	rel, ok := schema.RelationByName("one")
	if !ok || rel.NoIndex {
		t.Fatalf("one = %#v, %v, want NoIndex false (an index exists)", rel, ok)
	}
}

// TestIntrospectModelSchema_OneToOne_AlwaysIndexed checks that NoIndex is
// always false for RelationTypeOneToOne, even though nothing here
// explicitly creates an index beyond the UNIQUE constraint itself - the
// UNIQUE constraint's own backing index is enough (see Relation.NoIndex's
// doc).
func TestIntrospectModelSchema_OneToOne_AlwaysIndexed(t *testing.T) {
	db := setupDB(t)
	physA := pgTableName("RelIdxOneToOneA")
	physB := pgTableName("RelIdxOneToOneB")
	for _, tbl := range []string{physA, physB} {
		if _, err := db.Exec("DROP TABLE IF EXISTS " + tbl); err != nil {
			t.Fatalf("drop %s: %v", tbl, err)
		}
	}
	t.Cleanup(func() {
		db.Exec("DROP TABLE IF EXISTS " + physA)
		db.Exec("DROP TABLE IF EXISTS " + physB)
		db.Exec(`DELETE FROM lx_sys.model_relations WHERE home_table = $1`, "public."+physA)
	})

	if _, err := db.Exec(`CREATE TABLE ` + physB + ` (id serial PRIMARY KEY)`); err != nil {
		t.Fatalf("create %s: %v", physB, err)
	}
	fkName := "fk_rel_idx_one_to_one_a_b"
	if _, err := db.Exec(`
		CREATE TABLE ` + physA + ` (
			id serial PRIMARY KEY,
			b_id integer NOT NULL CONSTRAINT ` + fkName + ` REFERENCES ` + physB + `(id) UNIQUE
		)
	`); err != nil {
		t.Fatalf("create %s: %v", physA, err)
	}
	fk := model.RelationFk{
		Type: model.RelationTypeOneToOne, HomeTable: "public." + physA, HomeModel: "RelIdxOneToOneA", HomeAttribute: "b",
		RelatedTable: "public." + physB, RelatedModel: "RelIdxOneToOneB", RelatedAttribute: "a",
	}
	if err := model.SetRelationFk(db, fkName, fk); err != nil {
		t.Fatalf("SetRelationFk: %v", err)
	}

	schema, err := model.IntrospectModelSchema(db, physA, "public", false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema: %v", err)
	}
	rel, ok := schema.RelationByName("b")
	if !ok || rel.NoIndex {
		t.Fatalf("b = %#v, %v, want NoIndex false", rel, ok)
	}
}

// TestIntrospectModelSchema_ManyToMany_NoIndexIsPerSide checks that each
// side's index is independent - one side can be indexed while the other
// isn't, and introspection reports each accurately.
func TestIntrospectModelSchema_ManyToMany_NoIndexIsPerSide(t *testing.T) {
	db := setupDB(t)
	physA := pgTableName("RelIdxM2mA")
	physB := pgTableName("RelIdxM2mB")
	physJoin := "rel_idx_m2m_a_b"
	for _, tbl := range []string{physJoin, physA, physB} {
		if _, err := db.Exec("DROP TABLE IF EXISTS " + tbl); err != nil {
			t.Fatalf("drop %s: %v", tbl, err)
		}
	}
	t.Cleanup(func() {
		db.Exec("DROP TABLE IF EXISTS " + physJoin)
		db.Exec("DROP TABLE IF EXISTS " + physA)
		db.Exec("DROP TABLE IF EXISTS " + physB)
		db.Exec(`DELETE FROM lx_sys.model_relations WHERE home_table IN ($1, $2)`, "public."+physA, "public."+physB)
	})

	if _, err := db.Exec(`CREATE TABLE ` + physA + ` (id serial PRIMARY KEY)`); err != nil {
		t.Fatalf("create %s: %v", physA, err)
	}
	if _, err := db.Exec(`CREATE TABLE ` + physB + ` (id serial PRIMARY KEY)`); err != nil {
		t.Fatalf("create %s: %v", physB, err)
	}
	if _, err := db.Exec(`
		CREATE TABLE ` + physJoin + ` (
			a_id integer NOT NULL CONSTRAINT fk_rel_idx_m2m_a_side REFERENCES ` + physA + `(id),
			b_id integer NOT NULL CONSTRAINT fk_rel_idx_m2m_b_side REFERENCES ` + physB + `(id)
		)
	`); err != nil {
		t.Fatalf("create %s: %v", physJoin, err)
	}
	// Only a_id gets an index - b_id stays unindexed.
	if _, err := db.Exec(`CREATE INDEX ON ` + physJoin + ` (a_id)`); err != nil {
		t.Fatalf("create index: %v", err)
	}

	fkA := model.RelationFk{
		Type: model.RelationTypeManyToMany, HomeTable: "public." + physA, HomeModel: "RelIdxM2mA", HomeAttribute: "bs",
		RelatedTable: "public." + physB, RelatedModel: "RelIdxM2mB", RelatedAttribute: "as",
	}
	fkB := model.RelationFk{
		Type: model.RelationTypeManyToMany, HomeTable: "public." + physB, HomeModel: "RelIdxM2mB", HomeAttribute: "as",
		RelatedTable: "public." + physA, RelatedModel: "RelIdxM2mA", RelatedAttribute: "bs",
	}
	if err := model.SetRelationFk(db, "fk_rel_idx_m2m_a_side", fkA); err != nil {
		t.Fatalf("SetRelationFk A: %v", err)
	}
	if err := model.SetRelationFk(db, "fk_rel_idx_m2m_b_side", fkB); err != nil {
		t.Fatalf("SetRelationFk B: %v", err)
	}

	schemaA, err := model.IntrospectModelSchema(db, physA, "public", false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema(A): %v", err)
	}
	relA, ok := schemaA.RelationByName("bs")
	if !ok || relA.NoIndex {
		t.Fatalf("A.bs = %#v, %v, want NoIndex false (a_id is indexed)", relA, ok)
	}

	schemaB, err := model.IntrospectModelSchema(db, physB, "public", false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema(B): %v", err)
	}
	relB, ok := schemaB.RelationByName("as")
	if !ok || !relB.NoIndex {
		t.Fatalf("B.as = %#v, %v, want NoIndex true (b_id has no index)", relB, ok)
	}
}
