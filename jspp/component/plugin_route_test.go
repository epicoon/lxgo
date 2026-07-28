package component_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/epicoon/lxgo/jspp"
	"github.com/epicoon/lxgo/jspp/component"
	"github.com/epicoon/lxgo/jspp/plugins"
	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/apptest"
	lxHttp "github.com/epicoon/lxgo/kernel/http"
)

// testPlugin is a minimal jspp.IPlugin (embedding the base plugins.Plugin)
// with one ajax route, resolved through the DI container - mirroring how a
// real application registers a plugin's Go constructor under Eplugin.
type testPlugin struct {
	*plugins.Plugin
}

func (p *testPlugin) AjaxHandlers() kernel.HttpResourcesList {
	return kernel.HttpResourcesList{
		"ping": func() kernel.IHttpResource {
			return &pluginPingHandler{Resource: lxHttp.NewResource()}
		},
	}
}

type pluginPingHandler struct {
	*lxHttp.Resource
}

func (h *pluginPingHandler) Run() kernel.IHttpResponse {
	return h.JsonResponse(kernel.JsonResponseConfig{
		Data: map[string]any{"pong": true},
	})
}

// TestPluginRoute_RealHTTPRoundTrip registers a real plugin (on-disk
// lx-plugin.yaml + a Go constructor in the DI container, wired through the
// plugins map exactly as PluginManager.Get resolves it) and dispatches a
// real HTTP request to /lx/plugin, exercising PluginManager.Get, the
// plugin's own AjaxHandlers routing, and the request-body rewrite in
// PluginHandler.Run end to end.
func TestPluginRoute_RealHTTPRoundTrip(t *testing.T) {
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
	if err := app.DIContainer().Register(kernel.CAnyList{
		"testPluginCtor": func(...any) any {
			return &testPlugin{Plugin: plugins.NewPlugin()}
		},
	}); err != nil {
		t.Fatalf("DIContainer().Register: %v", err)
	}

	pluginDir := filepath.Join(sysPath, "myPlugin")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatalf("mkdir plugin dir: %v", err)
	}
	yamlPath := filepath.Join(pluginDir, "lx-plugin.yaml")
	if err := os.WriteFile(yamlPath, []byte("name: myPlugin\n"), 0644); err != nil {
		t.Fatalf("write lx-plugin.yaml: %v", err)
	}

	pp, err := component.AppComponent(app)
	if err != nil {
		t.Fatalf("AppComponent: %v", err)
	}
	pm := pp.PluginManager()
	if err := pm.Save([]jspp.IPluginData{pm.NewData("myPlugin", pluginDir, "testPluginCtor")}); err != nil {
		t.Fatalf("save plugins map: %v", err)
	}

	srv := apptest.Server(app)
	defer srv.Close()

	body, _ := json.Marshal(map[string]any{
		"plugin": "myPlugin",
		"path":   "ping",
		"params": map[string]any{},
	})
	resp, err := http.Post(srv.URL+"/lx/plugin", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /lx/plugin: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var decoded struct {
		Pong bool `json:"pong"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !decoded.Pong {
		t.Fatalf("expected pong:true in the response")
	}
}

// TestPluginRoute_UnknownPlugin_ReturnsError checks the "plugin not found"
// path over the same real HTTP round-trip.
func TestPluginRoute_UnknownPlugin_ReturnsError(t *testing.T) {
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

	body, _ := json.Marshal(map[string]any{
		"plugin": "neverRegistered",
		"path":   "ping",
		"params": map[string]any{},
	})
	resp, err := http.Post(srv.URL+"/lx/plugin", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /lx/plugin: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		t.Fatalf("expected a non-200 error response for an unknown plugin")
	}
}

var _ jspp.IPlugin = (*testPlugin)(nil)
