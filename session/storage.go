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

type Config struct {
	*lxApp.ComponentConfig
	CookieName  string
	MaxLifeTime int
}

/** kernel.CAppComponentConfig */
func NewConfig() kernel.IAppComponentConfig {
	return &Config{ComponentConfig: lxApp.NewComponentConfigStruct()}
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * Storage
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

/** @interface session.IStorage */
type Storage struct {
	*lxApp.AppComponent
	lock     sync.Mutex
	provider IProvider
}

var _ IStorage = (*Storage)(nil)

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

// @constructor
func NewStorage() IStorage {
	return &Storage{AppComponent: lxApp.NewAppComponent()}
}

func (s *Storage) Name() string {
	return "SessionsStorage"
}

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

func (c *Storage) CConfig() kernel.CAppComponentConfig {
	return NewConfig
}

func (c *Storage) Config() *Config {
	return (c.GetConfig()).(*Config)
}

func (s *Storage) Scaner() IScaner {
	return &Scaner{
		storage:  s,
		provider: s.provider,
	}
}

func (s *Storage) SessionCookieName() string {
	return s.Config().CookieName
}

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

func (s *Storage) SetSessionID(sess ISession, sid string) {
	provider := s.getProvider()
	provider.DestroySession(sess.ID())
	provider.AddSession(sess, sid)
}

func (s *Storage) GC() {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.getProvider().SessionGC(s.Config().MaxLifeTime)
	time.AfterFunc(time.Duration(s.Config().MaxLifeTime), func() { s.GC() })
}

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
