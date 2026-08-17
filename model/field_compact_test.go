package model

import (
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"gopkg.in/yaml.v3"
)

func TestUnmarshalField_Compact_TypeOnly(t *testing.T) {
	f := mustUnmarshalField(t, "string\n")
	if f.Type != FieldTypeString || f.Required || f.Default != nil {
		t.Fatalf("f = %#v", f)
	}
}

func TestUnmarshalField_Compact_Required(t *testing.T) {
	f := mustUnmarshalField(t, "string required\n")
	if !f.Required {
		t.Fatal("expected Required = true")
	}
}

func TestUnmarshalField_Compact_StringSize(t *testing.T) {
	f := mustUnmarshalField(t, "string(255) required\n")
	if f.Size != 255 || !f.Required {
		t.Fatalf("f = %#v", f)
	}
}

func TestUnmarshalField_Compact_DecimalPrecisionScale(t *testing.T) {
	f := mustUnmarshalField(t, "decimal(10, 2) default(1.50)\n")
	if f.Precision != 10 || f.Scale != 2 {
		t.Fatalf("f = %#v", f)
	}
	d, ok := f.Default.(decimal.Decimal)
	// decimal.Decimal.String() trims trailing zeros ("1.50" -> "1.5"), so
	// compare by value, not by the literal's exact spelling.
	if !ok || !d.Equal(decimal.NewFromFloat(1.5)) {
		t.Fatalf("Default = %#v", f.Default)
	}
}

func TestUnmarshalField_Compact_BoolDefault(t *testing.T) {
	f := mustUnmarshalField(t, "bool default(false)\n")
	if f.Default != false {
		t.Fatalf("Default = %#v, want false", f.Default)
	}
}

func TestUnmarshalField_Compact_IntDefault(t *testing.T) {
	f := mustUnmarshalField(t, "int default(42)\n")
	if f.Default != int64(42) {
		t.Fatalf("Default = %#v, want int64(42)", f.Default)
	}
}

func TestUnmarshalField_Compact_TokenOrderIndependent(t *testing.T) {
	a := mustUnmarshalField(t, "bool default(true) required\n")
	b := mustUnmarshalField(t, "bool required default(true)\n")
	if !a.IsEqual(b) || a.Required != b.Required {
		t.Fatalf("a = %#v, b = %#v, want the same regardless of token order", a, b)
	}
}

func TestUnmarshalField_Compact_QuotedStringDefault(t *testing.T) {
	f := mustUnmarshalField(t, "string default('hello world')\n")
	if f.Default != "hello world" {
		t.Fatalf("Default = %#v, want \"hello world\"", f.Default)
	}
}

func TestUnmarshalField_Compact_QuotedDefaultWithEscapedQuote(t *testing.T) {
	f := mustUnmarshalField(t, "string default('it''s ok')\n")
	if f.Default != "it's ok" {
		t.Fatalf("Default = %#v, want \"it's ok\"", f.Default)
	}
}

func TestUnmarshalField_Compact_QuotedDefaultContainingRequiredWord(t *testing.T) {
	f := mustUnmarshalField(t, "string default('is required') required\n")
	if f.Default != "is required" {
		t.Fatalf("Default = %#v, want \"is required\"", f.Default)
	}
	if !f.Required {
		t.Fatal("expected Required = true - the word inside the quoted default must not have consumed the real modifier")
	}
}

func TestUnmarshalField_Compact_Date(t *testing.T) {
	f := mustUnmarshalField(t, "date default(2026-01-02)\n")
	if f.Default != "2026-01-02" {
		t.Fatalf("Default = %#v", f.Default)
	}
}

func TestUnmarshalField_Compact_Time(t *testing.T) {
	f := mustUnmarshalField(t, "time default(15:04:05)\n")
	if f.Default != "15:04:05" {
		t.Fatalf("Default = %#v", f.Default)
	}
}

func TestUnmarshalField_Compact_DateTime(t *testing.T) {
	f := mustUnmarshalField(t, "datetime default(2026-01-02T15:04:05Z)\n")
	tm, ok := f.Default.(time.Time)
	if !ok || !tm.Equal(time.Date(2026, 1, 2, 15, 4, 5, 0, time.UTC)) {
		t.Fatalf("Default = %#v", f.Default)
	}
}

func TestUnmarshalField_Compact_Interval(t *testing.T) {
	f := mustUnmarshalField(t, "interval default(1h30m)\n")
	if f.Default != 90*time.Minute {
		t.Fatalf("Default = %#v", f.Default)
	}
}

func TestUnmarshalField_Compact_DictDefaultAsQuotedJSON(t *testing.T) {
	f := mustUnmarshalField(t, `dict default('{"count":1}')`+"\n")
	m, ok := f.Default.(map[string]any)
	if !ok {
		t.Fatalf("Default type = %T, want map[string]any", f.Default)
	}
	if m["count"] != float64(1) {
		t.Fatalf("Default[\"count\"] = %#v, want float64(1) (json numbers decode as float64)", m["count"])
	}
}

func TestUnmarshalField_Compact_DictWithoutDefault(t *testing.T) {
	f := mustUnmarshalField(t, "dict required\n")
	if f.Default != nil || !f.Required {
		t.Fatalf("f = %#v", f)
	}
}

func TestUnmarshalField_Compact_UnknownType(t *testing.T) {
	if _, err := unmarshalField(t, "not_a_type required\n"); err == nil {
		t.Fatal("expected an error for an unknown compact field type")
	}
}

func TestUnmarshalField_Compact_UnterminatedQuote(t *testing.T) {
	if _, err := unmarshalField(t, "string default('unterminated\n"); err == nil {
		t.Fatal("expected an error for an unterminated quoted default")
	}
}

func TestUnmarshalField_Compact_TrailingGarbage(t *testing.T) {
	if _, err := unmarshalField(t, "string bogus\n"); err == nil {
		t.Fatal("expected an error for an unrecognized trailing token")
	}
}

func TestUnmarshalField_Compact_InvalidDefaultGoesThroughSharedValidation(t *testing.T) {
	if _, err := unmarshalField(t, "bool default(nope)\n"); err == nil {
		t.Fatal("expected an error: \"nope\" is not true/false")
	}
	if _, err := unmarshalField(t, "int default(4.2)\n"); err == nil {
		t.Fatal("expected an error: \"4.2\" is not an integer")
	}
	if _, err := unmarshalField(t, "string(3) default(hello)\n"); err == nil {
		t.Fatal("expected an error: default exceeds size, same as the map form")
	}
}

func TestUnmarshalField_Compact_IrrelevantDetailsRejected(t *testing.T) {
	if _, err := unmarshalField(t, "int(5)\n"); err == nil {
		t.Fatal("expected an error: parenthesized details are only meaningful for string/decimal")
	}
}

func TestUnmarshalField_Compact_EmptyParensRejected(t *testing.T) {
	if _, err := unmarshalField(t, "string()\n"); err == nil {
		t.Fatal("expected an error: string() is empty details, not \"no limit\"")
	}
	if _, err := unmarshalField(t, "decimal()\n"); err == nil {
		t.Fatal("expected an error: decimal() is empty details, not \"no limit\"")
	}
}

func TestUnmarshalField_Compact_NegativeDetailsRejected(t *testing.T) {
	if _, err := unmarshalField(t, "string(-5)\n"); err == nil {
		t.Fatal("expected an error: negative size")
	}
	if _, err := unmarshalField(t, "decimal(-5, 2)\n"); err == nil {
		t.Fatal("expected an error: negative precision")
	}
}

func TestUnmarshalField_Compact_UnquotedWhitespaceRejected(t *testing.T) {
	if _, err := unmarshalField(t, "string default(hello world)\n"); err == nil {
		t.Fatal("expected an error: an unquoted default may not contain whitespace, it must be quoted")
	}
}

func TestUnmarshalField_Compact_UnquotedDefaultSurroundingWhitespaceTrimmed(t *testing.T) {
	f := mustUnmarshalField(t, "int default( 42 )\n")
	if f.Default != int64(42) {
		t.Fatalf("Default = %#v, want int64(42) (surrounding whitespace inside the parens should be trimmed)", f.Default)
	}
}

func TestUnmarshalField_Compact_QuotedDefaultContainingParens(t *testing.T) {
	f := mustUnmarshalField(t, "string default('f(x) = y')\n")
	if f.Default != "f(x) = y" {
		t.Fatalf("Default = %#v, want \"f(x) = y\"", f.Default)
	}
}

// TestUnmarshalField_Compact_BoolIntFloatMatchMapForm locks in the fix for a
// real divergence: compact-form bool/int/float defaults used to be
// hand-parsed via strconv, accepting/rejecting a different set of spellings
// than the map form's yaml decoding of the same text - e.g. "True" (a
// yaml.v3-recognized bool spelling) was accepted by the map form but
// rejected by the compact form. Both now decode the literal as yaml, so
// they must agree.
func TestUnmarshalField_Compact_BoolIntFloatMatchMapForm(t *testing.T) {
	cases := []struct {
		yamlType string
		literal  string
	}{
		{"bool", "True"},
		{"int", "0x2A"},
		{"int", "1_000"},
	}
	for _, c := range cases {
		t.Run(c.yamlType+"/"+c.literal, func(t *testing.T) {
			compact, compactErr := unmarshalField(t, c.yamlType+" default("+c.literal+")\n")
			mapForm, mapErr := unmarshalField(t, "Type: "+c.yamlType+"\nDefault: "+c.literal+"\n")

			if (compactErr == nil) != (mapErr == nil) {
				t.Fatalf("compact err = %v, map err = %v - both forms must agree on validity", compactErr, mapErr)
			}
			if compactErr == nil && !compact.IsEqual(mapForm) {
				t.Fatalf("compact = %#v, map = %#v - both forms must parse to the same value", compact, mapForm)
			}
		})
	}
}

func TestUnmarshalField_Compact_ReadonlyNotSupported(t *testing.T) {
	// The readonly modifier isn't implemented, so it must be rejected as
	// unrecognized trailing content, not silently accepted or ignored.
	if _, err := unmarshalField(t, "bool readonly\n"); err == nil {
		t.Fatal("expected an error: readonly is not a recognized compact-form modifier")
	}
}

// TestReadmeCompactExamples_Parse locks the README's per-type compact-form
// examples to what the parser actually accepts, so the table can't drift
// from real behavior. Each entry is the value side of the table's example
// (the field name before the ":" is documentation flavor, not part of the
// grammar).
func TestReadmeCompactExamples_Parse(t *testing.T) {
	cases := []string{
		"string(255) required default('unnamed')",
		"int default(0)",
		"float default(0.5)",
		"decimal(10, 2) default(19.99)",
		"bool default(false)",
		"date default(2000-01-01)",
		"time default(09:00:00)",
		"datetime default(2026-01-02T15:04:05Z)",
		"interval default(1h30m)",
		`dict default('{"theme":"dark"}')`,
	}
	for _, c := range cases {
		t.Run(c, func(t *testing.T) {
			if _, err := unmarshalField(t, c+"\n"); err != nil {
				t.Fatalf("unmarshal %q: %v", c, err)
			}
		})
	}
}

// TestReadmeMapFormExample_MatchesCompact locks the README's "same defaults,
// map form" example (decimal/dict) to actually being equal to their compact
// counterparts from the table above.
func TestReadmeMapFormExample_MatchesCompact(t *testing.T) {
	compactPrice := mustUnmarshalField(t, "decimal(10, 2) default(19.99)\n")
	mapPrice := mustUnmarshalField(t, "Type: decimal\nPrecision: 10\nScale: 2\nDefault: \"19.99\"\n")
	if !compactPrice.IsEqual(mapPrice) {
		t.Fatalf("compact = %#v, map = %#v", compactPrice, mapPrice)
	}

	compactSettings := mustUnmarshalField(t, `dict default('{"theme":"dark"}')`+"\n")
	mapSettings := mustUnmarshalField(t, "Type: dict\nDefault:\n  theme: dark\n")
	if !compactSettings.IsEqual(mapSettings) {
		t.Fatalf("compact = %#v, map = %#v", compactSettings, mapSettings)
	}
}

func TestField_MarshalYAML_RoundTrip_FromCompactForm(t *testing.T) {
	cases := []string{
		"string(255) required\n",
		"int default(42)\n",
		"bool default(false)\n",
		"decimal(10, 2) default(1.50)\n",
		"datetime default(2026-01-02T15:04:05Z)\n",
		"interval default(1h30m)\n",
		"dict default('{\"a\":1}')\n",
	}
	for _, src := range cases {
		t.Run(src, func(t *testing.T) {
			original := mustUnmarshalField(t, src)

			out, err := yaml.Marshal(original)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}

			roundTripped := mustUnmarshalField(t, string(out))
			if !original.IsEqual(roundTripped) {
				t.Fatalf("round-trip changed the field: %#v -> marshaled %q -> %#v", original, out, roundTripped)
			}
		})
	}
}

// TestField_MarshalYAML_PreservesForm locks in that a Field is written back
// the same way it was read - compact stays a one-line string, map stays a
// mapping - rather than MarshalYAML always normalizing to the map form.
func TestField_MarshalYAML_PreservesForm(t *testing.T) {
	// "hi" needs no quoting (safe bareword) - MarshalYAML doesn't preserve
	// the exact quoting choice, only the compact-vs-map form itself, see
	// TestField_MarshalYAML_CompactStringDefaultQuotedWhenNeeded for a case
	// where quoting is actually required and does round-trip.
	compact := mustUnmarshalField(t, "string(10) required default('hi')\n")
	out, err := yaml.Marshal(compact)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	got := strings.TrimSpace(string(out))
	if got != "string(10) required default(hi)" {
		t.Fatalf("marshaled = %q, want the compact form back", got)
	}

	mapForm := mustUnmarshalField(t, "Type: string\nSize: 10\nRequired: true\nDefault: hi\n")
	out, err = yaml.Marshal(mapForm)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(out), "Type: string") {
		t.Fatalf("marshaled = %q, want the map form back", out)
	}
}

// TestField_MarshalYAML_CompactTokenOrderIsFixed checks the documented
// "type, then required, then default" output order, regardless of how the
// source had them.
func TestField_MarshalYAML_CompactTokenOrderIsFixed(t *testing.T) {
	f := mustUnmarshalField(t, "bool default(true) required\n")
	out, err := yaml.Marshal(f)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	got := strings.TrimSpace(string(out))
	if got != "bool required default(true)" {
		t.Fatalf("marshaled = %q, want fixed token order \"bool required default(true)\"", got)
	}
}

func TestField_MarshalYAML_CompactStringDefaultQuotedWhenNeeded(t *testing.T) {
	f := mustUnmarshalField(t, "string default('hello world')\n")
	out, err := yaml.Marshal(f)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	got := strings.TrimSpace(string(out))
	if got != "string default('hello world')" {
		t.Fatalf("marshaled = %q, want the space-containing default quoted", got)
	}
}

func TestField_MarshalYAML_CompactStringDefaultBarewordWhenSafe(t *testing.T) {
	f := mustUnmarshalField(t, "string default(unnamed)\n")
	out, err := yaml.Marshal(f)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	got := strings.TrimSpace(string(out))
	if got != "string default(unnamed)" {
		t.Fatalf("marshaled = %q, want an unquoted bareword (no need to quote it)", got)
	}
}

func TestField_MarshalYAML_CompactDictDefaultRoundTrips(t *testing.T) {
	f := mustUnmarshalField(t, `dict default('{"count":1,"name":"a b"}')`+"\n")
	out, err := yaml.Marshal(f)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	roundTripped := mustUnmarshalField(t, string(out))
	if !f.IsEqual(roundTripped) {
		t.Fatalf("round-trip changed the field: marshaled %q -> %#v", out, roundTripped)
	}
}

// TestField_MarshalYAML_CompactWithRenamedFromFallsBackToMap covers a case
// that can't happen through parsing today (the compact grammar has no
// renamedFrom token) but is cheap to guard: if something ever builds such a
// Field by hand, marshaling must not silently drop renamedFrom just because
// the compact form has nowhere to put it.
func TestField_MarshalYAML_CompactWithRenamedFromFallsBackToMap(t *testing.T) {
	f := mustUnmarshalField(t, "string required\n")
	f.RenamedFrom = "oldName"

	out, err := yaml.Marshal(f)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(out), "RenamedFrom: oldName") {
		t.Fatalf("marshaled = %q, want the map form (with renamedFrom) since the compact form can't express it", out)
	}
}
