package model

import (
	"regexp"
	"strings"
	"testing"
)

// normalizeSpace collapses runs of whitespace to a single space - gofmt
// column-aligns struct field tags with extra spaces depending on what
// else is in the same block, which a plain substring check would
// otherwise have to know about.
var spaceRun = regexp.MustCompile(`[ \t]+`)

func normalizeSpace(s string) string {
	return spaceRun.ReplaceAllString(s, " ")
}

func TestParseGoTypeRef(t *testing.T) {
	tests := []struct {
		name    string
		s       string
		want    goTypeRef
		wantErr bool
	}{
		{"simple", "gorm.io/gorm.Model", goTypeRef{importPath: "gorm.io/gorm", typeName: "Model"}, false},
		{"deep path", "github.com/epicoon/lxgo/query.BaseModel", goTypeRef{importPath: "github.com/epicoon/lxgo/query", typeName: "BaseModel"}, false},
		{"no dot", "NoDotHere", goTypeRef{}, true},
		{"trailing dot", "gorm.io/gorm.", goTypeRef{}, true},
		{"leading dot", ".Model", goTypeRef{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseGoTypeRef("BaseModel", tt.s)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseGoTypeRef(%q) error = %v, wantErr %v", tt.s, err, tt.wantErr)
			}
			if err == nil && got != tt.want {
				t.Fatalf("parseGoTypeRef(%q) = %+v, want %+v", tt.s, got, tt.want)
			}
		})
	}
}

func TestPackageAlias(t *testing.T) {
	tests := map[string]string{
		"gorm.io/gorm":                  "gorm",
		"github.com/epicoon/lxgo/query": "query",
		"github.com/shopspring/decimal": "decimal",
		"noslash":                       "noslash",
	}
	for path, want := range tests {
		if got := packageAlias(path); got != want {
			t.Errorf("packageAlias(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestBuildModelCode_NoBaseModel(t *testing.T) {
	schema := &ModelSchema{
		Name: "Widget",
		Fields: []NamedField{
			{Name: "Name", Field: Field{Type: FieldTypeString, Size: 255, Required: true}},
		},
	}
	out, err := BuildModelCode("models", schema)
	if err != nil {
		t.Fatalf("BuildModelCode: %v", err)
	}
	src := string(out)

	for _, want := range []string{
		"package models",
		"type Widget struct {",
		"ID uint `gorm:\"column:id;primaryKey\"`",
		"Name string `gorm:\"column:name;not null\"`",
		"func (Widget) TableName() string {",
		`return "widgets"`,
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated code missing %q, got:\n%s", want, src)
		}
	}
}

func TestBuildModelCode_BaseModelEmbed(t *testing.T) {
	schema := &ModelSchema{Name: "Widget", BaseModel: "github.com/epicoon/lxgo/query.BaseModel"}
	out, err := BuildModelCode("models", schema)
	if err != nil {
		t.Fatalf("BuildModelCode: %v", err)
	}
	src := string(out)

	if !strings.Contains(src, `query "github.com/epicoon/lxgo/query"`) {
		t.Errorf("missing explicit aliased import, got:\n%s", src)
	}
	if !strings.Contains(src, "query.BaseModel") {
		t.Errorf("missing embed, got:\n%s", src)
	}
	if strings.Contains(src, "ID uint") {
		t.Errorf("bare ID field should not be generated when BaseModel is set, got:\n%s", src)
	}
}

func TestBuildModelCode_NoBaseModelWithTimestamps(t *testing.T) {
	tru := true
	schema := &ModelSchema{Name: "Widget", Timestamps: &tru}
	out, err := BuildModelCode("models", schema)
	if err != nil {
		t.Fatalf("BuildModelCode: %v", err)
	}
	src := normalizeSpace(string(out))
	for _, want := range []string{
		"ID uint `gorm:\"column:id;primaryKey\"`",
		"CreatedAt time.Time `gorm:\"column:created_at\"`",
		"UpdatedAt time.Time `gorm:\"column:updated_at\"`",
		"DeletedAt gorm.DeletedAt `gorm:\"column:deleted_at;index\"`",
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated code missing %q, got:\n%s", want, out)
		}
	}
}

func TestBuildModelCode_KnownBaseModelSkipsTimestampFields(t *testing.T) {
	tru := true
	schema := &ModelSchema{Name: "Widget", BaseModel: "gorm.io/gorm.Model", Timestamps: &tru}
	out, err := BuildModelCode("models", schema)
	if err != nil {
		t.Fatalf("BuildModelCode: %v", err)
	}
	src := string(out)
	if strings.Contains(src, "CreatedAt") || strings.Contains(src, "UpdatedAt") || strings.Contains(src, "DeletedAt") {
		t.Errorf("gorm.Model already carries Timestamps fields, generator should not duplicate them, got:\n%s", src)
	}
}

func TestBuildModelCode_UnknownBaseModelGetsTimestampFields(t *testing.T) {
	tru := true
	schema := &ModelSchema{Name: "Widget", BaseModel: "github.com/some/pkg.CustomBase", Timestamps: &tru}
	out, err := BuildModelCode("models", schema)
	if err != nil {
		t.Fatalf("BuildModelCode: %v", err)
	}
	src := normalizeSpace(string(out))
	for _, want := range []string{
		"CreatedAt time.Time `gorm:\"column:created_at\"`",
		"UpdatedAt time.Time `gorm:\"column:updated_at\"`",
		"DeletedAt gorm.DeletedAt `gorm:\"column:deleted_at;index\"`",
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated code missing %q, got:\n%s", want, out)
		}
	}
}

func TestBuildModelCode_NoTimestampsNoFields(t *testing.T) {
	schema := &ModelSchema{Name: "Widget"}
	out, err := BuildModelCode("models", schema)
	if err != nil {
		t.Fatalf("BuildModelCode: %v", err)
	}
	src := string(out)
	if strings.Contains(src, "CreatedAt") {
		t.Errorf("Timestamps is off, should not generate CreatedAt, got:\n%s", src)
	}
}

func TestBuildModelCode_InvalidBaseModel(t *testing.T) {
	schema := &ModelSchema{Name: "Widget", BaseModel: "NoDotHere"}
	if _, err := BuildModelCode("models", schema); err == nil {
		t.Fatal("expected an error for an invalid BaseModel string")
	}
}

func TestBuildModelCode_UnknownFieldType(t *testing.T) {
	schema := &ModelSchema{
		Name:   "Widget",
		Fields: []NamedField{{Name: "X", Field: Field{Type: FieldType("bogus")}}},
	}
	if _, err := BuildModelCode("models", schema); err == nil {
		t.Fatal("expected an error for an unknown field type")
	}
}

func TestBuildModelCode_ManyToOneRelation(t *testing.T) {
	schema := &ModelSchema{
		Name: "Post",
		Relations: []NamedRelation{
			{Name: "Author", Relation: Relation{Type: RelationTypeManyToOne, RelatedModel: "User"}},
		},
	}
	out, err := BuildModelCode("models", schema)
	if err != nil {
		t.Fatalf("BuildModelCode: %v", err)
	}
	src := normalizeSpace(string(out))
	for _, want := range []string{
		"AuthorID uint `gorm:\"column:author_id;not null\"`",
		"Author User `gorm:\"foreignKey:AuthorID;references:ID\"`",
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated code missing %q, got:\n%s", want, out)
		}
	}
	if strings.Contains(src, "unique") {
		t.Errorf("ManyToOne's FK column must not be unique, got:\n%s", out)
	}
}

func TestBuildModelCode_OneToOneNonHolder_RequiresRelatedAttribute(t *testing.T) {
	schema := &ModelSchema{
		Name: "WidgetCopy",
		Relations: []NamedRelation{
			{Name: "Orig", Relation: Relation{Type: RelationTypeOneToOne, RelatedModel: "Widget"}},
		},
	}
	if _, err := BuildModelCode("models", schema); err == nil {
		t.Fatal("expected an error for a uni non-FK-holding OneToOne relation")
	}
}

func TestBuildModelCode_SkipsOneToMany(t *testing.T) {
	schema := &ModelSchema{
		Name: "User",
		Relations: []NamedRelation{
			{Name: "Posts", Relation: Relation{Type: RelationTypeOneToMany, RelatedModel: "Post", RelatedAttribute: "Author"}},
		},
	}
	out, err := BuildModelCode("models", schema)
	if err != nil {
		t.Fatalf("BuildModelCode: %v", err)
	}
	if strings.Contains(string(out), "Posts") {
		t.Errorf("OneToMany's reverse side has no codegen of its own, should not appear, got:\n%s", out)
	}
}

func TestBuildModelCode_ManyToManyRelation(t *testing.T) {
	schema := &ModelSchema{
		Name: "User",
		Relations: []NamedRelation{
			{Name: "Tags", Relation: Relation{Type: RelationTypeManyToMany, RelatedModel: "Tag", RelatedAttribute: "Users"}},
		},
	}
	out, err := BuildModelCode("models", schema)
	if err != nil {
		t.Fatalf("BuildModelCode: %v", err)
	}
	src := normalizeSpace(string(out))
	want := "Tags []Tag `gorm:\"many2many:" + pgManyToManyTableName("User", "Tags", "Tag", "Users") +
		";joinForeignKey:" + pgJoinColumnName("User", "Tags") +
		";joinReferences:" + pgJoinColumnName("Tag", "Users") + "\"`"
	if !strings.Contains(src, want) {
		t.Errorf("generated code missing %q, got:\n%s", want, out)
	}
}

// TestBuildModelCode_SelfReferentialManyToMany checks that two
// many-to-many relations on the same model, to itself, get two distinct
// fields with two distinct physical join columns - both derivable purely
// from each relation's own declaration, without looking at the related
// schema (there being only one schema, User, in this scenario - if this
// needed cross-schema lookup it would fail) - but the SAME physical join
// table, referenced identically from both sides. Regression test:
// pgManyToManyTableName only sorted by model name to make its result
// order-independent, which is a no-op for a self-referential relation
// (the model is the same on both sides) - the two fields ended up
// pointing at two different, made-up join table names, neither of which
// is the one the real DDL (execAddManyToManyRelation, only ever run from
// the "acting" side, see canIgnoreRelation) actually creates. Fixed by
// also sorting by attribute name when the two models are equal - the
// literal expected table name below is computed by hand, not by calling
// the function under test, so a regression back to the old behavior
// still fails this test even if both fields agreed with each other.
func TestBuildModelCode_SelfReferentialManyToMany(t *testing.T) {
	schema := &ModelSchema{
		Name: "User",
		Relations: []NamedRelation{
			{Name: "Friends", Relation: Relation{Type: RelationTypeManyToMany, RelatedModel: "User", RelatedAttribute: "FriendOf"}},
			{Name: "FriendOf", Relation: Relation{Type: RelationTypeManyToMany, RelatedModel: "User", RelatedAttribute: "Friends"}},
		},
	}
	out, err := BuildModelCode("models", schema)
	if err != nil {
		t.Fatalf("BuildModelCode: %v", err)
	}
	src := normalizeSpace(string(out))

	const wantJoinTable = "rel__users__friend_of__users__friends"
	for _, want := range []string{
		"Friends []User `gorm:\"many2many:" + wantJoinTable +
			";joinForeignKey:users_friends_id;joinReferences:users_friend_of_id\"`",
		"FriendOf []User `gorm:\"many2many:" + wantJoinTable +
			";joinForeignKey:users_friend_of_id;joinReferences:users_friends_id\"`",
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated code missing %q, got:\n%s", want, out)
		}
	}
}

func TestBuildModelCode_NamespaceQualifiesTableName(t *testing.T) {
	schema := &ModelSchema{Name: "Widget", Namespace: "part"}
	out, err := BuildModelCode("models", schema)
	if err != nil {
		t.Fatalf("BuildModelCode: %v", err)
	}
	if !strings.Contains(string(out), `return "part.widgets"`) {
		t.Errorf("expected schema-qualified table name, got:\n%s", out)
	}
}

func TestBuildModelCode_DictFieldType(t *testing.T) {
	schema := &ModelSchema{
		Name:   "Widget",
		Fields: []NamedField{{Name: "Meta", Field: Field{Type: FieldTypeDict}}},
	}
	out, err := BuildModelCode("models", schema)
	if err != nil {
		t.Fatalf("BuildModelCode: %v", err)
	}
	src := string(out)
	if !strings.Contains(src, `datatypes "gorm.io/datatypes"`) {
		t.Errorf("missing datatypes import, got:\n%s", src)
	}
	if !strings.Contains(src, "Meta datatypes.JSON") {
		t.Errorf("missing Meta field, got:\n%s", src)
	}
}

func TestBuildModelCode_DefaultValueTag(t *testing.T) {
	schema := &ModelSchema{
		Name: "Widget",
		Fields: []NamedField{
			{Name: "Price", Field: Field{Type: FieldTypeInt, Default: int64(0)}},
		},
	}
	out, err := BuildModelCode("models", schema)
	if err != nil {
		t.Fatalf("BuildModelCode: %v", err)
	}
	if !strings.Contains(string(out), `Price int64 `+"`"+`gorm:"column:price;default:0"`+"`") {
		t.Errorf("missing default tag, got:\n%s", out)
	}
}

// TestBuildModelCode_DefaultValueRejectsSemicolon checks that a string
// default containing ";" is rejected rather than silently corrupting the
// generated gorm tag - gorm's own tag parser splits on ";" to find each
// setting, so an unescaped one in the default's own text would truncate
// it and inject whatever follows as an unrelated setting.
func TestBuildModelCode_DefaultValueRejectsSemicolon(t *testing.T) {
	schema := &ModelSchema{
		Name: "Widget",
		Fields: []NamedField{
			{Name: "Status", Field: Field{Type: FieldTypeString, Size: 20, Default: "a;unique"}},
		},
	}
	if _, err := BuildModelCode("models", schema); err == nil {
		t.Fatal("expected an error for a default value containing \";\"")
	}
}

func TestBuildModelCode_FieldNamedIDCollidesWithBareID(t *testing.T) {
	schema := &ModelSchema{
		Name:   "Widget",
		Fields: []NamedField{{Name: "ID", Field: Field{Type: FieldTypeInt}}},
	}
	if _, err := BuildModelCode("models", schema); err == nil {
		t.Fatal("expected an error - a Field named ID collides with the bare ID field this generator adds")
	}
}

func TestBuildModelCode_FieldAndRelationNameCollide(t *testing.T) {
	schema := &ModelSchema{
		Name:   "Widget",
		Fields: []NamedField{{Name: "Copy", Field: Field{Type: FieldTypeString, Size: 10}}},
		Relations: []NamedRelation{
			{Name: "Copy", Relation: Relation{Type: RelationTypeManyToOne, RelatedModel: "WidgetCopy"}},
		},
	}
	if _, err := BuildModelCode("models", schema); err == nil {
		t.Fatal("expected an error - a Field and a Relation sharing a name collide in the generated struct")
	}
}

func TestModelCodeFileName(t *testing.T) {
	tests := map[string]string{
		"Widget":     "widget_gen.go",
		"WidgetCopy": "widget_copy_gen.go",
	}
	for name, want := range tests {
		if got := ModelCodeFileName(name); got != want {
			t.Errorf("ModelCodeFileName(%q) = %q, want %q", name, got, want)
		}
	}
}
