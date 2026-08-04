package utils_test

import (
	"testing"

	"github.com/epicoon/lxgo/jspp/internal/utils"
)

func TestFindMatchingBrace_Basic(t *testing.T) {
	cases := []struct {
		name  string
		code  string
		start int
		brace rune
		want  int
	}{
		{"parens", "(a(b)c)d", 0, '(', 6},
		{"nested_same_kind", "{a{b}c}", 0, '{', 6},
		{"brackets", "[a[b]c]", 0, '[', 6},
		{"unterminated", "(a(b)c", 0, '(', -1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := utils.FindMatchingBrace(tc.code, tc.start, tc.brace)
			if got != tc.want {
				t.Fatalf("FindMatchingBrace(%q, %d, %q) = %d, want %d", tc.code, tc.start, tc.brace, got, tc.want)
			}
		})
	}
}

// TestFindMatchingBrace_IgnoresBracesInsideStringsAndComments is a
// regression test: FindMatchingBrace used to count brace/paren/bracket
// characters via a naive byte scan with no awareness of string literals or
// comments, so e.g. a ')' inside a quoted path argument
// (lx.import('a)b.js')) would be mistaken for the call's own closing
// paren, cutting the argument list short.
func TestFindMatchingBrace_IgnoresBracesInsideStringsAndComments(t *testing.T) {
	cases := []struct {
		name  string
		code  string
		start int
		brace rune
		want  int
	}{
		{"paren_inside_double_quoted_string", `lx.import("a)b.js")`, 9, '(', 18},
		{"paren_inside_single_quoted_string", `lx.import('a)b.js')`, 9, '(', 18},
		{"paren_inside_template_literal", "lx.import(`a)b.js`)", 9, '(', 18},
		{"escaped_quote_does_not_end_string_early", `f('a\'b)c', d)`, 1, '(', 13},
		{"brace_inside_line_comment", "{\n// } not real\n}", 0, '{', 16},
		{"brace_inside_block_comment", "{ /* } not real */ }", 0, '{', 19},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := utils.FindMatchingBrace(tc.code, tc.start, tc.brace)
			if got != tc.want {
				t.Fatalf("FindMatchingBrace(%q, %d, %q) = %d, want %d", tc.code, tc.start, tc.brace, got, tc.want)
			}
		})
	}
}

// TestFindMatchingBrace_IgnoresBracesInsideRegexLiterals is a regression
// test: a JS regex literal like /{/ or /}/ can contain a brace character
// with no matching counterpart inside that same literal - FindMatchingBrace
// used to count these as real code braces, throwing off the depth count
// (e.g. code.replace(/^\s*function\s*\(([^\)]*?)\)\s*{\s*/, '($1)=>')).
func TestFindMatchingBrace_IgnoresBracesInsideRegexLiterals(t *testing.T) {
	code := `{ a.replace(/{/, ''); b.replace(/}/, ''); if (c) { d(); } }`
	want := len(code) - 1
	got := utils.FindMatchingBrace(code, 0, '{')
	if got != want {
		t.Fatalf("FindMatchingBrace(%q, 0, '{') = %d, want %d", code, got, want)
	}
}
