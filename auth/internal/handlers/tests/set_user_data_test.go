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

// TestSetUserDataHandler_Success
// TestSetUserDataHandler_Upsert
// TestSetUserDataHandler_InvalidJSON
// TestSetUserDataHandler_InsufficientScope
// TestSetUserDataHandler_MissingParams

func TestSetUserDataHandler_Success(t *testing.T) {
	app := testutils.App()
	if app == nil {
		log.Fatalf("Cannot create test application")
	}

	client, err := app.ClientsRepo().FindByID(testutils.TestClientID)
	if err != nil {
		log.Fatalf("Can not get client: %v", err)
	}
	user, err := app.UsersRepo().Create("testuser", "Password123!")
	assert.NoError(t, err, "Failed to create test user")
	accessToken, err := app.TokensRepo().CreateAccessToken(client, user, models.SCOPE_PROFILE_DATA)
	if err != nil {
		log.Fatalf("Can not create access token: %s", err)
	}

	reqData := map[string]any{
		"client_id": testutils.TestClientID,
		"data":      `{"favorite_color":"blue"}`,
	}
	jsonData, _ := json.Marshal(reqData)
	req := httptest.NewRequest(http.MethodPost, "/user-data", bytes.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken.Value))
	w := httptest.NewRecorder()

	handler := handlers.NewSetUserDataHandler()
	app.Router().Handle(handler, "/user-data", w, req)
	resp := w.Result()

	defer resp.Body.Close()
	defer testutils.CleanupUsersTable()
	defer testutils.CleanupTokensTable()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var response handlers.SuccessResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err, "Response decoding failed")
	assert.True(t, response.Success)

	stored, err := app.UsersRepo().FindData(user, client)
	assert.NoError(t, err)
	assert.NotNil(t, stored)
	assert.Equal(t, "blue", stored.Data["favorite_color"])
}

func TestSetUserDataHandler_Upsert(t *testing.T) {
	app := testutils.App()
	if app == nil {
		log.Fatalf("Cannot create test application")
	}

	client, err := app.ClientsRepo().FindByID(testutils.TestClientID)
	if err != nil {
		log.Fatalf("Can not get client: %v", err)
	}
	user, err := app.UsersRepo().Create("testuser", "Password123!")
	assert.NoError(t, err, "Failed to create test user")
	accessToken, err := app.TokensRepo().CreateAccessToken(client, user, models.SCOPE_PROFILE_DATA)
	if err != nil {
		log.Fatalf("Can not create access token: %s", err)
	}
	_, err = app.UsersRepo().SetData(user, client, models.JSONB{"favorite_color": "blue"})
	assert.NoError(t, err, "Failed to store initial test user data")

	reqData := map[string]any{
		"client_id": testutils.TestClientID,
		"data":      `{"favorite_color":"green"}`,
	}
	jsonData, _ := json.Marshal(reqData)
	req := httptest.NewRequest(http.MethodPost, "/user-data", bytes.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken.Value))
	w := httptest.NewRecorder()

	handler := handlers.NewSetUserDataHandler()
	app.Router().Handle(handler, "/user-data", w, req)
	resp := w.Result()

	defer resp.Body.Close()
	defer testutils.CleanupUsersTable()
	defer testutils.CleanupTokensTable()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var count int64
	app.Gorm().Table("user_data").Where("user_id = ? AND client_id = ?", user.ID, client.ID).Count(&count)
	assert.Equal(t, int64(1), count, "expected exactly one user_data row, not a duplicate")

	stored, err := app.UsersRepo().FindData(user, client)
	assert.NoError(t, err)
	assert.NotNil(t, stored)
	assert.Equal(t, "green", stored.Data["favorite_color"])
}

func TestSetUserDataHandler_InvalidJSON(t *testing.T) {
	app := testutils.App()
	if app == nil {
		log.Fatalf("Cannot create test application")
	}

	client, err := app.ClientsRepo().FindByID(testutils.TestClientID)
	if err != nil {
		log.Fatalf("Can not get client: %v", err)
	}
	user, err := app.UsersRepo().Create("testuser", "Password123!")
	assert.NoError(t, err, "Failed to create test user")
	accessToken, err := app.TokensRepo().CreateAccessToken(client, user, models.SCOPE_PROFILE_DATA)
	if err != nil {
		log.Fatalf("Can not create access token: %s", err)
	}

	reqData := map[string]any{
		"client_id": testutils.TestClientID,
		"data":      "not-json",
	}
	jsonData, _ := json.Marshal(reqData)
	req := httptest.NewRequest(http.MethodPost, "/user-data", bytes.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken.Value))
	w := httptest.NewRecorder()

	handler := handlers.NewSetUserDataHandler()
	app.Router().Handle(handler, "/user-data", w, req)
	resp := w.Result()

	defer resp.Body.Close()
	defer testutils.CleanupUsersTable()
	defer testutils.CleanupTokensTable()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestSetUserDataHandler_InsufficientScope(t *testing.T) {
	app := testutils.App()
	if app == nil {
		log.Fatalf("Cannot create test application")
	}

	client, err := app.ClientsRepo().FindByID(testutils.TestClientID)
	if err != nil {
		log.Fatalf("Can not get client: %v", err)
	}
	user, err := app.UsersRepo().Create("testuser", "Password123!")
	assert.NoError(t, err, "Failed to create test user")
	accessToken, err := app.TokensRepo().CreateAccessToken(client, user, models.SCOPE_PROFILE)
	if err != nil {
		log.Fatalf("Can not create access token: %s", err)
	}

	reqData := map[string]any{
		"client_id": testutils.TestClientID,
		"data":      `{"favorite_color":"blue"}`,
	}
	jsonData, _ := json.Marshal(reqData)
	req := httptest.NewRequest(http.MethodPost, "/user-data", bytes.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken.Value))
	w := httptest.NewRecorder()

	handler := handlers.NewSetUserDataHandler()
	app.Router().Handle(handler, "/user-data", w, req)
	resp := w.Result()

	defer resp.Body.Close()
	defer testutils.CleanupUsersTable()
	defer testutils.CleanupTokensTable()

	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	var response handlers.FailResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err, "Response decoding failed")
	assert.Equal(t, handlers.ERR_INSUFFICIENT_SCOPE, response.ErrorCode)
}

func TestSetUserDataHandler_MissingParams(t *testing.T) {
	testutils.RunMissingReqParamsTest(t, http.MethodPost, "/user-data", handlers.NewSetUserDataHandler, map[string]any{
		"client_id": testutils.TestClientID,
		"data":      `{"favorite_color":"blue"}`,
	})
}
