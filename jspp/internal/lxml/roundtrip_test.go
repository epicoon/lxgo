package lxml_test

import (
	"strings"
	"testing"

	"github.com/epicoon/lxgo/jspp/internal/lxml"
)

func parseLxml(t *testing.T, src string) string {
	t.Helper()
	pp := newTestPreprocessor(t)
	code, err := lxml.NewParser(pp).ParseText(src)
	if err != nil {
		t.Fatalf("ParseText(%q): %v", src, err)
	}
	return code
}

func TestWidget_KeyAndCssAndGeom(t *testing.T) {
	code := parseLxml(t, "<lx.Box> @myKey .myCss [10:20:30:40]\n")

	for _, want := range []string{`key:"myKey"`, `css:["myCss"]`, `geom:[10,20,30,40]`} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in compiled code, got: %s", want, code)
		}
	}
}

func TestWidget_Config(t *testing.T) {
	code := parseLxml(t, "<lx.Box> (width:'100%')\n")
	if !strings.Contains(code, "width:'100%'") {
		t.Fatalf("expected the (config) content inlined, got: %s", code)
	}
}

func TestWidget_Data(t *testing.T) {
	code := parseLxml(t, "<lx.Box> {a:1}\n")
	if !strings.Contains(code, "data:{a:1}") {
		t.Fatalf("expected the {data} content wrapped as data:{...}, got: %s", code)
	}
}

func TestWidget_MethodCall(t *testing.T) {
	code := parseLxml(t, "<lx.Box> #method('hi')\n")
	if !strings.Contains(code, ".method('hi');") {
		t.Fatalf("expected the #method(...) call compiled, got: %s", code)
	}
}

func TestWidget_Field(t *testing.T) {
	code := parseLxml(t, "<lx.Box> [f:myField]\n")
	if !strings.Contains(code, `field:"myField"`) {
		t.Fatalf("expected field:\"myField\" in compiled code, got: %s", code)
	}
}

func TestWidget_VolumeGeom(t *testing.T) {
	code := parseLxml(t, "<lx.Box> [_]\n")
	if !strings.Contains(code, "geom:true") {
		t.Fatalf("expected geom:true for [_] volume marker, got: %s", code)
	}
}

func TestSyntax_IfElseifElse(t *testing.T) {
	src := "<*root>\n" +
		"    if cond1\n" +
		"        <lx.Box> @a\n" +
		"    elseif cond2\n" +
		"        <lx.Box> @b\n" +
		"    else\n" +
		"        <lx.Box> @c\n" +
		"<&root>\n"
	code := parseLxml(t, src)

	for _, want := range []string{"if (cond1)", "else if (cond2)", "else {", `key:"a"`, `key:"b"`, `key:"c"`} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in compiled if/elseif/else code, got: %s", want, code)
		}
	}
}

func TestSyntax_For(t *testing.T) {
	// "for 3:" - the "for lim:" sugar form: iterate _iter from 0 to 3
	// inclusive (see SyntaxParser.Run's "for" handling).
	src := "<*root>\n" +
		"    for 3:\n" +
		"        <lx.Box> @item\n" +
		"<&root>\n"
	code := parseLxml(t, src)

	if !strings.Contains(code, "let _iter=0") || !strings.Contains(code, "_iter<=3") || !strings.Contains(code, "_iter++") {
		t.Fatalf("expected a for(let _iter=0;_iter<=3;_iter++) loop in compiled code, got: %s", code)
	}
	if !strings.Contains(code, `key:"item"`) {
		t.Fatalf("expected the loop body compiled, got: %s", code)
	}
}

func TestParseText_MalformedBlockSyntax_Errors(t *testing.T) {
	pp := newTestPreprocessor(t)
	// Neither * nor & after < - invalid block syntax.
	_, err := lxml.NewParser(pp).ParseText("<%bad>\n")
	if err == nil {
		t.Fatalf("expected an error for malformed block syntax")
	}
}
