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
	"github.com/epicoon/lxgo/auth/internal/models"
	"github.com/epicoon/lxgo/auth/testutils"
	"github.com/stretchr/testify/assert"
)

// TestLogoutHandler_Success
// TestLogoutHandler_MissingParams
// TestLogoutHandler_ClientNotFound
// TestLogoutHandler_NoAuthHeader
// TestLogoutHandler_TokenNotFound

func TestLogoutHandler_Success(t *testing.T) {
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
	reqData := map[string]any{
		"client_id": testutils.TestClientID,
	}
	jsonData, _ := json.Marshal(reqData)
	req := httptest.NewRequest(http.MethodPost, "/logout", bytes.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken.Value))
	w := httptest.NewRecorder()

	// Run handler
	handler := handlers.NewLogoutHandler()
	app.Router().Handle(handler, "/logout", w, req)
	resp := w.Result()

	// Clear data
	defer resp.Body.Close()
	defer testutils.CleanupUsersTable()
	defer testutils.CleanupTokensTable()

	// Check tokens deleted
	var tryAccessToken, tryRefreshToken *models.Token
	result := app.Gorm().
		Where("client_id = ? AND user_id = ? AND is_refresh = FALSE", testutils.TestClientID, user.ID).
		First(&tryAccessToken)
	if result.RowsAffected != 0 {
		log.Fatalf("access token is still exist")
	}
	result = app.Gorm().
		Where("client_id = ? AND user_id = ? AND is_refresh = TRUE", testutils.TestClientID, user.ID).
		First(&tryRefreshToken)
	if result.RowsAffected != 0 {
		log.Fatalf("refresh token is still exist")
	}

	// Check response
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestLogoutHandler_MissingParams(t *testing.T) {
	testutils.RunMissingReqParamsTest(t, http.MethodPost, "/logout", handlers.NewLogoutHandler, map[string]any{
		"client_id": testutils.TestClientID,
	})
}

func TestLogoutHandler_ClientNotFound(t *testing.T) {
	app := testutils.App()
	if app == nil {
		log.Fatalf("Cannot create test application")
	}

	// Prepare request
	reqData := map[string]any{
		"client_id": 9999,
	}
	jsonData, _ := json.Marshal(reqData)
	req := httptest.NewRequest(http.MethodPost, "/logout", bytes.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test_token")
	w := httptest.NewRecorder()

	// Run handler
	handler := handlers.NewLogoutHandler()
	app.Router().Handle(handler, "/logout", w, req)
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

func TestLogoutHandler_NoAuthHeader(t *testing.T) {
	app := testutils.App()
	if app == nil {
		log.Fatalf("Cannot create test application")
	}

	// Prepare request
	reqData := map[string]any{
		"client_id": testutils.TestClientID,
	}
	jsonData, _ := json.Marshal(reqData)
	req := httptest.NewRequest(http.MethodPost, "/logout", bytes.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Run handler
	handler := handlers.NewLogoutHandler()
	app.Router().Handle(handler, "/logout", w, req)
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

func TestLogoutHandler_TokenNotFound(t *testing.T) {
	app := testutils.App()
	if app == nil {
		log.Fatalf("Cannot create test application")
	}

	// Prepare request
	reqData := map[string]any{
		"client_id": testutils.TestClientID,
	}
	jsonData, _ := json.Marshal(reqData)
	req := httptest.NewRequest(http.MethodPost, "/logout", bytes.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test_token")
	w := httptest.NewRecorder()

	// Run handler
	handler := handlers.NewLogoutHandler()
	app.Router().Handle(handler, "/logout", w, req)
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
