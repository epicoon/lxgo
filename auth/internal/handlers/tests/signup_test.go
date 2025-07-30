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

// TestSignupHandler_Success
// TestSignupHandler_MissingFields
// TestSignupHandler_InvalidLoginFormat
// TestSignupHandler_InvalidPasswordFormat
// TestSignupHandler_UserAlreadyExists

func TestSignupHandler_Success(t *testing.T) {
	// Get test application
	app := testutils.App()
	if app == nil {
		log.Fatalf("Can not create test application")
	}

	// Prepare request
	payload := `{"login": "testuser", "password": "Password123!"}`
	req := httptest.NewRequest(http.MethodPost, "/signup", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Run handler
	handler := handlers.NewSignupHandler()
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
	app.Router().Handle(handler, "/signup", w, req)
	resp := w.Result()

	// Clear data
	defer resp.Body.Close()
	defer testutils.CleanupSession()
	defer testutils.CleanupUsersTable()

	// Check response
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected status OK")
	var response handlers.FailResponse
	err := json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err, "Response decoding failed")
	assert.True(t, response.Success, "Expected success response")
}

func TestSignupHandler_MissingFields(t *testing.T) {
	app := testutils.App()
	if app == nil {
		log.Fatalf("Cannot create test application")
	}

	// Request without login
	payload := `{"password": "Password123!"}`
	req := httptest.NewRequest(http.MethodPost, "/signup", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler := handlers.NewSignupHandler()
	app.Router().Handle(handler, "/signup", w, req)
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
	req = httptest.NewRequest(http.MethodPost, "/signup", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	handler = handlers.NewSignupHandler()
	app.Router().Handle(handler, "/signup", w, req)
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

func TestSignupHandler_InvalidLoginFormat(t *testing.T) {
	app := testutils.App()
	if app == nil {
		log.Fatalf("Cannot create test application")
	}

	// List of invalid logins
	logins := []string{
		"invalid login!",
		"!@#invaliduser",
		"too_long_login_name_that_exceeds_the_limit_1234567890",
		"sh",
		"UPPER-CASE!",
		"user@domain.com",
		"double__dots..",
		".leadingdot",
		"_leadingunderscore",
		"",
		"      ",
	}

	for _, login := range logins {
		// Request with invalid login
		payload := fmt.Sprintf(`{"login": "%s", "password": "Password123!"}`, login)
		req := httptest.NewRequest(http.MethodPost, "/signup", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler := handlers.NewSignupHandler()
		app.Router().Handle(handler, "/signup", w, req)
		resp := w.Result()

		// Clear data
		defer resp.Body.Close()
		defer testutils.CleanupUsersTable()

		// Check response
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode, fmt.Sprintf("Expected status Bad Request. Login: %s", login))
		var response handlers.FailResponse
		err := json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err, "Response decoding failed")
		assert.Equal(t, handlers.ERR_INVAL_LOGIN, response.ErrorCode, fmt.Sprintf("Expected response code %v", handlers.ERR_INVAL_LOGIN))
	}
}

func TestSignupHandler_InvalidPasswordFormat(t *testing.T) {
	app := testutils.App()
	if app == nil {
		log.Fatalf("Cannot create test application")
	}

	// List of invalid logins
	pwds := []string{
		"short!",
		"nouppercase123!",
		"NOLOWERCASE123!",
		"NoSpecialChar123",
		"NoNumbers!!!!",
		"12345678",
		"!@#$%^&*",
		"Password",
	}

	for _, pwd := range pwds {
		// Request with invalid login
		payload := fmt.Sprintf(`{"login": "testuser", "password": "%s"}`, pwd)
		req := httptest.NewRequest(http.MethodPost, "/signup", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler := handlers.NewSignupHandler()
		app.Router().Handle(handler, "/signup", w, req)
		resp := w.Result()

		// Clear data
		defer resp.Body.Close()
		defer testutils.CleanupUsersTable()

		// Check response
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode, fmt.Sprintf("Expected status Bad Request. Password: %s", pwd))
		var response handlers.FailResponse
		err := json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err, "Response decoding failed")
		assert.Equal(t, handlers.ERR_INVAL_PWD, response.ErrorCode, fmt.Sprintf("Expected response code %v", handlers.ERR_INVAL_PWD))
	}
}

func TestSignupHandler_UserAlreadyExists(t *testing.T) {
	app := testutils.App()
	if app == nil {
		log.Fatalf("Cannot create test application")
	}

	// Create a user first
	login := "existinguser"
	password := "Password123!"
	user, err := app.UsersRepo().Create(login, password)
	_ = user
	assert.NoError(t, err, "Failed to create test user")

	// Attempt to create the same user again
	payload := `{"login": "existinguser", "password": "Password123!"}`
	req := httptest.NewRequest(http.MethodPost, "/signup", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler := handlers.NewSignupHandler()
	app.Router().Handle(handler, "/signup", w, req)
	resp := w.Result()

	// Clear data
	defer resp.Body.Close()
	defer testutils.CleanupUsersTable()

	// Check response
	assert.Equal(t, http.StatusConflict, resp.StatusCode, "Expected status Conflict")
	var response handlers.FailResponse
	_ = json.NewDecoder(resp.Body).Decode(&response)
	assert.Equal(t, handlers.ERR_LOGIN_EXISTS, response.ErrorCode, "Expected login exists error code")
}
