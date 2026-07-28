package src

import (
	"sync"
	"testing"
	"time"

	"github.com/epicoon/lxgo/ws"
)

func TestChannelRepo_CreateChannel_GeneratesKeyWhenEmpty(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)

	ch, reason := s.Channels().CreateChannel(ws.NewChannelBuilder())
	if ch == nil {
		t.Fatalf("expected a channel, got failure: %s", reason)
	}
	if ch.Key() == "" {
		t.Fatalf("expected a generated key, got empty string")
	}
	if !s.Channels().Has(ch.Key()) {
		t.Fatalf("the created channel should be registered in the repo")
	}
}

func TestChannelRepo_CreateChannel_RejectsDuplicateKey(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)

	if ch, reason := s.Channels().CreateChannel(ws.NewChannelBuilder().SetKey("dup")); ch == nil {
		t.Fatalf("setup: first CreateChannel failed: %s", reason)
	}

	ch, reason := s.Channels().CreateChannel(ws.NewChannelBuilder().SetKey("dup"))
	if ch != nil {
		t.Fatalf("expected the duplicate key to be rejected")
	}
	if reason == "" {
		t.Fatalf("expected a non-empty rejection reason")
	}
}

func TestChannelRepo_CreateChannel_ValidatorRejectsAndRollsBack(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)
	s.SetChannelValidator(func(b ws.IChannelBuilder) (bool, string) {
		return false, "denied by policy"
	})

	ch, reason := s.Channels().CreateChannel(ws.NewChannelBuilder().SetKey("k1"))
	if ch != nil {
		t.Fatalf("expected the validator's denial to reject creation")
	}
	if reason != "denied by policy" {
		t.Fatalf("expected the validator's reason, got %q", reason)
	}
	if s.Channels().Has("k1") {
		t.Fatalf("a validator-rejected channel must be rolled back out of the repo")
	}
}

func TestChannelRepo_CreateChannel_ValidatorAllows(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)
	var seenKey string
	s.SetChannelValidator(func(b ws.IChannelBuilder) (bool, string) {
		seenKey = b.Key()
		return true, ""
	})

	ch, reason := s.Channels().CreateChannel(ws.NewChannelBuilder().SetKey("k1"))
	if ch == nil {
		t.Fatalf("expected creation to succeed: %s", reason)
	}
	if seenKey != "k1" {
		t.Fatalf("expected the validator to see the final key, got %q", seenKey)
	}
}

func TestChannelRepo_CreateChannel_MaxChannelsPerConnection(t *testing.T) {
	s := newFakeServer(withMaxChannelsPerConnection(1))
	t.Cleanup(s.close)
	creator := newFakeConnection(s, "creator")
	s.Connections().Add(creator)

	ch1, reason := s.Channels().CreateChannel(ws.NewChannelBuilder().SetCreator(creator))
	if ch1 == nil {
		t.Fatalf("first channel should succeed: %s", reason)
	}
	if creator.CreatedChannelsCount() != 1 {
		t.Fatalf("expected the creator's count incremented to 1, got %d", creator.CreatedChannelsCount())
	}

	ch2, reason := s.Channels().CreateChannel(ws.NewChannelBuilder().SetCreator(creator))
	if ch2 != nil {
		t.Fatalf("expected the second channel to be rejected by the per-connection limit")
	}
	if reason == "" {
		t.Fatalf("expected a non-empty rejection reason")
	}
}

func TestChannelRepo_CreateChannel_ChannelCreatedHandlerRuns(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)
	var gotInitData map[string]any
	var gotChannel ws.IChannel
	s.SetChannelCreatedHandler(func(channel ws.IChannel, initData map[string]any) {
		gotChannel = channel
		gotInitData = initData
	})

	ch, reason := s.Channels().CreateChannel(ws.NewChannelBuilder().SetInitData(map[string]any{"x": 1}))
	if ch == nil {
		t.Fatalf("setup: CreateChannel failed: %s", reason)
	}
	if gotChannel != ch {
		t.Fatalf("expected the handler to receive the created channel")
	}
	if gotInitData["x"] != 1 {
		t.Fatalf("expected the handler to receive the builder's InitData, got %#v", gotInitData)
	}
}

func TestChannelRepo_CreateChannel_PublicAnnouncedToOthersNotCreator(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)
	creator := newFakeConnection(s, "creator")
	other := newFakeConnection(s, "other")
	s.Connections().Add(creator)
	s.Connections().Add(other)

	ch, reason := s.Channels().CreateChannel(ws.NewChannelBuilder().SetCreator(creator).SetPublic(true))
	if ch == nil {
		t.Fatalf("setup: CreateChannel failed: %s", reason)
	}

	if len(creator.sent) != 0 {
		t.Fatalf("the creator gets its own ack elsewhere (Connection.createChannel) - repo announce must skip it, got %#v", creator.sent)
	}
	if len(other.sent) != 1 {
		t.Fatalf("expected exactly one announce to the other connection, got %#v", other.sent)
	}
	payload := other.sent[0].payload.(map[string]any)
	if payload["__lxws_action__"] != "createChannel" {
		t.Fatalf("unexpected announce payload: %#v", payload)
	}
}

func TestChannelRepo_CreateChannel_PrivateNotAnnounced(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)
	other := newFakeConnection(s, "other")
	s.Connections().Add(other)

	ch, reason := s.Channels().CreateChannel(ws.NewChannelBuilder().SetPublic(false))
	if ch == nil {
		t.Fatalf("setup: CreateChannel failed: %s", reason)
	}
	if len(other.sent) != 0 {
		t.Fatalf("a private channel must not be announced to anyone, got %#v", other.sent)
	}
}

func TestChannelRepo_GetHasRemove(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)

	ch, reason := s.Channels().CreateChannel(ws.NewChannelBuilder().SetKey("k1"))
	if ch == nil {
		t.Fatalf("setup: CreateChannel failed: %s", reason)
	}
	if !s.Channels().Has("k1") || s.Channels().Get("k1") != ch {
		t.Fatalf("expected Has/Get to find the created channel")
	}

	s.Channels().Remove("k1")
	if s.Channels().Has("k1") {
		t.Fatalf("expected Remove to drop the channel from the repo")
	}
	if s.Channels().Get("k1") != nil {
		t.Fatalf("expected Get to return nil after Remove")
	}
}

func TestChannelRepo_Channels_ReturnsACopy(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)
	if _, reason := s.Channels().CreateChannel(ws.NewChannelBuilder().SetKey("k1")); reason != "" {
		t.Fatalf("setup: CreateChannel failed: %s", reason)
	}

	snapshot := s.Channels().Channels()
	delete(snapshot, "k1")

	if !s.Channels().Has("k1") {
		t.Fatalf("mutating the returned map must not affect the repo's own state")
	}
}

func TestChannelRepo_PublicChannels(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)
	if _, reason := s.Channels().CreateChannel(ws.NewChannelBuilder().SetKey("pub").SetPublic(true)); reason != "" {
		t.Fatalf("setup: %s", reason)
	}
	if _, reason := s.Channels().CreateChannel(ws.NewChannelBuilder().SetKey("priv").SetPublic(false)); reason != "" {
		t.Fatalf("setup: %s", reason)
	}

	pub := s.Channels().PublicChannels()
	if len(pub) != 1 || pub[0].Key() != "pub" {
		t.Fatalf("expected exactly the public channel, got %#v", pub)
	}
}

func TestChannelRepo_Sweeper_ClosesEmptyChannelPastTTL(t *testing.T) {
	s := newFakeServer(withEmptyChannelTTL(1)) // seconds - the sweeper ticks every second too
	t.Cleanup(s.close)

	ch, reason := s.Channels().CreateChannel(ws.NewChannelBuilder().SetKey("k1"))
	if ch == nil {
		t.Fatalf("setup: CreateChannel failed: %s", reason)
	}

	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		if !s.Channels().Has("k1") {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("expected the sweeper to auto-close the empty channel past its TTL")
}

func TestChannelRepo_Sweeper_NeverTouchesDefaultChannel(t *testing.T) {
	s := newFakeServer(withDefaultChannel("default", nil), withEmptyChannelTTL(1))
	t.Cleanup(s.close)
	s.Channels().Init()

	time.Sleep(2500 * time.Millisecond)
	if !s.Channels().Has("default") {
		t.Fatalf("the sweeper must never auto-close the DefaultChannel, regardless of EmptyChannelTTL")
	}
}

// TestChannelRepo_CreateChannel_ConcurrentKeyGeneration is a regression test
// for a previously-fixed TOCTOU race: concurrent key-generating CreateChannel
// calls used to be able to collide on the same generated key. Run with
// -race to catch any reintroduction of the bug.
func TestChannelRepo_CreateChannel_ConcurrentKeyGeneration(t *testing.T) {
	s := newFakeServer()
	t.Cleanup(s.close)

	const n = 50
	keys := make(chan string, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ch, reason := s.Channels().CreateChannel(ws.NewChannelBuilder())
			if ch == nil {
				t.Errorf("concurrent CreateChannel failed: %s", reason)
				return
			}
			keys <- ch.Key()
		}()
	}
	wg.Wait()
	close(keys)

	seen := map[string]bool{}
	for k := range keys {
		if seen[k] {
			t.Fatalf("duplicate generated key %q under concurrent CreateChannel", k)
		}
		seen[k] = true
	}
	if len(seen) != n {
		t.Fatalf("expected %d distinct channels, got %d", n, len(seen))
	}
}
