package src

import (
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/epicoon/lxgo/ws"
)

func TestConnRepo_AddHasGet(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)
	c1 := newFakeConnection(s, "c1")

	s.Connections().Add(c1)

	if !s.Connections().Has("c1") {
		t.Fatalf("expected Has(c1) true after Add")
	}
	if s.Connections().Get("c1") != c1 {
		t.Fatalf("expected Get(c1) to return the added connection")
	}
}

func TestConnRepo_RemoveImmediate(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)
	c1 := newFakeConnection(s, "c1")
	s.Connections().Add(c1)

	s.Connections().RemoveImmediate(c1)

	if s.Connections().Has("c1") {
		t.Fatalf("expected Has(c1) false after RemoveImmediate")
	}
	// No reconnection window at all - RemoveImmediate must not tombstone.
	if s.Connections().Reconnect(newFakeConnection(s, "c2"), "c1") {
		t.Fatalf("RemoveImmediate must not leave anything reconnectable")
	}
}

func TestConnRepo_MarkDisconnectedThenReconnect(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)
	old := newFakeConnection(s, "old")
	old.SetChannels(map[string]map[string]any{"ch1": {"nick": "bob"}})
	old.SetCreatedChannelsCount(3)
	s.Connections().Add(old)

	s.Connections().MarkDisconnected(old)
	if s.Connections().Has("old") {
		t.Fatalf("MarkDisconnected should drop the connection from the live set")
	}

	newConn := newFakeConnection(s, "new-temp-id")
	newConn.ip = old.ip
	ok := s.Connections().Reconnect(newConn, "old")
	if !ok {
		t.Fatalf("expected Reconnect to find the tombstoned connection")
	}
	if newConn.ID() != "old" {
		t.Fatalf("expected Reconnect to adopt the old ID, got %q", newConn.ID())
	}
	if newConn.Status() != ws.ConnStatusReconnecting {
		t.Fatalf("expected the reconnecting connection's status to be set")
	}
	// Regression: createdChannels used to not survive reconnect.
	if newConn.CreatedChannelsCount() != 3 {
		t.Fatalf("expected createdChannels to be restored across reconnect, got %d", newConn.CreatedChannelsCount())
	}
	if !s.Connections().Has("old") {
		t.Fatalf("expected the reconnected connection registered live under the restored ID")
	}
}

func TestConnRepo_Reconnect_WrongIPRejected(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)
	old := newFakeConnection(s, "old")
	old.ip = "1.1.1.1"
	s.Connections().Add(old)
	s.Connections().MarkDisconnected(old)

	newConn := newFakeConnection(s, "new")
	newConn.ip = "2.2.2.2"
	if s.Connections().Reconnect(newConn, "old") {
		t.Fatalf("Reconnect must reject a reconnection attempt from a different IP")
	}
}

func TestConnRepo_Reconnect_UnknownIDRejected(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)
	newConn := newFakeConnection(s, "new")
	if s.Connections().Reconnect(newConn, "never-existed") {
		t.Fatalf("Reconnect must reject an ID with no tombstone")
	}
}

func TestConnRepo_CheckIPLimit(t *testing.T) {
	s := newFakeServer(withMaxConnectionsPerIp(1))
	t.Cleanup(s.close)
	c1 := newFakeConnection(s, "c1")
	c1.ip = "1.2.3.4"
	c2 := newFakeConnection(s, "c2")
	c2.ip = "1.2.3.4"

	if !s.Connections().CheckIPLimit(c1) {
		t.Fatalf("first connection from an IP should be within the limit")
	}
	s.Connections().Add(c1)
	if s.Connections().CheckIPLimit(c2) {
		t.Fatalf("a second connection from the same IP should exceed the limit of 1")
	}
}

func TestConnRepo_CheckRequestLimit(t *testing.T) {
	s := newFakeServer(withMaxRequestsPerMinute(2))
	t.Cleanup(s.close)
	c1 := newFakeConnection(s, "c1")

	if !s.Connections().CheckRequestLimit(c1) {
		t.Fatalf("1st request should be within the limit")
	}
	if !s.Connections().CheckRequestLimit(c1) {
		t.Fatalf("2nd request should be within the limit")
	}
	if s.Connections().CheckRequestLimit(c1) {
		t.Fatalf("3rd request should exceed MaxRequestsPerMinute=2")
	}
}

func TestConnRepo_CheckRequestLimit_ZeroMeansNoLimit(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)
	c1 := newFakeConnection(s, "c1")
	for i := 0; i < 100; i++ {
		if !s.Connections().CheckRequestLimit(c1) {
			t.Fatalf("MaxRequestsPerMinute=0 (unset) should mean no limit at all")
		}
	}
}

func TestConnRepo_Sweeper_ExpiresTombstonePastReconnectionDuration(t *testing.T) {
	s := newFakeServer(withReconnectionDuration(50))
	t.Cleanup(s.close)
	old := newFakeConnection(s, "old")
	s.Connections().Add(old)
	s.Connections().MarkDisconnected(old)

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if !s.Connections().Reconnect(newFakeConnection(s, "probe"), "old") {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("expected the tombstone to expire and stop being reconnectable")
}

// TestConnRepo_Sweeper_ExpiredCreatorTombstoneClosesProprietaryChannel checks
// the specific concern raised in this task's "Требуется результат": if a
// proprietary channel's creator drops and never reconnects within
// ReconnectionDuration, the channel must actually get closed - not linger
// forever, since isAutoCloseDue() deliberately never selects proprietary
// channels for the ordinary empty-channel TTL sweep (see its doc-comment).
//
// The closing happens through a different path than that TTL sweep: when
// the creator's tombstone expires, ConnRepo's own sweeper calls
// Connection.Close() a second time (the first call, on the original drop,
// already flipped it to ConnStatusDisconnected without closing anything -
// see Channel.Leave's status check). That second Close() call flips the
// status to ConnStatusClosed *before* re-running LeaveAllChannels(), so
// Channel.Leave sees a creator that's gone for good this time and closes
// the channel via ChannelCloseCodeCreatorGone. This needs a real
// *Connection (not fakeConnection, whose Close() is an intentional no-op
// stub) to actually exercise that status-transition sequence.
func TestConnRepo_Sweeper_ExpiredCreatorTombstoneClosesProprietaryChannel(t *testing.T) {
	s := newFakeServer(withReconnectionDuration(50))
	t.Cleanup(s.close)

	clientConn, serverConn := net.Pipe()
	t.Cleanup(func() { clientConn.Close() })
	creator := NewConnection(s, serverConn)
	creator.SetID("creator")
	s.Connections().Add(creator)

	ch, reason := s.Channels().CreateChannel(ws.NewChannelBuilder().SetKey("k1").SetCreator(creator).SetProprietary(true))
	if ch == nil {
		t.Fatalf("setup: CreateChannel failed: %s", reason)
	}
	if ok, reason := creator.EnterChannel(ch, map[string]any{}); !ok {
		t.Fatalf("setup: creator EnterChannel failed: %s", reason)
	}

	// Simulate the creator's connection dropping, as Connection.Handle's
	// own "defer c.Close()" would on a real disconnect.
	creator.Close()
	if !s.Channels().Has("k1") {
		t.Fatalf("a merely-disconnected creator (still within the reconnection window) must not close the channel yet")
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if !s.Channels().Has("k1") {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("expected the proprietary channel to close once its creator's reconnection window expired without reconnecting")
}

// TestConnRepo_GetAll_ReturnsACopy is a regression test for a real,
// previously-fixed race: GetAll() used to return the repo's internal map
// directly - any concurrent Add/RemoveImmediate/MarkDisconnected mutating it
// while a caller ranged over the "snapshot" was a live "concurrent map
// iteration and map write" (see the ws-audit that fixed it). GetAll must
// return an independent copy.
func TestConnRepo_GetAll_ReturnsACopy(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)
	c1 := newFakeConnection(s, "c1")
	s.Connections().Add(c1)

	snapshot := s.Connections().GetAll()
	delete(snapshot, "c1")

	if !s.Connections().Has("c1") {
		t.Fatalf("mutating GetAll()'s result must not affect the repo's own state")
	}
}

// TestConnRepo_GetAll_ConcurrentWithAdd is a regression test for the same
// GetAll race as above, run under -race with actual concurrent mutation
// rather than just checking copy semantics after the fact.
func TestConnRepo_GetAll_ConcurrentWithAdd(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)

	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; ; i++ {
			select {
			case <-stop:
				return
			default:
			}
			c := newFakeConnection(s, "c")
			s.Connections().Add(c)
			s.Connections().RemoveImmediate(c)
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 200; i++ {
			for range s.Connections().GetAll() {
			}
		}
		close(stop)
	}()
	wg.Wait()
}

// TestConnRepo_Reconnect_ConcurrentReconnectsDontRace is a regression test
// for a real, previously-fixed bug: Reconnect used to take an RLock (a read
// lock) while writing to the conns map (assignment + delete) - two
// connections reconnecting at the same moment could both hold the RLock and
// both write concurrently. Run with -race to catch any reintroduction.
func TestConnRepo_Reconnect_ConcurrentReconnectsDontRace(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)

	const n = 30
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		old := newFakeConnection(s, fmt.Sprintf("old-%d", i))
		s.Connections().Add(old)
		s.Connections().MarkDisconnected(old)

		wg.Add(1)
		go func(oldID string) {
			defer wg.Done()
			newConn := newFakeConnection(s, "tmp")
			s.Connections().Reconnect(newConn, oldID)
		}(old.ID())
	}
	wg.Wait()
}
