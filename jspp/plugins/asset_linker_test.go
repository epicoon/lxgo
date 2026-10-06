package plugins_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/epicoon/lxgo/jspp"
	"github.com/epicoon/lxgo/jspp/component"
	"github.com/epicoon/lxgo/jspp/plugins"
	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/apptest"
)

// newAssetLinksTestPreprocessor builds a real jspp.IPreprocessor with
// AssetLinksPath.Outer set to outer - unlike newTestPreprocessor
// (plugin_manager_test.go), which leaves it unconfigured (fine for tests
// that never touch asset linking - defineLinks indexes Outer[0]
// unconditionally, so an empty Outer panics).
func newAssetLinksTestPreprocessor(t *testing.T, outer string) jspp.IPreprocessor {
	t.Helper()
	sysPath := t.TempDir()
	app, err := apptest.New(kernel.Dict{
		"Components": kernel.Dict{
			"JSPreprocessor": kernel.Dict{
				"SysPath":  sysPath,
				"MapsPath": sysPath,
				"AssetLinksPath": kernel.Dict{
					"Inner": filepath.Join(sysPath, "assets"),
					"Outer": outer,
				},
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

// newAssetLinksTestPreprocessorWithHost is newAssetLinksTestPreprocessor,
// additionally setting the app's own declared Host/Port (apptest.New
// otherwise defaults to "localhost"/0, which the self-host tests below
// need to override to something they can also put in an image URL).
func newAssetLinksTestPreprocessorWithHost(t *testing.T, outer, host string, port int) jspp.IPreprocessor {
	t.Helper()
	sysPath := t.TempDir()
	app, err := apptest.New(kernel.Dict{
		"Host": host,
		"Port": port,
		"Components": kernel.Dict{
			"JSPreprocessor": kernel.Dict{
				"SysPath":  sysPath,
				"MapsPath": sysPath,
				"AssetLinksPath": kernel.Dict{
					"Inner": filepath.Join(sysPath, "assets"),
					"Outer": outer,
				},
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

// renderImagePaths renders a plugin declaring images (an "images:" block
// in lx-plugin.yaml) with pp, and returns the root plugin's own
// conf.imagePaths - decoded the same way a real consumer (lx-games-lobby)
// would, through JSON, not by reaching into jspp's unexported render
// output types.
func renderImagePaths(t *testing.T, pp jspp.IPreprocessor, imagesYAML string) map[string]any {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "snippets"), 0755); err != nil {
		t.Fatalf("mkdir snippets: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "snippets", "_root.js"), []byte("new lx.Box({geom: true});"), 0644); err != nil {
		t.Fatalf("write snippet: %v", err)
	}
	yamlPath := writeYAML(t, dir, "name: assetPlugin\n"+imagesYAML+"server:\n  rootSnippet: snippets/_root.js\n")

	p := plugins.NewPlugin()
	p.Init(pp)
	p.SetName("assetPlugin")
	p.SetPath(dir)
	cfg := plugins.NewConfig()
	cfg.SetPlugin(p)
	if err := cfg.Load(yamlPath); err != nil {
		t.Fatalf("Load: %v", err)
	}
	p.SetConfig(cfg)

	info, err := pp.PluginManager().Render(p, "en-EN")
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	raw, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("Marshal render info: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("Unmarshal render info: %v", err)
	}

	root := decoded["root"].(string)
	lx := decoded["lx"].(map[string]any)
	rootPlugin := lx[root].(map[string]any)
	conf := rootPlugin["conf"].(map[string]any)
	imagePaths, _ := conf["imagePaths"].(map[string]any)
	return imagePaths
}

// A single image entry already given as an absolute http(s) URL in
// lx-plugin.yaml must be reported unchanged, regardless of Outer - this
// used to be mangled by pf.GetAbsPath (which only recognizes
// filesystem-style absolute paths) before defineLinks' own absolute-URL
// check ever saw it.
func TestAssetLinks_IndividualAbsoluteImage_LeftUnchanged(t *testing.T) {
	pp := newAssetLinksTestPreprocessor(t, "/web")

	imagePaths := renderImagePaths(t, pp, "images:\n  default: https://cdn.example.com/img\n")

	if got := imagePaths["default"]; got != "https://cdn.example.com/img" {
		t.Fatalf("imagePaths[default] = %#v, want the absolute URL unchanged", got)
	}
}

// A local image base with a relative Outer still resolves to a path
// rooted at Outer, as before the fix.
func TestAssetLinks_LocalImageWithRelativeOuter_ResolvesToOuterPath(t *testing.T) {
	pp := newAssetLinksTestPreprocessor(t, "/web")

	imagePaths := renderImagePaths(t, pp, "images:\n  default: assets/images\n")

	got, _ := imagePaths["default"].(string)
	if got == "" || got[0] != '/' {
		t.Fatalf("imagePaths[default] = %#v, want a path rooted at Outer (\"/web/...\")", imagePaths["default"])
	}
	if len(got) < len("/web/") || got[:len("/web/")] != "/web/" {
		t.Fatalf("imagePaths[default] = %q, want it to start with \"/web/\"", got)
	}
}

// A local image base with an ABSOLUTE Outer (the new capability) resolves
// to an absolute URL at that base - not a mangled "scheme:/host" string
// (filepath.Join/path.Join would collapse "://" into ":/").
func TestAssetLinks_LocalImageWithAbsoluteOuter_ResolvesToAbsoluteURL(t *testing.T) {
	pp := newAssetLinksTestPreprocessor(t, "https://cdn.example.com/assets")

	imagePaths := renderImagePaths(t, pp, "images:\n  default: assets/images\n")

	got, _ := imagePaths["default"].(string)
	const wantPrefix = "https://cdn.example.com/assets/"
	if len(got) < len(wantPrefix) || got[:len(wantPrefix)] != wantPrefix {
		t.Fatalf("imagePaths[default] = %q, want it to start with %q (not a mangled \"https:/cdn...\")", got, wantPrefix)
	}
}

// An absolute Outer must not make an already-absolute individual link's
// own URL get re-based under it.
func TestAssetLinks_IndividualAbsoluteImage_LeftUnchangedEvenWithAbsoluteOuter(t *testing.T) {
	pp := newAssetLinksTestPreprocessor(t, "https://cdn.example.com/assets")

	imagePaths := renderImagePaths(t, pp, "images:\n  default: https://other-cdn.example.com/img\n")

	if got := imagePaths["default"]; got != "https://other-cdn.example.com/img" {
		t.Fatalf("imagePaths[default] = %#v, want the absolute URL unchanged", got)
	}
}

// An absolute image URL whose host:port is this same app's own (e.g. a
// dependency sourced via "{@param(Params.StaticSrc)}/..." where StaticSrc
// happens to be this app's own address) must still go through the usual
// local symlinking - it isn't actually published anywhere external, it
// just happens to spell out this app's own address.
func TestAssetLinks_SelfReferencingAbsoluteImage_TreatedAsLocal(t *testing.T) {
	pp := newAssetLinksTestPreprocessorWithHost(t, "/web", "localhost", 8092)

	imagePaths := renderImagePaths(t, pp, "images:\n  default: http://localhost:8092/assets/images\n")

	got, _ := imagePaths["default"].(string)
	if len(got) < len("/web/") || got[:len("/web/")] != "/web/" {
		t.Fatalf("imagePaths[default] = %q, want it resolved through the local symlink path (\"/web/...\"), not left as the literal self-referencing URL", got)
	}
}

// An absolute image URL on a genuinely different host must not be
// mistaken for self just because the port happens to match.
func TestAssetLinks_DifferentHostAbsoluteImage_LeftUnchanged(t *testing.T) {
	pp := newAssetLinksTestPreprocessorWithHost(t, "/web", "localhost", 8092)

	imagePaths := renderImagePaths(t, pp, "images:\n  default: http://example.com:8092/assets/images\n")

	if got := imagePaths["default"]; got != "http://example.com:8092/assets/images" {
		t.Fatalf("imagePaths[default] = %#v, want the absolute URL unchanged (different host)", got)
	}
}

// An absolute image URL on the right host but with NO explicit port must
// not be guessed into matching - there's no reliable default port to
// assume for an app that was never actually told it's 80 or 443.
func TestAssetLinks_SelfHostNoPortInURL_NotTreatedAsSelf(t *testing.T) {
	pp := newAssetLinksTestPreprocessorWithHost(t, "/web", "localhost", 8092)

	imagePaths := renderImagePaths(t, pp, "images:\n  default: http://localhost/assets/images\n")

	if got := imagePaths["default"]; got != "http://localhost/assets/images" {
		t.Fatalf("imagePaths[default] = %#v, want the absolute URL unchanged (no port to compare)", got)
	}
}
