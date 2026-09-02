package compiler_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/epicoon/lxgo/jspp"
	"github.com/epicoon/lxgo/jspp/component"
	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/apptest"
)

// capturingLogger is a kernel.ILogger that just remembers the last message
// passed to each method - kernel.IApp.LogError falls back to the stdlib log
// package (bound to os.Stderr at that package's own init time) when no
// logger is set, which a test-local os.Stderr reassignment can't intercept;
// installing this via app.SetLogger before building the component is the
// reliable way to observe what got logged.
type capturingLogger struct {
	lastError string
}

func (l *capturingLogger) Log(msg, category string)        {}
func (l *capturingLogger) LogWarning(msg, category string) {}
func (l *capturingLogger) LogError(msg, category string)   { l.lastError = msg }

func newTestPreprocessor(t *testing.T) jspp.IPreprocessor {
	t.Helper()
	sysPath := t.TempDir()
	app, err := apptest.New(kernel.Dict{
		"Components": kernel.Dict{
			"JSPreprocessor": kernel.Dict{
				"SysPath":  sysPath,
				"MapsPath": sysPath,
			},
		},
	})
	if err != nil {
		t.Fatalf("apptest.New: %v", err)
	}
	if err := component.SetAppComponent(app, "Components.JSPreprocessor"); err != nil {
		t.Fatalf("SetAppComponent: %v", err)
	}
	pp, err := component.AppComponent(app)
	if err != nil {
		t.Fatalf("AppComponent: %v", err)
	}
	return pp
}

// TestCompiledModules_BuildModulesFalse_RecordsOnlyDirectlySeenNames checks
// the intentional shape of the BuildModules(false) pattern used by
// compileMainJs (plugins/plugin_renderer.go): it records the module names a
// bare lx.import(Name) call names directly in the compiled code, without
// reading those modules' own files to discover further transitive bare
// imports - that recursive expansion happens later, at runtime, via the
// BuildModules(true) round-trip (see TestCompiledModules_RuntimeNeedRoundTrip_ResolvesTransitively).
func TestCompiledModules_BuildModulesFalse_RecordsOnlyDirectlySeenNames(t *testing.T) {
	pp := newTestPreprocessor(t)
	modsDir := t.TempDir()

	innerPath := filepath.Join(modsDir, "Inner.js")
	if err := os.WriteFile(innerPath, []byte("@lx:module Inner;\nclass Inner {}\n"), 0644); err != nil {
		t.Fatalf("write Inner.js: %v", err)
	}
	outerPath := filepath.Join(modsDir, "Outer.js")
	if err := os.WriteFile(outerPath, []byte("@lx:module Outer;\nlx.import(Inner);\nclass Outer {}\n"), 0644); err != nil {
		t.Fatalf("write Outer.js: %v", err)
	}

	mm := pp.ModulesMap()
	if err := mm.Save([]jspp.IJSModuleData{
		mm.NewData("Inner", innerPath),
		mm.NewData("Outer", outerPath),
	}); err != nil {
		t.Fatalf("Save modules map: %v", err)
	}

	c := pp.CompilerBuilder().
		BuildModules(false).
		SetClientContext().
		SetUnwrapped().
		SetCode("lx.import(Outer);class Plugin extends lx.Plugin {}").
		Compiler()
	if _, err := c.Run(); err != nil {
		t.Fatalf("compile: %v", err)
	}

	if got := c.CompiledModules(); len(got) != 1 || got[0] != "Outer" {
		t.Fatalf("expected CompiledModules()=[Outer] (Inner is Outer's own concern, resolved later at runtime), got %v", got)
	}
}

// TestCompiledModules_RuntimeNeedRoundTrip_ResolvesTransitively is the other
// half of the pattern above: a BuildModules(true) (the default) compile pass
// over just the names a BuildModules(false) pass recorded DOES read those
// modules' own files and recursively pull in whatever they themselves
// bare-import - matching what ServiceHandler.modulesCodeResponse does for a
// client's runtime "need" request.
func TestCompiledModules_RuntimeNeedRoundTrip_ResolvesTransitively(t *testing.T) {
	pp := newTestPreprocessor(t)
	modsDir := t.TempDir()

	innerPath := filepath.Join(modsDir, "Inner.js")
	if err := os.WriteFile(innerPath, []byte("@lx:module Inner;\nclass Inner {}\n"), 0644); err != nil {
		t.Fatalf("write Inner.js: %v", err)
	}
	outerPath := filepath.Join(modsDir, "Outer.js")
	if err := os.WriteFile(outerPath, []byte("@lx:module Outer;\nlx.import(Inner);\nclass Outer {}\n"), 0644); err != nil {
		t.Fatalf("write Outer.js: %v", err)
	}

	mm := pp.ModulesMap()
	if err := mm.Save([]jspp.IJSModuleData{
		mm.NewData("Inner", innerPath),
		mm.NewData("Outer", outerPath),
	}); err != nil {
		t.Fatalf("Save modules map: %v", err)
	}

	c := pp.CompilerBuilder().
		SetClientContext().
		SetCode("lx.import(Outer);").
		Compiler()
	code, err := c.Run()
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	got := c.CompiledModules()
	if len(got) != 2 {
		t.Fatalf("expected both Outer and its transitively-needed Inner in CompiledModules(), got %v", got)
	}
	if !strings.Contains(code, "class Inner") {
		t.Fatalf("expected Inner's own compiled code inlined in the result, got: %s", code)
	}
}

// TestModulesCode_DependencyEmittedBeforeDependent is a regression test:
// checkModule used to append a module to modulesForBuild/filePaths BEFORE
// recursing into that module's own lx.import(...) dependencies, so the
// compiled bundle put every module ahead of whatever it depended on. A
// module extending another (e.g. "class Radio extends lx.Checkbox") then
// hit a ReferenceError at runtime, since its parent class wasn't defined
// yet when the "extends" ran.
func TestModulesCode_DependencyEmittedBeforeDependent(t *testing.T) {
	pp := newTestPreprocessor(t)
	modsDir := t.TempDir()

	basePath := filepath.Join(modsDir, "Base.js")
	baseCode := "@lx:module lx.Base;\n@lx:namespace lx;\nclass Base {}\n"
	if err := os.WriteFile(basePath, []byte(baseCode), 0644); err != nil {
		t.Fatalf("write Base.js: %v", err)
	}
	derivedPath := filepath.Join(modsDir, "Derived.js")
	derivedCode := "@lx:module lx.Derived;\nlx.import(lx.Base);\n@lx:namespace lx;\nclass Derived extends lx.Base {}\n"
	if err := os.WriteFile(derivedPath, []byte(derivedCode), 0644); err != nil {
		t.Fatalf("write Derived.js: %v", err)
	}

	mm := pp.ModulesMap()
	if err := mm.Save([]jspp.IJSModuleData{
		mm.NewData("lx.Base", basePath),
		mm.NewData("lx.Derived", derivedPath),
	}); err != nil {
		t.Fatalf("Save modules map: %v", err)
	}

	c := pp.CompilerBuilder().
		SetClientContext().
		SetCode("lx.import(lx.Derived);").
		Compiler()
	code, err := c.Run()
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	base := strings.Index(code, "class Base")
	derived := strings.Index(code, "class Derived")
	if base == -1 || derived == -1 {
		t.Fatalf("expected both classes in the compiled code, got: %s", code)
	}
	if base >= derived {
		t.Fatalf("expected 'class Base' before 'class Derived' in the compiled code, got: %s", code)
	}
}

// TestModulesCode_DependencyEmittedBeforeDependent_ThroughNestedRediscovery
// is a regression test for a subtler variant of the same bug fixed by
// TestModulesCode_DependencyEmittedBeforeDependent: each file compileFileGroup
// compiles runs its OWN, separate lx.import(...) resolution pass over its own
// raw text (needed since a widget referenced only inside an lx.ml(...)
// template only becomes a literal lx.import(...) once that file's own LXML
// gets expanded). Before checkModule's "already being resolved" guard became
// a Compiler-level field, that per-file pass used a fresh guard every time,
// so the SAME dependency (here, ParentLike, reachable through ChildLike)
// could be independently rediscovered and recompiled as its own, separate,
// disconnected file group - one whose position in the final bundle depended
// on unrelated file iteration order rather than on being a dependency of
// ChildLike, occasionally landing after it.
func TestModulesCode_DependencyEmittedBeforeDependent_ThroughNestedRediscovery(t *testing.T) {
	pp := newTestPreprocessor(t)
	modsDir := t.TempDir()

	write := func(name, code string) string {
		p := filepath.Join(modsDir, name+".js")
		if err := os.WriteFile(p, []byte(code), 0644); err != nil {
			t.Fatalf("write %s.js: %v", name, err)
		}
		return p
	}

	parentPath := write("ParentLike", "@lx:module lx.ParentLike;\n@lx:namespace lx;\nclass ParentLike {}\n")
	childPath := write("ChildLike", "@lx:module lx.ChildLike;\nlx.import(lx.ParentLike);\n@lx:namespace lx;\nclass ChildLike extends lx.ParentLike {}\n")
	sibling1Path := write("Sibling1", "@lx:module lx.Sibling1;\n@lx:namespace lx;\nclass Sibling1 {}\n")
	sibling2Path := write("Sibling2", "@lx:module lx.Sibling2;\n@lx:namespace lx;\nclass Sibling2 {}\n")
	outerPath := write("Outer", "@lx:module lx.Outer;\nlx.import(lx.Sibling1, lx.Sibling2, lx.ChildLike);\n@lx:namespace lx;\nclass Outer {}\n")

	mm := pp.ModulesMap()
	if err := mm.Save([]jspp.IJSModuleData{
		mm.NewData("lx.ParentLike", parentPath),
		mm.NewData("lx.ChildLike", childPath),
		mm.NewData("lx.Sibling1", sibling1Path),
		mm.NewData("lx.Sibling2", sibling2Path),
		mm.NewData("lx.Outer", outerPath),
	}); err != nil {
		t.Fatalf("Save modules map: %v", err)
	}

	for i := 0; i < 10; i++ {
		c := pp.CompilerBuilder().
			SetClientContext().
			SetCode("lx.import(lx.Outer);").
			Compiler()
		code, err := c.Run()
		if err != nil {
			t.Fatalf("compile: %v", err)
		}

		parent := strings.Index(code, "class ParentLike")
		child := strings.Index(code, "class ChildLike")
		if parent == -1 || child == -1 {
			t.Fatalf("expected both classes in the compiled code, got: %s", code)
		}
		if parent >= child {
			t.Fatalf("expected 'class ParentLike' before 'class ChildLike' in the compiled code, got: %s", code)
		}
	}
}

// TestModulesCode_PathImportedFileExtendingUnimportedSecondaryClass is a
// regression test: a file pulled in via a path-based lx.import('some/file.js')
// (not a bare module name) can "extend" a class that's declared in a
// completely different, bare-name-importable module file but isn't that
// file's own @lx:module namesake (e.g. WebSocketClient.js also declares
// lx.socket.EventListener) - with no lx.import(...) edge naming that class
// anywhere. compileFileGroup's dependency linking used to only look for the
// parent class among the files already in its own batch, so a class reached
// this way (secondary class in a module file, resolvable in the modules map
// - see checkModulePath in maps_builder.go - but never explicitly imported)
// had no way to be ordered before its dependent; whether it ended up before
// or after depended on unrelated compile-order timing.
func TestModulesCode_PathImportedFileExtendingUnimportedSecondaryClass(t *testing.T) {
	pp := newTestPreprocessor(t)
	dir := t.TempDir()

	multiPath := filepath.Join(dir, "Multi.js")
	multiCode := "@lx:module lx.util.Multi;\n\n" +
		"@lx:namespace lx.util;\nclass Multi {}\n\n" +
		"@lx:namespace lx.util;\nclass Secondary {}\n"
	if err := os.WriteFile(multiPath, []byte(multiCode), 0644); err != nil {
		t.Fatalf("write Multi.js: %v", err)
	}

	dependentPath := filepath.Join(dir, "Dependent.js")
	dependentCode := "@lx:namespace lx.util;\nclass Dependent extends lx.util.Secondary {}\n"
	if err := os.WriteFile(dependentPath, []byte(dependentCode), 0644); err != nil {
		t.Fatalf("write Dependent.js: %v", err)
	}

	mm := pp.ModulesMap()
	if err := mm.Save([]jspp.IJSModuleData{
		mm.NewData("lx.util.Multi", multiPath),
		// checkModulePath (maps_builder.go) would register this secondary
		// class the same way when scanning Multi.js for its modules map -
		// reproduced by hand here since this test builds the map directly.
		mm.NewData("lx.util.Secondary", multiPath),
	}); err != nil {
		t.Fatalf("Save modules map: %v", err)
	}

	for i := 0; i < 10; i++ {
		c := pp.CompilerBuilder().
			SetClientContext().
			SetCode("lx.import('" + dependentPath + "');").
			Compiler()
		code, err := c.Run()
		if err != nil {
			t.Fatalf("compile: %v", err)
		}

		secondary := strings.Index(code, "class Secondary")
		dependent := strings.Index(code, "class Dependent")
		if secondary == -1 || dependent == -1 {
			t.Fatalf("expected both classes in the compiled code, got: %s", code)
		}
		if secondary >= dependent {
			t.Fatalf("expected 'class Secondary' before 'class Dependent' in the compiled code, got: %s", code)
		}
	}
}

// TestModulesCode_LeafAlsoBareImportedBySibling_StillOrderedFirst is a
// regression test for the case the previous fallback (see
// TestModulesCode_PathImportedFileExtendingUnimportedSecondaryClass) didn't
// yet cover: a "grandparent" file bare-imports a leaf class-defining module
// directly (e.g. Tools.js importing lx.socket.WebSocketClient) *and*
// path-imports a subdirectory whose contents, several nesting levels down,
// extend a class from that same leaf module (e.g. Environment.js, itself
// only reachable through the grandparent's own path import, further
// path-importing a directory whose file extends lx.socket.EventListener) -
// with no lx.import(...) edge of its own naming that leaf class. Since the
// leaf and the grandparent both land as untied siblings in the SAME
// compileFileGroup batch (the grandparent's own top-level bare-name
// resolution discovers the leaf directly), the leaf could sort AFTER the
// grandparent in that batch's own file loop - so by the time the deeply
// nested dependent file's fallback ran, checkModule() already found the
// leaf "claimed" (visited) by the grandparent's own resolution, even though
// the leaf's actual code hadn't been written out anywhere yet. The fallback
// used to treat "already visited" as "safe to skip", trusting it was
// already flushed - which doesn't hold when the visit and the flush are two
// different points in time separated by an unresolved sibling-ordering tie.
func TestModulesCode_LeafAlsoBareImportedBySibling_StillOrderedFirst(t *testing.T) {
	pp := newTestPreprocessor(t)
	dir := t.TempDir()

	write := func(rel, code string) string {
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatalf("mkdir for %s: %v", rel, err)
		}
		if err := os.WriteFile(p, []byte(code), 0644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
		return p
	}

	leafPath := write("Leaf.js", "@lx:module lx.util.Leaf;\n@lx:namespace lx.util;\nclass Leaf {}\n")
	grandparentPath := write("Grandparent.js",
		"@lx:module lx.util.Grandparent;\nlx.import(lx.util.Leaf, 'mid/');\n@lx:namespace lx.util;\nclass Grandparent {}\n")
	write("mid/Mid.js", "lx.import(lx.util.Leaf, 'deep/');\nclass Mid {}\n")
	write("mid/deep/Deep.js", "@lx:namespace lx.util;\nclass Deep extends lx.util.Leaf {}\n")

	mm := pp.ModulesMap()
	if err := mm.Save([]jspp.IJSModuleData{
		mm.NewData("lx.util.Leaf", leafPath),
		mm.NewData("lx.util.Grandparent", grandparentPath),
	}); err != nil {
		t.Fatalf("Save modules map: %v", err)
	}

	for i := 0; i < 20; i++ {
		c := pp.CompilerBuilder().
			SetClientContext().
			SetCode("lx.import(lx.util.Grandparent);").
			Compiler()
		code, err := c.Run()
		if err != nil {
			t.Fatalf("compile: %v", err)
		}

		leaf := strings.Index(code, "class Leaf")
		deep := strings.Index(code, "class Deep")
		if leaf == -1 || deep == -1 {
			t.Fatalf("expected both classes in the compiled code, got: %s", code)
		}
		if leaf >= deep {
			t.Fatalf("expected 'class Leaf' before 'class Deep' in the compiled code, got: %s", code)
		}
	}
}

// TestModulesCode_UnrelatedBareModuleSiblings_KeepInputOrder is a regression
// test for compileFileGroup's final sort: two bare-module files pulled into
// the SAME batch by checkModuleDependencies's pre-scan (here, ModuleB bare-
// imports ModuleA, so both land in one compileFileGroup call together)
// don't extend each other or share any class-extends relationship of their
// own, so they get an equal Counter - but each also path-imports its own
// directory, and one of those (ModuleB's) extends a class the other
// (ModuleA's) provides, with no lx.import(...) edge naming that class
// directly and no registration of it in the modules map at all, so neither
// the classesMap nor the ModulesMap fallback in the "Set dependencies" loop
// can establish an ordering edge between them. compileFileGroup used to
// build its final sorted-output order by ranging over a map - a step whose
// iteration order Go deliberately randomizes on every run - so two equal-
// Counter files like these came out in whichever order the map happened to
// yield that run, independent of the deterministic order
// checkModuleDependencies discovered them in (ModuleA before ModuleB, since
// it recurses into a bare dependency before appending the file that named
// it).
func TestModulesCode_UnrelatedBareModuleSiblings_KeepInputOrder(t *testing.T) {
	pp := newTestPreprocessor(t)
	dir := t.TempDir()

	write := func(rel, code string) string {
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatalf("mkdir for %s: %v", rel, err)
		}
		if err := os.WriteFile(p, []byte(code), 0644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
		return p
	}

	write("a/Base.js", "@lx:namespace lx.util;\nclass Base {}\n")
	moduleAPath := write("ModuleA.js", "@lx:module lx.util.ModuleA;\nlx.import('a/');\n")
	write("b/Derived.js", "@lx:namespace lx.util;\nclass Derived extends lx.util.Base {}\n")
	moduleBPath := write("ModuleB.js", "@lx:module lx.util.ModuleB;\nlx.import(lx.util.ModuleA, '-R b/');\n")

	mm := pp.ModulesMap()
	if err := mm.Save([]jspp.IJSModuleData{
		mm.NewData("lx.util.ModuleA", moduleAPath),
		mm.NewData("lx.util.ModuleB", moduleBPath),
	}); err != nil {
		t.Fatalf("Save modules map: %v", err)
	}

	for i := 0; i < 30; i++ {
		c := pp.CompilerBuilder().
			SetClientContext().
			SetCode("lx.import(lx.util.ModuleB);").
			Compiler()
		code, err := c.Run()
		if err != nil {
			t.Fatalf("run %d: compile: %v", i, err)
		}

		base := strings.Index(code, "class Base")
		derived := strings.Index(code, "class Derived")
		if base == -1 || derived == -1 {
			t.Fatalf("run %d: expected both classes in the compiled code, got: %s", i, code)
		}
		if base >= derived {
			t.Fatalf("run %d: expected 'class Base' before 'class Derived' in the compiled code, got: %s", i, code)
		}
	}
}

// TestModulesCode_ModuleI18n_CoversPathImportedSiblings is a regression
// test: a module's i18n data (@lx:module-data: i18n = ...) is loaded once
// for the whole module and stored under keys prefixed with the module's own
// name (module-<Name>-<key>) so different modules' short keys don't
// collide - but the matching prefix used to be inserted only into the
// lx.i18n(...) calls found in the ONE file that literally carries the
// @lx:module marker. A file pulled into the same module's build through a
// path argument of its own lx.import(...) (e.g. a GUI state class reached
// via '-R gui/') has no @lx:module of its own, so its own lx.i18n(...)
// calls never got the prefix inserted, and could never match anything
// stored under it - the translation directive fell back to the untranslated
// key literally shown to the user, no matter how correct the yaml file was.
func TestModulesCode_ModuleI18n_CoversPathImportedSiblings(t *testing.T) {
	pp := newTestPreprocessor(t)
	dir := t.TempDir()

	i18nPath := filepath.Join(dir, "main.yaml")
	if err := os.WriteFile(i18nPath, []byte("en:\n  greeting: Hello\n"), 0644); err != nil {
		t.Fatalf("write main.yaml: %v", err)
	}

	subDir := filepath.Join(dir, "sub")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("mkdir sub: %v", err)
	}
	// Other.js is pulled in by Entry.js's own 'sub/' path argument - it has
	// no @lx:module of its own, matching a GUI state class reached through
	// a module's '-R gui/' argument.
	otherCode := "class Other {\n\tm() { return lx.i18n(greeting); }\n}\n"
	if err := os.WriteFile(filepath.Join(subDir, "Other.js"), []byte(otherCode), 0644); err != nil {
		t.Fatalf("write Other.js: %v", err)
	}

	entryPath := filepath.Join(dir, "Entry.js")
	entryCode := "@lx:module lx.i18ntree.Entry;\nlx.import('sub/');\n@lx:namespace lx.i18ntree;\nclass Entry {}\n"
	if err := os.WriteFile(entryPath, []byte(entryCode), 0644); err != nil {
		t.Fatalf("write Entry.js: %v", err)
	}

	mm := pp.ModulesMap()
	entryData := mm.NewData("lx.i18ntree.Entry", entryPath)
	entryData.AddData("i18n", i18nPath)
	if err := mm.Save([]jspp.IJSModuleData{entryData}); err != nil {
		t.Fatalf("Save modules map: %v", err)
	}

	c := pp.CompilerBuilder().
		SetLang("en").
		SetClientContext().
		SetCode("lx.import(lx.i18ntree.Entry);").
		Compiler()
	code, err := c.Run()
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	if !strings.Contains(code, "'Hello'") {
		t.Fatalf("expected the path-imported sibling's lx.i18n(...) call translated to 'Hello', got: %s", code)
	}
	if strings.Contains(code, "lx.i18n(") {
		t.Fatalf("expected no lx.i18n(...) call left unresolved, got: %s", code)
	}
}

// TestModulesCode_CyclicSecondaryClassDependency_DoesNotHang is a regression
// test for the previous fallback fix itself (see
// TestModulesCode_LeafAlsoBareImportedBySibling_StillOrderedFirst): forcing
// an unresolved dependency's file to compile right away, unconditionally,
// hangs forever if two files' secondary classes extend each other (A's
// class extends a class declared in B, B's own class extends one declared
// in A) - resolving A's missing parent force-compiles B, whose own missing
// parent resolves back to A, which isn't finished yet (still inside its own
// "Set dependencies" pass one stack frame up) so isn't in c.compiledFiles
// either, forcing A again, forever. compileFileGroup must recognize a file
// already being compiled by an ancestor call on the stack and leave it
// alone instead of re-forcing it.
func TestModulesCode_CyclicSecondaryClassDependency_DoesNotHang(t *testing.T) {
	pp := newTestPreprocessor(t)
	dir := t.TempDir()

	aPath := filepath.Join(dir, "A.js")
	aCode := "@lx:module lx.cyc.A;\n" +
		"@lx:namespace lx.cyc;\nclass A extends lx.cyc.BHelper {}\n\n" +
		"@lx:namespace lx.cyc;\nclass AHelper {}\n"
	if err := os.WriteFile(aPath, []byte(aCode), 0644); err != nil {
		t.Fatalf("write A.js: %v", err)
	}

	bPath := filepath.Join(dir, "B.js")
	bCode := "@lx:module lx.cyc.B;\n" +
		"@lx:namespace lx.cyc;\nclass B extends lx.cyc.AHelper {}\n\n" +
		"@lx:namespace lx.cyc;\nclass BHelper {}\n"
	if err := os.WriteFile(bPath, []byte(bCode), 0644); err != nil {
		t.Fatalf("write B.js: %v", err)
	}

	mm := pp.ModulesMap()
	if err := mm.Save([]jspp.IJSModuleData{
		mm.NewData("lx.cyc.A", aPath),
		mm.NewData("lx.cyc.AHelper", aPath),
		mm.NewData("lx.cyc.B", bPath),
		mm.NewData("lx.cyc.BHelper", bPath),
	}); err != nil {
		t.Fatalf("Save modules map: %v", err)
	}

	done := make(chan struct{})
	var code string
	var err error
	go func() {
		c := pp.CompilerBuilder().
			SetClientContext().
			SetCode("lx.import(lx.cyc.A);").
			Compiler()
		code, err = c.Run()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("compile did not terminate within 5s - likely an infinite loop")
	}

	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if !strings.Contains(code, "class A ") || !strings.Contains(code, "class B ") {
		t.Fatalf("expected both classes in the compiled code, got: %s", code)
	}
}

// TestModulesCode_CommentedOutBareImport_IsIgnoredByPreScan is a regression
// test: checkModuleDependencies's pre-scan (see the widget/lx.ml comment a
// few lines up in modules_compiler.go) reads a module file's raw text
// directly and, unlike the real per-file compile - where
// compileCodeInnerDirectives strips comments before processImport ever
// looks at the code - never stripped comments from it at all. Commenting
// out one lx.import(...) argument (e.g. "// lx.socket.WebSocketClient,",
// while testing whether a game module still needed it) was still read as a
// literal bare module name, comment marker included, and logged as
// "Module '// lx.socket.WebSocketClient' does not exist" - even though the
// real compile, which only ever sees the comment-stripped text, correctly
// left it out entirely.
func TestModulesCode_CommentedOutBareImport_IsIgnoredByPreScan(t *testing.T) {
	sysPath := t.TempDir()
	app, err := apptest.New(kernel.Dict{
		"Components": kernel.Dict{
			"JSPreprocessor": kernel.Dict{
				"SysPath":  sysPath,
				"MapsPath": sysPath,
			},
		},
	})
	if err != nil {
		t.Fatalf("apptest.New: %v", err)
	}
	logger := &capturingLogger{}
	app.SetLogger(logger)
	if err := component.SetAppComponent(app, "Components.JSPreprocessor"); err != nil {
		t.Fatalf("SetAppComponent: %v", err)
	}
	pp, err := component.AppComponent(app)
	if err != nil {
		t.Fatalf("AppComponent: %v", err)
	}
	dir := t.TempDir()

	entryPath := filepath.Join(dir, "Entry.js")
	entryCode := "@lx:module lx.cmt.Entry;\n" +
		"lx.import(\n" +
		"\t// lx.cmt.Sibling,\n" +
		"\t'sub/'\n" +
		");\n" +
		"@lx:namespace lx.cmt;\nclass Entry {}\n"
	if err := os.WriteFile(entryPath, []byte(entryCode), 0644); err != nil {
		t.Fatalf("write Entry.js: %v", err)
	}

	subDir := filepath.Join(dir, "sub")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("mkdir sub: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "Other.js"), []byte("class Other {}\n"), 0644); err != nil {
		t.Fatalf("write Other.js: %v", err)
	}

	siblingPath := filepath.Join(dir, "Sibling.js")
	siblingCode := "@lx:module lx.cmt.Sibling;\n@lx:namespace lx.cmt;\nclass Sibling {}\n"
	if err := os.WriteFile(siblingPath, []byte(siblingCode), 0644); err != nil {
		t.Fatalf("write Sibling.js: %v", err)
	}

	mm := pp.ModulesMap()
	if err := mm.Save([]jspp.IJSModuleData{
		mm.NewData("lx.cmt.Entry", entryPath),
		mm.NewData("lx.cmt.Sibling", siblingPath),
	}); err != nil {
		t.Fatalf("Save modules map: %v", err)
	}

	c := pp.CompilerBuilder().
		SetClientContext().
		SetCode("lx.import(lx.cmt.Entry);").
		Compiler()
	code, err := c.Run()
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if strings.Contains(logger.lastError, "lx.cmt.Sibling") {
		t.Fatalf("expected no error about the commented-out import, got: %s", logger.lastError)
	}

	if !strings.Contains(code, "class Entry") || !strings.Contains(code, "class Other") {
		t.Fatalf("expected Entry and Other in the compiled code, got: %s", code)
	}
	if strings.Contains(code, "class Sibling") {
		t.Fatalf("expected the commented-out import to stay ignored (no class Sibling), got: %s", code)
	}
}

// TestCompiledModules_MultipleIncludedFiles_AllContribute is a regression
// test: with BuildModules(false) (compileMainJs's own pattern), each file
// pulled in via a path-based lx.import('some/file.js') runs its own nested
// processImport call on the same *Compiler instance - processImport's
// bare-module branch used to do a plain assignment
// (c.compiledModules = allModuleNames) instead of accumulating, so if two
// different included files each had their own bare lx.import(Name), only
// the LAST one processed survived - the first one's module silently
// vanished from the page's asset manifest and was never fetched at
// runtime. Symptom shape: some imports from included files work, others
// don't, depending on include order.
func TestCompiledModules_MultipleIncludedFiles_AllContribute(t *testing.T) {
	pp := newTestPreprocessor(t)
	clientDir := t.TempDir()

	guiNodeAPath := filepath.Join(clientDir, "GuiNodeA.js")
	if err := os.WriteFile(guiNodeAPath, []byte("lx.import(ModuleA);\nclass GuiNodeA {}\n"), 0644); err != nil {
		t.Fatalf("write GuiNodeA.js: %v", err)
	}
	guiNodeBPath := filepath.Join(clientDir, "GuiNodeB.js")
	if err := os.WriteFile(guiNodeBPath, []byte("lx.import(ModuleB);\nclass GuiNodeB {}\n"), 0644); err != nil {
		t.Fatalf("write GuiNodeB.js: %v", err)
	}

	entryPath := filepath.Join(clientDir, "Plugin.js")
	entryCode := "lx.import('GuiNodeA.js');\nlx.import('GuiNodeB.js');\nclass Plugin extends lx.Plugin {}"
	if err := os.WriteFile(entryPath, []byte(entryCode), 0644); err != nil {
		t.Fatalf("write Plugin.js: %v", err)
	}

	c := pp.CompilerBuilder().
		BuildModules(false).
		SetClientContext().
		SetUnwrapped().
		SetFilePath(entryPath).
		Compiler()
	if _, err := c.Run(); err != nil {
		t.Fatalf("compile: %v", err)
	}

	found := map[string]bool{}
	for _, m := range c.CompiledModules() {
		found[m] = true
	}
	if !found["ModuleA"] || !found["ModuleB"] {
		t.Fatalf("expected both ModuleA and ModuleB in CompiledModules(), got %v", c.CompiledModules())
	}
}
