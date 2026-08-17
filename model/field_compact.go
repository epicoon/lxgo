package model

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// The compact single-line field form: `<type>[(<details>)] [required]
// [default(<literal>)]`, tokens in any order after the leading type. A
// <literal> is either a bareword (no whitespace, quotes or parens) or a
// single-quoted string (`'...'`, a literal quote inside written as `''` -
// the same escaping SQL string literals use) - quoting is required for any
// default that contains whitespace or structure (a dict's default is JSON:
// `default('{"n":1}')`), a bareword may not contain whitespace at all.
//
// Unlike the map form, there's no fixed key set to check for typos against
// - instead, anything left over after the recognized tokens are stripped
// out is a hard parse error, so a garbled or misspelled modifier can't
// silently vanish.

var (
	compactTypeRe     = regexp.MustCompile(`^(\w+)(\([^)]*\))?\s*`)
	compactRequiredRe = regexp.MustCompile(`\brequired\b`)
	compactDefaultRe  = regexp.MustCompile(`\bdefault\(`)
)

func (f *Field) unmarshalCompactForm(s string) error {
	m := compactTypeRe.FindStringSubmatch(s)
	if m == nil {
		return fmt.Errorf("can not find a field type at the start of %q", s)
	}
	fieldType := FieldType(m[1])
	rest := s[len(m[0]):]

	if !knownFieldTypes[fieldType] {
		return fmt.Errorf("unknown field type %q", fieldType)
	}

	// m[2] is "" both when the parens were entirely absent and when they
	// were given empty ("()") - detailsGiven tells those apart so `string()`
	// can be rejected instead of silently meaning "no size limit".
	detailsGiven := m[2] != ""
	var detailsRaw string
	if detailsGiven {
		detailsRaw = m[2][1 : len(m[2])-1]
	}
	size, precision, scale, err := parseCompactDetails(fieldType, detailsGiven, detailsRaw)
	if err != nil {
		return fmt.Errorf("invalid details for type %q: %w", fieldType, err)
	}

	// Extract default(...) before scanning for "required" - its quoted
	// content might itself contain that word (`default('is required')`),
	// which must not be mistaken for the modifier.
	rest, defaultLiteral, err := extractCompactDefault(rest)
	if err != nil {
		return fmt.Errorf("invalid default in %q: %w", s, err)
	}

	required := false
	if loc := compactRequiredRe.FindStringIndex(rest); loc != nil {
		required = true
		rest = rest[:loc[0]] + rest[loc[1]:]
	}

	if trailing := strings.TrimSpace(rest); trailing != "" {
		return fmt.Errorf("unexpected content %q in %q", trailing, s)
	}

	f.Type = fieldType
	f.Required = required
	f.RenamedFrom = ""
	f.Size = size
	f.Precision = precision
	f.Scale = scale
	f.Default = nil
	f.compactForm = true

	if defaultLiteral == nil {
		return nil
	}

	rawDefault, err := compactLiteralToRaw(fieldType, *defaultLiteral)
	if err != nil {
		return fmt.Errorf("invalid default for type %q: %w", fieldType, err)
	}
	def, err := parseDefault(fieldType, rawDefault, size, precision, scale)
	if err != nil {
		return fmt.Errorf("invalid default for type %q: %w", fieldType, err)
	}
	f.Default = def
	return nil
}

// parseCompactDetails parses the parenthesized part right after the type
// name (`(4000)` for string, `(10, 2)` for decimal) - given is false when no
// parens were written at all (as opposed to empty parens, `type()`, which
// is a parse error, not "no limit").
func parseCompactDetails(t FieldType, given bool, raw string) (size, precision, scale int, err error) {
	raw = strings.TrimSpace(raw)

	switch t {
	case FieldTypeString:
		if !given {
			return 0, 0, 0, nil
		}
		if raw == "" {
			return 0, 0, 0, fmt.Errorf("empty parenthesized details")
		}
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			return 0, 0, 0, fmt.Errorf("size must be a non-negative integer, got %q", raw)
		}
		return n, 0, 0, nil

	case FieldTypeDecimal:
		if !given {
			return 0, 0, 0, nil
		}
		if raw == "" {
			return 0, 0, 0, fmt.Errorf("empty parenthesized details")
		}
		parts := strings.SplitN(raw, ",", 2)
		if len(parts) != 2 {
			return 0, 0, 0, fmt.Errorf("expected \"precision, scale\", got %q", raw)
		}
		p, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil || p < 0 {
			return 0, 0, 0, fmt.Errorf("precision must be a non-negative integer, got %q", parts[0])
		}
		sc, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil || sc < 0 {
			return 0, 0, 0, fmt.Errorf("scale must be a non-negative integer, got %q", parts[1])
		}
		return 0, p, sc, nil

	default:
		if given {
			return 0, 0, 0, fmt.Errorf("type %q takes no parenthesized details", t)
		}
		return 0, 0, 0, nil
	}
}

// extractCompactDefault finds a default(...) token in s (if any), removes
// it, and returns its literal content alongside the rest of the string.
// The literal is unquoted/unescaped already - see the package doc above.
func extractCompactDefault(s string) (rest string, literal *string, err error) {
	loc := compactDefaultRe.FindStringIndex(s)
	if loc == nil {
		return s, nil, nil
	}
	start, afterOpen := loc[0], loc[1]

	if afterOpen < len(s) && s[afterOpen] == '\'' {
		lit, end, err := scanQuotedLiteral(s, afterOpen+1)
		if err != nil {
			return s, nil, err
		}
		return s[:start] + s[end:], &lit, nil
	}

	lit, end, err := scanUnquotedLiteral(s, afterOpen)
	if err != nil {
		return s, nil, err
	}
	return s[:start] + s[end:], &lit, nil
}

// scanUnquotedLiteral reads an unquoted default(...) literal - surrounding
// whitespace (`default( 42 )`) is trimmed, but the literal itself may not
// contain whitespace: a value that needs one must be quoted instead (this
// is also what keeps a stray trailing modifier, like a forgotten `required`,
// from silently being swallowed into an unquoted default).
func scanUnquotedLiteral(s string, start int) (literal string, end int, err error) {
	i := start
	for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	litStart := i
	for i < len(s) && s[i] != ' ' && s[i] != '\t' && s[i] != ')' {
		i++
	}
	litEnd := i
	for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	if i >= len(s) || s[i] != ')' {
		if i < len(s) && s[i] != ')' {
			return "", 0, fmt.Errorf("unquoted default must not contain whitespace - wrap it in single quotes")
		}
		return "", 0, fmt.Errorf("unterminated default(...)")
	}
	return s[litStart:litEnd], i + 1, nil
}

// scanQuotedLiteral reads a single-quoted literal starting right after its
// opening quote character - a literal quote inside the value is written as
// two consecutive quote characters, the same convention SQL string literals
// use. Returns the unescaped content and the index right after the whole
// quoted-and-closing-paren construct.
func scanQuotedLiteral(s string, start int) (literal string, end int, err error) {
	var b strings.Builder
	i := start
	for i < len(s) {
		if s[i] != '\'' {
			b.WriteByte(s[i])
			i++
			continue
		}
		if i+1 < len(s) && s[i+1] == '\'' {
			b.WriteByte('\'')
			i += 2
			continue
		}
		// closing quote
		i++
		if i >= len(s) || s[i] != ')' {
			return "", 0, fmt.Errorf("expected ')' right after the closing quote")
		}
		return b.String(), i + 1, nil
	}
	return "", 0, fmt.Errorf("unterminated quoted default")
}

// compactLiteralToRaw converts a default(...) literal (always a string in
// the compact form, quoted or not) into the same Go representation
// yaml.Node.Decode would have produced for the map form's `default:` value,
// so it can be validated through the exact same parseDefault used there -
// not a separate, weaker copy of that validation. For bool/int/float this
// means literally decoding the literal as a yaml scalar (not hand-rolling
// strconv calls that would inevitably accept/reject a slightly different
// set of spellings than yaml.v3's own scalar resolution does).
func compactLiteralToRaw(t FieldType, literal string) (any, error) {
	switch t {
	case FieldTypeBool, FieldTypeInt, FieldTypeFloat:
		var v any
		if err := yaml.Unmarshal([]byte(literal), &v); err != nil {
			return nil, fmt.Errorf("invalid literal %q: %w", literal, err)
		}
		return v, nil

	case FieldTypeDict:
		var v any
		if err := json.Unmarshal([]byte(literal), &v); err != nil {
			return nil, fmt.Errorf("must be valid JSON, got %q: %w", literal, err)
		}
		return v, nil

	default:
		// string/decimal/date/time/datetime/interval already take a
		// string in parseDefault.
		return literal, nil
	}
}

// marshalCompactForm renders f as a compact single-line string - the
// inverse of unmarshalCompactForm, used by Field.MarshalYAML for a Field
// that was parsed from that form. Token order isn't required to match how
// it was originally written (the grammar doesn't care, and preserving the
// original order isn't worth the bookkeeping) - always `type[(details)]`,
// then `required`, then `default(...)`.
func (f Field) marshalCompactForm() (string, error) {
	var b strings.Builder
	b.WriteString(string(f.Type))

	switch f.Type {
	case FieldTypeString:
		if f.Size > 0 {
			fmt.Fprintf(&b, "(%d)", f.Size)
		}
	case FieldTypeDecimal:
		if f.Precision > 0 || f.Scale > 0 {
			fmt.Fprintf(&b, "(%d, %d)", f.Precision, f.Scale)
		}
	}

	if f.Required {
		b.WriteString(" required")
	}

	if f.Default != nil {
		literal, err := compactDefaultLiteral(f.Type, formatDefault(f.Type, f.Default))
		if err != nil {
			return "", fmt.Errorf("default: %w", err)
		}
		b.WriteString(" default(")
		b.WriteString(literal)
		b.WriteString(")")
	}

	return b.String(), nil
}

// compactDefaultLiteral is compactLiteralToRaw's inverse - formatted is
// what formatDefault(t, f.Default) produced (i.e. already in the same
// shape the map form would write to `default:`). Quotes the result (with
// ” escaping) only when it actually needs it - a bareword is shorter and
// reads better when it's unambiguous.
func compactDefaultLiteral(t FieldType, formatted any) (string, error) {
	switch t {
	case FieldTypeDict:
		raw, err := json.Marshal(formatted)
		if err != nil {
			return "", fmt.Errorf("not JSON-serializable: %w", err)
		}
		return quoteCompactLiteral(string(raw)), nil

	case FieldTypeString:
		s := formatted.(string)
		if isCompactBareword(s) {
			return s, nil
		}
		return quoteCompactLiteral(s), nil

	case FieldTypeBool, FieldTypeInt, FieldTypeFloat:
		return fmt.Sprintf("%v", formatted), nil

	default:
		// decimal/date/time/datetime/interval - formatDefault already
		// produced a string that's always a safe bareword (digits,
		// letters, ':'/'-'/'+'/'.', never whitespace or parens/quotes).
		return formatted.(string), nil
	}
}

func quoteCompactLiteral(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// isCompactBareword reports whether s can be written as default(s) without
// quoting - i.e. parsing it back wouldn't stop early or need escaping.
func isCompactBareword(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		switch r {
		case ' ', '\t', '\'', '(', ')':
			return false
		}
	}
	return true
}
