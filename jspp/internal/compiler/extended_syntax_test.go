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

func TestFindClasses_NoClasses(t *testing.T) {
	classes, err := findClasses("const x = 1;")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if classes != nil {
		t.Fatalf("expected nil, got %#v", classes)
	}
}
