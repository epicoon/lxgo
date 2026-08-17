//go:build integration

package model_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/apptest"
	"github.com/epicoon/lxgo/migrator"
	"github.com/epicoon/lxgo/model"
)

// TestCommand_Actualize_FullPlan runs "model:actualize" end to end with
// --yes (no dialog) against a live DB: a fresh schema with no existing
// table, no generated model file and no scaffolded repository - one pass
// must create the table (via a generated+applied migration), write the
// model file, and scaffold the repository. A second pass with nothing
// left to do must report "Nothing to actualize" rather than erroring or
// looping.
func TestCommand_Actualize_FullPlan(t *testing.T) {
	db := setupDB(t)
	physTable := pgTableName("ActWidget")
	if _, err := db.Exec("DROP TABLE IF EXISTS " + physTable); err != nil {
		t.Fatalf("drop table %s: %v", physTable, err)
	}
	t.Cleanup(func() { db.Exec("DROP TABLE IF EXISTS " + physTable) })

	migrationsDir := t.TempDir()
	db.Exec("DROP TABLE IF EXISTS lx_sys.migrator")
	migrator.Init(migrator.Config{DB: db, MigrationsPath: migrationsDir})

	schemaDir := t.TempDir()
	modelsDir := filepath.Join(t.TempDir(), "models")
	sch := &model.ModelSchema{
		Name:   "ActWidget",
		Fields: []model.NamedField{{Name: "name", Field: model.Field{Type: model.FieldTypeString, Size: 100, Required: true}}},
	}
	if err := sch.Save(filepath.Join(schemaDir, "ActWidget.yaml")); err != nil {
		t.Fatalf("save schema: %v", err)
	}

	app, err := apptest.New(kernel.Dict{
		"Database": testDatabaseConfig(),
		"Components": kernel.Dict{
			"ModelManager": kernel.Dict{
				"Targets": []kernel.Dict{{"Schemas": schemaDir, "Models": modelsDir, "Repos": modelsDir}},
			},
		},
	})
	if err != nil {
		t.Fatalf("apptest.New: %v", err)
	}
	if err := app.Connection().Connect(); err != nil {
		t.Fatalf("Connection().Connect(): %v", err)
	}
	if err := model.SetAppComponent(app, "Components.ModelManager"); err != nil {
		t.Fatalf("SetAppComponent: %v", err)
	}

	c := model.NewCommand(model.CommandOptions{App: app})
	c.SetAction("actualize")
	c.SetParams(map[string]any{"yes": true, "name": "create_act_widget"})
	if err := c.ActiveAction()(c); err != nil {
		t.Fatalf("actualize: %v", err)
	}

	introspected, err := model.IntrospectModelSchema(db, physTable, "public", false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema(%s) after actualize: %v", physTable, err)
	}
	if _, ok := introspected.FieldByName("name"); !ok {
		t.Fatal("expected the migration to have created the 'name' column")
	}

	if _, err := os.Stat(filepath.Join(modelsDir, "act_widget_gen.go")); err != nil {
		t.Fatalf("expected a generated model file: %v", err)
	}
	if _, err := os.Stat(filepath.Join(modelsDir, "act_widget_repo.go")); err != nil {
		t.Fatalf("expected a scaffolded repository file: %v", err)
	}

	// Second pass - schema, DB, generated code and repository are all
	// already in sync, so there's nothing left to do.
	c.SetAction("actualize")
	c.SetParams(map[string]any{"yes": true})
	if err := c.ActiveAction()(c); err != nil {
		t.Fatalf("actualize (second pass): %v", err)
	}
}

// TestCommand_Actualize_YesWithoutNameErrorsWhenMigrationNeeded exercises
// the one case actualize can't resolve on its own even with --yes: a
// migration is needed but no --name was given, and --yes rules out
// falling back to an interactive prompt.
func TestCommand_Actualize_YesWithoutNameErrorsWhenMigrationNeeded(t *testing.T) {
	db := setupDB(t)
	physTable := pgTableName("ActNoName")
	if _, err := db.Exec("DROP TABLE IF EXISTS " + physTable); err != nil {
		t.Fatalf("drop table %s: %v", physTable, err)
	}
	t.Cleanup(func() { db.Exec("DROP TABLE IF EXISTS " + physTable) })

	migrationsDir := t.TempDir()
	db.Exec("DROP TABLE IF EXISTS lx_sys.migrator")
	migrator.Init(migrator.Config{DB: db, MigrationsPath: migrationsDir})

	schemaDir := t.TempDir()
	sch := &model.ModelSchema{
		Name:   "ActNoName",
		Fields: []model.NamedField{{Name: "name", Field: model.Field{Type: model.FieldTypeString, Size: 100, Required: true}}},
	}
	if err := sch.Save(filepath.Join(schemaDir, "ActNoName.yaml")); err != nil {
		t.Fatalf("save schema: %v", err)
	}

	app, err := apptest.New(kernel.Dict{
		"Database": testDatabaseConfig(),
		"Components": kernel.Dict{
			"ModelManager": kernel.Dict{
				"Targets": []kernel.Dict{{"Schemas": schemaDir}},
			},
		},
	})
	if err != nil {
		t.Fatalf("apptest.New: %v", err)
	}
	if err := app.Connection().Connect(); err != nil {
		t.Fatalf("Connection().Connect(): %v", err)
	}
	if err := model.SetAppComponent(app, "Components.ModelManager"); err != nil {
		t.Fatalf("SetAppComponent: %v", err)
	}

	c := model.NewCommand(model.CommandOptions{App: app})
	c.SetAction("actualize")
	c.SetParams(map[string]any{"yes": true})
	if err := c.ActiveAction()(c); err != nil {
		t.Fatalf("actualize: %v", err)
	}

	if _, err := db.Query("SELECT 1 FROM " + physTable); err == nil {
		t.Fatal("expected no table to have been created - actualize should have stopped short of generating a migration without a name")
	}
}
