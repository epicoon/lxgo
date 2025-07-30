package session

import (
	"time"

	"github.com/epicoon/lxgo/kernel"
)

const APP_COMPONENT_KEY = "lxgo_session_storage"
const HANDLE_CONTEXT_KEY = "lxgo_http_session"

type IStorage interface {
	kernel.IAppComponent
	Scaner() IScaner
	StartSession(ctx kernel.IHandleContext) ISession
	DestroySession(ISession)
	GC()
	Provider() IProvider
}

type ISession interface {
	Context() kernel.IHandleContext
	Set(key any, value any) error
	SetForce(key any, value any)
	Get(key any) any
	Has(key any) bool
	Keys() []any
	Delete(key any) error
	SessionID() string
	CreatedAt() time.Time
	LastAccessed() time.Time
}

type IProvider interface {
	Clear()
	SessionInit(sid string, ctx kernel.IHandleContext) (ISession, error)
	SessionExists(sid string) bool
	SessionRead(sid string) (ISession, error)
	DestroySession(sid string) error
	SessionGC(maxLifeTime int)
	len() int
	content() string
}

type IScaner interface {
	Len() int
	IsEmpty() bool
	PrintContent() string
}
