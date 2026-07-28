package component

import (
	"testing"
	"time"

	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/apptest"
)

// newTestWSServer builds a real *WSServer via a real (apptest-backed)
// kernel.IApp - Port 0 lets the OS pick a free ephemeral port; the test
// itself polls s.listener once Start() is running to find out which one
// (see TestWSServer_StopDoesNotHangAcceptLoop).
func newTestWSServer(t *testing.T, wsCfg kernel.Dict) *WSServer {
	t.Helper()
	merged := kernel.Dict{"Host": "127.0.0.1", "Port": 0}
	for k, v := range wsCfg {
		merged[k] = v
	}

	app, err := apptest.New(kernel.Dict{
		"Components": kernel.Dict{"WSServer": merged},
	})
	if err != nil {
		t.Fatalf("apptest.New: %v", err)
	}
	if err := SetAppComponent(app, "Components.WSServer"); err != nil {
		t.Fatalf("SetAppComponent: %v", err)
	}
	s, err := AppComponent(app)
	if err != nil {
		t.Fatalf("AppComponent: %v", err)
	}
	return s
}

// TestWSServer_StopDoesNotHangAcceptLoop is a regression test for a real,
// previously-fixed bug: Start()'s accept loop used to spin forever (treating
// the listener-closed error from Stop() like any other transient accept
// error and just continuing) instead of recognizing net.ErrClosed and
// returning. If that regressed, Start() would never return here and the
// test would time out.
func TestWSServer_StopDoesNotHangAcceptLoop(t *testing.T) {
	s := newTestWSServer(t, nil)

	done := make(chan error, 1)
	go func() { done <- s.Start() }()

	hasListener := func() bool {
		s.mu.Lock()
		defer s.mu.Unlock()
		return s.listener != nil
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && !hasListener() {
		time.Sleep(10 * time.Millisecond)
	}
	if !hasListener() {
		t.Fatalf("server never started listening")
	}

	s.Stop()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("expected Start() to return nil once Stop() closes the listener, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("Start()'s accept loop is still running after Stop() - regression on the historical infinite-loop bug")
	}
}
