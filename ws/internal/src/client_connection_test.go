package src

import (
	"bufio"
	"io"
	"net"
	"strings"
	"testing"
	"time"
)

// fakeWSServerSide plays the server half of the handshake by hand, over one
// end of a net.Pipe - just enough of RFC 6455 to drive ClientConnection's
// handshake() without a real TCP listener.
type fakeWSServerSide struct {
	t    *testing.T
	conn net.Conn
}

// readRequest reads the client's GET request line + headers and returns the
// headers, lowercased by name.
func (f *fakeWSServerSide) readRequest() map[string]string {
	f.t.Helper()
	reader := bufio.NewReader(f.conn)
	line, err := reader.ReadString('\n')
	if err != nil {
		f.t.Fatalf("read request line: %v", err)
	}
	if !strings.HasPrefix(line, "GET ") {
		f.t.Fatalf("expected a GET request line, got %q", line)
	}

	headers := map[string]string{}
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			f.t.Fatalf("read header line: %v", err)
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		if i := strings.Index(line, ":"); i != -1 {
			headers[strings.ToLower(strings.TrimSpace(line[:i]))] = strings.TrimSpace(line[i+1:])
		}
	}
	return headers
}

// respondSwitchingProtocols writes a 101 response with the given
// Sec-WebSocket-Accept value (deliberately wrong in the negative test).
func (f *fakeWSServerSide) respondSwitchingProtocols(accept string) {
	f.t.Helper()
	resp := "HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: " + accept + "\r\n" +
		"\r\n"
	if _, err := f.conn.Write([]byte(resp)); err != nil {
		f.t.Fatalf("write handshake response: %v", err)
	}
}

// TestClientConnection_Handshake_SucceedsOnValidAccept exercises the happy
// path: the fake server computes the real accept key from the client's
// Sec-WebSocket-Key, exactly like a real WSServer would.
func TestClientConnection_Handshake_SucceedsOnValidAccept(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	t.Cleanup(func() { clientConn.Close() })
	server := &fakeWSServerSide{t: t, conn: serverConn}

	result := make(chan error, 1)
	go func() {
		_, err := newClientConnection(clientConn, "example.com", "/")
		result <- err
	}()

	headers := server.readRequest()
	if headers["upgrade"] != "websocket" {
		t.Fatalf("expected an Upgrade: websocket header, got %q", headers["upgrade"])
	}
	key := headers["sec-websocket-key"]
	if key == "" {
		t.Fatalf("expected a non-empty Sec-WebSocket-Key")
	}
	server.respondSwitchingProtocols(computeAcceptKey(key))

	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("handshake should have succeeded, got: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for the handshake to finish")
	}
}

// TestClientConnection_Handshake_RejectsWrongAccept is a regression test for
// the whole reason to check Sec-WebSocket-Accept at all: an unrelated party
// speaking plain HTTP (or a misconfigured server on the wrong port) would
// otherwise be silently accepted as if it were a real WS peer.
func TestClientConnection_Handshake_RejectsWrongAccept(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	t.Cleanup(func() { clientConn.Close() })
	server := &fakeWSServerSide{t: t, conn: serverConn}

	result := make(chan error, 1)
	go func() {
		_, err := newClientConnection(clientConn, "example.com", "/")
		result <- err
	}()

	server.readRequest()
	server.respondSwitchingProtocols("not-the-right-accept-key")

	select {
	case err := <-result:
		if err == nil {
			t.Fatalf("expected the handshake to fail on a wrong Sec-WebSocket-Accept")
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for the handshake to finish")
	}
}

// TestClientConnection_Send_AlwaysMasks is a regression test for the RFC
// 6455 5.1 requirement that every client->server frame be masked -
// Connection.Send (the server side) always sends masked:false, and it would
// be an easy copy-paste mistake to do the same here.
func TestClientConnection_Send_AlwaysMasks(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	t.Cleanup(func() { clientConn.Close() })
	server := &fakeWSServerSide{t: t, conn: serverConn}

	handshakeDone := make(chan *ClientConnection, 1)
	go func() {
		c, err := newClientConnection(clientConn, "example.com", "/")
		if err != nil {
			t.Errorf("handshake: %v", err)
			handshakeDone <- nil
			return
		}
		handshakeDone <- c
	}()

	key := server.readRequest()["sec-websocket-key"]
	server.respondSwitchingProtocols(computeAcceptKey(key))

	c := <-handshakeDone
	if c == nil {
		t.Fatalf("setup: handshake failed")
	}

	sendErr := make(chan error, 1)
	go func() { sendErr <- c.Send(map[string]any{"hello": "world"}, "text") }()

	// Parse the frame header by hand instead of via hybi10Decode - decode
	// unmasks transparently either way, which would hide a masked:false bug
	// completely (the visible payload would still come out right). The mask
	// bit itself is what this test needs to see.
	reader := bufio.NewReader(serverConn)
	header := make([]byte, 2)
	if _, err := io.ReadFull(reader, header); err != nil {
		t.Fatalf("read frame header: %v", err)
	}
	if header[1]&0x80 == 0 {
		t.Fatalf("expected the mask bit set on a client->server frame (RFC 6455 5.1), got second byte %08b", header[1])
	}
	payloadLen := int(header[1] & 0x7F)
	maskKey := make([]byte, 4)
	if _, err := io.ReadFull(reader, maskKey); err != nil {
		t.Fatalf("read mask key: %v", err)
	}
	payload := make([]byte, payloadLen)
	if _, err := io.ReadFull(reader, payload); err != nil {
		t.Fatalf("read payload: %v", err)
	}
	for i := range payload {
		payload[i] ^= maskKey[i%4]
	}

	if err := <-sendErr; err != nil {
		t.Fatalf("Send: %v", err)
	}
	if string(payload) != `{"hello":"world"}` {
		t.Fatalf("unmasked payload = %q, want the original JSON", payload)
	}
}

// handshakeOverPipe completes a client/server handshake over a fresh
// net.Pipe() and returns the resulting *ClientConnection plus the server
// side's raw net.Conn, for tests that need to drive frames after the
// handshake is done.
func handshakeOverPipe(t *testing.T) (*ClientConnection, net.Conn) {
	t.Helper()
	clientConn, serverConn := net.Pipe()
	t.Cleanup(func() { clientConn.Close() })
	server := &fakeWSServerSide{t: t, conn: serverConn}

	result := make(chan *ClientConnection, 1)
	go func() {
		c, err := newClientConnection(clientConn, "example.com", "/")
		if err != nil {
			t.Errorf("handshake: %v", err)
			result <- nil
			return
		}
		result <- c
	}()

	key := server.readRequest()["sec-websocket-key"]
	server.respondSwitchingProtocols(computeAcceptKey(key))

	c := <-result
	if c == nil {
		t.Fatalf("setup: handshake failed")
	}
	return c, serverConn
}

// TestClientConnection_SendAfterClose_ReturnsCleanError is a regression test:
// Close used to leave c.conn/c.reader non-nil, so Send/Receive's own
// "already closed" checks were dead code - a Send after Close fell through
// to whatever raw error net.Conn.Write happened to return instead of the
// clean one the code otherwise looks like it guarantees.
func TestClientConnection_SendAfterClose_ReturnsCleanError(t *testing.T) {
	c, _ := handshakeOverPipe(t)

	if err := c.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	err := c.Send(map[string]any{"hello": "world"}, "text")
	if err == nil {
		t.Fatalf("expected Send after Close to fail")
	}
	if !strings.Contains(err.Error(), "already closed") {
		t.Fatalf(`expected the clean "already closed" error, got: %v`, err)
	}
}

// TestClientConnection_Receive_SkipsUnknownOpcode is a regression test:
// Receive's default case used to return a hard error on any unrecognized
// opcode, ending the whole read loop over one stray frame - unlike
// Connection.Handle's own default case, which just logs and keeps going.
// A single unexpected frame shouldn't need to be fatal for an otherwise
// live connection.
func TestClientConnection_Receive_SkipsUnknownOpcode(t *testing.T) {
	c, serverConn := handshakeOverPipe(t)

	go func() {
		// 0x3 is a reserved (non-control, non-continuation/text/binary)
		// opcode - never sent by this package's own server, but a
		// well-behaved client must not choke if it ever arrives.
		reserved, err := hybi10Encode([]byte("ignored"), 0x3, false)
		if err != nil {
			t.Errorf("encode reserved-opcode frame: %v", err)
			return
		}
		if _, err := serverConn.Write(reserved); err != nil {
			t.Errorf("write reserved-opcode frame: %v", err)
			return
		}

		real, err := hybi10Encode([]byte(`{"ok":true}`), 0x1, false)
		if err != nil {
			t.Errorf("encode text frame: %v", err)
			return
		}
		if _, err := serverConn.Write(real); err != nil {
			t.Errorf("write text frame: %v", err)
		}
	}()

	msg, err := c.Receive()
	if err != nil {
		t.Fatalf("Receive should have skipped the reserved opcode and returned the text frame, got error: %v", err)
	}
	m, ok := msg.(map[string]any)
	if !ok || m["ok"] != true {
		t.Fatalf("expected {\"ok\":true}, got %#v", msg)
	}
}
