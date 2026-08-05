package compiler

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// captureStderr redirects os.Stderr for the duration of fn and returns
// whatever was written to it.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	orig := os.Stderr
	os.Stderr = w
	defer func() { os.Stderr = orig }()

	fn()

	w.Close()
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read captured stderr: %v", err)
	}
	return string(out)
}

// A Compiler with no bound preprocessor (standalone mode, e.g. built via
// Builder() without SetPreprocessor) must not panic on any of the
// pp-dependent paths below - it either has a plain fallback (Pathfinder,
// logError) or reports a clear error instead of the feature that genuinely
// requires a live component (lx.module(...), lx.ml(...)).

func TestCompiler_Pathfinder_StandaloneMode_NoPathfinderSet(t *testing.T) {
	c := &Compiler{}
	if pf := c.Pathfinder(); pf != nil {
		t.Fatalf("expected nil Pathfinder, got %v", pf)
	}
}

func TestCompiler_LogError_StandaloneMode_WritesToStderr(t *testing.T) {
	c := &Compiler{}
	out := captureStderr(t, func() {
		c.logError("something went wrong: %s", "reason")
	})
	if !strings.Contains(out, "something went wrong: reason") {
		t.Fatalf("expected the formatted message on stderr, got %q", out)
	}
}

func TestCompiler_ResolvePath_EmptyPathDoesNotPanic(t *testing.T) {
	c := &Compiler{}
	if got := c.resolvePath("current.js", ""); got != "" {
		t.Fatalf("expected empty result for empty path, got %q", got)
	}
}

func TestCompiler_ResolvePath_RelativeToCurrentFile(t *testing.T) {
	// A plain relative path resolves next to the file that references it,
	// not against the pathfinder's root - this used to be a bug (relative
	// paths in @lx:js/lx.json/lx.yaml always resolved from the app root,
	// so a file couldn't pull in files sitting right next to it).
	c := &Compiler{}
	got := c.resolvePath("/app/widgets/box/Box.js", "styles.css")
	want := "/app/widgets/box/styles.css"
	if got != want {
		t.Fatalf("resolvePath() = %q, want %q", got, want)
	}
}

func TestCompiler_CheckModule_StandaloneMode_ReportsErrorNotPanic(t *testing.T) {
	c := &Compiler{}
	var modulesForBuild, filePaths []string

	out := captureStderr(t, func() {
		c.checkModule("SomeModule", &modulesForBuild, &filePaths)
	})

	if !strings.Contains(out, "SomeModule") {
		t.Fatalf("expected an error mentioning the module name, got %q", out)
	}
	if len(modulesForBuild) != 0 || len(filePaths) != 0 {
		t.Fatalf("expected nothing resolved, got modules=%v files=%v", modulesForBuild, filePaths)
	}
}

func TestCompiler_ProcessLxml_StandaloneMode_ReportsErrorNotPanic(t *testing.T) {
	c := &Compiler{}
	src := "before lx.ml(`<lx.Box> #text('hi')`) after"

	var got string
	out := captureStderr(t, func() {
		got = c.processLxml(src)
	})

	if !strings.Contains(out, "standalone mode") {
		t.Fatalf("expected a standalone-mode error on stderr, got %q", out)
	}
	if got != src {
		t.Fatalf("expected the lx.ml(...) call left untouched, got %q", got)
	}
}

// TestBuilder_StandaloneCompile compiles a real file straight off disk
// through the public Builder() - no kernel.IApp, no preprocessor component
// at all. This is the actual use case standalone mode exists for: an
// offline build script (see lxgo-auth/cmd/assets.go) that just wants a
// compiled bundle, without spinning up a throwaway app and registering a
// component purely to reach CompilerBuilder().
func TestBuilder_StandaloneCompile(t *testing.T) {
	dir := t.TempDir()
	entry := filepath.Join(dir, "App.js")
	if err := os.WriteFile(entry, []byte("class App { hi() { return 'hi'; } }\n"), 0644); err != nil {
		t.Fatalf("write entry file: %v", err)
	}

	code, err := Builder().
		SetClientContext().
		SetUnwrapped().
		SetFilePath(entry).
		Compiler().
		Run()
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(code, "class App") {
		t.Fatalf("expected the compiled source in the output, got: %s", code)
	}
}
