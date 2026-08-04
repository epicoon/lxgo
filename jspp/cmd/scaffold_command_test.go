package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/epicoon/lxgo/cmd"
	jsppComp "github.com/epicoon/lxgo/jspp/component"
	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/apptest"
)

// newTestCompileCommand builds a *CompileCommand wired to a real, minimal
// kernel.IApp with the JSPreprocessor component configured with the given
// plugin source directories - for testing pluginPaths, which needs a real
// app.Pathfinder() and jspp component behind it.
func newTestCompileCommand(t *testing.T, plugins []string) *CompileCommand {
	t.Helper()
	app, err := apptest.New(kernel.Dict{
		"Components": kernel.Dict{
			"JSPreprocessor": kernel.Dict{
				"Plugins": plugins,
			},
		},
	})
	if err != nil {
		t.Fatalf("apptest.New: %v", err)
	}
	if err := jsppComp.SetAppComponent(app, "Components.JSPreprocessor"); err != nil {
		t.Fatalf("SetAppComponent: %v", err)
	}
	return &CompileCommand{Command: cmd.NewCommand(), app: app}
}

func TestIsWithinRoot(t *testing.T) {
	root := string(filepath.Separator) + filepath.Join("some", "app", "root")
	cases := []struct {
		name string
		dir  string
		want bool
	}{
		{"root_itself", root, true},
		{"direct_subdir", filepath.Join(root, "plugins"), true},
		{"nested_subdir", filepath.Join(root, "frontend", "plugins"), true},
		{"sibling_dir", filepath.Join(filepath.Dir(root), "elsewhere"), false},
		{"parent_of_root", filepath.Dir(root), false},
		{"unrelated_absolute_path", string(filepath.Separator) + filepath.Join("etc", "passwd"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isWithinRoot(root, tc.dir); got != tc.want {
				t.Fatalf("isWithinRoot(%q, %q) = %v, want %v", root, tc.dir, got, tc.want)
			}
		})
	}
}

// TestPluginPaths_FiltersOutDirectoriesOutsideRoot is the actual
// design point: the picklist offered for "path" never suggests a
// configured directory that sits outside the package (app root) where the
// command is invoked, even if such an entry exists in
// Components.JSPreprocessor.Plugins.
func TestPluginPaths_FiltersOutDirectoriesOutsideRoot(t *testing.T) {
	outside := t.TempDir()
	c := newTestCompileCommand(t, []string{"frontend/plugins", outside})

	root := c.app.Pathfinder().GetRoot()
	inside := c.app.Pathfinder().GetAbsPath("frontend/plugins")

	got, err := pluginPaths(c)
	if err != nil {
		t.Fatalf("pluginPaths: %v", err)
	}
	dirs, _ := got.([]string)

	found := false
	for _, d := range dirs {
		if d == outside {
			t.Fatalf("pluginPaths must not offer %q, which is outside the package root %q; got %v", outside, root, dirs)
		}
		if d == inside {
			found = true
		}
	}
	if !found {
		t.Fatalf("pluginPaths should still offer %q (inside the package root); got %v", inside, dirs)
	}
}

// TestScaffoldPlugin_TrustsManualPathOutsideConfiguredPlugins checks that
// a "path" typed directly (as opposed to picked from pluginPaths'
// filtered list) is used as-is: the package-root restriction only applies
// to what's offered in the interactive picklist, not to a manually
// supplied --path.
func TestScaffoldPlugin_TrustsManualPathOutsideConfiguredPlugins(t *testing.T) {
	elsewhere := t.TempDir()
	c := newTestCompileCommand(t, nil)
	c.SetParams(map[string]any{"name": "MyPlugin", "path": elsewhere})

	if err := scaffoldPlugin(c); err != nil {
		t.Fatalf("scaffoldPlugin: %v", err)
	}
	if _, err := os.Stat(filepath.Join(elsewhere, "MyPlugin", "lx-plugin.yaml")); err != nil {
		t.Fatalf("expected the plugin to be scaffolded at the manually given path: %v", err)
	}
}

// TestScaffoldPlugin_FullFlag_CreatesFullSkeleton checks scaffoldPlugin
// dispatches to scaffoldPluginFilesFull when --full is passed, instead of
// the minimal skeleton.
func TestScaffoldPlugin_FullFlag_CreatesFullSkeleton(t *testing.T) {
	parent := t.TempDir()
	c := newTestCompileCommand(t, nil)
	c.SetParams(map[string]any{"name": "MyPlugin", "path": parent, "full": true})

	if err := scaffoldPlugin(c); err != nil {
		t.Fatalf("scaffoldPlugin: %v", err)
	}

	pluginDir := filepath.Join(parent, "MyPlugin")
	for _, rel := range []string{
		"assets/css/MainCss.js",
		"assets/i18n/tr.yaml",
		"client/guiNodes/Main.js",
	} {
		if _, err := os.Stat(filepath.Join(pluginDir, filepath.FromSlash(rel))); err != nil {
			t.Fatalf("expected %s under a --full scaffold: %v", rel, err)
		}
	}
}

func TestValidatePluginName(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"empty", "", true},
		{"valid", "MyPlugin", false},
		{"forward_slash_rejected", "a/b", true},
		{"back_slash_rejected", `a\b`, true},
		{"dotdot_rejected", "..", true},
		{"dotdot_inside_rejected", "my..plugin", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validatePluginName(tc.input)
			if tc.wantErr && err == nil {
				t.Fatalf("validatePluginName(%q): expected an error, got nil", tc.input)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("validatePluginName(%q): unexpected error: %v", tc.input, err)
			}
		})
	}
}

// TestPluginNamespace checks name -> JS-identifier conversion: an
// already-valid identifier passes through untouched, and anything else
// gets camelCased at the invalid-character boundaries (a plain
// validatePluginName pass allows things like dashes or spaces in "name"
// that can't appear as-is in a "@lx:namespace X;" directive or an
// "X.MainCss" reference).
func TestPluginNamespace(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"already_valid_unchanged", "MyPlugin", "MyPlugin"},
		{"underscore_and_digits_unchanged", "my_plugin2", "my_plugin2"},
		{"dash_camelCased", "my-plugin", "myPlugin"},
		{"space_camelCased", "My Plugin", "MyPlugin"},
		{"multiple_separators", "my--cool  plugin", "myCoolPlugin"},
		{"leading_digit_gets_prefixed", "3d-plugin", "_3dPlugin"},
		{"all_digits", "123", "_123"},
		{"only_separators", "---", "Plugin"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := pluginNamespace(tc.input); got != tc.want {
				t.Fatalf("pluginNamespace(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// TestScaffoldPluginFilesFull_NormalizesNamespaceForInvalidName checks the
// end-to-end effect through scaffoldPluginFilesFull itself: a plugin name
// that isn't a valid JS identifier (dashes) still produces a working
// @lx:namespace and cssAssets/guiNodes references, using the same
// converted namespace everywhere.
func TestScaffoldPluginFilesFull_NormalizesNamespaceForInvalidName(t *testing.T) {
	parent := t.TempDir()
	pluginDir := filepath.Join(parent, "my-plugin")

	if err := scaffoldPluginFilesFull(pluginDir, "my-plugin"); err != nil {
		t.Fatalf("scaffoldPluginFilesFull: %v", err)
	}

	yaml, err := os.ReadFile(filepath.Join(pluginDir, "lx-plugin.yaml"))
	if err != nil {
		t.Fatalf("lx-plugin.yaml: %v", err)
	}
	for _, want := range []string{"name: my-plugin", "- myPlugin.MainCss", "main: myPlugin.Main"} {
		if !strings.Contains(string(yaml), want) {
			t.Fatalf("lx-plugin.yaml missing %q, got:\n%s", want, yaml)
		}
	}

	pluginJS, err := os.ReadFile(filepath.Join(pluginDir, "Plugin.js"))
	if err != nil {
		t.Fatalf("Plugin.js: %v", err)
	}
	if !strings.Contains(string(pluginJS), "// @lx:namespace myPlugin;") {
		t.Fatalf("Plugin.js missing the converted namespace directive, got: %s", pluginJS)
	}
}

// TestScaffoldPluginFiles_CreatesMinimalSkeleton checks the actual files
// created match doc/plugins.md's "genuinely minimal plugin" plus Plugin.js:
// lx-plugin.yaml (just "name"), snippets/_root.js, and a bare lx.Plugin
// subclass.
func TestScaffoldPluginFiles_CreatesMinimalSkeleton(t *testing.T) {
	parent := t.TempDir()
	pluginDir := filepath.Join(parent, "MyPlugin")

	if err := scaffoldPluginFiles(pluginDir, "MyPlugin"); err != nil {
		t.Fatalf("scaffoldPluginFiles: %v", err)
	}

	yaml, err := os.ReadFile(filepath.Join(pluginDir, "lx-plugin.yaml"))
	if err != nil {
		t.Fatalf("lx-plugin.yaml: %v", err)
	}
	if string(yaml) != "name: MyPlugin\n" {
		t.Fatalf("lx-plugin.yaml = %q, want %q", yaml, "name: MyPlugin\n")
	}

	if _, err := os.Stat(filepath.Join(pluginDir, "snippets", "_root.js")); err != nil {
		t.Fatalf("snippets/_root.js: %v", err)
	}

	pluginJS, err := os.ReadFile(filepath.Join(pluginDir, "Plugin.js"))
	if err != nil {
		t.Fatalf("Plugin.js: %v", err)
	}
	if !strings.Contains(string(pluginJS), "@lx:namespace MyPlugin;") {
		t.Fatalf("Plugin.js missing the namespace directive, got: %s", pluginJS)
	}
	if !strings.Contains(string(pluginJS), "class Plugin extends lx.Plugin") {
		t.Fatalf("Plugin.js missing the lx.Plugin subclass, got: %s", pluginJS)
	}
}

// TestScaffoldPluginFiles_RefusesToOverwriteIsNotItsJob checks
// scaffoldPluginFiles itself has no existence check (that's scaffoldPlugin's
// job, before calling it) - this just pins that MkdirAll on an
// already-existing directory doesn't error, so the caller-side check is
// the only guard against clobbering an existing plugin.
func TestScaffoldPluginFiles_RefusesToOverwriteIsNotItsJob(t *testing.T) {
	parent := t.TempDir()
	pluginDir := filepath.Join(parent, "MyPlugin")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatalf("setup MkdirAll: %v", err)
	}

	if err := scaffoldPluginFiles(pluginDir, "MyPlugin"); err != nil {
		t.Fatalf("scaffoldPluginFiles on a pre-existing directory: %v", err)
	}
}

// TestScaffoldPluginFilesFull_CreatesFullSkeleton checks the fuller
// skeleton --full asks for: assets/i18n and assets/css alongside a GUI
// node, wired together (the root snippet's element uses the GUI node's
// key and the css class MainCss declares; the config lists cssAssets and
// client.guiNodes).
func TestScaffoldPluginFilesFull_CreatesFullSkeleton(t *testing.T) {
	parent := t.TempDir()
	pluginDir := filepath.Join(parent, "MyPlugin")

	if err := scaffoldPluginFilesFull(pluginDir, "MyPlugin"); err != nil {
		t.Fatalf("scaffoldPluginFilesFull: %v", err)
	}

	yaml, err := os.ReadFile(filepath.Join(pluginDir, "lx-plugin.yaml"))
	if err != nil {
		t.Fatalf("lx-plugin.yaml: %v", err)
	}
	for _, want := range []string{"name: MyPlugin", "i18n: assets/i18n/tr.yaml", "cssAssets:", "- MyPlugin.MainCss", "guiNodes:", "main: MyPlugin.Main"} {
		if !strings.Contains(string(yaml), want) {
			t.Fatalf("lx-plugin.yaml missing %q, got:\n%s", want, yaml)
		}
	}

	i18n, err := os.ReadFile(filepath.Join(pluginDir, "assets", "i18n", "tr.yaml"))
	if err != nil {
		t.Fatalf("assets/i18n/tr.yaml: %v", err)
	}
	if !strings.Contains(string(i18n), "hi: Hi") {
		t.Fatalf("assets/i18n/tr.yaml missing the 'hi' key, got: %s", i18n)
	}

	css, err := os.ReadFile(filepath.Join(pluginDir, "assets", "css", "MainCss.js"))
	if err != nil {
		t.Fatalf("assets/css/MainCss.js: %v", err)
	}
	if !strings.Contains(string(css), "// @lx:namespace MyPlugin;") {
		t.Fatalf("assets/css/MainCss.js missing the namespace directive, got: %s", css)
	}
	if !strings.Contains(string(css), "class MainCss extends lx.PluginCssAsset") {
		t.Fatalf("assets/css/MainCss.js missing the PluginCssAsset subclass, got: %s", css)
	}
	if !strings.Contains(string(css), "myplugin-main") {
		t.Fatalf("assets/css/MainCss.js should declare the css class the snippet uses, got: %s", css)
	}

	guiNode, err := os.ReadFile(filepath.Join(pluginDir, "client", "guiNodes", "Main.js"))
	if err != nil {
		t.Fatalf("client/guiNodes/Main.js: %v", err)
	}
	if !strings.Contains(string(guiNode), "// @lx:namespace MyPlugin;") {
		t.Fatalf("client/guiNodes/Main.js missing the namespace directive, got: %s", guiNode)
	}
	if !strings.Contains(string(guiNode), "class Main extends lx.GuiNode") {
		t.Fatalf("client/guiNodes/Main.js missing the GuiNode subclass, got: %s", guiNode)
	}

	// The root snippet is LXML (lx.ml`...`), not plain JS - it should
	// register the element under the "main" GUI node key and use exactly
	// the css class MainCss.js declares.
	snippet, err := os.ReadFile(filepath.Join(pluginDir, "snippets", "_root.js"))
	if err != nil {
		t.Fatalf("snippets/_root.js: %v", err)
	}
	for _, want := range []string{"@main.myplugin-main", "lx.i18n('hi')"} {
		if !strings.Contains(string(snippet), want) {
			t.Fatalf("snippets/_root.js missing %q, got:\n%s", want, snippet)
		}
	}
}
