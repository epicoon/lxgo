//go:build integration

package handlers_test

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/epicoon/lxgo/auth/internal/handlers"
	"github.com/epicoon/lxgo/auth/testutils"
	"github.com/stretchr/testify/assert"
)

// TestCreateClientHandler_Success is an integration test for
// CreateClientHandler - self-service OAuth client registration,
// intentionally open (no auth required) same as e.g. Google/GitHub let any
// developer register an app.
func TestCreateClientHandler_Success(t *testing.T) {
	app := testutils.App()
	if app == nil {
		log.Fatalf("Cannot create test application")
	}

	payload := `{"redirect_uri": "https://example.com/callback"}`
	req := httptest.NewRequest(http.MethodPost, "/clients", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler := handlers.NewCreateClientHandler()
	if httpResp := app.Router().Handle(handler, "/clients", w, req); httpResp != nil {
		httpResp.Send(w)
	}
	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var response handlers.CreateClientResponse
	err := json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err, "Response decoding failed")
	assert.True(t, response.Success)
	assert.NotZero(t, response.ID)
	assert.NotEmpty(t, response.Secret)

	// The returned client must actually be usable - FindByID should resolve it.
	client, err := app.ClientsRepo().FindByID(response.ID)
	assert.NoError(t, err)
	assert.Equal(t, "https://example.com/callback", client.RedirectUri)
}

func TestCreateClientHandler_MissingRedirectUri(t *testing.T) {
	testutils.RunMissingReqParamsTest(t, http.MethodPost, "/clients", handlers.NewCreateClientHandler, map[string]any{
		"redirect_uri": "https://example.com/callback",
	})
}

func TestCreateClientHandler_DifferentClientsGetDifferentSecrets(t *testing.T) {
	app := testutils.App()
	if app == nil {
		log.Fatalf("Cannot create test application")
	}

	create := func() *handlers.CreateClientResponse {
		payload := `{"redirect_uri": "https://example.com/callback"}`
		req := httptest.NewRequest(http.MethodPost, "/clients", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler := handlers.NewCreateClientHandler()
		if httpResp := app.Router().Handle(handler, "/clients", w, req); httpResp != nil {
			httpResp.Send(w)
		}
		resp := w.Result()
		defer resp.Body.Close()
		var response handlers.CreateClientResponse
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			t.Fatalf("decode: %v", err)
		}
		return &response
	}

	first := create()
	second := create()

	if first.ID == second.ID {
		t.Fatalf("expected distinct client IDs, got %d twice", first.ID)
	}
	if first.Secret == second.Secret {
		t.Fatal("expected distinct secrets for distinct clients")
	}
}
