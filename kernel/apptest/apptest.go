// Package apptest builds a minimal kernel.IApp for other lxgo-* packages'
// integration tests - New builds and initializes a ready-to-use app.App
// from an in-memory config (no config.yaml file needed); register the
// component under test on it via app.RegisterComponent (or register
// routes/middleware directly on its Router()), then use Server to get a
// real net/http test server for HTTP round-trips.
package apptest

import (
	"net/http"
	"net/http/httptest"

	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/app"
)

// New builds and initializes a minimal app.App for integration tests - no
// config.yaml file is read, cfg (if given) is used as-is. "Port" defaults
// to 0 unless cfg sets it; a "Database" section (if any) wires up a
// connection, but Connect() still needs to be called explicitly. Only the
// first cfg is used - it's variadic so New() with no config is the common case.
func New(cfg ...kernel.Dict) (kernel.IApp, error) {
	merged := kernel.Dict{"Port": 0}
	if len(cfg) > 0 {
		for k, v := range cfg[0] {
			merged[k] = v
		}
	}

	a := app.NewApp()
	if err := app.InitApp(a, merged); err != nil {
		return nil, err
	}
	return a, nil
}

// Server wraps a's router in a real net/http test server, listening on a
// free local port - the caller must Close() it when done. Panics if a
// wasn't built by New (its router doesn't implement http.Handler).
func Server(a kernel.IApp) *httptest.Server {
	handler, ok := a.Router().(http.Handler)
	if !ok {
		panic("apptest: app's router does not implement http.Handler")
	}
	return httptest.NewServer(handler)
}
