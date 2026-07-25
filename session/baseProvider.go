package session

import (
	"fmt"
	"sync"
	"time"

	"github.com/epicoon/lxgo/kernel"
)

/** @interface IProvider */

// BaseProvider is the default IProvider implementation - an in-process,
// in-memory session store. Not durable across restarts and not shared
// across multiple app instances; swap in a custom IProvider for that.
type BaseProvider struct {
	sessions map[string]ISession
	lock     sync.RWMutex
}

var _ IProvider = (*BaseProvider)(nil)

/** @constructor */

// NewBaseProvider constructs an empty BaseProvider.
func NewBaseProvider() *BaseProvider {
	return &BaseProvider{sessions: make(map[string]ISession)}
}

// Clear removes all sessions.
func (p *BaseProvider) Clear() {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.sessions = make(map[string]ISession)
}

// AddSession stores sess under sid, replacing sess's own ID.
func (p *BaseProvider) AddSession(sess ISession, sid string) {
	p.lock.Lock()
	defer p.lock.Unlock()
	sess.SetID(sid)
	p.sessions[sid] = sess
}

// SessionInit creates and stores a new session under sid.
func (p *BaseProvider) SessionInit(sid string, ctx kernel.IHandleContext) (ISession, error) {
	p.lock.Lock()
	defer p.lock.Unlock()

	session := NewSession(sid, ctx)
	p.sessions[sid] = session

	return session, nil
}

// SessionExists reports whether a session with the given ID is stored.
func (p *BaseProvider) SessionExists(sid string) bool {
	_, exists := p.sessions[sid]
	return exists
}

// SessionRead returns the stored session with the given ID.
func (p *BaseProvider) SessionRead(sid string) (ISession, error) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	session, exists := p.sessions[sid]
	if !exists {
		return nil, fmt.Errorf("session with id %s not found", sid)
	}

	return session, nil
}

// DestroySession removes the session with the given ID.
func (p *BaseProvider) DestroySession(sid string) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	_, exists := p.sessions[sid]
	if exists {
		delete(p.sessions, sid)
	}
	return nil
}

// SessionGC removes every session whose CreatedAt is older than maxLifeTime seconds.
func (p *BaseProvider) SessionGC(maxLifeTime int) {
	p.lock.Lock()
	defer p.lock.Unlock()

	now := time.Now()
	for sid, session := range p.sessions {
		if session.CreatedAt().Add(time.Duration(maxLifeTime) * time.Second).Before(now) {
			delete(p.sessions, sid)
		}
	}
}

func (p *BaseProvider) len() int {
	return len(p.sessions)
}

func (p *BaseProvider) content() string {
	str := "Current sessions:\n"
	for sid, session := range p.sessions {
		str += "* SessionID = " + sid + "\n"
		for _, key := range session.Keys() {
			str += fmt.Sprintf("  - key: %v\n    value: %v\n", key, session.Get(key))
		}
	}
	return str
}
