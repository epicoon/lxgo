package model

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestPgIdent(t *testing.T) {
	if got := pgIdent("widgets"); got != `"widgets"` {
		t.Fatalf("got %q", got)
	}
	if got := pgIdent(`weird"name`); got != `"weird""name"` {
		t.Fatalf("got %q, want doubled embedded quote", got)
	}
}

func TestPgQuoteLiteral(t *testing.T) {
	if got := pgQuoteLiteral("hello"); got != "'hello'" {
		t.Fatalf("got %q", got)
	}
	if got := pgQuoteLiteral("it's ok"); got != "'it''s ok'" {
		t.Fatalf("got %q, want doubled embedded quote", got)
	}
}

func TestPgColumnType(t *testing.T) {
	tests := []struct {
		name string
		f    Field
		want string
	}{
		{"string no size", Field{Type: FieldTypeString}, "text"},
		{"string with size", Field{Type: FieldTypeString, Size: 255}, "character varying(255)"},
		{"int", Field{Type: FieldTypeInt}, "integer"},
		{"float", Field{Type: FieldTypeFloat}, "double precision"},
		{"decimal no precision", Field{Type: FieldTypeDecimal}, "numeric"},
		{"decimal with precision/scale", Field{Type: FieldTypeDecimal, Precision: 10, Scale: 2}, "numeric(10,2)"},
		{"bool", Field{Type: FieldTypeBool}, "boolean"},
		{"date", Field{Type: FieldTypeDate}, "date"},
		{"time", Field{Type: FieldTypeTime}, "time without time zone"},
		{"datetime is timestamptz, not naive timestamp", Field{Type: FieldTypeDateTime}, "timestamp with time zone"},
		{"interval", Field{Type: FieldTypeInterval}, "interval"},
		{"dict", Field{Type: FieldTypeDict}, "jsonb"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := pgColumnType(tt.f)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPgColumnType_UnknownTypeIsError(t *testing.T) {
	if _, err := pgColumnType(Field{Type: "bogus"}); err == nil {
		t.Fatal("expected an error for an unknown field type")
	}
}

func TestPgDefaultSQL(t *testing.T) {
	tests := []struct {
		name string
		f    Field
		want string
	}{
		{"string", Field{Type: FieldTypeString, Default: "hello"}, "'hello'"},
		{"string with quote", Field{Type: FieldTypeString, Default: "it's ok"}, "'it''s ok'"},
		{"int", Field{Type: FieldTypeInt, Default: int64(42)}, "42"},
		{"float", Field{Type: FieldTypeFloat, Default: 0.5}, "0.5"},
		{"bool true", Field{Type: FieldTypeBool, Default: true}, "TRUE"},
		{"bool false", Field{Type: FieldTypeBool, Default: false}, "FALSE"},
		{"date", Field{Type: FieldTypeDate, Default: "2000-01-01"}, "'2000-01-01'::date"},
		{"time", Field{Type: FieldTypeTime, Default: "09:00:00"}, "'09:00:00'::time"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := pgDefaultSQL(tt.f)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPgDefaultSQL_Decimal(t *testing.T) {
	f := Field{Type: FieldTypeDecimal, Default: decimal.NewFromFloat(19.99)}
	got, err := pgDefaultSQL(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "'19.99'::numeric" {
		t.Fatalf("got %q", got)
	}
}

func TestPgDefaultSQL_DateTime(t *testing.T) {
	f := Field{Type: FieldTypeDateTime, Default: time.Date(2026, 1, 2, 15, 4, 5, 0, time.UTC)}
	got, err := pgDefaultSQL(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "'2026-01-02T15:04:05Z'::timestamptz" {
		t.Fatalf("got %q", got)
	}
}

func TestPgDefaultSQL_Interval(t *testing.T) {
	f := Field{Type: FieldTypeInterval, Default: 90 * time.Minute}
	got, err := pgDefaultSQL(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "'5400.000000 seconds'::interval" {
		t.Fatalf("got %q", got)
	}
}

func TestPgDefaultSQL_Dict(t *testing.T) {
	f := Field{Type: FieldTypeDict, Default: map[string]any{"theme": "dark"}}
	got, err := pgDefaultSQL(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != `'{"theme":"dark"}'::jsonb` {
		t.Fatalf("got %q", got)
	}
}

func TestPgDefaultSQL_TypeMismatchIsError(t *testing.T) {
	if _, err := pgDefaultSQL(Field{Type: FieldTypeInt, Default: "not an int"}); err == nil {
		t.Fatal("expected an error for a Default whose Go type doesn't match Type")
	}
}

func TestPgColumnDefinition(t *testing.T) {
	f := Field{Type: FieldTypeString, Size: 255, Required: true, Default: "unnamed"}
	got, err := pgColumnDefinition("name", f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `"name" character varying(255) NOT NULL DEFAULT 'unnamed'`
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestPgColumnDefinition_NullableNoDefault(t *testing.T) {
	f := Field{Type: FieldTypeInt}
	got, err := pgColumnDefinition("sort", f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != `"sort" integer` {
		t.Fatalf("got %q", got)
	}
}

func TestApplyInvert_UnparsableContentIsError(t *testing.T) {
	bad := []byte("not: [valid yaml structure for a migrationFile")
	if err := Apply(nil, bad); err == nil {
		t.Fatal("expected Apply to fail parsing malformed content before touching tx")
	}
	if err := Invert(nil, bad); err == nil {
		t.Fatal("expected Invert to fail parsing malformed content before touching tx")
	}
}

func TestApply_EmptyActionsIsNoOp(t *testing.T) {
	content := []byte("Name: empty\nType: model\n")
	if err := Apply(nil, content); err != nil {
		t.Fatalf("Apply with no actions should not touch tx at all: %v", err)
	}
	if err := Invert(nil, content); err != nil {
		t.Fatalf("Invert with no actions should not touch tx at all: %v", err)
	}
}
