package session

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/epicoon/lxgo/kernel"
)

type Session struct {
	id           string
	ctx          kernel.IHandleContext
	data         map[any]any
	createdAt    time.Time
	lastAccessed time.Time
}

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

func NewSession(id string, ctx kernel.IHandleContext) ISession {
	return &Session{id: id, ctx: ctx, createdAt: time.Now(), data: make(map[any]any)}
}

func (s *Session) Context() kernel.IHandleContext {
	return s.ctx
}

func (s *Session) Set(key any, val any) error {
	s.touch()
	if s.Has(key) {
		return fmt.Errorf("session already has param %s", key)
	}
	s.data[key] = val
	return nil
}

func (s *Session) SetForce(key any, val any) {
	s.touch()
	s.data[key] = val
}

func (s *Session) Has(key any) bool {
	_, exists := s.data[key]
	return exists
}

func (s *Session) Get(key any) any {
	s.touch()
	val, exists := s.data[key]
	if !exists {
		return nil
	}
	return val
}

func (s *Session) Keys() []any {
	return slices.Collect(maps.Keys(s.data))
}

func (s *Session) Delete(key any) error {
	delete(s.data, key)
	return nil
}

func (s *Session) SessionID() string {
	return s.id
}

func (s *Session) CreatedAt() time.Time {
	return s.createdAt
}

func (s *Session) LastAccessed() time.Time {
	return s.lastAccessed
}

func (s *Session) touch() {
	s.lastAccessed = time.Now()
}
