//go:build integration

package handlers_test

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/epicoon/lxgo/auth/internal/handlers"
	"github.com/epicoon/lxgo/auth/testutils"
	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAuthHandler_GetAuth_Success
// TestAuthHandler_GetAuth_MissingAuthParams
// TestAuthHandler_GetAuth_InvalidClientID
// TestAuthHandler_PostAuth_Success
// TestAuthHandler_PostAuth_MissingAuthParams
// TestAuthHandler_PostAuth_InvalidClientID

func TestAuthHandler_GetAuth_Success(t *testing.T) {
	app := testutils.App()
	if app == nil {
		log.Fatalf("Cannot create test application")
	}

	// Prepare request
	req := httptest.NewRequest(http.MethodGet, "/auth", nil)
	w := httptest.NewRecorder()

	// Run handler
	handler := handlers.NewGetAuthHandler()
	handler.BeforeRun(func(h kernel.IHttpResource) {
		sess, err := session.ExtractSession(h.Context())
		if err != nil {
			log.Fatalf("can not get session: %v", err)
		}
		sess.SetForce("lxgo_auth_params", &handlers.AuthParams{
			ResponseType: "code",
			ClientID:     testutils.TestClientID,
			RedirectUri:  testutils.TestClientRedirectUri,
			State:        "test_state",
		})
	})
	if httpResp := app.Router().Handle(handler, "/auth", w, req); httpResp != nil {
		httpResp.Send(w)
	}
	resp := w.Result()

	// Clear data
	defer resp.Body.Close()
	defer testutils.CleanupSession()

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Auth")
}

func TestAuthHandler_GetAuth_MissingAuthParams(t *testing.T) {
	app := testutils.App()
	if app == nil {
		log.Fatalf("Cannot create test application")
	}

	// Prepare request
	req := httptest.NewRequest(http.MethodGet, "/auth", nil)
	w := httptest.NewRecorder()

	// Run handler
	handler := handlers.NewGetAuthHandler()
	if httpResp := app.Router().Handle(handler, "/auth", w, req); httpResp != nil {
		httpResp.Send(w)
	}
	resp := w.Result()

	// Clear data
	defer resp.Body.Close()
	defer testutils.CleanupSession()

	// Check response
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Don't have auth data")
}

func TestAuthHandler_GetAuth_InvalidClientID(t *testing.T) {
	app := testutils.App()
	if app == nil {
		log.Fatalf("Cannot create test application")
	}

	// Prepare request
	req := httptest.NewRequest(http.MethodGet, "/auth", nil)
	w := httptest.NewRecorder()

	// Run handler
	handler := handlers.NewGetAuthHandler()
	handler.BeforeRun(func(h kernel.IHttpResource) {
		sess, err := session.ExtractSession(h.Context())
		if err != nil {
			log.Fatalf("can not get session: %v", err)
		}
		sess.SetForce("lxgo_auth_params", &handlers.AuthParams{
			ResponseType: "code",
			ClientID:     testutils.TestClientID + 999,
			RedirectUri:  testutils.TestClientRedirectUri,
			State:        "test_state",
		})
	})
	if httpResp := app.Router().Handle(handler, "/auth", w, req); httpResp != nil {
		httpResp.Send(w)
	}
	resp := w.Result()

	// Clear data
	defer resp.Body.Close()
	defer testutils.CleanupSession()

	// Check response
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Client does not exist")
}

func TestAuthHandler_PostAuth_Success(t *testing.T) {
	app := testutils.App()
	if app == nil {
		log.Fatalf("Cannot create test application")
	}

	// Check session
	sessStorage, err := session.AppComponent(app)
	if err != nil {
		log.Fatalf("Can not get session app component: %v", err)
	}
	if !sessStorage.Scanner().IsEmpty() {
		log.Fatal("Session storage has to be empty")
	}

	// Prepare request
	payload := fmt.Sprintf(`{"response_type": "code", "client_id": %d, "redirect_uri": "test_redirect", "state": "test_state"}`, testutils.TestClientID)
	req := httptest.NewRequest(http.MethodPost, "/auth", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Run handler
	handler := handlers.NewPostAuthHandler()
	if httpResp := app.Router().Handle(handler, "/auth", w, req); httpResp != nil {
		httpResp.Send(w)
	}
	resp := w.Result()

	// Clear data
	defer resp.Body.Close()
	defer testutils.CleanupSession()

	// Check session
	sess, err := session.ExtractSession(handler.Context())
	if err != nil {
		log.Fatalf("Can not get session: %v", err)
	}
	el := sess.Get("lxgo_auth_params")
	require.NotNil(t, el, "Session does not contain 'lxgo_auth_params'")
	ap, ok := el.(*handlers.AuthParams)
	if !ok {
		log.Fatal("Session param 'lxgo_auth_params' has to be '*handlers.AuthParams'")
	}
	assert.Equal(t, uint(testutils.TestClientID), ap.ClientID)
	assert.Equal(t, "test_state", ap.State)

	// Check response
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, w.Body.String(), "Auth")
}

// TestAuthHandler_PostAuth_RedirectUriMismatch is a security-relevant test:
// redirect_uri must match the client's registered one exactly (RFC 6749) -
// a request with a different one must be rejected, not silently accepted.
func TestAuthHandler_PostAuth_RedirectUriMismatch(t *testing.T) {
	app := testutils.App()
	if app == nil {
		log.Fatalf("Cannot create test application")
	}

	payload := fmt.Sprintf(`{"response_type": "code", "client_id": %d, "redirect_uri": "https://attacker.example/callback", "state": "test_state"}`, testutils.TestClientID)
	req := httptest.NewRequest(http.MethodPost, "/auth", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler := handlers.NewPostAuthHandler()
	if httpResp := app.Router().Handle(handler, "/auth", w, req); httpResp != nil {
		httpResp.Send(w)
	}
	resp := w.Result()
	defer resp.Body.Close()
	defer testutils.CleanupSession()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Contains(t, w.Body.String(), "Invalid redirect_uri")
}

func TestAuthHandler_PostAuth_MissingAuthParams(t *testing.T) {
	testutils.RunMissingReqParamsTest(t, http.MethodPost, "/auth", handlers.NewPostAuthHandler, map[string]any{
		"response_type": "code",
		"client_id":     testutils.TestClientID,
		"redirect_uri":  "test_redirect",
		"state":         "test_state",
	})
}

func TestAuthHandler_PostAuth_InvalidClientID(t *testing.T) {
	app := testutils.App()
	if app == nil {
		log.Fatalf("Cannot create test application")
	}

	// Prepare request
	payload := `{"response_type": "code", "client_id": 9999, "redirect_uri": "test_redirect", "state": "test_state"}`
	req := httptest.NewRequest(http.MethodPost, "/auth", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Run handler
	handler := handlers.NewPostAuthHandler()
	if httpResp := app.Router().Handle(handler, "/auth", w, req); httpResp != nil {
		httpResp.Send(w)
	}
	resp := w.Result()

	// Clear data
	defer resp.Body.Close()
	defer testutils.CleanupSession()

	// Check response
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Client does not exist")
}
