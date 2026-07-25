package session

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/epicoon/lxgo/kernel"
)

/** @interface IScanner */

// Scanner is the default IScanner implementation - see Storage.Scanner.
type Scanner struct {
	storage  IStorage
	provider IProvider
}

var _ IScanner = (*Scanner)(nil)

// Len returns the number of sessions currently stored.
func (s *Scanner) Len() int {
	return s.provider.len()
}

// IsEmpty reports whether the store holds no sessions.
func (s *Scanner) IsEmpty() bool {
	return s.Len() == 0
}

// PrintContent renders every stored session's data as a string.
func (s *Scanner) PrintContent() string {
	return s.provider.content()
}

// PrintContextContent renders the current request's session data as a string.
func (s *Scanner) PrintContextContent(ctx kernel.IHandleContext) string {
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
