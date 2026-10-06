package http

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/epicoon/lxgo/kernel"
)

func writeAssetFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, filepath.Dir(name)), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func TestRegisterFileAssets_ServesFile(t *testing.T) {
	dir := t.TempDir()
	writeAssetFile(t, dir, "foo.png", "fake-png-bytes")

	router := NewRouter(nil).(*Router)
	router.RegisterFileAssets(map[string]string{"/img/": dir})

	srv := httptest.NewServer(router)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/img/foo.png")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "fake-png-bytes" {
		t.Fatalf("body = %q, want %q", body, "fake-png-bytes")
	}
}

// A nested path under the registered prefix must resolve too - the
// wildcard route captures every remaining segment, not just one.
func TestRegisterFileAssets_ServesNestedPath(t *testing.T) {
	dir := t.TempDir()
	writeAssetFile(t, dir, "icons/close.png", "icon-bytes")

	router := NewRouter(nil).(*Router)
	router.RegisterFileAssets(map[string]string{"/img/": dir})

	srv := httptest.NewServer(router)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/img/icons/close.png")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "icon-bytes" {
		t.Fatalf("body = %q, want %q", body, "icon-bytes")
	}
}

// A prefix registered without a trailing slash must work the same as one
// with it.
func TestRegisterFileAssets_PrefixWithoutTrailingSlash(t *testing.T) {
	dir := t.TempDir()
	writeAssetFile(t, dir, "foo.css", "body{}")

	router := NewRouter(nil).(*Router)
	router.RegisterFileAssets(map[string]string{"/css": dir})

	srv := httptest.NewServer(router)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/css/foo.css")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

// Regression: asset routes used to be registered via plain http.Handle,
// serving any method - registering the new template route under "GET"
// specifically (instead of "" -> "ALL") would silently 405 a HEAD request.
func TestRegisterFileAssets_ServesHeadRequest(t *testing.T) {
	dir := t.TempDir()
	writeAssetFile(t, dir, "foo.png", "fake-png-bytes")

	router := NewRouter(nil).(*Router)
	router.RegisterFileAssets(map[string]string{"/img/": dir})

	srv := httptest.NewServer(router)
	defer srv.Close()

	resp, err := http.Head(srv.URL + "/img/foo.png")
	if err != nil {
		t.Fatalf("Head: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

func TestRegisterFileAssets_MissingFile_NotFound(t *testing.T) {
	router := NewRouter(nil).(*Router)
	router.RegisterFileAssets(map[string]string{"/img/": t.TempDir()})

	srv := httptest.NewServer(router)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/img/does-not-exist.png")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

// This is the whole point of the refactor this test accompanies: asset
// routes used to be registered directly on http.DefaultServeMux, bypassing
// Router's middleware pipeline entirely - a middleware (e.g. the CORS
// component this unification unblocks) never ran for them. It must now.
func TestRegisterFileAssets_RunsThroughMiddleware(t *testing.T) {
	dir := t.TempDir()
	writeAssetFile(t, dir, "foo.png", "x")

	router := NewRouter(nil).(*Router)
	var middlewareRan bool
	router.AddMiddleware(func(ctx kernel.IHandleContext) error {
		middlewareRan = true
		return nil
	})
	router.RegisterFileAssets(map[string]string{"/img/": dir})

	srv := httptest.NewServer(router)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/img/foo.png")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer resp.Body.Close()

	if !middlewareRan {
		t.Fatal("expected the router's middleware to run for an asset request, it didn't")
	}
}

// GetAssetRoute's reverse lookup doesn't depend on how the route was
// registered - regression check that it still works post-refactor.
func TestGetAssetRoute_FindsRegisteredPrefix(t *testing.T) {
	dir := t.TempDir()
	router := NewRouter(nil).(*Router)
	router.RegisterFileAssets(map[string]string{"/img/": dir})

	if got := router.GetAssetRoute(dir); got != "/img/" {
		t.Fatalf("GetAssetRoute(%q) = %q, want \"/img/\"", dir, got)
	}
	if got := router.GetAssetRoute("/no/such/dir"); got != "" {
		t.Fatalf("GetAssetRoute for an unregistered dir = %q, want \"\"", got)
	}
}
