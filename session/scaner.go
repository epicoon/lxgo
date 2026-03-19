package session

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/epicoon/lxgo/kernel"
)

type Scaner struct {
	storage  IStorage
	provider IProvider
}

func (s *Scaner) Len() int {
	return s.provider.len()
}

func (s *Scaner) IsEmpty() bool {
	return s.Len() == 0
}

func (s *Scaner) PrintContent() string {
	return s.provider.content()
}

func (s *Scaner) PrintContextContent(ctx kernel.IHandleContext) string {
	cookie, err := ctx.Request().Cookie(s.storage.SessionCookieName())
	if err != nil {
		return fmt.Sprintf("can not get session name from cookie: %v", err)
	}
	if cookie.Value == "" {
		return "session name from cookie is empty"
	}

	sid, _ := url.QueryUnescape(cookie.Value)
	if !s.provider.SessionExists(sid) {
		return fmt.Sprintf("session %s does not exist", sid)
	}

	session, err := s.provider.SessionRead(sid)
	if err != nil {
		return fmt.Sprintf("can not read session %s: %v", sid, err)
	}

	keys := session.Keys()
	pares := make([]string, len(keys))
	for i, key := range keys {
		val := session.Get(key)
		pares[i] = fmt.Sprintf("%v: %v", key, val)
	}

	return strings.Join(pares, ", ")
}
