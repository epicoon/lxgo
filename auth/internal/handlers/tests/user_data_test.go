package handlers_test

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/epicoon/lxgo/auth/internal/handlers"
	"github.com/epicoon/lxgo/auth/testutils"
	"github.com/stretchr/testify/assert"
)

// TestUserDataHandler_Success
// TestUserDataHandler_MissingParams
// TestUserDataHandler_ClientNotFound
// TestUserDataHandler_NoAuthHeader
// TestUserDataHandler_TokenNotFound
// TestUserDataHandler_TokenExpired

func TestUserDataHandler_Success(t *testing.T) {
	app := testutils.App()
	if app == nil {
		log.Fatalf("Cannot create test application")
	}

	// Prepare server data
	client, err := app.ClientsRepo().FindByID(testutils.TestClientID)
	if err != nil {
		log.Fatalf("Can not get client: %v", err)
	}
	login := "testuser"
	password := "Password123!"
	user, err := app.UsersRepo().Create(login, password)
	assert.NoError(t, err, "Failed to create test user")
	accessToken, err := app.TokensRepo().CreateAccessToken(client, user)
	if err != nil {
		log.Fatalf("Can not create access token: %s", err)
	}
	refreshToken, err := app.TokensRepo().CreateRefreshToken(client, user)
	if err != nil {
		log.Fatalf("Can not create refresh token: %s", err)
	}
	_ = refreshToken

	// Prepare request
	params := url.Values{}
	params.Add("client_id", strconv.Itoa(testutils.TestClientID))
	req := httptest.NewRequest(http.MethodGet, "/user-data?"+params.Encode(), nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken.Value))
	w := httptest.NewRecorder()

	// Run handler
	handler := handlers.NewGetUserHandler()
	app.Router().Handle(handler, "/user-data", w, req)
	resp := w.Result()

	// Clear data
	defer resp.Body.Close()
	defer testutils.CleanupUsersTable()
	defer testutils.CleanupTokensTable()

	// Check response
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var response handlers.GetUserResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err, "Response decoding failed")
	assert.Equal(t, user.Login, response.Login)
}

func TestUserDataHandler_MissingParams(t *testing.T) {
	testutils.RunMissingReqParamsTest(t, http.MethodGet, "/user-data", handlers.NewLogoutHandler, map[string]any{
		"client_id": testutils.TestClientID,
	})
}

func TestUserDataHandler_ClientNotFound(t *testing.T) {
	app := testutils.App()
	if app == nil {
		log.Fatalf("Cannot create test application")
	}

	// Prepare request
	params := url.Values{}
	params.Add("client_id", "9999")
	req := httptest.NewRequest(http.MethodGet, "/user-data?"+params.Encode(), nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test_token")
	w := httptest.NewRecorder()

	// Run handler
	handler := handlers.NewGetUserHandler()
	app.Router().Handle(handler, "/user-data", w, req)
	resp := w.Result()

	// Clear data
	defer resp.Body.Close()

	// Check response
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	var response handlers.FailResponse
	err := json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err, "Response decoding failed")
	assert.Equal(t, handlers.ERR_CLIENT_NOT_FOUND, response.ErrorCode, fmt.Sprintf("Expected response code %v", handlers.ERR_CLIENT_NOT_FOUND))
}

func TestUserDataHandler_NoAuthHeader(t *testing.T) {
	app := testutils.App()
	if app == nil {
		log.Fatalf("Cannot create test application")
	}

	// Prepare request
	params := url.Values{}
	params.Add("client_id", strconv.Itoa(testutils.TestClientID))
	req := httptest.NewRequest(http.MethodGet, "/user-data?"+params.Encode(), nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Run handler
	handler := handlers.NewGetUserHandler()
	app.Router().Handle(handler, "/user-data", w, req)
	resp := w.Result()

	// Clear data
	defer resp.Body.Close()

	// Check response
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	var response handlers.FailResponse
	err := json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err, "Response decoding failed")
	assert.Equal(t, handlers.ERR_INVAL_AUTH_HEADER, response.ErrorCode, fmt.Sprintf("Expected response code %v", handlers.ERR_INVAL_AUTH_HEADER))
}

func TestUserDataHandler_TokenNotFound(t *testing.T) {
	app := testutils.App()
	if app == nil {
		log.Fatalf("Cannot create test application")
	}

	// Prepare request
	params := url.Values{}
	params.Add("client_id", strconv.Itoa(testutils.TestClientID))
	req := httptest.NewRequest(http.MethodGet, "/user-data?"+params.Encode(), nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test_token")
	w := httptest.NewRecorder()

	// Run handler
	handler := handlers.NewGetUserHandler()
	app.Router().Handle(handler, "/user-data", w, req)
	resp := w.Result()

	// Clear data
	defer resp.Body.Close()

	// Check response
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	var response handlers.FailResponse
	err := json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err, "Response decoding failed")
	assert.Equal(t, handlers.ERR_TOKEN_NOT_FOUND, response.ErrorCode, fmt.Sprintf("Expected response code %v", handlers.ERR_TOKEN_NOT_FOUND))
}

func TestUserDataHandler_TokenExpired(t *testing.T) {
	app := testutils.App()
	if app == nil {
		log.Fatalf("Cannot create test application")
	}

	// Prepare server data
	client, err := app.ClientsRepo().FindByID(testutils.TestClientID)
	if err != nil {
		log.Fatalf("Can not get client: %v", err)
	}
	login := "testuser"
	password := "Password123!"
	user, err := app.UsersRepo().Create(login, password)
	assert.NoError(t, err, "Failed to create test user")
	accessToken, err := app.TokensRepo().CreateAccessToken(client, user)
	if err != nil {
		log.Fatalf("Can not create access token: %s", err)
	}
	refreshToken, err := app.TokensRepo().CreateRefreshToken(client, user)
	if err != nil {
		log.Fatalf("Can not create refresh token: %s", err)
	}
	_ = refreshToken
	accessToken.ExpiredAt = time.Now().UTC().Add(-2 * time.Duration(client.AccessTokenLifetime) * time.Second)
	app.Gorm().Save(accessToken)

	// Prepare request
	params := url.Values{}
	params.Add("client_id", strconv.Itoa(testutils.TestClientID))
	req := httptest.NewRequest(http.MethodGet, "/user-data?"+params.Encode(), nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken.Value))
	w := httptest.NewRecorder()

	// Run handler
	handler := handlers.NewGetUserHandler()
	app.Router().Handle(handler, "/user-data", w, req)
	resp := w.Result()

	// Clear data
	defer resp.Body.Close()
	defer testutils.CleanupUsersTable()
	defer testutils.CleanupTokensTable()

	// Check response
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	var response handlers.FailResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err, "Response decoding failed")
	assert.Equal(t, handlers.ERR_TOKEN_EXPIRED, response.ErrorCode, fmt.Sprintf("Expected response code %v", handlers.ERR_TOKEN_EXPIRED))
}
