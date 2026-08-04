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

func TestLoad_MergesLocalConfigRecursively(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	writeFile(t, cfgPath, ""+
		"Local: config-local.yaml\n"+
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
	// The local config's Database.Host overrides the main one...
	if db["Host"] != "localhost" {
		t.Fatalf("expected local override Database.Host=localhost, got %v", db["Host"])
	}
	// ...but merging is recursive, not a wholesale replace: Database.Port
	// wasn't mentioned in the local config, so it must survive untouched.
	if v, ok := db["Port"]; !ok || v != 5432 {
		t.Fatalf("expected untouched Database.Port=5432, got %v (ok=%v)", v, ok)
	}
	// A top-level key the local config doesn't touch at all must survive too.
	name, err := GetParam[string](conf, "Name")
	if err != nil || name != "main" {
		t.Fatalf("expected untouched Name=main, got %v (err=%v)", name, err)
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
