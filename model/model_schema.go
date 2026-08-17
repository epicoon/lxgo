package model

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// NamedField pairs a Field with the name it's declared under in a schema
// file - a model's fields are a list, not a map, so their file order
// survives (a plain Go map would randomize it on every iteration, making
// generated migrations look different across runs with no actual schema
// change).
type NamedField struct {
	Name string
	Field
}

// ModelSchema is one model's parsed yaml schema file - see LoadModelSchema/
// LoadModelSchemas.
type ModelSchema struct {
	// Name is the model's name - the schema file's basename, not
	// necessarily whatever its own (optional, purely informational) `Name:`
	// key says.
	Name string
	// SourceDir is the directory the schema was loaded from - set by
	// LoadModelSchemas/LoadModelSchemasFromDirs, empty for a ModelSchema
	// built by hand or through LoadModelSchema directly. Used to write the
	// schema back to the directory it actually came from when more than
	// one is in play (see GenerateMigration).
	SourceDir string
	// Namespace declares this model's own explicit Postgres schema
	// override - the schema file's own `Namespace:` key, empty if not
	// declared. Round-trips through Save/MarshalYAML like Fields/
	// Relations - never silently rewritten with an inherited default (see
	// ResolvedNamespace, which is where a directory-level default ends up
	// instead).
	Namespace string
	// ResolvedNamespace is this model's effective Postgres schema after
	// applying the directory-level default it was loaded under (see
	// ModelManager.LoadModelSchemas) - Namespace if that's set, else the
	// directory's own default. Empty for a ModelSchema loaded any other
	// way (LoadModelSchema, LoadModelSchemasFromDirs, or built by hand) -
	// see EffectiveNamespace for the fallback callers should actually use.
	// Never marshaled - purely a load-time computed value, so saving a
	// schema back never bakes in an inherited default as if it were an
	// explicit override.
	ResolvedNamespace string
	// BaseModel declares this model's own explicit Go type to embed as the
	// generated struct's base (a code generator's concern, not this
	// package's own DDL/comparison path - see EffectiveBaseModel) - the
	// schema file's own `BaseModel:` key, a bare "package/path.Type"
	// reference (e.g. "github.com/epicoon/lxgo/query.BaseModel"), empty if
	// not declared. Not validated or interpreted by this package at all -
	// opaque as far as ModelManager.LoadModelSchemas's cascade resolution
	// is concerned, same as Namespace. Round-trips through Save/MarshalYAML
	// never silently rewritten with an inherited default (see ResolvedBaseModel,
	// which is where a directory-level default ends up instead).
	BaseModel string
	// ResolvedBaseModel is this model's effective base type after applying
	// the directory-level default it was loaded under (see
	// ModelManager.LoadModelSchemas) - BaseModel if that's set, else the
	// directory's own default. Empty for a ModelSchema loaded any other way
	// - see EffectiveBaseModel for the fallback callers should actually
	// use. Never marshaled, same reasoning as ResolvedNamespace.
	ResolvedBaseModel string
	// BaseRepo declares this model's own explicit generic repository type
	// to embed when scaffolding a repository (a code generator's concern,
	// same as BaseModel) - the schema file's own `BaseRepo:` key, a bare
	// "package/path.Type" reference (e.g.
	// "github.com/epicoon/lxgo/query.BaseRepo"), empty if not declared.
	// Not validated or interpreted by this package at all, same as
	// BaseModel. Round-trips through Save/MarshalYAML - never silently
	// rewritten with an inherited default (see ResolvedBaseRepo, which is
	// where a directory-level default ends up instead).
	BaseRepo string
	// ResolvedBaseRepo is this model's effective repository base type
	// after applying the directory-level default it was loaded under (see
	// ModelManager.LoadModelSchemas) - BaseRepo if that's set, else the
	// directory's own default. Empty for a ModelSchema loaded any other
	// way - see EffectiveBaseRepo for the fallback callers should
	// actually use. Never marshaled, same reasoning as ResolvedBaseModel.
	ResolvedBaseRepo string
	// Timestamps declares this model's own explicit override for whether
	// execCreateTable adds created_at/updated_at/deleted_at columns (see
	// EffectiveTimestamps) - the schema file's own `Timestamps:` key, nil
	// if not declared. A pointer, unlike Namespace/BaseModel's plain
	// strings, because a bool has no natural "not declared" value distinct
	// from an explicit false. Round-trips through Save/MarshalYAML like
	// Namespace/BaseModel - never silently rewritten with an inherited
	// default (see ResolvedTimestamps, which is where a directory-level
	// default ends up instead).
	Timestamps *bool
	// ResolvedTimestamps is this model's effective Timestamps switch after
	// applying the directory-level default it was loaded under (see
	// ModelManager.LoadModelSchemas) - Timestamps if that's set, else the
	// directory's own default, else the component-wide default. nil for a
	// ModelSchema loaded any other way (LoadModelSchema,
	// LoadModelSchemasFromDirs, or built by hand) - see EffectiveTimestamps
	// for the fallback callers should actually use. Never marshaled -
	// purely a load-time computed value, same reasoning as
	// ResolvedNamespace/ResolvedBaseModel.
	ResolvedTimestamps *bool
	// Fields are the model's fields, in the order they appear in the file.
	Fields []NamedField
	// Relations are the model's relations to other models, in the order
	// they appear in the file - see Relation.
	Relations []NamedRelation
}

// EffectiveNamespace returns s's actual Postgres schema regardless of how
// s was obtained - ResolvedNamespace if the cascade already computed one
// (see ModelManager.LoadModelSchemas), else Namespace directly (correct
// both for a ModelSchema loaded without directory-level cascading and for
// one built by hand). Empty means no schema override anywhere.
func (s *ModelSchema) EffectiveNamespace() string {
	if s.ResolvedNamespace != "" {
		return s.ResolvedNamespace
	}
	return s.Namespace
}

// EffectiveBaseModel returns s's actual base type to embed regardless of
// how s was obtained - ResolvedBaseModel if the cascade already computed
// one (see ModelManager.LoadModelSchemas), else BaseModel directly (correct
// both for a ModelSchema loaded without directory-level cascading and for
// one built by hand). Empty means no base type override anywhere - a code
// generator's own default applies.
func (s *ModelSchema) EffectiveBaseModel() string {
	if s.ResolvedBaseModel != "" {
		return s.ResolvedBaseModel
	}
	return s.BaseModel
}

// EffectiveBaseRepo returns s's actual repository base type to embed when
// scaffolding a repository, regardless of how s was obtained -
// ResolvedBaseRepo if the cascade already computed one (see
// ModelManager.LoadModelSchemas), else BaseRepo directly. Empty means no
// override anywhere - a code generator's own default applies.
func (s *ModelSchema) EffectiveBaseRepo() string {
	if s.ResolvedBaseRepo != "" {
		return s.ResolvedBaseRepo
	}
	return s.BaseRepo
}

// EffectiveTimestamps returns whether s should get created_at/updated_at/
// deleted_at columns, regardless of how s was obtained - ResolvedTimestamps
// if the cascade already computed one (see ModelManager.LoadModelSchemas),
// else Timestamps directly, else false - the cascade's own terminal default
// when nothing overrides it anywhere.
func (s *ModelSchema) EffectiveTimestamps() bool {
	if s.ResolvedTimestamps != nil {
		return *s.ResolvedTimestamps
	}
	if s.Timestamps != nil {
		return *s.Timestamps
	}
	return false
}

// FieldByName returns the field declared under name and whether it exists.
func (s *ModelSchema) FieldByName(name string) (Field, bool) {
	for _, f := range s.Fields {
		if f.Name == name {
			return f.Field, true
		}
	}
	return Field{}, false
}

// RelationByName returns the relation declared under name and whether it
// exists.
func (s *ModelSchema) RelationByName(name string) (Relation, bool) {
	for _, r := range s.Relations {
		if r.Name == name {
			return r.Relation, true
		}
	}
	return Relation{}, false
}

// LoadModelSchema parses one model schema file's content - name is the
// resulting ModelSchema.Name (the caller decides it, typically the file's
// basename - see LoadModelSchemas).
func LoadModelSchema(name string, data []byte) (*ModelSchema, error) {
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, err
	}

	schema := &ModelSchema{Name: name}
	if len(root.Content) == 0 {
		return schema, nil
	}

	doc := root.Content[0]
	if doc.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("a model schema must be a yaml mapping, got %s", yamlKindName(doc.Kind))
	}

	seenKeys := make(map[string]bool, len(doc.Content)/2)
	for i := 0; i < len(doc.Content); i += 2 {
		key := doc.Content[i].Value
		value := doc.Content[i+1]

		if seenKeys[key] {
			return nil, fmt.Errorf("duplicate top-level key %q", key)
		}
		seenKeys[key] = true

		switch key {
		case "Name":
			var declared string
			if err := value.Decode(&declared); err != nil {
				return nil, fmt.Errorf("invalid 'Name': %w", err)
			}
		case "Namespace":
			if err := value.Decode(&schema.Namespace); err != nil {
				return nil, fmt.Errorf("invalid 'Namespace': %w", err)
			}
		case "BaseModel":
			if err := value.Decode(&schema.BaseModel); err != nil {
				return nil, fmt.Errorf("invalid 'BaseModel': %w", err)
			}
		case "BaseRepo":
			if err := value.Decode(&schema.BaseRepo); err != nil {
				return nil, fmt.Errorf("invalid 'BaseRepo': %w", err)
			}
		case "Timestamps":
			var declared bool
			if err := value.Decode(&declared); err != nil {
				return nil, fmt.Errorf("invalid 'Timestamps': %w", err)
			}
			schema.Timestamps = &declared
		case "Fields":
			fields, err := parseFieldsNode(value)
			if err != nil {
				return nil, fmt.Errorf("invalid 'Fields': %w", err)
			}
			schema.Fields = fields
		case "Relations":
			relations, err := parseRelationsNode(value)
			if err != nil {
				return nil, fmt.Errorf("invalid 'Relations': %w", err)
			}
			schema.Relations = relations
		default:
			return nil, fmt.Errorf("unknown top-level key %q", key)
		}
	}

	return schema, nil
}

func parseFieldsNode(node *yaml.Node) ([]NamedField, error) {
	if node.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("must be a yaml mapping, got %s", yamlKindName(node.Kind))
	}

	fields := make([]NamedField, 0, len(node.Content)/2)
	seen := make(map[string]bool, len(node.Content)/2)
	for i := 0; i < len(node.Content); i += 2 {
		fieldName := node.Content[i].Value
		if seen[fieldName] {
			return nil, fmt.Errorf("duplicate field %q", fieldName)
		}
		seen[fieldName] = true

		var f Field
		if err := node.Content[i+1].Decode(&f); err != nil {
			return nil, fmt.Errorf("field %q: %w", fieldName, err)
		}
		fields = append(fields, NamedField{Name: fieldName, Field: f})
	}
	return fields, nil
}

// loadModelSchemaFiles is LoadModelSchemas without the relation
// cross-validation step - split out so LoadModelSchemasFromDirs can gather
// every directory's schemas first and validate relations once across the
// whole combined set, instead of one directory at a time (which would
// wrongly reject a relation to a model declared in a different directory).
func loadModelSchemaFiles(dir string) ([]*ModelSchema, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", dir, err)
	}

	var schemas []*ModelSchema
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read schema file %q: %w", entry.Name(), err)
		}

		name := strings.TrimSuffix(entry.Name(), ".yaml")
		schema, err := LoadModelSchema(name, data)
		if err != nil {
			return nil, fmt.Errorf("failed to parse schema file %q: %w", entry.Name(), err)
		}
		schema.SourceDir = dir
		schemas = append(schemas, schema)
	}

	sort.Slice(schemas, func(i, j int) bool { return schemas[i].Name < schemas[j].Name })
	return schemas, nil
}

// MarshalYAML writes s as {Name, Namespace, BaseModel, BaseRepo,
// Timestamps, Fields, Relations} - the same shape LoadModelSchema parses
// (each key through its own element type's yaml marshaling). Used both by
// Save and by anything embedding a *ModelSchema in a larger yaml.Marshal
// call (e.g. a generated migration's createTable/dropTable action) -
// unlike a schema file loaded through LoadModelSchema (where the file's
// own Name key is purely informational, overridden by the filename), a
// ModelSchema marshaled and later unmarshaled this way round-trips Name
// through the Name key directly, there being no filename to derive it
// from instead. Namespace/BaseModel/BaseRepo/Timestamps/Relations are
// each omitted entirely when empty/nil, matching a schema file that never
// had that key at all - Namespace/BaseModel/BaseRepo/Timestamps in
// particular write s.Namespace/s.BaseModel/s.BaseRepo/s.Timestamps (the
// model's own declared override), never s.ResolvedNamespace/
// s.ResolvedBaseModel/s.ResolvedBaseRepo/s.ResolvedTimestamps (an
// inherited directory-level default, which would otherwise get baked into
// the file as if it were an explicit override - see ResolvedNamespace/
// ResolvedBaseModel/ResolvedBaseRepo/ResolvedTimestamps's own doc).
func (s *ModelSchema) MarshalYAML() (any, error) {
	doc := &yaml.Node{Kind: yaml.MappingNode}
	doc.Content = append(doc.Content, strNode("Name"), strNode(s.Name))
	if s.Namespace != "" {
		doc.Content = append(doc.Content, strNode("Namespace"), strNode(s.Namespace))
	}
	if s.BaseModel != "" {
		doc.Content = append(doc.Content, strNode("BaseModel"), strNode(s.BaseModel))
	}
	if s.BaseRepo != "" {
		doc.Content = append(doc.Content, strNode("BaseRepo"), strNode(s.BaseRepo))
	}
	if s.Timestamps != nil {
		var tsNode yaml.Node
		if err := tsNode.Encode(*s.Timestamps); err != nil {
			return nil, fmt.Errorf("Timestamps: %w", err)
		}
		doc.Content = append(doc.Content, strNode("Timestamps"), &tsNode)
	}

	fieldsNode := &yaml.Node{Kind: yaml.MappingNode}
	for _, f := range s.Fields {
		var valueNode yaml.Node
		if err := valueNode.Encode(f.Field); err != nil {
			return nil, fmt.Errorf("field %q: %w", f.Name, err)
		}
		fieldsNode.Content = append(fieldsNode.Content, strNode(f.Name), &valueNode)
	}
	doc.Content = append(doc.Content, strNode("Fields"), fieldsNode)

	if len(s.Relations) > 0 {
		relationsNode := &yaml.Node{Kind: yaml.MappingNode}
		for _, r := range s.Relations {
			var valueNode yaml.Node
			if err := valueNode.Encode(r.Relation); err != nil {
				return nil, fmt.Errorf("relation %q: %w", r.Name, err)
			}
			relationsNode.Content = append(relationsNode.Content, strNode(r.Name), &valueNode)
		}
		doc.Content = append(doc.Content, strNode("Relations"), relationsNode)
	}

	return doc, nil
}

// UnmarshalYAML parses the {Name, Namespace, BaseModel, Timestamps, Fields,
// Relations} shape MarshalYAML writes - the counterpart used when a *ModelSchema is
// unmarshaled generically rather than through LoadModelSchema (which
// derives Name from the schema file's own filename, not its content, and
// so parses the document by hand instead of using this method).
func (s *ModelSchema) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.MappingNode {
		return fmt.Errorf("a model schema must be a yaml mapping, got %s", yamlKindName(value.Kind))
	}

	for i := 0; i < len(value.Content); i += 2 {
		key := value.Content[i].Value
		v := value.Content[i+1]

		switch key {
		case "Name":
			if err := v.Decode(&s.Name); err != nil {
				return fmt.Errorf("invalid 'Name': %w", err)
			}
		case "Namespace":
			if err := v.Decode(&s.Namespace); err != nil {
				return fmt.Errorf("invalid 'Namespace': %w", err)
			}
		case "BaseModel":
			if err := v.Decode(&s.BaseModel); err != nil {
				return fmt.Errorf("invalid 'BaseModel': %w", err)
			}
		case "BaseRepo":
			if err := v.Decode(&s.BaseRepo); err != nil {
				return fmt.Errorf("invalid 'BaseRepo': %w", err)
			}
		case "Timestamps":
			var declared bool
			if err := v.Decode(&declared); err != nil {
				return fmt.Errorf("invalid 'Timestamps': %w", err)
			}
			s.Timestamps = &declared
		case "Fields":
			fields, err := parseFieldsNode(v)
			if err != nil {
				return fmt.Errorf("invalid 'Fields': %w", err)
			}
			s.Fields = fields
		case "Relations":
			relations, err := parseRelationsNode(v)
			if err != nil {
				return fmt.Errorf("invalid 'Relations': %w", err)
			}
			s.Relations = relations
		default:
			return fmt.Errorf("unknown top-level key %q", key)
		}
	}
	return nil
}

// Save writes s back to path as yaml (see MarshalYAML).
func (s *ModelSchema) Save(path string) error {
	out, err := marshalYAML(s)
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0644)
}

func marshalYAML(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func strNode(s string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: s}
}
