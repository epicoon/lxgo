package session_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func newRequestWithoutCookie(t *testing.T) *http.Request {
	t.Helper()
	return httptest.NewRequest("GET", "/whoami", nil)
}

func newRecorder() *httptest.ResponseRecorder {
	return httptest.NewRecorder()
}
