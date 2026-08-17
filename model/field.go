// Package model describes database models as yaml schemas and generates
// migrations from the difference between a declared schema and the live
// database - see the package README for the full workflow. This has no
// relation to lxgo/jspp's `lx.Model`/`lx.ModelSchema` (client-side JS
// binding, no database involved) despite the similar name.
package model

import (
	"encoding/json"
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/shopspring/decimal"
	"gopkg.in/yaml.v3"
)

// FieldType is a model field's declared type - see Field.
type FieldType string

const (
	FieldTypeString   FieldType = "string"
	FieldTypeInt      FieldType = "int"
	FieldTypeFloat    FieldType = "float"
	FieldTypeDecimal  FieldType = "decimal"
	FieldTypeBool     FieldType = "bool"
	FieldTypeDate     FieldType = "date"
	FieldTypeTime     FieldType = "time"
	FieldTypeDateTime FieldType = "datetime"
	FieldTypeInterval FieldType = "interval"
	FieldTypeDict     FieldType = "dict"
)

// dateLayout/timeLayout are the Go layouts date/time literals must match -
// sortable and unambiguous. datetime uses time.RFC3339 instead (a timezone
// offset is mandatory - see parseDefault).
const (
	dateLayout = "2006-01-02"
	timeLayout = "15:04:05"
)

var knownFieldTypes = map[FieldType]bool{
	FieldTypeString:   true,
	FieldTypeInt:      true,
	FieldTypeFloat:    true,
	FieldTypeDecimal:  true,
	FieldTypeBool:     true,
	FieldTypeDate:     true,
	FieldTypeTime:     true,
	FieldTypeDateTime: true,
	FieldTypeInterval: true,
	FieldTypeDict:     true,
}

// Field is one model attribute's schema declaration: its type, whether it's
// required, an optional default value and type-specific details (Size for
// FieldTypeString, Precision/Scale for FieldTypeDecimal - ignored for every
// other type). Default's Go representation depends on Type:
//
//   - FieldTypeString: string
//   - FieldTypeInt: int64
//   - FieldTypeFloat: float64
//   - FieldTypeDecimal: decimal.Decimal
//   - FieldTypeBool: bool
//   - FieldTypeDate, FieldTypeTime: string (validated against dateLayout/timeLayout)
//   - FieldTypeDateTime: time.Time, normalized to UTC
//   - FieldTypeInterval: time.Duration
//   - FieldTypeDict: map[string]any or []any
//
// Construct a Field by unmarshaling yaml (UnmarshalYAML validates Type and
// Default together) - the zero value is not a valid Field.
type Field struct {
	Type        FieldType
	Required    bool
	Default     any
	RenamedFrom string
	Size        int
	Precision   int
	Scale       int

	// compactForm records whether this Field was parsed from the compact
	// single-line form rather than the map form - MarshalYAML writes it
	// back the same way it was read, see field_compact.go. A
	// programmatically-built Field (not parsed) writes as the map form.
	compactForm bool
}

// IsEqual reports whether f and other declare the same field, ignoring
// their name (compared separately by the caller - see the model comparator)
// and RenamedFrom (rename-hint metadata, not part of the field's
// definition - a DB-introspected Field never sets it, so including it here
// would make any explicitly-marked rename look like a spurious "changed").
func (f Field) IsEqual(other Field) bool {
	if f.Type != other.Type || f.Required != other.Required {
		return false
	}
	if f.Size != other.Size || f.Precision != other.Precision || f.Scale != other.Scale {
		return false
	}
	return defaultsEqual(f.Type, f.Default, other.Default)
}

func defaultsEqual(t FieldType, a, b any) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}

	switch t {
	case FieldTypeDecimal:
		da, aok := a.(decimal.Decimal)
		db, bok := b.(decimal.Decimal)
		return aok && bok && da.Equal(db)
	case FieldTypeDateTime:
		ta, aok := a.(time.Time)
		tb, bok := b.(time.Time)
		return aok && bok && ta.Equal(tb)
	case FieldTypeDict:
		return jsonEqual(a, b)
	default:
		return a == b
	}
}

func jsonEqual(a, b any) bool {
	ja, errA := json.Marshal(a)
	jb, errB := json.Marshal(b)
	if errA != nil || errB != nil {
		return false
	}
	return string(ja) == string(jb)
}

// yamlField mirrors Field's on-disk shape - kept separate from Field so
// Default can be decoded lazily (its expected Go type depends on Type,
// decided only once Type itself is known).
type yamlField struct {
	Type        FieldType `yaml:"Type"`
	Required    bool      `yaml:"Required,omitempty"`
	Default     yaml.Node `yaml:"Default,omitempty"`
	RenamedFrom string    `yaml:"RenamedFrom,omitempty"`
	Size        int       `yaml:"Size,omitempty"`
	Precision   int       `yaml:"Precision,omitempty"`
	Scale       int       `yaml:"Scale,omitempty"`
}

// allowedFieldKeys are yamlField's yaml keys - checked explicitly because
// yaml.Node.Decode (unlike a top-level yaml.Decoder) has no strict/
// known-fields mode of its own, and a silently-ignored typo (`Siez` instead
// of `Size`) would drop a constraint without a trace.
var allowedFieldKeys = map[string]bool{
	"Type": true, "Required": true, "Default": true,
	"RenamedFrom": true, "Size": true, "Precision": true, "Scale": true,
}

// validateFieldShape enforces that t is a known type, that size/precision/
// scale are only given for the type they're meaningful for (size for
// FieldTypeString, precision/scale for FieldTypeDecimal), and that none of
// them are negative - shared by the map form's UnmarshalYAML and by the
// system types table's write path (system_types.go), so a record written
// there can never disagree with what a schema file itself is allowed to
// declare.
func validateFieldShape(t FieldType, size, precision, scale int) error {
	if !knownFieldTypes[t] {
		return fmt.Errorf("unknown field type %q", t)
	}
	if t != FieldTypeString && size != 0 {
		return fmt.Errorf("size is only meaningful for type %q, not %q", FieldTypeString, t)
	}
	if t != FieldTypeDecimal && (precision != 0 || scale != 0) {
		return fmt.Errorf("precision/scale are only meaningful for type %q, not %q", FieldTypeDecimal, t)
	}
	if size < 0 {
		return fmt.Errorf("size must not be negative, got %d", size)
	}
	if precision < 0 {
		return fmt.Errorf("precision must not be negative, got %d", precision)
	}
	if scale < 0 {
		return fmt.Errorf("scale must not be negative, got %d", scale)
	}
	return nil
}

// UnmarshalYAML parses a Field from either of two forms a schema author can
// write: the structured map form (this function) or the compact single-line
// form (`<type>[(<details>)] [required] [default(<literal>)]`, see
// field_compact.go) - neither is a shorthand for the other, both are
// first-class. Which one applies is decided by value's yaml.Kind.
//
// The map form validates: Type (must be a known FieldType), that no
// unknown/misspelled keys are present, that Size is only given for
// FieldTypeString and Precision/Scale only for FieldTypeDecimal, and, when
// given, that Default structurally matches Type (see the Field doc) - all
// hard parse errors, not silently accepted or deferred to whatever later
// code happens to use the value.
func (f *Field) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		return f.unmarshalCompactForm(value.Value)
	}
	if value.Kind != yaml.MappingNode {
		return fmt.Errorf("a field must be a yaml mapping or a compact string, got %s", yamlKindName(value.Kind))
	}
	for i := 0; i < len(value.Content); i += 2 {
		key := value.Content[i].Value
		if !allowedFieldKeys[key] {
			return fmt.Errorf("unknown field attribute %q", key)
		}
	}

	var raw yamlField
	if err := value.Decode(&raw); err != nil {
		return err
	}

	if err := validateFieldShape(raw.Type, raw.Size, raw.Precision, raw.Scale); err != nil {
		return err
	}

	f.Type = raw.Type
	f.Required = raw.Required
	f.RenamedFrom = raw.RenamedFrom
	f.Size = raw.Size
	f.Precision = raw.Precision
	f.Scale = raw.Scale
	f.Default = nil

	if raw.Default.Kind == 0 {
		return nil
	}

	var rawDefault any
	if err := raw.Default.Decode(&rawDefault); err != nil {
		return fmt.Errorf("invalid default: %w", err)
	}
	if rawDefault == nil {
		return nil
	}

	def, err := parseDefault(raw.Type, rawDefault, raw.Size, raw.Precision, raw.Scale)
	if err != nil {
		return fmt.Errorf("invalid default for type %q: %w", raw.Type, err)
	}
	f.Default = def
	return nil
}

// MarshalYAML writes Field back to its on-disk shape, formatting Default
// (whatever Go representation UnmarshalYAML normalized it to) back into the
// same literal form UnmarshalYAML accepts - as the compact single-line form
// if that's how it was parsed (see field_compact.go's marshalCompactForm),
// the map form otherwise (including for a Field built by hand rather than
// parsed, and for one that sets RenamedFrom - the compact form has no
// token for it, so it always falls back to the map form regardless of
// compactForm).
func (f Field) MarshalYAML() (any, error) {
	if f.compactForm && f.RenamedFrom == "" {
		return f.marshalCompactForm()
	}

	out := yamlFieldOut{
		Type:        f.Type,
		Required:    f.Required,
		RenamedFrom: f.RenamedFrom,
		Size:        f.Size,
		Precision:   f.Precision,
		Scale:       f.Scale,
	}
	if f.Default != nil {
		out.Default = formatDefault(f.Type, f.Default)
	}
	return out, nil
}

// yamlFieldOut is yamlField's output-side counterpart - Default is `any`
// here (already formatted to its literal form) rather than a lazy yaml.Node.
type yamlFieldOut struct {
	Type        FieldType `yaml:"Type"`
	Required    bool      `yaml:"Required,omitempty"`
	Default     any       `yaml:"Default,omitempty"`
	RenamedFrom string    `yaml:"RenamedFrom,omitempty"`
	Size        int       `yaml:"Size,omitempty"`
	Precision   int       `yaml:"Precision,omitempty"`
	Scale       int       `yaml:"Scale,omitempty"`
}

func formatDefault(t FieldType, v any) any {
	switch t {
	case FieldTypeDecimal:
		return v.(decimal.Decimal).String()
	case FieldTypeDateTime:
		return v.(time.Time).Format(time.RFC3339)
	case FieldTypeInterval:
		return v.(time.Duration).String()
	default:
		return v
	}
}

// parseDefault validates raw (as decoded from yaml into a plain Go value by
// gopkg.in/yaml.v3) against t, and normalizes it to the Go representation
// documented on Field.
func parseDefault(t FieldType, raw any, size, precision, scale int) (any, error) {
	switch t {
	case FieldTypeString:
		s, ok := raw.(string)
		if !ok {
			return nil, fmt.Errorf("must be a string, got %T", raw)
		}
		if size > 0 {
			if n := utf8.RuneCountInString(s); n > size {
				return nil, fmt.Errorf("is %d runes long, exceeds size %d", n, size)
			}
		}
		return s, nil

	case FieldTypeInt:
		switch v := raw.(type) {
		case int:
			return int64(v), nil
		case int64:
			return v, nil
		case uint64:
			return nil, fmt.Errorf("%d is outside int64's range", v)
		default:
			return nil, fmt.Errorf("must be an integer, got %T", raw)
		}

	case FieldTypeFloat:
		switch v := raw.(type) {
		case float64:
			return v, nil
		case int:
			return float64(v), nil
		case int64:
			return float64(v), nil
		default:
			return nil, fmt.Errorf("must be a number, got %T", raw)
		}

	case FieldTypeDecimal:
		s, ok := raw.(string)
		if !ok {
			return nil, fmt.Errorf("must be a string decimal literal, got %T", raw)
		}
		d, err := decimal.NewFromString(s)
		if err != nil {
			return nil, fmt.Errorf("invalid decimal literal %q: %w", s, err)
		}
		if err := validateDecimalDigits(d, precision, scale); err != nil {
			return nil, err
		}
		return d, nil

	case FieldTypeBool:
		b, ok := raw.(bool)
		if !ok {
			return nil, fmt.Errorf("must be a bool, got %T", raw)
		}
		return b, nil

	case FieldTypeDate:
		s, ok := raw.(string)
		if !ok {
			return nil, fmt.Errorf("must be a string, got %T", raw)
		}
		if _, err := time.Parse(dateLayout, s); err != nil {
			return nil, fmt.Errorf("invalid date %q (want layout %q): %w", s, dateLayout, err)
		}
		return s, nil

	case FieldTypeTime:
		s, ok := raw.(string)
		if !ok {
			return nil, fmt.Errorf("must be a string, got %T", raw)
		}
		if _, err := time.Parse(timeLayout, s); err != nil {
			return nil, fmt.Errorf("invalid time %q (want layout %q): %w", s, timeLayout, err)
		}
		return s, nil

	case FieldTypeDateTime:
		s, ok := raw.(string)
		if !ok {
			return nil, fmt.Errorf("must be a string, got %T", raw)
		}
		parsed, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return nil, fmt.Errorf("invalid datetime %q (want RFC3339 with an explicit offset, e.g. %q): %w", s, time.RFC3339, err)
		}
		return parsed.UTC(), nil

	case FieldTypeInterval:
		s, ok := raw.(string)
		if !ok {
			return nil, fmt.Errorf("must be a string, got %T", raw)
		}
		d, err := time.ParseDuration(s)
		if err != nil {
			return nil, fmt.Errorf("invalid interval %q: %w", s, err)
		}
		return d, nil

	case FieldTypeDict:
		switch raw.(type) {
		case map[string]any, []any:
		default:
			return nil, fmt.Errorf("must be a mapping or a list, got %T", raw)
		}
		if _, err := json.Marshal(raw); err != nil {
			return nil, fmt.Errorf("not JSON-serializable: %w", err)
		}
		return raw, nil

	default:
		return nil, fmt.Errorf("unknown field type %q", t)
	}
}

// validateDecimalDigits enforces precision (total significant digits) and
// scale (digits after the decimal point) - either left at 0 (not declared
// on the field) is treated as "no limit", not "zero allowed".
func validateDecimalDigits(d decimal.Decimal, precision, scale int) error {
	digitsAfterPoint := 0
	if exp := int(d.Exponent()); exp < 0 {
		digitsAfterPoint = -exp
	}

	if precision > 0 {
		if n := d.NumDigits(); n > precision {
			return fmt.Errorf("%s has %d significant digits, exceeds precision %d", d.String(), n, precision)
		}
	}
	if scale > 0 && digitsAfterPoint > scale {
		return fmt.Errorf("%s has %d digits after the decimal point, exceeds scale %d", d.String(), digitsAfterPoint, scale)
	}

	return nil
}

// yamlKindName names a yaml.Node's Kind for error messages - yaml.Kind has
// no String() method of its own, so left to %v it prints as a bare number.
func yamlKindName(k yaml.Kind) string {
	switch k {
	case yaml.DocumentNode:
		return "a document"
	case yaml.SequenceNode:
		return "a list"
	case yaml.MappingNode:
		return "a mapping"
	case yaml.ScalarNode:
		return "a scalar"
	case yaml.AliasNode:
		return "an alias"
	default:
		return fmt.Sprintf("kind %d", k)
	}
}
