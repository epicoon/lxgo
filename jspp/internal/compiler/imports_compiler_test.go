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
