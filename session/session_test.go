package session_test

import (
	"sync"
	"testing"

	"github.com/epicoon/lxgo/session"
)

func TestSession_SetGetHasRemove(t *testing.T) {
	s := session.NewSession("sid1")

	if s.Has("key") {
		t.Fatal("expected Has to be false before Set")
	}
	if err := s.Set("key", "value"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if !s.Has("key") {
		t.Fatal("expected Has to be true after Set")
	}
	if got := s.Get("key"); got != "value" {
		t.Fatalf("Get = %v, want 'value'", got)
	}
	if got := s.Get("missing"); got != nil {
		t.Fatalf("Get(missing) = %v, want nil", got)
	}

	if err := s.Remove("key"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if s.Has("key") {
		t.Fatal("expected Has to be false after Remove")
	}
}

func TestSession_Set_ErrorsOnDuplicateKey(t *testing.T) {
	s := session.NewSession("sid1")
	if err := s.Set("key", "value"); err != nil {
		t.Fatalf("first Set: %v", err)
	}
	if err := s.Set("key", "other"); err == nil {
		t.Fatal("expected an error setting an already-set key")
	}
	if got := s.Get("key"); got != "value" {
		t.Fatalf("Get = %v, want the original 'value' - Set should not have overwritten it", got)
	}
}

func TestSession_SetForce_OverwritesExistingValue(t *testing.T) {
	s := session.NewSession("sid1")
	if err := s.Set("key", "value"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	s.SetForce("key", "other")
	if got := s.Get("key"); got != "other" {
		t.Fatalf("Get = %v, want 'other'", got)
	}
}

func TestSession_Keys(t *testing.T) {
	s := session.NewSession("sid1")
	s.SetForce("a", 1)
	s.SetForce("b", 2)

	keys := s.Keys()
	if len(keys) != 2 {
		t.Fatalf("Keys() = %v, want 2 entries", keys)
	}
}

// TestSession_Concurrent_NoRace is a regression test: the same *Session is
// shared (by ID) across concurrent requests carrying the same session
// cookie, so Set/Get/SetForce/Has/Remove/SetID must be safe to call
// concurrently on one instance. Run with -race.
func TestSession_Concurrent_NoRace(t *testing.T) {
	s := session.NewSession("sid1")

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(4)
		go func(i int) {
			defer wg.Done()
			s.SetForce("key", i)
		}(i)
		go func() {
			defer wg.Done()
			s.Get("key")
		}()
		go func() {
			defer wg.Done()
			s.Has("key")
		}()
		go func() {
			defer wg.Done()
			_ = s.Keys()
			_ = s.LastAccessed()
			_ = s.ID()
		}()
	}
	wg.Wait()
}
