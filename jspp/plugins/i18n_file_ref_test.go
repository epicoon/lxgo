package plugins_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/epicoon/lxgo/jspp/plugins"
)

// setupI18nFileRefPlugin writes a plugin directory with an i18n YAML file
// referencing (via ${^path}) the given extra files, and returns the loaded
// plugin ready for I18n().
func setupI18nFileRefPlugin(t *testing.T, i18nYAML string, extraFiles map[string]string) *plugins.Plugin {
	t.Helper()
	dir := t.TempDir()

	if err := os.MkdirAll(filepath.Join(dir, "i18n"), 0755); err != nil {
		t.Fatalf("mkdir i18n: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "i18n", "main.yaml"), []byte(i18nYAML), 0644); err != nil {
		t.Fatalf("write i18n/main.yaml: %v", err)
	}
	for name, content := range extraFiles {
		if err := os.WriteFile(filepath.Join(dir, "i18n", name), []byte(content), 0644); err != nil {
			t.Fatalf("write i18n/%s: %v", name, err)
		}
	}

	yamlPath := writeYAML(t, dir, "name: myPlugin\ni18n: i18n/main.yaml\n")

	p := newTestPlugin(t, dir)
	cfg := plugins.NewConfig()
	cfg.SetPlugin(p)
	if err := cfg.Load(yamlPath); err != nil {
		t.Fatalf("Load: %v", err)
	}
	p.SetConfig(cfg)
	return p
}

func TestPlugin_I18n_PlainValueUnaffected(t *testing.T) {
	p := setupI18nFileRefPlugin(t, "en-EN:\n  greeting: Hello\n", nil)

	if got := p.I18n().Get("en-EN", "greeting"); got != "Hello" {
		t.Fatalf("Get(greeting) = %q, want Hello", got)
	}
}

func TestPlugin_I18n_FileRef_PlainText(t *testing.T) {
	p := setupI18nFileRefPlugin(t,
		"en-EN:\n  notes: \"${^notes.txt}\"\n",
		map[string]string{"notes.txt": "plain text content"},
	)

	if got := p.I18n().Get("en-EN", "notes"); got != "plain text content" {
		t.Fatalf("Get(notes) = %q, want the referenced file's raw content", got)
	}
}

func TestPlugin_I18n_FileRef_MarkdownRendered(t *testing.T) {
	p := setupI18nFileRefPlugin(t,
		"en-EN:\n  rules: \"${^rules.md}\"\n",
		map[string]string{"rules.md": "# Title\n\nSome text."},
	)

	got := p.I18n().Get("en-EN", "rules")
	if got == "# Title\n\nSome text." {
		t.Fatalf("Get(rules) returned the raw markdown, want it rendered to HTML")
	}
	if got == "" {
		t.Fatalf("Get(rules) is empty, want the rendered markdown")
	}
}

// TestPlugin_I18n_FileRef_MissingFile is a regression-shaped test: a
// missing referenced file must not fail the build (matching lx.md(...)'s
// own behavior) - the key resolves to an empty string instead.
func TestPlugin_I18n_FileRef_MissingFile(t *testing.T) {
	p := setupI18nFileRefPlugin(t, "en-EN:\n  rules: \"${^nope.md}\"\n", nil)

	if got := p.I18n().Get("en-EN", "rules"); got != "" {
		t.Fatalf("Get(rules) = %q, want empty string for a missing referenced file", got)
	}
}

// TestPlugin_I18n_FileRef_SurroundingTextIsNotAFileRef is a regression-
// shaped test: only a value that is ENTIRELY "${^path}" is a file
// reference - the pattern anchors on both ends specifically so ordinary
// text around it (or a real ${param} placeholder elsewhere in the same
// value) is left untouched as literal translation text, not misread as a
// reference.
func TestPlugin_I18n_FileRef_SurroundingTextIsNotAFileRef(t *testing.T) {
	p := setupI18nFileRefPlugin(t,
		"en-EN:\n  greeting: \"see ${^notes.txt} for details, ${name}\"\n",
		map[string]string{"notes.txt": "should not be read"},
	)

	want := "see ${^notes.txt} for details, ${name}"
	if got := p.I18n().Get("en-EN", "greeting"); got != want {
		t.Fatalf("Get(greeting) = %q, want %q (unchanged)", got, want)
	}
}

// TestPlugin_I18n_FileRef_PerLanguage covers the actual motivating use case:
// each language section points at its own file, resolved independently.
func TestPlugin_I18n_FileRef_PerLanguage(t *testing.T) {
	p := setupI18nFileRefPlugin(t,
		"ru-RU:\n  rules: \"${^rules.ru.txt}\"\nen-EN:\n  rules: \"${^rules.en.txt}\"\n",
		map[string]string{
			"rules.ru.txt": "русские правила",
			"rules.en.txt": "english rules",
		},
	)

	if got := p.I18n().Get("ru-RU", "rules"); got != "русские правила" {
		t.Fatalf("Get(ru-RU, rules) = %q", got)
	}
	if got := p.I18n().Get("en-EN", "rules"); got != "english rules" {
		t.Fatalf("Get(en-EN, rules) = %q", got)
	}
}
