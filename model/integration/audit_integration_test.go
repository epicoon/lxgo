//go:build integration

package model_test

import (
	"testing"

	"github.com/epicoon/lxgo/model"
)

func TestAuditRelationFks_CleanWhenNothingOrphaned(t *testing.T) {
	db := setupDB(t)
	physOne := pgTableName("AuditRelCleanOne")
	physMany := pgTableName("AuditRelCleanMany")
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
	fkName := "fk_audit_rel_clean"
	if _, err := db.Exec(`
		CREATE TABLE ` + physMany + ` (
			id serial PRIMARY KEY,
			one_id integer NOT NULL CONSTRAINT ` + fkName + ` REFERENCES ` + physOne + `(id)
		)
	`); err != nil {
		t.Fatalf("create %s: %v", physMany, err)
	}
	fk := model.RelationFk{
		Type: model.RelationTypeManyToOne, HomeTable: "public." + physMany, HomeModel: "AuditRelCleanMany", HomeAttribute: "one",
		RelatedTable: "public." + physOne, RelatedModel: "AuditRelCleanOne", RelatedAttribute: "manys",
	}
	if err := model.SetRelationFk(db, fkName, fk); err != nil {
		t.Fatalf("SetRelationFk: %v", err)
	}

	orphaned, err := model.AuditRelationFks(db)
	if err != nil {
		t.Fatalf("AuditRelationFks: %v", err)
	}
	for _, o := range orphaned {
		if o.FkName == fkName {
			t.Fatalf("a still-valid foreign key was reported as orphaned: %#v", o)
		}
	}
}

func TestAuditRelationFks_FindsOrphanedManyToOne(t *testing.T) {
	db := setupDB(t)
	physOne := pgTableName("AuditRelOrphanOne")
	physMany := pgTableName("AuditRelOrphanMany")
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
	fkName := "fk_audit_rel_orphan"
	if _, err := db.Exec(`
		CREATE TABLE ` + physMany + ` (
			id serial PRIMARY KEY,
			one_id integer CONSTRAINT ` + fkName + ` REFERENCES ` + physOne + `(id)
		)
	`); err != nil {
		t.Fatalf("create %s: %v", physMany, err)
	}
	fk := model.RelationFk{
		Type: model.RelationTypeManyToOne, HomeTable: "public." + physMany, HomeModel: "AuditRelOrphanMany", HomeAttribute: "one",
		RelatedTable: "public." + physOne, RelatedModel: "AuditRelOrphanOne", RelatedAttribute: "manys",
	}
	if err := model.SetRelationFk(db, fkName, fk); err != nil {
		t.Fatalf("SetRelationFk: %v", err)
	}

	// The constraint is dropped outside SetRelationFk/DeleteRelationFk's
	// own path - the metadata row is now orphaned.
	if _, err := db.Exec(`ALTER TABLE ` + physMany + ` DROP CONSTRAINT ` + fkName); err != nil {
		t.Fatalf("drop constraint: %v", err)
	}

	orphaned, err := model.AuditRelationFks(db)
	if err != nil {
		t.Fatalf("AuditRelationFks: %v", err)
	}
	found := false
	for _, o := range orphaned {
		if o.FkName == fkName {
			found = true
			if o.Type != model.RelationTypeManyToOne || o.HomeModel != "AuditRelOrphanMany" {
				t.Fatalf("orphaned row = %#v, unexpected fields", o)
			}
		}
	}
	if !found {
		t.Fatalf("expected %q to be reported as orphaned, got %#v", fkName, orphaned)
	}
}

func TestAuditRelationFks_FindsOrphanedManyToMany(t *testing.T) {
	db := setupDB(t)
	physA := pgTableName("AuditRelM2mOrphanA")
	physB := pgTableName("AuditRelM2mOrphanB")
	physJoin := "audit_rel_m2m_orphan_a_b"
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
	fkName := "fk_audit_rel_m2m_orphan"
	if _, err := db.Exec(`
		CREATE TABLE ` + physJoin + ` (
			a_id integer NOT NULL CONSTRAINT ` + fkName + ` REFERENCES ` + physA + `(id),
			b_id integer NOT NULL REFERENCES ` + physB + `(id)
		)
	`); err != nil {
		t.Fatalf("create %s: %v", physJoin, err)
	}
	fk := model.RelationFk{
		Type: model.RelationTypeManyToMany, HomeTable: "public." + physA, HomeModel: "AuditRelM2mOrphanA", HomeAttribute: "bs",
		RelatedTable: "public." + physB, RelatedModel: "AuditRelM2mOrphanB", RelatedAttribute: "as",
	}
	if err := model.SetRelationFk(db, fkName, fk); err != nil {
		t.Fatalf("SetRelationFk: %v", err)
	}

	if _, err := db.Exec("DROP TABLE " + physJoin); err != nil {
		t.Fatalf("drop %s: %v", physJoin, err)
	}

	orphaned, err := model.AuditRelationFks(db)
	if err != nil {
		t.Fatalf("AuditRelationFks: %v", err)
	}
	found := false
	for _, o := range orphaned {
		if o.FkName == fkName {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected %q to be reported as orphaned, got %#v", fkName, orphaned)
	}
}

// TestAuditRelationFks_OrphanedManyToManyNotResolvedByConstraintTypeCollision
// is relationFkExists's ManyToMany branch under adversarial conditions - a
// same-named UNIQUE constraint directly on home_table (a legitimate
// coincidence, constraint names are only unique per table) must not make
// an orphaned row look valid, the same risk
// TestIntrospectModelSchema_ManyToMany_OrphanedRowNotResolvedByConstraintTypeCollision
// checks for IntrospectModelSchema.
func TestAuditRelationFks_OrphanedManyToManyNotResolvedByConstraintTypeCollision(t *testing.T) {
	db := setupDB(t)
	physA := pgTableName("AuditRelM2mTypeCollideA")
	physB := pgTableName("AuditRelM2mTypeCollideB")
	physJoin := "audit_rel_m2m_type_collide_a_b"
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
	fkName := "fk_audit_rel_m2m_type_collide"
	if _, err := db.Exec(`
		CREATE TABLE ` + physJoin + ` (
			a_id integer NOT NULL CONSTRAINT ` + fkName + ` REFERENCES ` + physA + `(id),
			b_id integer NOT NULL REFERENCES ` + physB + `(id)
		)
	`); err != nil {
		t.Fatalf("create %s: %v", physJoin, err)
	}
	fk := model.RelationFk{
		Type: model.RelationTypeManyToMany, HomeTable: "public." + physA, HomeModel: "AuditRelM2mTypeCollideA", HomeAttribute: "bs",
		RelatedTable: "public." + physB, RelatedModel: "AuditRelM2mTypeCollideB", RelatedAttribute: "as",
	}
	if err := model.SetRelationFk(db, fkName, fk); err != nil {
		t.Fatalf("SetRelationFk: %v", err)
	}

	if _, err := db.Exec("DROP TABLE " + physJoin); err != nil {
		t.Fatalf("drop %s: %v", physJoin, err)
	}
	// home_table itself gets an ordinary UNIQUE constraint reusing the
	// same name as the (now-gone) foreign key.
	if _, err := db.Exec(`ALTER TABLE ` + physA + ` ADD CONSTRAINT ` + fkName + ` UNIQUE (id)`); err != nil {
		t.Fatalf("add unique constraint: %v", err)
	}

	orphaned, err := model.AuditRelationFks(db)
	if err != nil {
		t.Fatalf("AuditRelationFks: %v", err)
	}
	found := false
	for _, o := range orphaned {
		if o.FkName == fkName {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected %q to still be reported as orphaned despite the same-named UNIQUE constraint, got %#v", fkName, orphaned)
	}
}

func TestAuditColumnTypes_CleanWhenNothingOrphaned(t *testing.T) {
	db := setupDB(t)
	if _, err := db.Exec("DROP TABLE IF EXISTS audit_types_clean"); err != nil {
		t.Fatalf("drop audit_types_clean: %v", err)
	}
	t.Cleanup(func() {
		db.Exec("DROP TABLE IF EXISTS audit_types_clean")
		db.Exec(`DELETE FROM lx_sys.model_types WHERE table_name = 'public.audit_types_clean'`)
	})

	if _, err := db.Exec(`CREATE TABLE audit_types_clean (id serial PRIMARY KEY, settings text)`); err != nil {
		t.Fatalf("create audit_types_clean: %v", err)
	}
	if err := model.SetColumnType(db, "public.audit_types_clean", "settings", model.Field{Type: model.FieldTypeDict}); err != nil {
		t.Fatalf("SetColumnType: %v", err)
	}

	orphaned, err := model.AuditColumnTypes(db)
	if err != nil {
		t.Fatalf("AuditColumnTypes: %v", err)
	}
	for _, o := range orphaned {
		if o.TableName == "public.audit_types_clean" && o.ColumnName == "settings" {
			t.Fatalf("a still-valid column type was reported as orphaned: %#v", o)
		}
	}
}

func TestAuditColumnTypes_FindsOrphanedColumn(t *testing.T) {
	db := setupDB(t)
	if _, err := db.Exec("DROP TABLE IF EXISTS audit_types_orphan"); err != nil {
		t.Fatalf("drop audit_types_orphan: %v", err)
	}
	t.Cleanup(func() {
		db.Exec("DROP TABLE IF EXISTS audit_types_orphan")
		db.Exec(`DELETE FROM lx_sys.model_types WHERE table_name = 'public.audit_types_orphan'`)
	})

	if _, err := db.Exec(`CREATE TABLE audit_types_orphan (id serial PRIMARY KEY, settings text)`); err != nil {
		t.Fatalf("create audit_types_orphan: %v", err)
	}
	if err := model.SetColumnType(db, "public.audit_types_orphan", "settings", model.Field{Type: model.FieldTypeDict}); err != nil {
		t.Fatalf("SetColumnType: %v", err)
	}

	// The column is dropped outside DeleteColumnType's own path - the
	// metadata row is now orphaned.
	if _, err := db.Exec(`ALTER TABLE audit_types_orphan DROP COLUMN settings`); err != nil {
		t.Fatalf("drop column: %v", err)
	}

	orphaned, err := model.AuditColumnTypes(db)
	if err != nil {
		t.Fatalf("AuditColumnTypes: %v", err)
	}
	found := false
	for _, o := range orphaned {
		if o.TableName == "public.audit_types_orphan" && o.ColumnName == "settings" {
			found = true
			if o.Type != model.FieldTypeDict {
				t.Fatalf("orphaned row = %#v, want FieldTypeDict", o)
			}
		}
	}
	if !found {
		t.Fatalf("expected audit_types_orphan.settings to be reported as orphaned, got %#v", orphaned)
	}
}
