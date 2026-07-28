package migrator

import "testing"

func TestNewMigration_Valid(t *testing.T) {
	m, err := NewMigration("20260728120000.000_add_users.yaml")
	if err != nil {
		t.Fatalf("NewMigration: %v", err)
	}
	if m.getTimestamp() != "20260728120000.000" {
		t.Fatalf("timestamp = %q", m.getTimestamp())
	}
	if m.getName() != "add_users" {
		t.Fatalf("name = %q, want 'add_users' (underscores in the name must survive the split)", m.getName())
	}
	if m.extension != ".yaml" {
		t.Fatalf("extension = %q", m.extension)
	}
	if m.String() != "20260728120000.000_add_users.yaml" {
		t.Fatalf("String() = %q", m.String())
	}
}

func TestNewMigration_NoExtension(t *testing.T) {
	if _, err := NewMigration("20260728120000_add_users"); err == nil {
		t.Fatal("expected an error for a filename with no extension")
	}
}

func TestNewMigration_NoUnderscoreSeparator(t *testing.T) {
	if _, err := NewMigration("20260728120000.yaml"); err == nil {
		t.Fatal("expected an error for a filename with no 'timestamp_name' separator")
	}
}

func TestNewMigration_StripsDirectoryFromFile(t *testing.T) {
	m, err := NewMigration("/some/path/20260728120000_add_users.yaml")
	if err != nil {
		t.Fatalf("NewMigration: %v", err)
	}
	if m.getName() != "add_users" {
		t.Fatalf("name = %q", m.getName())
	}
}

func TestMigration_AppliedFlag(t *testing.T) {
	m, err := NewMigration("20260728120000_add_users.yaml")
	if err != nil {
		t.Fatalf("NewMigration: %v", err)
	}
	if m.isApplied() {
		t.Fatal("expected a freshly-parsed migration to not be applied")
	}
	m.setApplied(true)
	if !m.isApplied() {
		t.Fatal("expected isApplied() to be true after setApplied(true)")
	}
}

func newTestMigration(t *testing.T, filename string, applied bool) *migration {
	t.Helper()
	m, err := NewMigration(filename)
	if err != nil {
		t.Fatalf("NewMigration(%q): %v", filename, err)
	}
	m.setApplied(applied)
	return m
}

func TestFilterMigrations_All(t *testing.T) {
	list := []*migration{
		newTestMigration(t, "1_a.yaml", true),
		newTestMigration(t, "2_b.yaml", false),
	}
	got := filterMigrations(list, cGET_ALL)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
}

func TestFilterMigrations_AppliedOnly(t *testing.T) {
	list := []*migration{
		newTestMigration(t, "1_a.yaml", true),
		newTestMigration(t, "2_b.yaml", false),
		newTestMigration(t, "3_c.yaml", true),
	}
	got := filterMigrations(list, cGET_APPLIED_ONLY)
	if len(got) != 2 || got[0].getName() != "a" || got[1].getName() != "c" {
		t.Fatalf("got = %#v", got)
	}
}

func TestFilterMigrations_UnappliedOnly(t *testing.T) {
	list := []*migration{
		newTestMigration(t, "1_a.yaml", true),
		newTestMigration(t, "2_b.yaml", false),
		newTestMigration(t, "3_c.yaml", true),
	}
	got := filterMigrations(list, cGET_UNAPPLIED_ONLY)
	if len(got) != 1 || got[0].getName() != "b" {
		t.Fatalf("got = %#v", got)
	}
}
