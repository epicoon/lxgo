//go:build integration

package model_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/epicoon/lxgo/migrator"
	"github.com/epicoon/lxgo/model"
)

func TestGenerateMigration_CreateTable(t *testing.T) {
	db := setupDB(t)
	physTable := pgTableName("gen_widgets")
	if _, err := db.Exec("DROP TABLE IF EXISTS " + physTable); err != nil {
		t.Fatalf("drop %s: %v", physTable, err)
	}
	t.Cleanup(func() { db.Exec("DROP TABLE IF EXISTS " + physTable) })

	migrationsDir := t.TempDir()
	if _, err := db.Exec("DROP TABLE IF EXISTS lx_sys.migrator"); err != nil {
		t.Fatalf("drop lx_sys.migrator: %v", err)
	}
	migrator.Init(migrator.Config{DB: db, MigrationsPath: migrationsDir})

	schemaDir := t.TempDir()
	schema := &model.ModelSchema{
		Name: "gen_widgets",
		Fields: []model.NamedField{
			{Name: "name", Field: model.Field{Type: model.FieldTypeString}},
		},
	}
	if err := schema.Save(filepath.Join(schemaDir, "gen_widgets.yaml")); err != nil {
		t.Fatalf("save schema: %v", err)
	}

	schemas := loadTestSchemas(t, schemaDir)
	actions, err := model.GenerateMigration(db, schemas, "create_widgets")
	if err != nil {
		t.Fatalf("GenerateMigration: %v", err)
	}
	if len(actions) != 1 || actions[0].Type != model.ActionCreateTable {
		t.Fatalf("actions = %#v, want exactly one createTable", actions)
	}

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected exactly one migration file, got %d", len(entries))
	}
	content, err := os.ReadFile(filepath.Join(migrationsDir, entries[0].Name()))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(content), "Type: model") || !strings.Contains(string(content), "CreateTable:") {
		t.Fatalf("content = %s, want Type: model and a CreateTable block", content)
	}
}

func TestGenerateMigration_NothingToMigrate(t *testing.T) {
	db := setupDB(t)
	physTable := pgTableName("gen_nochange")
	if _, err := db.Exec("DROP TABLE IF EXISTS " + physTable); err != nil {
		t.Fatalf("drop %s: %v", physTable, err)
	}
	t.Cleanup(func() { db.Exec("DROP TABLE IF EXISTS " + physTable) })
	if _, err := db.Exec("CREATE TABLE " + physTable + " (id serial PRIMARY KEY, name text)"); err != nil {
		t.Fatalf("create %s: %v", physTable, err)
	}

	migrationsDir := t.TempDir()
	if _, err := db.Exec("DROP TABLE IF EXISTS lx_sys.migrator"); err != nil {
		t.Fatalf("drop lx_sys.migrator: %v", err)
	}
	migrator.Init(migrator.Config{DB: db, MigrationsPath: migrationsDir})

	schemaDir := t.TempDir()
	schema := &model.ModelSchema{
		Name: "gen_nochange",
		Fields: []model.NamedField{
			{Name: "name", Field: model.Field{Type: model.FieldTypeString}},
		},
	}
	if err := schema.Save(filepath.Join(schemaDir, "gen_nochange.yaml")); err != nil {
		t.Fatalf("save schema: %v", err)
	}

	schemas := loadTestSchemas(t, schemaDir)
	actions, err := model.GenerateMigration(db, schemas, "noop")
	if err != nil {
		t.Fatalf("GenerateMigration: %v", err)
	}
	if actions != nil {
		t.Fatalf("actions = %#v, want nil", actions)
	}

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected no migration file written, got %d", len(entries))
	}
}

func TestGenerateMigration_ClearsExplicitRenameFromSchemaFile(t *testing.T) {
	db := setupDB(t)
	physTable := pgTableName("gen_renamed")
	if _, err := db.Exec("DROP TABLE IF EXISTS " + physTable); err != nil {
		t.Fatalf("drop %s: %v", physTable, err)
	}
	t.Cleanup(func() { db.Exec("DROP TABLE IF EXISTS " + physTable) })
	if _, err := db.Exec("CREATE TABLE " + physTable + " (id serial PRIMARY KEY, name text)"); err != nil {
		t.Fatalf("create %s: %v", physTable, err)
	}

	migrationsDir := t.TempDir()
	if _, err := db.Exec("DROP TABLE IF EXISTS lx_sys.migrator"); err != nil {
		t.Fatalf("drop lx_sys.migrator: %v", err)
	}
	migrator.Init(migrator.Config{DB: db, MigrationsPath: migrationsDir})

	schemaDir := t.TempDir()
	schemaPath := filepath.Join(schemaDir, "gen_renamed.yaml")
	schema := &model.ModelSchema{
		Name: "gen_renamed",
		Fields: []model.NamedField{
			{Name: "fullName", Field: model.Field{Type: model.FieldTypeString, RenamedFrom: "name"}},
		},
	}
	if err := schema.Save(schemaPath); err != nil {
		t.Fatalf("save schema: %v", err)
	}

	schemas := loadTestSchemas(t, schemaDir)
	actions, err := model.GenerateMigration(db, schemas, "rename_name")
	if err != nil {
		t.Fatalf("GenerateMigration: %v", err)
	}
	if len(actions) != 1 || actions[0].Type != model.ActionRenameField {
		t.Fatalf("actions = %#v, want exactly one renameField", actions)
	}
	if actions[0].RenameField.OldFieldName != "name" || actions[0].RenameField.NewFieldName != "fullName" {
		t.Fatalf("renameField = %#v", actions[0].RenameField)
	}

	data, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	reloaded, err := model.LoadModelSchema("gen_renamed", data)
	if err != nil {
		t.Fatalf("LoadModelSchema: %v", err)
	}
	f, ok := reloaded.FieldByName("fullName")
	if !ok || f.RenamedFrom != "" {
		t.Fatalf("fullName = %#v, want RenamedFrom cleared", f)
	}
}

// TestGenerateMigration_MultiDirClearsExplicitRenameInCorrectDir is the
// main risk of comparing/generating across more than one schema
// directory: an explicit rename must be saved back to the directory the
// model actually came from (ModelSchema.SourceDir), not to whichever
// directory happens to be passed first, and an unrelated model's file in
// a different directory must stay untouched. The renamed model's
// directory is deliberately passed second to GenerateMigration - saving
// to schemaDirs[0] instead of the schema's own SourceDir would still
// pass this test if the rename happened to be in the first directory.
func TestGenerateMigration_MultiDirClearsExplicitRenameInCorrectDir(t *testing.T) {
	db := setupDB(t)
	physA, physB := pgTableName("gen_multi_a"), pgTableName("gen_multi_b")
	if _, err := db.Exec("DROP TABLE IF EXISTS " + physA); err != nil {
		t.Fatalf("drop %s: %v", physA, err)
	}
	if _, err := db.Exec("DROP TABLE IF EXISTS " + physB); err != nil {
		t.Fatalf("drop %s: %v", physB, err)
	}
	t.Cleanup(func() {
		db.Exec("DROP TABLE IF EXISTS " + physA)
		db.Exec("DROP TABLE IF EXISTS " + physB)
	})
	if _, err := db.Exec("CREATE TABLE " + physA + " (id serial PRIMARY KEY, name text)"); err != nil {
		t.Fatalf("create %s: %v", physA, err)
	}
	if _, err := db.Exec("CREATE TABLE " + physB + " (id serial PRIMARY KEY, x integer)"); err != nil {
		t.Fatalf("create %s: %v", physB, err)
	}

	migrationsDir := t.TempDir()
	if _, err := db.Exec("DROP TABLE IF EXISTS lx_sys.migrator"); err != nil {
		t.Fatalf("drop lx_sys.migrator: %v", err)
	}
	migrator.Init(migrator.Config{DB: db, MigrationsPath: migrationsDir})

	dirA, dirB := t.TempDir(), t.TempDir()
	pathA := filepath.Join(dirA, "gen_multi_a.yaml")
	schemaA := &model.ModelSchema{
		Name: "gen_multi_a",
		Fields: []model.NamedField{
			{Name: "fullName", Field: model.Field{Type: model.FieldTypeString, RenamedFrom: "name"}},
		},
	}
	if err := schemaA.Save(pathA); err != nil {
		t.Fatalf("save schema A: %v", err)
	}

	pathB := filepath.Join(dirB, "gen_multi_b.yaml")
	schemaB := &model.ModelSchema{
		Name: "gen_multi_b",
		Fields: []model.NamedField{
			{Name: "x", Field: model.Field{Type: model.FieldTypeInt}},
		},
	}
	if err := schemaB.Save(pathB); err != nil {
		t.Fatalf("save schema B: %v", err)
	}

	schemas := loadTestSchemas(t, dirB, dirA)
	actions, err := model.GenerateMigration(db, schemas, "rename_multi")
	if err != nil {
		t.Fatalf("GenerateMigration: %v", err)
	}
	if len(actions) != 1 || actions[0].Type != model.ActionRenameField {
		t.Fatalf("actions = %#v, want exactly one renameField (schema B has no diff at all)", actions)
	}

	dataA, err := os.ReadFile(pathA)
	if err != nil {
		t.Fatalf("ReadFile A: %v", err)
	}
	reloadedA, err := model.LoadModelSchema("gen_multi_a", dataA)
	if err != nil {
		t.Fatalf("LoadModelSchema A: %v", err)
	}
	fA, ok := reloadedA.FieldByName("fullName")
	if !ok || fA.RenamedFrom != "" {
		t.Fatalf("fullName in dir A = %#v, want RenamedFrom cleared", fA)
	}

	dataB, err := os.ReadFile(pathB)
	if err != nil {
		t.Fatalf("ReadFile B: %v", err)
	}
	reloadedB, err := model.LoadModelSchema("gen_multi_b", dataB)
	if err != nil {
		t.Fatalf("LoadModelSchema B: %v", err)
	}
	if _, ok := reloadedB.FieldByName("x"); !ok {
		t.Fatalf("schema B should be unaffected, got %#v", reloadedB)
	}
}
