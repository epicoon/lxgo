package session

import (
	"fmt"
	"sync"
	"time"

	"github.com/epicoon/lxgo/kernel"
)

type BaseProvider struct {
	sessions map[string]ISession
	lock     sync.RWMutex
}

var _ IProvider = (*BaseProvider)(nil)

func NewBaseProvider() *BaseProvider {
	return &BaseProvider{sessions: make(map[string]ISession)}
}

func (p *BaseProvider) Clear() {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.sessions = make(map[string]ISession)
}

func (p *BaseProvider) SessionInit(sid string, ctx kernel.IHandleContext) (ISession, error) {
	p.lock.Lock()
	defer p.lock.Unlock()

	session := NewSession(sid, ctx)
	p.sessions[sid] = session

	return session, nil
}

func (p *BaseProvider) SessionExists(sid string) bool {
	_, exists := p.sessions[sid]
	return exists
}

func (p *BaseProvider) SessionRead(sid string) (ISession, error) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	session, exists := p.sessions[sid]
	if !exists {
		return nil, fmt.Errorf("session with id %s not found", sid)
	}

	return session, nil
}

func (p *BaseProvider) DestroySession(sid string) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	_, exists := p.sessions[sid]
	if exists {
		delete(p.sessions, sid)
	}
	return nil
}

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
