//go:build integration

package model_test

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/epicoon/lxgo/migrator"
	"github.com/epicoon/lxgo/model"
)

// setupMigratorForCompare gives migrator a clean, empty migrations
// directory - CompareSchemas refuses to run if migrator.Check() reports
// any unapplied migration, so tests that don't care about that precondition
// need it to report none.
func setupMigratorForCompare(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec("DROP TABLE IF EXISTS lx_sys.migrator"); err != nil {
		t.Fatalf("drop lx_sys.migrator: %v", err)
	}
	migrator.Init(migrator.Config{DB: db, MigrationsPath: t.TempDir()})
}

func TestCompareModel_NeedsTable(t *testing.T) {
	db := setupDB(t)
	if _, err := db.Exec("DROP TABLE IF EXISTS " + pgTableName("compare_ghost")); err != nil {
		t.Fatalf("drop compare_ghost: %v", err)
	}

	codeSchema := &model.ModelSchema{Name: "compare_ghost"}
	diff, err := model.CompareModel(db, codeSchema)
	if err != nil {
		t.Fatalf("CompareModel: %v", err)
	}
	if !diff.NeedsTable {
		t.Fatalf("got %#v, want NeedsTable = true", diff)
	}
	if !diff.Fields.IsEmpty() {
		t.Fatalf("Fields should be zero-value when NeedsTable, got %#v", diff.Fields)
	}
}

func TestCompareModel_NoDiff(t *testing.T) {
	db := setupDB(t)
	physTable := pgTableName("compare_widgets")
	if _, err := db.Exec("DROP TABLE IF EXISTS " + physTable); err != nil {
		t.Fatalf("drop %s: %v", physTable, err)
	}
	t.Cleanup(func() { db.Exec("DROP TABLE IF EXISTS " + physTable) })

	_, err := db.Exec(`
		CREATE TABLE ` + physTable + ` (
			id serial PRIMARY KEY,
			name character varying(50) NOT NULL,
			sort integer NOT NULL DEFAULT 0
		)
	`)
	if err != nil {
		t.Fatalf("create %s: %v", physTable, err)
	}

	codeSchema := &model.ModelSchema{
		Name: "compare_widgets",
		Fields: []model.NamedField{
			{Name: "name", Field: model.Field{Type: model.FieldTypeString, Required: true, Size: 50}},
			{Name: "sort", Field: model.Field{Type: model.FieldTypeInt, Required: true, Default: int64(0)}},
		},
	}

	diff, err := model.CompareModel(db, codeSchema)
	if err != nil {
		t.Fatalf("CompareModel: %v", err)
	}
	if !diff.IsEmpty() {
		t.Fatalf("expected an empty diff, got %#v", diff)
	}
}

func TestCompareModel_FieldChanged(t *testing.T) {
	db := setupDB(t)
	physTable := pgTableName("compare_widgets")
	if _, err := db.Exec("DROP TABLE IF EXISTS " + physTable); err != nil {
		t.Fatalf("drop %s: %v", physTable, err)
	}
	t.Cleanup(func() { db.Exec("DROP TABLE IF EXISTS " + physTable) })

	_, err := db.Exec(`
		CREATE TABLE ` + physTable + ` (
			id serial PRIMARY KEY,
			name character varying(50) NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("create %s: %v", physTable, err)
	}

	codeSchema := &model.ModelSchema{
		Name: "compare_widgets",
		Fields: []model.NamedField{
			{Name: "name", Field: model.Field{Type: model.FieldTypeString, Required: true, Size: 100}},
		},
	}

	diff, err := model.CompareModel(db, codeSchema)
	if err != nil {
		t.Fatalf("CompareModel: %v", err)
	}
	if len(diff.Fields.Changed) != 1 || diff.Fields.Changed[0] != "name" {
		t.Fatalf("Fields.Changed = %v, want [name]", diff.Fields.Changed)
	}
}

// TestCompareModel_GormCompatiblePhysicalNaming locks down the actual point
// of the physical-naming translation: a model/field name lxgo-model creates
// must land under the same table/column name GORM's own zero-config
// schema.NamingStrategy would derive from a Go struct of the same name -
// checked against real information_schema data, not just the translation
// function in isolation (see naming_test.go for that).
func TestCompareModel_GormCompatiblePhysicalNaming(t *testing.T) {
	db := setupDB(t)
	setupMigratorForCompare(t, db)

	physTable := pgTableName("WidgetCopy")
	if _, err := db.Exec("DROP TABLE IF EXISTS " + physTable); err != nil {
		t.Fatalf("drop %s: %v", physTable, err)
	}
	t.Cleanup(func() { db.Exec("DROP TABLE IF EXISTS " + physTable) })

	schemaDir := t.TempDir()
	codeSchema := &model.ModelSchema{
		Name: "WidgetCopy",
		Fields: []model.NamedField{
			{Name: "NameCopy", Field: model.Field{Type: model.FieldTypeString, Size: 255, Required: true}},
		},
	}
	if err := codeSchema.Save(filepath.Join(schemaDir, "WidgetCopy.yaml")); err != nil {
		t.Fatalf("save schema: %v", err)
	}

	schemas := loadTestSchemas(t, schemaDir)
	if _, err := model.GenerateMigration(db, schemas, "create_widget_copy"); err != nil {
		t.Fatalf("GenerateMigration: %v", err)
	}
	migrator.Up()

	var tableExists bool
	if err := db.QueryRow(`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = $1)`, physTable).Scan(&tableExists); err != nil {
		t.Fatalf("checking %s: %v", physTable, err)
	}
	if !tableExists {
		t.Fatalf("expected a physical table %q (gorm's NamingStrategy for %q)", physTable, "WidgetCopy")
	}
	physColumn := pgColumnName("NameCopy")
	var columnExists bool
	if err := db.QueryRow(
		`SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = $1 AND column_name = $2)`,
		physTable, physColumn,
	).Scan(&columnExists); err != nil {
		t.Fatalf("checking %s.%s: %v", physTable, physColumn, err)
	}
	if !columnExists {
		t.Fatalf("expected a physical column %q (gorm's NamingStrategy for %q) on %q", physColumn, "NameCopy", physTable)
	}

	diff, err := model.CompareModel(db, codeSchema)
	if err != nil {
		t.Fatalf("CompareModel: %v", err)
	}
	if !diff.IsEmpty() {
		t.Fatalf("expected an empty diff once the physically-named table/column exist, got %#v", diff)
	}
}

func TestCompareSchemas_UnappliedMigrationsBlocks(t *testing.T) {
	db := setupDB(t)
	if _, err := db.Exec("DROP TABLE IF EXISTS lx_sys.migrator"); err != nil {
		t.Fatalf("drop lx_sys.migrator: %v", err)
	}

	migrationsDir := t.TempDir()
	content := "Name: pending\nType: query\n\nUp: SELECT 1\n\nDown: SELECT 1\n"
	if err := os.WriteFile(filepath.Join(migrationsDir, "00000001_pending.yaml"), []byte(content), 0644); err != nil {
		t.Fatalf("write migration: %v", err)
	}
	migrator.Init(migrator.Config{DB: db, MigrationsPath: migrationsDir})

	_, err := model.CompareSchemas(db, nil)
	if !errors.Is(err, model.ErrUnappliedMigrations) {
		t.Fatalf("err = %v, want ErrUnappliedMigrations", err)
	}
}

func TestCompareSchemas_Directory(t *testing.T) {
	db := setupDB(t)
	setupMigratorForCompare(t, db)

	physTable := pgTableName("compare_widgets")
	if _, err := db.Exec("DROP TABLE IF EXISTS " + physTable); err != nil {
		t.Fatalf("drop %s: %v", physTable, err)
	}
	t.Cleanup(func() { db.Exec("DROP TABLE IF EXISTS " + physTable) })
	if _, err := db.Exec(`CREATE TABLE ` + physTable + ` (id serial PRIMARY KEY, name text)`); err != nil {
		t.Fatalf("create %s: %v", physTable, err)
	}

	schemaDir := t.TempDir()
	matching := &model.ModelSchema{
		Name: "compare_widgets",
		Fields: []model.NamedField{
			{Name: "name", Field: model.Field{Type: model.FieldTypeString}},
		},
	}
	if err := matching.Save(filepath.Join(schemaDir, "compare_widgets.yaml")); err != nil {
		t.Fatalf("save compare_widgets schema: %v", err)
	}
	needsTable := &model.ModelSchema{Name: "compare_ghost"}
	if err := needsTable.Save(filepath.Join(schemaDir, "compare_ghost.yaml")); err != nil {
		t.Fatalf("save compare_ghost schema: %v", err)
	}

	schemas := loadTestSchemas(t, schemaDir)
	diffs, err := model.CompareSchemas(db, schemas)
	if err != nil {
		t.Fatalf("CompareSchemas: %v", err)
	}
	if len(diffs) != 1 {
		t.Fatalf("diffs = %#v, want exactly one entry (compare_widgets has no diff, compare_ghost needs a table)", diffs)
	}
	if diffs[0].Name != "compare_ghost" || !diffs[0].NeedsTable {
		t.Fatalf("diffs[0] = %#v, want compare_ghost needing a table", diffs[0])
	}
}
