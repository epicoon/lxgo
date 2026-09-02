package compiler

import (
	"strings"
	"testing"
)

func TestApplyExtendedSyntax_SelfKey(t *testing.T) {
	got, err := applyExtendedSyntax("x = lx.self(SOME_KEY);", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "x = this.constructor.SOME_KEY;" {
		t.Fatalf("unexpected result: %q", got)
	}
}

func TestApplyExtendedSyntax_ChainedFindGet(t *testing.T) {
	// lx(elem)>>child>grandchild => .find('child').get('grandchild')
	got, err := applyExtendedSyntax("lx(elem)>>child>grandchild", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "elem.find('child').get('grandchild')"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestApplyExtendedSyntax_SingleGet(t *testing.T) {
	// lx(elem)>child => .get('child') (single >, no leading >>)
	got, err := applyExtendedSyntax("lx(elem)>child", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "elem.get('child')"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestApplyExtendedSyntax_Const(t *testing.T) {
	src := "class Foo {\n@lx:const BAR = 42;\n}"
	got, err := applyExtendedSyntax(src, "test.js")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "static get BAR(){return 42;}") {
		t.Fatalf("expected @lx:const rewritten to a static getter, got %q", got)
	}
	if strings.Contains(got, "@lx:const") {
		t.Fatalf("expected the @lx:const directive stripped, got %q", got)
	}
}

// TestApplyExtendedSyntax_Const_NoTrailingSemicolon checks that a trailing
// ";" is optional - its presence or absence must not change the result.
func TestApplyExtendedSyntax_Const_NoTrailingSemicolon(t *testing.T) {
	src := "class Foo {\n@lx:const BAR = 42\n}"
	got, err := applyExtendedSyntax(src, "test.js")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "static get BAR(){return 42;}") {
		t.Fatalf("expected @lx:const rewritten to a static getter, got %q", got)
	}
}

// TestApplyExtendedSyntax_Const_MultilineMarker checks the "@lx:const alone
// on its own line, NAME = value on the next" shape.
func TestApplyExtendedSyntax_Const_MultilineMarker(t *testing.T) {
	src := "class Foo {\n@lx:const\nBAR = 42;\n}"
	got, err := applyExtendedSyntax(src, "test.js")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "static get BAR(){return 42;}") {
		t.Fatalf("expected @lx:const rewritten to a static getter, got %q", got)
	}
}

// TestApplyExtendedSyntax_Const_StringValueWithInternalSemicolon is a
// regression test: the old regex-based value capture ([^;]+?) stopped at
// the FIRST ";" it saw, so a string value containing its own ";" (a
// perfectly normal thing for a string to contain) got truncated instead of
// running to its own closing quote.
func TestApplyExtendedSyntax_Const_StringValueWithInternalSemicolon(t *testing.T) {
	src := `class Foo {` + "\n" + `@lx:const STR = "a b; c d";` + "\n" + `}`
	got, err := applyExtendedSyntax(src, "test.js")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `static get STR(){return "a b; c d";}`
	if !strings.Contains(got, want) {
		t.Fatalf("expected %q in result, got %q", want, got)
	}
}

// TestApplyExtendedSyntax_Const_MultilineArrayValue checks a multi-line,
// nested array value - the old regex-based capture couldn't span lines or
// track bracket nesting at all.
func TestApplyExtendedSyntax_Const_MultilineArrayValue(t *testing.T) {
	src := "class Foo {\n@lx:const\nA = [\n\t[1, 2],\n\t[3, 4]\n]\n}"
	got, err := applyExtendedSyntax(src, "test.js")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "static get A(){return [\n\t[1, 2],\n\t[3, 4]\n];}"
	if !strings.Contains(got, want) {
		t.Fatalf("expected %q in result, got %q", want, got)
	}
}

// TestApplyExtendedSyntax_Const_MultilineObjectValue checks a multi-line
// object value with nested arrays/objects of its own.
func TestApplyExtendedSyntax_Const_MultilineObjectValue(t *testing.T) {
	src := "class Foo {\n@lx:const\nA = {\n\ta: [1, 2],\n\tb: [3, 4],\n\tc: {e: 123},\n}\n}"
	got, err := applyExtendedSyntax(src, "test.js")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "static get A(){return {\n\ta: [1, 2],\n\tb: [3, 4],\n\tc: {e: 123},\n};}"
	if !strings.Contains(got, want) {
		t.Fatalf("expected %q in result, got %q", want, got)
	}
}

// TestApplyExtendedSyntax_Const_MultipleDeclarations checks that several
// @lx:const declarations in the same class - mixing shapes - are all found
// and rewritten, not just the first one.
func TestApplyExtendedSyntax_Const_MultipleDeclarations(t *testing.T) {
	src := "class Foo {\n@lx:const A = 1;\n@lx:const B = 2;\n}"
	got, err := applyExtendedSyntax(src, "test.js")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "static get A(){return 1;}") || !strings.Contains(got, "static get B(){return 2;}") {
		t.Fatalf("expected both constants rewritten, got %q", got)
	}
}

// TestApplyExtendedSyntax_Const_UnterminatedStringIsAnError checks that a
// malformed value (a string that never closes) is reported as an error
// instead of silently producing broken output.
func TestApplyExtendedSyntax_Const_UnterminatedStringIsAnError(t *testing.T) {
	src := "class Foo {\n@lx:const BAR = \"never closed\n}"
	if _, err := applyExtendedSyntax(src, "test.js"); err == nil {
		t.Fatal("expected an error for the unterminated string value, got nil")
	}
}

func TestApplyExtendedSyntax_Behavior(t *testing.T) {
	src := "class Foo extends Bar {\n@lx:behavior SomeBehavior;\n}"
	got, err := applyExtendedSyntax(src, "test.js")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "static __injectBehaviors(){SomeBehavior.injectInto(this);}") {
		t.Fatalf("expected @lx:behavior rewritten to __injectBehaviors(), got %q", got)
	}
	if strings.Contains(got, "@lx:behavior") {
		t.Fatalf("expected the @lx:behavior directive stripped, got %q", got)
	}
}

func TestApplyExtendedSyntax_Behaviors_Plural_MultipleNames(t *testing.T) {
	src := "class Foo extends Bar {\n@lx:behaviors First, Second;\n}"
	got, err := applyExtendedSyntax(src, "test.js")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "static __injectBehaviors(){First.injectInto(this);Second.injectInto(this);}"
	if !strings.Contains(got, want) {
		t.Fatalf("expected both behaviors injected in order, got %q", got)
	}
}

// TestApplyExtendedSyntax_Behavior_NoExtends_LeftAlone is a regression test:
// @lx:behavior only makes sense on a class whose ancestry reaches
// lx.Object.__afterDefinition (which calls __injectBehaviors() if present) -
// generating that static method on a class with no "extends" at all would
// define a method nothing ever calls, silently no-op'ing the behavior.
func TestApplyExtendedSyntax_Behavior_NoExtends_LeftAlone(t *testing.T) {
	src := "class Foo {\n@lx:behavior SomeBehavior;\n}"
	got, err := applyExtendedSyntax(src, "test.js")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(got, "__injectBehaviors") {
		t.Fatalf("expected @lx:behavior left untouched on a class with no extends, got %q", got)
	}
}

func TestApplyExtendedSyntax_Namespace(t *testing.T) {
	src := "@lx:namespace lx;\nclass Foo extends Bar {\n}"
	got, err := applyExtendedSyntax(src, "test.js")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "lx.createNamespace('lx')") {
		t.Fatalf("expected namespace registration code, got %q", got)
	}
	if !strings.Contains(got, "lx.globalContext.lx.Foo=Foo;") {
		t.Fatalf("expected the class registered on the namespace, got %q", got)
	}
	if strings.Contains(got, "@lx:namespace") {
		t.Fatalf("expected the @lx:namespace directive stripped, got %q", got)
	}
}

func TestApplyExtendedSyntax_NoClasses_LeavesCodeAlone(t *testing.T) {
	src := "const x = 1;"
	got, err := applyExtendedSyntax(src, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != src {
		t.Fatalf("expected unchanged code, got %q", got)
	}
}

func TestFindClasses(t *testing.T) {
	src := "@lx:namespace lx;\nclass Foo extends lx.Bar {\nmethod(){}\n}\nclass Plain {\n}"
	classes, err := findClasses(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(classes) != 2 {
		t.Fatalf("expected 2 classes found, got %d: %#v", len(classes), classes)
	}
	// classInfo.extends is computed via a lazy (non-greedy) regex capture
	// with nothing forcing it to consume the full identifier, so a
	// dotted/namespaced parent class name like "lx.Bar" only ever yields
	// its first character - a real quirk, but classInfo.extends is never
	// actually read anywhere in this package (grep confirms), so it has no
	// observable effect; documented here as-is rather than "fixed" as a
	// drive-by change while writing tests.
	if classes[0].name != "Foo" || classes[0].namespace != "lx" || classes[0].extends != "l" {
		t.Fatalf("unexpected first class info: %#v", classes[0])
	}
	if classes[1].name != "Plain" || classes[1].namespace != "" || classes[1].extends != "" {
		t.Fatalf("unexpected second class info: %#v", classes[1])
	}
}

// TestFindClasses_BraceInsideStringDoesNotEndClassBodyEarly is a
// regression test: the class body's closing brace is located via
// FindMatchingBrace, which used to count '}' characters with no awareness
// of string literals - a method body returning a string containing its
// own '}' would cut the class off before its real end.
func TestFindClasses_BraceInsideStringDoesNotEndClassBodyEarly(t *testing.T) {
	src := "class Foo {\nmethod(){return 'a } b';}\n}"
	classes, err := findClasses(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(classes) != 1 {
		t.Fatalf("expected 1 class found, got %d: %#v", len(classes), classes)
	}
	if classes[0].fullCode != src {
		t.Fatalf("expected the full class body (including the string's '}') captured, got %q", classes[0].fullCode)
	}
}

func TestFindClasses_NoClasses(t *testing.T) {
	classes, err := findClasses("const x = 1;")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if classes != nil {
		t.Fatalf("expected nil, got %#v", classes)
	}
}
