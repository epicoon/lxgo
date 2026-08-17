package model

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/epicoon/lxgo/cmd"
	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/apptest"
)

func writeGoMod(t *testing.T, dir, module string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module "+module+"\n\ngo 1.23\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
}

func TestParseGoModModule(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
		wantErr bool
	}{
		{"simple", "module example.com/foo\n\ngo 1.23\n", "example.com/foo", false},
		{"leading comment", "// some comment\nmodule example.com/bar\n", "example.com/bar", false},
		{"no module directive", "go 1.23\n", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseGoModModule([]byte(tt.content))
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseGoModModule error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && got != tt.want {
				t.Fatalf("parseGoModModule = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGoModuleImportPath_ModuleRoot(t *testing.T) {
	root := t.TempDir()
	writeGoMod(t, root, "example.com/myapp")

	got, err := goModuleImportPath(root)
	if err != nil {
		t.Fatalf("goModuleImportPath: %v", err)
	}
	if got != "example.com/myapp" {
		t.Fatalf("goModuleImportPath(root) = %q, want %q", got, "example.com/myapp")
	}
}

func TestGoModuleImportPath_Subdirectory(t *testing.T) {
	root := t.TempDir()
	writeGoMod(t, root, "example.com/myapp")
	sub := filepath.Join(root, "internal", "models")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	got, err := goModuleImportPath(sub)
	if err != nil {
		t.Fatalf("goModuleImportPath: %v", err)
	}
	if want := "example.com/myapp/internal/models"; got != want {
		t.Fatalf("goModuleImportPath(sub) = %q, want %q", got, want)
	}
}

func TestGoModuleImportPath_NoGoModFound(t *testing.T) {
	dir := t.TempDir()
	if _, err := goModuleImportPath(dir); err == nil {
		t.Fatal("expected an error when no go.mod exists above dir")
	}
}

func TestCommand_Config_HasCodegenReposAction(t *testing.T) {
	c := &Command{Command: cmd.NewCommand()}
	conf := c.Config()
	if _, ok := conf.Actions["codegen-repos"]; !ok {
		t.Fatal(`expected a "codegen-repos" action`)
	}
}

func TestCodegenRepos_NoModelManagerRegisteredPrintsInsteadOfErroring(t *testing.T) {
	app := newTestAppNoModelManager(t)
	c := &Command{Command: cmd.NewCommand(), app: app}

	if err := codegenRepos(c); err != nil {
		t.Fatalf("codegenRepos: %v", err)
	}
}

// newRepoTestApp builds an app with one Target - schemaDir/modelsDir/
// reposDir (modelsDir/reposDir may be empty, meaning "not configured").
func newRepoTestApp(t *testing.T, schemaDir, modelsDir, reposDir string) *ModelManager {
	t.Helper()
	target := kernel.Dict{"Schemas": schemaDir}
	if modelsDir != "" {
		target["Models"] = modelsDir
	}
	if reposDir != "" {
		target["Repos"] = reposDir
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

func TestCodegenRepos_ScaffoldsSamePackage(t *testing.T) {
	schemaDir := t.TempDir()
	modelsDir := filepath.Join(t.TempDir(), "models")
	saveSchema(t, schemaDir, &ModelSchema{Name: "Widget"})

	mm := newRepoTestApp(t, schemaDir, modelsDir, modelsDir)
	c := &Command{Command: cmd.NewCommand(), app: mm.App()}

	if err := codegenRepos(c); err != nil {
		t.Fatalf("codegenRepos: %v", err)
	}

	outPath := filepath.Join(modelsDir, "widget_repo.go")
	content, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("expected %s to exist: %v", outPath, err)
	}
	if !strings.Contains(string(content), "*query.BaseRepo[Widget]") {
		t.Fatalf("generated repo doesn't reference Widget as a bare identifier (same package), got:\n%s", content)
	}
	if strings.Contains(string(content), "DO NOT EDIT") {
		t.Fatalf("scaffolded repo should not carry a DO NOT EDIT banner, got:\n%s", content)
	}
}

func TestCodegenRepos_ScaffoldsDifferentPackage(t *testing.T) {
	root := t.TempDir()
	writeGoMod(t, root, "example.com/myapp")
	schemaDir := filepath.Join(root, "schemas")
	modelsDir := filepath.Join(root, "models")
	reposDir := filepath.Join(root, "repos")
	if err := os.MkdirAll(schemaDir, 0755); err != nil {
		t.Fatalf("MkdirAll schemaDir: %v", err)
	}
	saveSchema(t, schemaDir, &ModelSchema{Name: "Widget"})

	mm := newRepoTestApp(t, schemaDir, modelsDir, reposDir)
	c := &Command{Command: cmd.NewCommand(), app: mm.App()}

	if err := codegenRepos(c); err != nil {
		t.Fatalf("codegenRepos: %v", err)
	}

	outPath := filepath.Join(reposDir, "widget_repo.go")
	content, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("expected %s to exist: %v", outPath, err)
	}
	if !strings.Contains(string(content), `models "example.com/myapp/models"`) {
		t.Fatalf("expected an explicit import of the models package by its real module import path, got:\n%s", content)
	}
	if !strings.Contains(string(content), "*query.BaseRepo[models.Widget]") {
		t.Fatalf("expected a models.Widget reference, got:\n%s", content)
	}
}

// TestCodegenRepos_SkipsExistingFile is the core invariant this whole
// action exists to guarantee: a repository file is scaffolded once and
// never touched again, however much the schema changes afterward.
func TestCodegenRepos_SkipsExistingFile(t *testing.T) {
	schemaDir := t.TempDir()
	modelsDir := filepath.Join(t.TempDir(), "models")
	saveSchema(t, schemaDir, &ModelSchema{Name: "Widget"})

	mm := newRepoTestApp(t, schemaDir, modelsDir, modelsDir)
	c := &Command{Command: cmd.NewCommand(), app: mm.App()}

	outPath := filepath.Join(modelsDir, "widget_repo.go")
	if err := os.MkdirAll(modelsDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	const handWritten = "package models\n\n// hand-written, never touch me\n"
	if err := os.WriteFile(outPath, []byte(handWritten), 0644); err != nil {
		t.Fatalf("seed existing repo file: %v", err)
	}

	if err := codegenRepos(c); err != nil {
		t.Fatalf("codegenRepos: %v", err)
	}

	content, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(content) != handWritten {
		t.Fatalf("existing repo file was overwritten - got:\n%s\nwant unchanged:\n%s", content, handWritten)
	}
}

func TestCodegenRepos_ReposWithoutModelsErrors(t *testing.T) {
	schemaDir := t.TempDir()
	reposDir := t.TempDir()
	saveSchema(t, schemaDir, &ModelSchema{Name: "Widget"})

	mm := newRepoTestApp(t, schemaDir, "", reposDir)
	c := &Command{Command: cmd.NewCommand(), app: mm.App()}

	if err := codegenRepos(c); err != nil {
		t.Fatalf("codegenRepos: %v", err)
	}
	if _, err := os.Stat(filepath.Join(reposDir, "widget_repo.go")); !os.IsNotExist(err) {
		t.Fatalf("expected no repo file to be written when Models isn't configured, stat err = %v", err)
	}
}

func TestCodegenRepos_TargetWithoutReposIsSkipped(t *testing.T) {
	schemaDir := t.TempDir()
	modelsDir := filepath.Join(t.TempDir(), "models")
	saveSchema(t, schemaDir, &ModelSchema{Name: "Widget"})

	mm := newRepoTestApp(t, schemaDir, modelsDir, "")
	c := &Command{Command: cmd.NewCommand(), app: mm.App()}

	if err := codegenRepos(c); err != nil {
		t.Fatalf("codegenRepos: %v", err)
	}
}

func TestCodegenStatus_ReportsRepoScaffoldState(t *testing.T) {
	schemaDir := t.TempDir()
	modelsDir := filepath.Join(t.TempDir(), "models")
	saveSchema(t, schemaDir, &ModelSchema{Name: "Widget"})

	mm := newRepoTestApp(t, schemaDir, modelsDir, modelsDir)
	c := &Command{Command: cmd.NewCommand(), app: mm.App()}

	if err := codegenGenerate(c); err != nil {
		t.Fatalf("codegenGenerate: %v", err)
	}
	// Before scaffolding the repo: codegenStatus must not panic/error
	// while reporting "not scaffolded".
	if err := codegenStatus(c); err != nil {
		t.Fatalf("codegenStatus (before repo scaffold): %v", err)
	}

	if err := codegenRepos(c); err != nil {
		t.Fatalf("codegenRepos: %v", err)
	}
	// After scaffolding: same, now reporting "scaffolded".
	if err := codegenStatus(c); err != nil {
		t.Fatalf("codegenStatus (after repo scaffold): %v", err)
	}
}
