// This file is package http_test (not http) specifically to reach
// apptest - package http itself can't import apptest without an import
// cycle (apptest -> app -> http).
package http_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/apptest"
	lxHttp "github.com/epicoon/lxgo/kernel/http"
)

// EVENT_APP_BEFORE_SEND_ASSET must still fire, with the resolved file path
// and the original (unstripped) request - RegisterFileAssets' handler
// used to see a request already mutated by http.StripPrefix; the one
// going through the router's own pipeline doesn't strip anything, it
// reads the remainder from PathSegments instead.
func TestRegisterFileAssets_TriggersBeforeSendAssetEvent(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "foo.png"), []byte("x"), 0644); err != nil {
		t.Fatalf("write foo.png: %v", err)
	}

	app, err := apptest.New(kernel.Dict{"Components": kernel.Dict{}})
	if err != nil {
		t.Fatalf("apptest.New: %v", err)
	}

	var gotFile string
	var gotPath string
	app.Events().Subscribe(kernel.EVENT_APP_BEFORE_SEND_ASSET, func(e kernel.IEvent) {
		gotFile, _ = e.Payload().Get("file").(string)
		if req, ok := e.Payload().Get("request").(*http.Request); ok {
			gotPath = req.URL.Path
		}
	})

	router := app.Router().(*lxHttp.Router)
	router.RegisterFileAssets(map[string]string{"/img/": dir})

	srv := httptest.NewServer(router)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/img/foo.png")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer resp.Body.Close()

	if gotPath != "/img/foo.png" {
		t.Fatalf("event's request.URL.Path = %q, want \"/img/foo.png\"", gotPath)
	}
	wantSuffix := filepath.Join(dir, "foo.png")
	if gotFile != wantSuffix {
		t.Fatalf("event's file = %q, want %q", gotFile, wantSuffix)
	}
}
