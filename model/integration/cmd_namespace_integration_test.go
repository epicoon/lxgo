//go:build integration

package model_test

import (
	"path/filepath"
	"testing"

	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/apptest"
	"github.com/epicoon/lxgo/migrator"
	"github.com/epicoon/lxgo/model"
)

// TestCommand_StatusAndMigrate_NonPublicSchema runs the full model:db-status/
// model:db-migrate console path (not the library calls directly, the actual
// cmd.ICommand actions cmd.go registers) against a model resolved to a
// non-"public" schema, through a real app with a live DB connection - the
// db-status/db-migrate handlers read ModelSchema.Namespace end to end
// (CompareSchemas/GenerateMigration), not just the underlying functions
// exercised directly.
func TestCommand_StatusAndMigrate_NonPublicSchema(t *testing.T) {
	db := setupDB(t)
	const ns = "model_test_ns_cmd"
	physTable := pgTableName("CmdWidget")
	if _, err := db.Exec("DROP SCHEMA IF EXISTS " + ns + " CASCADE"); err != nil {
		t.Fatalf("drop schema %s: %v", ns, err)
	}
	t.Cleanup(func() { db.Exec("DROP SCHEMA IF EXISTS " + ns + " CASCADE") })

	migrationsDir := t.TempDir()
	db.Exec("DROP TABLE IF EXISTS lx_sys.migrator")
	migrator.Init(migrator.Config{DB: db, MigrationsPath: migrationsDir})

	schemaDir := t.TempDir()
	sch := &model.ModelSchema{
		Name: "CmdWidget", Namespace: ns,
		Fields: []model.NamedField{{Name: "name", Field: model.Field{Type: model.FieldTypeString, Size: 100, Required: true}}},
	}
	if err := sch.Save(filepath.Join(schemaDir, "CmdWidget.yaml")); err != nil {
		t.Fatalf("save schema: %v", err)
	}

	app, err := apptest.New(kernel.Dict{
		// apptest.New's own "Database" section wires up a connection but
		// doesn't Connect() it - the same test DB testDSN()/setupDB(t)
		// point at, honoring LXGO_MODEL_TEST_DSN too (see testDatabaseConfig).
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
	c.SetAction("db-status")
	if err := c.ActiveAction()(c); err != nil {
		t.Fatalf("db-status: %v", err)
	}

	c.SetAction("db-migrate")
	c.SetParams(map[string]any{"name": "create_cmd_widget", "apply": true})
	if err := c.ActiveAction()(c); err != nil {
		t.Fatalf("db-migrate: %v", err)
	}

	introspected, err := model.IntrospectModelSchema(db, physTable, ns, false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema(%s, %s) after migrate --apply: %v", ns, physTable, err)
	}
	name, ok := introspected.FieldByName("name")
	if !ok || name.Type != model.FieldTypeString || name.Size != 100 {
		t.Fatalf("name = %#v, %v", name, ok)
	}

	// db-status again - schema files and the database are back in sync, so
	// a second run should find nothing to report (no error either way, but
	// this exercises the CompareSchemas path once the table actually
	// exists in ns, not just before).
	c.SetAction("db-status")
	if err := c.ActiveAction()(c); err != nil {
		t.Fatalf("db-status (after db-migrate): %v", err)
	}
}
