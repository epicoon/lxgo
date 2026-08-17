package model

import (
	"errors"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// RelationType is a model relation's declared kind - see Relation.
type RelationType string

const (
	RelationTypeOneToOne   RelationType = "OneToOne"
	RelationTypeOneToMany  RelationType = "OneToMany"
	RelationTypeManyToOne  RelationType = "ManyToOne"
	RelationTypeManyToMany RelationType = "ManyToMany"
)

var knownRelationTypes = map[RelationType]bool{
	RelationTypeOneToOne:   true,
	RelationTypeOneToMany:  true,
	RelationTypeManyToOne:  true,
	RelationTypeManyToMany: true,
}

// Relation is one model's declared relation to another model - a schema
// file's `Relations:` entry. Two equivalent forms are accepted - a compact
// string, `(TYPE) RelatedModel[.relatedAttribute]`, where TYPE is:
// `--` - RelationTypeOneToOne
// `-<` - RelationTypeOneToMany
// `>-` - RelationTypeManyToOne
// `><` - RelationTypeManyToMany
//
// and a structured map form:
//
//	FieldName:
//	  Type: ManyToMany
//	  Model: ContrModelName
//	  Field: ContrFieldName
//
// Neither form is a shorthand for the other - both accept the exact same
// values, and a Relation marshaled back to yaml is written the same way it
// was read.
//
// RelatedAttribute is the matching relation's name on RelatedModel's own
// schema - required for RelationTypeOneToMany/RelationTypeManyToMany
// (there's no other way to tell which of possibly several relations to
// RelatedModel this one pairs with), optional for
// RelationTypeOneToOne/RelationTypeManyToOne, in which case this side's
// declaration stands alone (a "uni" relation - the other model doesn't
// need to declare anything back).
//
// FkHolder marks this side as the physical foreign key column's owner -
// only meaningful for RelationTypeOneToOne, written as `(FK--)` instead of
// `(--)` in the compact form, `OneToOne(FK)` instead of `OneToOne` in the
// map form. RelationTypeManyToOne is always the FK holder regardless of
// this field (the "many" side's table is where the column naturally
// lives); RelationTypeOneToMany/RelationTypeManyToMany never hold a single
// FK column at all (the former's FK lives on its RelationTypeManyToOne
// counterpart, the latter uses a join table).
//
// NoIndex opts this side's own physical foreign key column out of the
// index a generated migration otherwise creates on it by default (Postgres
// doesn't index a foreign key column automatically, and the typical
// access pattern for a relation - join/lookup by it - usually wants one;
// NoIndex is the exception, not the rule). Written `no-index` in the
// compact form (a third, optional token after RelatedModel[.attribute]),
// `Index: false` in the map form (the key is inverted in the schema file
// relative to this field - omitted or `true` means indexed, the common
// case). Only meaningful for RelationTypeManyToOne (always the FK holder)
// and RelationTypeManyToMany (each side has its own join-table column,
// indexed independently) - invalid for RelationTypeOneToMany (no physical
// column on this side to index at all) and RelationTypeOneToOne (its
// FkHolder side's column is backed by a UNIQUE constraint, which always
// carries its own index in Postgres - there's no way to have one without
// the other, so declaring NoIndex here could never actually be honored).
type Relation struct {
	Type             RelationType
	RelatedModel     string
	RelatedAttribute string
	FkHolder         bool
	NoIndex          bool

	// compactForm records whether this Relation was parsed from the
	// compact single-line form rather than the map form - MarshalYAML
	// writes it back the same way it was read, see marshalCompactForm. A
	// programmatically-built Relation (not parsed) writes as the map form.
	compactForm bool
}

// IsEqual reports whether r and other declare the same relation, ignoring
// their name (compared separately by the caller, same reasoning as
// Field.IsEqual).
func (r Relation) IsEqual(other Relation) bool {
	return r.Type == other.Type &&
		r.RelatedModel == other.RelatedModel &&
		r.RelatedAttribute == other.RelatedAttribute &&
		r.FkHolder == other.FkHolder &&
		r.NoIndex == other.NoIndex
}

// NamedRelation pairs a Relation with the name it's declared under in a
// schema file.
type NamedRelation struct {
	Name string
	Relation
}

// relationTokens maps a compact form's type token to the
// RelationType/FkHolder pair it declares - the sole place either is
// spelled out, used by both parsing and marshaling.
var relationTokens = map[string]struct {
	typ      RelationType
	fkHolder bool
}{
	"(--)":   {RelationTypeOneToOne, false},
	"(FK--)": {RelationTypeOneToOne, true},
	"(-<)":   {RelationTypeOneToMany, false},
	"(>-)":   {RelationTypeManyToOne, false},
	"(><)":   {RelationTypeManyToMany, false},
}

// UnmarshalYAML parses a Relation from either of two forms a schema author
// can write: the compact single-line string or the structured map form -
// see Relation's doc. Which one applies is decided by value's yaml.Kind.
func (r *Relation) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		parsed, err := parseRelationString(value.Value)
		if err != nil {
			return err
		}
		*r = parsed
		return nil
	case yaml.MappingNode:
		parsed, err := parseRelationMap(value)
		if err != nil {
			return err
		}
		*r = parsed
		return nil
	default:
		return fmt.Errorf("a relation must be a compact string or a yaml mapping, got %s", yamlKindName(value.Kind))
	}
}

// parseRelationString parses `[(]TYPE[)] RelatedModel[.relatedAttribute]
// [no-index]` - the compact shape UnmarshalYAML accepts.
func parseRelationString(s string) (Relation, error) {
	parts := strings.Fields(s)
	if len(parts) < 2 || len(parts) > 3 {
		return Relation{}, fmt.Errorf(`relation must look like "(type) RelatedModel[.attribute] [no-index]", got %q`, s)
	}

	token, ok := relationTokens[parts[0]]
	if !ok {
		return Relation{}, fmt.Errorf("unknown relation type token %q", parts[0])
	}

	target := strings.SplitN(parts[1], ".", 2)
	relatedModel := target[0]
	var relatedAttribute string
	if len(target) == 2 {
		relatedAttribute = target[1]
	}

	noIndex := false
	if len(parts) == 3 {
		if parts[2] != "no-index" {
			return Relation{}, fmt.Errorf("unknown relation modifier %q", parts[2])
		}
		noIndex = true
	}

	return newRelation(token.typ, relatedModel, relatedAttribute, token.fkHolder, noIndex, true)
}

// newRelation validates and builds a Relation shared by both the compact
// and map form parsers - relatedModel must be given, relatedAttribute may
// only be omitted (a "uni" relation) for RelationTypeOneToOne/
// RelationTypeManyToOne (see Relation's doc), and noIndex may only be set
// for RelationTypeManyToOne/RelationTypeManyToMany (see Relation.NoIndex's
// doc for why RelationTypeOneToMany/RelationTypeOneToOne can never honor
// it).
func newRelation(t RelationType, relatedModel, relatedAttribute string, fkHolder, noIndex, compactForm bool) (Relation, error) {
	if relatedModel == "" {
		return Relation{}, fmt.Errorf("relation is missing the related model")
	}
	if relatedAttribute == "" && (t == RelationTypeOneToMany || t == RelationTypeManyToMany) {
		return Relation{}, fmt.Errorf("%s relation to %q must name the related attribute (RelatedModel.attribute)", t, relatedModel)
	}
	if noIndex {
		switch t {
		case RelationTypeOneToOne:
			return Relation{}, fmt.Errorf("relation to %q: no-index is not valid for %s - its UNIQUE constraint always carries its own index", relatedModel, t)
		case RelationTypeOneToMany:
			return Relation{}, fmt.Errorf("relation to %q: no-index is not valid for %s - this side has no physical foreign key column to index", relatedModel, t)
		}
	}

	return Relation{
		Type:             t,
		RelatedModel:     relatedModel,
		RelatedAttribute: relatedAttribute,
		FkHolder:         fkHolder,
		NoIndex:          noIndex,
		compactForm:      compactForm,
	}, nil
}

// yamlRelation mirrors Relation's map-form on-disk shape. Index is a
// pointer so "absent" (defaults to indexed, i.e. NoIndex false) can be
// told apart from an explicit `Index: false`.
type yamlRelation struct {
	Type  string `yaml:"Type"`
	Model string `yaml:"Model"`
	Field string `yaml:"Field,omitempty"`
	Index *bool  `yaml:"Index,omitempty"`
}

// allowedRelationKeys are yamlRelation's yaml keys - checked explicitly
// for the same reason as field.go's allowedFieldKeys (a typo'd key would
// otherwise be silently ignored rather than rejected).
var allowedRelationKeys = map[string]bool{
	"Type": true, "Model": true, "Field": true, "Index": true,
}

// parseRelationMap parses the structured map form - see Relation's doc.
func parseRelationMap(value *yaml.Node) (Relation, error) {
	for i := 0; i < len(value.Content); i += 2 {
		key := value.Content[i].Value
		if !allowedRelationKeys[key] {
			return Relation{}, fmt.Errorf("unknown relation attribute %q", key)
		}
	}

	var raw yamlRelation
	if err := value.Decode(&raw); err != nil {
		return Relation{}, err
	}

	t, fkHolder, err := parseRelationTypeValue(raw.Type)
	if err != nil {
		return Relation{}, err
	}

	noIndex := raw.Index != nil && !*raw.Index
	return newRelation(t, raw.Model, raw.Field, fkHolder, noIndex, false)
}

// parseRelationTypeValue parses the map form's `Type:` value - a
// RelationType name, optionally suffixed `(FK)` to mark this side as the
// FK holder (only valid for RelationTypeOneToOne, matching the compact
// form's `(FK--)`).
func parseRelationTypeValue(s string) (RelationType, bool, error) {
	fkHolder := strings.HasSuffix(s, "(FK)")
	if fkHolder {
		s = strings.TrimSuffix(s, "(FK)")
	}

	t := RelationType(s)
	if !knownRelationTypes[t] {
		return "", false, fmt.Errorf("unknown relation type %q", s)
	}
	if fkHolder && t != RelationTypeOneToOne {
		return "", false, fmt.Errorf("the FK marker is only valid for %q, not %q", RelationTypeOneToOne, t)
	}
	return t, fkHolder, nil
}

// MarshalYAML writes Relation back to its on-disk shape - the compact
// single-line form if that's how it was parsed (see marshalCompactForm),
// the map form otherwise (including for a Relation built by hand rather
// than parsed).
func (r Relation) MarshalYAML() (any, error) {
	if r.compactForm {
		return r.marshalCompactForm()
	}
	return r.marshalMapForm()
}

// marshalCompactForm writes r as `(TYPE) RelatedModel[.relatedAttribute]
// [no-index]`.
func (r Relation) marshalCompactForm() (any, error) {
	token, err := relationToken(r.Type, r.FkHolder)
	if err != nil {
		return nil, err
	}

	target := r.RelatedModel
	if r.RelatedAttribute != "" {
		target += "." + r.RelatedAttribute
	}
	s := token + " " + target
	if r.NoIndex {
		s += " no-index"
	}
	return s, nil
}

func relationToken(t RelationType, fkHolder bool) (string, error) {
	switch {
	case t == RelationTypeOneToOne && fkHolder:
		return "(FK--)", nil
	case t == RelationTypeOneToOne:
		return "(--)", nil
	case t == RelationTypeOneToMany:
		return "(-<)", nil
	case t == RelationTypeManyToOne:
		return "(>-)", nil
	case t == RelationTypeManyToMany:
		return "(><)", nil
	default:
		return "", fmt.Errorf("unknown relation type %q", t)
	}
}

// marshalMapForm writes r as {Type, Model, Field, Index} - the map form's
// Type value is r.Type suffixed `(FK)` when r.FkHolder is set. Index is
// only written (as `false`) when r.NoIndex is set - the common case
// (indexed) stays implicit, matching how the compact form only writes
// `no-index` for the exception.
func (r Relation) marshalMapForm() (any, error) {
	if !knownRelationTypes[r.Type] {
		return nil, fmt.Errorf("unknown relation type %q", r.Type)
	}

	typeValue := string(r.Type)
	if r.FkHolder {
		typeValue += "(FK)"
	}
	out := yamlRelation{Type: typeValue, Model: r.RelatedModel, Field: r.RelatedAttribute}
	if r.NoIndex {
		indexed := false
		out.Index = &indexed
	}
	return out, nil
}

// parseRelationsNode parses a schema file's `Relations:` mapping - the
// relation counterpart of parseFieldsNode.
func parseRelationsNode(node *yaml.Node) ([]NamedRelation, error) {
	if node.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("must be a yaml mapping, got %s", yamlKindName(node.Kind))
	}

	relations := make([]NamedRelation, 0, len(node.Content)/2)
	seen := make(map[string]bool, len(node.Content)/2)
	for i := 0; i < len(node.Content); i += 2 {
		name := node.Content[i].Value
		if seen[name] {
			return nil, fmt.Errorf("duplicate relation %q", name)
		}
		seen[name] = true

		var r Relation
		if err := node.Content[i+1].Decode(&r); err != nil {
			return nil, fmt.Errorf("relation %q: %w", name, err)
		}
		relations = append(relations, NamedRelation{Name: name, Relation: r})
	}
	return relations, nil
}

// contrRelationType returns the relation type the other side of a
// declared relation is expected to have - RelationTypeOneToOne pairs with
// itself, RelationTypeOneToMany/RelationTypeManyToOne pair with each
// other, RelationTypeManyToMany pairs with itself.
func contrRelationType(t RelationType) RelationType {
	switch t {
	case RelationTypeOneToOne:
		return RelationTypeOneToOne
	case RelationTypeOneToMany:
		return RelationTypeManyToOne
	case RelationTypeManyToOne:
		return RelationTypeOneToMany
	case RelationTypeManyToMany:
		return RelationTypeManyToMany
	default:
		return ""
	}
}

// validateRelations cross-checks every non-uni relation (one that names a
// RelatedAttribute) across the whole schemas batch - a relation with no
// RelatedAttribute stands on its own and needs no counterpart (see
// Relation's doc). Checked for each such relation: the related model is
// among schemas, it declares a relation under RelatedAttribute, that
// relation's type is the correct contr-type, it points back to this exact
// model/relation, and - RelationTypeOneToOne only - exactly one of the two
// sides is marked FkHolder.
//
// Every problem found is collected - callers get the whole set of broken
// relations in one pass. Both sides of a broken pair independently detect
// and report the mismatch (matching how each side's declaration is checked
// from its own perspective) - a single broken relation can surface as two
// related errors, one anchored at each side.
func validateRelations(schemas []*ModelSchema) error {
	byName := make(map[string]*ModelSchema, len(schemas))
	for _, s := range schemas {
		byName[s.Name] = s
	}

	var errs []error
	for _, s := range schemas {
		for _, nr := range s.Relations {
			if nr.RelatedAttribute == "" {
				continue
			}
			if err := validateRelationPair(s, nr, byName); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errors.Join(errs...)
}

func validateRelationPair(s *ModelSchema, nr NamedRelation, byName map[string]*ModelSchema) error {
	label := fmt.Sprintf("%s.%s", s.Name, nr.Name)
	contrLabel := fmt.Sprintf("%s.%s", nr.RelatedModel, nr.RelatedAttribute)

	if nr.Type == RelationTypeManyToMany && nr.RelatedModel == s.Name && nr.RelatedAttribute == nr.Name {
		return fmt.Errorf("relation %s: many-to-many relation can't be its own counterpart - declare two distinct attributes for a self-referential relation instead", label)
	}

	related, ok := byName[nr.RelatedModel]
	if !ok {
		return fmt.Errorf("relation %s: related model %q not found", label, nr.RelatedModel)
	}
	contr, ok := related.RelationByName(nr.RelatedAttribute)
	if !ok {
		return fmt.Errorf("relation %s: %q has no relation %q", label, nr.RelatedModel, nr.RelatedAttribute)
	}

	wantType := contrRelationType(nr.Type)
	if contr.Type != wantType {
		return fmt.Errorf("relation %s: expects %s to be %q, got %q", label, contrLabel, wantType, contr.Type)
	}
	if contr.RelatedModel != s.Name || contr.RelatedAttribute != nr.Name {
		return fmt.Errorf("relation %s: expects %s to point back to %s, but it points to %s.%s",
			label, contrLabel, label, contr.RelatedModel, contr.RelatedAttribute)
	}

	if nr.Type == RelationTypeOneToOne && nr.FkHolder == contr.FkHolder {
		if nr.FkHolder {
			return fmt.Errorf("relation %s and %s: both sides are marked as the FK holder, expected exactly one", label, contrLabel)
		}
		return fmt.Errorf("relation %s and %s: neither side is marked as the FK holder, expected exactly one", label, contrLabel)
	}

	return nil
}
