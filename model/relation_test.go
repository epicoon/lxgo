package model

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func unmarshalRelation(t *testing.T, src string) (Relation, error) {
	t.Helper()
	var r Relation
	err := yaml.Unmarshal([]byte(src), &r)
	return r, err
}

func mustUnmarshalRelation(t *testing.T, src string) Relation {
	t.Helper()
	r, err := unmarshalRelation(t, src)
	if err != nil {
		t.Fatalf("unmarshal %q: %v", src, err)
	}
	return r
}

func TestUnmarshalRelation_OneToOne(t *testing.T) {
	r := mustUnmarshalRelation(t, "(--) Profile.user\n")
	if r.Type != RelationTypeOneToOne || r.RelatedModel != "Profile" || r.RelatedAttribute != "user" || r.FkHolder {
		t.Fatalf("r = %#v", r)
	}
}

func TestUnmarshalRelation_OneToOneFkHolder(t *testing.T) {
	r := mustUnmarshalRelation(t, "(FK--) Profile.user\n")
	if r.Type != RelationTypeOneToOne || !r.FkHolder {
		t.Fatalf("r = %#v", r)
	}
}

func TestUnmarshalRelation_OneToMany(t *testing.T) {
	r := mustUnmarshalRelation(t, "(-<) GamerInGame.gameSave\n")
	if r.Type != RelationTypeOneToMany || r.RelatedModel != "GamerInGame" || r.RelatedAttribute != "gameSave" {
		t.Fatalf("r = %#v", r)
	}
}

func TestUnmarshalRelation_ManyToOne(t *testing.T) {
	r := mustUnmarshalRelation(t, "(>-) GameSave.gamers\n")
	if r.Type != RelationTypeManyToOne || r.RelatedModel != "GameSave" || r.RelatedAttribute != "gamers" {
		t.Fatalf("r = %#v", r)
	}
}

func TestUnmarshalRelation_ManyToMany(t *testing.T) {
	r := mustUnmarshalRelation(t, "(><) Tag.words\n")
	if r.Type != RelationTypeManyToMany || r.RelatedModel != "Tag" || r.RelatedAttribute != "words" {
		t.Fatalf("r = %#v", r)
	}
}

func TestUnmarshalRelation_NoIndex_ManyToOne(t *testing.T) {
	r := mustUnmarshalRelation(t, "(>-) GameSave.gamers no-index\n")
	if r.Type != RelationTypeManyToOne || !r.NoIndex {
		t.Fatalf("r = %#v", r)
	}
}

func TestUnmarshalRelation_NoIndex_ManyToMany(t *testing.T) {
	r := mustUnmarshalRelation(t, "(><) Tag.words no-index\n")
	if r.Type != RelationTypeManyToMany || !r.NoIndex {
		t.Fatalf("r = %#v", r)
	}
}

func TestUnmarshalRelation_NoIndexOnOneToOneRejected(t *testing.T) {
	// oneToOne's FK-holding column is backed by a UNIQUE constraint, which
	// always carries its own index in Postgres - no-index could never
	// actually be honored, so it's a hard error rather than a silent no-op,
	// on either side (FkHolder or not).
	for _, tok := range []string{"(--)", "(FK--)"} {
		if _, err := unmarshalRelation(t, tok+" Profile.user no-index\n"); err == nil {
			t.Fatalf("%s: expected an error: no-index is not valid for oneToOne", tok)
		}
	}
}

func TestUnmarshalRelation_NoIndexOnOneToManyRejected(t *testing.T) {
	// oneToMany's side has no physical FK column of its own to index.
	if _, err := unmarshalRelation(t, "(-<) GamerInGame.gameSave no-index\n"); err == nil {
		t.Fatal("expected an error: no-index is not valid for oneToMany")
	}
}

func TestUnmarshalRelation_UnknownModifierRejected(t *testing.T) {
	if _, err := unmarshalRelation(t, "(>-) GameSave.gamers bogus\n"); err == nil {
		t.Fatal("expected an error for an unrecognized third token")
	}
}

func TestUnmarshalRelation_UniOneToOne(t *testing.T) {
	r := mustUnmarshalRelation(t, "(--) Profile\n")
	if r.RelatedAttribute != "" {
		t.Fatalf("r = %#v, want no RelatedAttribute", r)
	}
}

func TestUnmarshalRelation_UniManyToOne(t *testing.T) {
	r := mustUnmarshalRelation(t, "(>-) Client\n")
	if r.Type != RelationTypeManyToOne || r.RelatedAttribute != "" {
		t.Fatalf("r = %#v", r)
	}
}

func TestUnmarshalRelation_UniOneToManyRejected(t *testing.T) {
	if _, err := unmarshalRelation(t, "(-<) GamerInGame\n"); err == nil {
		t.Fatal("expected an error: oneToMany must name the related attribute")
	}
}

func TestUnmarshalRelation_UniManyToManyRejected(t *testing.T) {
	if _, err := unmarshalRelation(t, "(><) Tag\n"); err == nil {
		t.Fatal("expected an error: manyToMany must name the related attribute")
	}
}

func TestUnmarshalRelation_UnknownTokenRejected(t *testing.T) {
	if _, err := unmarshalRelation(t, "(-->) X.y\n"); err == nil {
		t.Fatal("expected an error for an unknown relation type token")
	}
}

func TestUnmarshalRelation_FkMarkerOnWrongTypeRejected(t *testing.T) {
	// The FK marker is only meaningful for oneToOne - manyToOne is always
	// the FK holder regardless (see Relation's doc), oneToMany/manyToMany
	// aren't about a single FK column at all.
	for _, tok := range []string{"(FK-<)", "(FK>-)", "(FK><)"} {
		if _, err := unmarshalRelation(t, tok+" X.y\n"); err == nil {
			t.Fatalf("%s: expected an error: FK marker only valid on oneToOne", tok)
		}
	}
}

func TestUnmarshalRelation_LowercaseFkMarkerNotSupported(t *testing.T) {
	// Only the uppercase "FK" spelling is recognized - matching this
	// package's other protocol keys, all capitalized.
	if _, err := unmarshalRelation(t, "(fk--) X.y\n"); err == nil {
		t.Fatal("expected an error: lowercase fk is not a supported token")
	}
}

func TestUnmarshalRelation_SuffixFkMarkerNotSupported(t *testing.T) {
	// Only the leading "(FK--)" form marks the FK holder - a trailing
	// marker like "(--FK)" is just an unknown token, not an alternative
	// spelling of the same thing (see Relation's doc).
	if _, err := unmarshalRelation(t, "(--FK) X.y\n"); err == nil {
		t.Fatal("expected an error: (--FK) is not a supported token")
	}
}

func TestUnmarshalRelation_MissingTargetRejected(t *testing.T) {
	if _, err := unmarshalRelation(t, "(--)\n"); err == nil {
		t.Fatal("expected an error for a relation with no target at all")
	}
}

func TestUnmarshalRelation_SequenceRejected(t *testing.T) {
	if _, err := unmarshalRelation(t, "[a, b]\n"); err == nil {
		t.Fatal("expected an error: a relation must be a compact string or a yaml mapping, not a list")
	}
}

func TestUnmarshalRelation_BareTokenRejected(t *testing.T) {
	// Parentheses around the type token are mandatory, not decorative -
	// unlike Field's compact form, a bare token isn't a recognized one.
	for _, tok := range []string{"--", "FK--", "-<", ">-", "><"} {
		if _, err := unmarshalRelation(t, tok+" X.y\n"); err == nil {
			t.Fatalf("%s: expected an error for a token without parentheses", tok)
		}
	}
}

func TestRelation_MarshalYAML_RoundTrip_CompactForm(t *testing.T) {
	cases := []string{
		"(--) Profile.user",
		"(FK--) Profile.user",
		"(-<) GamerInGame.gameSave",
		"(>-) GameSave.gamers",
		"(><) Tag.words",
		"(--) Profile",
		"(>-) Client",
		"(>-) GameSave.gamers no-index",
		"(><) Tag.words no-index",
	}
	for _, src := range cases {
		r := mustUnmarshalRelation(t, src+"\n")
		out, err := yaml.Marshal(r)
		if err != nil {
			t.Fatalf("Marshal(%q): %v", src, err)
		}
		got, err := unmarshalRelation(t, string(out))
		if err != nil {
			t.Fatalf("re-unmarshal %q (from %q): %v", out, src, err)
		}
		if got != r {
			t.Fatalf("round-trip %q: got %#v, want %#v", src, got, r)
		}
	}
}

func TestUnmarshalRelation_MapForm(t *testing.T) {
	r := mustUnmarshalRelation(t, "Type: ManyToMany\nModel: ContrModelName\nField: ContrFieldName\n")
	if r.Type != RelationTypeManyToMany || r.RelatedModel != "ContrModelName" || r.RelatedAttribute != "ContrFieldName" || r.FkHolder {
		t.Fatalf("r = %#v", r)
	}
}

func TestUnmarshalRelation_MapForm_FkHolder(t *testing.T) {
	r := mustUnmarshalRelation(t, "Type: OneToOne(FK)\nModel: Profile\nField: user\n")
	if r.Type != RelationTypeOneToOne || !r.FkHolder {
		t.Fatalf("r = %#v", r)
	}
}

func TestUnmarshalRelation_MapForm_Uni(t *testing.T) {
	r := mustUnmarshalRelation(t, "Type: ManyToOne\nModel: Client\n")
	if r.Type != RelationTypeManyToOne || r.RelatedAttribute != "" {
		t.Fatalf("r = %#v", r)
	}
}

func TestUnmarshalRelation_MapForm_UniOneToManyRejected(t *testing.T) {
	if _, err := unmarshalRelation(t, "Type: OneToMany\nModel: GamerInGame\n"); err == nil {
		t.Fatal("expected an error: oneToMany must name the related attribute")
	}
}

func TestUnmarshalRelation_MapForm_UnknownTypeRejected(t *testing.T) {
	if _, err := unmarshalRelation(t, "Type: OneToOneToOne\nModel: X\nField: y\n"); err == nil {
		t.Fatal("expected an error for an unknown relation type")
	}
}

func TestUnmarshalRelation_MapForm_FkOnWrongTypeRejected(t *testing.T) {
	if _, err := unmarshalRelation(t, "Type: ManyToOne(FK)\nModel: X\nField: y\n"); err == nil {
		t.Fatal("expected an error: FK marker only valid on OneToOne")
	}
}

func TestUnmarshalRelation_MapForm_NoIndex(t *testing.T) {
	r := mustUnmarshalRelation(t, "Type: ManyToOne\nModel: GameSave\nField: gamers\nIndex: false\n")
	if r.Type != RelationTypeManyToOne || !r.NoIndex {
		t.Fatalf("r = %#v", r)
	}
}

func TestUnmarshalRelation_MapForm_IndexTrueIsSameAsAbsent(t *testing.T) {
	r := mustUnmarshalRelation(t, "Type: ManyToOne\nModel: GameSave\nField: gamers\nIndex: true\n")
	if r.NoIndex {
		t.Fatalf("r = %#v, want NoIndex false", r)
	}
}

func TestUnmarshalRelation_MapForm_NoIndexOnOneToOneRejected(t *testing.T) {
	if _, err := unmarshalRelation(t, "Type: OneToOne(FK)\nModel: Profile\nField: user\nIndex: false\n"); err == nil {
		t.Fatal("expected an error: no-index is not valid for oneToOne")
	}
}

func TestUnmarshalRelation_MapForm_NoIndexOnOneToManyRejected(t *testing.T) {
	if _, err := unmarshalRelation(t, "Type: OneToMany\nModel: GamerInGame\nField: gameSave\nIndex: false\n"); err == nil {
		t.Fatal("expected an error: no-index is not valid for oneToMany")
	}
}

func TestUnmarshalRelation_MapForm_UnknownKeyRejected(t *testing.T) {
	if _, err := unmarshalRelation(t, "Type: OneToOne\nModle: X\n"); err == nil {
		t.Fatal("expected an error for a typo'd key")
	}
}

func TestUnmarshalRelation_MapForm_MissingModelRejected(t *testing.T) {
	if _, err := unmarshalRelation(t, "Type: OneToOne\n"); err == nil {
		t.Fatal("expected an error: missing related model")
	}
}

func TestRelation_MarshalYAML_RoundTrip_MapForm(t *testing.T) {
	cases := []string{
		"Type: OneToOne\nModel: Profile\nField: user\n",
		"Type: OneToOne(FK)\nModel: Profile\nField: user\n",
		"Type: OneToMany\nModel: GamerInGame\nField: gameSave\n",
		"Type: ManyToOne\nModel: GameSave\nField: gamers\n",
		"Type: ManyToMany\nModel: Tag\nField: words\n",
		"Type: OneToOne\nModel: Profile\n",
		"Type: ManyToOne\nModel: Client\n",
		"Type: ManyToOne\nModel: GameSave\nField: gamers\nIndex: false\n",
		"Type: ManyToMany\nModel: Tag\nField: words\nIndex: false\n",
	}
	for _, src := range cases {
		r := mustUnmarshalRelation(t, src)
		out, err := yaml.Marshal(r)
		if err != nil {
			t.Fatalf("Marshal(%q): %v", src, err)
		}
		got, err := unmarshalRelation(t, string(out))
		if err != nil {
			t.Fatalf("re-unmarshal %q (from %q): %v", out, src, err)
		}
		if got != r {
			t.Fatalf("round-trip %q: got %#v, want %#v", src, got, r)
		}
	}
}

func TestRelation_MarshalYAML_HandBuiltWritesMapForm(t *testing.T) {
	// A programmatically-built Relation (not parsed) writes as the map
	// form, same reasoning as Field's own compactForm default.
	r := Relation{Type: RelationTypeManyToOne, RelatedModel: "GameSave", RelatedAttribute: "gamers"}
	out, err := yaml.Marshal(r)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var raw map[string]any
	if err := yaml.Unmarshal(out, &raw); err != nil {
		t.Fatalf("re-unmarshal as a mapping: %v (output was %q)", err, out)
	}
	if raw["Type"] != "ManyToOne" || raw["Model"] != "GameSave" || raw["Field"] != "gamers" {
		t.Fatalf("raw = %#v", raw)
	}
}

func TestContrRelationType(t *testing.T) {
	cases := map[RelationType]RelationType{
		RelationTypeOneToOne:   RelationTypeOneToOne,
		RelationTypeOneToMany:  RelationTypeManyToOne,
		RelationTypeManyToOne:  RelationTypeOneToMany,
		RelationTypeManyToMany: RelationTypeManyToMany,
	}
	for in, want := range cases {
		if got := contrRelationType(in); got != want {
			t.Errorf("contrRelationType(%q) = %q, want %q", in, got, want)
		}
	}
}

// schemaWithRelation builds a minimal *ModelSchema for validateRelations
// tests - name is the model, relName/relDef declare its one relation
// (parsed the same way LoadModelSchema would parse it).
func schemaWithRelation(t *testing.T, name, relName, relDef string) *ModelSchema {
	t.Helper()
	r, err := parseRelationString(relDef)
	if err != nil {
		t.Fatalf("parseRelationString(%q): %v", relDef, err)
	}
	return &ModelSchema{Name: name, Relations: []NamedRelation{{Name: relName, Relation: r}}}
}

func TestValidateRelations_ValidPairs(t *testing.T) {
	cases := []struct {
		name        string
		aName, aDef string
		bName, bDef string
	}{
		{"oneToOne", "user", "(FK--) B.profile", "profile", "(--) A.user"},
		{"oneToMany_manyToOne", "gamers", "(-<) B.gameSave", "gameSave", "(>-) A.gamers"},
		{"manyToMany", "words", "(><) B.tags", "tags", "(><) A.words"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a := schemaWithRelation(t, "A", c.aName, c.aDef)
			b := schemaWithRelation(t, "B", c.bName, c.bDef)
			if err := validateRelations([]*ModelSchema{a, b}); err != nil {
				t.Fatalf("validateRelations: %v", err)
			}
		})
	}
}

func TestValidateRelations_UniIgnored(t *testing.T) {
	a := schemaWithRelation(t, "A", "profile", "(--) B")
	if err := validateRelations([]*ModelSchema{a}); err != nil {
		t.Fatalf("validateRelations: %v", err)
	}
}

func TestValidateRelations_RelatedModelNotFound(t *testing.T) {
	a := schemaWithRelation(t, "A", "gamers", "(-<) Ghost.a")
	if err := validateRelations([]*ModelSchema{a}); err == nil {
		t.Fatal("expected an error: related model doesn't exist")
	}
}

func TestValidateRelations_RelatedAttributeNotDeclared(t *testing.T) {
	a := schemaWithRelation(t, "A", "gamers", "(-<) B.gameSave")
	b := &ModelSchema{Name: "B"}
	if err := validateRelations([]*ModelSchema{a, b}); err == nil {
		t.Fatal("expected an error: B doesn't declare gameSave at all")
	}
}

func TestValidateRelations_MismatchedContrType(t *testing.T) {
	a := schemaWithRelation(t, "A", "gamers", "(-<) B.gameSave")
	// B declares its side as oneToMany too, but the correct contr-type for
	// A's oneToMany is manyToOne.
	b := schemaWithRelation(t, "B", "gameSave", "(-<) A.gamers")
	if err := validateRelations([]*ModelSchema{a, b}); err == nil {
		t.Fatal("expected an error: mismatched contr-type")
	}
}

func TestValidateRelations_DoesNotPointBack(t *testing.T) {
	a := schemaWithRelation(t, "A", "gamers", "(-<) B.gameSave")
	// B's relation names a different model/attribute than A.gamers.
	b := schemaWithRelation(t, "B", "gameSave", "(>-) C.somethingElse")
	c := &ModelSchema{Name: "C"}
	if err := validateRelations([]*ModelSchema{a, b, c}); err == nil {
		t.Fatal("expected an error: B doesn't point back to A.gamers")
	}
}

func TestValidateRelations_FkHolderConflict_BothMarked(t *testing.T) {
	a := schemaWithRelation(t, "A", "profile", "(FK--) B.user")
	b := schemaWithRelation(t, "B", "user", "(FK--) A.profile")
	if err := validateRelations([]*ModelSchema{a, b}); err == nil {
		t.Fatal("expected an error: both sides marked as FK holder")
	}
}

func TestValidateRelations_FkHolderConflict_NeitherMarked(t *testing.T) {
	a := schemaWithRelation(t, "A", "profile", "(--) B.user")
	b := schemaWithRelation(t, "B", "user", "(--) A.profile")
	if err := validateRelations([]*ModelSchema{a, b}); err == nil {
		t.Fatal("expected an error: neither side marked as FK holder")
	}
}

// TestValidateRelations_ManyToManySelfCounterpart checks that a
// RelationTypeManyToMany relation declared as its own counterpart
// (RelatedModel/RelatedAttribute both naming this exact declaration) is
// rejected - its join table would need two distinct FK column names, but
// there's only one (model, attribute) pair to compute them from (see
// pgJoinColumnName), so both sides would collide. A genuine
// self-referential relation under two distinct attribute names on the
// same model (e.g. "friends"/"friendOf") is unaffected by this check -
// its two attribute names still give pgJoinColumnName something to tell
// the columns apart by.
func TestValidateRelations_ManyToManySelfCounterpart(t *testing.T) {
	user := schemaWithRelation(t, "User", "siblings", "(><) User.siblings")
	if err := validateRelations([]*ModelSchema{user}); err == nil {
		t.Fatal("expected an error: a many-to-many relation can't be its own counterpart")
	}
}

func TestValidateRelations_CollectsAllErrors(t *testing.T) {
	a := schemaWithRelation(t, "A", "gamers", "(-<) Ghost1.a")
	b := schemaWithRelation(t, "B", "gameSave", "(-<) Ghost2.b")

	err := validateRelations([]*ModelSchema{a, b})
	if err == nil {
		t.Fatal("expected an error")
	}
	unwrapped, ok := err.(interface{ Unwrap() []error })
	if !ok {
		t.Fatalf("expected a joined error (errors.Join), got %T: %v", err, err)
	}
	if len(unwrapped.Unwrap()) != 2 {
		t.Fatalf("expected both problems reported together, got %d: %v", len(unwrapped.Unwrap()), err)
	}
}
