package handlers_test

import (
	"bytes"
	"encoding/json"
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
)

// TestLoginHandler_Success
// TestLoginHandler_MissingFields
// TestLoginHandler_WrongFields

func TestLoginHandler_Success(t *testing.T) {
	// Get test application
	app := testutils.App()
	if app == nil {
		log.Fatalf("Can not create test application")
	}

	// Create a user first
	login := "testuser"
	password := "Password123!"
	user, err := app.UsersRepo().Create(login, password)
	_ = user
	assert.NoError(t, err, "Failed to create test user")

	// Prepare request
	payload := `{"login": "testuser", "password": "Password123!"}`
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Run handler
	handler := handlers.NewLoginHandler()
	handler.BeforeRun(func(h kernel.IHttpResource) {
		sess, err := session.ExtractSession(h.Context())
		if err != nil {
			log.Fatalf("can not get session: %v", err)
		}
		sess.SetForce("lxgo_auth_params", &handlers.AuthParams{
			ResponseType: "code",
			ClientID:     testutils.TestClientID,
			RedirectUri:  "/test_uri",
			State:        "test_secret",
		})
	})
	app.Router().Handle(handler, "/login", w, req)
	resp := w.Result()

	// Clear data
	defer resp.Body.Close()
	defer testutils.CleanupUsersTable()

	// Check response
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected status OK")
	var response handlers.FailResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err, "Response decoding failed")
	assert.True(t, response.Success, "Expected success response")
}

func TestLoginHandler_MissingFields(t *testing.T) {
	app := testutils.App()
	if app == nil {
		log.Fatalf("Cannot create test application")
	}

	// Request without login
	payload := `{"password": "Password123!"}`
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler := handlers.NewLoginHandler()
	app.Router().Handle(handler, "/login", w, req)
	resp := w.Result()

	// Clear data
	defer resp.Body.Close()
	defer testutils.CleanupUsersTable()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "Expected status Bad Request")
	var response handlers.FailResponse
	err := json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err, "Response decoding failed")
	assert.Equal(t, handlers.ERR_NO_LOGIN_PWD, response.ErrorCode, fmt.Sprintf("Expected response code %v", handlers.ERR_NO_LOGIN_PWD))

	// Request without password
	payload = `{"login": "testuser"}`
	req = httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	handler = handlers.NewLoginHandler()
	app.Router().Handle(handler, "/login", w, req)
	resp = w.Result()

	// Clear data
	defer resp.Body.Close()
	defer testutils.CleanupUsersTable()

	// Check response
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "Expected status Bad Request")
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err, "Response decoding failed")
	assert.Equal(t, handlers.ERR_NO_LOGIN_PWD, response.ErrorCode, fmt.Sprintf("Expected response code %v", handlers.ERR_NO_LOGIN_PWD))
}

func TestLoginHandler_WrongFields(t *testing.T) {
	// Get test application
	app := testutils.App()
	if app == nil {
		log.Fatalf("Can not create test application")
	}

	// Prepare request
	payload := `{"login": "testuser", "password": "Password123!"}`
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Run handler
	handler := handlers.NewLoginHandler()
	app.Router().Handle(handler, "/login", w, req)
	resp := w.Result()

	// Clear data
	defer resp.Body.Close()
	defer testutils.CleanupUsersTable()

	// Check response
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, "Expected status 401")
	var response handlers.FailResponse
	err := json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err, "Response decoding failed")
	assert.Equal(t, handlers.ERR_WRONG_LOGIN_PWD, response.ErrorCode, fmt.Sprintf("Expected response code %v", handlers.ERR_WRONG_LOGIN_PWD))
}
