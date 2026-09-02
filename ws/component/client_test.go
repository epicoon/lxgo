package component

import (
	"testing"
	"time"
)

// listeningAddr polls s.listener until Start() (running in its own
// goroutine) has bound it, same wait pattern as
// TestWSServer_StopDoesNotHangAcceptLoop.
func listeningAddr(t *testing.T, s *WSServer) string {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		s.mu.Lock()
		l := s.listener
		s.mu.Unlock()
		if l != nil {
			return l.Addr().String()
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("server never started listening")
	return ""
}

// recvPush waits for the next message onPush collected into pushes, or
// fails the test after a short deadline - Dial's IClient owns its
// connection's read loop from the moment it's returned, so a test can't
// call Receive() directly any more; onPush is the only way to observe what
// the server sends unprompted (here: the handshake and connect acks,
// neither of which is a __lxws_request__/__lxws_response__ round trip).
func recvPush(t *testing.T, pushes <-chan any) any {
	t.Helper()
	select {
	case msg := <-pushes:
		return msg
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for a pushed message")
		return nil
	}
}

// TestDial_RoundTripsWithRealServer is an end-to-end test: unlike
// internal/src's own client_connection_test.go (net.Pipe, no real socket),
// this drives Dial's net.Dial half for real, against a real WSServer over
// an actual TCP loopback connection.
func TestDial_RoundTripsWithRealServer(t *testing.T) {
	s := newTestWSServer(t, nil)
	go s.Start()
	t.Cleanup(s.Stop)
	addr := listeningAddr(t, s)

	pushes := make(chan any, 8)
	client, err := Dial(addr, "/", func(msg any) { pushes <- msg }, nil)
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
