package handlers_test

import (
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/epicoon/lxgo/auth/internal/handlers"
	"github.com/epicoon/lxgo/auth/testutils"
	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/session"
	"github.com/stretchr/testify/assert"
)

// TestReturnHandler_Success
// TestReturnHandler_MissingAuthParams
// TestReturnHandler_MissingAuthCode

func TestReturnHandler_Success(t *testing.T) {
	app := testutils.App()
	if app == nil {
		log.Fatalf("Cannot create test application")
	}

	// Prepare request
	req := httptest.NewRequest(http.MethodGet, "/return", nil)
	w := httptest.NewRecorder()

	// Run handler
	handler := handlers.NewReturnHandler()
	handler.BeforeRun(func(h kernel.IHttpResource) {
		sess, err := session.ExtractSession(h.Context())
		if err != nil {
			log.Fatalf("can not get session: %v", err)
		}
		sess.SetForce("lxgo_auth_params", &handlers.AuthParams{
			ResponseType: "code",
			ClientID:     testutils.TestClientID,
			RedirectUri:  "/test_redirect",
			State:        "test_state",
		})
		sess.SetForce("lxgo_auth_code", "auth_code_123")
	})
	app.Router().Handle(handler, "/return", w, req)
	resp := w.Result()

	// Clear data
	defer resp.Body.Close()
	defer testutils.CleanupSession()

	// Check response
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected status OK")
	assert.Contains(t, w.Body.String(), "auth_code_123", "Expected authorization code in response")
	assert.Contains(t, w.Body.String(), "test_state", "Expected state in response")
	assert.Contains(t, w.Body.String(), "/test_redirect", "Expected redirect URI in response")
}

func TestReturnHandler_MissingAuthParams(t *testing.T) {
	app := testutils.App()
	if app == nil {
		log.Fatalf("Cannot create test application")
	}

	// Prepare request
	req := httptest.NewRequest(http.MethodGet, "/return", nil)
	w := httptest.NewRecorder()

	// Run handler
	handler := handlers.NewReturnHandler()
	app.Router().Handle(handler, "/return", w, req)
	resp := w.Result()

	// Clear data
	defer resp.Body.Close()

	// Check response
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode, "Expected internal server error")
	var response handlers.FailResponse
	err := json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err, "Response decoding failed")
	assert.Equal(t, "Something went wrong", response.ErrorMessage, "Expected message: Something went wrong")
}

func TestReturnHandler_MissingAuthCode(t *testing.T) {
	app := testutils.App()
	if app == nil {
		log.Fatalf("Cannot create test application")
	}

	// Prepare request
	req := httptest.NewRequest(http.MethodGet, "/return", nil)
	w := httptest.NewRecorder()

	// Run handler
	handler := handlers.NewReturnHandler()
	handler.BeforeRun(func(h kernel.IHttpResource) {
		sess, err := session.ExtractSession(h.Context())
		if err != nil {
			log.Fatalf("can not get session: %v", err)
		}
		sess.SetForce("lxgo_auth_params", &handlers.AuthParams{
			ResponseType: "code",
			ClientID:     testutils.TestClientID,
			RedirectUri:  "/test_redirect",
			State:        "test_state",
		})
	})
	app.Router().Handle(handler, "/return", w, req)
	resp := w.Result()

	// Clear data
	defer resp.Body.Close()
	defer testutils.CleanupSession()

	// Check response
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode, "Expected internal server error")
	var response handlers.FailResponse
	err := json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err, "Response decoding failed")
	assert.Equal(t, "Something went wrong", response.ErrorMessage, "Expected message: Something went wrong")
}
