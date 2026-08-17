package model

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestLoadModelSchema_MapAndCompactFields(t *testing.T) {
	src := `Name: GameSave
Fields:
  gameType: string required
  data:
    Type: string
    Size: 4000
    Required: true
`
	schema, err := LoadModelSchema("GameSave", []byte(src))
	if err != nil {
		t.Fatalf("LoadModelSchema: %v", err)
	}
	if schema.Name != "GameSave" {
		t.Fatalf("Name = %q, want %q", schema.Name, "GameSave")
	}
	if len(schema.Fields) != 2 {
		t.Fatalf("len(Fields) = %d, want 2", len(schema.Fields))
	}
	if schema.Fields[0].Name != "gameType" || schema.Fields[1].Name != "data" {
		t.Fatalf("field order = %q, %q - want file order preserved", schema.Fields[0].Name, schema.Fields[1].Name)
	}
	f, ok := schema.FieldByName("data")
	if !ok || f.Size != 4000 {
		t.Fatalf("FieldByName(\"data\") = %#v, %v", f, ok)
	}
}

func TestLoadModelSchema_NameKeyIsInformationalOnly(t *testing.T) {
	// The caller-supplied name (typically the filename) wins even if the
	// file's own "Name:" says something else - it's decorative, not
	// authoritative.
	schema, err := LoadModelSchema("ActualName", []byte("Name: SomethingElse\nFields:\n  x: int\n"))
	if err != nil {
		t.Fatalf("LoadModelSchema: %v", err)
	}
	if schema.Name != "ActualName" {
		t.Fatalf("Name = %q, want %q", schema.Name, "ActualName")
	}
}

func TestLoadModelSchema_NoFieldsKey(t *testing.T) {
	schema, err := LoadModelSchema("Empty", []byte("Name: Empty\n"))
	if err != nil {
		t.Fatalf("LoadModelSchema: %v", err)
	}
	if len(schema.Fields) != 0 {
		t.Fatalf("Fields = %#v, want empty", schema.Fields)
	}
}

func TestLoadModelSchema_EmptyFile(t *testing.T) {
	schema, err := LoadModelSchema("Empty", []byte(""))
	if err != nil {
		t.Fatalf("LoadModelSchema: %v", err)
	}
	if schema.Name != "Empty" || len(schema.Fields) != 0 {
		t.Fatalf("schema = %#v", schema)
	}
}

func TestLoadModelSchema_DuplicateTopLevelKeyRejected(t *testing.T) {
	if _, err := LoadModelSchema("X", []byte("Fields:\n  x: int\nFields:\n  y: int\n")); err == nil {
		t.Fatal("expected an error for a duplicate top-level key")
	}
}

func TestLoadModelSchema_ErrorMessagesNameKindNotBareNumber(t *testing.T) {
	_, err := LoadModelSchema("X", []byte("Fields:\n"))
	if err == nil {
		t.Fatal("expected an error: Fields: with no value is not a mapping")
	}
	if strings.Contains(err.Error(), "got 8") || !strings.Contains(err.Error(), "got a scalar") {
		t.Fatalf("error = %q, want a readable kind name (\"a scalar\"), not a bare yaml.Kind number", err)
	}
}

func TestLoadModelSchema_UnknownTopLevelKeyRejected(t *testing.T) {
	if _, err := LoadModelSchema("X", []byte("Name: X\nFeilds:\n  x: int\n")); err == nil {
		t.Fatal("expected an error for the misspelled top-level key \"Feilds\"")
	}
}

func TestLoadModelSchema_NotAMapping(t *testing.T) {
	if _, err := LoadModelSchema("X", []byte("just a string\n")); err == nil {
		t.Fatal("expected an error: a model schema must be a yaml mapping")
	}
}

func TestLoadModelSchema_FieldsNotAMapping(t *testing.T) {
	if _, err := LoadModelSchema("X", []byte("Fields:\n  - a\n  - b\n")); err == nil {
		t.Fatal("expected an error: Fields must be a yaml mapping")
	}
}

func TestLoadModelSchema_InvalidField(t *testing.T) {
	if _, err := LoadModelSchema("X", []byte("Fields:\n  x: not_a_type\n")); err == nil {
		t.Fatal("expected an error: unknown field type propagates with the field's name in context")
	}
}

func TestLoadModelSchema_DuplicateFieldName(t *testing.T) {
	// A literal yaml duplicate key - gopkg.in/yaml.v3's own map decoding
	// would silently keep only the last one; parsing fields.Content pairs
	// by hand lets this be caught instead.
	if _, err := LoadModelSchema("X", []byte("Fields:\n  x: int\n  x: string\n")); err == nil {
		t.Fatal("expected an error for a duplicate field name")
	}
}

func TestLoadModelSchema_RelationsParsed(t *testing.T) {
	src := "Name: GamerInGame\nFields:\n  gamerId: string required\nRelations:\n  gameSave: (>-) GameSave.gamers\n"
	schema, err := LoadModelSchema("GamerInGame", []byte(src))
	if err != nil {
		t.Fatalf("LoadModelSchema: %v", err)
	}
	r, ok := schema.RelationByName("gameSave")
	if !ok {
		t.Fatal("expected a \"gameSave\" relation")
	}
	if r.Type != RelationTypeManyToOne || r.RelatedModel != "GameSave" || r.RelatedAttribute != "gamers" {
		t.Fatalf("gameSave = %#v", r)
	}
}

func TestLoadModelSchema_NoRelationsKey(t *testing.T) {
	schema, err := LoadModelSchema("X", []byte("Fields:\n  x: int\n"))
	if err != nil {
		t.Fatalf("LoadModelSchema: %v", err)
	}
	if len(schema.Relations) != 0 {
		t.Fatal("expected no relations when the file has no Relations: key")
	}
}

func TestModelSchema_Save_RoundTrip(t *testing.T) {
	src := "Name: GamerInGame\nFields:\n  gamerId: string required\n  isHolder: bool default(false)\nRelations:\n  gameSave: (>-) GameSave.gamers\n"
	original, err := LoadModelSchema("GamerInGame", []byte(src))
	if err != nil {
		t.Fatalf("LoadModelSchema: %v", err)
	}

	path := filepath.Join(t.TempDir(), "GamerInGame.yaml")
	if err := original.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	reloaded, err := LoadModelSchema("GamerInGame", data)
	if err != nil {
		t.Fatalf("LoadModelSchema(saved): %v\nsaved content:\n%s", err, data)
	}

	if len(reloaded.Fields) != len(original.Fields) {
		t.Fatalf("Fields = %#v, want %#v", reloaded.Fields, original.Fields)
	}
	for i, f := range original.Fields {
		if reloaded.Fields[i].Name != f.Name || !reloaded.Fields[i].Field.IsEqual(f.Field) {
			t.Fatalf("field %d = %#v, want %#v", i, reloaded.Fields[i], f)
		}
	}

	if len(reloaded.Relations) != 1 {
		t.Fatal("expected relations to survive Save")
	}
	r, ok := reloaded.RelationByName("gameSave")
	if !ok || r.Type != RelationTypeManyToOne || r.RelatedModel != "GameSave" || r.RelatedAttribute != "gamers" {
		t.Fatalf("gameSave after round-trip = %#v, %v", r, ok)
	}
}

func TestModelSchema_Save_NoRelations(t *testing.T) {
	schema, err := LoadModelSchema("X", []byte("Fields:\n  x: int\n"))
	if err != nil {
		t.Fatalf("LoadModelSchema: %v", err)
	}

	path := filepath.Join(t.TempDir(), "X.yaml")
	if err := schema.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if _, has := raw["Relations"]; has {
		t.Fatalf("expected no Relations: key to be written when there was none, got: %s", data)
	}
}

// loadModelSchemas mirrors what ModelManager.LoadModelSchemas does for a
// single directory (load, then validate relations) - a package-level
// entry point isn't exported for that anymore, only the component-wide
// cascade is, so these tests exercise the same two building blocks
// directly instead of going through a full app component just to load one
// directory.
func loadModelSchemas(dir string) ([]*ModelSchema, error) {
	schemas, err := loadModelSchemaFiles(dir)
	if err != nil {
		return nil, err
	}
	if err := validateRelations(schemas); err != nil {
		return nil, err
	}
	return schemas, nil
}

func TestLoadModelSchemas_Directory(t *testing.T) {
	dir := t.TempDir()
	writeSchemaFile(t, dir, "Zeta.yaml", "Fields:\n  a: int\n")
	writeSchemaFile(t, dir, "Alpha.yaml", "Fields:\n  b: int\n")
	// Non-.yaml files must be ignored.
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("not a schema"), 0644); err != nil {
		t.Fatalf("write README.md: %v", err)
	}

	schemas, err := loadModelSchemas(dir)
	if err != nil {
		t.Fatalf("LoadModelSchemas: %v", err)
	}
	if len(schemas) != 2 {
		t.Fatalf("len(schemas) = %d, want 2", len(schemas))
	}
	if schemas[0].Name != "Alpha" || schemas[1].Name != "Zeta" {
		t.Fatalf("schemas = %q, %q - want sorted by name", schemas[0].Name, schemas[1].Name)
	}
}

func TestLoadModelSchemas_PropagatesFileError(t *testing.T) {
	dir := t.TempDir()
	writeSchemaFile(t, dir, "Bad.yaml", "Fields:\n  x: not_a_type\n")

	if _, err := loadModelSchemas(dir); err == nil {
		t.Fatal("expected an error for an invalid schema file in the directory")
	}
}

func TestLoadModelSchemas_SetsSourceDir(t *testing.T) {
	dir := t.TempDir()
	writeSchemaFile(t, dir, "Widget.yaml", "Fields:\n  a: int\n")

	schemas, err := loadModelSchemas(dir)
	if err != nil {
		t.Fatalf("LoadModelSchemas: %v", err)
	}
	if len(schemas) != 1 || schemas[0].SourceDir != dir {
		t.Fatalf("SourceDir = %q, want %q", schemas[0].SourceDir, dir)
	}
}

func TestLoadModelSchemas_ValidatesRelationsAcrossDirectory(t *testing.T) {
	dir := t.TempDir()
	writeSchemaFile(t, dir, "GameSave.yaml", "Fields:\n  a: int\nRelations:\n  gamers: (-<) GamerInGame.gameSave\n")
	writeSchemaFile(t, dir, "GamerInGame.yaml", "Fields:\n  b: int\nRelations:\n  gameSave: (>-) GameSave.gamers\n")

	schemas, err := loadModelSchemas(dir)
	if err != nil {
		t.Fatalf("LoadModelSchemas: %v", err)
	}
	if len(schemas) != 2 {
		t.Fatalf("len(schemas) = %d, want 2", len(schemas))
	}
}

func TestLoadModelSchemas_BrokenRelationIsError(t *testing.T) {
	dir := t.TempDir()
	writeSchemaFile(t, dir, "GameSave.yaml", "Fields:\n  a: int\nRelations:\n  gamers: (-<) GamerInGame.gameSave\n")
	// GamerInGame doesn't declare the gameSave relation back at all.
	writeSchemaFile(t, dir, "GamerInGame.yaml", "Fields:\n  b: int\n")

	if _, err := loadModelSchemas(dir); err == nil {
		t.Fatal("expected an error: GameSave.gamers has no matching relation on GamerInGame")
	}
}

func TestLoadModelSchema_NamespaceParsed(t *testing.T) {
	schema, err := LoadModelSchema("Widget", []byte("Namespace: part1\nFields:\n  x: int\n"))
	if err != nil {
		t.Fatalf("LoadModelSchema: %v", err)
	}
	if schema.Namespace != "part1" {
		t.Fatalf("Namespace = %q, want %q", schema.Namespace, "part1")
	}
	if schema.ResolvedNamespace != "" {
		t.Fatalf("ResolvedNamespace = %q, want empty (LoadModelSchema has no directory context)", schema.ResolvedNamespace)
	}
}

func TestLoadModelSchema_NoNamespaceKey(t *testing.T) {
	schema, err := LoadModelSchema("Widget", []byte("Fields:\n  x: int\n"))
	if err != nil {
		t.Fatalf("LoadModelSchema: %v", err)
	}
	if schema.Namespace != "" {
		t.Fatalf("Namespace = %q, want empty", schema.Namespace)
	}
}

func TestModelSchema_EffectiveNamespace(t *testing.T) {
	cases := []struct {
		name              string
		namespace         string
		resolvedNamespace string
		want              string
	}{
		{"neither set", "", "", ""},
		{"only Namespace set (no directory context)", "own", "", "own"},
		{"only ResolvedNamespace set (loaded via cascade, no own override)", "", "cascaded", "cascaded"},
		{"both set - ResolvedNamespace wins", "own", "cascaded", "cascaded"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := &ModelSchema{Namespace: c.namespace, ResolvedNamespace: c.resolvedNamespace}
			if got := s.EffectiveNamespace(); got != c.want {
				t.Errorf("EffectiveNamespace() = %q, want %q", got, c.want)
			}
		})
	}
}

func TestModelSchema_Save_NamespaceRoundTrip(t *testing.T) {
	schema, err := LoadModelSchema("Widget", []byte("Namespace: part1\nFields:\n  x: int\n"))
	if err != nil {
		t.Fatalf("LoadModelSchema: %v", err)
	}

	path := filepath.Join(t.TempDir(), "Widget.yaml")
	if err := schema.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	reloaded, err := LoadModelSchema("Widget", data)
	if err != nil {
		t.Fatalf("LoadModelSchema(saved): %v\nsaved content:\n%s", err, data)
	}
	if reloaded.Namespace != "part1" {
		t.Fatalf("Namespace after round-trip = %q, want %q", reloaded.Namespace, "part1")
	}
}

// TestModelSchema_Save_NamespaceOmittedWhenEmpty checks that Save doesn't
// write a Namespace: key at all when it's empty, matching how Relations
// is already handled - a schema that never declared one shouldn't gain
// one just by being loaded and saved back.
func TestModelSchema_Save_NamespaceOmittedWhenEmpty(t *testing.T) {
	schema, err := LoadModelSchema("Widget", []byte("Fields:\n  x: int\n"))
	if err != nil {
		t.Fatalf("LoadModelSchema: %v", err)
	}

	path := filepath.Join(t.TempDir(), "Widget.yaml")
	if err := schema.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if _, has := raw["Namespace"]; has {
		t.Fatalf("expected no Namespace: key to be written when there was none, got: %s", data)
	}
}

// TestModelSchema_Save_NeverWritesResolvedNamespace checks that a
// directory-level default cascaded into ResolvedNamespace (see
// ModelManager.LoadModelSchemas) never gets written back as if it were
// an explicit override - only the model's own Namespace round-trips.
func TestModelSchema_Save_NeverWritesResolvedNamespace(t *testing.T) {
	schema, err := LoadModelSchema("Widget", []byte("Fields:\n  x: int\n"))
	if err != nil {
		t.Fatalf("LoadModelSchema: %v", err)
	}
	schema.ResolvedNamespace = "inherited-default"

	path := filepath.Join(t.TempDir(), "Widget.yaml")
	if err := schema.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if _, has := raw["Namespace"]; has {
		t.Fatalf("expected ResolvedNamespace to never be written back as Namespace:, got: %s", data)
	}
}

func TestLoadModelSchema_BaseModelParsed(t *testing.T) {
	schema, err := LoadModelSchema("Widget", []byte("BaseModel: github.com/epicoon/lxgo/query.BaseModel\nFields:\n  x: int\n"))
	if err != nil {
		t.Fatalf("LoadModelSchema: %v", err)
	}
	if schema.BaseModel != "github.com/epicoon/lxgo/query.BaseModel" {
		t.Fatalf("BaseModel = %q, want %q", schema.BaseModel, "github.com/epicoon/lxgo/query.BaseModel")
	}
	if schema.ResolvedBaseModel != "" {
		t.Fatalf("ResolvedBaseModel = %q, want empty (LoadModelSchema has no directory context)", schema.ResolvedBaseModel)
	}
}

func TestLoadModelSchema_NoBaseModelKey(t *testing.T) {
	schema, err := LoadModelSchema("Widget", []byte("Fields:\n  x: int\n"))
	if err != nil {
		t.Fatalf("LoadModelSchema: %v", err)
	}
	if schema.BaseModel != "" {
		t.Fatalf("BaseModel = %q, want empty", schema.BaseModel)
	}
}

func TestModelSchema_EffectiveBaseModel(t *testing.T) {
	cases := []struct {
		name              string
		baseModel         string
		resolvedBaseModel string
		want              string
	}{
		{"neither set", "", "", ""},
		{"only BaseModel set (no directory context)", "own.Base", "", "own.Base"},
		{"only ResolvedBaseModel set (loaded via cascade, no own override)", "", "cascaded.Base", "cascaded.Base"},
		{"both set - ResolvedBaseModel wins", "own.Base", "cascaded.Base", "cascaded.Base"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := &ModelSchema{BaseModel: c.baseModel, ResolvedBaseModel: c.resolvedBaseModel}
			if got := s.EffectiveBaseModel(); got != c.want {
				t.Errorf("EffectiveBaseModel() = %q, want %q", got, c.want)
			}
		})
	}
}

func TestModelSchema_Save_BaseModelRoundTrip(t *testing.T) {
	schema, err := LoadModelSchema("Widget", []byte("BaseModel: github.com/epicoon/lxgo/query.BaseModel\nFields:\n  x: int\n"))
	if err != nil {
		t.Fatalf("LoadModelSchema: %v", err)
	}

	path := filepath.Join(t.TempDir(), "Widget.yaml")
	if err := schema.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	reloaded, err := LoadModelSchema("Widget", data)
	if err != nil {
		t.Fatalf("LoadModelSchema(saved): %v\nsaved content:\n%s", err, data)
	}
	if reloaded.BaseModel != "github.com/epicoon/lxgo/query.BaseModel" {
		t.Fatalf("BaseModel after round-trip = %q, want %q", reloaded.BaseModel, "github.com/epicoon/lxgo/query.BaseModel")
	}
}

// TestModelSchema_Save_BaseModelOmittedWhenEmpty checks that Save doesn't
// write a BaseModel: key at all when it's empty, matching how Namespace is
// already handled - a schema that never declared one shouldn't gain one
// just by being loaded and saved back.
func TestModelSchema_Save_BaseModelOmittedWhenEmpty(t *testing.T) {
	schema, err := LoadModelSchema("Widget", []byte("Fields:\n  x: int\n"))
	if err != nil {
		t.Fatalf("LoadModelSchema: %v", err)
	}

	path := filepath.Join(t.TempDir(), "Widget.yaml")
	if err := schema.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if _, has := raw["BaseModel"]; has {
		t.Fatalf("expected no BaseModel: key to be written when there was none, got: %s", data)
	}
}

// TestModelSchema_Save_NeverWritesResolvedBaseModel checks that a
// directory-level default cascaded into ResolvedBaseModel (see
// ModelManager.LoadModelSchemas) never gets written back as if it were an
// explicit override - only the model's own BaseModel round-trips.
func TestModelSchema_Save_NeverWritesResolvedBaseModel(t *testing.T) {
	schema, err := LoadModelSchema("Widget", []byte("Fields:\n  x: int\n"))
	if err != nil {
		t.Fatalf("LoadModelSchema: %v", err)
	}
	schema.ResolvedBaseModel = "inherited/default.Base"

	path := filepath.Join(t.TempDir(), "Widget.yaml")
	if err := schema.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if _, has := raw["BaseModel"]; has {
		t.Fatalf("expected ResolvedBaseModel to never be written back as BaseModel:, got: %s", data)
	}
}

func TestLoadModelSchema_BaseRepoParsed(t *testing.T) {
	schema, err := LoadModelSchema("Widget", []byte("BaseRepo: github.com/epicoon/lxgo/query.BaseRepo\nFields:\n  x: int\n"))
	if err != nil {
		t.Fatalf("LoadModelSchema: %v", err)
	}
	if schema.BaseRepo != "github.com/epicoon/lxgo/query.BaseRepo" {
		t.Fatalf("BaseRepo = %q, want %q", schema.BaseRepo, "github.com/epicoon/lxgo/query.BaseRepo")
	}
	if schema.ResolvedBaseRepo != "" {
		t.Fatalf("ResolvedBaseRepo = %q, want empty (LoadModelSchema has no directory context)", schema.ResolvedBaseRepo)
	}
}

func TestLoadModelSchema_NoBaseRepoKey(t *testing.T) {
	schema, err := LoadModelSchema("Widget", []byte("Fields:\n  x: int\n"))
	if err != nil {
		t.Fatalf("LoadModelSchema: %v", err)
	}
	if schema.BaseRepo != "" {
		t.Fatalf("BaseRepo = %q, want empty", schema.BaseRepo)
	}
}

func TestModelSchema_EffectiveBaseRepo(t *testing.T) {
	cases := []struct {
		name             string
		baseRepo         string
		resolvedBaseRepo string
		want             string
	}{
		{"neither set", "", "", ""},
		{"only BaseRepo set (no directory context)", "own.Repo", "", "own.Repo"},
		{"only ResolvedBaseRepo set (loaded via cascade, no own override)", "", "cascaded.Repo", "cascaded.Repo"},
		{"both set - ResolvedBaseRepo wins", "own.Repo", "cascaded.Repo", "cascaded.Repo"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := &ModelSchema{BaseRepo: c.baseRepo, ResolvedBaseRepo: c.resolvedBaseRepo}
			if got := s.EffectiveBaseRepo(); got != c.want {
				t.Errorf("EffectiveBaseRepo() = %q, want %q", got, c.want)
			}
		})
	}
}

func TestModelSchema_Save_BaseRepoRoundTrip(t *testing.T) {
	schema, err := LoadModelSchema("Widget", []byte("BaseRepo: github.com/epicoon/lxgo/query.BaseRepo\nFields:\n  x: int\n"))
	if err != nil {
		t.Fatalf("LoadModelSchema: %v", err)
	}

	path := filepath.Join(t.TempDir(), "Widget.yaml")
	if err := schema.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	reloaded, err := LoadModelSchema("Widget", data)
	if err != nil {
		t.Fatalf("LoadModelSchema(saved): %v\nsaved content:\n%s", err, data)
	}
	if reloaded.BaseRepo != "github.com/epicoon/lxgo/query.BaseRepo" {
		t.Fatalf("BaseRepo after round-trip = %q, want %q", reloaded.BaseRepo, "github.com/epicoon/lxgo/query.BaseRepo")
	}
}

// TestModelSchema_Save_BaseRepoOmittedWhenEmpty checks that Save doesn't
// write a BaseRepo: key at all when it's empty, matching how BaseModel is
// already handled - a schema that never declared one shouldn't gain one
// just by being loaded and saved back.
func TestModelSchema_Save_BaseRepoOmittedWhenEmpty(t *testing.T) {
	schema, err := LoadModelSchema("Widget", []byte("Fields:\n  x: int\n"))
	if err != nil {
		t.Fatalf("LoadModelSchema: %v", err)
	}

	path := filepath.Join(t.TempDir(), "Widget.yaml")
	if err := schema.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if _, has := raw["BaseRepo"]; has {
		t.Fatalf("expected no BaseRepo: key to be written when there was none, got: %s", data)
	}
}

// TestModelSchema_Save_NeverWritesResolvedBaseRepo checks that a
// directory-level default cascaded into ResolvedBaseRepo (see
// ModelManager.LoadModelSchemas) never gets written back as if it were an
// explicit override - only the model's own BaseRepo round-trips.
func TestModelSchema_Save_NeverWritesResolvedBaseRepo(t *testing.T) {
	schema, err := LoadModelSchema("Widget", []byte("Fields:\n  x: int\n"))
	if err != nil {
		t.Fatalf("LoadModelSchema: %v", err)
	}
	schema.ResolvedBaseRepo = "inherited/default.Repo"

	path := filepath.Join(t.TempDir(), "Widget.yaml")
	if err := schema.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if _, has := raw["BaseRepo"]; has {
		t.Fatalf("expected ResolvedBaseRepo to never be written back as BaseRepo:, got: %s", data)
	}
}

func TestLoadModelSchema_TimestampsParsed(t *testing.T) {
	schema, err := LoadModelSchema("Widget", []byte("Timestamps: true\nFields:\n  x: int\n"))
	if err != nil {
		t.Fatalf("LoadModelSchema: %v", err)
	}
	if schema.Timestamps == nil || *schema.Timestamps != true {
		t.Fatalf("Timestamps = %v, want pointer to true", schema.Timestamps)
	}
	if schema.ResolvedTimestamps != nil {
		t.Fatalf("ResolvedTimestamps = %v, want nil (LoadModelSchema has no directory context)", schema.ResolvedTimestamps)
	}
}

// TestLoadModelSchema_TimestampsExplicitFalseParsed checks that an
// explicit `Timestamps: false` is distinguishable from the key being
// absent entirely - both a plain bool zero value, but only the pointer
// stays nil when the key was never there (see ModelSchema.Timestamps).
func TestLoadModelSchema_TimestampsExplicitFalseParsed(t *testing.T) {
	schema, err := LoadModelSchema("Widget", []byte("Timestamps: false\nFields:\n  x: int\n"))
	if err != nil {
		t.Fatalf("LoadModelSchema: %v", err)
	}
	if schema.Timestamps == nil || *schema.Timestamps != false {
		t.Fatalf("Timestamps = %v, want pointer to false", schema.Timestamps)
	}
}

func TestLoadModelSchema_NoTimestampsKey(t *testing.T) {
	schema, err := LoadModelSchema("Widget", []byte("Fields:\n  x: int\n"))
	if err != nil {
		t.Fatalf("LoadModelSchema: %v", err)
	}
	if schema.Timestamps != nil {
		t.Fatalf("Timestamps = %v, want nil", schema.Timestamps)
	}
}

func TestModelSchema_EffectiveTimestamps(t *testing.T) {
	boolPtr := func(b bool) *bool { return &b }

	cases := []struct {
		name               string
		timestamps         *bool
		resolvedTimestamps *bool
		want               bool
	}{
		{"neither set", nil, nil, false},
		{"only Timestamps set (no directory context)", boolPtr(true), nil, true},
		{"only ResolvedTimestamps set (loaded via cascade, no own override)", nil, boolPtr(true), true},
		{"both set - ResolvedTimestamps wins", boolPtr(false), boolPtr(true), true},
		{"both set - ResolvedTimestamps false wins over own true", boolPtr(true), boolPtr(false), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := &ModelSchema{Timestamps: c.timestamps, ResolvedTimestamps: c.resolvedTimestamps}
			if got := s.EffectiveTimestamps(); got != c.want {
				t.Errorf("EffectiveTimestamps() = %v, want %v", got, c.want)
			}
		})
	}
}

func TestModelSchema_Save_TimestampsRoundTrip(t *testing.T) {
	schema, err := LoadModelSchema("Widget", []byte("Timestamps: true\nFields:\n  x: int\n"))
	if err != nil {
		t.Fatalf("LoadModelSchema: %v", err)
	}

	path := filepath.Join(t.TempDir(), "Widget.yaml")
	if err := schema.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	reloaded, err := LoadModelSchema("Widget", data)
	if err != nil {
		t.Fatalf("LoadModelSchema(saved): %v\nsaved content:\n%s", err, data)
	}
	if reloaded.Timestamps == nil || *reloaded.Timestamps != true {
		t.Fatalf("Timestamps after round-trip = %v, want pointer to true", reloaded.Timestamps)
	}
}

// TestModelSchema_Save_TimestampsOmittedWhenNil checks that Save doesn't
// write a Timestamps: key at all when it's nil, matching how Namespace/
// BaseModel are already handled - a schema that never declared one
// shouldn't gain one just by being loaded and saved back.
func TestModelSchema_Save_TimestampsOmittedWhenNil(t *testing.T) {
	schema, err := LoadModelSchema("Widget", []byte("Fields:\n  x: int\n"))
	if err != nil {
		t.Fatalf("LoadModelSchema: %v", err)
	}

	path := filepath.Join(t.TempDir(), "Widget.yaml")
	if err := schema.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if _, has := raw["Timestamps"]; has {
		t.Fatalf("expected no Timestamps: key to be written when there was none, got: %s", data)
	}
}

// TestModelSchema_Save_NeverWritesResolvedTimestamps checks that a
// directory-level default cascaded into ResolvedTimestamps (see
// ModelManager.LoadModelSchemas) never gets written back as if it were an
// explicit override - only the model's own Timestamps round-trips.
func TestModelSchema_Save_NeverWritesResolvedTimestamps(t *testing.T) {
	schema, err := LoadModelSchema("Widget", []byte("Fields:\n  x: int\n"))
	if err != nil {
		t.Fatalf("LoadModelSchema: %v", err)
	}
	resolved := true
	schema.ResolvedTimestamps = &resolved

	path := filepath.Join(t.TempDir(), "Widget.yaml")
	if err := schema.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if _, has := raw["Timestamps"]; has {
		t.Fatalf("expected ResolvedTimestamps to never be written back as Timestamps:, got: %s", data)
	}
}

// TestModelSchema_Save_PreservesPerFieldForm checks that Save writes each
// field back the way it was read - a schema with one compact and one map
// field must not have Save turn the compact one into a map too.
func TestModelSchema_Save_PreservesPerFieldForm(t *testing.T) {
	src := "Fields:\n  gameType: string required\n  data:\n    Type: string\n    Size: 4000\n    Required: true\n"
	schema, err := LoadModelSchema("GameSave", []byte(src))
	if err != nil {
		t.Fatalf("LoadModelSchema: %v", err)
	}

	path := filepath.Join(t.TempDir(), "GameSave.yaml")
	if err := schema.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	got := string(data)
	if !strings.Contains(got, "gameType: string required") {
		t.Fatalf("saved content = %s\nwant \"gameType\" to stay in the compact form", got)
	}
	if !strings.Contains(got, "Type: string") || !strings.Contains(got, "Size: 4000") {
		t.Fatalf("saved content = %s\nwant \"data\" to stay in the map form", got)
	}

	reloaded, err := LoadModelSchema("GameSave", data)
	if err != nil {
		t.Fatalf("LoadModelSchema(saved): %v\nsaved content:\n%s", err, data)
	}
	for i, f := range schema.Fields {
		if !reloaded.Fields[i].IsEqual(f.Field) {
			t.Fatalf("field %q = %#v, want %#v", f.Name, reloaded.Fields[i].Field, f.Field)
		}
	}
}

func writeSchemaFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}
