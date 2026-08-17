package model

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"gopkg.in/yaml.v3"
)

func unmarshalField(t *testing.T, src string) (Field, error) {
	t.Helper()
	var f Field
	err := yaml.Unmarshal([]byte(src), &f)
	return f, err
}

func mustUnmarshalField(t *testing.T, src string) Field {
	t.Helper()
	f, err := unmarshalField(t, src)
	if err != nil {
		t.Fatalf("unmarshal %q: %v", src, err)
	}
	return f
}

func TestUnmarshalField_UnknownType(t *testing.T) {
	if _, err := unmarshalField(t, "Type: not_a_type\n"); err == nil {
		t.Fatal("expected an error for an unknown field type")
	}
}

func TestUnmarshalField_NoDefault(t *testing.T) {
	f := mustUnmarshalField(t, "Type: string\n")
	if f.Default != nil {
		t.Fatalf("Default = %#v, want nil when omitted", f.Default)
	}
}

func TestUnmarshalField_ExplicitNullDefault(t *testing.T) {
	f := mustUnmarshalField(t, "Type: string\nDefault: null\n")
	if f.Default != nil {
		t.Fatalf("Default = %#v, want nil for an explicit null", f.Default)
	}
}

func TestUnmarshalField_String(t *testing.T) {
	f := mustUnmarshalField(t, "Type: string\nDefault: hello\n")
	if f.Default != "hello" {
		t.Fatalf("Default = %#v, want \"hello\"", f.Default)
	}
}

func TestUnmarshalField_String_SizeExceeded(t *testing.T) {
	_, err := unmarshalField(t, "Type: string\nSize: 3\nDefault: hello\n")
	if err == nil {
		t.Fatal("expected an error: default exceeds size")
	}
}

func TestUnmarshalField_String_WrongScalarType(t *testing.T) {
	if _, err := unmarshalField(t, "Type: string\nDefault: 5\n"); err == nil {
		t.Fatal("expected an error: default is not a string")
	}
}

func TestUnmarshalField_NegativeSizeRejected(t *testing.T) {
	if _, err := unmarshalField(t, "Type: string\nSize: -5\n"); err == nil {
		t.Fatal("expected an error: negative size")
	}
}

func TestUnmarshalField_NegativePrecisionOrScaleRejected(t *testing.T) {
	if _, err := unmarshalField(t, "Type: decimal\nPrecision: -5\nScale: 2\n"); err == nil {
		t.Fatal("expected an error: negative precision")
	}
	if _, err := unmarshalField(t, "Type: decimal\nPrecision: 5\nScale: -2\n"); err == nil {
		t.Fatal("expected an error: negative scale")
	}
}

func TestUnmarshalField_Int(t *testing.T) {
	f := mustUnmarshalField(t, "Type: int\nDefault: 42\n")
	if f.Default != int64(42) {
		t.Fatalf("Default = %#v, want int64(42)", f.Default)
	}
}

func TestUnmarshalField_Int_FractionalRejected(t *testing.T) {
	if _, err := unmarshalField(t, "Type: int\nDefault: 4.2\n"); err == nil {
		t.Fatal("expected an error: default has a fractional part")
	}
}

func TestUnmarshalField_Float(t *testing.T) {
	f := mustUnmarshalField(t, "Type: float\nDefault: 4.2\n")
	if f.Default != 4.2 {
		t.Fatalf("Default = %#v, want 4.2", f.Default)
	}
}

func TestUnmarshalField_Float_WholeNumberWidened(t *testing.T) {
	f := mustUnmarshalField(t, "Type: float\nDefault: 4\n")
	if f.Default != float64(4) {
		t.Fatalf("Default = %#v, want float64(4) (whole yaml numbers decode as int)", f.Default)
	}
}

func TestUnmarshalField_Float_WrongScalarType(t *testing.T) {
	if _, err := unmarshalField(t, "Type: float\nDefault: \"abc\"\n"); err == nil {
		t.Fatal("expected an error: default is not a number")
	}
}

func TestUnmarshalField_Int_WrongScalarType(t *testing.T) {
	if _, err := unmarshalField(t, "Type: int\nDefault: \"abc\"\n"); err == nil {
		t.Fatal("expected an error: default is not an integer")
	}
}

func TestUnmarshalField_Bool(t *testing.T) {
	f := mustUnmarshalField(t, "Type: bool\nDefault: true\n")
	if f.Default != true {
		t.Fatalf("Default = %#v, want true", f.Default)
	}
}

func TestUnmarshalField_Bool_StringRejected(t *testing.T) {
	if _, err := unmarshalField(t, "Type: bool\nDefault: \"true\"\n"); err == nil {
		t.Fatal("expected an error: default is the string \"true\", not a bool")
	}
}

func TestUnmarshalField_Decimal(t *testing.T) {
	f := mustUnmarshalField(t, "Type: decimal\nPrecision: 5\nScale: 2\nDefault: \"123.45\"\n")
	d, ok := f.Default.(decimal.Decimal)
	if !ok {
		t.Fatalf("Default type = %T, want decimal.Decimal", f.Default)
	}
	if d.String() != "123.45" {
		t.Fatalf("Default = %s, want 123.45", d.String())
	}
}

func TestUnmarshalField_Decimal_ExceedsPrecision(t *testing.T) {
	if _, err := unmarshalField(t, "Type: decimal\nPrecision: 3\nScale: 2\nDefault: \"123.45\"\n"); err == nil {
		t.Fatal("expected an error: 5 significant digits exceeds precision 3")
	}
}

func TestUnmarshalField_Decimal_ExceedsScale(t *testing.T) {
	if _, err := unmarshalField(t, "Type: decimal\nPrecision: 5\nScale: 1\nDefault: \"123.45\"\n"); err == nil {
		t.Fatal("expected an error: 2 digits after the point exceeds scale 1")
	}
}

func TestUnmarshalField_Decimal_InvalidLiteral(t *testing.T) {
	if _, err := unmarshalField(t, "Type: decimal\nDefault: \"not-a-number\"\n"); err == nil {
		t.Fatal("expected an error: not a valid decimal literal")
	}
}

func TestUnmarshalField_Decimal_NumberScalarRejected(t *testing.T) {
	if _, err := unmarshalField(t, "Type: decimal\nDefault: 123.45\n"); err == nil {
		t.Fatal("expected an error: decimal default must be a yaml string, not a float scalar")
	}
}

func TestUnmarshalField_Date(t *testing.T) {
	f := mustUnmarshalField(t, "Type: date\nDefault: \"2026-01-02\"\n")
	if f.Default != "2026-01-02" {
		t.Fatalf("Default = %#v, want \"2026-01-02\"", f.Default)
	}
}

func TestUnmarshalField_Date_InvalidFormat(t *testing.T) {
	if _, err := unmarshalField(t, "Type: date\nDefault: \"2026-1-2\"\n"); err == nil {
		t.Fatal("expected an error: date must be zero-padded (2006-01-02)")
	}
}

func TestUnmarshalField_Time(t *testing.T) {
	f := mustUnmarshalField(t, "Type: time\nDefault: \"15:04:05\"\n")
	if f.Default != "15:04:05" {
		t.Fatalf("Default = %#v, want \"15:04:05\"", f.Default)
	}
}

func TestUnmarshalField_Time_InvalidFormat(t *testing.T) {
	if _, err := unmarshalField(t, "Type: time\nDefault: \"3:04pm\"\n"); err == nil {
		t.Fatal("expected an error: time must be 24-hour HH:MM:SS")
	}
}

func TestUnmarshalField_DateTime(t *testing.T) {
	f := mustUnmarshalField(t, "Type: datetime\nDefault: \"2026-01-02T15:04:05+03:00\"\n")
	tm, ok := f.Default.(time.Time)
	if !ok {
		t.Fatalf("Default type = %T, want time.Time", f.Default)
	}
	if tm.Location() != time.UTC {
		t.Fatalf("Default location = %v, want normalized to UTC", tm.Location())
	}
	want := time.Date(2026, 1, 2, 12, 4, 5, 0, time.UTC)
	if !tm.Equal(want) {
		t.Fatalf("Default = %v, want %v (offset applied and normalized to UTC)", tm, want)
	}
}

func TestUnmarshalField_DateTime_MissingOffsetRejected(t *testing.T) {
	if _, err := unmarshalField(t, "Type: datetime\nDefault: \"2026-01-02T15:04:05\"\n"); err == nil {
		t.Fatal("expected an error: datetime literal without an explicit offset must be rejected")
	}
}

func TestUnmarshalField_Interval(t *testing.T) {
	f := mustUnmarshalField(t, "Type: interval\nDefault: \"1h30m\"\n")
	d, ok := f.Default.(time.Duration)
	if !ok {
		t.Fatalf("Default type = %T, want time.Duration", f.Default)
	}
	if d != 90*time.Minute {
		t.Fatalf("Default = %v, want 90m", d)
	}
}

func TestUnmarshalField_Interval_InvalidFormat(t *testing.T) {
	if _, err := unmarshalField(t, "Type: interval\nDefault: \"PT1H30M\"\n"); err == nil {
		t.Fatal("expected an error: ISO-8601 duration is not the chosen format")
	}
}

func TestUnmarshalField_Dict_Map(t *testing.T) {
	f := mustUnmarshalField(t, "Type: dict\nDefault:\n  a: 1\n  b: two\n")
	m, ok := f.Default.(map[string]any)
	if !ok {
		t.Fatalf("Default type = %T, want map[string]any", f.Default)
	}
	if m["b"] != "two" {
		t.Fatalf("Default[\"b\"] = %#v, want \"two\"", m["b"])
	}
}

func TestUnmarshalField_Dict_List(t *testing.T) {
	f := mustUnmarshalField(t, "Type: dict\nDefault:\n  - a\n  - b\n")
	if _, ok := f.Default.([]any); !ok {
		t.Fatalf("Default type = %T, want []any", f.Default)
	}
}

func TestUnmarshalField_Dict_ScalarRejected(t *testing.T) {
	if _, err := unmarshalField(t, "Type: dict\nDefault: 5\n"); err == nil {
		t.Fatal("expected an error: dict default must be a mapping or a list")
	}
}

func TestUnmarshalField_RenamedFrom(t *testing.T) {
	f := mustUnmarshalField(t, "Type: string\nRenamedFrom: oldName\n")
	if f.RenamedFrom != "oldName" {
		t.Fatalf("RenamedFrom = %q, want \"oldName\"", f.RenamedFrom)
	}
}

func TestField_IsEqual(t *testing.T) {
	base := Field{Type: FieldTypeString, Required: true, Default: "hello", Size: 10}

	cases := []struct {
		name  string
		other Field
		want  bool
	}{
		{"identical", Field{Type: FieldTypeString, Required: true, Default: "hello", Size: 10}, true},
		{"different type", Field{Type: FieldTypeInt, Required: true, Default: "hello", Size: 10}, false},
		{"different required", Field{Type: FieldTypeString, Required: false, Default: "hello", Size: 10}, false},
		{"different default", Field{Type: FieldTypeString, Required: true, Default: "bye", Size: 10}, false},
		{"different size", Field{Type: FieldTypeString, Required: true, Default: "hello", Size: 5}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := base.IsEqual(c.other); got != c.want {
				t.Fatalf("IsEqual = %v, want %v", got, c.want)
			}
		})
	}
}

func TestField_IsEqual_DecimalPrecisionScale(t *testing.T) {
	base := Field{Type: FieldTypeDecimal, Precision: 5, Scale: 2}

	if !base.IsEqual(Field{Type: FieldTypeDecimal, Precision: 5, Scale: 2}) {
		t.Fatal("expected identical precision/scale to compare equal")
	}
	if base.IsEqual(Field{Type: FieldTypeDecimal, Precision: 6, Scale: 2}) {
		t.Fatal("expected different precision to compare unequal")
	}
	if base.IsEqual(Field{Type: FieldTypeDecimal, Precision: 5, Scale: 3}) {
		t.Fatal("expected different scale to compare unequal")
	}
}

func TestUnmarshalField_UnknownKeyRejected(t *testing.T) {
	if _, err := unmarshalField(t, "Type: string\nSiez: 5\nDefault: hello\n"); err == nil {
		t.Fatal("expected an error: \"siez\" is a typo, not a recognized field attribute")
	}
}

func TestUnmarshalField_IrrelevantSizeRejected(t *testing.T) {
	if _, err := unmarshalField(t, "Type: int\nSize: 5\n"); err == nil {
		t.Fatal("expected an error: size is only meaningful for type string")
	}
}

func TestUnmarshalField_IrrelevantPrecisionScaleRejected(t *testing.T) {
	if _, err := unmarshalField(t, "Type: string\nPrecision: 5\nScale: 2\n"); err == nil {
		t.Fatal("expected an error: precision/scale are only meaningful for type decimal")
	}
}

func TestField_IsEqual_IgnoresRenamedFrom(t *testing.T) {
	a := Field{Type: FieldTypeString, RenamedFrom: "oldName"}
	b := Field{Type: FieldTypeString}
	if !a.IsEqual(b) {
		t.Fatal("expected IsEqual to ignore RenamedFrom - it's rename-hint metadata, not part of the field definition")
	}
}

func TestField_IsEqual_BothDefaultsNil(t *testing.T) {
	a := Field{Type: FieldTypeString}
	b := Field{Type: FieldTypeString}
	if !a.IsEqual(b) {
		t.Fatal("expected two fields with no default to be equal")
	}
}

func TestField_IsEqual_OneDefaultNil(t *testing.T) {
	a := Field{Type: FieldTypeString, Default: "x"}
	b := Field{Type: FieldTypeString}
	if a.IsEqual(b) {
		t.Fatal("expected a field with a default to differ from one without")
	}
}

func TestField_IsEqual_DecimalNormalizesRepresentation(t *testing.T) {
	a := mustUnmarshalField(t, "Type: decimal\nPrecision: 10\nScale: 2\nDefault: \"1.50\"\n")
	b := mustUnmarshalField(t, "Type: decimal\nPrecision: 10\nScale: 2\nDefault: \"1.5\"\n")
	if !a.IsEqual(b) {
		t.Fatal("expected \"1.50\" and \"1.5\" to compare equal as decimals")
	}
}

func TestField_IsEqual_IntervalNormalizesRepresentation(t *testing.T) {
	a := mustUnmarshalField(t, "Type: interval\nDefault: \"90m\"\n")
	b := mustUnmarshalField(t, "Type: interval\nDefault: \"1h30m\"\n")
	if !a.IsEqual(b) {
		t.Fatal("expected \"90m\" and \"1h30m\" to compare equal as durations")
	}
}

func TestField_IsEqual_DateTimeNormalizesOffset(t *testing.T) {
	a := mustUnmarshalField(t, "Type: datetime\nDefault: \"2026-01-01T12:00:00Z\"\n")
	b := mustUnmarshalField(t, "Type: datetime\nDefault: \"2026-01-01T15:00:00+03:00\"\n")
	if !a.IsEqual(b) {
		t.Fatal("expected the same instant recorded with different offsets to compare equal")
	}
}

func TestField_IsEqual_DictComparesByContent(t *testing.T) {
	a := mustUnmarshalField(t, "Type: dict\nDefault:\n  a: 1\n")
	b := mustUnmarshalField(t, "Type: dict\nDefault:\n  a: 1\n")
	c := mustUnmarshalField(t, "Type: dict\nDefault:\n  a: 2\n")
	if !a.IsEqual(b) {
		t.Fatal("expected identical dict defaults to compare equal")
	}
	if a.IsEqual(c) {
		t.Fatal("expected different dict defaults to compare unequal")
	}
}

func TestField_MarshalYAML_RoundTrip(t *testing.T) {
	cases := []string{
		"Type: string\nDefault: hello\n",
		"Type: int\nDefault: 42\n",
		"Type: float\nDefault: 4.2\n",
		"Type: decimal\nPrecision: 5\nScale: 2\nDefault: \"123.45\"\n",
		"Type: bool\nDefault: true\n",
		"Type: date\nDefault: \"2026-01-02\"\n",
		"Type: time\nDefault: \"15:04:05\"\n",
		"Type: datetime\nDefault: \"2026-01-02T15:04:05Z\"\n",
		"Type: interval\nDefault: \"1h30m\"\n",
		"Type: dict\nDefault:\n  a: 1\n",
	}
	for _, src := range cases {
		t.Run(src, func(t *testing.T) {
			original := mustUnmarshalField(t, src)

			out, err := yaml.Marshal(original)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}

			var roundTripped Field
			if err := yaml.Unmarshal(out, &roundTripped); err != nil {
				t.Fatalf("Unmarshal(Marshal(f)) failed: %v\nmarshaled:\n%s", err, out)
			}

			if !original.IsEqual(roundTripped) {
				t.Fatalf("round-trip changed the field: %#v -> marshaled %q -> %#v", original, out, roundTripped)
			}
		})
	}
}
