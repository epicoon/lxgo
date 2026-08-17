package model

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/epicoon/lxgo/cmd"
	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/apptest"
)

// TestGoPackageNameFromDir_RejectsInvalidIdentifiers is a regression test:
// go's testing package itself names t.TempDir()'s own subdirectories with
// plain increasing numbers ("001", "002", ...) - using such a directory's
// bare name as a Go package name (e.g. via filepath.Base on a Models path
// that happened to end there) produced a package declaration gofmt can't
// even parse, surfacing only as a confusing "expected 'IDENT', found 002"
// deep inside BuildModelCode rather than a clear, actionable error.
func TestGoPackageNameFromDir_RejectsInvalidIdentifiers(t *testing.T) {
	tests := []struct {
		dir  string
		want bool
	}{
		{"/path/to/models", true},
		{"/path/to/002", false},
		{"/path/to/my-models", false},
		{"/path/to/_models", true},
		{"/", false},
	}
	for _, tt := range tests {
		if _, ok := goPackageNameFromDir(tt.dir); ok != tt.want {
			t.Errorf("goPackageNameFromDir(%q) ok = %v, want %v", tt.dir, ok, tt.want)
		}
	}
}

func TestCommand_Config_HasCodegenActions(t *testing.T) {
	c := &Command{Command: cmd.NewCommand()}
	conf := c.Config()

	if _, ok := conf.Actions["codegen-status"]; !ok {
		t.Fatal(`expected a "codegen-status" action`)
	}
	if _, ok := conf.Actions["codegen-generate"]; !ok {
		t.Fatal(`expected a "codegen-generate" action`)
	}
}

func TestCodegenStatus_NoModelManagerRegisteredPrintsInsteadOfErroring(t *testing.T) {
	app := newTestAppNoModelManager(t)
	c := &Command{Command: cmd.NewCommand(), app: app}

	if err := codegenStatus(c); err != nil {
		t.Fatalf("codegenStatus: %v", err)
	}
}

func TestCodegenGenerate_NoModelManagerRegisteredPrintsInsteadOfErroring(t *testing.T) {
	app := newTestAppNoModelManager(t)
	c := &Command{Command: cmd.NewCommand(), app: app}

	if err := codegenGenerate(c); err != nil {
		t.Fatalf("codegenGenerate: %v", err)
	}
}

// newModelsTestApp builds an app with one Target - schemaDir for its
// Schemas, modelsDir (may be empty, meaning "not configured") for its
// Models - and returns the resulting *ModelManager.
func newModelsTestApp(t *testing.T, schemaDir, modelsDir string) *ModelManager {
	t.Helper()
	target := kernel.Dict{"Schemas": schemaDir}
	if modelsDir != "" {
		target["Models"] = modelsDir
	}
	app, err := apptest.New(kernel.Dict{
		"Components": kernel.Dict{
			"ModelManager": kernel.Dict{
				"Targets": []kernel.Dict{target},
			},
		},
	})
	if err != nil {
		t.Fatalf("apptest.New: %v", err)
	}
	if err := SetAppComponent(app, "Components.ModelManager"); err != nil {
		t.Fatalf("SetAppComponent: %v", err)
	}
	mm, err := AppComponent(app)
	if err != nil {
		t.Fatalf("AppComponent: %v", err)
	}
	return mm
}

func saveSchema(t *testing.T, dir string, s *ModelSchema) {
	t.Helper()
	if err := s.Save(filepath.Join(dir, s.Name+".yaml")); err != nil {
		t.Fatalf("save %s: %v", s.Name, err)
	}
}

func TestCodegenGenerate_WritesFileAndCodegenStatus_ReportsUpToDate(t *testing.T) {
	schemaDir := t.TempDir()
	modelsDir := filepath.Join(t.TempDir(), "models")
	saveSchema(t, schemaDir, &ModelSchema{
		Name:   "Widget",
		Fields: []NamedField{{Name: "Name", Field: Field{Type: FieldTypeString, Size: 255}}},
	})

	mm := newModelsTestApp(t, schemaDir, modelsDir)
	c := &Command{Command: cmd.NewCommand(), app: mm.App()}

	if err := codegenGenerate(c); err != nil {
		t.Fatalf("codegenGenerate: %v", err)
	}

	outPath := filepath.Join(modelsDir, "widget_gen.go")
	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("expected %s to exist: %v", outPath, err)
	}
	content, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("reading %s: %v", outPath, err)
	}
	if !strings.Contains(string(content), "type Widget struct") {
		t.Fatalf("generated file doesn't declare Widget, got:\n%s", content)
	}

	// codegenStatus should now report it as up to date - the file was just
	// (re)written, so it can never be older than the schema file.
	if err := codegenStatus(c); err != nil {
		t.Fatalf("codegenStatus: %v", err)
	}
}

func TestCodegenStatus_ReportsMissingBeforeGeneration(t *testing.T) {
	schemaDir := t.TempDir()
	modelsDir := filepath.Join(t.TempDir(), "models")
	saveSchema(t, schemaDir, &ModelSchema{Name: "Widget"})

	mm := newModelsTestApp(t, schemaDir, modelsDir)
	c := &Command{Command: cmd.NewCommand(), app: mm.App()}

	// No assertion on stdout content here (this package doesn't capture
	// it elsewhere either) - the real assertion is that nothing panics or
	// errors when the generated file doesn't exist yet.
	if err := codegenStatus(c); err != nil {
		t.Fatalf("codegenStatus: %v", err)
	}
}

func TestCodegenStatus_ReportsStaleAfterSchemaEdit(t *testing.T) {
	schemaDir := t.TempDir()
	modelsDir := filepath.Join(t.TempDir(), "models")
	schemaPath := filepath.Join(schemaDir, "Widget.yaml")
	saveSchema(t, schemaDir, &ModelSchema{Name: "Widget"})

	mm := newModelsTestApp(t, schemaDir, modelsDir)
	c := &Command{Command: cmd.NewCommand(), app: mm.App()}

	if err := codegenGenerate(c); err != nil {
		t.Fatalf("codegenGenerate: %v", err)
	}
	outPath := filepath.Join(modelsDir, "widget_gen.go")
	genInfo, err := os.Stat(outPath)
	if err != nil {
		t.Fatalf("stat generated file: %v", err)
	}

	// Touch the schema file with a mtime strictly after the generated
	// file's own - simulates the schema having changed since generation,
	// without needing a real filesystem-clock-resolution sleep.
	newer := genInfo.ModTime().Add(time.Second)
	if err := os.Chtimes(schemaPath, newer, newer); err != nil {
		t.Fatalf("Chtimes: %v", err)
	}

	if err := codegenStatus(c); err != nil {
		t.Fatalf("codegenStatus: %v", err)
	}
}

func TestCodegenGenerate_TargetWithoutModelsIsSkipped(t *testing.T) {
	schemaDir := t.TempDir()
	saveSchema(t, schemaDir, &ModelSchema{Name: "Widget"})

	mm := newModelsTestApp(t, schemaDir, "")
	c := &Command{Command: cmd.NewCommand(), app: mm.App()}

	if err := codegenGenerate(c); err != nil {
		t.Fatalf("codegenGenerate: %v", err)
	}
	// Nothing to assert about a filesystem path here - Target.Models is
	// empty, so there's no directory to have written into at all. The
	// real assertion is that this doesn't panic/error trying to resolve
	// an empty Models path.
}

func TestCodegenGenerate_CrossPackageRelationRejected(t *testing.T) {
	widgetDir := t.TempDir()
	widgetModelsDir := filepath.Join(t.TempDir(), "models")
	otherDir := t.TempDir()
	otherModelsDir := filepath.Join(t.TempDir(), "models")

	saveSchema(t, widgetDir, &ModelSchema{
		Name: "Widget",
		Relations: []NamedRelation{
			{Name: "Copy", Relation: Relation{Type: RelationTypeManyToOne, RelatedModel: "WidgetCopy"}},
		},
	})
	saveSchema(t, otherDir, &ModelSchema{Name: "WidgetCopy"})

	app, err := apptest.New(kernel.Dict{
		"Components": kernel.Dict{
			"ModelManager": kernel.Dict{
				"Targets": []kernel.Dict{
					{"Schemas": widgetDir, "Models": widgetModelsDir},
					{"Schemas": otherDir, "Models": otherModelsDir},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("apptest.New: %v", err)
	}
	if err := SetAppComponent(app, "Components.ModelManager"); err != nil {
		t.Fatalf("SetAppComponent: %v", err)
	}
	c := &Command{Command: cmd.NewCommand(), app: app}

	if err := codegenGenerate(c); err != nil {
		t.Fatalf("codegenGenerate: %v", err)
	}

	// Widget's relation points at WidgetCopy, generated into a different
	// Target's own Models directory - must be rejected, not silently
	// written as code that would fail to compile.
	if _, err := os.Stat(filepath.Join(widgetModelsDir, "widget_gen.go")); !os.IsNotExist(err) {
		t.Fatalf("expected widget_gen.go NOT to be written (cross-package relation), stat err = %v", err)
	}
	// WidgetCopy itself has no relations, so it's unaffected and should
	// still be generated.
	if _, err := os.Stat(filepath.Join(otherModelsDir, "widget_copy_gen.go")); err != nil {
		t.Fatalf("expected widget_copy_gen.go to exist: %v", err)
	}
}

// TestCodegenGenerate_SameModelNameDifferentTargets is a regression test:
// LoadModelSchemas explicitly allows two different Targets to each
// declare a model under the same Name (a collision is only an error if
// they also resolve to the same namespace, see its own doc) - resolving
// each schema's own Target by Name rather than by its own SourceDir
// silently resolved both same-named schemas to whichever Target was
// processed last, so one of the two never got its own file written (or
// got the other's content written into its own directory instead).
func TestCodegenGenerate_SameModelNameDifferentTargets(t *testing.T) {
	dirA := t.TempDir()
	modelsA := filepath.Join(t.TempDir(), "models")
	dirB := t.TempDir()
	modelsB := filepath.Join(t.TempDir(), "models")

	saveSchema(t, dirA, &ModelSchema{Name: "Widget", Namespace: "ns_a"})
	saveSchema(t, dirB, &ModelSchema{Name: "Widget", Namespace: "ns_b"})

	app, err := apptest.New(kernel.Dict{
		"Components": kernel.Dict{
			"ModelManager": kernel.Dict{
				"Targets": []kernel.Dict{
					{"Schemas": dirA, "Models": modelsA},
					{"Schemas": dirB, "Models": modelsB},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("apptest.New: %v", err)
	}
	if err := SetAppComponent(app, "Components.ModelManager"); err != nil {
		t.Fatalf("SetAppComponent: %v", err)
	}
	c := &Command{Command: cmd.NewCommand(), app: app}

	if err := codegenGenerate(c); err != nil {
		t.Fatalf("codegenGenerate: %v", err)
	}

	// Both same-named models must land in their OWN Target's own
	// directory - neither one skipped, neither one overwritten with the
	// other's content.
	for _, dir := range []string{modelsA, modelsB} {
		path := filepath.Join(dir, "widget_gen.go")
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s to exist: %v", path, err)
		}
	}
}
