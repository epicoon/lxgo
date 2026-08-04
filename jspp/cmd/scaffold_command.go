package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/epicoon/lxgo/cmd"
	jsppComp "github.com/epicoon/lxgo/jspp/component"
	"github.com/epicoon/lxgo/jspp/internal/utils"
)

// pluginPaths is "path"'s FTypeDetails (see compile_command.go's Config) -
// the app's configured plugin source directories, resolved lazily, only
// once the user is actually being prompted to pick one, and filtered down
// to directories inside the app's own package root - the picklist only
// ever offers choices within the package where the command is invoked, it
// never suggests a configured directory that happens to sit outside it.
// This filter only applies to the offered choices: a directory typed
// directly via --path is trusted as-is, whatever it is (see scaffoldPlugin).
// Deliberately doesn't look at "name" at all (unlike the rest of
// scaffoldPlugin) - ParamsConfig is a map, so ParamTypeEnum options for
// "path" could get resolved before "name" has been validated/prompted for;
// listing plugin source directories doesn't need a plugin name anyway.
func pluginPaths(com cmd.ICommand) (any, error) {
	c, ok := com.(*CompileCommand)
	if !ok {
		return nil, errors.New("scaffold-plugin: unexpected command type")
	}
	app := c.app
	if app == nil {
		return nil, errors.New("command require access to application through 'app' option")
	}

	pp, _ := jsppComp.AppComponent(app)
	srcList, err := utils.GetPluginsSrcList(pp)
	if err != nil {
		return nil, err
	}

	root := app.Pathfinder().GetRoot()
	inPackage := make([]string, 0, len(srcList))
	for _, dir := range srcList {
		if isWithinRoot(root, dir) {
			inPackage = append(inPackage, dir)
		}
	}
	if len(inPackage) == 0 {
		return nil, errors.New("no configured plugin source directories inside this package (Components.JSPreprocessor.Plugins)")
	}

	return inPackage, nil
}

// isWithinRoot reports whether dir is root itself or nested under it.
func isWithinRoot(root, dir string) bool {
	rel, err := filepath.Rel(root, dir)
	if err != nil {
		return false
	}
	return rel == "." || (!filepath.IsAbs(rel) && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

// scaffoldPlugin is the "scaffold-plugin" action's cmd.FAction - creates a
// plugin skeleton at "path" (see doc/plugins.md, "Anatomy of a plugin"):
// the genuinely minimal one by default, or, with --full, a more complete
// starting point (assets/i18n, assets/css, one GUI node) via
// scaffoldPluginFilesFull. "path" is only restricted to a configured
// plugin directory when it comes from the interactive picklist (see
// pluginPaths) - typed directly via --path, any existing target directory
// is accepted as-is.
//
// Deliberately out of scope: a scaffold for a plain (non-plugin) module -
// a module is just one file with an @lx:module header, so there's much
// less boilerplate to save by generating one; left for a separate task if
// it turns out to be wanted.
func scaffoldPlugin(com cmd.ICommand) error {
	name, _ := com.Param("name").(string)
	if err := validatePluginName(name); err != nil {
		return err
	}

	dir, _ := com.Param("path").(string)
	if dir == "" {
		return errors.New("parameter 'path' required")
	}

	pluginDir := filepath.Join(dir, name)
	if _, err := os.Stat(pluginDir); err == nil {
		return fmt.Errorf("'%s' already exists", pluginDir)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("can not check '%s': %w", pluginDir, err)
	}

	scaffold := scaffoldPluginFiles
	if com.Flag("full") {
		scaffold = scaffoldPluginFilesFull
	}
	if err := scaffold(pluginDir, name); err != nil {
		return err
	}

	fmt.Printf("Created plugin '%s' at %s\n", name, pluginDir)
	return nil
}

// validatePluginName rejects an empty name or one that isn't a plain
// directory-name-safe string - name ends up in filepath.Join(dir, name),
// so a path separator or ".." could otherwise write outside dir.
func validatePluginName(name string) error {
	if name == "" {
		return errors.New("parameter 'name' is required")
	}
	if strings.ContainsAny(name, `/\`) || strings.Contains(name, "..") {
		return errors.New("parameter 'name' must be a plain name, not a path")
	}
	return nil
}

// pluginNamespace derives a valid JS identifier from name, for the
// @lx:namespace directives and the qualified MainCss/Main references in
// scaffoldPluginFilesFull's lx-plugin.yaml (cssAssets, client.guiNodes) -
// unlike the plugin's own "name" (free-form beyond validatePluginName's
// path-safety check), a namespace has to be an actual JS identifier. An
// already-valid identifier is returned unchanged (a plain "MyPlugin"
// isn't reshaped); otherwise every run of characters that can't appear in
// a JS identifier becomes a word boundary and the pieces are camelCased
// back together ("my-plugin" -> "myPlugin"), with a leading digit (an
// identifier can't start with one) getting an underscore prefix.
func pluginNamespace(name string) string {
	if isValidJSIdent(name) {
		return name
	}

	var b strings.Builder
	newWord := false
	for _, r := range name {
		if r == '_' || r == '$' || unicode.IsLetter(r) || unicode.IsDigit(r) {
			if newWord && b.Len() > 0 {
				r = unicode.ToUpper(r)
			}
			b.WriteRune(r)
			newWord = false
		} else {
			newWord = true
		}
	}

	ns := b.String()
	if ns == "" {
		return "Plugin"
	}
	if unicode.IsDigit([]rune(ns)[0]) {
		ns = "_" + ns
	}
	return ns
}

// isValidJSIdent reports whether s is already a valid JS identifier as-is
// (letters/digits/_/$ , not starting with a digit).
func isValidJSIdent(s string) bool {
	r := []rune(s)
	if len(r) == 0 {
		return false
	}
	if r[0] != '_' && r[0] != '$' && !unicode.IsLetter(r[0]) {
		return false
	}
	for _, c := range r[1:] {
		if c != '_' && c != '$' && !unicode.IsLetter(c) && !unicode.IsDigit(c) {
			return false
		}
	}
	return true
}

// scaffoldPluginFiles creates dir and writes a minimal plugin skeleton into
// it: lx-plugin.yaml (just the required "name"), snippets/_root.js (the
// root snippet's default location), and Plugin.js (a bare lx.Plugin
// subclass) - see doc/plugins.md's "genuinely minimal plugin".
func scaffoldPluginFiles(dir, name string) error {
	if err := os.MkdirAll(filepath.Join(dir, "snippets"), 0o755); err != nil {
		return fmt.Errorf("can not create plugin directory: %w", err)
	}

	ns := pluginNamespace(name)

	files := map[string]string{
		"lx-plugin.yaml":    "name: " + name + "\n",
		"snippets/_root.js": "/**\n * @var {lx.Plugin}  $plugin\n * @var {lx.Snippet} $snippet\n */\n",
		"Plugin.js": "// @lx:namespace " + ns + ";\n" +
			"class Plugin extends lx.Plugin {\n" +
			"    run() {\n" +
			"        // widgets, this.core and this.guiNodes are all ready here\n" +
			"    }\n" +
			"}\n",
	}

	for rel, content := range files {
		path := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return fmt.Errorf("can not write %s: %w", rel, err)
		}
	}

	return nil
}

// scaffoldPluginFilesFull creates dir and writes a more complete starting
// point than scaffoldPluginFiles - a typical shape from doc/plugins.md's
// "Anatomy of a plugin": assets/i18n (one string), assets/css (one
// lx.PluginCssAsset), one GUI node ("Main"), and a root snippet with one
// element wired up to that GUI node and the i18n string.
func scaffoldPluginFilesFull(dir, name string) error {
	dirs := []string{"snippets", "assets/css", "assets/i18n", "client/guiNodes"}
	for _, rel := range dirs {
		if err := os.MkdirAll(filepath.Join(dir, filepath.FromSlash(rel)), 0o755); err != nil {
			return fmt.Errorf("can not create plugin directory: %w", err)
		}
	}

	ns := pluginNamespace(name)
	cssClass := strings.ToLower(name) + "-main"

	files := map[string]string{
		"lx-plugin.yaml": "name: " + name + "\n" +
			"\n" +
			"i18n: assets/i18n/tr.yaml\n" +
			"\n" +
			"require:\n" +
			"  - assets/css/\n" +
			"cssAssets:\n" +
			"  - " + ns + ".MainCss\n" +
			"\n" +
			"client:\n" +
			"  file: Plugin.js\n" +
			"  require:\n" +
			"    - client/guiNodes/\n" +
			"  guiNodes:\n" +
			"    main: " + ns + ".Main\n",
		"Plugin.js": "// @lx:namespace " + ns + ";\n" +
			"class Plugin extends lx.Plugin {\n" +
			"    run() {\n" +
			"        // widgets, this.core and this.guiNodes are all ready here\n" +
			"    }\n" +
			"}\n",
		"snippets/_root.js": "/**\n" +
			" * @var {lx.Plugin}  $plugin\n" +
			" * @var {lx.Snippet} $snippet\n" +
			" */\n" +
			"\n" +
			"lx.ml(`\n" +
			"    <lx.Box> @main." + cssClass + " [_] \"${lx.i18n('hi')}\"\n" +
			"`);\n",
		"assets/css/MainCss.js": "// @lx:namespace " + ns + ";\n" +
			"class MainCss extends lx.PluginCssAsset {\n" +
			"    /**\n" +
			"     * @param {lx.CssContext} css\n" +
			"     */\n" +
			"    init(css) {\n" +
			"        css.addClass('" + cssClass + "', {\n" +
			"            // styles for the plugin's root element\n" +
			"        });\n" +
			"    }\n" +
			"}\n",
		"assets/i18n/tr.yaml": "en-EN:\n" +
			"  hi: Hi\n" +
			"\n" +
			"ru-RU:\n" +
			"  hi: Привет\n",
		"client/guiNodes/Main.js": "// @lx:namespace " + ns + ";\n" +
			"class Main extends lx.GuiNode {\n" +
			"    init() {\n" +
			"        // this.getWidget() is ready here\n" +
			"    }\n" +
			"\n" +
			"    initHandlers() {\n" +
			"        // for client events handling\n" +
			"    }\n" +
			"\n" +
			"    subscribeEvents() {\n" +
			"        // for plugin events handling\n" +
			"    }\n" +
			"}\n",
	}

	for rel, content := range files {
		path := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return fmt.Errorf("can not write %s: %w", rel, err)
		}
	}

	return nil
}
