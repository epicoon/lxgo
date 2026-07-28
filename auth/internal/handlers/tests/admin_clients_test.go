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

	"github.com/epicoon/lxgo/auth/internal/handlers"
	"github.com/epicoon/lxgo/auth/internal/models"
	"github.com/epicoon/lxgo/auth/testutils"
	"github.com/stretchr/testify/assert"
)

func createThrowawayClient(t *testing.T) uint {
	t.Helper()
	app := testutils.App()
	client := &models.Client{
		AccessTokenLifetime:  models.DefaultAccessTokenLifetime,
		RefreshTokenLifetime: models.DefaultRefreshTokenLifetime,
		RedirectUri:          "throwaway_redirect",
	}
	if _, err := app.ClientsRepo().Create(client); err != nil {
		t.Fatalf("can not create throwaway client: %v", err)
	}
	return client.ID
}

// TestDeleteClientHandler_Success is an integration test for the
// admin-gated DeleteClientHandler: a bearer token issued through the
// configured admin client (TestAdminClientID), belonging to a User with an
// Admin record, must be able to delete an arbitrary client.
func TestDeleteClientHandler_Success(t *testing.T) {
	app := testutils.App()
	if app == nil {
		log.Fatalf("Cannot create test application")
	}
	defer testutils.CleanupUsersTable()
	defer testutils.CleanupAdminsTable()
	defer testutils.CleanupTokensTable()

	_, accessToken := testutils.CreateAdminUser("admin_success", "Password123!")
	targetID := createThrowawayClient(t)

	payload := fmt.Sprintf(`{"id": %d}`, targetID)
	req := httptest.NewRequest(http.MethodPost, "/admin/clients/delete", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	w := httptest.NewRecorder()

	handler := handlers.NewDeleteClientHandler()
	if httpResp := app.Router().Handle(handler, "/admin/clients/delete", w, req); httpResp != nil {
		httpResp.Send(w)
	}
	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	_, err := app.ClientsRepo().FindByID(targetID)
	assert.Error(t, err, "expected the client to be gone after deletion")
}

func TestDeleteClientHandler_NotAdminToken_Forbidden(t *testing.T) {
	app := testutils.App()
	if app == nil {
		log.Fatalf("Cannot create test application")
	}
	defer testutils.CleanupUsersTable()
	defer testutils.CleanupTokensTable()

	// A regular (non-admin) user, holding a token from the ADMIN client -
	// still must be rejected, since they have no Admin record.
	client, err := app.ClientsRepo().FindByID(testutils.TestAdminClientID)
	assert.NoError(t, err)
	user, err := app.UsersRepo().Create("regular_user", "Password123!")
	assert.NoError(t, err)
	accessToken, err := app.TokensRepo().CreateAccessToken(client, user, "profile")
	assert.NoError(t, err)

	payload := `{"id": 99999}`
	req := httptest.NewRequest(http.MethodPost, "/admin/clients/delete", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken.Value)
	w := httptest.NewRecorder()

	handler := handlers.NewDeleteClientHandler()
	if httpResp := app.Router().Handle(handler, "/admin/clients/delete", w, req); httpResp != nil {
		httpResp.Send(w)
	}
	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	var response handlers.FailResponse
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&response))
	assert.Equal(t, handlers.ERR_NOT_ADMIN, response.ErrorCode)
}

// TestDeleteClientHandler_TokenFromWrongClient_Unauthorized exercises the
// exact security property authenticateAdmin exists for: an admin User's
// token issued through the ORDINARY TestClientID (not the configured admin
// client) must not grant admin access, even though the same User has a
// real Admin record.
func TestDeleteClientHandler_TokenFromWrongClient_Unauthorized(t *testing.T) {
	app := testutils.App()
	if app == nil {
		log.Fatalf("Cannot create test application")
	}
	defer testutils.CleanupUsersTable()
	defer testutils.CleanupAdminsTable()
	defer testutils.CleanupTokensTable()

	user, _ := testutils.CreateAdminUser("admin_wrong_client", "Password123!")

	ordinaryClient, err := app.ClientsRepo().FindByID(testutils.TestClientID)
	assert.NoError(t, err)
	accessToken, err := app.TokensRepo().CreateAccessToken(ordinaryClient, user, "profile")
	assert.NoError(t, err)

	payload := `{"id": 99999}`
	req := httptest.NewRequest(http.MethodPost, "/admin/clients/delete", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken.Value)
	w := httptest.NewRecorder()

	handler := handlers.NewDeleteClientHandler()
	if httpResp := app.Router().Handle(handler, "/admin/clients/delete", w, req); httpResp != nil {
		httpResp.Send(w)
	}
	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	var response handlers.FailResponse
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&response))
	assert.Equal(t, handlers.ERR_TOKEN_NOT_FOUND, response.ErrorCode)
}

func TestDeleteClientHandler_NoAuthHeader(t *testing.T) {
	app := testutils.App()
	if app == nil {
		log.Fatalf("Cannot create test application")
	}

	payload := `{"id": 99999}`
	req := httptest.NewRequest(http.MethodPost, "/admin/clients/delete", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler := handlers.NewDeleteClientHandler()
	if httpResp := app.Router().Handle(handler, "/admin/clients/delete", w, req); httpResp != nil {
		httpResp.Send(w)
	}
	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestDeleteClientHandler_ClientNotFound(t *testing.T) {
	app := testutils.App()
	if app == nil {
		log.Fatalf("Cannot create test application")
	}
	defer testutils.CleanupUsersTable()
	defer testutils.CleanupAdminsTable()
	defer testutils.CleanupTokensTable()

	_, accessToken := testutils.CreateAdminUser("admin_notfound", "Password123!")

	payload := `{"id": 999999}`
	req := httptest.NewRequest(http.MethodPost, "/admin/clients/delete", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	w := httptest.NewRecorder()

	handler := handlers.NewDeleteClientHandler()
	if httpResp := app.Router().Handle(handler, "/admin/clients/delete", w, req); httpResp != nil {
		httpResp.Send(w)
	}
	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	var response handlers.FailResponse
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&response))
	assert.Equal(t, handlers.ERR_CLIENT_NOT_FOUND, response.ErrorCode)
}
