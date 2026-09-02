package src

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/epicoon/lxgo/ws"
)

// Client is the sole outbound WS connection type - it dials a remote
// server (via ClientConnection) and additionally speaks the
// __lxws_request__/__lxws_response__ wire protocol a server's
// IRouter.Handle answers, the same protocol the browser client's request()
// speaks. It owns the connection's Receive() loop from the moment it's
// created, multiplexing it between whichever Request call is currently
// waiting on a response (correlated by a random key, since several can be
// in flight on the same connection at once) and anything else the remote
// peer sends unprompted.
type Client struct {
	conn *ClientConnection

	mu      sync.Mutex
	pending map[string]chan responseFrame
	closed  bool
}

var _ ws.IClient = (*Client)(nil)

type responseFrame struct {
	resp ws.Response
	err  error
}

// NewClient wraps conn - typically freshly dialed and not yet read from -
// starting its read loop in the background. onPush is called for every
// received message that isn't a response to a pending Request (e.g. a
// notice the remote peer sends on its own initiative). onDropped is called
// once if the read loop ends because of a read error rather than an
// explicit Close.
func NewClient(conn *ClientConnection, onPush func(msg any), onDropped func()) *Client {
	c := &Client{
		conn:    conn,
		pending: make(map[string]chan responseFrame),
	}
	go c.readLoop(onPush, onDropped)
	return c
}

func (c *Client) Send(payload any, typ string) error {
	return c.conn.Send(payload, typ)
}

func (c *Client) Request(route string, params map[string]any, timeout time.Duration) (ws.Response, error) {
	if params == nil {
		params = map[string]any{}
	}

	key := randRequestKey()
	ch := make(chan responseFrame, 1)

	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return ws.Response{}, fmt.Errorf("request %s: connection already closed", route)
	}
	c.pending[key] = ch
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		delete(c.pending, key)
		c.mu.Unlock()
	}()

	msg := map[string]any{
		"__lxws_request__": map[string]any{"route": route, "key": key},
		"__data__":         params,
	}
	if err := c.conn.Send(msg, "text"); err != nil {
		return ws.Response{}, fmt.Errorf("request %s: %w", route, err)
	}

	select {
	case frame := <-ch:
		return frame.resp, frame.err
	case <-time.After(timeout):
		return ws.Response{}, fmt.Errorf("request %s: timed out", route)
	}
}

func (c *Client) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	c.mu.Unlock()
	return c.conn.Close()
}

// readLoop is the connection's single reader - every response and every
// unsolicited push both flow through here, dispatched by shape.
func (c *Client) readLoop(onPush func(msg any), onDropped func()) {
	for {
		msg, err := c.conn.Receive()
		if err != nil {
			c.mu.Lock()
			closed := c.closed
			c.mu.Unlock()
			if !closed && onDropped != nil {
				onDropped()
			}
			return
		}

		m, ok := msg.(map[string]any)
		if !ok {
			if onPush != nil {
				onPush(msg)
			}
			continue
		}
		if isResp, _ := m["__lxws_response__"].(bool); isResp {
			c.handleResponse(m)
			continue
		}
		if onPush != nil {
			onPush(msg)
		}
	}
}

func (c *Client) handleResponse(m map[string]any) {
	key, _ := m["key"].(string)
	c.mu.Lock()
	ch, ok := c.pending[key]
	c.mu.Unlock()
	if !ok {
		return
	}

	resp := ws.Response{
		Code: intRequestField(m, "code"),
	}
	if headers, ok := m["headers"].(map[string]any); ok {
		resp.Headers = headers
	}
	resp.Body, _ = decodeResponseBody(m["body"])
	ch <- responseFrame{resp: resp}
}

// decodeResponseBody parses a response's body - IHttpResponse.Data() (what
// a server's Router.Handle actually carries back) is a JSON-ENCODED STRING,
// not the raw decoded value, so this unmarshals it rather than returning it
// as-is, mirroring JSON.parse(msg.body) on the browser side.
func decodeResponseBody(body any) (any, error) {
	s, ok := body.(string)
	if !ok {
		return body, nil
	}
	if s == "" || s == "null" {
		return nil, nil
	}
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return nil, fmt.Errorf("decode response body: %w", err)
	}
	return v, nil
}

func randRequestKey() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func intRequestField(m map[string]any, key string) int {
	switch v := m[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	default:
		return 0
	}
}
