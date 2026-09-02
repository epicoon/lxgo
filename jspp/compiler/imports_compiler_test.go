package compiler

import (
	"reflect"
	"testing"
)

func TestSplitImportArgs(t *testing.T) {
	cases := []struct {
		name, in string
		want     []string
	}{
		{"empty", "", nil},
		{"single", "'a.js'", []string{"'a.js'"}},
		{"multiple", "'a.js', ModuleName, '-R'", []string{"'a.js'", "ModuleName", "'-R'"}},
		{"comma_inside_quotes_not_split", "'a,b.js', c", []string{"'a,b.js'", "c"}},
		{"comma_inside_brackets_not_split", "{a: 1, b: 2}, c", []string{"{a: 1, b: 2}", "c"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := splitImportArgs(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("splitImportArgs(%q) = %#v, want %#v", tc.in, got, tc.want)
			}
		})
	}
}

func TestParseImportArg_Path(t *testing.T) {
	pathArg, module, flag, isPath := parseImportArg("'foo.js'")
	if !isPath || module != "" || flag != nil {
		t.Fatalf("expected a plain path, got pathArg=%#v module=%q flag=%#v isPath=%v", pathArg, module, flag, isPath)
	}
	if pathArg.Path != "foo.js" {
		t.Fatalf("expected Path=foo.js, got %q", pathArg.Path)
	}
	if pathArg.Flags != (Flags{}) {
		t.Fatalf("expected no flags, got %#v", pathArg.Flags)
	}
}

func TestParseImportArg_PathWithGluedFlags(t *testing.T) {
	pathArg, _, _, isPath := parseImportArg("'-R foo.js'")
	if !isPath {
		t.Fatalf("expected a path")
	}
	if pathArg.Path != "foo.js" {
		t.Fatalf("expected the leading flag stripped from the path, got %q", pathArg.Path)
	}
	if !pathArg.Flags.Recursive {
		t.Fatalf("expected the Recursive flag set")
	}
}

func TestParseImportArg_StandaloneFlag(t *testing.T) {
	_, module, flag, isPath := parseImportArg("'-RF'")
	if isPath || module != "" {
		t.Fatalf("expected a standalone flag, not a path/module")
	}
	if flag == nil || !flag.Recursive || !flag.Force {
		t.Fatalf("expected Recursive+Force set, got %#v", flag)
	}
}

func TestParseImportArg_Module(t *testing.T) {
	pathArg, module, flag, isPath := parseImportArg("SomeModule")
	if isPath || flag != nil {
		t.Fatalf("expected a bare module reference")
	}
	if module != "SomeModule" {
		t.Fatalf("expected module=SomeModule, got %q", module)
	}
	if pathArg != nil {
		t.Fatalf("expected no path arg, got %#v", pathArg)
	}
}

func TestParseImportArg_Empty(t *testing.T) {
	pathArg, module, flag, isPath := parseImportArg("")
	if pathArg != nil || module != "" || flag != nil || isPath {
		t.Fatalf("expected all zero values for an empty arg")
	}
}

func TestFindImportCalls_PathsAndModules(t *testing.T) {
	code := `lx.import('a.js', SomeModule, '-R');`
	calls := findImportCalls(code)
	if len(calls) != 1 {
		t.Fatalf("expected exactly 1 call, got %d", len(calls))
	}
	call := calls[0]
	if len(call.paths) != 1 || call.paths[0].Path != "a.js" || !call.paths[0].Flags.Recursive {
		t.Fatalf("unexpected paths: %#v", call.paths)
	}
	if len(call.modules) != 1 || call.modules[0] != "SomeModule" {
		t.Fatalf("unexpected modules: %#v", call.modules)
	}
}

// TestFindImportCalls_ParenInsidePathIsNotMistakenForCallEnd is a
// regression test: findImportCalls locates the call's closing paren via
// FindMatchingBrace, which used to count ')' characters with no awareness
// of string literals - a path argument containing its own ')' (a real,
// if unusual, filename) would cut the call short, leaving the rest of the
// path and the actual closing paren as leftover code.
func TestFindImportCalls_ParenInsidePathIsNotMistakenForCallEnd(t *testing.T) {
	calls := findImportCalls(`lx.import('a)b.js');`)
	if len(calls) != 1 {
		t.Fatalf("expected exactly 1 call, got %d: %#v", len(calls), calls)
	}
	if len(calls[0].paths) != 1 || calls[0].paths[0].Path != "a)b.js" {
		t.Fatalf("unexpected paths: %#v", calls[0].paths)
	}
}

func TestFindImportCalls_MultipleCalls(t *testing.T) {
	code := `lx.import(A); const x = 1; lx.import(B);`
	calls := findImportCalls(code)
	if len(calls) != 2 {
		t.Fatalf("expected 2 calls, got %d: %#v", len(calls), calls)
	}
	if calls[0].modules[0] != "A" || calls[1].modules[0] != "B" {
		t.Fatalf("unexpected module order: %#v", calls)
	}
}

func TestFindImportCalls_IdentifierPrefixedOccurrenceIgnored(t *testing.T) {
	// "xlx.import(" - preceded by an identifier byte, must not count.
	calls := findImportCalls("xlx.import(A);")
	if len(calls) != 0 {
		t.Fatalf("expected no calls found, got %#v", calls)
	}
}

func TestFindImportCalls_UnmatchedParenIsSkipped(t *testing.T) {
	calls := findImportCalls("lx.import(A")
	if len(calls) != 0 {
		t.Fatalf("expected no calls found for an unterminated call, got %#v", calls)
	}
}

func TestFindImportCalls_NoCalls(t *testing.T) {
	calls := findImportCalls("const x = 1;")
	if calls != nil {
		t.Fatalf("expected nil, got %#v", calls)
	}
}

// TestFindImportCalls_CommentedOutArgumentIsDropped is a regression test:
// findImportCalls used to read a module file's raw text with no comment
// awareness at all - a commented-out argument inside an otherwise-real call
// (left over from someone testing whether a module was still needed) was
// read literally, "//" included, as a bare module name that then failed to
// resolve ("Module '// Name' does not exist").
func TestFindImportCalls_CommentedOutArgumentIsDropped(t *testing.T) {
	code := "lx.import(\n\t// Name,\n\t'dir/'\n);"
	calls := findImportCalls(code)
	if len(calls) != 1 {
		t.Fatalf("expected exactly 1 call, got %d: %#v", len(calls), calls)
	}
	if len(calls[0].modules) != 0 {
		t.Fatalf("expected the commented-out module dropped, got: %#v", calls[0].modules)
	}
	if len(calls[0].paths) != 1 || calls[0].paths[0].Path != "dir/" {
		t.Fatalf("expected the real path argument kept, got: %#v", calls[0].paths)
	}
}

// TestFindImportCalls_FullyCommentedCallIsIgnored is a regression test for
// the same underlying gap as above, one level up: an entire lx.import(...)
// statement commented out on one line must not be found as a real call at
// all.
func TestFindImportCalls_FullyCommentedCallIsIgnored(t *testing.T) {
	calls := findImportCalls("// lx.import(Name);\nlx.import(Real);")
	if len(calls) != 1 {
		t.Fatalf("expected exactly 1 call (the real one), got %d: %#v", len(calls), calls)
	}
	if len(calls[0].modules) != 1 || calls[0].modules[0] != "Real" {
		t.Fatalf("unexpected modules: %#v", calls[0].modules)
	}
}

// TestFindImportCalls_OffsetsStayValidAgainstOriginalCode is a regression
// test: findImportCalls must return start/end offsets usable to splice the
// CALLER's original, unstripped code (see processImport) - a comment-aware
// scan that shrinks the text before searching it would throw those off.
func TestFindImportCalls_OffsetsStayValidAgainstOriginalCode(t *testing.T) {
	code := "const x = 1; // a comment\nlx.import(Real);"
	calls := findImportCalls(code)
	if len(calls) != 1 {
		t.Fatalf("expected exactly 1 call, got %d: %#v", len(calls), calls)
	}
	call := calls[0]
	if code[call.start:call.end] != "lx.import(Real);" && code[call.start:call.end] != "lx.import(Real)" {
		t.Fatalf("start/end don't slice out the real call from the original code: %q", code[call.start:call.end])
	}
}
