package session

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/epicoon/lxgo/kernel"
)

/** @interface ISession */

// Session is the default ISession implementation - an in-memory key/value
// store scoped to one session ID.
type Session struct {
	id           string
	ctx          kernel.IHandleContext
	data         map[any]any
	createdAt    time.Time
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
func NewSession(id string, ctx kernel.IHandleContext) ISession {
	return &Session{id: id, ctx: ctx, createdAt: time.Now(), data: make(map[any]any)}
}

// ID returns the session's ID.
func (s *Session) ID() string {
	return s.id
}

// SetID changes the session's ID.
func (s *Session) SetID(sid string) {
	s.id = sid
}

// Context returns the kernel.IHandleContext the session was created for.
func (s *Session) Context() kernel.IHandleContext {
	return s.ctx
}

// Set stores val under key, failing if key is already set.
func (s *Session) Set(key any, val any) error {
	s.touch()
	if s.Has(key) {
		return fmt.Errorf("session already has param %s", key)
	}
	s.data[key] = val
	return nil
}

// SetForce stores val under key, overwriting any existing value.
func (s *Session) SetForce(key any, val any) {
	s.touch()
	s.data[key] = val
}

// Has reports whether key is set.
func (s *Session) Has(key any) bool {
	_, exists := s.data[key]
	return exists
}

// Get returns the value stored under key, or nil if it isn't set.
func (s *Session) Get(key any) any {
	s.touch()
	val, exists := s.data[key]
	if !exists {
		return nil
	}
	return val
}

// Keys returns all keys currently stored in the session.
func (s *Session) Keys() []any {
	return slices.Collect(maps.Keys(s.data))
}

// Remove deletes key from the session.
func (s *Session) Remove(key any) error {
	delete(s.data, key)
	return nil
}

// CreatedAt returns when the session was created.
func (s *Session) CreatedAt() time.Time {
	return s.createdAt
}

// LastAccessed returns when the session was last read or written.
func (s *Session) LastAccessed() time.Time {
	return s.lastAccessed
}

func (s *Session) touch() {
	s.lastAccessed = time.Now()
}
