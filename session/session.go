package session

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/epicoon/lxgo/kernel"
)

/** @interface ISession */

// Session is the default ISession implementation - an in-memory key/value
// store scoped to one session ID.
//
// mu guards id/data/lastAccessed - the same *Session is shared (by ID)
// across concurrent requests carrying the same session cookie, so
// Set/Get/SetForce/etc. can be called concurrently on one instance.
// createdAt is set once in NewSession and never mutated after, so it
// doesn't need the lock.
type Session struct {
	createdAt time.Time

	mu           sync.RWMutex
	id           string
	data         map[any]any
	lastAccessed time.Time
}

var _ ISession = (*Session)(nil)

// ExtractSession returns the ISession that Storage's middleware attached to
// ctx - see HANDLE_CONTEXT_KEY.
func ExtractSession(ctx kernel.IHandleContext) (ISession, error) {
	s := ctx.Get(HANDLE_CONTEXT_KEY)
	if s == nil {
		return nil, errors.New("session not found")
	}

	session, ok := s.(ISession)
	if !ok {
		return nil, errors.New("session is not 'session.ISession'")
	}

	return session, nil
}

/** @constructor */

// NewSession creates an empty Session with the given ID.
func NewSession(id string) ISession {
	return &Session{id: id, createdAt: time.Now(), data: make(map[any]any)}
}

// ID returns the session's ID.
func (s *Session) ID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.id
}

// SetID changes the session's ID.
func (s *Session) SetID(sid string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.id = sid
}

// Set stores val under key, failing if key is already set.
func (s *Session) Set(key any, val any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastAccessed = time.Now()
	if _, exists := s.data[key]; exists {
		return fmt.Errorf("session already has param %s", key)
	}
	s.data[key] = val
	return nil
}

// SetForce stores val under key, overwriting any existing value.
func (s *Session) SetForce(key any, val any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastAccessed = time.Now()
	s.data[key] = val
}

// Has reports whether key is set.
func (s *Session) Has(key any) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.data[key]
	return exists
}

// Get returns the value stored under key, or nil if it isn't set.
func (s *Session) Get(key any) any {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastAccessed = time.Now()
	val, exists := s.data[key]
	if !exists {
		return nil
	}
	return val
}

// Keys returns all keys currently stored in the session.
func (s *Session) Keys() []any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return slices.Collect(maps.Keys(s.data))
}

// Remove deletes key from the session.
func (s *Session) Remove(key any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
	return nil
}

// CreatedAt returns when the session was created.
func (s *Session) CreatedAt() time.Time {
	return s.createdAt
}

// LastAccessed returns when the session was last read or written.
func (s *Session) LastAccessed() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastAccessed
}
