//go:build integration

package handlers_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/epicoon/lxgo/auth/internal/handlers"
	"github.com/epicoon/lxgo/auth/internal/models"
	"github.com/epicoon/lxgo/auth/testutils"
	"github.com/stretchr/testify/assert"
)

// TestRefreshHandler_Success
// TestRefreshHandler_MissingParams
// TestRefreshHandler_ClientNotFound
// TestRefreshHandler_TokenNotFound
// TestRefreshHandler_TokenExpired

func TestRefreshHandler_Success(t *testing.T) {
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
	accessToken, err := app.TokensRepo().CreateAccessToken(client, user, models.SCOPE_PROFILE_DATA)
	if err != nil {
		log.Fatalf("Can not create access token: %s", err)
	}
	_ = accessToken
	refreshToken, err := app.TokensRepo().CreateRefreshToken(client, user, models.SCOPE_PROFILE_DATA)
	if err != nil {
		log.Fatalf("Can not create refresh token: %s", err)
	}

	// Prepare request
	reqData := map[string]any{
		"grant_type":    "refresh_token",
		"client_id":     testutils.TestClientID,
		"client_secret": testutils.TestClientSecret,
		"refresh_token": refreshToken.Value,
	}
	jsonData, _ := json.Marshal(reqData)
	req := httptest.NewRequest(http.MethodPost, "/tokens", bytes.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Run handler
	handler := handlers.NewRefreshHandler()
	if httpResp := app.Router().Handle(handler, "/refresh", w, req); httpResp != nil {
		httpResp.Send(w)
	}
	resp := w.Result()

	// Clear data
	defer resp.Body.Close()
	defer testutils.CleanupUsersTable()
	defer testutils.CleanupTokensTable()

	// Get generated tokens
	var newAccessToken, newRefreshToken *models.Token
	result := app.Gorm().
		Where("client_id = ? AND user_id = ? AND is_refresh = FALSE", testutils.TestClientID, user.ID).
		First(&newAccessToken)
	if result.RowsAffected == 0 {
		log.Fatalf("access token not found")
	}
	if result.Error != nil {
		log.Fatalf("access token not found: %v", result.Error)
	}
	result = app.Gorm().
		Where("client_id = ? AND user_id = ? AND is_refresh = TRUE", testutils.TestClientID, user.ID).
		First(&newRefreshToken)
	if result.RowsAffected == 0 {
		log.Fatalf("refresh token not found")
	}
	if result.Error != nil {
		log.Fatalf("refresh token not found: %v", result.Error)
	}

	// Check response
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var response handlers.TokensResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err, "Response decoding failed")
	assert.Equal(t, newAccessToken.Value, response.AccessToken)
	assert.Equal(t, newRefreshToken.Value, response.RefreshToken)
}

func TestRefreshHandler_MissingParams(t *testing.T) {
	testutils.RunMissingReqParamsTest(t, http.MethodPost, "/refresh", handlers.NewRefreshHandler, map[string]any{
		"grant_type":    "refresh_token",
		"client_id":     testutils.TestClientID,
		"client_secret": testutils.TestClientSecret,
		"refresh_token": "test_token",
	})
}

func TestRefreshHandler_ClientNotFound(t *testing.T) {
	app := testutils.App()
	if app == nil {
		log.Fatalf("Cannot create test application")
	}

	// Prepare request
	reqData := map[string]any{
		"grant_type":    "refresh_token",
		"client_id":     9999,
		"client_secret": "wrong_secret",
		"refresh_token": "test_token",
	}
	jsonData, _ := json.Marshal(reqData)
	req := httptest.NewRequest(http.MethodPost, "/tokens", bytes.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Run handler
	handler := handlers.NewRefreshHandler()
	if httpResp := app.Router().Handle(handler, "/refresh", w, req); httpResp != nil {
		httpResp.Send(w)
	}
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

func TestRefreshHandler_TokenNotFound(t *testing.T) {
	app := testutils.App()
	if app == nil {
		log.Fatalf("Cannot create test application")
	}

	// Prepare request
	reqData := map[string]any{
		"grant_type":    "refresh_token",
		"client_id":     testutils.TestClientID,
		"client_secret": testutils.TestClientSecret,
		"refresh_token": "unexisted_token",
	}
	jsonData, _ := json.Marshal(reqData)
	req := httptest.NewRequest(http.MethodPost, "/tokens", bytes.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Run handler
	handler := handlers.NewRefreshHandler()
	if httpResp := app.Router().Handle(handler, "/refresh", w, req); httpResp != nil {
		httpResp.Send(w)
	}
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

func TestRefreshHandler_TokenExpired(t *testing.T) {
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
	accessToken, err := app.TokensRepo().CreateAccessToken(client, user, models.SCOPE_PROFILE_DATA)
	if err != nil {
		log.Fatalf("Can not create access token: %s", err)
	}
	_ = accessToken
	refreshToken, err := app.TokensRepo().CreateRefreshToken(client, user, models.SCOPE_PROFILE_DATA)
	if err != nil {
		log.Fatalf("Can not create refresh token: %s", err)
	}
	refreshToken.ExpiredAt = time.Now().UTC().Add(-2 * time.Duration(client.RefreshTokenLifetime) * time.Second)
	app.Gorm().Save(refreshToken)

	// Prepare request
	reqData := map[string]any{
		"grant_type":    "refresh_token",
		"client_id":     testutils.TestClientID,
		"client_secret": testutils.TestClientSecret,
		"refresh_token": refreshToken.Value,
	}
	jsonData, _ := json.Marshal(reqData)
	req := httptest.NewRequest(http.MethodPost, "/tokens", bytes.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Run handler
	handler := handlers.NewRefreshHandler()
	if httpResp := app.Router().Handle(handler, "/refresh", w, req); httpResp != nil {
		httpResp.Send(w)
	}
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

// TestRefreshHandler_ScopeExceedsGranted is a security-relevant test of the
// RFC 6749 §6 rule enforced in refresh.go: a /refresh request may narrow
// scope, never broaden it. A token granted only "profile" must not be
// refreshable into "profile:data".
func TestRefreshHandler_ScopeExceedsGranted(t *testing.T) {
	app := testutils.App()
	if app == nil {
		log.Fatalf("Cannot create test application")
	}
	defer testutils.CleanupUsersTable()
	defer testutils.CleanupTokensTable()

	client, err := app.ClientsRepo().FindByID(testutils.TestClientID)
	if err != nil {
		log.Fatalf("Can not get client: %v", err)
	}
	user, err := app.UsersRepo().Create("scope_test_user", "Password123!")
	assert.NoError(t, err)
	if _, err := app.TokensRepo().CreateAccessToken(client, user, models.SCOPE_PROFILE); err != nil {
		log.Fatalf("Can not create access token: %s", err)
	}
	refreshToken, err := app.TokensRepo().CreateRefreshToken(client, user, models.SCOPE_PROFILE)
	if err != nil {
		log.Fatalf("Can not create refresh token: %s", err)
	}

	reqData := map[string]any{
		"grant_type":    "refresh_token",
		"client_id":     testutils.TestClientID,
		"client_secret": testutils.TestClientSecret,
		"refresh_token": refreshToken.Value,
		"scope":         models.SCOPE_PROFILE_DATA,
	}
	jsonData, _ := json.Marshal(reqData)
	req := httptest.NewRequest(http.MethodPost, "/tokens", bytes.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler := handlers.NewRefreshHandler()
	if httpResp := app.Router().Handle(handler, "/refresh", w, req); httpResp != nil {
		httpResp.Send(w)
	}
	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	var response handlers.FailResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err, "Response decoding failed")
	assert.Equal(t, handlers.ERR_INVAL_SCOPE, response.ErrorCode)

	// The refresh token must still be usable afterwards (rejected, not consumed).
	_, _, err = app.TokensRepo().FindTokensByRefresh(client, refreshToken.Value)
	assert.NoError(t, err, "expected the refresh token to remain valid after a rejected scope-broadening attempt")
}
