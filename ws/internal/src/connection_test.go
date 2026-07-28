package src

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net"
	"strings"
	"testing"

	"github.com/epicoon/lxgo/ws"
)

func TestHybi10EncodeDecode_RoundTrip(t *testing.T) {
	cases := []struct {
		name    string
		payload []byte
		masked  bool
	}{
		{"empty", []byte{}, false},
		{"short-unmasked", []byte("hello"), false},
		{"short-masked", []byte("hello"), true},
		{"medium-16bit-length", make([]byte, 200), false},
		{"large-64bit-length", make([]byte, 70000), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			frame, err := hybi10Encode(tc.payload, 0x1, tc.masked)
			if err != nil {
				t.Fatalf("encode: %v", err)
			}
			opcode, payload, fin, err := hybi10Decode(bufio.NewReader(bytes.NewReader(frame)))
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			if opcode != 0x1 {
				t.Fatalf("expected opcode 0x1, got %#x", opcode)
			}
			if !fin {
				t.Fatalf("expected fin=true (this package always sends single-frame messages)")
			}
			if len(payload) != len(tc.payload) {
				t.Fatalf("expected payload length %d, got %d", len(tc.payload), len(payload))
			}
			for i := range tc.payload {
				if payload[i] != tc.payload[i] {
					t.Fatalf("payload mismatch at byte %d: want %d got %d", i, tc.payload[i], payload[i])
				}
			}
		})
	}
}

// wsTestClient drives a real *Connection (via net.Pipe, no TCP) through its
// handshake and message loop - it's a minimal hand-rolled WS client, just
// enough to exercise Connection.Handle()'s dispatch without a browser.
type wsTestClient struct {
	t      *testing.T
	client net.Conn
	reader *bufio.Reader
	// conn is the real server-side ws.IConnection backing this client - most
	// tests only need the client side, but a few integration tests call
	// methods on it directly (e.g. simulating this connection's own
	// disconnect synchronously, rather than through Handle()'s background
	// goroutine).
	conn ws.IConnection
}

func newWSTestClient(t *testing.T, s *fakeServer, origin string) *wsTestClient {
	t.Helper()
	clientConn, serverConn := net.Pipe()
	conn := NewConnection(s, serverConn)
	go conn.Handle()
	t.Cleanup(func() { clientConn.Close() })

	req := "GET / HTTP/1.1\r\n" +
		"Host: example.com\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\n" +
		"Sec-WebSocket-Version: 13\r\n"
	if origin != "" {
		req += "Origin: " + origin + "\r\n"
	}
	req += "\r\n"

	if _, err := clientConn.Write([]byte(req)); err != nil {
		t.Fatalf("write handshake request: %v", err)
	}

	c := &wsTestClient{t: t, client: clientConn, reader: bufio.NewReader(clientConn), conn: conn}
	return c
}

func (c *wsTestClient) readHandshakeResponse() map[string]string {
	c.t.Helper()
	headers := map[string]string{}
	status, err := c.reader.ReadString('\n')
	if err != nil {
		c.t.Fatalf("read status line: %v", err)
	}
	headers["__status__"] = strings.TrimRight(status, "\r\n")
	for {
		line, err := c.reader.ReadString('\n')
		if err != nil {
			c.t.Fatalf("read header line: %v", err)
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		if i := strings.Index(line, ":"); i != -1 {
			headers[strings.TrimSpace(line[:i])] = strings.TrimSpace(line[i+1:])
		}
	}
	return headers
}

func (c *wsTestClient) readCloseFrame() (code uint16, reason string) {
	c.t.Helper()
	opcode, payload, _, err := hybi10Decode(c.reader)
	if err != nil {
		c.t.Fatalf("read close frame: %v", err)
	}
	if opcode != 0x8 {
		c.t.Fatalf("expected a close frame (0x8), got opcode %#x", opcode)
	}
	if len(payload) < 2 {
		return 0, ""
	}
	code = uint16(payload[0])<<8 | uint16(payload[1])
	return code, string(payload[2:])
}

func (c *wsTestClient) readJSON() map[string]any {
	c.t.Helper()
	opcode, payload, _, err := hybi10Decode(c.reader)
	if err != nil {
		c.t.Fatalf("read frame: %v", err)
	}
	if opcode != 0x1 {
		c.t.Fatalf("expected a text frame (0x1), got opcode %#x", opcode)
	}
	var m map[string]any
	if err := json.Unmarshal(payload, &m); err != nil {
		c.t.Fatalf("unmarshal frame payload %q: %v", payload, err)
	}
	return m
}

func (c *wsTestClient) sendJSON(v any) {
	c.t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		c.t.Fatalf("marshal: %v", err)
	}
	frame, err := hybi10Encode(b, 0x1, false)
	if err != nil {
		c.t.Fatalf("encode: %v", err)
	}
	if _, err := c.client.Write(frame); err != nil {
		c.t.Fatalf("write frame: %v", err)
	}
}

func TestConnection_Handshake_ComputesRFC6455AcceptKey(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)
	c := newWSTestClient(t, s, "")

	headers := c.readHandshakeResponse()
	if !strings.Contains(headers["__status__"], "101") {
		t.Fatalf("expected a 101 Switching Protocols status line, got %q", headers["__status__"])
	}
	// The official RFC 6455 test vector: this exact key must produce this
	// exact accept value - independent of this package's own formula, so it
	// actually verifies computeAcceptKey rather than just echoing it back.
	const wantAccept = "s3pPLMBiTxaQ9kYGzzhZRbK+xOo="
	if headers["Sec-WebSocket-Accept"] != wantAccept {
		t.Fatalf("expected Sec-WebSocket-Accept=%q, got %q", wantAccept, headers["Sec-WebSocket-Accept"])
	}

	// The very first frame after a successful handshake is always the
	// {"id": ...} handshake ack (see Connection.Handle).
	msg := c.readJSON()
	if _, ok := msg["id"].(string); !ok {
		t.Fatalf("expected the handshake ack to carry an id, got %#v", msg)
	}
}

func TestConnection_OriginCheck_DeniedSendsCloseCode1002(t *testing.T) {
	s := newFakeServer(withAllowedOrigins("https://good.example"))
	t.Cleanup(s.close)
	c := newWSTestClient(t, s, "https://evil.example")

	c.readHandshakeResponse()
	code, _ := c.readCloseFrame()
	if code != CloseCodeAccessError {
		t.Fatalf("expected close code %d for a disallowed Origin, got %d", CloseCodeAccessError, code)
	}
}

func TestConnection_OriginCheck_AllowedProceedsToConnect(t *testing.T) {
	s := newFakeServer(withAllowedOrigins("https://good.example"))
	t.Cleanup(s.close)
	c := newWSTestClient(t, s, "https://good.example")

	c.readHandshakeResponse()
	// No close frame - straight to the handshake ack.
	msg := c.readJSON()
	if _, ok := msg["id"]; !ok {
		t.Fatalf("expected an allowed Origin to proceed to the normal handshake ack, got %#v", msg)
	}
}

func TestConnection_ConnectAction_EntersDefaultChannel(t *testing.T) {
	s := newFakeServer(withDefaultChannel("lobby", nil))
	t.Cleanup(s.close)
	if _, reason := s.Channels().CreateChannel(ws.NewChannelBuilder().SetKey("lobby")); reason != "" {
		t.Fatalf("setup: creating the default channel failed: %s", reason)
	}
	c := newWSTestClient(t, s, "")
	c.readHandshakeResponse()
	c.readJSON() // handshake ack

	c.sendJSON(map[string]any{"__lxws_action__": "connect"})
	resp := c.readJSON()
	if resp["__lxws_action__"] != "connect" {
		t.Fatalf("expected a connect action response, got %#v", resp)
	}
	ch := s.Channels().Get("lobby")
	if ch == nil || len(ch.MateIDs()) != 1 {
		t.Fatalf("expected the connecting client to be auto-entered into the default channel")
	}
}

func TestConnection_UnknownAction_SendsActionError(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)
	c := newWSTestClient(t, s, "")
	c.readHandshakeResponse()
	c.readJSON() // handshake ack

	c.sendJSON(map[string]any{"__lxws_action__": "not-a-real-action"})
	resp := c.readJSON()
	if resp["error"] == nil || resp["error"] == "" {
		t.Fatalf("expected a non-empty error for an unknown action, got %#v", resp)
	}
	if resp["__lxws_action__"] != "not-a-real-action" {
		t.Fatalf("expected the unknown action name echoed back, got %#v", resp)
	}
}

func TestConnection_RateLimit_Breaks(t *testing.T) {
	s := newFakeServer(withMaxRequestsPerMinute(1))
	t.Cleanup(s.close)
	c := newWSTestClient(t, s, "")
	c.readHandshakeResponse()
	c.readJSON() // handshake ack

	c.sendJSON(map[string]any{"__lxws_action__": "close"})
	c.readJSON() // ack of the 1st (within-limit) message

	c.sendJSON(map[string]any{"__lxws_action__": "close"})
	// The 2nd message within the same window exceeds MaxRequestsPerMinute=1:
	// Connection.Break sends a close-typed frame carrying a JSON error body
	// (not a spec close frame - see Break's doc-comment), then tears down.
	opcode, payload, _, err := hybi10Decode(c.reader)
	if err != nil {
		t.Fatalf("read break frame: %v", err)
	}
	if opcode != 0x8 {
		t.Fatalf("expected a close-typed frame for the rate-limit break, got opcode %#x", opcode)
	}
	var m map[string]any
	if err := json.Unmarshal(payload, &m); err != nil {
		t.Fatalf("unmarshal break payload %q: %v", payload, err)
	}
	if m["error"] != "rate_limit_exceeded" {
		t.Fatalf("expected error=rate_limit_exceeded, got %#v", m)
	}
}
