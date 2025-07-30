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

// TestTokensHandler_Success
// TestTokensHandler_MissingParameters
// TestTokensHandler_CodeNotFound
// TestTokensHandler_InvalidCode

func TestTokensHandler_Success(t *testing.T) {
	app := testutils.App()
	if app == nil {
		log.Fatalf("Cannot create test application")
	}

	// Prepare server data
	login := "testuser"
	password := "Password123!"
	user, err := app.UsersRepo().Create(login, password)
	assert.NoError(t, err, "Failed to create test user")
	authCode, err := app.CodesRepo().Create(testutils.TestClientID, user.ID)
	if err != nil {
		log.Fatalf("Can not create code: %s", err)
	}

	// Prepare request
	reqData := map[string]any{
		"code":          authCode.Value,
		"client_id":     testutils.TestClientID,
		"client_secret": testutils.TestClientSecret,
	}
	jsonData, _ := json.Marshal(reqData)
	req := httptest.NewRequest(http.MethodPost, "/tokens", bytes.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Run handler
	handler := handlers.NewTokensHandler()
	app.Router().Handle(handler, "/tokens", w, req)
	resp := w.Result()

	// Clear data
	defer resp.Body.Close()
	defer testutils.CleanupUsersTable()
	defer testutils.CleanupCodesTable()
	defer testutils.CleanupTokensTable()

	// Get generated tokens
	var accessToken, refreshToken *models.Token
	result := app.Gorm().
		Where("client_id = ? AND user_id = ? AND is_refresh = FALSE", testutils.TestClientID, user.ID).
		First(&accessToken)
	if result.RowsAffected == 0 {
		log.Fatalf("access token not found")
	}
	if result.Error != nil {
		log.Fatalf("access token not found: %v", result.Error)
	}
	result = app.Gorm().
		Where("client_id = ? AND user_id = ? AND is_refresh = TRUE", testutils.TestClientID, user.ID).
		First(&refreshToken)
	if result.RowsAffected == 0 {
		log.Fatalf("refresh token not found")
	}
	if result.Error != nil {
		log.Fatalf("refresh token not found: %v", result.Error)
	}

	// Check response
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected status OK")
	var response handlers.TokensResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err, "Response decoding failed")
	assert.Equal(t, accessToken.Value, response.AccessToken)
	assert.Equal(t, refreshToken.Value, response.RefreshToken)
}

func TestTokensHandler_MissingParameters(t *testing.T) {
	testutils.RunMissingReqParamsTest(t, http.MethodPost, "/tokens", handlers.NewTokensHandler, map[string]any{
		"code":          "testcode",
		"client_id":     testutils.TestClientID,
		"client_secret": testutils.TestClientSecret,
	})
}

func TestTokensHandler_CodeNotFound(t *testing.T) {
	app := testutils.App()
	if app == nil {
		log.Fatalf("Cannot create test application")
	}

	// Prepare request
	reqData := map[string]any{
		"code":          "unexisted_code",
		"client_id":     testutils.TestClientID,
		"client_secret": testutils.TestClientSecret,
	}
	jsonData, _ := json.Marshal(reqData)
	req := httptest.NewRequest(http.MethodPost, "/tokens", bytes.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Run handler
	handler := handlers.NewTokensHandler()
	app.Router().Handle(handler, "/tokens", w, req)
	resp := w.Result()

	// Clear data
	defer resp.Body.Close()

	// Check response
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "Expected status 400")
	var response handlers.FailResponse
	err := json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err, "Response decoding failed")
	assert.Equal(t, handlers.ERR_INVAL_CODE, response.ErrorCode, fmt.Sprintf("Expected response code %v", handlers.ERR_INVAL_CODE))
}

func TestTokensHandler_InvalidCode(t *testing.T) {
	app := testutils.App()
	if app == nil {
		log.Fatalf("Cannot create test application")
	}

	// Prepare server data
	login := "testuser"
	password := "Password123!"
	user, err := app.UsersRepo().Create(login, password)
	assert.NoError(t, err, "Failed to create test user")
	authCode, err := app.CodesRepo().Create(1, user.ID)
	if err != nil {
		log.Fatalf("Can not create code: %s", err)
	}

	// Prepare request
	reqData := map[string]any{
		"code":          authCode.Value,
		"client_id":     testutils.TestClientID,
		"client_secret": testutils.TestClientSecret,
	}
	jsonData, _ := json.Marshal(reqData)
	req := httptest.NewRequest(http.MethodPost, "/tokens", bytes.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Run handler
	handler := handlers.NewTokensHandler()
	app.Router().Handle(handler, "/tokens", w, req)
	resp := w.Result()

	// Clear data
	defer resp.Body.Close()
	defer testutils.CleanupUsersTable()
	defer testutils.CleanupCodesTable()
	defer testutils.CleanupTokensTable()

	// Check response
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "Expected status 400")
	var response handlers.FailResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err, "Response decoding failed")
	assert.Equal(t, handlers.ERR_INVAL_CODE, response.ErrorCode, fmt.Sprintf("Expected response code %v", handlers.ERR_INVAL_CODE))
}
