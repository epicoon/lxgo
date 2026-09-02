package compiler

import (
	"reflect"
	"strings"
	"testing"

	"github.com/epicoon/lxgo/jspp/internal/i18n"
)

func TestCutComments(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{"single_line", "a();\n// comment\nb();\n", "a();\nb();\n"},
		{"multi_line", "a();/* comment\nspanning */b();", "a();b();"},
		// A trailing "//" comment with no terminating newline used to be
		// left untouched (the old regex required a following line
		// terminator to match at all) - a correct scanner has no reason
		// to special-case end-of-input, so this is stripped correctly now.
		{"trailing_single_line_no_newline", "a(); // trailing", "a(); "},
		{"unterminated_block_comment_left_as_is", "a();/* never closed", "a();/* never closed"},
		// The comment markers used to be stripped by regex regardless of
		// string-literal context, silently corrupting compiled JS that
		// happened to contain "//" or "/* */" inside a string.
		{"double_quoted_string_with_line_comment_marker", `let x = "a // b";`, `let x = "a // b";`},
		{"double_quoted_string_with_block_comment_markers", `let re = "/* not a comment */ literally";`, `let re = "/* not a comment */ literally";`},
		{"single_quoted_string_with_comment_marker", `let x = 'a // b';`, `let x = 'a // b';`},
		{"template_literal_with_comment_marker", "let x = `a // b`;", "let x = `a // b`;"},
		{"escaped_quote_inside_string_does_not_end_it", `let x = "a \" // still a string";`, `let x = "a \" // still a string";`},
		{"real_comment_after_string_still_stripped", "let x = \"a\"; // real comment\nb();", "let x = \"a\"; b();"},
		// A JS regex literal can contain an unescaped "/*" inside a [...]
		// character class (e.g. /[/*]/ matches either '/' or '*') without
		// that being a real comment - cutComments used to not recognize
		// regex literals at all, so it would mistake this for a block
		// comment start and consume everything up to the next real "*/"
		// it could find, corrupting unrelated code in between.
		{
			"regex_literal_with_slash_star_in_char_class_not_a_comment",
			"let re = /[/*]/; let y = 1; /* actual comment */ done();",
			"let re = /[/*]/; let y = 1;  done();",
		},
	}
	c := &Compiler{}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := c.cutComments(tc.in)
			if got != tc.want {
				t.Fatalf("cutComments(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestBlankComments checks the position-preserving sibling of cutComments:
// same detection rules (strings/regex literals are left alone, only real
// comments are touched), but every input byte has a same-length output byte
// - a comment becomes spaces (its own newlines kept as newlines), nothing is
// removed. findImportCalls relies on this to keep byte offsets valid against
// the original code it's searching.
func TestBlankComments(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{"single_line", "a();\n// comment\nb();\n", "a();\n          \nb();\n"},
		{"multi_line", "a();/* comment\nspanning */b();", "a();          \n           b();"},
		{"string_with_comment_marker_untouched", `let x = "a // b";`, `let x = "a // b";`},
		{"regex_literal_with_slash_star_in_char_class_not_a_comment", "let re = /[/*]/; b();", "let re = /[/*]/; b();"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := blankComments(tc.in)
			if got != tc.want {
				t.Fatalf("blankComments(%q) = %q, want %q", tc.in, got, tc.want)
			}
			if len(got) != len(tc.in) {
				t.Fatalf("blankComments(%q) changed length: got %d, want %d", tc.in, len(got), len(tc.in))
			}
		})
	}
}

func TestApplyContext(t *testing.T) {
	src := "before\n@lx:<context CLIENT:\nclientCode();\n@lx:context>\n@lx:<context SERVER:\nserverCode();\n@lx:context>\nafter"

	c := &Compiler{context: contextClient}
	got := c.applyContext(src)
	if !strings.Contains(got, "clientCode();") || strings.Contains(got, "serverCode();") {
		t.Fatalf("CLIENT context: expected clientCode kept and serverCode dropped, got %q", got)
	}
	if !strings.HasPrefix(got, "before\n") || !strings.HasSuffix(got, "after") {
		t.Fatalf("expected the surrounding text preserved, got %q", got)
	}

	c = &Compiler{context: contextServer}
	got = c.applyContext(src)
	if !strings.Contains(got, "serverCode();") || strings.Contains(got, "clientCode();") {
		t.Fatalf("SERVER context: expected serverCode kept and clientCode dropped, got %q", got)
	}
}

func TestApplyMode(t *testing.T) {
	src := "@lx:<mode DEV:\ndevCode();\n@lx:mode>\nrest"

	c := &Compiler{mode: "DEV"}
	got := c.applyMode(src)
	if got != "devCode();\n\nrest" {
		t.Fatalf("DEV mode: unexpected result: %q", got)
	}

	c = &Compiler{mode: "PROD"}
	got = c.applyMode(src)
	if got != "\nrest" {
		t.Fatalf("PROD mode: unexpected result: %q", got)
	}
}

func TestCutCoordinationDirectives(t *testing.T) {
	c := &Compiler{}
	src := "before @lx:module myMod; middle @lx:module-data: {\"a\":1}; after"
	got := c.cutCoordinationDirectives(src)
	if got != "before  middle  after" {
		t.Fatalf("unexpected result: %q", got)
	}
}

func TestClearI18n(t *testing.T) {
	got := clearI18n(`x = lx.i18n('module-mymod-greeting');`)
	if got != `x = 'greeting';` {
		t.Fatalf("unexpected result: %q", got)
	}

	got = clearI18n(`x = lx.i18n(plain);`)
	if got != `x = 'plain';` {
		t.Fatalf("unexpected result for bare identifier: %q", got)
	}
}

// TestClearI18n_WithPlaceholdersArg is a regression test: clearI18n used to
// match a call's key with a single regex requiring the call to end right
// after it ([\w\d_\-.]+ then a literal ")"), so a call with a second,
// placeholders argument (lx.i18n(key, {...})) never matched at all and was
// left completely untouched in the output - a bare identifier reference
// (or, for a quoted key, a call encoding/json-invalid to eval) rather than
// the documented "falls back to the key as a plain string" behavior.
func TestClearI18n_WithPlaceholdersArg(t *testing.T) {
	got := clearI18n(`x = lx.i18n(applyChip.fill, {points: totalPoints});`)
	if got != `x = 'applyChip.fill';` {
		t.Fatalf("unexpected result for bare identifier with placeholders: %q", got)
	}

	got = clearI18n(`x = lx.i18n('module-mymod-greeting', {name: user.name});`)
	if got != `x = 'greeting';` {
		t.Fatalf("unexpected result for quoted key with placeholders: %q", got)
	}
}

// TestApplyI18n_ModuleMissKeyStillResolvedByPluginI18n is a regression
// test: applyI18n used to run the module-level translation source
// (modulesI18n) and the app/plugin-level one (i18n) as two independent
// Localize passes, one after the other. A key present ONLY in the second
// source but not the first (a very ordinary case - module data and
// plugin/app data cover different keys) got "resolved" by the first pass's
// own fallback (no translation found - replace with the bare key) before
// the second pass ever got a chance to look at it, since by then the call
// had already been rewritten from lx.i18n(...) into a plain string.
func TestApplyI18n_ModuleMissKeyStillResolvedByPluginI18n(t *testing.T) {
	c := &Compiler{
		modulesI18n: map[string]map[string]string{
			"en-EN": {"module-Some-key": "irrelevant"},
		},
		i18n: i18n.NewI18nMap(map[string]map[string]string{
			"en-EN": {"root.pointsTable": "Score table"},
		}),
	}
	got, err := c.applyI18n(`a = lx.i18n(root.pointsTable);`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != `a = 'Score table';` {
		t.Fatalf("got %q, want the plugin-level translation applied", got)
	}
}

func TestDeepCopyMap(t *testing.T) {
	src := map[string]any{
		"a": 1,
		"b": map[string]any{"c": 2},
	}
	got := deepCopyMap(src)
	if !reflect.DeepEqual(got, src) {
		t.Fatalf("expected an equal copy, got %#v", got)
	}

	// Mutating the nested map in the copy must not affect the original.
	got["b"].(map[string]any)["c"] = 99
	if src["b"].(map[string]any)["c"] != 2 {
		t.Fatalf("deepCopyMap did not actually deep-copy the nested map")
	}

	if deepCopyMap(nil) != nil {
		t.Fatalf("expected deepCopyMap(nil) to return nil")
	}
}

func TestMergeRecursive(t *testing.T) {
	dst := map[string]any{
		"a": 1,
		"nested": map[string]any{
			"x": 1,
			"y": 2,
		},
	}
	src := map[string]any{
		"a": 2, // overwrite scalar
		"b": 3, // new key
		"nested": map[string]any{
			"y": 20, // overwrite nested scalar
			"z": 30, // new nested key
		},
	}
	mergeRecursive(dst, src)

	want := map[string]any{
		"a": 2,
		"b": 3,
		"nested": map[string]any{
			"x": 1,
			"y": 20,
			"z": 30,
		},
	}
	if !reflect.DeepEqual(dst, want) {
		t.Fatalf("mergeRecursive result = %#v, want %#v", dst, want)
	}
}

func TestMergeRecursive_NilArgsAreNoOps(t *testing.T) {
	// Must not panic.
	mergeRecursive(nil, map[string]any{"a": 1})
	dst := map[string]any{"a": 1}
	mergeRecursive(dst, nil)
	if len(dst) != 1 {
		t.Fatalf("expected dst unchanged, got %#v", dst)
	}
}

func TestProcessLxml_PlainTemplate(t *testing.T) {
	pp := newFakePreprocessor()
	c := &Compiler{pp: pp}

	got := c.processLxml("before lx.ml(`<lx.Box> #text('hi')`) after")
	if got == "before  after" || got == "" {
		t.Fatalf("expected the lx.ml(...) call to be replaced with compiled LXML, got %q", got)
	}
	if len(pp.errs) != 0 {
		t.Fatalf("expected no errors, got %v", pp.errs)
	}
}

func TestProcessLxml_AssignmentAbsorption(t *testing.T) {
	pp := newFakePreprocessor()
	c := &Compiler{pp: pp}

	got := c.processLxml("const tree = lx.ml(`<lx.Box>`);")
	if got == "" {
		t.Fatalf("expected non-empty output")
	}
	// The "const tree = " prefix must be absorbed into the generated
	// output (the LXML compiler assigns to it), not left dangling before a
	// separate expression statement.
	if !strings.HasPrefix(got, "const tree=") {
		t.Fatalf("expected the const assignment absorbed at the start of the output, got %q", got)
	}
}

func TestProcessLxml_EscapedBacktick(t *testing.T) {
	pp := newFakePreprocessor()
	c := &Compiler{pp: pp}

	// The escaped backtick inside the template must not be treated as the
	// template's closing delimiter.
	got := c.processLxml("lx.ml(`<lx.Box> #text('a\\`b')`)")
	if len(pp.errs) != 0 {
		t.Fatalf("expected no errors (escaped backtick should not end the template early), got %v", pp.errs)
	}
	if got == "" {
		t.Fatalf("expected non-empty output")
	}
}

func TestProcessLxml_UnterminatedTemplate_LogsError(t *testing.T) {
	pp := newFakePreprocessor()
	c := &Compiler{pp: pp}

	c.processLxml("lx.ml(`<lx.Box> #text('unterminated')")
	if len(pp.errs) == 0 {
		t.Fatalf("expected an error for an unterminated template literal")
	}
}

func TestProcessLxml_NotAFunctionCallIsLeftAlone(t *testing.T) {
	pp := newFakePreprocessor()
	c := &Compiler{pp: pp}

	// "xlx.ml(" - "lx.ml(" is preceded by an identifier byte, so this must
	// not be treated as a real lx.ml(...) call.
	src := "xlx.ml(`abc`)"
	got := c.processLxml(src)
	if got != src {
		t.Fatalf("expected the identifier-prefixed occurrence to be left untouched, got %q", got)
	}
}
