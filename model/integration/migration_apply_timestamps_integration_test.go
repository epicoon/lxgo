//go:build integration

package model_test

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/epicoon/lxgo/migrator"
	"github.com/epicoon/lxgo/model"
)

// timestampColumns reports which of created_at/updated_at/deleted_at exist
// on physTable ("public" schema) - used to check execCreateTable/
// execAddTimestamps/execDelTimestamps's actual physical effect directly,
// since these columns are deliberately excluded from IntrospectModelSchema's
// own Fields result (see model.IntrospectModelSchema's doc).
func timestampColumns(t *testing.T, db *sql.DB, physTable string) map[string]bool {
	t.Helper()
	rows, err := db.Query(`
		SELECT column_name FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = $1
			AND column_name IN ('created_at', 'updated_at', 'deleted_at')
	`, physTable)
	if err != nil {
		t.Fatalf("querying timestamp columns of %q: %v", physTable, err)
	}
	defer rows.Close()

	got := map[string]bool{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scanning timestamp column of %q: %v", physTable, err)
		}
		got[name] = true
	}
	return got
}

func timestampsIndexExists(t *testing.T, db *sql.DB, physTable string) bool {
	t.Helper()
	var exists bool
	err := db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM pg_indexes WHERE schemaname = 'public' AND tablename = $1 AND indexname = $2
		)
	`, physTable, "idx__"+physTable+"__deleted_at").Scan(&exists)
	if err != nil {
		t.Fatalf("checking deleted_at index on %q: %v", physTable, err)
	}
	return exists
}

// indexNamesOnDeletedAt lists every single-column index on physTable's own
// deleted_at column (deleted_at alone in the index's own column list, not
// merely present in a composite one), by whatever name it was created
// under - used to check that AddTimestamps never creates a redundant
// duplicate index next to an existing one under an arbitrary name.
func indexNamesOnDeletedAt(t *testing.T, db *sql.DB, physTable string) []string {
	t.Helper()
	rows, err := db.Query(`
		SELECT indexname FROM pg_indexes
		WHERE schemaname = 'public' AND tablename = $1 AND indexdef LIKE '%(deleted_at)%'
		ORDER BY indexname
	`, physTable)
	if err != nil {
		t.Fatalf("listing indexes on deleted_at of %q: %v", physTable, err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("reading index name on %q: %v", physTable, err)
		}
		names = append(names, name)
	}
	return names
}

// TestApplyInvert_CreateTable_WithTimestamps checks execCreateTable's own
// Timestamps handling end to end: a model with Timestamps: true gets
// created_at/updated_at/deleted_at columns (NOT NULL DEFAULT now() for the
// first two, nullable for the third) and an index on deleted_at, none of
// which show up as a Field via IntrospectModelSchema (they're implicit,
// the same way "id" is) - and Down/Invert removes the table (and, with it,
// the columns) entirely, the same as any other CreateTable/DropTable pair.
func TestApplyInvert_CreateTable_WithTimestamps(t *testing.T) {
	db := setupDB(t)
	physTable := pgTableName("ts_ddl_widgets")
	db.Exec("DROP TABLE IF EXISTS " + physTable)
	t.Cleanup(func() { db.Exec("DROP TABLE IF EXISTS " + physTable) })

	migrationsDir := t.TempDir()
	db.Exec("DROP TABLE IF EXISTS lx_sys.migrator")
	migrator.Init(migrator.Config{DB: db, MigrationsPath: migrationsDir})

	schemaDir := t.TempDir()
	tru := true
	sch := &model.ModelSchema{
		Name:       "ts_ddl_widgets",
		Timestamps: &tru,
		Fields: []model.NamedField{
			{Name: "name", Field: model.Field{Type: model.FieldTypeString, Size: 100, Required: true}},
		},
	}
	if err := sch.Save(filepath.Join(schemaDir, "ts_ddl_widgets.yaml")); err != nil {
		t.Fatalf("save schema: %v", err)
	}

	schemas := loadTestSchemas(t, schemaDir)
	if _, err := model.GenerateMigration(db, schemas, "create_ts_ddl_widgets"); err != nil {
		t.Fatalf("GenerateMigration: %v", err)
	}
	migrator.Up()

	cols := timestampColumns(t, db, physTable)
	if !cols["created_at"] || !cols["updated_at"] || !cols["deleted_at"] {
		t.Fatalf("timestamp columns after Up = %v, want all three present", cols)
	}
	if !timestampsIndexExists(t, db, physTable) {
		t.Fatal("expected deleted_at index after Up")
	}

	var createdNullable, deletedNullable string
	if err := db.QueryRow(`
		SELECT is_nullable FROM information_schema.columns WHERE table_schema = 'public' AND table_name = $1 AND column_name = 'created_at'
	`, physTable).Scan(&createdNullable); err != nil {
		t.Fatalf("querying created_at nullability: %v", err)
	}
	if createdNullable != "NO" {
		t.Fatalf("created_at is_nullable = %q, want NO", createdNullable)
	}
	if err := db.QueryRow(`
		SELECT is_nullable FROM information_schema.columns WHERE table_schema = 'public' AND table_name = $1 AND column_name = 'deleted_at'
	`, physTable).Scan(&deletedNullable); err != nil {
		t.Fatalf("querying deleted_at nullability: %v", err)
	}
	if deletedNullable != "YES" {
		t.Fatalf("deleted_at is_nullable = %q, want YES", deletedNullable)
	}

	introspected, err := model.IntrospectModelSchema(db, physTable, "public", true)
	if err != nil {
		t.Fatalf("IntrospectModelSchema after Up: %v", err)
	}
	for _, name := range []string{"created_at", "updated_at", "deleted_at"} {
		if _, ok := introspected.FieldByName(name); ok {
			t.Fatalf("IntrospectModelSchema reported %q as a Field, want it implicit like id", name)
		}
	}

	migrator.Down(0)

	if _, err := model.IntrospectModelSchema(db, physTable, "public", true); err != model.ErrTableNotFound {
		t.Fatalf("err after Down = %v, want ErrTableNotFound", err)
	}
}

// TestApplyInvert_CreateTable_WithoutTimestamps checks that a model with no
// Timestamps override (and no component-wide default, see loadTestSchemas)
// gets none of the three columns - execCreateTable's Timestamps handling is
// opt-in, not the new default.
func TestApplyInvert_CreateTable_WithoutTimestamps(t *testing.T) {
	db := setupDB(t)
	physTable := pgTableName("ts_ddl_no_widgets")
	db.Exec("DROP TABLE IF EXISTS " + physTable)
	t.Cleanup(func() { db.Exec("DROP TABLE IF EXISTS " + physTable) })

	migrationsDir := t.TempDir()
	db.Exec("DROP TABLE IF EXISTS lx_sys.migrator")
	migrator.Init(migrator.Config{DB: db, MigrationsPath: migrationsDir})

	schemaDir := t.TempDir()
	sch := &model.ModelSchema{
		Name: "ts_ddl_no_widgets",
		Fields: []model.NamedField{
			{Name: "name", Field: model.Field{Type: model.FieldTypeString, Size: 100, Required: true}},
		},
	}
	if err := sch.Save(filepath.Join(schemaDir, "ts_ddl_no_widgets.yaml")); err != nil {
		t.Fatalf("save schema: %v", err)
	}

	schemas := loadTestSchemas(t, schemaDir)
	if _, err := model.GenerateMigration(db, schemas, "create_ts_ddl_no_widgets"); err != nil {
		t.Fatalf("GenerateMigration: %v", err)
	}
	migrator.Up()

	cols := timestampColumns(t, db, physTable)
	if len(cols) != 0 {
		t.Fatalf("timestamp columns after Up = %v, want none", cols)
	}
	if timestampsIndexExists(t, db, physTable) {
		t.Fatal("expected no deleted_at index")
	}

	migrator.Down(0)
}

// TestApplyInvert_ToggleTimestamps_OnExistingTable exercises the full
// toggle cycle on an already-existing table: flipping the schema file's
// Timestamps from unset to true generates and applies a dedicated
// AddTimestamps action (columns + index appear, see ModelDiff.
// AddTimestamps's doc) - flipping it back to false is NOT a dedicated
// action at all, it's an ordinary Fields diff (CompareModel stops
// excluding the three columns the moment Timestamps resolves false, so
// CompareFields sees them as plain Deleted columns and BuildModelActions
// emits a DelField per column, same as any other field a schema file no
// longer declares). The whole sequence (create/add/implicit del) still
// inverts cleanly with a single Down(3) - including the interaction where
// undoing "add" (execDelTimestamps) runs AFTER undoing "del" already
// recreated the same three columns via plain AddField (no index) - see
// execDelTimestamps's own doc for why its own index drop tolerates that.
func TestApplyInvert_ToggleTimestamps_OnExistingTable(t *testing.T) {
	db := setupDB(t)
	physTable := pgTableName("ts_ddl_toggle")
	db.Exec("DROP TABLE IF EXISTS " + physTable)
	t.Cleanup(func() { db.Exec("DROP TABLE IF EXISTS " + physTable) })

	migrationsDir := t.TempDir()
	db.Exec("DROP TABLE IF EXISTS lx_sys.migrator")
	migrator.Init(migrator.Config{DB: db, MigrationsPath: migrationsDir})

	schemaDir := t.TempDir()
	schemaPath := filepath.Join(schemaDir, "ts_ddl_toggle.yaml")
	sch := &model.ModelSchema{
		Name: "ts_ddl_toggle",
		Fields: []model.NamedField{
			{Name: "name", Field: model.Field{Type: model.FieldTypeString, Size: 100, Required: true}},
		},
	}
	if err := sch.Save(schemaPath); err != nil {
		t.Fatalf("save schema: %v", err)
	}

	schemas := loadTestSchemas(t, schemaDir)
	if _, err := model.GenerateMigration(db, schemas, "create"); err != nil {
		t.Fatalf("GenerateMigration (create): %v", err)
	}
	migrator.Up()

	if cols := timestampColumns(t, db, physTable); len(cols) != 0 {
		t.Fatalf("timestamp columns after create = %v, want none", cols)
	}

	// Flip Timestamps on - AddTimestamps.
	tru := true
	sch.Timestamps = &tru
	if err := sch.Save(schemaPath); err != nil {
		t.Fatalf("save with Timestamps=true: %v", err)
	}
	schemas = loadTestSchemas(t, schemaDir)
	actions, err := model.GenerateMigration(db, schemas, "add_timestamps")
	if err != nil {
		t.Fatalf("GenerateMigration (add timestamps): %v", err)
	}
	if len(actions) != 1 || actions[0].Type != model.ActionAddTimestamps {
		t.Fatalf("actions = %+v, want a single addTimestamps action", actions)
	}
	migrator.Up()

	cols := timestampColumns(t, db, physTable)
	if !cols["created_at"] || !cols["updated_at"] || !cols["deleted_at"] {
		t.Fatalf("timestamp columns after add = %v, want all three", cols)
	}
	if !timestampsIndexExists(t, db, physTable) {
		t.Fatal("expected deleted_at index after add")
	}

	// Flip Timestamps back off - handled as 3 ordinary DelField actions,
	// not a dedicated one (see this test's own doc).
	fls := false
	sch.Timestamps = &fls
	if err := sch.Save(schemaPath); err != nil {
		t.Fatalf("save with Timestamps=false: %v", err)
	}
	schemas = loadTestSchemas(t, schemaDir)
	actions, err = model.GenerateMigration(db, schemas, "del_timestamps")
	if err != nil {
		t.Fatalf("GenerateMigration (del timestamps): %v", err)
	}
	if len(actions) != 3 {
		t.Fatalf("actions = %+v, want 3 delField actions", actions)
	}
	for _, a := range actions {
		if a.Type != model.ActionDelField {
			t.Fatalf("action = %+v, want delField", a)
		}
	}
	migrator.Up()

	if cols := timestampColumns(t, db, physTable); len(cols) != 0 {
		t.Fatalf("timestamp columns after del = %v, want none", cols)
	}
	if timestampsIndexExists(t, db, physTable) {
		t.Fatal("expected no deleted_at index after del")
	}

	// Unwind all 3 migrations (create/add/del) in one shot.
	migrator.Down(3)

	if _, err := model.IntrospectModelSchema(db, physTable, "public", false); err != model.ErrTableNotFound {
		t.Fatalf("err after full Down = %v, want ErrTableNotFound", err)
	}
}

// TestApplyInvert_ManualTimestampNamedField_TimestampsFalse is a
// regression test: a model with Timestamps off is free to declare its own
// ordinary Field under one of the three implicit names (CreatedAt here) -
// it's created once, like any other field, and a later GenerateMigration
// against the unchanged schema must find no diff at all (previously,
// IntrospectModelSchema excluded the physical column unconditionally,
// so CompareFields saw it as perpetually missing and kept regenerating an
// AddField for a column that already existed, failing at apply time).
func TestApplyInvert_ManualTimestampNamedField_TimestampsFalse(t *testing.T) {
	db := setupDB(t)
	physTable := pgTableName("ts_manual_created_at")
	db.Exec("DROP TABLE IF EXISTS " + physTable)
	t.Cleanup(func() { db.Exec("DROP TABLE IF EXISTS " + physTable) })

	migrationsDir := t.TempDir()
	db.Exec("DROP TABLE IF EXISTS lx_sys.migrator")
	migrator.Init(migrator.Config{DB: db, MigrationsPath: migrationsDir})

	schemaDir := t.TempDir()
	sch := &model.ModelSchema{
		Name: "ts_manual_created_at",
		Fields: []model.NamedField{
			{Name: "CreatedAt", Field: model.Field{Type: model.FieldTypeDateTime, Required: true}},
		},
	}
	if err := sch.Save(filepath.Join(schemaDir, "ts_manual_created_at.yaml")); err != nil {
		t.Fatalf("save schema: %v", err)
	}

	schemas := loadTestSchemas(t, schemaDir)
	if _, err := model.GenerateMigration(db, schemas, "create"); err != nil {
		t.Fatalf("GenerateMigration (create): %v", err)
	}
	migrator.Up()

	if cols := timestampColumns(t, db, physTable); !cols["created_at"] {
		t.Fatalf("timestamp columns after create = %v, want created_at present", cols)
	}

	// Re-running GenerateMigration against the SAME, unchanged schema must
	// find nothing to do - the manually declared field is visible to
	// IntrospectModelSchema (Timestamps is off) and matches exactly.
	schemas = loadTestSchemas(t, schemaDir)
	actions, err := model.GenerateMigration(db, schemas, "noop")
	if err != nil {
		t.Fatalf("GenerateMigration (second run): %v", err)
	}
	if actions != nil {
		t.Fatalf("actions = %+v, want none - the schema hasn't changed", actions)
	}

	migrator.Down(0)
}

// TestApplyInvert_AddTimestamps_AdoptsExistingCompatibleColumn is a
// regression test for the "turn Timestamps on after having declared one
// of the three columns by hand" scenario: a model starts with Timestamps
// off and an ordinary CreatedAt field (already holding data); flipping
// Timestamps on (and removing the now-forbidden explicit CreatedAt
// declaration, see TestModelManager_LoadModelSchemas_TimestampsForbidsCollidingField)
// must add only updated_at/deleted_at - created_at is adopted as-is, not
// recreated, so its existing row survives - and Down must undo exactly
// that (drop updated_at/deleted_at + the index, leave created_at and its
// data alone). Previously, AddTimestamps detection only checked whether
// created_at existed at all (as a proxy for "all three exist together",
// an assumption that held before Timestamps:false let a column exist
// independently) - so with created_at already present, no AddTimestamps
// action was generated at all, silently leaving updated_at/deleted_at
// missing forever.
func TestApplyInvert_AddTimestamps_AdoptsExistingCompatibleColumn(t *testing.T) {
	db := setupDB(t)
	physTable := pgTableName("ts_adopt")
	db.Exec("DROP TABLE IF EXISTS " + physTable)
	t.Cleanup(func() { db.Exec("DROP TABLE IF EXISTS " + physTable) })

	migrationsDir := t.TempDir()
	db.Exec("DROP TABLE IF EXISTS lx_sys.migrator")
	migrator.Init(migrator.Config{DB: db, MigrationsPath: migrationsDir})

	schemaDir := t.TempDir()
	schemaPath := filepath.Join(schemaDir, "ts_adopt.yaml")
	sch := &model.ModelSchema{
		Name: "ts_adopt",
		Fields: []model.NamedField{
			{Name: "CreatedAt", Field: model.Field{Type: model.FieldTypeDateTime, Required: true}},
		},
	}
	if err := sch.Save(schemaPath); err != nil {
		t.Fatalf("save schema: %v", err)
	}

	schemas := loadTestSchemas(t, schemaDir)
	if _, err := model.GenerateMigration(db, schemas, "create"); err != nil {
		t.Fatalf("GenerateMigration (create): %v", err)
	}
	migrator.Up()

	if _, err := db.Exec("INSERT INTO " + physTable + " (created_at) VALUES (now())"); err != nil {
		t.Fatalf("insert probe row: %v", err)
	}

	// Turn Timestamps on, removing the now-forbidden explicit field.
	tru := true
	sch.Timestamps = &tru
	sch.Fields = nil
	if err := sch.Save(schemaPath); err != nil {
		t.Fatalf("save with Timestamps=true: %v", err)
	}
	schemas = loadTestSchemas(t, schemaDir)
	actions, err := model.GenerateMigration(db, schemas, "enable_timestamps")
	if err != nil {
		t.Fatalf("GenerateMigration (enable): %v", err)
	}
	if len(actions) != 1 || actions[0].Type != model.ActionAddTimestamps {
		t.Fatalf("actions = %+v, want a single addTimestamps action", actions)
	}
	if got := actions[0].AddTimestamps.Columns; len(got) != 2 || got[0] != "updated_at" || got[1] != "deleted_at" {
		t.Fatalf("Columns = %v, want [updated_at deleted_at] (created_at already existed, adopted)", got)
	}
	migrator.Up()

	cols := timestampColumns(t, db, physTable)
	if !cols["created_at"] || !cols["updated_at"] || !cols["deleted_at"] {
		t.Fatalf("timestamp columns after enable = %v, want all three", cols)
	}
	if !timestampsIndexExists(t, db, physTable) {
		t.Fatal("expected deleted_at index after enable")
	}

	var rowCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM " + physTable).Scan(&rowCount); err != nil {
		t.Fatalf("counting rows: %v", err)
	}
	if rowCount != 1 {
		t.Fatalf("row count = %d, want 1 - created_at's own row must survive being adopted", rowCount)
	}

	// Undo "enable" - must drop only updated_at/deleted_at (+ the index),
	// leaving created_at (adopted, never this action's to remove) and its
	// data alone.
	migrator.Down(1)

	cols = timestampColumns(t, db, physTable)
	if !cols["created_at"] || cols["updated_at"] || cols["deleted_at"] {
		t.Fatalf("timestamp columns after undo = %v, want only created_at", cols)
	}
	if timestampsIndexExists(t, db, physTable) {
		t.Fatal("expected no deleted_at index after undo")
	}
	if err := db.QueryRow("SELECT COUNT(*) FROM " + physTable).Scan(&rowCount); err != nil {
		t.Fatalf("counting rows after undo: %v", err)
	}
	if rowCount != 1 {
		t.Fatalf("row count after undo = %d, want 1 - created_at's data must survive the rollback too", rowCount)
	}

	migrator.Down(1)
}

// TestApplyInvert_AddTimestamps_IncompatibleExistingColumnErrors checks
// that turning Timestamps on is rejected with a clear error, at
// GenerateMigration time, when an existing column under one of the three
// implicit names can't be adopted as-is - here CreatedAt was declared as
// FieldTypeString (a mismatched physical type), rather than silently
// accepted or silently altered.
func TestApplyInvert_AddTimestamps_IncompatibleExistingColumnErrors(t *testing.T) {
	db := setupDB(t)
	physTable := pgTableName("ts_incompatible")
	db.Exec("DROP TABLE IF EXISTS " + physTable)
	t.Cleanup(func() { db.Exec("DROP TABLE IF EXISTS " + physTable) })

	migrationsDir := t.TempDir()
	db.Exec("DROP TABLE IF EXISTS lx_sys.migrator")
	migrator.Init(migrator.Config{DB: db, MigrationsPath: migrationsDir})

	schemaDir := t.TempDir()
	schemaPath := filepath.Join(schemaDir, "ts_incompatible.yaml")
	sch := &model.ModelSchema{
		Name: "ts_incompatible",
		Fields: []model.NamedField{
			{Name: "CreatedAt", Field: model.Field{Type: model.FieldTypeString, Size: 50, Required: true}},
		},
	}
	if err := sch.Save(schemaPath); err != nil {
		t.Fatalf("save schema: %v", err)
	}

	schemas := loadTestSchemas(t, schemaDir)
	if _, err := model.GenerateMigration(db, schemas, "create"); err != nil {
		t.Fatalf("GenerateMigration (create): %v", err)
	}
	migrator.Up()

	tru := true
	sch.Timestamps = &tru
	sch.Fields = nil
	if err := sch.Save(schemaPath); err != nil {
		t.Fatalf("save with Timestamps=true: %v", err)
	}
	schemas = loadTestSchemas(t, schemaDir)
	if _, err := model.GenerateMigration(db, schemas, "enable_timestamps"); err == nil {
		t.Fatal("expected an error - existing created_at is character varying, not timestamp with time zone")
	}

	migrator.Down(0)
}

// TestApplyInvert_AddTimestamps_PreservesPreexistingIndexOnRollback checks
// that undoing an AddTimestamps action never drops deleted_at's own index
// if that index already existed before the migration ran (adopted the
// same way an already-compatible column is, see AddTimestampsAction's
// doc) - only an index this migration itself created is this action's to
// remove on rollback. created_at and deleted_at (plus deleted_at's index)
// are declared/created by hand while Timestamps is off; only updated_at is
// genuinely missing once Timestamps turns on.
func TestApplyInvert_AddTimestamps_PreservesPreexistingIndexOnRollback(t *testing.T) {
	db := setupDB(t)
	physTable := pgTableName("ts_index_adopt")
	db.Exec("DROP TABLE IF EXISTS " + physTable)
	t.Cleanup(func() { db.Exec("DROP TABLE IF EXISTS " + physTable) })

	migrationsDir := t.TempDir()
	db.Exec("DROP TABLE IF EXISTS lx_sys.migrator")
	migrator.Init(migrator.Config{DB: db, MigrationsPath: migrationsDir})

	schemaDir := t.TempDir()
	schemaPath := filepath.Join(schemaDir, "ts_index_adopt.yaml")
	sch := &model.ModelSchema{
		Name: "ts_index_adopt",
		Fields: []model.NamedField{
			{Name: "CreatedAt", Field: model.Field{Type: model.FieldTypeDateTime, Required: true}},
			{Name: "DeletedAt", Field: model.Field{Type: model.FieldTypeDateTime}},
		},
	}
	if err := sch.Save(schemaPath); err != nil {
		t.Fatalf("save schema: %v", err)
	}

	schemas := loadTestSchemas(t, schemaDir)
	if _, err := model.GenerateMigration(db, schemas, "create"); err != nil {
		t.Fatalf("GenerateMigration (create): %v", err)
	}
	migrator.Up()

	if _, err := db.Exec(
		`CREATE INDEX "idx__` + physTable + `__deleted_at" ON ` + physTable + ` (deleted_at)`,
	); err != nil {
		t.Fatalf("creating pre-existing deleted_at index by hand: %v", err)
	}
	if !timestampsIndexExists(t, db, physTable) {
		t.Fatal("pre-existing index setup failed")
	}

	// Turn Timestamps on, removing the now-forbidden explicit fields.
	tru := true
	sch.Timestamps = &tru
	sch.Fields = nil
	if err := sch.Save(schemaPath); err != nil {
		t.Fatalf("save with Timestamps=true: %v", err)
	}
	schemas = loadTestSchemas(t, schemaDir)
	actions, err := model.GenerateMigration(db, schemas, "enable_timestamps")
	if err != nil {
		t.Fatalf("GenerateMigration (enable): %v", err)
	}
	if len(actions) != 1 || actions[0].Type != model.ActionAddTimestamps {
		t.Fatalf("actions = %+v, want a single addTimestamps action", actions)
	}
	if got := actions[0].AddTimestamps.Columns; len(got) != 1 || got[0] != "updated_at" {
		t.Fatalf("Columns = %v, want [updated_at] (created_at/deleted_at already existed, adopted)", got)
	}
	if !actions[0].AddTimestamps.IndexExisted {
		t.Fatal("AddTimestamps.IndexExisted = false, want true - the index was already there")
	}
	migrator.Up()

	cols := timestampColumns(t, db, physTable)
	if !cols["created_at"] || !cols["updated_at"] || !cols["deleted_at"] {
		t.Fatalf("timestamp columns after enable = %v, want all three", cols)
	}
	if !timestampsIndexExists(t, db, physTable) {
		t.Fatal("expected deleted_at index to still be there after enable")
	}

	// Undo "enable" - must drop only updated_at, and must NOT drop the
	// index: it predates this migration, so it isn't this rollback's to
	// remove.
	migrator.Down(1)

	cols = timestampColumns(t, db, physTable)
	if !cols["created_at"] || cols["updated_at"] || !cols["deleted_at"] {
		t.Fatalf("timestamp columns after undo = %v, want created_at and deleted_at, not updated_at", cols)
	}
	if !timestampsIndexExists(t, db, physTable) {
		t.Fatal("index was dropped by rollback, but it predates this migration and should have survived")
	}

	migrator.Down(1)
}

// TestApplyInvert_AddTimestamps_IndexOnly checks that turning Timestamps on
// still creates deleted_at's own index even when all three columns already
// exist (nothing for AddTimestamps.Columns to add) - a table can have
// adopted every column by hand and still never have had deleted_at
// indexed, see ModelDiff.AddTimestampsIndexMissing's doc. Rollback then
// drops exactly that index, since this migration is the one that created
// it (IndexExisted is false here, the mirror of the "preserved" case in
// TestApplyInvert_AddTimestamps_PreservesPreexistingIndexOnRollback).
func TestApplyInvert_AddTimestamps_IndexOnly(t *testing.T) {
	db := setupDB(t)
	physTable := pgTableName("ts_index_only")
	db.Exec("DROP TABLE IF EXISTS " + physTable)
	t.Cleanup(func() { db.Exec("DROP TABLE IF EXISTS " + physTable) })

	migrationsDir := t.TempDir()
	db.Exec("DROP TABLE IF EXISTS lx_sys.migrator")
	migrator.Init(migrator.Config{DB: db, MigrationsPath: migrationsDir})

	schemaDir := t.TempDir()
	schemaPath := filepath.Join(schemaDir, "ts_index_only.yaml")
	sch := &model.ModelSchema{
		Name: "ts_index_only",
		Fields: []model.NamedField{
			{Name: "CreatedAt", Field: model.Field{Type: model.FieldTypeDateTime, Required: true}},
			{Name: "UpdatedAt", Field: model.Field{Type: model.FieldTypeDateTime, Required: true}},
			{Name: "DeletedAt", Field: model.Field{Type: model.FieldTypeDateTime}},
		},
	}
	if err := sch.Save(schemaPath); err != nil {
		t.Fatalf("save schema: %v", err)
	}

	schemas := loadTestSchemas(t, schemaDir)
	if _, err := model.GenerateMigration(db, schemas, "create"); err != nil {
		t.Fatalf("GenerateMigration (create): %v", err)
	}
	migrator.Up()

	if timestampsIndexExists(t, db, physTable) {
		t.Fatal("setup should not have created the index")
	}

	// Turn Timestamps on, removing the now-forbidden explicit fields -
	// every column is already there, only the index is missing.
	tru := true
	sch.Timestamps = &tru
	sch.Fields = nil
	if err := sch.Save(schemaPath); err != nil {
		t.Fatalf("save with Timestamps=true: %v", err)
	}
	schemas = loadTestSchemas(t, schemaDir)
	actions, err := model.GenerateMigration(db, schemas, "enable_timestamps")
	if err != nil {
		t.Fatalf("GenerateMigration (enable): %v", err)
	}
	if len(actions) != 1 || actions[0].Type != model.ActionAddTimestamps {
		t.Fatalf("actions = %+v, want a single addTimestamps action", actions)
	}
	if got := actions[0].AddTimestamps.Columns; len(got) != 0 {
		t.Fatalf("Columns = %v, want none - every column already existed", got)
	}
	if actions[0].AddTimestamps.IndexExisted {
		t.Fatal("AddTimestamps.IndexExisted = true, want false - the index was missing")
	}
	migrator.Up()

	if !timestampsIndexExists(t, db, physTable) {
		t.Fatal("expected deleted_at index to be created")
	}

	migrator.Down(1)

	if timestampsIndexExists(t, db, physTable) {
		t.Fatal("expected deleted_at index to be dropped by rollback - this migration is the one that created it")
	}

	migrator.Down(1)
}

// TestApplyInvert_AddTimestamps_ArbitrarilyNamedIndexIsAdopted checks that
// an existing index on deleted_at under a name other than this package's
// own naming convention (timestampsIndexName) still counts as "already
// indexed" (see timestampsIndexExists's own doc - it checks by column, the
// same way columnHasIndex already does for a relation's own FK column, not
// by name) - turning Timestamps on for a table that already has every
// column and an index on deleted_at (regardless of that index's name)
// needs no migration at all, so no redundant duplicate index ever gets
// created next to the existing one.
func TestApplyInvert_AddTimestamps_ArbitrarilyNamedIndexIsAdopted(t *testing.T) {
	db := setupDB(t)
	physTable := pgTableName("ts_arbitrary_index")
	db.Exec("DROP TABLE IF EXISTS " + physTable)
	t.Cleanup(func() { db.Exec("DROP TABLE IF EXISTS " + physTable) })

	migrationsDir := t.TempDir()
	db.Exec("DROP TABLE IF EXISTS lx_sys.migrator")
	migrator.Init(migrator.Config{DB: db, MigrationsPath: migrationsDir})

	schemaDir := t.TempDir()
	schemaPath := filepath.Join(schemaDir, "ts_arbitrary_index.yaml")
	sch := &model.ModelSchema{
		Name: "ts_arbitrary_index",
		Fields: []model.NamedField{
			{Name: "CreatedAt", Field: model.Field{Type: model.FieldTypeDateTime, Required: true}},
			{Name: "UpdatedAt", Field: model.Field{Type: model.FieldTypeDateTime, Required: true}},
			{Name: "DeletedAt", Field: model.Field{Type: model.FieldTypeDateTime}},
		},
	}
	if err := sch.Save(schemaPath); err != nil {
		t.Fatalf("save schema: %v", err)
	}

	schemas := loadTestSchemas(t, schemaDir)
	if _, err := model.GenerateMigration(db, schemas, "create"); err != nil {
		t.Fatalf("GenerateMigration (create): %v", err)
	}
	migrator.Up()

	const arbitraryIndexName = "my_own_soft_delete_idx"
	if _, err := db.Exec(
		"CREATE INDEX " + arbitraryIndexName + " ON " + physTable + " (deleted_at)",
	); err != nil {
		t.Fatalf("creating arbitrarily-named deleted_at index by hand: %v", err)
	}

	// Turn Timestamps on, removing the now-forbidden explicit fields -
	// every column and an index on deleted_at are already there.
	tru := true
	sch.Timestamps = &tru
	sch.Fields = nil
	if err := sch.Save(schemaPath); err != nil {
		t.Fatalf("save with Timestamps=true: %v", err)
	}
	schemas = loadTestSchemas(t, schemaDir)
	actions, err := model.GenerateMigration(db, schemas, "enable_timestamps")
	if err != nil {
		t.Fatalf("GenerateMigration (enable): %v", err)
	}
	if len(actions) != 0 {
		t.Fatalf("actions = %+v, want none - nothing is actually missing", actions)
	}

	if got := indexNamesOnDeletedAt(t, db, physTable); len(got) != 1 || got[0] != arbitraryIndexName {
		t.Fatalf("indexes on deleted_at = %v, want only [%s] - no duplicate should have been created", got, arbitraryIndexName)
	}
}
