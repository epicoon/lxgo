package compiler_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/epicoon/lxgo/jspp"
	"github.com/epicoon/lxgo/jspp/component"
	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/apptest"
)

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

// TestCompiledModules_MultipleIncludedFiles_AllContribute is a regression
// test for backlog task 0080 ("lx.import from a GuiNode doesn't work"): with
// BuildModules(false) (compileMainJs's own pattern), each file pulled in via
// a path-based lx.import('some/file.js') runs its own nested processImport
// call on the same *Compiler instance - processImport's bare-module branch
// used to do a plain assignment (c.compiledModules = allModuleNames)
// instead of accumulating, so if two different included files (e.g. two
// different GuiNode client files) each had their own bare lx.import(Name),
// only the LAST one processed survived - the first one's module silently
// vanished from the page's asset manifest and was never fetched at runtime.
// This matched the reported symptom shape exactly: some GuiNode-originated
// imports work, others don't, depending on include order - not "GuiNode
// imports never work".
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
