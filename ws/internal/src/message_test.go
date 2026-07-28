package src

import (
	"reflect"
	"sort"
	"testing"
)

func TestMessage_AddData(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)

	m := NewMessage(s)
	m.SetData(map[string]any{"a": 1})
	m.AddData(map[string]any{"b": 2})

	got, ok := m.Data().(map[string]any)
	if !ok {
		t.Fatalf("Data() is not a map: %#v", m.Data())
	}
	if got["a"] != 1 || got["b"] != 2 {
		t.Fatalf("AddData didn't merge into the existing map: %#v", got)
	}
}

func TestMessage_AddData_NonMapNestsUnderDataKey(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)

	m := NewMessage(s)
	m.SetData("plain string")
	m.AddData(map[string]any{"b": 2})

	got, ok := m.Data().(map[string]any)
	if !ok {
		t.Fatalf("Data() is not a map: %#v", m.Data())
	}
	if got["b"] != 2 || got["__data__"] != "plain string" {
		t.Fatalf("AddData didn't nest the non-map original data: %#v", got)
	}
}

func TestMessage_AddDataForConnection(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)
	conn := newFakeConnection(s, "c1")

	m := NewMessage(s)
	m.SetDataForConnection(conn, map[string]any{"a": 1})
	m.AddDataForConnection(conn, map[string]any{"b": 2})

	got := m.PrepareDataForConnection("c1").(map[string]any)["__lxws_message__"].(map[string]any)
	if got["a"] != 1 || got["b"] != 2 {
		t.Fatalf("AddDataForConnection didn't merge: %#v", got)
	}
}

func TestMessage_AddDataForConnection_NonMapNestsUnderDataKey(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)
	conn := newFakeConnection(s, "c1")

	m := NewMessage(s)
	m.SetDataForConnection(conn, "plain")
	m.AddDataForConnection(conn, map[string]any{"b": 2})

	got := m.PrepareDataForConnection("c1").(map[string]any)["__lxws_message__"].(map[string]any)
	if got["b"] != 2 || got["__data__"] != "plain" {
		t.Fatalf("AddDataForConnection didn't nest the non-map original: %#v", got)
	}
}

func TestMessage_ReceiverIDs_DefaultsToEveryConnection(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)
	c1 := newFakeConnection(s, "c1")
	c2 := newFakeConnection(s, "c2")
	s.Connections().Add(c1)
	s.Connections().Add(c2)

	m := NewMessage(s)
	ids := m.ReceiverIDs()
	sort.Strings(ids)
	if !reflect.DeepEqual(ids, []string{"c1", "c2"}) {
		t.Fatalf("expected every connected id with no explicit receivers, got %v", ids)
	}
}

func TestMessage_ReceiverIDs_ExplicitReceivers(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)
	c1 := newFakeConnection(s, "c1")
	c2 := newFakeConnection(s, "c2")
	s.Connections().Add(c1)
	s.Connections().Add(c2)

	m := NewMessage(s)
	m.SetReceiverIds([]string{"c1"})
	if got := m.ReceiverIDs(); !reflect.DeepEqual(got, []string{"c1"}) {
		t.Fatalf("expected only the explicit receiver, got %v", got)
	}
}

func TestMessage_ValidateConnectionID(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)
	c1 := newFakeConnection(s, "c1")
	s.Connections().Add(c1)

	m := NewMessage(s)
	if !m.ValidateConnectionID("c1") {
		t.Fatalf("c1 is connected and not excepted - should validate")
	}
	if m.ValidateConnectionID("unknown") {
		t.Fatalf("unknown connection id should not validate")
	}

	m.ExceptReceiver(c1)
	if m.ValidateConnectionID("c1") {
		t.Fatalf("c1 was excepted - should no longer validate")
	}
}

func TestMessage_PrepareDataForConnection_CommonDataOnly(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)

	m := NewMessage(s)
	m.SetData(map[string]any{"a": 1})

	got := m.PrepareDataForConnection("whoever").(map[string]any)
	if !reflect.DeepEqual(got["__lxws_message__"], map[string]any{"a": 1}) {
		t.Fatalf("expected the common data unchanged, got %#v", got)
	}
}

func TestMessage_PrepareDataForConnection_MergesCommonAndPerConnection(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)
	conn := newFakeConnection(s, "c1")

	m := NewMessage(s)
	m.SetData(map[string]any{"a": 1})
	m.SetDataForConnection(conn, map[string]any{"b": 2})

	got := m.PrepareDataForConnection("c1").(map[string]any)["__lxws_message__"].(map[string]any)
	if got["a"] != 1 || got["b"] != 2 {
		t.Fatalf("expected common+per-connection data merged, got %#v", got)
	}

	// A connection with no override still gets just the common data.
	other := m.PrepareDataForConnection("c2").(map[string]any)["__lxws_message__"]
	if !reflect.DeepEqual(other, map[string]any{"a": 1}) {
		t.Fatalf("expected plain common data for a connection with no override, got %#v", other)
	}
}
