package reconf

import (
	"testing"

	"github.com/epicoon/lxgo/kernel"
)

func TestCompareConfigs_EmptyOldSliceWithNewElements(t *testing.T) {
	// Regression: compareSlices used to always sample oldSlice.Index(0) to
	// infer the "expected" element type for newly-added elements, which
	// panicked whenever the old slice was empty (e.g. "Servers: []" plus
	// manage:inject-config --add Servers=[...]).
	orig := kernel.Dict{"Servers": []any{}}
	updated := kernel.Dict{"Servers": []any{"a", "b"}}

	rep := compareConfigs(orig, updated)

	if len(rep.errs) != 0 {
		t.Fatalf("expected no type-mismatch errors, got %+v", rep.errs)
	}
	if len(rep.added) != 2 {
		t.Fatalf("expected 2 added elements, got %+v", rep.added)
	}
}

func TestCompareConfigs_TypeMismatchAgainstExistingElement(t *testing.T) {
	orig := kernel.Dict{"Servers": []any{"a"}}
	updated := kernel.Dict{"Servers": []any{"a", 42}}

	rep := compareConfigs(orig, updated)

	if len(rep.errs) != 1 {
		t.Fatalf("expected 1 type-mismatch error, got %+v", rep.errs)
	}
}

func TestCompareConfigs_NestedDictAndScalarChange(t *testing.T) {
	orig := kernel.Dict{"Database": kernel.Dict{"Port": 5432}}
	updated := kernel.Dict{"Database": kernel.Dict{"Port": 5433}}

	rep := compareConfigs(orig, updated)

	if len(rep.changed) != 1 || rep.changed[0].Path != "Database.Port" {
		t.Fatalf("expected 1 change at 'Database.Port', got %+v", rep.changed)
	}
}
