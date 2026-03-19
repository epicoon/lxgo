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
	SessionCookieName() string
	StartSession(ctx kernel.IHandleContext) ISession
	DestroySession(ISession)
	SessionByID(sid string) ISession
	SetSessionID(sess ISession, sid string)
	GC()
	Provider() IProvider
}

type ISession interface {
	ID() string
	SetID(sid string)
	Context() kernel.IHandleContext
	Set(key any, value any) error
	SetForce(key any, value any)
	Get(key any) any
	Has(key any) bool
	Keys() []any
	Remove(key any) error
	CreatedAt() time.Time
	LastAccessed() time.Time
}

type IProvider interface {
	Clear()
	AddSession(sess ISession, sid string)
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
	PrintContextContent(ctx kernel.IHandleContext) string
}
