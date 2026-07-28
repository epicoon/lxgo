package component_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/epicoon/lxgo/jspp"
	"github.com/epicoon/lxgo/jspp/component"
	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/apptest"
)

// TestServiceRoute_RealHTTPRoundTrip is an integration test: a real
// kernel.IApp, with the JSPreprocessor component set up exactly as a real
// application would (component.SetAppComponent), serving a real HTTP
// request to /lx/service through httptest.Server - end to end through the
// router, the AfterInit-registered middleware that injects "jspp" into the
// request context, ServiceHandler, and a real (if trivial) compiler run.
func TestServiceRoute_RealHTTPRoundTrip(t *testing.T) {
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

	// Register one real module so "need" has something to actually fetch -
	// with nothing needed there's genuinely nothing to compile, which the
	// handler correctly reports as a 400 (see TestServiceRoute_UnknownAction
	// for that path; this test is about the success path).
	pp, err := component.AppComponent(app)
	if err != nil {
		t.Fatalf("AppComponent: %v", err)
	}
	modPath := filepath.Join(sysPath, "Greeter.js")
	if err := os.WriteFile(modPath, []byte("@lx:module Greeter;\nclass Greeter {}\n"), 0644); err != nil {
		t.Fatalf("write module file: %v", err)
	}
	mm := pp.ModulesMap()
	if err := mm.Save([]jspp.IJSModuleData{mm.NewData("Greeter", modPath)}); err != nil {
		t.Fatalf("save modules map: %v", err)
	}

	srv := apptest.Server(app)
	defer srv.Close()

	body, _ := json.Marshal(map[string]any{
		"action": "get-modules",
		"params": map[string]any{
			"have": []string{},
			"need": []string{"Greeter"},
		},
	})
	resp, err := http.Post(srv.URL+"/lx/service", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /lx/service: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var decoded struct {
		Success bool `json:"success"`
		Data    struct {
			Code            string   `json:"code"`
			CompiledModules []string `json:"compiledModules"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !decoded.Success {
		t.Fatalf("expected success:true in the response")
	}
	if !strings.Contains(decoded.Data.Code, "class Greeter") {
		t.Fatalf("expected Greeter's compiled code in the response, got: %s", decoded.Data.Code)
	}
}

// TestServiceRoute_UnknownAction is the same real HTTP round-trip, checking
// the "unknown action" error path returns a clean 400, not a crash.
func TestServiceRoute_UnknownAction(t *testing.T) {
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

	srv := apptest.Server(app)
	defer srv.Close()

	body, _ := json.Marshal(map[string]any{"action": "not-a-real-action"})
	resp, err := http.Post(srv.URL+"/lx/service", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /lx/service: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}
