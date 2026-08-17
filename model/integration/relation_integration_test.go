//go:build integration

package model_test

import (
	"testing"

	"github.com/epicoon/lxgo/model"
)

func TestSetRelationFk_DeleteRelationFk(t *testing.T) {
	db := setupDB(t)
	t.Cleanup(func() {
		db.Exec(`DELETE FROM lx_sys.model_relations WHERE fk_name = 'fk_test_relation_fk'`)
	})

	fk := model.RelationFk{
		Type:             model.RelationTypeManyToOne,
		HomeTable:        "gadgets",
		HomeModel:        "Gadget",
		HomeAttribute:    "widget",
		RelatedTable:     "widgets",
		RelatedModel:     "Widget",
		RelatedAttribute: "gadgets",
	}
	if err := model.SetRelationFk(db, "fk_test_relation_fk", fk); err != nil {
		t.Fatalf("SetRelationFk: %v", err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM lx_sys.model_relations WHERE fk_name = 'fk_test_relation_fk'`).Scan(&count); err != nil {
		t.Fatalf("querying: %v", err)
	}
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}

	// Overwrite - SetRelationFk on the same fk_name updates, not duplicates.
	fk.HomeAttribute = "renamedWidget"
	if err := model.SetRelationFk(db, "fk_test_relation_fk", fk); err != nil {
		t.Fatalf("SetRelationFk (overwrite): %v", err)
	}
	var homeAttribute string
	if err := db.QueryRow(`SELECT home_attribute FROM lx_sys.model_relations WHERE fk_name = 'fk_test_relation_fk'`).Scan(&homeAttribute); err != nil {
		t.Fatalf("querying after overwrite: %v", err)
	}
	if homeAttribute != "renamedWidget" {
		t.Fatalf("home_attribute = %q, want %q", homeAttribute, "renamedWidget")
	}

	if err := model.DeleteRelationFk(db, "fk_test_relation_fk"); err != nil {
		t.Fatalf("DeleteRelationFk: %v", err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM lx_sys.model_relations WHERE fk_name = 'fk_test_relation_fk'`).Scan(&count); err != nil {
		t.Fatalf("querying after delete: %v", err)
	}
	if count != 0 {
		t.Fatalf("count after delete = %d, want 0", count)
	}

	// Deleting an fk_name that was never recorded is not an error.
	if err := model.DeleteRelationFk(db, "fk_never_recorded"); err != nil {
		t.Fatalf("DeleteRelationFk (never recorded): %v", err)
	}
}

func TestIntrospectModelSchema_OutgoingManyToOne(t *testing.T) {
	db := setupDB(t)
	physOne := pgTableName("RelOneSide")
	physMany := pgTableName("RelManySide")
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
	fkName := "fk_rel_many_side_one_side"
	if _, err := db.Exec(`
		CREATE TABLE ` + physMany + ` (
			id serial PRIMARY KEY,
			one_side_id integer NOT NULL CONSTRAINT ` + fkName + ` REFERENCES ` + physOne + `(id)
		)
	`); err != nil {
		t.Fatalf("create %s: %v", physMany, err)
	}
	fk := model.RelationFk{
		Type:             model.RelationTypeManyToOne,
		HomeTable:        "public." + physMany,
		HomeModel:        "RelManySide",
		HomeAttribute:    "oneSide",
		RelatedTable:     "public." + physOne,
		RelatedModel:     "RelOneSide",
		RelatedAttribute: "manySides",
	}
	if err := model.SetRelationFk(db, fkName, fk); err != nil {
		t.Fatalf("SetRelationFk: %v", err)
	}

	schema, err := model.IntrospectModelSchema(db, physMany, "public", false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema: %v", err)
	}
	if _, ok := schema.FieldByName("one_side_id"); ok {
		t.Fatal("the FK column should not appear in Fields")
	}
	rel, ok := schema.RelationByName("oneSide")
	if !ok {
		t.Fatal("expected an \"oneSide\" relation")
	}
	if rel.Type != model.RelationTypeManyToOne || rel.RelatedModel != "RelOneSide" || rel.RelatedAttribute != "manySides" || rel.FkHolder {
		t.Fatalf("oneSide = %#v", rel)
	}
}

func TestIntrospectModelSchema_OutgoingOneToOneFkHolder(t *testing.T) {
	db := setupDB(t)
	physA := pgTableName("RelOneToOneA")
	physB := pgTableName("RelOneToOneB")
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
	fkName := "fk_rel_one_to_one_a_b"
	if _, err := db.Exec(`
		CREATE TABLE ` + physA + ` (
			id serial PRIMARY KEY,
			b_id integer NOT NULL CONSTRAINT ` + fkName + ` REFERENCES ` + physB + `(id) UNIQUE
		)
	`); err != nil {
		t.Fatalf("create %s: %v", physA, err)
	}
	fk := model.RelationFk{
		Type:             model.RelationTypeOneToOne,
		HomeTable:        "public." + physA,
		HomeModel:        "RelOneToOneA",
		HomeAttribute:    "b",
		RelatedTable:     "public." + physB,
		RelatedModel:     "RelOneToOneB",
		RelatedAttribute: "a",
	}
	if err := model.SetRelationFk(db, fkName, fk); err != nil {
		t.Fatalf("SetRelationFk: %v", err)
	}

	schema, err := model.IntrospectModelSchema(db, physA, "public", false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema: %v", err)
	}
	rel, ok := schema.RelationByName("b")
	if !ok || rel.Type != model.RelationTypeOneToOne || !rel.FkHolder || rel.RelatedModel != "RelOneToOneB" || rel.RelatedAttribute != "a" {
		t.Fatalf("b = %#v, %v", rel, ok)
	}
}

func TestIntrospectModelSchema_IncomingRelations(t *testing.T) {
	db := setupDB(t)
	physOne := pgTableName("RelIncomingOne")
	physMany := pgTableName("RelIncomingMany")
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
	fkName := "fk_rel_incoming_many_one"
	if _, err := db.Exec(`
		CREATE TABLE ` + physMany + ` (
			id serial PRIMARY KEY,
			one_id integer NOT NULL CONSTRAINT ` + fkName + ` REFERENCES ` + physOne + `(id)
		)
	`); err != nil {
		t.Fatalf("create %s: %v", physMany, err)
	}
	fk := model.RelationFk{
		Type:             model.RelationTypeManyToOne,
		HomeTable:        "public." + physMany,
		HomeModel:        "RelIncomingMany",
		HomeAttribute:    "one",
		RelatedTable:     "public." + physOne,
		RelatedModel:     "RelIncomingOne",
		RelatedAttribute: "manys",
	}
	if err := model.SetRelationFk(db, fkName, fk); err != nil {
		t.Fatalf("SetRelationFk: %v", err)
	}

	schema, err := model.IntrospectModelSchema(db, physOne, "public", false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema: %v", err)
	}
	rel, ok := schema.RelationByName("manys")
	if !ok {
		t.Fatal("expected a \"manys\" relation restored from the incoming FK")
	}
	if rel.Type != model.RelationTypeOneToMany || rel.RelatedModel != "RelIncomingMany" || rel.RelatedAttribute != "one" || rel.FkHolder {
		t.Fatalf("manys = %#v", rel)
	}
}

// TestIntrospectModelSchema_IncomingRelations_NotFooledByConstraintTypeCollision
// checks incomingRelations's discovery query directly: a UNIQUE constraint
// on tableName itself, coincidentally sharing a name with a real but
// entirely unrelated foreign key elsewhere (one that doesn't reference
// tableName at all), must not be discovered as an "incoming foreign key"
// - joining information_schema.table_constraints/constraint_column_usage
// by name alone (rather than through pg_constraint) could otherwise match
// the real FK's table_constraints row against the UNIQUE constraint's own
// constraint_column_usage row (both share the same constraint_name, and
// constraint_column_usage lists a UNIQUE constraint's own table under
// that name), falsely reporting the real FK as referencing tableName -
// which, since nothing was ever recorded under that name for tableName,
// would break IntrospectModelSchema entirely with a spurious
// "no recorded meaning" error, even though the collision has nothing to
// do with tableName's actual relations.
func TestIntrospectModelSchema_IncomingRelations_NotFooledByConstraintTypeCollision(t *testing.T) {
	db := setupDB(t)
	physOne := pgTableName("RelIncomingTypeCollideOne")
	physRealOwner := pgTableName("RelIncomingTypeCollideOwner")
	physRealTarget := pgTableName("RelIncomingTypeCollideTarget")
	for _, tbl := range []string{physRealOwner, physOne, physRealTarget} {
		if _, err := db.Exec("DROP TABLE IF EXISTS " + tbl); err != nil {
			t.Fatalf("drop %s: %v", tbl, err)
		}
	}
	t.Cleanup(func() {
		db.Exec("DROP TABLE IF EXISTS " + physRealOwner)
		db.Exec("DROP TABLE IF EXISTS " + physOne)
		db.Exec("DROP TABLE IF EXISTS " + physRealTarget)
	})

	sharedName := "fk_rel_incoming_type_collide"
	if _, err := db.Exec(`CREATE TABLE ` + physOne + ` (id serial PRIMARY KEY, CONSTRAINT ` + sharedName + ` UNIQUE (id))`); err != nil {
		t.Fatalf("create %s: %v", physOne, err)
	}
	if _, err := db.Exec(`CREATE TABLE ` + physRealTarget + ` (id serial PRIMARY KEY)`); err != nil {
		t.Fatalf("create %s: %v", physRealTarget, err)
	}
	// A real foreign key elsewhere, unrelated to physOne, happens to reuse
	// the exact same constraint name as physOne's own UNIQUE constraint -
	// a legitimate coincidence, constraint names are only unique per table.
	if _, err := db.Exec(`
		CREATE TABLE ` + physRealOwner + ` (
			id serial PRIMARY KEY,
			ref_id integer NOT NULL CONSTRAINT ` + sharedName + ` REFERENCES ` + physRealTarget + `(id)
		)
	`); err != nil {
		t.Fatalf("create %s: %v", physRealOwner, err)
	}

	schema, err := model.IntrospectModelSchema(db, physOne, "public", false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema: %v", err)
	}
	if len(schema.Relations) != 0 {
		t.Fatalf("expected no relations from the constraint-type collision, got %#v", schema.Relations)
	}
}

func TestIntrospectModelSchema_IncomingUniRelationSkipped(t *testing.T) {
	db := setupDB(t)
	physOne := pgTableName("RelUniOne")
	physMany := pgTableName("RelUniMany")
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
	fkName := "fk_rel_uni_many_one"
	if _, err := db.Exec(`
		CREATE TABLE ` + physMany + ` (
			id serial PRIMARY KEY,
			one_id integer NOT NULL CONSTRAINT ` + fkName + ` REFERENCES ` + physOne + `(id)
		)
	`); err != nil {
		t.Fatalf("create %s: %v", physMany, err)
	}
	// A uni relation - the "many" side declares no RelatedAttribute at all,
	// meaning the "one" side never named a relation back.
	fk := model.RelationFk{
		Type:          model.RelationTypeManyToOne,
		HomeTable:     "public." + physMany,
		HomeModel:     "RelUniMany",
		HomeAttribute: "one",
		RelatedTable:  "public." + physOne,
		RelatedModel:  "RelUniOne",
	}
	if err := model.SetRelationFk(db, fkName, fk); err != nil {
		t.Fatalf("SetRelationFk: %v", err)
	}

	schema, err := model.IntrospectModelSchema(db, physOne, "public", false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema: %v", err)
	}
	if len(schema.Relations) != 0 {
		t.Fatalf("expected no relations restored on the uni side, got %#v", schema.Relations)
	}
}

func TestIntrospectModelSchema_ManyToMany(t *testing.T) {
	db := setupDB(t)
	physA := pgTableName("RelM2mA")
	physB := pgTableName("RelM2mB")
	physJoin := "rel_m2m_a_rel_m2m_b"
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
	// The join table's own FK constraints are explicitly named to match
	// the fk_name each side's metadata row is recorded under below - real
	// DDL that creates a many-to-many join table is expected to do the
	// same, so that IntrospectModelSchema's generic incoming-FK scan
	// (which also sees these two columns from each side) can recognize
	// and skip them (see db_introspect.go's incomingRelations).
	if _, err := db.Exec(`
		CREATE TABLE ` + physJoin + ` (
			a_id integer NOT NULL CONSTRAINT fk_rel_m2m_a_side REFERENCES ` + physA + `(id),
			b_id integer NOT NULL CONSTRAINT fk_rel_m2m_b_side REFERENCES ` + physB + `(id)
		)
	`); err != nil {
		t.Fatalf("create %s: %v", physJoin, err)
	}

	// Two metadata rows, one per side - matching how a real many-to-many
	// relation's DDL execution records it.
	fkA := model.RelationFk{
		Type: model.RelationTypeManyToMany, HomeTable: "public." + physA, HomeModel: "RelM2mA", HomeAttribute: "bs",
		RelatedTable: "public." + physB, RelatedModel: "RelM2mB", RelatedAttribute: "as",
	}
	fkB := model.RelationFk{
		Type: model.RelationTypeManyToMany, HomeTable: "public." + physB, HomeModel: "RelM2mB", HomeAttribute: "as",
		RelatedTable: "public." + physA, RelatedModel: "RelM2mA", RelatedAttribute: "bs",
	}
	if err := model.SetRelationFk(db, "fk_rel_m2m_a_side", fkA); err != nil {
		t.Fatalf("SetRelationFk A: %v", err)
	}
	if err := model.SetRelationFk(db, "fk_rel_m2m_b_side", fkB); err != nil {
		t.Fatalf("SetRelationFk B: %v", err)
	}

	schemaA, err := model.IntrospectModelSchema(db, physA, "public", false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema(A): %v", err)
	}
	relA, ok := schemaA.RelationByName("bs")
	if !ok || relA.Type != model.RelationTypeManyToMany || relA.RelatedModel != "RelM2mB" || relA.RelatedAttribute != "as" {
		t.Fatalf("A.bs = %#v, %v", relA, ok)
	}

	schemaB, err := model.IntrospectModelSchema(db, physB, "public", false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema(B): %v", err)
	}
	relB, ok := schemaB.RelationByName("as")
	if !ok || relB.Type != model.RelationTypeManyToMany || relB.RelatedModel != "RelM2mA" || relB.RelatedAttribute != "bs" {
		t.Fatalf("B.as = %#v, %v", relB, ok)
	}
}

// TestIntrospectModelSchema_ManyToMany_OrphanedRowIgnored checks the gap a
// many-to-many relation's recorded metadata has, unlike the outgoing/
// incoming paths (see loadManyToManyRelationFks's doc): if the join table
// is dropped outside this package, the metadata row alone must not make
// IntrospectModelSchema report a relation that no longer physically
// exists.
func TestIntrospectModelSchema_ManyToMany_OrphanedRowIgnored(t *testing.T) {
	db := setupDB(t)
	physA := pgTableName("RelM2mOrphanA")
	physB := pgTableName("RelM2mOrphanB")
	physJoin := "rel_m2m_orphan_a_rel_m2m_orphan_b"
	for _, tbl := range []string{physJoin, physA, physB} {
		if _, err := db.Exec("DROP TABLE IF EXISTS " + tbl); err != nil {
			t.Fatalf("drop %s: %v", tbl, err)
		}
	}
	t.Cleanup(func() {
		db.Exec("DROP TABLE IF EXISTS " + physJoin)
		db.Exec("DROP TABLE IF EXISTS " + physA)
		db.Exec("DROP TABLE IF EXISTS " + physB)
		db.Exec(`DELETE FROM lx_sys.model_relations WHERE home_table = $1`, "public."+physA)
	})

	if _, err := db.Exec(`CREATE TABLE ` + physA + ` (id serial PRIMARY KEY)`); err != nil {
		t.Fatalf("create %s: %v", physA, err)
	}
	if _, err := db.Exec(`CREATE TABLE ` + physB + ` (id serial PRIMARY KEY)`); err != nil {
		t.Fatalf("create %s: %v", physB, err)
	}
	fkName := "fk_rel_m2m_orphan_a_side"
	if _, err := db.Exec(`
		CREATE TABLE ` + physJoin + ` (
			a_id integer NOT NULL CONSTRAINT ` + fkName + ` REFERENCES ` + physA + `(id),
			b_id integer NOT NULL REFERENCES ` + physB + `(id)
		)
	`); err != nil {
		t.Fatalf("create %s: %v", physJoin, err)
	}

	fk := model.RelationFk{
		Type: model.RelationTypeManyToMany, HomeTable: "public." + physA, HomeModel: "RelM2mOrphanA", HomeAttribute: "bs",
		RelatedTable: "public." + physB, RelatedModel: "RelM2mOrphanB", RelatedAttribute: "as",
	}
	if err := model.SetRelationFk(db, fkName, fk); err != nil {
		t.Fatalf("SetRelationFk: %v", err)
	}

	schema, err := model.IntrospectModelSchema(db, physA, "public", false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema (before drop): %v", err)
	}
	if _, ok := schema.RelationByName("bs"); !ok {
		t.Fatal("expected the relation to be visible before the join table is dropped")
	}

	// The join table is dropped outside this package - the metadata row in
	// the service table is now orphaned (fkName no longer names anything).
	if _, err := db.Exec("DROP TABLE " + physJoin); err != nil {
		t.Fatalf("drop %s: %v", physJoin, err)
	}

	schema, err = model.IntrospectModelSchema(db, physA, "public", false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema (after drop): %v", err)
	}
	if _, ok := schema.RelationByName("bs"); ok {
		t.Fatal("the orphaned metadata row should not resurrect a relation whose join table no longer exists")
	}
}

// TestIntrospectModelSchema_ManyToMany_OrphanedRowNotResolvedByNameCollision
// is TestIntrospectModelSchema_ManyToMany_OrphanedRowIgnored's adversarial
// case: after the join table is dropped, an unrelated foreign key
// elsewhere in the schema reuses the exact same constraint name (Postgres
// constraint names are only unique per table, not database-wide, so this
// is a legitimate coincidence, not a contrived setup). The orphaned row
// must stay ignored - a same-named FK that references a different table
// must not be mistaken for the dropped join table's own FK.
func TestIntrospectModelSchema_ManyToMany_OrphanedRowNotResolvedByNameCollision(t *testing.T) {
	db := setupDB(t)
	physA := pgTableName("RelM2mCollideA")
	physB := pgTableName("RelM2mCollideB")
	physJoin := "rel_m2m_collide_a_rel_m2m_collide_b"
	physUnrelated := pgTableName("RelM2mCollideUnrelated")
	physUnrelatedOwner := pgTableName("RelM2mCollideUnrelatedOwner")
	for _, tbl := range []string{physJoin, physUnrelatedOwner, physA, physB, physUnrelated} {
		if _, err := db.Exec("DROP TABLE IF EXISTS " + tbl); err != nil {
			t.Fatalf("drop %s: %v", tbl, err)
		}
	}
	t.Cleanup(func() {
		db.Exec("DROP TABLE IF EXISTS " + physUnrelatedOwner)
		db.Exec("DROP TABLE IF EXISTS " + physJoin)
		db.Exec("DROP TABLE IF EXISTS " + physA)
		db.Exec("DROP TABLE IF EXISTS " + physB)
		db.Exec("DROP TABLE IF EXISTS " + physUnrelated)
		db.Exec(`DELETE FROM lx_sys.model_relations WHERE home_table = $1`, "public."+physA)
	})

	if _, err := db.Exec(`CREATE TABLE ` + physA + ` (id serial PRIMARY KEY)`); err != nil {
		t.Fatalf("create %s: %v", physA, err)
	}
	if _, err := db.Exec(`CREATE TABLE ` + physB + ` (id serial PRIMARY KEY)`); err != nil {
		t.Fatalf("create %s: %v", physB, err)
	}
	fkName := "fk_rel_m2m_collide_shared_name"
	if _, err := db.Exec(`
		CREATE TABLE ` + physJoin + ` (
			a_id integer NOT NULL CONSTRAINT ` + fkName + ` REFERENCES ` + physA + `(id),
			b_id integer NOT NULL REFERENCES ` + physB + `(id)
		)
	`); err != nil {
		t.Fatalf("create %s: %v", physJoin, err)
	}

	fk := model.RelationFk{
		Type: model.RelationTypeManyToMany, HomeTable: "public." + physA, HomeModel: "RelM2mCollideA", HomeAttribute: "bs",
		RelatedTable: "public." + physB, RelatedModel: "RelM2mCollideB", RelatedAttribute: "as",
	}
	if err := model.SetRelationFk(db, fkName, fk); err != nil {
		t.Fatalf("SetRelationFk: %v", err)
	}

	// The join table is dropped - the metadata row is now orphaned.
	if _, err := db.Exec("DROP TABLE " + physJoin); err != nil {
		t.Fatalf("drop %s: %v", physJoin, err)
	}

	// A completely unrelated foreign key, elsewhere in the schema, happens
	// to reuse the exact same constraint name - a same-table Postgres
	// constraint namespace, nothing prevents this by itself.
	if _, err := db.Exec(`CREATE TABLE ` + physUnrelated + ` (id serial PRIMARY KEY)`); err != nil {
		t.Fatalf("create %s: %v", physUnrelated, err)
	}
	if _, err := db.Exec(`
		CREATE TABLE ` + physUnrelatedOwner + ` (
			id serial PRIMARY KEY,
			ref_id integer NOT NULL CONSTRAINT ` + fkName + ` REFERENCES ` + physUnrelated + `(id)
		)
	`); err != nil {
		t.Fatalf("create %s: %v", physUnrelatedOwner, err)
	}

	schema, err := model.IntrospectModelSchema(db, physA, "public", false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema: %v", err)
	}
	if _, ok := schema.RelationByName("bs"); ok {
		t.Fatal("an unrelated FK reusing the same constraint name must not resurrect the orphaned many-to-many relation")
	}
}

// TestIntrospectModelSchema_ManyToMany_OrphanedRowNotResolvedByConstraintTypeCollision
// is a sharper adversarial case than
// TestIntrospectModelSchema_ManyToMany_OrphanedRowNotResolvedByNameCollision:
// a same-named constraint sitting directly ON home_table (not elsewhere)
// but of a different constraint type (UNIQUE, not FOREIGN KEY). Joining
// information_schema.table_constraints/constraint_column_usage by name
// alone (rather than through pg_constraint, which resolves a specific
// constraint's referenced table directly from the same row) could match
// the unrelated FOREIGN KEY's table_constraints row against this UNIQUE
// constraint's constraint_column_usage row (both share the same
// constraint_name, and constraint_column_usage lists a UNIQUE
// constraint's own table under that name) - falsely "proving" the dropped
// join table's foreign key still references home_table.
func TestIntrospectModelSchema_ManyToMany_OrphanedRowNotResolvedByConstraintTypeCollision(t *testing.T) {
	db := setupDB(t)
	physA := pgTableName("RelM2mTypeCollideA")
	physB := pgTableName("RelM2mTypeCollideB")
	physJoin := "rel_m2m_type_collide_a_b"
	physUnrelated := pgTableName("RelM2mTypeCollideUnrelated")
	physUnrelatedOwner := pgTableName("RelM2mTypeCollideUnrelatedOwner")
	for _, tbl := range []string{physJoin, physUnrelatedOwner, physA, physB, physUnrelated} {
		if _, err := db.Exec("DROP TABLE IF EXISTS " + tbl); err != nil {
			t.Fatalf("drop %s: %v", tbl, err)
		}
	}
	t.Cleanup(func() {
		db.Exec("DROP TABLE IF EXISTS " + physUnrelatedOwner)
		db.Exec("DROP TABLE IF EXISTS " + physJoin)
		db.Exec("DROP TABLE IF EXISTS " + physA)
		db.Exec("DROP TABLE IF EXISTS " + physB)
		db.Exec("DROP TABLE IF EXISTS " + physUnrelated)
		db.Exec(`DELETE FROM lx_sys.model_relations WHERE home_table = $1`, "public."+physA)
	})

	if _, err := db.Exec(`CREATE TABLE ` + physA + ` (id serial PRIMARY KEY)`); err != nil {
		t.Fatalf("create %s: %v", physA, err)
	}
	if _, err := db.Exec(`CREATE TABLE ` + physB + ` (id serial PRIMARY KEY)`); err != nil {
		t.Fatalf("create %s: %v", physB, err)
	}
	fkName := "fk_rel_m2m_type_collide_shared"
	if _, err := db.Exec(`
		CREATE TABLE ` + physJoin + ` (
			a_id integer NOT NULL CONSTRAINT ` + fkName + ` REFERENCES ` + physA + `(id),
			b_id integer NOT NULL REFERENCES ` + physB + `(id)
		)
	`); err != nil {
		t.Fatalf("create %s: %v", physJoin, err)
	}

	fk := model.RelationFk{
		Type: model.RelationTypeManyToMany, HomeTable: "public." + physA, HomeModel: "RelM2mTypeCollideA", HomeAttribute: "bs",
		RelatedTable: "public." + physB, RelatedModel: "RelM2mTypeCollideB", RelatedAttribute: "as",
	}
	if err := model.SetRelationFk(db, fkName, fk); err != nil {
		t.Fatalf("SetRelationFk: %v", err)
	}

	// The join table is dropped - the metadata row is now orphaned.
	if _, err := db.Exec("DROP TABLE " + physJoin); err != nil {
		t.Fatalf("drop %s: %v", physJoin, err)
	}

	// An unrelated foreign key elsewhere reuses the same constraint name...
	if _, err := db.Exec(`CREATE TABLE ` + physUnrelated + ` (id serial PRIMARY KEY)`); err != nil {
		t.Fatalf("create %s: %v", physUnrelated, err)
	}
	if _, err := db.Exec(`
		CREATE TABLE ` + physUnrelatedOwner + ` (
			id serial PRIMARY KEY,
			ref_id integer NOT NULL CONSTRAINT ` + fkName + ` REFERENCES ` + physUnrelated + `(id)
		)
	`); err != nil {
		t.Fatalf("create %s: %v", physUnrelatedOwner, err)
	}
	// ...and, separately, home_table itself gets an ordinary UNIQUE
	// constraint that happens to reuse the same name too - a legitimate
	// coincidence, constraint names are only unique per table.
	if _, err := db.Exec(`ALTER TABLE ` + physA + ` ADD CONSTRAINT ` + fkName + ` UNIQUE (id)`); err != nil {
		t.Fatalf("add unique constraint: %v", err)
	}

	schema, err := model.IntrospectModelSchema(db, physA, "public", false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema: %v", err)
	}
	if _, ok := schema.RelationByName("bs"); ok {
		t.Fatal("a same-named UNIQUE constraint on home_table must not resurrect the orphaned many-to-many relation")
	}
}

// TestCompareModel_RelationsDedup is the main risk canIgnoreRelation
// exists for: a relation declared symmetrically in both schema files must
// only ever produce an Added/Deleted entry on the acting side - the
// passive side's ModelDiff must not also report it, or applying "every
// diff found" would try to create/drop the same physical FK twice.
func TestCompareModel_RelationsDedup(t *testing.T) {
	db := setupDB(t)
	physOne := pgTableName("RelDedupOne")
	physMany := pgTableName("RelDedupMany")
	for _, tbl := range []string{physMany, physOne} {
		if _, err := db.Exec("DROP TABLE IF EXISTS " + tbl); err != nil {
			t.Fatalf("drop %s: %v", tbl, err)
		}
	}
	t.Cleanup(func() {
		db.Exec("DROP TABLE IF EXISTS " + physMany)
		db.Exec("DROP TABLE IF EXISTS " + physOne)
	})

	if _, err := db.Exec(`CREATE TABLE ` + physOne + ` (id serial PRIMARY KEY)`); err != nil {
		t.Fatalf("create %s: %v", physOne, err)
	}
	if _, err := db.Exec(`CREATE TABLE ` + physMany + ` (id serial PRIMARY KEY)`); err != nil {
		t.Fatalf("create %s: %v", physMany, err)
	}

	// Neither table has the physical FK yet - both schema files declare
	// the relation (symmetric), so a naive per-side diff would report
	// "Added" on BOTH sides. Only RelDedupMany (manyToOne, always acting)
	// should actually report it.
	oneSchema := &model.ModelSchema{
		Name: "RelDedupOne",
		Relations: []model.NamedRelation{
			{Name: "manys", Relation: model.Relation{Type: model.RelationTypeOneToMany, RelatedModel: "RelDedupMany", RelatedAttribute: "one"}},
		},
	}
	manySchema := &model.ModelSchema{
		Name: "RelDedupMany",
		Relations: []model.NamedRelation{
			{Name: "one", Relation: model.Relation{Type: model.RelationTypeManyToOne, RelatedModel: "RelDedupOne", RelatedAttribute: "manys"}},
		},
	}

	oneDiff, err := model.CompareModel(db, oneSchema)
	if err != nil {
		t.Fatalf("CompareModel(one): %v", err)
	}
	if !oneDiff.Relations.IsEmpty() {
		t.Fatalf("the oneToMany (passive) side should report no relation diff, got %#v", oneDiff.Relations)
	}

	manyDiff, err := model.CompareModel(db, manySchema)
	if err != nil {
		t.Fatalf("CompareModel(many): %v", err)
	}
	if len(manyDiff.Relations.Added) != 1 || manyDiff.Relations.Added[0] != "one" {
		t.Fatalf("the manyToOne (acting) side should report Added: [one], got %#v", manyDiff.Relations)
	}
}

// TestCompareModel_RelationsDedup_OneToOne is TestCompareModel_RelationsDedup
// for RelationTypeOneToOne - only the FkHolder side should act.
func TestCompareModel_RelationsDedup_OneToOne(t *testing.T) {
	db := setupDB(t)
	physA := pgTableName("RelDedupOneToOneA")
	physB := pgTableName("RelDedupOneToOneB")
	for _, tbl := range []string{physA, physB} {
		if _, err := db.Exec("DROP TABLE IF EXISTS " + tbl); err != nil {
			t.Fatalf("drop %s: %v", tbl, err)
		}
	}
	t.Cleanup(func() {
		db.Exec("DROP TABLE IF EXISTS " + physA)
		db.Exec("DROP TABLE IF EXISTS " + physB)
	})

	if _, err := db.Exec(`CREATE TABLE ` + physA + ` (id serial PRIMARY KEY)`); err != nil {
		t.Fatalf("create %s: %v", physA, err)
	}
	if _, err := db.Exec(`CREATE TABLE ` + physB + ` (id serial PRIMARY KEY)`); err != nil {
		t.Fatalf("create %s: %v", physB, err)
	}

	// A holds the FK (FkHolder: true), B doesn't - only A should act.
	aSchema := &model.ModelSchema{
		Name: "RelDedupOneToOneA",
		Relations: []model.NamedRelation{
			{Name: "b", Relation: model.Relation{Type: model.RelationTypeOneToOne, RelatedModel: "RelDedupOneToOneB", RelatedAttribute: "a", FkHolder: true}},
		},
	}
	bSchema := &model.ModelSchema{
		Name: "RelDedupOneToOneB",
		Relations: []model.NamedRelation{
			{Name: "a", Relation: model.Relation{Type: model.RelationTypeOneToOne, RelatedModel: "RelDedupOneToOneA", RelatedAttribute: "b", FkHolder: false}},
		},
	}

	aDiff, err := model.CompareModel(db, aSchema)
	if err != nil {
		t.Fatalf("CompareModel(A): %v", err)
	}
	if len(aDiff.Relations.Added) != 1 || aDiff.Relations.Added[0] != "b" {
		t.Fatalf("the FkHolder side should report Added: [b], got %#v", aDiff.Relations)
	}

	bDiff, err := model.CompareModel(db, bSchema)
	if err != nil {
		t.Fatalf("CompareModel(B): %v", err)
	}
	if !bDiff.Relations.IsEmpty() {
		t.Fatalf("the non-FkHolder side should report no relation diff, got %#v", bDiff.Relations)
	}
}

// TestCompareModel_RelationsDedup_ManyToMany is
// TestCompareModel_RelationsDedup for RelationTypeManyToMany - only the
// alphabetically-first model name should act, checked both ways round so
// the outcome isn't an accident of which schema happens to be compared
// first.
func TestCompareModel_RelationsDedup_ManyToMany(t *testing.T) {
	db := setupDB(t)
	physFirst := pgTableName("RelDedupM2mFirst")
	physSecond := pgTableName("RelDedupM2mSecond")
	for _, tbl := range []string{physFirst, physSecond} {
		if _, err := db.Exec("DROP TABLE IF EXISTS " + tbl); err != nil {
			t.Fatalf("drop %s: %v", tbl, err)
		}
	}
	t.Cleanup(func() {
		db.Exec("DROP TABLE IF EXISTS " + physFirst)
		db.Exec("DROP TABLE IF EXISTS " + physSecond)
	})

	if _, err := db.Exec(`CREATE TABLE ` + physFirst + ` (id serial PRIMARY KEY)`); err != nil {
		t.Fatalf("create %s: %v", physFirst, err)
	}
	if _, err := db.Exec(`CREATE TABLE ` + physSecond + ` (id serial PRIMARY KEY)`); err != nil {
		t.Fatalf("create %s: %v", physSecond, err)
	}

	// "RelDedupM2mFirst" < "RelDedupM2mSecond" alphabetically - only the
	// first should act, regardless of which side CompareModel is called on.
	firstSchema := &model.ModelSchema{
		Name: "RelDedupM2mFirst",
		Relations: []model.NamedRelation{
			{Name: "seconds", Relation: model.Relation{Type: model.RelationTypeManyToMany, RelatedModel: "RelDedupM2mSecond", RelatedAttribute: "firsts"}},
		},
	}
	secondSchema := &model.ModelSchema{
		Name: "RelDedupM2mSecond",
		Relations: []model.NamedRelation{
			{Name: "firsts", Relation: model.Relation{Type: model.RelationTypeManyToMany, RelatedModel: "RelDedupM2mFirst", RelatedAttribute: "seconds"}},
		},
	}

	firstDiff, err := model.CompareModel(db, firstSchema)
	if err != nil {
		t.Fatalf("CompareModel(first): %v", err)
	}
	if len(firstDiff.Relations.Added) != 1 || firstDiff.Relations.Added[0] != "seconds" {
		t.Fatalf("the alphabetically-first side should report Added: [seconds], got %#v", firstDiff.Relations)
	}

	secondDiff, err := model.CompareModel(db, secondSchema)
	if err != nil {
		t.Fatalf("CompareModel(second): %v", err)
	}
	if !secondDiff.Relations.IsEmpty() {
		t.Fatalf("the alphabetically-later side should report no relation diff, got %#v", secondDiff.Relations)
	}
}
