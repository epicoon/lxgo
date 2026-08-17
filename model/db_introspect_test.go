package model

import (
	"database/sql"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestPgDefaultLiteral(t *testing.T) {
	tests := []struct {
		name       string
		raw        string
		wantLit    string
		wantQuoted bool
		wantOK     bool
	}{
		{"quoted with cast", "'unnamed'::character varying", "unnamed", true, true},
		{"quoted doubled inner quote", "'it''s ok'::text", "it's ok", true, true},
		{"quoted content with parens is fine", "'it''s a (test)'::text", "it's a (test)", true, true},
		{"bare bool", "true", "true", false, true},
		{"bare int", "42", "42", false, true},
		{"numeric with cast", "19.99::numeric", "19.99", false, true},
		{"unterminated quote", "'oops", "", false, false},
		{"empty after cast strip", "::integer", "", false, false},
		{"unquoted function call is rejected", "now()", "", false, false},
		{"unquoted function call with cast is rejected", "(gen_random_uuid())::text", "", false, false},
		{"unquoted function call wrapping a quoted arg is rejected", "upper('hello'::text)", "", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lit, quoted, ok := pgDefaultLiteral(tt.raw)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if lit != tt.wantLit || quoted != tt.wantQuoted {
				t.Fatalf("got (%q, %v), want (%q, %v)", lit, quoted, tt.wantLit, tt.wantQuoted)
			}
		})
	}
}

func TestNormalizePgTimestamptz(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    string
		wantErr bool
	}{
		{"short offset gets minutes appended", "2026-01-02 15:04:05+00", "2026-01-02T15:04:05+00:00", false},
		{"full offset left as-is", "2026-01-02 15:04:05+05:30", "2026-01-02T15:04:05+05:30", false},
		{"negative short offset", "2026-01-02 15:04:05-08", "2026-01-02T15:04:05-08:00", false},
		{"no offset at all is rejected", "2026-01-02 15:04:05", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizePgTimestamptz(tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected an error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParsePgInterval(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    time.Duration
		wantErr bool
	}{
		{"time only", "01:30:00", time.Hour + 30*time.Minute, false},
		{"one day", "1 day", 24 * time.Hour, false},
		{"day plus time", "1 day 01:30:00", 25*time.Hour + 30*time.Minute, false},
		{"negative day, positive time", "-1 days +01:00:00", -23 * time.Hour, false},
		{"fractional seconds", "00:00:00.5", 500 * time.Millisecond, false},
		{"months are rejected", "3 mons", 0, true},
		{"years are rejected", "1 year", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parsePgInterval(tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected an error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPgDefaultToField(t *testing.T) {
	tests := []struct {
		name      string
		fieldType FieldType
		raw       string
		want      any
	}{
		{"string", FieldTypeString, "'unnamed'::character varying", "unnamed"},
		{"int", FieldTypeInt, "42", int64(42)},
		{"float", FieldTypeFloat, "0.5", float64(0.5)},
		{"bool", FieldTypeBool, "true", true},
		{"date", FieldTypeDate, "'2000-01-01'::date", "2000-01-01"},
		{"time", FieldTypeTime, "'09:00:00'::time without time zone", "09:00:00"},
		{"sequence default has no literal value", FieldTypeInt, "nextval('widgets_id_seq'::regclass)", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := pgDefaultToField(tt.fieldType, tt.raw, 0, 0, 0)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestPgDefaultToField_Decimal(t *testing.T) {
	got, err := pgDefaultToField(FieldTypeDecimal, "19.99::numeric", 0, 10, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	d, ok := got.(decimal.Decimal)
	if !ok || !d.Equal(decimal.NewFromFloat(19.99)) {
		t.Fatalf("got %#v, want decimal 19.99", got)
	}
}

func TestPgDefaultToField_DateTime(t *testing.T) {
	got, err := pgDefaultToField(FieldTypeDateTime, "'2026-01-02 15:04:05+00'::timestamp with time zone", 0, 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tm, ok := got.(time.Time)
	if !ok {
		t.Fatalf("got %#v, want time.Time", got)
	}
	want := time.Date(2026, 1, 2, 15, 4, 5, 0, time.UTC)
	if !tm.Equal(want) {
		t.Fatalf("got %v, want %v", tm, want)
	}
}

func TestPgDefaultToField_Interval(t *testing.T) {
	got, err := pgDefaultToField(FieldTypeInterval, "'01:30:00'::interval", 0, 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 90*time.Minute {
		t.Fatalf("got %v, want 1h30m", got)
	}
}

func TestPgDefaultToField_Dict(t *testing.T) {
	got, err := pgDefaultToField(FieldTypeDict, `'{"theme":"dark"}'::jsonb`, 0, 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, ok := got.(map[string]any)
	if !ok || m["theme"] != "dark" {
		t.Fatalf("got %#v, want map with theme=dark", got)
	}
}

func TestPgDefaultToField_UnquotedDictIsRejected(t *testing.T) {
	if _, err := pgDefaultToField(FieldTypeDict, "'{}'::jsonb", 0, 0, 0); err != nil {
		t.Fatalf("unexpected error for a normally-quoted dict default: %v", err)
	}
	if _, err := pgDefaultToField(FieldTypeDict, "gen_random_uuid()", 0, 0, 0); err == nil {
		t.Fatal("expected an error for a non-literal dict default expression")
	}
}

// TestPgDefaultToField_NowHasNoRepresentableDefault checks that "now()" -
// the one function-call default this package's own DDL ever produces (see
// execCreateTable/execAddTimestamps) - comes back as nil, nil ("no
// representable static default"), the same way a sequence default
// (nextval(...)) already does, for every field type it could plausibly
// appear on - see pgDefaultToField's doc. Any OTHER function call stays
// rejected (see TestPgDefaultToField_RejectsFunctionCallExpressions).
func TestPgDefaultToField_NowHasNoRepresentableDefault(t *testing.T) {
	for _, ft := range []FieldType{FieldTypeDateTime, FieldTypeDict, FieldTypeString} {
		t.Run(string(ft), func(t *testing.T) {
			got, err := pgDefaultToField(ft, "now()", 0, 0, 0)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != nil {
				t.Fatalf("got %#v, want nil", got)
			}
		})
	}
}

func TestColumnToField_UnsupportedType(t *testing.T) {
	_, err := columnToField("tsvector", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "YES", sql.NullString{}, nil)
	if err == nil {
		t.Fatal("expected an error for an unrecognized Postgres column type")
	}
}

func TestColumnToField_StringWithSizeAndDefault(t *testing.T) {
	f, err := columnToField(
		"character varying",
		sql.NullInt64{Int64: 255, Valid: true}, sql.NullInt64{}, sql.NullInt64{},
		"NO", sql.NullString{String: "'unnamed'::character varying", Valid: true},
		nil,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.Type != FieldTypeString || !f.Required || f.Size != 255 || f.Default != "unnamed" {
		t.Fatalf("got %#v", f)
	}
}

func TestColumnToField_NullableNoDefault(t *testing.T) {
	f, err := columnToField(
		"text",
		sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{},
		"YES", sql.NullString{},
		nil,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.Required || f.Default != nil {
		t.Fatalf("got %#v, want not required and no default", f)
	}
}

func TestColumnToField_OverrideTakesPrecedenceOverPhysicalType(t *testing.T) {
	f, err := columnToField(
		"text",
		sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{},
		"YES", sql.NullString{String: `'{"a":1}'::text`, Valid: true},
		&columnOverride{Type: FieldTypeDict},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, ok := f.Default.(map[string]any)
	if f.Type != FieldTypeDict || !ok || m["a"] != float64(1) {
		t.Fatalf("got %#v, want FieldTypeDict with default {a:1}", f)
	}
}

func TestColumnToField_OverrideWithUnknownTypeIsRejected(t *testing.T) {
	_, err := columnToField(
		"text",
		sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{},
		"YES", sql.NullString{},
		&columnOverride{Type: FieldType("bogus")},
	)
	if err == nil {
		t.Fatal("expected an error for a recorded type that isn't a known FieldType")
	}
}

// TestPgTypeToFieldType_AllKnownTypes locks down every Postgres data_type
// pgTypeToFieldType is supposed to recognize - a typo or accidental
// deletion in the map would otherwise only surface as a runtime "unsupported
// column type" error against a real database, not a test failure.
func TestPgTypeToFieldType_AllKnownTypes(t *testing.T) {
	want := map[string]FieldType{
		"character varying":        FieldTypeString,
		"character":                FieldTypeString,
		"text":                     FieldTypeString,
		"integer":                  FieldTypeInt,
		"bigint":                   FieldTypeInt,
		"smallint":                 FieldTypeInt,
		"double precision":         FieldTypeFloat,
		"real":                     FieldTypeFloat,
		"numeric":                  FieldTypeDecimal,
		"boolean":                  FieldTypeBool,
		"date":                     FieldTypeDate,
		"time without time zone":   FieldTypeTime,
		"timestamp with time zone": FieldTypeDateTime,
		"interval":                 FieldTypeInterval,
		"jsonb":                    FieldTypeDict,
		"json":                     FieldTypeDict,
	}
	if len(pgTypeToFieldType) != len(want) {
		t.Fatalf("pgTypeToFieldType has %d entries, want %d - update this test alongside the map", len(pgTypeToFieldType), len(want))
	}
	for dataType, wantType := range want {
		if got := pgTypeToFieldType[dataType]; got != wantType {
			t.Errorf("pgTypeToFieldType[%q] = %q, want %q", dataType, got, wantType)
		}
	}
}

// TestPgDefaultToField_RejectsFunctionCallExpressions is a regression test:
// String/Decimal/Date/Time defaults used to pass an unquoted, unvalidated
// literal straight through, so a function-call default (upper('hello'),
// gen_random_uuid()) silently became mangled leftover text instead of a
// rejected default - only Dict had this check.
func TestPgDefaultToField_RejectsFunctionCallExpressions(t *testing.T) {
	tests := []struct {
		fieldType FieldType
		raw       string
	}{
		{FieldTypeString, "upper('hello'::text)"},
		{FieldTypeString, "(gen_random_uuid())::text"},
		{FieldTypeDecimal, "round(19.99)::numeric"},
		{FieldTypeDate, "(now())::date"},
		{FieldTypeTime, "(now())::time"},
	}
	for _, tt := range tests {
		t.Run(string(tt.fieldType), func(t *testing.T) {
			if _, err := pgDefaultToField(tt.fieldType, tt.raw, 0, 0, 0); err == nil {
				t.Fatalf("pgDefaultToField(%q, %q): expected an error, got nil", tt.fieldType, tt.raw)
			}
		})
	}
}

func TestCheckTimestampColumnCompatible(t *testing.T) {
	createdAt := timestampColumnSpec{name: "created_at", required: true}
	deletedAt := timestampColumnSpec{name: "deleted_at", required: false}

	tests := []struct {
		name    string
		spec    timestampColumnSpec
		info    timestampColumnInfo
		wantErr bool
	}{
		{"required column, compatible (NOT NULL)", createdAt, timestampColumnInfo{dataType: "timestamp with time zone", nullable: false}, false},
		{"nullable column, compatible (nullable)", deletedAt, timestampColumnInfo{dataType: "timestamp with time zone", nullable: true}, false},
		{"required column but nullable", createdAt, timestampColumnInfo{dataType: "timestamp with time zone", nullable: true}, true},
		{"nullable column but NOT NULL", deletedAt, timestampColumnInfo{dataType: "timestamp with time zone", nullable: false}, true},
		{"wrong physical type", createdAt, timestampColumnInfo{dataType: "timestamp without time zone", nullable: false}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkTimestampColumnCompatible(tt.spec, tt.info)
			if (err != nil) != tt.wantErr {
				t.Fatalf("checkTimestampColumnCompatible(%+v, %+v) = %v, want error: %v", tt.spec, tt.info, err, tt.wantErr)
			}
		})
	}
}
