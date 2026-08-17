//go:build integration

package model_test

import (
	"database/sql"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/shopspring/decimal"
	"gorm.io/gorm/schema"

	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/apptest"
	"github.com/epicoon/lxgo/model"
)

// pgNaming mirrors the package's own unexported pgTableName/pgColumnName
// (naming.go) - can't call those directly from this external model_test
// package, so tests that talk to the database directly (raw SQL, a direct
// IntrospectModelSchema call) rather than through GenerateMigration/Apply/
// CompareModel need this to know what physical name those functions
// actually used.
var pgNaming = schema.NamingStrategy{}

func pgTableName(name string) string  { return pgNaming.TableName(name) }
func pgColumnName(name string) string { return pgNaming.ColumnName("", name) }

func testDSN() string {
	dsn := os.Getenv("LXGO_MODEL_TEST_DSN")
	if dsn == "" {
		dsn = "host=localhost user=lx password=123456 dbname=lxgomodeltest port=55434 sslmode=disable"
	}
	return dsn
}

func setupDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("postgres", testDSN())
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Fatalf("db.Ping: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// testDatabaseConfig splits testDSN()'s "key=value ..." pairs into the
// kernel.Dict shape apptest's own "Database" config section expects - for
// a test that needs a live kernel.IApp Connection (app.Connection().DB(),
// not just a bare *sql.DB from setupDB) pointed at the exact same test
// database, honoring LXGO_MODEL_TEST_DSN the same way testDSN itself does.
func testDatabaseConfig() kernel.Dict {
	d := kernel.Dict{}
	for _, pair := range strings.Fields(testDSN()) {
		key, value, ok := strings.Cut(pair, "=")
		if !ok {
			continue
		}
		switch key {
		case "host":
			d["Host"] = value
		case "port":
			port, _ := strconv.Atoi(value)
			d["Port"] = port
		case "user":
			d["User"] = value
		case "password":
			d["Password"] = value
		case "dbname":
			d["DBName"] = value
		case "sslmode":
			d["SSLMode"] = value
		}
	}
	return d
}

// loadTestSchemas loads model schemas through a throwaway ModelManager app
// component - one Target per dir, in order - the same cascade-resolving
// path production code uses (model.ModelManager.LoadModelSchemas), instead
// of reaching into the package's internals from this external test package.
func loadTestSchemas(t *testing.T, dirs ...string) []*model.ModelSchema {
	t.Helper()
	targets := make([]kernel.Dict, len(dirs))
	for i, dir := range dirs {
		targets[i] = kernel.Dict{"Schemas": dir}
	}
	app, err := apptest.New(kernel.Dict{
		"Components": kernel.Dict{
			"ModelManager": kernel.Dict{
				"Targets": targets,
			},
		},
	})
	if err != nil {
		t.Fatalf("apptest.New: %v", err)
	}
	if err := model.SetAppComponent(app, "Components.ModelManager"); err != nil {
		t.Fatalf("SetAppComponent: %v", err)
	}
	mm, err := model.AppComponent(app)
	if err != nil {
		t.Fatalf("AppComponent: %v", err)
	}
	schemas, err := mm.LoadModelSchemas()
	if err != nil {
		t.Fatalf("LoadModelSchemas: %v", err)
	}
	return schemas
}

func TestIntrospectModelSchema_AllFieldTypes(t *testing.T) {
	db := setupDB(t)

	if _, err := db.Exec("DROP TABLE IF EXISTS widgets"); err != nil {
		t.Fatalf("drop widgets: %v", err)
	}
	t.Cleanup(func() { db.Exec("DROP TABLE IF EXISTS widgets") })

	_, err := db.Exec(`
		CREATE TABLE widgets (
			id serial PRIMARY KEY,
			name character varying(255) NOT NULL DEFAULT 'unnamed',
			notes text,
			code character(5),
			sort integer NOT NULL DEFAULT 0,
			small_count smallint,
			big_count bigint,
			ratio double precision,
			factor real,
			price numeric(10, 2) DEFAULT 19.99,
			is_active boolean NOT NULL DEFAULT false,
			birth_date date DEFAULT '2000-01-01',
			start_time time without time zone DEFAULT '09:00:00',
			published_at timestamp with time zone DEFAULT '2026-01-02 15:04:05+00',
			session_timeout interval DEFAULT '1 day 01:30:00',
			settings jsonb DEFAULT '{"theme":"dark"}'::jsonb,
			extra json
		)
	`)
	if err != nil {
		t.Fatalf("create widgets: %v", err)
	}

	schema, err := model.IntrospectModelSchema(db, "widgets", "public", false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema: %v", err)
	}
	if schema.Name != "widgets" {
		t.Fatalf("Name = %q, want %q", schema.Name, "widgets")
	}

	fields := make(map[string]model.Field, len(schema.Fields))
	var order []string
	for _, f := range schema.Fields {
		fields[f.Name] = f.Field
		order = append(order, f.Name)
	}

	if _, ok := fields["id"]; ok {
		t.Fatal("expected \"id\" to be excluded from Fields")
	}

	wantOrder := []string{
		"name", "notes", "code", "sort", "small_count", "big_count", "ratio", "factor",
		"price", "is_active", "birth_date", "start_time", "published_at",
		"session_timeout", "settings", "extra",
	}
	if len(order) != len(wantOrder) {
		t.Fatalf("field order = %v, want %v", order, wantOrder)
	}
	for i, name := range wantOrder {
		if order[i] != name {
			t.Fatalf("field order = %v, want %v", order, wantOrder)
		}
	}

	name := fields["name"]
	if name.Type != model.FieldTypeString || !name.Required || name.Size != 255 || name.Default != "unnamed" {
		t.Fatalf("name = %#v", name)
	}

	notes := fields["notes"]
	if notes.Type != model.FieldTypeString || notes.Required || notes.Default != nil {
		t.Fatalf("notes = %#v", notes)
	}

	code := fields["code"]
	if code.Type != model.FieldTypeString || code.Required || code.Size != 5 {
		t.Fatalf("code = %#v", code)
	}

	sort := fields["sort"]
	if sort.Type != model.FieldTypeInt || !sort.Required || sort.Default != int64(0) {
		t.Fatalf("sort = %#v", sort)
	}

	smallCount := fields["small_count"]
	if smallCount.Type != model.FieldTypeInt || smallCount.Required {
		t.Fatalf("small_count = %#v", smallCount)
	}

	bigCount := fields["big_count"]
	if bigCount.Type != model.FieldTypeInt || bigCount.Required {
		t.Fatalf("big_count = %#v", bigCount)
	}

	ratio := fields["ratio"]
	if ratio.Type != model.FieldTypeFloat || ratio.Required || ratio.Default != nil {
		t.Fatalf("ratio = %#v", ratio)
	}

	factor := fields["factor"]
	if factor.Type != model.FieldTypeFloat || factor.Required {
		t.Fatalf("factor = %#v", factor)
	}

	price := fields["price"]
	wantPrice := decimal.NewFromFloat(19.99)
	gotPrice, ok := price.Default.(decimal.Decimal)
	if price.Type != model.FieldTypeDecimal || price.Precision != 10 || price.Scale != 2 || !ok || !gotPrice.Equal(wantPrice) {
		t.Fatalf("price = %#v", price)
	}

	isActive := fields["is_active"]
	if isActive.Type != model.FieldTypeBool || !isActive.Required || isActive.Default != false {
		t.Fatalf("is_active = %#v", isActive)
	}

	birthDate := fields["birth_date"]
	if birthDate.Type != model.FieldTypeDate || birthDate.Default != "2000-01-01" {
		t.Fatalf("birth_date = %#v", birthDate)
	}

	startTime := fields["start_time"]
	if startTime.Type != model.FieldTypeTime || startTime.Default != "09:00:00" {
		t.Fatalf("start_time = %#v", startTime)
	}

	publishedAt := fields["published_at"]
	wantPublishedAt := time.Date(2026, 1, 2, 15, 4, 5, 0, time.UTC)
	gotPublishedAt, ok := publishedAt.Default.(time.Time)
	if publishedAt.Type != model.FieldTypeDateTime || !ok || !gotPublishedAt.Equal(wantPublishedAt) {
		t.Fatalf("published_at = %#v", publishedAt)
	}

	sessionTimeout := fields["session_timeout"]
	wantTimeout := 25*time.Hour + 30*time.Minute
	if sessionTimeout.Type != model.FieldTypeInterval || sessionTimeout.Default != wantTimeout {
		t.Fatalf("session_timeout = %#v, want default %v", sessionTimeout, wantTimeout)
	}

	settings := fields["settings"]
	settingsMap, ok := settings.Default.(map[string]any)
	if settings.Type != model.FieldTypeDict || !ok || settingsMap["theme"] != "dark" {
		t.Fatalf("settings = %#v", settings)
	}

	extra := fields["extra"]
	if extra.Type != model.FieldTypeDict || extra.Required || extra.Default != nil {
		t.Fatalf("extra = %#v", extra)
	}
}

func TestSetColumnType_OverridesIntrospectedType(t *testing.T) {
	db := setupDB(t)

	if _, err := db.Exec("DROP TABLE IF EXISTS gizmos"); err != nil {
		t.Fatalf("drop gizmos: %v", err)
	}
	t.Cleanup(func() {
		db.Exec("DROP TABLE IF EXISTS gizmos")
		db.Exec("DELETE FROM lx_sys.model_types WHERE table_name = 'public.gizmos'")
	})

	// settings is a plain text column - nothing in information_schema
	// distinguishes it from an ordinary string column, which is exactly
	// the ambiguity SetColumnType exists to resolve.
	_, err := db.Exec(`CREATE TABLE gizmos (id serial PRIMARY KEY, settings text DEFAULT '{"a":1}')`)
	if err != nil {
		t.Fatalf("create gizmos: %v", err)
	}

	schema, err := model.IntrospectModelSchema(db, "gizmos", "public", false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema (before override): %v", err)
	}
	settings, ok := schema.FieldByName("settings")
	if !ok || settings.Type != model.FieldTypeString {
		t.Fatalf("settings before override = %#v, want FieldTypeString", settings)
	}

	if err := model.SetColumnType(db, "public.gizmos", "settings", model.Field{Type: model.FieldTypeDict}); err != nil {
		t.Fatalf("SetColumnType: %v", err)
	}

	schema, err = model.IntrospectModelSchema(db, "gizmos", "public", false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema (after override): %v", err)
	}
	settings, ok = schema.FieldByName("settings")
	settingsMap, mapOK := settings.Default.(map[string]any)
	if !ok || settings.Type != model.FieldTypeDict || !mapOK || settingsMap["a"] != float64(1) {
		t.Fatalf("settings after override = %#v, want FieldTypeDict with default {a:1}", settings)
	}

	if err := model.DeleteColumnType(db, "public.gizmos", "settings"); err != nil {
		t.Fatalf("DeleteColumnType: %v", err)
	}

	schema, err = model.IntrospectModelSchema(db, "gizmos", "public", false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema (after delete): %v", err)
	}
	settings, ok = schema.FieldByName("settings")
	if !ok || settings.Type != model.FieldTypeString {
		t.Fatalf("settings after delete = %#v, want back to FieldTypeString", settings)
	}
}

func TestIntrospectModelSchema_TableNotFound(t *testing.T) {
	db := setupDB(t)

	if _, err := db.Exec("DROP TABLE IF EXISTS nonexistent_widgets"); err != nil {
		t.Fatalf("drop nonexistent_widgets: %v", err)
	}

	_, err := model.IntrospectModelSchema(db, "nonexistent_widgets", "public", false)
	if err != model.ErrTableNotFound {
		t.Fatalf("err = %v, want ErrTableNotFound", err)
	}
}
