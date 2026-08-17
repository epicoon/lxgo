package model_test

import (
	"path/filepath"
	"testing"

	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/apptest"
	"github.com/epicoon/lxgo/migrator"
	"github.com/epicoon/lxgo/model"
)

func newTestApp(t *testing.T) kernel.IApp {
	t.Helper()
	app, err := apptest.New(kernel.Dict{
		"Components": kernel.Dict{
			"ModelManager": kernel.Dict{
				"Targets": []kernel.Dict{
					{"Schemas": "/tmp/schemas"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("apptest.New: %v", err)
	}
	return app
}

// TestSetAppComponent covers both SetAppComponent/AfterInit outcomes in
// one function, deliberately - migrator.RegisterMigrationType writes into
// migrator's own package-level registry, shared by every test in this
// binary and with no way to unregister a name. Checking "AfterInit
// registered MigrationType" only works the first time it happens per
// process (every later SetAppComponent call - a different app or not -
// hits the same name and its AfterInit's registration attempt silently
// fails, logged, not surfaced as an error). Keeping both assertions in one
// function, run in a fixed sequence, makes that ordering explicit instead
// of relying on which of two separate Test functions the test runner
// happens to execute first.
func TestSetAppComponent(t *testing.T) {
	app := newTestApp(t)

	if err := model.SetAppComponent(app, "Components.ModelManager"); err != nil {
		t.Fatalf("SetAppComponent: %v", err)
	}

	if _, err := model.AppComponent(app); err != nil {
		t.Fatalf("AppComponent: %v", err)
	}

	// AfterInit (run synchronously inside SetAppComponent, above) should
	// have already registered MigrationType with migrator's registry - a
	// bare re-registration attempt must fail. This is the one point in the
	// whole test binary where that's actually verified - see the doc
	// comment above.
	if err := migrator.RegisterMigrationType(model.MigrationType, model.Apply, model.Invert); err == nil {
		t.Fatal("expected MigrationType to already be registered by AfterInit")
	}

	// A second SetAppComponent on the same app fails at the app-level
	// HasComponent guard, before ever reaching AfterInit/migrator again.
	if err := model.SetAppComponent(app, "Components.ModelManager"); err == nil {
		t.Fatal("expected a second SetAppComponent on the same app to fail")
	}
}

func TestAppComponent_NotRegisteredFails(t *testing.T) {
	app := newTestApp(t)

	if _, err := model.AppComponent(app); err == nil {
		t.Fatal("expected AppComponent to fail before SetAppComponent is called")
	}
}

// TestModelManager_LoadModelSchemas_NamespaceCascade checks the full
// 3-level cascade end to end through the real component config: a
// model's own Namespace wins over its Target's, which wins over the
// component-wide default, which applies when neither of the narrower two
// says anything.
func TestModelManager_LoadModelSchemas_NamespaceCascade(t *testing.T) {
	dirA := t.TempDir()
	dirB := t.TempDir()

	// dirA: no Target-level override - falls back to the component-wide
	// default, except OwnOverride, which sets its own Namespace directly.
	fallsBack := &model.ModelSchema{Name: "FallsBack", Fields: []model.NamedField{{Name: "x", Field: model.Field{Type: model.FieldTypeInt}}}}
	if err := fallsBack.Save(filepath.Join(dirA, "FallsBack.yaml")); err != nil {
		t.Fatalf("save FallsBack: %v", err)
	}
	ownOverride := &model.ModelSchema{Name: "OwnOverride", Namespace: "own_ns", Fields: []model.NamedField{{Name: "x", Field: model.Field{Type: model.FieldTypeInt}}}}
	if err := ownOverride.Save(filepath.Join(dirA, "OwnOverride.yaml")); err != nil {
		t.Fatalf("save OwnOverride: %v", err)
	}

	// dirB: Target-level override - wins over the component-wide default.
	targetLevel := &model.ModelSchema{Name: "TargetLevel", Fields: []model.NamedField{{Name: "x", Field: model.Field{Type: model.FieldTypeInt}}}}
	if err := targetLevel.Save(filepath.Join(dirB, "TargetLevel.yaml")); err != nil {
		t.Fatalf("save TargetLevel: %v", err)
	}

	app, err := apptest.New(kernel.Dict{
		"Components": kernel.Dict{
			"ModelManager": kernel.Dict{
				"Namespace": "default_ns",
				"Targets": []kernel.Dict{
					{"Schemas": dirA},
					{"Schemas": dirB, "Namespace": "target_ns"},
				},
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
	if len(schemas) != 3 {
		t.Fatalf("len(schemas) = %d, want 3", len(schemas))
	}

	byName := make(map[string]*model.ModelSchema, len(schemas))
	for _, s := range schemas {
		byName[s.Name] = s
	}

	if got := byName["FallsBack"].ResolvedNamespace; got != "default_ns" {
		t.Errorf("FallsBack.ResolvedNamespace = %q, want %q (component-wide default)", got, "default_ns")
	}
	if got := byName["OwnOverride"].ResolvedNamespace; got != "own_ns" {
		t.Errorf("OwnOverride.ResolvedNamespace = %q, want %q (model's own Namespace wins)", got, "own_ns")
	}
	if got := byName["TargetLevel"].ResolvedNamespace; got != "target_ns" {
		t.Errorf("TargetLevel.ResolvedNamespace = %q, want %q (Target's own Namespace wins over component default)", got, "target_ns")
	}
}

// TestModelManager_LoadModelSchemas_BaseModelCascade mirrors
// TestModelManager_LoadModelSchemas_NamespaceCascade for BaseModel - the
// same 3-level cascade (model's own, else its Target's, else the
// component-wide default), resolved independently of Namespace's own
// cascade.
func TestModelManager_LoadModelSchemas_BaseModelCascade(t *testing.T) {
	dirA := t.TempDir()
	dirB := t.TempDir()

	fallsBack := &model.ModelSchema{Name: "FallsBack", Fields: []model.NamedField{{Name: "x", Field: model.Field{Type: model.FieldTypeInt}}}}
	if err := fallsBack.Save(filepath.Join(dirA, "FallsBack.yaml")); err != nil {
		t.Fatalf("save FallsBack: %v", err)
	}
	ownOverride := &model.ModelSchema{Name: "OwnOverride", BaseModel: "own/pkg.Base", Fields: []model.NamedField{{Name: "x", Field: model.Field{Type: model.FieldTypeInt}}}}
	if err := ownOverride.Save(filepath.Join(dirA, "OwnOverride.yaml")); err != nil {
		t.Fatalf("save OwnOverride: %v", err)
	}

	targetLevel := &model.ModelSchema{Name: "TargetLevel", Fields: []model.NamedField{{Name: "x", Field: model.Field{Type: model.FieldTypeInt}}}}
	if err := targetLevel.Save(filepath.Join(dirB, "TargetLevel.yaml")); err != nil {
		t.Fatalf("save TargetLevel: %v", err)
	}

	app, err := apptest.New(kernel.Dict{
		"Components": kernel.Dict{
			"ModelManager": kernel.Dict{
				"BaseModel": "default/pkg.Base",
				"Targets": []kernel.Dict{
					{"Schemas": dirA},
					{"Schemas": dirB, "BaseModel": "target/pkg.Base"},
				},
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
	if len(schemas) != 3 {
		t.Fatalf("len(schemas) = %d, want 3", len(schemas))
	}

	byName := make(map[string]*model.ModelSchema, len(schemas))
	for _, s := range schemas {
		byName[s.Name] = s
	}

	if got := byName["FallsBack"].ResolvedBaseModel; got != "default/pkg.Base" {
		t.Errorf("FallsBack.ResolvedBaseModel = %q, want %q (component-wide default)", got, "default/pkg.Base")
	}
	if got := byName["OwnOverride"].ResolvedBaseModel; got != "own/pkg.Base" {
		t.Errorf("OwnOverride.ResolvedBaseModel = %q, want %q (model's own BaseModel wins)", got, "own/pkg.Base")
	}
	if got := byName["TargetLevel"].ResolvedBaseModel; got != "target/pkg.Base" {
		t.Errorf("TargetLevel.ResolvedBaseModel = %q, want %q (Target's own BaseModel wins over component default)", got, "target/pkg.Base")
	}
}

// TestModelManager_LoadModelSchemas_BaseRepoCascade mirrors
// TestModelManager_LoadModelSchemas_BaseModelCascade for BaseRepo - the
// same 3-level cascade, resolved independently of Namespace/BaseModel/
// Timestamps's own cascades.
func TestModelManager_LoadModelSchemas_BaseRepoCascade(t *testing.T) {
	dirA := t.TempDir()
	dirB := t.TempDir()

	fallsBack := &model.ModelSchema{Name: "FallsBack", Fields: []model.NamedField{{Name: "x", Field: model.Field{Type: model.FieldTypeInt}}}}
	if err := fallsBack.Save(filepath.Join(dirA, "FallsBack.yaml")); err != nil {
		t.Fatalf("save FallsBack: %v", err)
	}
	ownOverride := &model.ModelSchema{Name: "OwnOverride", BaseRepo: "own/pkg.Repo", Fields: []model.NamedField{{Name: "x", Field: model.Field{Type: model.FieldTypeInt}}}}
	if err := ownOverride.Save(filepath.Join(dirA, "OwnOverride.yaml")); err != nil {
		t.Fatalf("save OwnOverride: %v", err)
	}

	targetLevel := &model.ModelSchema{Name: "TargetLevel", Fields: []model.NamedField{{Name: "x", Field: model.Field{Type: model.FieldTypeInt}}}}
	if err := targetLevel.Save(filepath.Join(dirB, "TargetLevel.yaml")); err != nil {
		t.Fatalf("save TargetLevel: %v", err)
	}

	app, err := apptest.New(kernel.Dict{
		"Components": kernel.Dict{
			"ModelManager": kernel.Dict{
				"BaseRepo": "default/pkg.Repo",
				"Targets": []kernel.Dict{
					{"Schemas": dirA},
					{"Schemas": dirB, "BaseRepo": "target/pkg.Repo"},
				},
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
	if len(schemas) != 3 {
		t.Fatalf("len(schemas) = %d, want 3", len(schemas))
	}

	byName := make(map[string]*model.ModelSchema, len(schemas))
	for _, s := range schemas {
		byName[s.Name] = s
	}

	if got := byName["FallsBack"].ResolvedBaseRepo; got != "default/pkg.Repo" {
		t.Errorf("FallsBack.ResolvedBaseRepo = %q, want %q (component-wide default)", got, "default/pkg.Repo")
	}
	if got := byName["OwnOverride"].ResolvedBaseRepo; got != "own/pkg.Repo" {
		t.Errorf("OwnOverride.ResolvedBaseRepo = %q, want %q (model's own BaseRepo wins)", got, "own/pkg.Repo")
	}
	if got := byName["TargetLevel"].ResolvedBaseRepo; got != "target/pkg.Repo" {
		t.Errorf("TargetLevel.ResolvedBaseRepo = %q, want %q (Target's own BaseRepo wins over component default)", got, "target/pkg.Repo")
	}
}

// TestModelManager_LoadModelSchemas_TimestampsCascade mirrors
// TestModelManager_LoadModelSchemas_NamespaceCascade for Timestamps - the
// same 3-level cascade (model's own, else its Target's, else the
// component-wide default), resolved independently of Namespace/BaseModel's
// own cascades. The component-wide default here is true (unlike the
// Namespace/BaseModel tests' empty/zero default) specifically to exercise
// an explicit false winning over an inherited true - the case a plain bool
// field couldn't represent at all, which is why Timestamps/ResolvedTimestamps
// are *bool (see ModelSchema.Timestamps's doc).
func TestModelManager_LoadModelSchemas_TimestampsCascade(t *testing.T) {
	dirA := t.TempDir()
	dirB := t.TempDir()

	falseVal := false
	fallsBack := &model.ModelSchema{Name: "FallsBack", Fields: []model.NamedField{{Name: "x", Field: model.Field{Type: model.FieldTypeInt}}}}
	if err := fallsBack.Save(filepath.Join(dirA, "FallsBack.yaml")); err != nil {
		t.Fatalf("save FallsBack: %v", err)
	}
	ownOverride := &model.ModelSchema{Name: "OwnOverride", Timestamps: &falseVal, Fields: []model.NamedField{{Name: "x", Field: model.Field{Type: model.FieldTypeInt}}}}
	if err := ownOverride.Save(filepath.Join(dirA, "OwnOverride.yaml")); err != nil {
		t.Fatalf("save OwnOverride: %v", err)
	}

	targetLevel := &model.ModelSchema{Name: "TargetLevel", Fields: []model.NamedField{{Name: "x", Field: model.Field{Type: model.FieldTypeInt}}}}
	if err := targetLevel.Save(filepath.Join(dirB, "TargetLevel.yaml")); err != nil {
		t.Fatalf("save TargetLevel: %v", err)
	}

	app, err := apptest.New(kernel.Dict{
		"Components": kernel.Dict{
			"ModelManager": kernel.Dict{
				"Timestamps": true,
				"Targets": []kernel.Dict{
					{"Schemas": dirA},
					{"Schemas": dirB, "Timestamps": false},
				},
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
	if len(schemas) != 3 {
		t.Fatalf("len(schemas) = %d, want 3", len(schemas))
	}

	byName := make(map[string]*model.ModelSchema, len(schemas))
	for _, s := range schemas {
		byName[s.Name] = s
	}

	if got := byName["FallsBack"].ResolvedTimestamps; got == nil || *got != true {
		t.Errorf("FallsBack.ResolvedTimestamps = %v, want pointer to true (component-wide default)", got)
	}
	if got := byName["OwnOverride"].ResolvedTimestamps; got == nil || *got != false {
		t.Errorf("OwnOverride.ResolvedTimestamps = %v, want pointer to false (model's own Timestamps wins, even over an inherited true)", got)
	}
	if got := byName["TargetLevel"].ResolvedTimestamps; got == nil || *got != false {
		t.Errorf("TargetLevel.ResolvedTimestamps = %v, want pointer to false (Target's own Timestamps wins over component default)", got)
	}
}

// TestModelManager_LoadModelSchemas_SameNameDifferentNamespaceAllowed checks
// that two models sharing a Name no longer collide once DDL/introspection
// schema-qualify physical objects (see IntrospectModelSchema/Apply/Invert) -
// they resolve to different Postgres schemas, so there's no actual physical
// collision for LoadModelSchemas to guard against.
func TestModelManager_LoadModelSchemas_SameNameDifferentNamespaceAllowed(t *testing.T) {
	dirA := t.TempDir()
	dirB := t.TempDir()

	a := &model.ModelSchema{Name: "Shared", Fields: []model.NamedField{{Name: "x", Field: model.Field{Type: model.FieldTypeInt}}}}
	if err := a.Save(filepath.Join(dirA, "Shared.yaml")); err != nil {
		t.Fatalf("save A: %v", err)
	}
	b := &model.ModelSchema{Name: "Shared", Fields: []model.NamedField{{Name: "y", Field: model.Field{Type: model.FieldTypeInt}}}}
	if err := b.Save(filepath.Join(dirB, "Shared.yaml")); err != nil {
		t.Fatalf("save B: %v", err)
	}

	app, err := apptest.New(kernel.Dict{
		"Components": kernel.Dict{
			"ModelManager": kernel.Dict{
				"Targets": []kernel.Dict{
					{"Schemas": dirA, "Namespace": "ns_a"},
					{"Schemas": dirB, "Namespace": "ns_b"},
				},
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
		t.Fatalf("LoadModelSchemas: %v, want success - same Name, different resolved schema", err)
	}
	if len(schemas) != 2 {
		t.Fatalf("len(schemas) = %d, want 2", len(schemas))
	}
}

// TestModelManager_LoadModelSchemas_SameNameSameNamespaceStillErrors checks
// that the duplicate-name guard still fires when two models would actually
// collide physically - same Name, same resolved namespace.
func TestModelManager_LoadModelSchemas_SameNameSameNamespaceStillErrors(t *testing.T) {
	dirA := t.TempDir()
	dirB := t.TempDir()

	a := &model.ModelSchema{Name: "Shared", Fields: []model.NamedField{{Name: "x", Field: model.Field{Type: model.FieldTypeInt}}}}
	if err := a.Save(filepath.Join(dirA, "Shared.yaml")); err != nil {
		t.Fatalf("save A: %v", err)
	}
	b := &model.ModelSchema{Name: "Shared", Fields: []model.NamedField{{Name: "y", Field: model.Field{Type: model.FieldTypeInt}}}}
	if err := b.Save(filepath.Join(dirB, "Shared.yaml")); err != nil {
		t.Fatalf("save B: %v", err)
	}

	app, err := apptest.New(kernel.Dict{
		"Components": kernel.Dict{
			"ModelManager": kernel.Dict{
				"Targets": []kernel.Dict{
					{"Schemas": dirA},
					{"Schemas": dirB},
				},
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

	if _, err := mm.LoadModelSchemas(); err == nil {
		t.Fatal("expected an error - same Name, same resolved schema (both default) collide physically")
	}
}

// TestModelManager_LoadModelSchemas_TimestampsForbidsCollidingField checks
// that a model whose resolved Timestamps is on can't also declare a Field
// colliding with created_at/updated_at/deleted_at - those columns are
// implicit under Timestamps (see IntrospectModelSchema), so an explicit
// Field under one of their names could never actually be represented
// physically (execCreateTable would emit a CREATE TABLE with a duplicate
// column).
func TestModelManager_LoadModelSchemas_TimestampsForbidsCollidingField(t *testing.T) {
	dir := t.TempDir()
	tru := true
	sch := &model.ModelSchema{
		Name:       "Widget",
		Timestamps: &tru,
		Fields: []model.NamedField{
			{Name: "CreatedAt", Field: model.Field{Type: model.FieldTypeDateTime, Required: true}},
		},
	}
	if err := sch.Save(filepath.Join(dir, "Widget.yaml")); err != nil {
		t.Fatalf("save schema: %v", err)
	}

	app, err := apptest.New(kernel.Dict{
		"Components": kernel.Dict{
			"ModelManager": kernel.Dict{
				"Targets": []kernel.Dict{{"Schemas": dir}},
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

	if _, err := mm.LoadModelSchemas(); err == nil {
		t.Fatal("expected an error for a Timestamps-enabled model declaring a field colliding with created_at/updated_at/deleted_at")
	}
}

// TestModelManager_LoadModelSchemas_TimestampsFalseAllowsCollidingField
// checks the mirror image - with Timestamps off, a model is free to
// declare an ordinary Field under any of the three names (see this
// test's counterpart above) - no restriction applies at all.
func TestModelManager_LoadModelSchemas_TimestampsFalseAllowsCollidingField(t *testing.T) {
	dir := t.TempDir()
	sch := &model.ModelSchema{
		Name: "Widget",
		Fields: []model.NamedField{
			{Name: "CreatedAt", Field: model.Field{Type: model.FieldTypeDateTime, Required: true}},
		},
	}
	if err := sch.Save(filepath.Join(dir, "Widget.yaml")); err != nil {
		t.Fatalf("save schema: %v", err)
	}

	app, err := apptest.New(kernel.Dict{
		"Components": kernel.Dict{
			"ModelManager": kernel.Dict{
				"Targets": []kernel.Dict{{"Schemas": dir}},
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

	if _, err := mm.LoadModelSchemas(); err != nil {
		t.Fatalf("LoadModelSchemas: %v", err)
	}
}
