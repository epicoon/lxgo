package model

import (
	"strings"
	"testing"
)

func TestBuildRepoCode_DefaultBaseRepo_SamePackage(t *testing.T) {
	schema := &ModelSchema{Name: "Widget"}
	out, err := BuildRepoCode("repos", schema, "")
	if err != nil {
		t.Fatalf("BuildRepoCode: %v", err)
	}
	src := normalizeSpace(string(out))
	for _, want := range []string{
		"package repos",
		`query "github.com/epicoon/lxgo/query"`,
		`gorm "gorm.io/gorm"`,
		"type WidgetRepo struct {",
		"*query.BaseRepo[Widget]",
		"func NewWidgetRepo(db *gorm.DB) *WidgetRepo {",
		"return &WidgetRepo{BaseRepo: query.NewBaseRepo[Widget](db, nil)}",
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated code missing %q, got:\n%s", want, out)
		}
	}
	if strings.Contains(src, "models") {
		t.Errorf("same-package case should not import/reference a models package, got:\n%s", out)
	}
}

func TestBuildRepoCode_DefaultBaseRepo_DifferentPackage(t *testing.T) {
	schema := &ModelSchema{Name: "Widget"}
	out, err := BuildRepoCode("repos", schema, "myapp/models")
	if err != nil {
		t.Fatalf("BuildRepoCode: %v", err)
	}
	src := normalizeSpace(string(out))
	for _, want := range []string{
		`models "myapp/models"`,
		"*query.BaseRepo[models.Widget]",
		"return &WidgetRepo{BaseRepo: query.NewBaseRepo[models.Widget](db, nil)}",
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated code missing %q, got:\n%s", want, out)
		}
	}
}

func TestBuildRepoCode_CustomBaseRepo(t *testing.T) {
	schema := &ModelSchema{Name: "Gadget", BaseRepo: "github.com/some/pkg.MyRepo"}
	out, err := BuildRepoCode("repos", schema, "")
	if err != nil {
		t.Fatalf("BuildRepoCode: %v", err)
	}
	src := normalizeSpace(string(out))
	for _, want := range []string{
		`pkg "github.com/some/pkg"`,
		"type GadgetRepo struct {",
		"*pkg.MyRepo[Gadget]",
		"func NewGadgetRepo(db *gorm.DB) *GadgetRepo {",
		"return &GadgetRepo{MyRepo: pkg.NewMyRepo[Gadget](db, nil)}",
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated code missing %q, got:\n%s", want, out)
		}
	}
}

func TestBuildRepoCode_ResolvedBaseRepoWins(t *testing.T) {
	schema := &ModelSchema{Name: "Widget", ResolvedBaseRepo: "github.com/cascaded/pkg.CascadedRepo"}
	out, err := BuildRepoCode("repos", schema, "")
	if err != nil {
		t.Fatalf("BuildRepoCode: %v", err)
	}
	if !strings.Contains(string(out), "CascadedRepo") {
		t.Errorf("expected the cascade-resolved BaseRepo to be used, got:\n%s", out)
	}
}

func TestBuildRepoCode_InvalidBaseRepo(t *testing.T) {
	schema := &ModelSchema{Name: "Widget", BaseRepo: "NoDotHere"}
	if _, err := BuildRepoCode("repos", schema, ""); err == nil {
		t.Fatal("expected an error for an invalid BaseRepo string")
	}
}

func TestBuildRepoCode_NoBanner(t *testing.T) {
	schema := &ModelSchema{Name: "Widget"}
	out, err := BuildRepoCode("repos", schema, "")
	if err != nil {
		t.Fatalf("BuildRepoCode: %v", err)
	}
	if strings.Contains(string(out), "DO NOT EDIT") {
		t.Errorf("a repository file is meant to be hand-edited - should not carry a DO NOT EDIT banner, got:\n%s", out)
	}
}

func TestRepoCodeFileName(t *testing.T) {
	tests := map[string]string{
		"Widget":     "widget_repo.go",
		"WidgetCopy": "widget_copy_repo.go",
	}
	for name, want := range tests {
		if got := RepoCodeFileName(name); got != want {
			t.Errorf("RepoCodeFileName(%q) = %q, want %q", name, got, want)
		}
	}
}
