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
		ctx.Set(HANDLE_CONTEXT_KEY, s.StartSession(ctx))
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
	return &Scaner{provider: s.provider}
}

func (s *Storage) StartSession(ctx kernel.IHandleContext) (session ISession) {
	s.lock.Lock()
	defer s.lock.Unlock()
	cookie, err := ctx.Request().Cookie(s.Config().CookieName)
	provider := s.getProvider()
	if err != nil || cookie.Value == "" {
		sid := s.sessionId()
		session, _ = provider.SessionInit(sid, ctx)
		cookie := http.Cookie{Name: s.Config().CookieName, Value: url.QueryEscape(sid), Path: "/", HttpOnly: true, MaxAge: int(s.Config().MaxLifeTime)}
		http.SetCookie(ctx.ResponseWriter(), &cookie)
	} else if sid, _ := url.QueryUnescape(cookie.Value); cookie.Value != "" && !provider.SessionExists(sid) {
		session, _ = provider.SessionInit(sid, ctx)
	} else {
		sid, _ := url.QueryUnescape(cookie.Value)
		session, _ = provider.SessionRead(sid)
	}
	return
}

func (s *Storage) DestroySession(sess ISession) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.getProvider().DestroySession(sess.SessionID())

	cookie, err := sess.Context().Request().Cookie(s.Config().CookieName)
	if err != nil || cookie.Value == "" {
		return
	}
	expiration := time.Now()
	newCookie := http.Cookie{Name: s.Config().CookieName, Path: "/", HttpOnly: true, Expires: expiration, MaxAge: -1}
	http.SetCookie(sess.Context().ResponseWriter(), &newCookie)
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
