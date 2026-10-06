package component

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/apptest"
)

// newAppWithWSServerConfig builds a real (apptest-backed) kernel.IApp with
// Components.WSServer set to exactly wsCfg - unlike newTestWSServer, it
// never injects a default Port, so a caller can genuinely leave the key
// out entirely (as opposed to setting it to 0, the legitimate "let the OS
// pick an ephemeral port" value - see wsConfigPortMissing).
func newAppWithWSServerConfig(t *testing.T, wsCfg kernel.Dict) kernel.IApp {
	t.Helper()
	app, err := apptest.New(kernel.Dict{
		"Components": kernel.Dict{"WSServer": wsCfg},
	})
	if err != nil {
		t.Fatalf("apptest.New: %v", err)
	}
	return app
}

func TestWsConfigPortMissing_DistinguishesAbsentFromZero(t *testing.T) {
	cases := []struct {
		name string
		cfg  kernel.Dict
		want bool
	}{
		{"absent entirely", kernel.Dict{"Host": "127.0.0.1"}, true},
		{"explicit zero (ephemeral port)", kernel.Dict{"Host": "127.0.0.1", "Port": 0}, false},
		{"explicit nonzero", kernel.Dict{"Host": "127.0.0.1", "Port": 8093}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			app := newAppWithWSServerConfig(t, c.cfg)
			if err := SetAppComponent(app, "Components.WSServer"); err != nil {
				t.Fatalf("SetAppComponent: %v", err)
			}
			s, err := AppComponent(app)
			if err != nil {
				t.Fatalf("AppComponent: %v", err)
			}
			if got := wsConfigPortMissing(s); got != c.want {
				t.Fatalf("wsConfigPortMissing() = %v, want %v", got, c.want)
			}
		})
	}
}

// TestWSServer_ServeUpgrade_DialRoundTrips drives serveUpgrade directly
// against a real httptest.Server (a real listener, a real net/http
// connection that genuinely supports http.Hijacker) - deliberately not
// through Start()/http.DefaultServeMux, since registering wsHTTPPath on
// the process-wide default mux from a test would collide across runs/
// other tests with no way to unregister it afterward. serveUpgrade itself
// - the actual hijack-and-handshake logic - doesn't care how it got
// mounted, so this still genuinely exercises it end-to-end.
func TestWSServer_ServeUpgrade_DialRoundTrips(t *testing.T) {
	app := newAppWithWSServerConfig(t, kernel.Dict{})
	if err := SetAppComponent(app, "Components.WSServer"); err != nil {
		t.Fatalf("SetAppComponent: %v", err)
	}
	s, err := AppComponent(app)
	if err != nil {
		t.Fatalf("AppComponent: %v", err)
	}
	s.channels.Init()

	srv := httptest.NewServer(http.HandlerFunc(s.serveUpgrade))
	t.Cleanup(srv.Close)
	addr := strings.TrimPrefix(srv.URL, "http://")

	pushes := make(chan any, 8)
	client, err := Dial(addr, wsHTTPPath, func(msg any) { pushes <- msg }, nil)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	t.Cleanup(func() { client.Close() })

	msg := recvPush(t, pushes) // handshake ack
	ack, ok := msg.(map[string]any)
	if !ok {
		t.Fatalf("expected the handshake ack to be a JSON object, got %#v", msg)
	}
	if id, _ := ack["id"].(string); id == "" {
		t.Fatalf("expected the handshake ack to carry a non-empty id, got %#v", ack)
	}

	if err := client.Send(map[string]any{"__lxws_action__": "connect"}, "text"); err != nil {
		t.Fatalf("Send(connect): %v", err)
	}

	msg = recvPush(t, pushes) // connect ack
	connectAck, ok := msg.(map[string]any)
	if !ok || connectAck["__lxws_action__"] != "connect" {
		t.Fatalf("expected a connect ack, got %#v", msg)
	}
}

// TestWSServer_ServeUpgrade_RejectsPlainHTTPRequest confirms serveUpgrade
// doesn't hijack (and thus doesn't try to speak WS frames over) a request
// that was never asking to be upgraded in the first place - an ordinary
// request to wsHTTPPath, same as any other path the app's HTTP router
// happens to answer, must get a normal HTTP error response, not a
// protocol violation.
func TestWSServer_ServeUpgrade_RejectsPlainHTTPRequest(t *testing.T) {
	app := newAppWithWSServerConfig(t, kernel.Dict{})
	if err := SetAppComponent(app, "Components.WSServer"); err != nil {
		t.Fatalf("SetAppComponent: %v", err)
	}
	s, err := AppComponent(app)
	if err != nil {
		t.Fatalf("AppComponent: %v", err)
	}
	s.channels.Init()

	srv := httptest.NewServer(http.HandlerFunc(s.serveUpgrade))
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + wsHTTPPath)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected %d for a non-upgrade request, got %d", http.StatusBadRequest, resp.StatusCode)
	}
}

// TestWSServer_ServeUpgrade_RejectsMalformedUpgradeRequest mirrors
// TestWSServer_ServeUpgrade_RejectsPlainHTTPRequest for the other
// handshake-failure shape: an Upgrade header is present, but the request
// is still missing what buildUpgradeResponse requires (Sec-WebSocket-Key).
// Unlike the plain-HTTP case, this exercises the path through Hijack()
// itself failing validation before ever hijacking the connection.
func TestWSServer_ServeUpgrade_RejectsMalformedUpgradeRequest(t *testing.T) {
	app := newAppWithWSServerConfig(t, kernel.Dict{})
	if err := SetAppComponent(app, "Components.WSServer"); err != nil {
		t.Fatalf("SetAppComponent: %v", err)
	}
	s, err := AppComponent(app)
	if err != nil {
		t.Fatalf("AppComponent: %v", err)
	}
	s.channels.Init()

	srv := httptest.NewServer(http.HandlerFunc(s.serveUpgrade))
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodGet, srv.URL+wsHTTPPath, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	// Deliberately no Sec-WebSocket-Key.

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected %d for an upgrade request missing Sec-WebSocket-Key, got %d", http.StatusBadRequest, resp.StatusCode)
	}
}

// TestWSServer_ServeUpgrade_IPLimitRejectsBeforeHandshakeResponse is a
// regression test: the IP connection-count limit used to be checked only
// inside HandleHijacked, which by then had already written and flushed a
// successful "101 Switching Protocols" response - an over-the-limit client
// would see a successful-looking upgrade and only then get dropped,
// instead of never getting a successful response at all (as a raw-listener
// connection over the same limit never does, see Handle). The fix checks
// the limit in serveUpgrade, before WriteUpgradeResponse - verified here
// by actually dialing twice against a MaxConnectionsPerIp: 1 server and
// requiring the second Dial to fail outright, not succeed and then drop.
func TestWSServer_ServeUpgrade_IPLimitRejectsBeforeHandshakeResponse(t *testing.T) {
	app := newAppWithWSServerConfig(t, kernel.Dict{"MaxConnectionsPerIp": 1})
	if err := SetAppComponent(app, "Components.WSServer"); err != nil {
		t.Fatalf("SetAppComponent: %v", err)
	}
	s, err := AppComponent(app)
	if err != nil {
		t.Fatalf("AppComponent: %v", err)
	}
	s.channels.Init()

	srv := httptest.NewServer(http.HandlerFunc(s.serveUpgrade))
	t.Cleanup(srv.Close)
	addr := strings.TrimPrefix(srv.URL, "http://")

	pushes := make(chan any, 8)
	client, err := Dial(addr, wsHTTPPath, func(msg any) { pushes <- msg }, nil)
	if err != nil {
		t.Fatalf("Dial (first, takes the one allowed slot): %v", err)
	}
	t.Cleanup(func() { client.Close() })
	recvPush(t, pushes) // handshake ack

	if _, err := Dial(addr, wsHTTPPath, func(any) {}, nil); err == nil {
		t.Fatalf("expected the second Dial (same IP, over MaxConnectionsPerIp) to fail outright with no handshake response, got a successful connection")
	}
}

// TestWSServer_Stop_DoesNotHangOnLiveConnection is a regression test: Stop()
// used to wait on its WaitGroup for every connection handler goroutine to
// return before closing anything - but a hijacked connection's handler sits
// in a blocking read on its socket, which never returns on its own for a
// connection that's simply idle (nothing was ever sent to make it error
// out). A real browser tab left connected and otherwise idle reproduced
// this directly: Stop() hung until the OS-level process kill, since
// s.conns.Close() (which would have run next) only ever closes already-
// disconnected (tombstoned) connections, never a still-live one. Here, the
// client deliberately never disconnects or sends anything after the
// handshake - Stop() must still close the live connection itself and
// return promptly.
func TestWSServer_Stop_DoesNotHangOnLiveConnection(t *testing.T) {
	app := newAppWithWSServerConfig(t, kernel.Dict{})
	if err := SetAppComponent(app, "Components.WSServer"); err != nil {
		t.Fatalf("SetAppComponent: %v", err)
	}
	s, err := AppComponent(app)
	if err != nil {
		t.Fatalf("AppComponent: %v", err)
	}
	s.channels.Init()

	srv := httptest.NewServer(http.HandlerFunc(s.serveUpgrade))
	t.Cleanup(srv.Close)
	addr := strings.TrimPrefix(srv.URL, "http://")

	pushes := make(chan any, 8)
	client, err := Dial(addr, wsHTTPPath, func(msg any) { pushes <- msg }, nil)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	t.Cleanup(func() { client.Close() })

	recvPush(t, pushes) // handshake ack - the connection is now live and idle

	done := make(chan struct{})
	go func() {
		s.Stop()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("Stop() did not return within 2s - a live, idle connection's blocking read is never unblocked on its own")
	}
}
