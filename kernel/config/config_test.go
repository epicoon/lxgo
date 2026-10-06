package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/epicoon/lxgo/kernel"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestLoad_BasicYAML(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	writeFile(t, cfgPath, "Port: 8080\nName: myapp\n")

	conf, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	port, err := GetParam[int](conf, "Port")
	if err != nil || port != 8080 {
		t.Fatalf("expected Port=8080, got %v (err=%v)", port, err)
	}
	name, err := GetParam[string](conf, "Name")
	if err != nil || name != "myapp" {
		t.Fatalf("expected Name=myapp, got %v (err=%v)", name, err)
	}
}

func TestLoad_MergesImportedConfigRecursively(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	writeFile(t, cfgPath, ""+
		"Import: [config-local.yaml]\n"+
		"Name: main\n"+
		"Database:\n"+
		"  Host: prod-host\n"+
		"  Port: 5432\n")
	writeFile(t, filepath.Join(dir, "config-local.yaml"), ""+
		"Database:\n"+
		"  Host: localhost\n")

	conf, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	db, err := GetParam[kernel.Dict](conf, "Database")
	if err != nil {
		t.Fatalf("GetParam Database: %v", err)
	}
	// The imported config's Database.Host overrides the main one...
	if db["Host"] != "localhost" {
		t.Fatalf("expected imported override Database.Host=localhost, got %v", db["Host"])
	}
	// ...but merging is recursive, not a wholesale replace: Database.Port
	// wasn't mentioned in the imported config, so it must survive untouched.
	if v, ok := db["Port"]; !ok || v != 5432 {
		t.Fatalf("expected untouched Database.Port=5432, got %v (ok=%v)", v, ok)
	}
	// A top-level key the imported config doesn't touch at all must survive too.
	name, err := GetParam[string](conf, "Name")
	if err != nil || name != "main" {
		t.Fatalf("expected untouched Name=main, got %v (err=%v)", name, err)
	}
}

func TestLoad_ImportOrder_LastEntryWins(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	writeFile(t, cfgPath, "Import: [a.yaml, b.yaml]\nName: base\n")
	writeFile(t, filepath.Join(dir, "a.yaml"), "Name: from-a\n")
	writeFile(t, filepath.Join(dir, "b.yaml"), "Name: from-b\n")

	conf, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	name, err := GetParam[string](conf, "Name")
	if err != nil || name != "from-b" {
		t.Fatalf("expected the later Import entry (b.yaml) to win, got %v (err=%v)", name, err)
	}
}

func TestLoad_ImportCascadesAndIsRelativeToImportingFile(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "sub")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	cfgPath := filepath.Join(root, "config.yaml")
	writeFile(t, cfgPath, "Import: [sub/mid.yaml]\nName: base\n")
	// mid.yaml's own Import is relative to sub/, not root/ - "leaf.yaml"
	// here must resolve to sub/leaf.yaml, not root/leaf.yaml.
	writeFile(t, filepath.Join(sub, "mid.yaml"), "Import: [leaf.yaml]\n")
	writeFile(t, filepath.Join(sub, "leaf.yaml"), "Name: from-leaf\n")

	conf, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	name, err := GetParam[string](conf, "Name")
	if err != nil || name != "from-leaf" {
		t.Fatalf("expected the cascaded leaf import to win, got %v (err=%v)", name, err)
	}
}

func TestLoad_ImportNotAList_Errors(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	writeFile(t, cfgPath, "Import: not-a-list\n")

	if _, err := Load(cfgPath); err == nil {
		t.Fatal("expected an error for a non-list \"Import\", got nil")
	}
}

func TestLoad_ImportEntryNotAString_Errors(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	writeFile(t, cfgPath, "Import: [42]\n")

	if _, err := Load(cfgPath); err == nil {
		t.Fatal("expected an error for a non-string Import entry, got nil")
	}
}

func TestLoad_EnvRequiredWhenExplicit(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	writeFile(t, cfgPath, "Env: custom.env\n")
	// custom.env is intentionally not created.

	if _, err := Load(cfgPath); err == nil {
		t.Fatalf("expected an error when the explicitly-named Env file is missing")
	}
}

func TestLoad_EnvNotRequiredWhenAbsent(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	writeFile(t, cfgPath, "Port: 8080\n")
	// No Env key, and no .env file either - Load must not fail because of it.

	if _, err := Load(cfgPath); err != nil {
		t.Fatalf("Load should succeed without an Env key/file, got: %v", err)
	}
}

func TestApplyEnv_SubstitutesFromEnvFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".env"), "GREETING=hello\n")

	conf := &kernel.Dict{"Message": "${GREETING}"}
	if err := applyEnv(conf, filepath.Join(dir, ".env"), true); err != nil {
		t.Fatalf("applyEnv: %v", err)
	}
	if (*conf)["Message"] != "hello" {
		t.Fatalf("expected Message=hello, got %v", (*conf)["Message"])
	}
}

func TestApplyEnv_SubstitutesFromProcessEnvWhenNotInFile(t *testing.T) {
	t.Setenv("LXGO_CONFIG_TEST_VAR", "from-process-env")

	dir := t.TempDir()
	// The .env file must exist for applyEnv to even attempt substitution -
	// its content just doesn't mention this particular variable.
	writeFile(t, filepath.Join(dir, ".env"), "UNRELATED=1\n")

	conf := &kernel.Dict{"Value": "${LXGO_CONFIG_TEST_VAR}"}
	if err := applyEnv(conf, filepath.Join(dir, ".env"), true); err != nil {
		t.Fatalf("applyEnv: %v", err)
	}
	if (*conf)["Value"] != "from-process-env" {
		t.Fatalf("expected Value=from-process-env, got %v", (*conf)["Value"])
	}
}

// TestApplyEnv_SubstitutesFromProcessEnvWhenFileMissing is a regression
// test: applyEnv used to return early (before ever calling envToConfig) if
// the .env file didn't exist and wasn't required, leaving every
// "${VAR}" placeholder untouched even when the variable was set in the
// process environment or had its own ":-default". Substitution must still
// run in that case - there's just nothing to pre-load from a file.
func TestApplyEnv_SubstitutesFromProcessEnvWhenFileMissing(t *testing.T) {
	t.Setenv("LXGO_CONFIG_TEST_VAR_NO_FILE", "from-process-env")

	dir := t.TempDir()
	missingEnvPath := filepath.Join(dir, ".env") // intentionally not created

	conf := &kernel.Dict{
		"FromEnv":     "${LXGO_CONFIG_TEST_VAR_NO_FILE}",
		"WithDefault": "${LXGO_CONFIG_TEST_VAR_NO_FILE_MISSING:-fallback}",
	}
	if err := applyEnv(conf, missingEnvPath, false); err != nil {
		t.Fatalf("applyEnv: %v", err)
	}
	if (*conf)["FromEnv"] != "from-process-env" {
		t.Fatalf("expected FromEnv=from-process-env, got %v", (*conf)["FromEnv"])
	}
	if (*conf)["WithDefault"] != "fallback" {
		t.Fatalf("expected WithDefault=fallback, got %v", (*conf)["WithDefault"])
	}
}

func TestApplyEnv_DefaultValueWhenVarMissing(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".env"), "")

	conf := &kernel.Dict{"Value": "${LXGO_CONFIG_TEST_MISSING:-fallback}"}
	if err := applyEnv(conf, filepath.Join(dir, ".env"), true); err != nil {
		t.Fatalf("applyEnv: %v", err)
	}
	if (*conf)["Value"] != "fallback" {
		t.Fatalf("expected Value=fallback, got %v", (*conf)["Value"])
	}
}

func TestApplyEnv_ErrorWhenVarMissingAndNoDefault(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".env"), "")

	conf := &kernel.Dict{"Value": "${LXGO_CONFIG_TEST_DEFINITELY_MISSING}"}
	if err := applyEnv(conf, filepath.Join(dir, ".env"), true); err == nil {
		t.Fatalf("expected an error for a missing env variable with no default")
	}
}

func TestApplyEnv_RecursesIntoNestedDictAndSlice(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".env"), "NESTED_HOST=db.local\nLIST_ITEM=item-a\n")

	conf := &kernel.Dict{
		"Database": kernel.Dict{
			"Host": "${NESTED_HOST}",
			"Port": 5432,
		},
		"Servers": []any{"${LIST_ITEM}", "plain-item"},
	}
	if err := applyEnv(conf, filepath.Join(dir, ".env"), true); err != nil {
		t.Fatalf("applyEnv: %v", err)
	}

	db := (*conf)["Database"].(kernel.Dict)
	if db["Host"] != "db.local" {
		t.Fatalf("expected nested Database.Host=db.local, got %v", db["Host"])
	}
	if db["Port"] != 5432 {
		t.Fatalf("expected untouched Database.Port=5432, got %v", db["Port"])
	}

	servers := (*conf)["Servers"].([]any)
	if servers[0] != "item-a" {
		t.Fatalf("expected Servers[0]=item-a, got %v", servers[0])
	}
	if servers[1] != "plain-item" {
		t.Fatalf("expected untouched Servers[1]=plain-item, got %v", servers[1])
	}
}

// TestApplyEnv_SubstitutesPlaceholderEmbeddedInLargerString is a
// regression test: substitution used to only work when a value was
// exactly one "${VAR}" and nothing else - strings.Trim(str, "${}") on
// "${STATIC_SRC}/web" stripped the leading "${" but left the trailing
// "}/web" untouched (nothing in "/web" is in the "${}" cutset), so it was
// read as a variable literally named "STATIC_SRC}/web" and failed.
func TestApplyEnv_SubstitutesPlaceholderEmbeddedInLargerString(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".env"), "STATIC_SRC=http://localhost:8092\n")

	conf := &kernel.Dict{
		"Prefix": "${STATIC_SRC}/web",
		"Suffix": "prefix/${STATIC_SRC}",
		"Middle": "a-${STATIC_SRC}-b",
	}
	if err := applyEnv(conf, filepath.Join(dir, ".env"), true); err != nil {
		t.Fatalf("applyEnv: %v", err)
	}
	if want := "http://localhost:8092/web"; (*conf)["Prefix"] != want {
		t.Fatalf("Prefix = %v, want %v", (*conf)["Prefix"], want)
	}
	if want := "prefix/http://localhost:8092"; (*conf)["Suffix"] != want {
		t.Fatalf("Suffix = %v, want %v", (*conf)["Suffix"], want)
	}
	if want := "a-http://localhost:8092-b"; (*conf)["Middle"] != want {
		t.Fatalf("Middle = %v, want %v", (*conf)["Middle"], want)
	}
}

func TestApplyEnv_SubstitutesMultiplePlaceholdersInOneValue(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".env"), "HOST=example.com\nPORT=8092\n")

	conf := &kernel.Dict{"Addr": "${HOST}:${PORT}"}
	if err := applyEnv(conf, filepath.Join(dir, ".env"), true); err != nil {
		t.Fatalf("applyEnv: %v", err)
	}
	if want := "example.com:8092"; (*conf)["Addr"] != want {
		t.Fatalf("Addr = %v, want %v", (*conf)["Addr"], want)
	}
}

// An embedded placeholder with its own ":-default" still works. Uses its
// own env var name, distinct from the other tests in this file -
// applyEnv's .env loading calls the real os.Setenv (not t.Setenv), so a
// name reused across tests would leak between them via the process
// environment.
func TestApplyEnv_EmbeddedPlaceholderWithDefault(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".env"), "")

	conf := &kernel.Dict{"Outer": "${LXGO_CONFIG_TEST_EMBEDDED_DEFAULT:-/web}/assets"}
	if err := applyEnv(conf, filepath.Join(dir, ".env"), true); err != nil {
		t.Fatalf("applyEnv: %v", err)
	}
	if want := "/web/assets"; (*conf)["Outer"] != want {
		t.Fatalf("Outer = %v, want %v", (*conf)["Outer"], want)
	}
}

// A missing variable with no default still fails substitution, even when
// it's only one of several placeholders in the same value.
func TestApplyEnv_EmbeddedPlaceholderMissingNoDefault_Errors(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".env"), "KNOWN=1\n")

	conf := &kernel.Dict{"Value": "${KNOWN}-${LXGO_CONFIG_TEST_DEFINITELY_MISSING}"}
	if err := applyEnv(conf, filepath.Join(dir, ".env"), true); err == nil {
		t.Fatalf("expected an error for a missing embedded env variable with no default")
	}
}
