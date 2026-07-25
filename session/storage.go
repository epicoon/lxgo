package session

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/epicoon/lxgo/kernel"
	lxApp "github.com/epicoon/lxgo/kernel/app"
)

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * Config
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

/** @interface kernel.IAppComponentConfig */

// Config is Storage's app-component configuration.
type Config struct {
	*lxApp.ComponentConfig
	// CookieName is the cookie sessions are tracked under - see Storage.SessionCookieName.
	CookieName string
	// MaxLifeTime is how long, in seconds, a session survives without being
	// accessed before GC removes it.
	MaxLifeTime int
}

/** @constructor kernel.CAppComponentConfig */

// NewConfig constructs a Config.
func NewConfig() kernel.IAppComponentConfig {
	return &Config{ComponentConfig: lxApp.NewComponentConfigStruct()}
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * Storage
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

/** @interface kernel.IAppComponent */
/** @interface kernel.IStorage */

// Storage is the default IStorage implementation - see SetAppComponent to
// register it on an app.
type Storage struct {
	*lxApp.AppComponent
	lock     sync.Mutex
	provider IProvider
}

var _ IStorage = (*Storage)(nil)

// SetAppComponent registers a new Storage on app under APP_COMPONENT_KEY,
// configured from the config section named by configKey.
func SetAppComponent(app kernel.IApp, configKey string) error {
	if app.HasComponent(APP_COMPONENT_KEY) {
		return fmt.Errorf("the application already has component: %s", APP_COMPONENT_KEY)
	}

	storage := NewStorage()
	err := lxApp.InitComponent(storage, app, configKey)
	if err != nil {
		return fmt.Errorf("can not init session storage component: %s", err)
	}

	app.SetComponent(APP_COMPONENT_KEY, storage)
	return nil
}

// AppComponent returns the Storage registered on app under APP_COMPONENT_KEY.
func AppComponent(app kernel.IApp) (IStorage, error) {
	c := app.Component(APP_COMPONENT_KEY)
	if c == nil {
		return nil, fmt.Errorf("application component '%s' not found", APP_COMPONENT_KEY)
	}

	storage, ok := c.(IStorage)
	if !ok {
		return nil, fmt.Errorf("application component '%s' is not 'session.IStorage'", APP_COMPONENT_KEY)
	}

	return storage, nil
}

/** @constructor */

// NewStorage constructs a Storage.
func NewStorage() IStorage {
	return &Storage{AppComponent: lxApp.NewAppComponent()}
}

// Name returns the component's name - see kernel.IAppComponent.
func (s *Storage) Name() string {
	return "SessionsStorage"
}

// AfterInit registers the session-loading middleware and starts the GC loop - see kernel.IAppComponent.
func (s *Storage) AfterInit() {
	s.App().Router().AddMiddleware(func(ctx kernel.IHandleContext) error {
		session, err := s.StartSession(ctx)
		if err != nil {
			return err
		}
		ctx.Set(HANDLE_CONTEXT_KEY, session)
		return nil
	})
	s.GC()
}

// CConfig returns Config's constructor - see kernel.IAppComponent.
func (c *Storage) CConfig() kernel.CAppComponentConfig {
	return NewConfig
}

// Config returns the component's Config.
func (c *Storage) Config() *Config {
	return (c.GetConfig()).(*Config)
}

// Scanner returns an IScanner for inspecting the current session storage.
func (s *Storage) Scanner() IScanner {
	return &Scanner{
		storage:  s,
		provider: s.provider,
	}
}

// SessionCookieName returns the cookie name sessions are tracked under.
func (s *Storage) SessionCookieName() string {
	return s.Config().CookieName
}

// StartSession reads ctx's session cookie and returns the matching session,
// creating a new one (and setting the cookie) if none exists yet.
func (s *Storage) StartSession(ctx kernel.IHandleContext) (session ISession, err error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	cookie, cookieErr := ctx.Request().Cookie(s.SessionCookieName())
	provider := s.getProvider()
	if cookieErr != nil || cookie.Value == "" {
		sid := s.sessionId()
		session, err = provider.SessionInit(sid, ctx)
		if err != nil {
			return nil, fmt.Errorf("can not init session: %s", err)
		}
		newCookie := http.Cookie{Name: s.Config().CookieName, Value: url.QueryEscape(sid), Path: "/", HttpOnly: true, MaxAge: int(s.Config().MaxLifeTime)}
		http.SetCookie(ctx.ResponseWriter(), &newCookie)
	} else if sid, _ := url.QueryUnescape(cookie.Value); cookie.Value != "" && !provider.SessionExists(sid) {
		session, err = provider.SessionInit(sid, ctx)
		if err != nil {
			return nil, fmt.Errorf("can not init session: %s", err)
		}
	} else {
		sid, _ := url.QueryUnescape(cookie.Value)
		session, err = provider.SessionRead(sid)
		if err != nil {
			return nil, fmt.Errorf("can not read session: %s", err)
		}
	}
	return session, nil
}

// DestroySession removes sess from storage and clears its cookie.
func (s *Storage) DestroySession(sess ISession) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.getProvider().DestroySession(sess.ID())

	cookie, err := sess.Context().Request().Cookie(s.Config().CookieName)
	if err != nil || cookie.Value == "" {
		return
	}
	expiration := time.Now()
	newCookie := http.Cookie{Name: s.Config().CookieName, Path: "/", HttpOnly: true, Expires: expiration, MaxAge: -1}
	http.SetCookie(sess.Context().ResponseWriter(), &newCookie)
}

// SessionByID looks up a session by ID, returning (nil, nil) if it doesn't exist.
func (s *Storage) SessionByID(sid string) (ISession, error) {
	provider := s.getProvider()
	if !provider.SessionExists(sid) {
		return nil, nil
	}
	session, err := provider.SessionRead(sid)
	if err != nil {
		return nil, fmt.Errorf("can not read session: %s", err)
	}
	return session, nil
}

// SetSessionID re-keys sess under a new ID, replacing its old entry in storage.
func (s *Storage) SetSessionID(sess ISession, sid string) {
	provider := s.getProvider()
	provider.DestroySession(sess.ID())
	provider.AddSession(sess, sid)
}

// GC sweeps expired sessions and reschedules itself for the next sweep,
// MaxLifeTime seconds later.
func (s *Storage) GC() {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.getProvider().SessionGC(s.Config().MaxLifeTime)
	time.AfterFunc(time.Duration(s.Config().MaxLifeTime)*time.Second, func() { s.GC() })
}

// Provider returns the underlying IProvider, initializing the default
// BaseProvider on first call.
func (s *Storage) Provider() IProvider {
	return s.getProvider()
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * PRIVATE
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

func (s *Storage) sessionId() string {
	b := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return ""
	}
	return base64.URLEncoding.EncodeToString(b)
}

func (s *Storage) getProvider() IProvider {
	if s.provider == nil {
		s.provider = NewBaseProvider()
	}
	return s.provider
}
