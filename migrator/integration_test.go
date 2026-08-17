//go:build integration

package migrator_test

import (
	"database/sql"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	_ "github.com/lib/pq"

	"github.com/epicoon/lxgo/migrator"
)

func testDSN() string {
	dsn := os.Getenv("LXGO_MIGRATOR_TEST_DSN")
	if dsn == "" {
		dsn = "host=localhost user=lx password=123456 dbname=lxgomigratortest port=55433 sslmode=disable"
	}
	return dsn
}

// setupMigrator wires up a fresh migrations directory and a clean
// lx_sys.migrator table for one test - migrator's state is package-level
// (var m), so tests must run sequentially (no t.Parallel()), each
// resetting it via Init.
func setupMigrator(t *testing.T) (db *sql.DB, migrationsDir string) {
	t.Helper()
	db, err := sql.Open("postgres", testDSN())
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Fatalf("db.Ping: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	if _, err := db.Exec("DROP TABLE IF EXISTS lx_sys.migrator"); err != nil {
		t.Fatalf("drop lx_sys.migrator: %v", err)
	}

	migrationsDir = t.TempDir()
	migrator.Init(migrator.Config{DB: db, MigrationsPath: migrationsDir})

	return db, migrationsDir
}

func writeMigration(t *testing.T, dir, filename, up, down string) {
	t.Helper()
	content := "Name: test\nType: query\n\nUp: " + up + "\n\nDown: " + down + "\n"
	if err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0644); err != nil {
		t.Fatalf("write migration file: %v", err)
	}
}

func tableExists(t *testing.T, db *sql.DB, name string) bool {
	t.Helper()
	var count int
	err := db.QueryRow(
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_name = $1", name,
	).Scan(&count)
	if err != nil {
		t.Fatalf("check table %s: %v", name, err)
	}
	return count > 0
}

func appliedCount(t *testing.T, db *sql.DB) int {
	t.Helper()
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM lx_sys.migrator").Scan(&count); err != nil {
		t.Fatalf("count lx_sys.migrator: %v", err)
	}
	return count
}

func TestUp_AppliesMigrationsAndTracksThem(t *testing.T) {
	db, dir := setupMigrator(t)
	t.Cleanup(func() {
		db.Exec("DROP TABLE IF EXISTS widgets_a")
		db.Exec("DROP TABLE IF EXISTS widgets_b")
	})

	writeMigration(t, dir, "00000001_create_a.yaml",
		"CREATE TABLE widgets_a (id serial primary key)",
		"DROP TABLE widgets_a")
	writeMigration(t, dir, "00000002_create_b.yaml",
		"CREATE TABLE widgets_b (id serial primary key)",
		"DROP TABLE widgets_b")

	migrator.Up()

	if !tableExists(t, db, "widgets_a") || !tableExists(t, db, "widgets_b") {
		t.Fatal("expected both migrations' tables to exist after Up()")
	}
	if got := appliedCount(t, db); got != 2 {
		t.Fatalf("lx_sys.migrator rows = %d, want 2", got)
	}
}

func TestDown_RollsBackLastMigrationOnly(t *testing.T) {
	db, dir := setupMigrator(t)
	t.Cleanup(func() {
		db.Exec("DROP TABLE IF EXISTS widgets_a")
		db.Exec("DROP TABLE IF EXISTS widgets_b")
	})

	writeMigration(t, dir, "00000001_create_a.yaml",
		"CREATE TABLE widgets_a (id serial primary key)",
		"DROP TABLE widgets_a")
	writeMigration(t, dir, "00000002_create_b.yaml",
		"CREATE TABLE widgets_b (id serial primary key)",
		"DROP TABLE widgets_b")
	migrator.Up()

	migrator.Down(0)

	if tableExists(t, db, "widgets_b") {
		t.Fatal("expected the last migration's table to be dropped by Down(0)")
	}
	if !tableExists(t, db, "widgets_a") {
		t.Fatal("expected the first migration's table to survive Down(0)")
	}
	if got := appliedCount(t, db); got != 1 {
		t.Fatalf("lx_sys.migrator rows = %d, want 1", got)
	}
}

// TestDown_NegativeStepsRollsBackOne is a regression test: Down(-1) used to
// roll back nothing at all (the clamping only handled steps > max and
// steps == 0), while still printing a success message.
func TestDown_NegativeStepsRollsBackOne(t *testing.T) {
	db, dir := setupMigrator(t)
	t.Cleanup(func() {
		db.Exec("DROP TABLE IF EXISTS widgets_a")
		db.Exec("DROP TABLE IF EXISTS widgets_b")
	})

	writeMigration(t, dir, "00000001_create_a.yaml",
		"CREATE TABLE widgets_a (id serial primary key)",
		"DROP TABLE widgets_a")
	writeMigration(t, dir, "00000002_create_b.yaml",
		"CREATE TABLE widgets_b (id serial primary key)",
		"DROP TABLE widgets_b")
	migrator.Up()

	migrator.Down(-1)

	if tableExists(t, db, "widgets_b") {
		t.Fatal("expected Down(-1) to roll back exactly one migration (the last)")
	}
	if !tableExists(t, db, "widgets_a") {
		t.Fatal("expected the first migration's table to survive Down(-1)")
	}
	if got := appliedCount(t, db); got != 1 {
		t.Fatalf("lx_sys.migrator rows = %d, want 1 (Down(-1) must roll back exactly one)", got)
	}
}

var createdFileNameRe = regexp.MustCompile(`^\d{14}\.\d{3}_hello\.yaml$`)

func TestCreate_WritesExpectedTemplate(t *testing.T) {
	_, dir := setupMigrator(t)

	if err := migrator.Create("hello"); err != nil {
		t.Fatalf("Create: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected exactly one created file, got %d", len(entries))
	}

	name := entries[0].Name()
	if !createdFileNameRe.MatchString(name) {
		t.Fatalf("filename = %q, want it to match %s", name, createdFileNameRe.String())
	}

	content, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	got := string(content)
	if !regexp.MustCompile(`(?m)^Name: hello$`).MatchString(got) {
		t.Fatalf("content = %q, want it to declare 'Name: hello'", got)
	}
	if !regexp.MustCompile(`(?m)^Up: `).MatchString(got) || !regexp.MustCompile(`(?m)^Down: `).MatchString(got) {
		t.Fatalf("content = %q, want Up:/Down: template placeholders", got)
	}
}

func TestUpSeeds_InsertsRows(t *testing.T) {
	db, migDir := setupMigrator(t)
	t.Cleanup(func() { db.Exec("DROP TABLE IF EXISTS seed_target") })

	writeMigration(t, migDir, "00000001_create_seed_target.yaml",
		"CREATE TABLE seed_target (id serial primary key, name text)",
		"DROP TABLE seed_target")
	migrator.Up()

	seedsDir := t.TempDir()
	seedContent := "- name: alice\n- name: bob\n"
	if err := os.WriteFile(filepath.Join(seedsDir, "seed_target.yaml"), []byte(seedContent), 0644); err != nil {
		t.Fatalf("write seed file: %v", err)
	}
	migrator.Init(migrator.Config{DB: db, MigrationsPath: migDir, SeedsPath: seedsDir})

	migrator.UpSeeds()

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM seed_target").Scan(&count); err != nil {
		t.Fatalf("count seed_target: %v", err)
	}
	if count != 2 {
		t.Fatalf("seed_target row count = %d, want 2", count)
	}
}
