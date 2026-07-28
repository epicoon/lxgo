package src

import (
	"testing"
	"time"

	"github.com/epicoon/lxgo/ws"
)

func lastSent(c *fakeConnection) fakeSent {
	if len(c.sent) == 0 {
		return fakeSent{}
	}
	return c.sent[len(c.sent)-1]
}

func TestChannel_AddConnection_BroadcastsMateEntered(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)
	ch := NewChannel(s, "ch1", nil, false, false, "")
	c1 := newFakeConnection(s, "c1")
	c2 := newFakeConnection(s, "c2")
	s.Connections().Add(c1)
	s.Connections().Add(c2)

	ch.AddConnection(c1)
	ch.AddConnection(c2)

	// c1 should have gotten exactly one broadcast, about c2 joining.
	if len(c1.sent) != 1 {
		t.Fatalf("expected exactly one broadcast to c1, got %d: %#v", len(c1.sent), c1.sent)
	}
	payload := lastSent(c1).payload.(map[string]any)
	if payload["__lxws_channel__"] != "mateEntered" || payload["id"] != "c2" {
		t.Fatalf("unexpected broadcast to c1: %#v", payload)
	}
	// c2 (the one joining) doesn't get its own join event.
	if len(c2.sent) != 0 {
		t.Fatalf("the joining connection shouldn't get its own mateEntered, got %#v", c2.sent)
	}
}

func TestChannel_AddConnection_ReconnectingBroadcastsMateReconnected(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)
	ch := NewChannel(s, "ch1", nil, false, false, "")
	c1 := newFakeConnection(s, "c1")
	c2 := newFakeConnection(s, "c2")
	s.Connections().Add(c1)
	s.Connections().Add(c2)
	ch.AddConnection(c1)

	c2.SetStatus(ws.ConnStatusReconnecting)
	ch.AddConnection(c2)

	payload := lastSent(c1).payload.(map[string]any)
	if payload["__lxws_channel__"] != "mateReconnected" {
		t.Fatalf("expected mateReconnected for a reconnecting joiner, got %#v", payload)
	}
}

func TestChannel_Enter_AlreadyMemberIsNoOp(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)
	ch := NewChannel(s, "ch1", nil, false, false, "")
	c1 := newFakeConnection(s, "c1")
	s.Connections().Add(c1)
	ch.AddConnection(c1)

	ok, reason := ch.Enter(c1, map[string]any{})
	if !ok || reason != "" {
		t.Fatalf("re-entering as an existing member should succeed with no reason, got ok=%v reason=%q", ok, reason)
	}
}

func TestChannel_Enter_AuthHandlerDenies(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)
	ch := NewChannel(s, "ch1", nil, false, false, "")
	ch.SetAuthHandler(func(conn ws.IConnection, message map[string]any) (bool, string) {
		return false, "nope"
	})
	c1 := newFakeConnection(s, "c1")
	s.Connections().Add(c1)

	ok, reason := ch.Enter(c1, map[string]any{})
	if ok || reason != "nope" {
		t.Fatalf("expected denial with the handler's reason, got ok=%v reason=%q", ok, reason)
	}
	if ch.Has(c1) {
		t.Fatalf("a denied connection must not become a member")
	}
}

func TestChannel_Enter_AuthHandlerDeniesWithEmptyReason(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)
	ch := NewChannel(s, "ch1", nil, false, false, "")
	ch.SetAuthHandler(func(conn ws.IConnection, message map[string]any) (bool, string) {
		return false, ""
	})
	c1 := newFakeConnection(s, "c1")
	s.Connections().Add(c1)

	_, reason := ch.Enter(c1, map[string]any{})
	if reason != "access denied" {
		t.Fatalf("expected the default reason 'access denied', got %q", reason)
	}
}

func TestChannel_Enter_AuthHandlerAllows(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)
	ch := NewChannel(s, "ch1", nil, false, false, "")
	called := false
	ch.SetAuthHandler(func(conn ws.IConnection, message map[string]any) (bool, string) {
		called = true
		return true, ""
	})
	c1 := newFakeConnection(s, "c1")
	s.Connections().Add(c1)

	ok, _ := ch.Enter(c1, map[string]any{})
	if !ok || !called || !ch.Has(c1) {
		t.Fatalf("expected entry to succeed and the connection to become a member")
	}
}

func TestChannel_Leave_BroadcastsMateLeft(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)
	ch := NewChannel(s, "ch1", nil, false, false, "")
	c1 := newFakeConnection(s, "c1")
	c2 := newFakeConnection(s, "c2")
	s.Connections().Add(c1)
	s.Connections().Add(c2)
	ch.AddConnection(c1)
	ch.AddConnection(c2)

	ch.Leave(c2)

	if ch.Has(c2) {
		t.Fatalf("c2 should no longer be a member after Leave")
	}
	payload := lastSent(c1).payload.(map[string]any)
	if payload["__lxws_channel__"] != "mateLeft" || payload["id"] != "c2" {
		t.Fatalf("expected mateLeft broadcast to remaining members, got %#v", payload)
	}
}

func TestChannel_Leave_DisconnectedBroadcastsMateDisconnected(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)
	ch := NewChannel(s, "ch1", nil, false, false, "")
	c1 := newFakeConnection(s, "c1")
	c2 := newFakeConnection(s, "c2")
	s.Connections().Add(c1)
	s.Connections().Add(c2)
	ch.AddConnection(c1)
	ch.AddConnection(c2)

	c2.SetStatus(ws.ConnStatusDisconnected)
	ch.Leave(c2)

	payload := lastSent(c1).payload.(map[string]any)
	if payload["__lxws_channel__"] != "mateDisconnected" {
		t.Fatalf("expected mateDisconnected for a disconnected leaver, got %#v", payload)
	}
}

func TestChannel_Leave_ProprietaryCreatorGoneClosesChannel(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)
	creator := newFakeConnection(s, "creator")
	mate := newFakeConnection(s, "mate")
	s.Connections().Add(creator)
	s.Connections().Add(mate)
	creator.IncrementCreatedChannels()

	ch := NewChannel(s, "ch1", nil, false, true, "creator")
	ch.AddConnection(creator)
	ch.AddConnection(mate)

	ch.Leave(creator)

	if ch.Has(mate) {
		t.Fatalf("Close should have kicked every remaining member, including mate")
	}
	if len(mate.sent) == 0 {
		t.Fatalf("expected the remaining mate to get a 'closed' notification")
	}
	payload := lastSent(mate).payload.(map[string]any)
	if payload["__lxws_channel__"] != "closed" || payload["code"] != ws.ChannelCloseCodeCreatorGone {
		t.Fatalf("expected a ChannelCloseCodeCreatorGone close notification, got %#v", payload)
	}
	if creator.CreatedChannelsCount() != 0 {
		t.Fatalf("expected the creator's created-channels quota to be credited back, got %d", creator.CreatedChannelsCount())
	}
}

func TestChannel_Leave_ProprietaryCreatorMerelyDisconnectingDoesNotClose(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)
	creator := newFakeConnection(s, "creator")
	mate := newFakeConnection(s, "mate")
	s.Connections().Add(creator)
	s.Connections().Add(mate)

	ch := NewChannel(s, "ch1", nil, false, true, "creator")
	ch.AddConnection(creator)
	ch.AddConnection(mate)

	creator.SetStatus(ws.ConnStatusDisconnected)
	ch.Leave(creator)

	if ch.Has(creator) {
		t.Fatalf("creator should no longer be a member")
	}
	// The channel itself must still be open - a temporary disconnect isn't
	// "gone for good" (see Leave's doc-comment).
	payload := lastSent(mate).payload.(map[string]any)
	if payload["__lxws_channel__"] != "mateDisconnected" {
		t.Fatalf("expected a plain mateDisconnected, not a channel close, got %#v", payload)
	}
}

// TestChannel_Close_CleansUpKickedMembersChannelsMap is a regression test for
// a real, previously-fixed bug: closing a channel used to leave kicked
// members' own Connection.channels map still pointing at the closed
// channel's key, causing a panic later (e.g. on their own disconnect trying
// to Leave a channel that no longer exists). Close must scrub the channel
// out of every remaining member's own Channels() map, not just its own
// membership set.
func TestChannel_Close_CleansUpKickedMembersChannelsMap(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)
	c1 := newFakeConnection(s, "c1")
	s.Connections().Add(c1)

	ch := NewChannel(s, "ch1", nil, false, false, "")
	ch.AddConnection(c1)
	c1.channels[ch.Key()] = map[string]any{}

	ch.Close(ws.ChannelCloseCodeServer)

	if _, exists := c1.Channels()[ch.Key()]; exists {
		t.Fatalf("Close must remove the channel from a kicked member's own Channels() map, still present: %#v", c1.Channels())
	}
}

func TestChannel_Close_CreditsBackCreatorQuota(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)
	creator := newFakeConnection(s, "creator")
	s.Connections().Add(creator)
	creator.IncrementCreatedChannels()
	creator.IncrementCreatedChannels()

	ch := NewChannel(s, "ch1", nil, false, false, "creator")
	ch.Close(ws.ChannelCloseCodeServer)

	if creator.CreatedChannelsCount() != 1 {
		t.Fatalf("expected exactly one created-channel credited back, got %d", creator.CreatedChannelsCount())
	}
}

func TestChannel_Close_RemovesFromRepo(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)
	builder := ws.NewChannelBuilder().SetKey("ch1")
	ch, reason := s.Channels().CreateChannel(builder)
	if ch == nil {
		t.Fatalf("setup: CreateChannel failed: %s", reason)
	}

	ch.Close(ws.ChannelCloseCodeServer)

	if s.Channels().Has("ch1") {
		t.Fatalf("Close should remove the channel from the repo")
	}
}

func TestChannel_Close_IsNoOpIfAlreadyClosed(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)
	c1 := newFakeConnection(s, "c1")
	s.Connections().Add(c1)

	ch := NewChannel(s, "ch1", nil, false, false, "")
	ch.AddConnection(c1)

	ch.Close(ws.ChannelCloseCodeServer)
	sentAfterFirstClose := len(c1.sent)

	// A second Close must not re-notify anyone or panic.
	ch.Close(ws.ChannelCloseCodeServer)
	if len(c1.sent) != sentAfterFirstClose {
		t.Fatalf("a second Close() should be a no-op, but sent %d more messages", len(c1.sent)-sentAfterFirstClose)
	}
}

func TestChannel_IsAutoCloseDue(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)
	ch := NewChannel(s, "ch1", nil, false, false, "").(*Channel)

	if ch.isAutoCloseDue(time.Millisecond) {
		t.Fatalf("a brand-new empty channel shouldn't be due yet (emptySince just set to now)")
	}
	time.Sleep(5 * time.Millisecond)
	if !ch.isAutoCloseDue(time.Millisecond) {
		t.Fatalf("expected the channel to be due for auto-close after the TTL elapsed")
	}
}

func TestChannel_IsAutoCloseDue_ProprietaryNeverDue(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)
	ch := NewChannel(s, "ch1", nil, false, true, "creator").(*Channel)
	time.Sleep(5 * time.Millisecond)

	if ch.isAutoCloseDue(time.Millisecond) {
		t.Fatalf("a proprietary channel must never be swept, regardless of how long it's been empty")
	}
}

func TestChannel_IsAutoCloseDue_NonEmptyNeverDue(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)
	c1 := newFakeConnection(s, "c1")
	s.Connections().Add(c1)
	ch := NewChannel(s, "ch1", nil, false, false, "")
	ch.AddConnection(c1)
	concrete := ch.(*Channel)
	time.Sleep(5 * time.Millisecond)

	if concrete.isAutoCloseDue(time.Millisecond) {
		t.Fatalf("a channel with a member present must never be due for the empty-channel sweep")
	}
}
