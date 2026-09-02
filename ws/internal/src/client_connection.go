package src

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
)

// ClientConnection is an outbound WS connection - the mirror of Connection,
// which only ever accepts. It dials a remote WS server, performs the client
// half of the RFC 6455 handshake, and exchanges the same JSON-framed
// messages a Connection does - but carries none of Connection's
// many-connections bookkeeping (no ws.IWSServer, no ConnRepo, no
// reconnection window): there's exactly one remote peer, known in advance.
// It's Client's own underlying transport (see client.go) - not exposed as a
// public type itself, since nothing outside this package ever needs a raw
// send/receive handle independent of Client's request/response wrapping.
//
// The remote server's AllowedOrigins check (see Connection.checkOrigin) is
// a browser concept with no equivalent here - ClientConnection sends no
// Origin header at all, so a remote configured with a non-empty
// AllowedOrigins list will reject it. Point ClientConnection only at
// servers that leave AllowedOrigins unset for their server-to-server port.
type ClientConnection struct {
	mu     sync.RWMutex
	conn   net.Conn
	reader *bufio.Reader
	closed bool
}

/** @constructor */

// DialClientConnection opens addr ("host:port") as a plain TCP connection
// and performs the client half of the WS handshake against path (e.g.
// "/"), returning a connection ready for Send/Receive.
func DialClientConnection(addr, path string) (*ClientConnection, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", addr, err)
	}

	c, err := newClientConnection(conn, addr, path)
	if err != nil {
		conn.Close()
		return nil, err
	}
	return c, nil
}

// newClientConnection runs the client handshake over an already-open conn -
// split out from DialClientConnection so tests can drive it over a
// net.Pipe(), without a real TCP dial.
func newClientConnection(conn net.Conn, addr, path string) (*ClientConnection, error) {
	c := &ClientConnection{conn: conn}
	if err := c.handshake(addr, path); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *ClientConnection) Send(payload any, typ string) error {
	var data []byte
	switch v := payload.(type) {
	case nil:
		data = []byte{}
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("marshal payload: %w", err)
		}
		data = b
	}

	var opcode byte
	switch typ {
	case "text":
		opcode = 0x1
	case "binary":
		opcode = 0x2
	case "close":
		opcode = 0x8
	case "ping":
		opcode = 0x9
	case "pong":
		opcode = 0xA
	default:
		return fmt.Errorf("unsupported frame type: %s", typ)
	}

	// Client->server frames must be masked (RFC 6455 5.1) - unlike
	// Connection.Send, which always sends unmasked (server->client).
	frame, err := hybi10Encode(data, opcode, true)
	if err != nil {
		return err
	}

	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()
	if conn == nil {
		return fmt.Errorf("write error: connection already closed")
	}

	n, err := conn.Write(frame)
	if err != nil {
		return fmt.Errorf("write error: %w", err)
	}
	if n != len(frame) {
		return io.ErrShortWrite
	}
	return nil
}

func (c *ClientConnection) Receive() (any, error) {
	for {
		c.mu.RLock()
		reader := c.reader
		c.mu.RUnlock()
		if reader == nil {
			return nil, fmt.Errorf("read error: connection already closed")
		}

		opcode, payload, _, err := hybi10Decode(reader)
		if err != nil {
			return nil, fmt.Errorf("readFrame error: %w", err)
		}

		switch opcode {
		case 0x1: // text
			var msg any
			if err := json.Unmarshal(payload, &msg); err != nil {
				return nil, fmt.Errorf("invalid JSON payload: %w", err)
			}
			return msg, nil
		case 0x2: // binary - no consumer needs it yet
			continue
		case 0x8: // close
			return nil, io.EOF
		case 0x9: // ping -> pong
			if err := c.Send(payload, "pong"); err != nil {
				return nil, fmt.Errorf("pong send error: %w", err)
			}
		case 0xA: // pong
			continue
		default:
			// An unrecognized opcode doesn't end the connection, just this
			// one frame.
			continue
		}
	}
}

func (c *ClientConnection) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	conn := c.conn
	c.conn = nil
	c.reader = nil
	c.mu.Unlock()
	return conn.Close()
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * PRIVATE
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

// handshake performs the client half of the RFC 6455 handshake: send the
// upgrade request with a fresh Sec-WebSocket-Key, then read the response and
// check its Sec-WebSocket-Accept against what that key should produce (the
// same computeAcceptKey the server side uses to produce it).
func (c *ClientConnection) handshake(addr, path string) error {
	keyBytes := make([]byte, 16)
	if _, err := rand.Read(keyBytes); err != nil {
		return fmt.Errorf("generate Sec-WebSocket-Key: %w", err)
	}
	key := base64.StdEncoding.EncodeToString(keyBytes)

	if path == "" {
		path = "/"
	}
	req := "GET " + path + " HTTP/1.1\r\n" +
		"Host: " + addr + "\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Key: " + key + "\r\n" +
		"Sec-WebSocket-Version: 13\r\n" +
		"\r\n"
	if _, err := c.conn.Write([]byte(req)); err != nil {
		return fmt.Errorf("write handshake request: %w", err)
	}

	reader := bufio.NewReader(c.conn)
	statusLine, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("read status line: %w", err)
	}
	statusLine = strings.TrimRight(statusLine, "\r\n")
	if !strings.Contains(statusLine, " 101 ") {
		return fmt.Errorf("handshake rejected: %q", statusLine)
	}

	headers := make(map[string]string)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("read handshake headers: %w", err)
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		if i := strings.Index(line, ":"); i != -1 {
			headers[strings.ToLower(strings.TrimSpace(line[:i]))] = strings.TrimSpace(line[i+1:])
		}
	}

	want := computeAcceptKey(key)
	if got := headers["sec-websocket-accept"]; got != want {
		return fmt.Errorf("invalid Sec-WebSocket-Accept: got %q, want %q", got, want)
	}

	c.reader = reader
	return nil
}
