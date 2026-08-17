package model

import (
	"fmt"
	"go/format"
	"sort"
	"strings"
)

// fieldGoType describes the Go type BuildModelCode renders for one
// FieldType, and the import it needs (empty if none) - mirrors the exact
// mapping Field's own doc documents for Default's Go representation
// (field.go), except FieldTypeDict: Default there is "map[string]any or
// []any" (no single concrete type a struct field could declare), so the
// generated field instead uses gorm.io/datatypes.JSON - the type GORM
// itself already knows how to scan/value against a jsonb column without
// AutoMigrate ever running (see fieldGoTypes).
type fieldGoType struct {
	goType     string
	importPath string
}

// fieldGoTypes maps every FieldType to fieldGoType - see its own doc.
var fieldGoTypes = map[FieldType]fieldGoType{
	FieldTypeString:   {goType: "string"},
	FieldTypeInt:      {goType: "int64"},
	FieldTypeFloat:    {goType: "float64"},
	FieldTypeDecimal:  {goType: "decimal.Decimal", importPath: "github.com/shopspring/decimal"},
	FieldTypeBool:     {goType: "bool"},
	FieldTypeDate:     {goType: "string"},
	FieldTypeTime:     {goType: "string"},
	FieldTypeDateTime: {goType: "time.Time", importPath: "time"},
	FieldTypeInterval: {goType: "time.Duration", importPath: "time"},
	FieldTypeDict:     {goType: "datatypes.JSON", importPath: "gorm.io/datatypes"},
}

// goTypeRef is a BaseModel/BaseRepo string (see ModelSchema.BaseModel/
// BaseRepo's own doc) parsed into its import path and type name.
type goTypeRef struct {
	importPath string
	typeName   string
}

// parseGoTypeRef parses a bare "import/path.Type" string (a BaseModel or
// BaseRepo value - field names which one, purely for the error message)
// - split on the LAST "." in s, which works even when the import path
// itself contains dots ("gorm.io/gorm.Model" -> importPath "gorm.io/gorm",
// typeName "Model"), the same way "github.com/epicoon/lxgo/query.BaseModel"
// splits into "github.com/epicoon/lxgo/query" + "BaseModel". This package
// never accepts a short alias in place of the real import path (e.g. bare
// "gorm.Model") - a custom BaseModel/BaseRepo can be any import path at
// all, so there's no fixed set of aliases this could resolve against; the
// two well-known BaseModel types are additionally recognized by their
// full (importPath, typeName) pair - see baseModelsWithTimestamps.
func parseGoTypeRef(field, s string) (goTypeRef, error) {
	i := strings.LastIndex(s, ".")
	if i <= 0 || i == len(s)-1 {
		return goTypeRef{}, fmt.Errorf("invalid %s %q: want \"import/path.Type\"", field, s)
	}
	return goTypeRef{importPath: s[:i], typeName: s[i+1:]}, nil
}

// packageAlias returns the Go identifier BuildModelCode imports importPath
// under - always explicit (import alias "importPath"), rather than relying
// on whatever name importPath's own package declares itself under (a bare
// import only works correctly when the last path segment happens to match
// the package's own declared name - true for the two well-known base
// types this package recognizes, but not guaranteed for an arbitrary
// custom BaseModel). The alias itself is still derived from the last
// "/"-separated segment, purely for a readable generated file - being
// explicit about it in the import statement is what actually makes it
// correct regardless of the real package name.
func packageAlias(importPath string) string {
	if i := strings.LastIndex(importPath, "/"); i >= 0 {
		return importPath[i+1:]
	}
	return importPath
}

// baseModelsWithTimestamps are the base types BuildModelCode already knows
// carry CreatedAt/UpdatedAt/DeletedAt themselves - lxgo-query.BaseModel and
// GORM's own gorm.Model. A model embedding one of these with Timestamps
// also on doesn't get its own explicit CreatedAt/UpdatedAt/DeletedAt
// fields generated - GORM already picks the embedded ones up by name.
var baseModelsWithTimestamps = map[goTypeRef]bool{
	{importPath: "gorm.io/gorm", typeName: "Model"}:                      true,
	{importPath: "github.com/epicoon/lxgo/query", typeName: "BaseModel"}: true,
}

// goImports is a set of import paths a generated file needs, each always
// under its own explicit alias (see packageAlias) - collected while
// building a struct's fields, then rendered once, sorted by path.
type goImports map[string]string // importPath -> alias

func (imps goImports) add(importPath string) {
	if importPath == "" {
		return
	}
	imps[importPath] = packageAlias(importPath)
}

// render writes imps as a single "import (...)" block, sorted by path -
// empty string if imps is empty (no import block at all).
func (imps goImports) render() string {
	if len(imps) == 0 {
		return ""
	}
	paths := make([]string, 0, len(imps))
	for p := range imps {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	var b strings.Builder
	b.WriteString("import (\n")
	for _, p := range paths {
		fmt.Fprintf(&b, "\t%s %q\n", imps[p], p)
	}
	b.WriteString(")\n")
	return b.String()
}

// structField is one line of a generated struct's field list - Name/Type
// rendered as-is, Tag wrapped in `gorm:"..."` when non-empty.
type structField struct {
	Name string
	Type string
	Tag  string
}

// render writes f as one struct field line - an embedded field (f.Name
// empty, an embedded BaseModel) has no leading identifier, just its type.
func (f structField) render() string {
	name := f.Name
	if name != "" {
		name += " "
	}
	if f.Tag == "" {
		return fmt.Sprintf("\t%s%s\n", name, f.Type)
	}
	return fmt.Sprintf("\t%s%s `gorm:%q`\n", name, f.Type, f.Tag)
}

// identifier returns the Go identifier f occupies in the generated struct
// - f.Name if set, else (an embedded field, f.Name empty) the embedded
// type's own unqualified name, the last "."-segment of f.Type (e.g.
// "BaseModel" for an embedded "query.BaseModel") - Go requires struct
// field names to be unique regardless of whether a field is named
// explicitly or occupies its name implicitly through embedding.
func (f structField) identifier() string {
	if f.Name != "" {
		return f.Name
	}
	if i := strings.LastIndex(f.Type, "."); i >= 0 {
		return f.Type[i+1:]
	}
	return f.Type
}

// checkNoFieldCollisions returns an error if any two fields across groups
// would occupy the same Go identifier (see structField.identifier) - a
// schema is free to declare a Field or Relation named e.g. "ID" (nothing
// about the yaml schema/DDL pipeline itself cares, "ID" isn't a reserved
// word there), but the generated struct can't have two fields with the
// same name, whether that's a declared Field colliding with the bare "ID"
// this generator adds when no BaseModel is set, a Field and a Relation
// sharing a name (declared in separate `Fields:`/`Relations:` sections of
// the schema file, never cross-checked against each other outside of
// this), or a relation's own generated "<name>ID" FK field colliding with
// something else entirely.
func checkNoFieldCollisions(groups [][]structField) error {
	seen := map[string]bool{}
	for _, group := range groups {
		for _, f := range group {
			id := f.identifier()
			if seen[id] {
				return fmt.Errorf("two fields would both be named %q in the generated struct - rename one of the colliding fields/relations in the schema", id)
			}
			seen[id] = true
		}
	}
	return nil
}

// fieldGormTag builds the gorm struct tag for a declared Field - an
// explicit column name (never relies on NamingStrategy happening to
// reproduce the same physical name this package's own DDL already
// computed - see pgColumnName/pgTruncateIdent), "not null" when Required,
// and "default:..." when Default is set. The default tag isn't decorative
// the way a "type:" tag would be (this package deliberately never
// generates one - AutoMigrate is never invoked against generated code, so
// it would carry no functional weight): GORM's Create still reads
// "default" at runtime, to decide whether a field left at its Go zero
// value should be omitted from the INSERT column list, letting the
// column's own DEFAULT apply instead of explicitly writing the zero value
// over it.
//
// Returns an error if Default's own formatted text contains ";" - GORM's
// own tag parser splits a struct tag's gorm value on exactly that
// character to find each setting, so a default containing one (only
// possible for FieldTypeString - every other type's formatted text is a
// fixed, safe shape) would silently truncate the default and inject
// whatever follows the ";" as an unrelated tag setting, rather than
// surfacing as an error anywhere.
func fieldGormTag(physicalName string, f Field) (string, error) {
	parts := []string{"column:" + physicalName}
	if f.Required {
		parts = append(parts, "not null")
	}
	if f.Default != nil {
		def := fmt.Sprintf("%v", formatDefault(f.Type, f.Default))
		if strings.Contains(def, ";") {
			return "", fmt.Errorf("default value %q can't be represented in a gorm tag - it contains \";\", which gorm's own tag parser treats as a setting separator", def)
		}
		parts = append(parts, "default:"+def)
	}
	return strings.Join(parts, ";"), nil
}

// relationFields returns the Go struct fields BuildModelCode generates for
// r (declared under name on modelName, the model being generated) - nil
// if r's type isn't in this generator's scope (RelationTypeOneToMany's
// own reverse side has no codegen of its own at all - its physical shape
// is entirely the RelationTypeManyToOne side's, see BuildModelCode's own
// doc).
//
// The FK holder (RelationTypeManyToOne always, RelationTypeOneToOne when
// r.FkHolder) gets two fields: the physical <name>ID column (always
// "not null", additionally "unique" for RelationTypeOneToOne - its FK
// column always carries a UNIQUE constraint, see execAddToOneRelation) and
// the association itself, value type (never pointer - the physical column
// is NOT NULL, so a matching related row always exists). The
// RelationTypeOneToOne non-holder gets one field: the association alone,
// pointer type (nothing physical on this side guarantees a matching row
// exists) - its own foreignKey tag names the HOLDER's own <RelatedAttribute>ID
// field, which is why a RelationTypeOneToOne non-holder declared "uni"
// (FkHolder false, RelatedAttribute empty) is rejected: there would be no
// way to know which field on the related struct actually holds the FK.
//
// RelationTypeManyToMany gets one field, a slice association
// (`[]RelatedModel`, tagged `many2many:<joinTable>;joinForeignKey:<own
// column>;joinReferences:<related column>`) - the join table/column names
// are computed by the exact same pgManyToManyTableName/pgJoinColumnName
// this package's own DDL already uses (see execAddManyToManyRelation),
// not left for GORM's own NamingStrategy to derive from the association
// alone (which never matches - see BuildModelCode's own doc). Both
// sides' own physical column names are derivable purely from r itself
// (modelName/name for this side, r.RelatedModel/r.RelatedAttribute -
// always set for RelationTypeManyToMany, "uni" isn't valid for it - for
// the other), so a self-referential relation (RelatedModel == modelName,
// two distinct attributes on the same schema, e.g. "friends"/"friendOf")
// still gets two distinct fields with two distinct physical join columns,
// without this function ever needing to look at the related schema.
func relationFields(modelName, name string, r Relation) ([]structField, error) {
	switch r.Type {
	case RelationTypeManyToOne:
		return toOneHolderFields(name, r, false), nil

	case RelationTypeOneToOne:
		if r.FkHolder {
			return toOneHolderFields(name, r, true), nil
		}
		if r.RelatedAttribute == "" {
			return nil, fmt.Errorf("relation %q: a non-FK-holding OneToOne relation must name RelatedAttribute - there's no other way to know which field on %q holds the foreign key", name, r.RelatedModel)
		}
		fkGoField := r.RelatedAttribute + "ID"
		return []structField{{
			Name: name, Type: "*" + r.RelatedModel,
			Tag: fmt.Sprintf("foreignKey:%s;references:ID", fkGoField),
		}}, nil

	case RelationTypeManyToMany:
		joinTable := pgManyToManyTableName(modelName, name, r.RelatedModel, r.RelatedAttribute)
		ownColumn := pgJoinColumnName(modelName, name)
		relColumn := pgJoinColumnName(r.RelatedModel, r.RelatedAttribute)
		return []structField{{
			Name: name, Type: "[]" + r.RelatedModel,
			Tag: fmt.Sprintf("many2many:%s;joinForeignKey:%s;joinReferences:%s", joinTable, ownColumn, relColumn),
		}}, nil

	default:
		return nil, nil
	}
}

// toOneHolderFields builds the FK-holder pair of fields (raw FK column +
// association) shared by RelationTypeManyToOne and a RelationTypeOneToOne
// FK holder - see relationFields's own doc.
func toOneHolderFields(name string, r Relation, unique bool) []structField {
	fkGoField := name + "ID"
	fkTag := "column:" + pgRelationColumnName(name) + ";not null"
	if unique {
		fkTag += ";unique"
	}
	return []structField{
		{Name: fkGoField, Type: "uint", Tag: fkTag},
		{Name: name, Type: r.RelatedModel, Tag: fmt.Sprintf("foreignKey:%s;references:ID", fkGoField)},
	}
}

// timestampGoFields is the CreatedAt/UpdatedAt/DeletedAt fields
// BuildModelCode generates when Timestamps is on and the resolved
// BaseModel doesn't already carry them (see baseModelsWithTimestamps) -
// the same Go types/tag lxgo-query.BaseModel itself declares them with
// (gorm.DeletedAt, not time.Time, so GORM's own soft-delete filtering on
// Find/Delete applies the same way it would through an embedded
// BaseModel), column names explicit for the same reason every other field
// gets one.
func timestampGoFields() []structField {
	return []structField{
		{Name: "CreatedAt", Type: "time.Time", Tag: "column:created_at"},
		{Name: "UpdatedAt", Type: "time.Time", Tag: "column:updated_at"},
		{Name: "DeletedAt", Type: "gorm.DeletedAt", Tag: "column:deleted_at;index"},
	}
}

// ModelCodeFileName returns the file name BuildModelCode's own output for
// modelName is written under - <snake_case(modelName)>_gen.go, the same
// snake_case conversion this package already applies to a field name
// (pgColumnName), not pgTableName's (which additionally pluralizes - a
// file name isn't a table name).
func ModelCodeFileName(modelName string) string {
	return pgColumnName(modelName) + "_gen.go"
}

// BuildModelCode generates the Go source of schema's GORM-mapped struct -
// its declared Fields (see fieldGoTypes/fieldGormTag), RelationTypeOneToOne/
// RelationTypeManyToOne/RelationTypeManyToMany relations (see
// relationFields - RelationTypeOneToMany's own reverse side has no
// codegen of its own, its physical shape is entirely the
// RelationTypeManyToOne side's), a resolved BaseModel embed or a bare "ID
// uint" field if none is set (see EffectiveBaseModel), explicit
// CreatedAt/UpdatedAt/DeletedAt fields if EffectiveTimestamps is on and
// the resolved BaseModel doesn't already carry them, and a TableName()
// method (schema-qualified when EffectiveNamespace is set).
//
// A relation's related type is always referenced as a bare identifier,
// never import-qualified - this assumes every related
// model is generated into the same Go package as schema itself. A schema
// whose relation points at a model generated into a different package
// (e.g. a different Target's own Models directory) produces code that
// fails to compile with a clear "undefined: X" error rather than silently
// wrong code - resolving that is left to the caller (e.g. a console
// command could reject such a relation before ever calling this).
//
// The result is gofmt-formatted (go/format.Source) - a syntax error in the
// assembled source (e.g. an invalid BaseModel reference, see
// parseGoTypeRef) surfaces here as an error rather than as unreadable
// or non-compiling output.
func BuildModelCode(pkgName string, schema *ModelSchema) ([]byte, error) {
	imps := goImports{}

	// base is always exactly one field (the BaseModel embed, or a bare ID),
	// its own group so a blank line always separates it from whatever
	// follows.
	var base structField
	baseModel := schema.EffectiveBaseModel()
	var baseRef goTypeRef
	if baseModel != "" {
		ref, err := parseGoTypeRef("BaseModel", baseModel)
		if err != nil {
			return nil, fmt.Errorf("model %q: %w", schema.Name, err)
		}
		baseRef = ref
		imps.add(ref.importPath)
		base = structField{Type: imps[ref.importPath] + "." + ref.typeName}
	} else {
		base = structField{Name: "ID", Type: "uint", Tag: "column:id;primaryKey"}
	}

	var declared []structField
	for _, f := range schema.Fields {
		gt, ok := fieldGoTypes[f.Field.Type]
		if !ok {
			return nil, fmt.Errorf("model %q: field %q: unknown field type %q", schema.Name, f.Name, f.Field.Type)
		}
		tag, err := fieldGormTag(pgColumnName(f.Name), f.Field)
		if err != nil {
			return nil, fmt.Errorf("model %q: field %q: %w", schema.Name, f.Name, err)
		}
		imps.add(gt.importPath)
		declared = append(declared, structField{Name: f.Name, Type: gt.goType, Tag: tag})
	}

	var timestamps []structField
	if schema.EffectiveTimestamps() && !baseModelsWithTimestamps[baseRef] {
		imps.add("time")
		imps.add("gorm.io/gorm")
		timestamps = timestampGoFields()
	}

	var relations []structField
	for _, r := range schema.Relations {
		rf, err := relationFields(schema.Name, r.Name, r.Relation)
		if err != nil {
			return nil, fmt.Errorf("model %q: %w", schema.Name, err)
		}
		relations = append(relations, rf...)
	}

	groups := [][]structField{{base}, declared, timestamps, relations}

	if err := checkNoFieldCollisions(groups); err != nil {
		return nil, fmt.Errorf("model %q: %w", schema.Name, err)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "// Code generated by lxgo-model; DO NOT EDIT.\n\n")
	fmt.Fprintf(&b, "package %s\n\n", pkgName)
	if rendered := imps.render(); rendered != "" {
		b.WriteString(rendered)
		b.WriteString("\n")
	}
	fmt.Fprintf(&b, "// %s is generated from the %q model schema.\n", schema.Name, schema.Name)
	fmt.Fprintf(&b, "type %s struct {\n", schema.Name)
	for i, group := range groups {
		if len(group) == 0 {
			continue
		}
		if i > 0 {
			b.WriteString("\n")
		}
		for _, f := range group {
			b.WriteString(f.render())
		}
	}
	b.WriteString("}\n\n")

	fmt.Fprintf(&b, "func (%s) TableName() string {\n", schema.Name)
	physTable := pgTableName(schema.Name)
	if ns := schema.EffectiveNamespace(); ns != "" {
		fmt.Fprintf(&b, "\treturn %q\n", pgQualifiedTable(ns, physTable))
	} else {
		fmt.Fprintf(&b, "\treturn %q\n", physTable)
	}
	b.WriteString("}\n")

	formatted, err := format.Source([]byte(b.String()))
	if err != nil {
		return nil, fmt.Errorf("model %q: generated invalid Go source: %w", schema.Name, err)
	}
	return formatted, nil
}
