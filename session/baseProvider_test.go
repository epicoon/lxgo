package session_test

import (
	"sync"
	"testing"
	"time"

	"github.com/epicoon/lxgo/session"
)

func TestBaseProvider_SessionInitAndRead(t *testing.T) {
	p := session.NewBaseProvider()

	if p.SessionExists("sid1") {
		t.Fatal("expected SessionExists to be false before init")
	}

	sess, err := p.SessionInit("sid1")
	if err != nil {
		t.Fatalf("SessionInit: %v", err)
	}
	if sess.ID() != "sid1" {
		t.Fatalf("ID() = %q, want 'sid1'", sess.ID())
	}
	if !p.SessionExists("sid1") {
		t.Fatal("expected SessionExists to be true after init")
	}

	read, err := p.SessionRead("sid1")
	if err != nil {
		t.Fatalf("SessionRead: %v", err)
	}
	if read != sess {
		t.Fatal("expected SessionRead to return the same *Session instance as SessionInit")
	}
}

func TestBaseProvider_SessionRead_NotFound(t *testing.T) {
	p := session.NewBaseProvider()
	if _, err := p.SessionRead("nope"); err == nil {
		t.Fatal("expected an error reading a session that was never created")
	}
}

func TestBaseProvider_DestroySession(t *testing.T) {
	p := session.NewBaseProvider()
	if _, err := p.SessionInit("sid1"); err != nil {
		t.Fatalf("SessionInit: %v", err)
	}
	if err := p.DestroySession("sid1"); err != nil {
		t.Fatalf("DestroySession: %v", err)
	}
	if p.SessionExists("sid1") {
		t.Fatal("expected SessionExists to be false after DestroySession")
	}
}

// TestBaseProvider_AddSession_StoresUnderNewKey: AddSession only adds the
// new mapping and updates the session's own ID - it does not remove
// whatever the session was previously keyed under (that's the caller's
// job, see Storage.SetSessionID, which calls DestroySession(old ID) first).
func TestBaseProvider_AddSession_StoresUnderNewKey(t *testing.T) {
	p := session.NewBaseProvider()
	sess, err := p.SessionInit("old")
	if err != nil {
		t.Fatalf("SessionInit: %v", err)
	}

	p.AddSession(sess, "new")

	if !p.SessionExists("new") {
		t.Fatal("expected the new key to exist after AddSession")
	}
	if sess.ID() != "new" {
		t.Fatalf("ID() = %q, want 'new'", sess.ID())
	}
}

// TestBaseProvider_SessionGC_KeepsFreshSessions is a regression test:
// SessionGC's self-rescheduling used to treat maxLifeTime as nanoseconds
// instead of seconds (already fixed, see lxgo-session/CHANGE_LOG.md
// v0.1.0-alpha.4). Under that bug, a maxLifeTime of 3600 means 3600
// nanoseconds - far less than the real time it takes to return from
// SessionInit and call SessionGC - so a freshly-created session would
// already look expired and get swept immediately. With the fix (seconds),
// it must survive.
func TestBaseProvider_SessionGC_KeepsFreshSessions(t *testing.T) {
	p := session.NewBaseProvider()
	if _, err := p.SessionInit("sid1"); err != nil {
		t.Fatalf("SessionInit: %v", err)
	}

	p.SessionGC(3600)

	if !p.SessionExists("sid1") {
		t.Fatal("expected a freshly-created session to survive GC with a 1-hour maxLifeTime")
	}
}

// TestBaseProvider_SessionGC_SweepsExpiredSessions confirms the other
// direction: a session older than maxLifeTime (measured in real,
// wall-clock seconds) does get swept.
func TestBaseProvider_SessionGC_SweepsExpiredSessions(t *testing.T) {
	p := session.NewBaseProvider()
	if _, err := p.SessionInit("sid1"); err != nil {
		t.Fatalf("SessionInit: %v", err)
	}

	time.Sleep(1100 * time.Millisecond)
	p.SessionGC(1)

	if p.SessionExists("sid1") {
		t.Fatal("expected a session older than a 1-second maxLifeTime to be swept")
	}
}

// TestBaseProvider_ConcurrentReadsAndWrites_NoRace is a regression test:
// SessionExists/len/content (the latter two reachable via Scanner) used to
// read the sessions map without taking p.lock, racing with
// AddSession/SessionInit/DestroySession/SessionGC. Run with -race.
func TestBaseProvider_ConcurrentReadsAndWrites_NoRace(t *testing.T) {
	p := session.NewBaseProvider()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(3)
		go func(i int) {
			defer wg.Done()
			sid := string(rune('a' + i%26))
			_, _ = p.SessionInit(sid)
		}(i)
		go func() {
			defer wg.Done()
			p.SessionExists("a")
		}()
		go func() {
			defer wg.Done()
			p.SessionGC(3600)
		}()
	}
	wg.Wait()
}
