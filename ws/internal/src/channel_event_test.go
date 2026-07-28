package src

import "testing"

func TestChannelEvent_NameAndInitiator(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)
	ch := NewChannel(s, "ch1", nil, false, false, "")
	initiator := newFakeConnection(s, "c1")
	s.Connections().Add(initiator)

	event := NewChannelEvent("ping", ch, initiator)
	if event.Name() != "ping" {
		t.Fatalf("expected Name()=ping, got %q", event.Name())
	}
	if event.Initiator().ID() != "c1" {
		t.Fatalf("expected Initiator() to resolve back to c1, got %v", event.Initiator())
	}
}

func TestChannelEvent_StopIsStopped(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)
	ch := NewChannel(s, "ch1", nil, false, false, "")
	initiator := newFakeConnection(s, "c1")
	s.Connections().Add(initiator)

	event := NewChannelEvent("ping", ch, initiator)
	if event.IsStopped() {
		t.Fatalf("a fresh event should not be stopped")
	}
	event.Stop()
	if !event.IsStopped() {
		t.Fatalf("expected IsStopped() true after Stop()")
	}
}

func TestChannelEvent_PrepareDataForConnection(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)
	ch := NewChannel(s, "ch1", nil, false, false, "")
	initiator := newFakeConnection(s, "c1")
	s.Connections().Add(initiator)

	event := NewChannelEvent("ping", ch, initiator)
	event.SetData(map[string]any{"x": 1})

	got := event.PrepareDataForConnection("mate").(map[string]any)
	if got["__lxws_channel__"] != "event" {
		t.Fatalf("expected __lxws_channel__=event, got %#v", got["__lxws_channel__"])
	}
	if got["event"] != "ping" {
		t.Fatalf("expected event=ping, got %#v", got["event"])
	}
	if got["from"] != "c1" {
		t.Fatalf("expected from=c1 (the initiator), got %#v", got["from"])
	}
}
