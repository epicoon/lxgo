package model

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCodegenPlan_MissingModelIsStale(t *testing.T) {
	schemaDir := t.TempDir()
	modelsDir := filepath.Join(t.TempDir(), "models")
	saveSchema(t, schemaDir, &ModelSchema{Name: "Widget"})

	mm := newRepoTestApp(t, schemaDir, modelsDir, "")
	schemas, err := mm.LoadModelSchemas()
	if err != nil {
		t.Fatalf("LoadModelSchemas: %v", err)
	}

	staleModels, missingRepos := mm.codegenPlan(schemas)
	if len(staleModels) != 1 || staleModels[0] != "Widget" {
		t.Fatalf("staleModels = %v, want [Widget]", staleModels)
	}
	if len(missingRepos) != 0 {
		t.Fatalf("missingRepos = %v, want none (Repos not configured)", missingRepos)
	}
}

func TestCodegenPlan_UpToDateModelIsNotStale(t *testing.T) {
	schemaDir := t.TempDir()
	modelsDir := t.TempDir()
	saveSchema(t, schemaDir, &ModelSchema{Name: "Widget"})
	schemaPath := filepath.Join(schemaDir, "Widget.yaml")

	mm := newRepoTestApp(t, schemaDir, modelsDir, "")
	schemas, err := mm.LoadModelSchemas()
	if err != nil {
		t.Fatalf("LoadModelSchemas: %v", err)
	}

	outPath := filepath.Join(modelsDir, ModelCodeFileName("Widget"))
	if err := os.WriteFile(outPath, []byte("package models\n"), 0644); err != nil {
		t.Fatalf("write generated file: %v", err)
	}
	future := time.Now().Add(time.Hour)
	if err := os.Chtimes(outPath, future, future); err != nil {
		t.Fatalf("Chtimes: %v", err)
	}
	// Sanity: the generated file must actually be newer than the schema
	// file for this to test the "not stale" branch, not just happen to
	// pass because both share a timestamp.
	if info, _ := os.Stat(schemaPath); info != nil && !info.ModTime().Before(future) {
		t.Fatal("schema file isn't older than the generated file - test setup is broken")
	}

	staleModels, _ := mm.codegenPlan(schemas)
	if len(staleModels) != 0 {
		t.Fatalf("staleModels = %v, want none (generated file is newer than the schema)", staleModels)
	}
}

func TestCodegenPlan_MissingRepoIsReported(t *testing.T) {
	schemaDir := t.TempDir()
	modelsDir := t.TempDir()
	saveSchema(t, schemaDir, &ModelSchema{Name: "Widget"})

	mm := newRepoTestApp(t, schemaDir, modelsDir, modelsDir)
	schemas, err := mm.LoadModelSchemas()
	if err != nil {
		t.Fatalf("LoadModelSchemas: %v", err)
	}

	outPath := filepath.Join(modelsDir, ModelCodeFileName("Widget"))
	if err := os.WriteFile(outPath, []byte("package models\n"), 0644); err != nil {
		t.Fatalf("write generated file: %v", err)
	}
	future := time.Now().Add(time.Hour)
	os.Chtimes(outPath, future, future)

	staleModels, missingRepos := mm.codegenPlan(schemas)
	if len(staleModels) != 0 {
		t.Fatalf("staleModels = %v, want none", staleModels)
	}
	if len(missingRepos) != 1 || missingRepos[0] != "Widget" {
		t.Fatalf("missingRepos = %v, want [Widget]", missingRepos)
	}
}

func TestCodegenPlan_ScaffoldedRepoIsNotReported(t *testing.T) {
	schemaDir := t.TempDir()
	modelsDir := t.TempDir()
	saveSchema(t, schemaDir, &ModelSchema{Name: "Widget"})

	mm := newRepoTestApp(t, schemaDir, modelsDir, modelsDir)
	schemas, err := mm.LoadModelSchemas()
	if err != nil {
		t.Fatalf("LoadModelSchemas: %v", err)
	}

	if err := os.WriteFile(filepath.Join(modelsDir, ModelCodeFileName("Widget")), []byte("package models\n"), 0644); err != nil {
		t.Fatalf("write generated file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(modelsDir, RepoCodeFileName("Widget")), []byte("package models\n"), 0644); err != nil {
		t.Fatalf("write repo file: %v", err)
	}

	_, missingRepos := mm.codegenPlan(schemas)
	if len(missingRepos) != 0 {
		t.Fatalf("missingRepos = %v, want none (already scaffolded)", missingRepos)
	}
}

func TestCodegenPlan_TargetWithoutModelsIsSkipped(t *testing.T) {
	schemaDir := t.TempDir()
	saveSchema(t, schemaDir, &ModelSchema{Name: "Widget"})

	mm := newRepoTestApp(t, schemaDir, "", "")
	schemas, err := mm.LoadModelSchemas()
	if err != nil {
		t.Fatalf("LoadModelSchemas: %v", err)
	}

	staleModels, missingRepos := mm.codegenPlan(schemas)
	if len(staleModels) != 0 || len(missingRepos) != 0 {
		t.Fatalf("staleModels = %v, missingRepos = %v, want both empty", staleModels, missingRepos)
	}
}
