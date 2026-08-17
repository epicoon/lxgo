package migrator

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func noopMigrationHandler(_ *sql.Tx, _ []byte) error { return nil }

func TestRegisterMigrationType_Registers(t *testing.T) {
	name := "test_register_once"
	t.Cleanup(func() { delete(migrationTypes, name) })

	if err := RegisterMigrationType(name, noopMigrationHandler, noopMigrationHandler); err != nil {
		t.Fatalf("RegisterMigrationType: %v", err)
	}
	if _, ok := migrationTypes[name]; !ok {
		t.Fatal("expected the type to be registered")
	}
}

func TestRegisterMigrationType_DuplicateNameFails(t *testing.T) {
	name := "test_register_dup"
	t.Cleanup(func() { delete(migrationTypes, name) })

	if err := RegisterMigrationType(name, noopMigrationHandler, noopMigrationHandler); err != nil {
		t.Fatalf("first RegisterMigrationType: %v", err)
	}
	if err := RegisterMigrationType(name, noopMigrationHandler, noopMigrationHandler); err == nil {
		t.Fatal("expected an error re-registering an already-used type name")
	}
}

func TestResolveMigrationAction_QueryTypeExplicit(t *testing.T) {
	content := []byte("Type: query\nUp: CREATE TABLE t (id int)\nDown: DROP TABLE t\n")

	action, err := resolveMigrationAction(content, "Up", "test.yaml")
	if err != nil {
		t.Fatalf("resolveMigrationAction: %v", err)
	}
	if action.handler != nil {
		t.Fatal("expected no handler for the query type")
	}
	if len(action.commands) != 1 || action.commands[0] != "CREATE TABLE t (id int)" {
		t.Fatalf("commands = %#v", action.commands)
	}
}

func TestResolveMigrationAction_UntypedIsQuery(t *testing.T) {
	content := []byte("Up: CREATE TABLE t (id int)\nDown: DROP TABLE t\n")

	upAction, err := resolveMigrationAction(content, "Up", "test.yaml")
	if err != nil {
		t.Fatalf("resolveMigrationAction(up): %v", err)
	}
	if len(upAction.commands) != 1 || upAction.commands[0] != "CREATE TABLE t (id int)" {
		t.Fatalf("up commands = %#v", upAction.commands)
	}

	downAction, err := resolveMigrationAction(content, "Down", "test.yaml")
	if err != nil {
		t.Fatalf("resolveMigrationAction(down): %v", err)
	}
	if len(downAction.commands) != 1 || downAction.commands[0] != "DROP TABLE t" {
		t.Fatalf("down commands = %#v", downAction.commands)
	}
}

func TestResolveMigrationAction_RegisteredTypeDispatchesAndPassesRawUntouched(t *testing.T) {
	name := "test_resolve_registered"
	t.Cleanup(func() { delete(migrationTypes, name) })

	var gotApplyRaw, gotInvertRaw []byte
	apply := func(_ *sql.Tx, raw []byte) error { gotApplyRaw = raw; return nil }
	invert := func(_ *sql.Tx, raw []byte) error { gotInvertRaw = raw; return nil }
	if err := RegisterMigrationType(name, apply, invert); err != nil {
		t.Fatalf("RegisterMigrationType: %v", err)
	}

	content := []byte("Type: " + name + "\nname: whatever\ncustom: data\n")

	upAction, err := resolveMigrationAction(content, "Up", "test.yaml")
	if err != nil {
		t.Fatalf("resolveMigrationAction(up): %v", err)
	}
	if upAction.handler == nil {
		t.Fatal("expected a handler for a registered type")
	}
	if err := upAction.run(nil); err != nil {
		t.Fatalf("run(up): %v", err)
	}
	if string(gotApplyRaw) != string(content) {
		t.Fatalf("apply got raw = %q, want the untouched original file content %q", gotApplyRaw, content)
	}

	downAction, err := resolveMigrationAction(content, "Down", "test.yaml")
	if err != nil {
		t.Fatalf("resolveMigrationAction(down): %v", err)
	}
	if err := downAction.run(nil); err != nil {
		t.Fatalf("run(down): %v", err)
	}
	if string(gotInvertRaw) != string(content) {
		t.Fatalf("invert got raw = %q, want the untouched original file content %q", gotInvertRaw, content)
	}
}

func TestResolveMigrationAction_UnknownTypeFails(t *testing.T) {
	content := []byte("Type: definitely_not_registered\n")

	if _, err := resolveMigrationAction(content, "Up", "test.yaml"); err == nil {
		t.Fatal("expected an error for an unregistered migration type")
	}
}

func TestCreateWithContent_WritesRawBytesUnchanged(t *testing.T) {
	dir := t.TempDir()
	origPath := m.migrationsPath
	m.migrationsPath = dir
	t.Cleanup(func() { m.migrationsPath = origPath })

	content := []byte("type: custom\nname: hello\nfoo: bar\n")
	if err := CreateWithContent("hello", content); err != nil {
		t.Fatalf("CreateWithContent: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected exactly one created file, got %d", len(entries))
	}

	got, err := os.ReadFile(filepath.Join(dir, entries[0].Name()))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != string(content) {
		t.Fatalf("file content = %q, want the content passed to CreateWithContent unchanged: %q", got, content)
	}
}
