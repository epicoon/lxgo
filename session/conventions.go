// Package session provides an HTTP session component for lxgo/kernel
// applications - Storage is registered as an app component, starts/reads a
// session on every request through a cookie-carried session ID, and stores
// arbitrary per-session data via the in-memory BaseProvider by default.
package session

import (
	"net/http"
	"time"

	"github.com/epicoon/lxgo/kernel"
)

// APP_COMPONENT_KEY is the key Storage registers itself under - see
// SetAppComponent/AppComponent.
const APP_COMPONENT_KEY = "lxgo_session_storage"

// HANDLE_CONTEXT_KEY is the key the current request's ISession is stored
// under in kernel.IHandleContext - see ExtractSession.
const HANDLE_CONTEXT_KEY = "lxgo_http_session"

// IStorage is the app component that manages sessions: it starts/reads a
// session per request and delegates the actual storage to an IProvider.
type IStorage interface {
	kernel.IAppComponent

	// Scanner returns an IScanner for inspecting the current session storage.
	Scanner() IScanner

	// SessionCookieName returns the cookie name sessions are tracked under.
	SessionCookieName() string

	// StartSession reads ctx's session cookie and returns the matching
	// session, creating a new one (and setting the cookie) if none exists yet.
	StartSession(ctx kernel.IHandleContext) (ISession, error)

	// DestroySession removes sess from storage and clears its cookie by
	// writing an expiring Set-Cookie to w - the current response writer.
	DestroySession(w http.ResponseWriter, sess ISession)

	// SessionByID looks up a session by ID, returning (nil, nil) if it
	// doesn't exist.
	SessionByID(sid string) (ISession, error)

	// SetSessionID re-keys sess under a new ID, replacing its old entry in storage.
	SetSessionID(sess ISession, sid string)

	// GC sweeps expired sessions and reschedules itself for the next sweep.
	GC()

	// Provider returns the underlying IProvider, initializing the default
	// BaseProvider on first call.
	Provider() IProvider
}

// ISession holds one session's data, keyed by ID, alongside its lifecycle
// timestamps.
type ISession interface {
	// ID returns the session's ID.
	ID() string

	// SetID changes the session's ID.
	SetID(sid string)

	// Set stores value under key, failing if key is already set.
	Set(key any, value any) error

	// SetForce stores value under key, overwriting any existing value.
	SetForce(key any, value any)

	// Get returns the value stored under key, or nil if it isn't set.
	Get(key any) any

	// Has reports whether key is set.
	Has(key any) bool

	// Keys returns all keys currently stored in the session.
	Keys() []any

	// Remove deletes key from the session.
	Remove(key any) error

	// CreatedAt returns when the session was created.
	CreatedAt() time.Time

	// LastAccessed returns when the session was last read or written.
	LastAccessed() time.Time
}

// IProvider is the actual session store behind an IStorage - swap in a
// custom implementation (e.g. backed by Redis) via a type embedding
// Storage's BaseProvider-shaped storage, see Storage.getProvider.
type IProvider interface {
	// Clear removes all sessions.
	Clear()

	// AddSession stores sess under sid, replacing sess's own ID.
	AddSession(sess ISession, sid string)

	// SessionInit creates and stores a new session under sid.
	SessionInit(sid string) (ISession, error)

	// SessionExists reports whether a session with the given ID is stored.
	SessionExists(sid string) bool

	// SessionRead returns the stored session with the given ID.
	SessionRead(sid string) (ISession, error)

	// DestroySession removes the session with the given ID.
	DestroySession(sid string) error

	// SessionGC removes every session whose CreatedAt is older than maxLifeTime seconds.
	SessionGC(maxLifeTime int)

	len() int
	content() string
}

// IScanner inspects a session store for debugging/diagnostics - see Storage.Scanner.
type IScanner interface {
	// Len returns the number of sessions currently stored.
	Len() int

	// IsEmpty reports whether the store holds no sessions.
	IsEmpty() bool

	// PrintContent renders every stored session's data as a string.
	PrintContent() string

	// PrintContextContent renders the current request's session data as a string.
	PrintContextContent(ctx kernel.IHandleContext) string
}
