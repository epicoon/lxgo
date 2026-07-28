package src

import (
	"reflect"
	"sort"
	"testing"
)

func TestChannelMessage_ReceiverIDs_DefaultsToWholeChannel(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)
	ch := NewChannel(s, "ch1", nil, false, false, "")
	c1 := newFakeConnection(s, "c1")
	c2 := newFakeConnection(s, "c2")
	s.Connections().Add(c1)
	s.Connections().Add(c2)
	ch.AddConnection(c1)
	ch.AddConnection(c2)

	msg := NewChannelMessage(ch)
	ids := msg.ReceiverIDs()
	sort.Strings(ids)
	if !reflect.DeepEqual(ids, []string{"c1", "c2"}) {
		t.Fatalf("expected every channel mate with no explicit receivers, got %v", ids)
	}
}

func TestChannelMessage_ReceiverIDs_ExplicitReceiversFilteredByMembership(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)
	ch := NewChannel(s, "ch1", nil, false, false, "")
	c1 := newFakeConnection(s, "c1")
	c2 := newFakeConnection(s, "c2")
	s.Connections().Add(c1)
	s.Connections().Add(c2)
	ch.AddConnection(c1)
	// c2 is deliberately NOT a member of ch.

	msg := NewChannelMessage(ch)
	msg.SetReceiverIds([]string{"c1", "c2"})
	got := msg.ReceiverIDs()
	if !reflect.DeepEqual(got, []string{"c1"}) {
		t.Fatalf("expected only the actual channel mate among explicit receivers, got %v", got)
	}
}

func TestChannelMessage_ReturnToSender(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)
	ch := NewChannel(s, "ch1", nil, false, false, "")
	sender := newFakeConnection(s, "sender")
	other := newFakeConnection(s, "other")
	s.Connections().Add(sender)
	s.Connections().Add(other)
	ch.AddConnection(sender)
	ch.AddConnection(other)

	msg := NewChannelMessage(ch)
	msg.SetSender("sender")
	msg.SetReceiverIds([]string{"other"})
	msg.ReturnToSender(true)

	ids := msg.ReceiverIDs()
	sort.Strings(ids)
	if !reflect.DeepEqual(ids, []string{"other", "sender"}) {
		t.Fatalf("ReturnToSender(true) should add the sender to explicit receivers, got %v", ids)
	}
}

func TestChannelMessage_ReturnToSender_NoSenderIsNoOp(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)
	ch := NewChannel(s, "ch1", nil, false, false, "")

	msg := NewChannelMessage(ch)
	// SetSender was never called ("" sender) - ReturnToSender must not panic
	// or otherwise mark anything.
	msg.ReturnToSender(false)
	msg.ReturnToSender(true)
}

func TestChannelMessage_ReturnToSender_FalseExceptsSenderFromBroadcast(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)
	ch := NewChannel(s, "ch1", nil, false, false, "")
	sender := newFakeConnection(s, "sender")
	other := newFakeConnection(s, "other")
	s.Connections().Add(sender)
	s.Connections().Add(other)
	ch.AddConnection(sender)
	ch.AddConnection(other)

	msg := NewChannelMessage(ch)
	msg.SetSender("sender")
	msg.ReturnToSender(false)

	if msg.ValidateConnectionID("sender") {
		t.Fatalf("ReturnToSender(false) should except the sender from delivery")
	}
	if !msg.ValidateConnectionID("other") {
		t.Fatalf("other mate should still validate")
	}
}

func TestChannelMessage_PrepareDataForConnection(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)
	ch := NewChannel(s, "ch1", nil, false, false, "")

	msg := NewChannelMessage(ch)
	msg.SetSender("sender")
	msg.SetReceiverIds([]string{"mate"})
	msg.SetPrivate(true)
	msg.SetData(map[string]any{"a": 1})

	got := msg.PrepareDataForConnection("mate").(map[string]any)
	if got["__lxws_channel__"] != "message" {
		t.Fatalf("expected __lxws_channel__=message, got %#v", got["__lxws_channel__"])
	}
	if got["channel"] != "ch1" || got["from"] != "sender" || got["private"] != true {
		t.Fatalf("unexpected envelope: %#v", got)
	}
	if got["addressed"] != true {
		t.Fatalf("expected addressed=true for an explicit receiver, got %#v", got["addressed"])
	}
	if !reflect.DeepEqual(got["data"], map[string]any{"a": 1}) {
		t.Fatalf("expected data to carry the payload (renamed from __lxws_message__), got %#v", got["data"])
	}
	if _, exists := got["__lxws_message__"]; exists {
		t.Fatalf("__lxws_message__ should have been renamed to data")
	}

	notAddressed := msg.PrepareDataForConnection("someoneElse").(map[string]any)
	if notAddressed["addressed"] != false {
		t.Fatalf("expected addressed=false for a non-explicit receiver, got %#v", notAddressed["addressed"])
	}
}
