//go:build integration

package model_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/epicoon/lxgo/migrator"
	"github.com/epicoon/lxgo/model"
	"github.com/shopspring/decimal"
)

// TestMain registers model.Apply/model.Invert with lxgo-migrator's
// migration type registry once for the whole integration run -
// migrator.RegisterMigrationType errors on a duplicate name, so this can't
// be done per-test (each Test function would collide with the others).
func TestMain(m *testing.M) {
	if err := migrator.RegisterMigrationType(model.MigrationType, model.Apply, model.Invert); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

func TestApplyInvert_CreateAndDropTable(t *testing.T) {
	db := setupDB(t)
	physTable := pgTableName("apply_widgets")
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
	sch := &model.ModelSchema{
		Name: "apply_widgets",
		Fields: []model.NamedField{
			{Name: "name", Field: model.Field{Type: model.FieldTypeString, Size: 100, Required: true, Default: "unnamed"}},
			{Name: "settings", Field: model.Field{Type: model.FieldTypeDict}},
		},
	}
	if err := sch.Save(filepath.Join(schemaDir, "apply_widgets.yaml")); err != nil {
		t.Fatalf("save schema: %v", err)
	}

	schemas := loadTestSchemas(t, schemaDir)
	if _, err := model.GenerateMigration(db, schemas, "create_apply_widgets"); err != nil {
		t.Fatalf("GenerateMigration: %v", err)
	}

	migrator.Up()

	introspected, err := model.IntrospectModelSchema(db, physTable, "public", false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema after Up: %v", err)
	}
	name, ok := introspected.FieldByName("name")
	if !ok || name.Type != model.FieldTypeString || name.Size != 100 || !name.Required || name.Default != "unnamed" {
		t.Fatalf("name after Up = %#v", name)
	}
	settings, ok := introspected.FieldByName("settings")
	if !ok || settings.Type != model.FieldTypeDict {
		t.Fatalf("settings after Up = %#v", settings)
	}

	migrator.Down(0)

	_, err = model.IntrospectModelSchema(db, physTable, "public", false)
	if err != model.ErrTableNotFound {
		t.Fatalf("err after Down = %v, want ErrTableNotFound", err)
	}
}

func TestApplyInvert_AddChangeRenameDelField(t *testing.T) {
	db := setupDB(t)
	physTable := pgTableName("apply_gadgets")
	if _, err := db.Exec("DROP TABLE IF EXISTS " + physTable); err != nil {
		t.Fatalf("drop %s: %v", physTable, err)
	}
	t.Cleanup(func() { db.Exec("DROP TABLE IF EXISTS " + physTable) })
	if _, err := db.Exec(`
		CREATE TABLE ` + physTable + ` (
			id serial PRIMARY KEY,
			name character varying(50) NOT NULL,
			old_count integer,
			leaving boolean
		)
	`); err != nil {
		t.Fatalf("create %s: %v", physTable, err)
	}

	migrationsDir := t.TempDir()
	if _, err := db.Exec("DROP TABLE IF EXISTS lx_sys.migrator"); err != nil {
		t.Fatalf("drop lx_sys.migrator: %v", err)
	}
	migrator.Init(migrator.Config{DB: db, MigrationsPath: migrationsDir})

	schemaDir := t.TempDir()
	schema := &model.ModelSchema{
		Name: "apply_gadgets",
		Fields: []model.NamedField{
			{Name: "name", Field: model.Field{Type: model.FieldTypeString, Size: 200, Required: true}},  // changed: 50 -> 200
			{Name: "new_count", Field: model.Field{Type: model.FieldTypeInt, RenamedFrom: "old_count"}}, // renamed
			{Name: "brand_new", Field: model.Field{Type: model.FieldTypeBool, Default: false}},          // added
			// "leaving" is absent from the code schema -> deleted
		},
	}
	if err := schema.Save(filepath.Join(schemaDir, "apply_gadgets.yaml")); err != nil {
		t.Fatalf("save schema: %v", err)
	}

	schemas := loadTestSchemas(t, schemaDir)
	actions, err := model.GenerateMigration(db, schemas, "change_apply_gadgets")
	if err != nil {
		t.Fatalf("GenerateMigration: %v", err)
	}
	if len(actions) == 0 {
		t.Fatal("expected a non-empty diff")
	}

	migrator.Up()

	after, err := model.IntrospectModelSchema(db, physTable, "public", false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema after Up: %v", err)
	}
	name, ok := after.FieldByName("name")
	if !ok || name.Size != 200 {
		t.Fatalf("name after Up = %#v, want Size 200", name)
	}
	if _, ok := after.FieldByName("old_count"); ok {
		t.Fatal("old_count should be gone after rename")
	}
	newCount, ok := after.FieldByName("new_count")
	if !ok || newCount.Type != model.FieldTypeInt {
		t.Fatalf("new_count after Up = %#v", newCount)
	}
	brandNew, ok := after.FieldByName("brand_new")
	if !ok || brandNew.Type != model.FieldTypeBool {
		t.Fatalf("brand_new after Up = %#v", brandNew)
	}
	if _, ok := after.FieldByName("leaving"); ok {
		t.Fatal("leaving should have been dropped")
	}

	migrator.Down(0)

	before, err := model.IntrospectModelSchema(db, physTable, "public", false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema after Down: %v", err)
	}
	name, ok = before.FieldByName("name")
	if !ok || name.Size != 50 {
		t.Fatalf("name after Down = %#v, want Size 50 (reverted)", name)
	}
	if _, ok := before.FieldByName("new_count"); ok {
		t.Fatal("new_count should be gone after Down (renamed back)")
	}
	if _, ok := before.FieldByName("old_count"); !ok {
		t.Fatal("old_count should be back after Down")
	}
	if _, ok := before.FieldByName("brand_new"); ok {
		t.Fatal("brand_new should have been removed by Down (was added by Up)")
	}
	if _, ok := before.FieldByName("leaving"); !ok {
		t.Fatal("leaving should be back after Down")
	}
}

func TestApplyInvert_MixedBatchWithQueryMigration(t *testing.T) {
	db := setupDB(t)
	physTyped := pgTableName("apply_typed")
	if _, err := db.Exec("DROP TABLE IF EXISTS apply_plain"); err != nil {
		t.Fatalf("drop apply_plain: %v", err)
	}
	if _, err := db.Exec("DROP TABLE IF EXISTS " + physTyped); err != nil {
		t.Fatalf("drop %s: %v", physTyped, err)
	}
	t.Cleanup(func() {
		db.Exec("DROP TABLE IF EXISTS apply_plain")
		db.Exec("DROP TABLE IF EXISTS " + physTyped)
	})

	migrationsDir := t.TempDir()
	if _, err := db.Exec("DROP TABLE IF EXISTS lx_sys.migrator"); err != nil {
		t.Fatalf("drop lx_sys.migrator: %v", err)
	}
	migrator.Init(migrator.Config{DB: db, MigrationsPath: migrationsDir})

	// The query migration must be applied before GenerateMigration runs -
	// CompareSchemas (via migrator.Check()) refuses to compare while any
	// migration, including a plain "query" one, is still unapplied.
	queryContent := "Name: plain\nType: query\n\nUp: CREATE TABLE apply_plain (id serial primary key)\n\nDown: DROP TABLE apply_plain\n"
	if err := os.WriteFile(filepath.Join(migrationsDir, "00000001_plain.yaml"), []byte(queryContent), 0644); err != nil {
		t.Fatalf("write query migration: %v", err)
	}
	migrator.Up()

	schemaDir := t.TempDir()
	schema := &model.ModelSchema{
		Name: "apply_typed",
		Fields: []model.NamedField{
			{Name: "price", Field: model.Field{Type: model.FieldTypeDecimal, Precision: 10, Scale: 2, Default: decimal.NewFromFloat(9.99)}},
		},
	}
	if err := schema.Save(filepath.Join(schemaDir, "apply_typed.yaml")); err != nil {
		t.Fatalf("save schema: %v", err)
	}
	schemas := loadTestSchemas(t, schemaDir)
	if _, err := model.GenerateMigration(db, schemas, "create_apply_typed"); err != nil {
		t.Fatalf("GenerateMigration: %v", err)
	}

	migrator.Up()

	var plainExists bool
	if err := db.QueryRow(`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'apply_plain')`).Scan(&plainExists); err != nil {
		t.Fatalf("checking apply_plain: %v", err)
	}
	if !plainExists {
		t.Fatal("apply_plain should exist after Up")
	}
	typedSchema, err := model.IntrospectModelSchema(db, physTyped, "public", false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema(apply_typed) after Up: %v", err)
	}
	price, ok := typedSchema.FieldByName("price")
	wantPrice := decimal.NewFromFloat(9.99)
	gotPrice, priceOK := price.Default.(decimal.Decimal)
	if !ok || !priceOK || !gotPrice.Equal(wantPrice) {
		t.Fatalf("price after Up = %#v", price)
	}

	migrator.Down(2)

	if err := db.QueryRow(`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'apply_plain')`).Scan(&plainExists); err != nil {
		t.Fatalf("checking apply_plain: %v", err)
	}
	if plainExists {
		t.Fatal("apply_plain should be gone after Down(2)")
	}
	if _, err := model.IntrospectModelSchema(db, physTyped, "public", false); err != model.ErrTableNotFound {
		t.Fatalf("apply_typed err after Down(2) = %v, want ErrTableNotFound", err)
	}
}

// TestApply_ChangeFieldIncompatibleTypeFails checks that changing a
// column to a type its actual data can't cast to fails with a clear
// Postgres error (via the USING cast in execChangeField), rather than
// panicking or silently truncating/nulling the data.
func TestApply_ChangeFieldIncompatibleTypeFails(t *testing.T) {
	db := setupDB(t)
	physTable := pgTableName("apply_incompatible")
	if _, err := db.Exec("DROP TABLE IF EXISTS " + physTable); err != nil {
		t.Fatalf("drop %s: %v", physTable, err)
	}
	t.Cleanup(func() { db.Exec("DROP TABLE IF EXISTS " + physTable) })
	if _, err := db.Exec(`CREATE TABLE ` + physTable + ` (id serial PRIMARY KEY, val text)`); err != nil {
		t.Fatalf("create %s: %v", physTable, err)
	}
	if _, err := db.Exec(`INSERT INTO ` + physTable + ` (val) VALUES ('not-a-number')`); err != nil {
		t.Fatalf("insert: %v", err)
	}

	content := []byte(`
Name: incompatible
Type: model
Actions:
  - Type: changeField
    ModelName: apply_incompatible
    ChangeField:
      FieldName: val
      OldDefinition:
        Type: string
      NewDefinition:
        Type: int
`)

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	defer tx.Rollback()

	if err := model.Apply(tx, content); err == nil {
		t.Fatal("expected Apply to fail casting 'not-a-number' to int")
	}
}

// TestApplyInvert_IntervalDefaultRoundTrip locks down that an interval
// field's default survives a real Apply (rendered as a Postgres interval
// literal) followed by IntrospectModelSchema reading the column back -
// not just the literal-formatting logic in isolation.
func TestApplyInvert_IntervalDefaultRoundTrip(t *testing.T) {
	db := setupDB(t)
	physTable := pgTableName("apply_interval")
	if _, err := db.Exec("DROP TABLE IF EXISTS " + physTable); err != nil {
		t.Fatalf("drop %s: %v", physTable, err)
	}
	t.Cleanup(func() { db.Exec("DROP TABLE IF EXISTS " + physTable) })

	migrationsDir := t.TempDir()
	if _, err := db.Exec("DROP TABLE IF EXISTS lx_sys.migrator"); err != nil {
		t.Fatalf("drop lx_sys.migrator: %v", err)
	}
	migrator.Init(migrator.Config{DB: db, MigrationsPath: migrationsDir})

	wantTimeout := 90*time.Minute + 30*time.Second
	schemaDir := t.TempDir()
	schema := &model.ModelSchema{
		Name: "apply_interval",
		Fields: []model.NamedField{
			{Name: "timeout", Field: model.Field{Type: model.FieldTypeInterval, Default: wantTimeout}},
		},
	}
	if err := schema.Save(filepath.Join(schemaDir, "apply_interval.yaml")); err != nil {
		t.Fatalf("save schema: %v", err)
	}
	schemas := loadTestSchemas(t, schemaDir)
	if _, err := model.GenerateMigration(db, schemas, "create_apply_interval"); err != nil {
		t.Fatalf("GenerateMigration: %v", err)
	}

	migrator.Up()

	introspected, err := model.IntrospectModelSchema(db, physTable, "public", false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema after Up: %v", err)
	}
	timeout, ok := introspected.FieldByName("timeout")
	if !ok || timeout.Type != model.FieldTypeInterval || timeout.Default != wantTimeout {
		t.Fatalf("timeout after Up = %#v, want default %v", timeout, wantTimeout)
	}
}
