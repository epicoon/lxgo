package src

import (
	"testing"
	"time"

	"github.com/epicoon/lxgo/ws"
)

// wsTestClient additions used only by these integration-style tests - kept
// here rather than connection_test.go since they're specific to multi-client
// scenarios.

func (c *wsTestClient) readHandshakeID() string {
	c.t.Helper()
	msg := c.readJSON()
	id, _ := msg["id"].(string)
	if id == "" {
		c.t.Fatalf("expected the handshake ack to carry a non-empty id, got %#v", msg)
	}
	return id
}

// createChannel drives a createChannel action to completion and reports
// whether the server accepted it.
func (c *wsTestClient) createChannel(public, proprietary bool) (key string, ok bool, reason string) {
	c.t.Helper()
	c.sendJSON(map[string]any{
		"__lxws_action__": "createChannel",
		"public":          public,
		"proprietary":     proprietary,
	})
	resp := c.readJSON()
	if errMsg, hasErr := resp["error"]; hasErr {
		reason, _ = errMsg.(string)
		return "", false, reason
	}
	channel, _ := resp["channel"].(map[string]any)
	key, _ = channel["key"].(string)
	return key, true, ""
}

func (c *wsTestClient) enterChannel(key string) bool {
	c.t.Helper()
	c.sendJSON(map[string]any{
		"__lxws_action__": "enterChannel",
		"channelKey":      key,
	})
	resp := c.readJSON()
	_, hasErr := resp["error"]
	return !hasErr
}

// TestIntegration_Reconnect_PreservesCreatedChannelsLimit is a regression
// test for the historical bug "createdChannels didn't survive reconnect"
// (see this task's description) - driven through real Connection.Handle()
// instances over net.Pipe (real handshake/framing/action-dispatch), not
// through ConnRepo's Go API directly like the lower-level unit test does.
func TestIntegration_Reconnect_PreservesCreatedChannelsLimit(t *testing.T) {
	s := newFakeServer(withMaxChannelsPerConnection(1))
	t.Cleanup(s.close)

	a := newWSTestClient(t, s, "")
	a.readHandshakeResponse()
	oldID := a.readHandshakeID()

	a.sendJSON(map[string]any{"__lxws_action__": "connect"})
	a.readJSON() // connect ack

	if _, ok, reason := a.createChannel(false, false); !ok {
		t.Fatalf("first createChannel should succeed within the limit: %s", reason)
	}
	if _, ok, _ := a.createChannel(false, false); ok {
		t.Fatalf("a second createChannel should be denied - MaxChannelsPerConnection=1 already reached")
	}

	// Simulate the connection dropping abruptly (network loss, not a
	// graceful "close" action) - closing the pipe makes a's Handle() read
	// loop error out and run its own deferred Close(), same as a real
	// disconnect would. That runs in a's own Handle() goroutine, so wait for
	// it to actually finish (MarkDisconnected removing the live entry)
	// before b tries to reconnect - otherwise b's reconnect could race
	// ahead of the tombstone even existing yet.
	a.client.Close()
	deadline := time.Now().Add(2 * time.Second)
	for s.Connections().Has(oldID) && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if s.Connections().Has(oldID) {
		t.Fatalf("setup: a's connection never finished disconnecting")
	}

	b := newWSTestClient(t, s, "")
	b.readHandshakeResponse()
	b.readHandshakeID()
	b.sendJSON(map[string]any{
		"__lxws_action__": "reconnect",
		"oldConnectionId": oldID,
	})
	resp := b.readJSON()
	if resp["__lxws_action__"] != "reconnect" || resp["idRestored"] != oldID {
		t.Fatalf("expected a successful reconnect restoring %q, got %#v", oldID, resp)
	}

	if _, ok, _ := b.createChannel(false, false); ok {
		t.Fatalf("MaxChannelsPerConnection must still be enforced after reconnect - the limit must have survived")
	}
}

// TestIntegration_KickedMember_DisconnectDoesNotPanic is a regression test
// for the historical bug "panic in forceClose() when kicking a member" -
// closing a channel out from under a member used to leave that member's own
// Connection.channels map still pointing at the now-gone channel key, which
// panicked later when that member's own Close()/LeaveAllChannels tried to
// look the channel back up. Driven through two real Connection instances so
// the actual disconnect code path (not a hand-built fake) is what's
// exercised.
func TestIntegration_KickedMember_DisconnectDoesNotPanic(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)

	creator := newWSTestClient(t, s, "")
	creator.readHandshakeResponse()
	creator.readHandshakeID()
	creator.sendJSON(map[string]any{"__lxws_action__": "connect"})
	creator.readJSON()

	key, ok, reason := creator.createChannel(true, false)
	if !ok {
		t.Fatalf("setup: createChannel failed: %s", reason)
	}

	member := newWSTestClient(t, s, "")
	member.readHandshakeResponse()
	member.readHandshakeID()
	member.sendJSON(map[string]any{"__lxws_action__": "connect"})
	member.readJSON()

	// AddConnection broadcasts "mateEntered" to the creator synchronously,
	// from within member's own Handle() goroutine, before that goroutine
	// goes on to send member's own enterChannel ack - and net.Pipe is
	// unbuffered, so that broadcast write blocks until something reads it.
	// Drain it concurrently rather than after member.enterChannel returns,
	// or the two goroutines deadlock on each other's pipe.
	creatorMsg := make(chan map[string]any, 1)
	go func() { creatorMsg <- creator.readJSON() }()

	if !member.enterChannel(key) {
		t.Fatalf("setup: member could not enter the channel")
	}

	select {
	case msg := <-creatorMsg:
		if msg["__lxws_channel__"] != "mateEntered" {
			t.Fatalf("expected the creator to get a mateEntered broadcast, got %#v", msg)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for the mateEntered broadcast to the creator")
	}

	ch := s.Channels().Get(key)
	if ch == nil {
		t.Fatalf("setup: channel %q not found in the repo", key)
	}

	// Close() itself (called directly here, from the test's own goroutine)
	// sends a "closed" notification to every remaining member in turn - both
	// creator and member here - and each of those Send calls blocks on the
	// same unbuffered net.Pipe until read. Prime both readers first so
	// Close() doesn't stall on whichever it happens to reach first.
	creatorClosed := make(chan map[string]any, 1)
	memberClosed := make(chan map[string]any, 1)
	go func() { creatorClosed <- creator.readJSON() }()
	go func() { memberClosed <- member.readJSON() }()

	ch.Close(ws.ChannelCloseCodeServer)

	var closedMsg map[string]any
	select {
	case closedMsg = <-memberClosed:
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for the kicked member's 'closed' notification")
	}
	select {
	case <-creatorClosed:
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for the creator's own 'closed' notification")
	}
	if closedMsg["__lxws_channel__"] != "closed" {
		t.Fatalf("expected the kicked member to get a 'closed' notification, got %#v", closedMsg)
	}

	// Call Close() directly on the server-side connection, synchronously in
	// this goroutine - member.client.Close() would only trigger it
	// asynchronously inside Handle()'s own background goroutine, where a
	// recover() here couldn't catch anything.
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("kicked member's own disconnect must not panic, got: %v", r)
			}
		}()
		member.conn.Close()
	}()
}
