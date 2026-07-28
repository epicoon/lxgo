//go:build integration

package testutils

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	cvn "github.com/epicoon/lxgo/auth/internal/conventions"
	"github.com/epicoon/lxgo/auth/internal/core"
	"github.com/epicoon/lxgo/auth/internal/models"
	"github.com/epicoon/lxgo/auth/internal/repos"
	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/migrator"
	"github.com/epicoon/lxgo/session"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

const TestClientID = 2
const TestClientSecret = "testsecret"
const TestClientRedirectUri = "test_redirect"

// TestAdminClientID must match config.yaml's Settings.AdminClientID -
// authenticateAdmin (internal/handlers/common.go) only grants admin access
// to tokens issued through this specific client, never TestClientID.
const TestAdminClientID = 3
const TestAdminClientSecret = "admintestsecret"

// Test application
var app cvn.IApp

func SetupTests(m *testing.M) {
	// Create test application
	tryApp, err := core.PrepareApp("testutils/config.yaml")
	if err != nil {
		log.Fatalf("Can not create test application: %s", err)
	}

	app = tryApp

	// Preparing
	actualizeMigrations(app)
	prepareTestClient(app)
	prepareAdminClient(app)
	// TestClientID/TestAdminClientID are inserted with explicit IDs,
	// bypassing the "clients" table's own serial sequence entirely - left
	// alone, the sequence stays at its initial value and the next
	// auto-assigned client (e.g. via CreateClientHandler in a test) can
	// collide with one of these explicit IDs. Advance it past both.
	app.Gorm().Exec("SELECT SETVAL(pg_get_serial_sequence('clients', 'id'), (SELECT MAX(id) FROM clients))")

	// Start tests
	code := m.Run()

	// Return result
	os.Exit(code)
}

func App() cvn.IApp {
	return app
}

func RunMissingReqParamsTest(t *testing.T, method, url string, cRes kernel.CHttpResource, allParams map[string]any) {
	app := App()
	if app == nil {
		log.Fatalf("Cannot create test application")
	}

	var testDatum []struct {
		params     map[string]any
		errSnippet string
	}

	// Gen test data
	for missingKey := range allParams {
		// Gen submap
		subset := make(map[string]any)
		for key, value := range allParams {
			if key != missingKey {
				subset[key] = value
			}
		}

		// Gen error message
		errSnippet := fmt.Sprintf("missing required parameters: %s", missingKey)

		// Add test data
		testDatum = append(testDatum, struct {
			params     map[string]any
			errSnippet string
		}{
			params:     subset,
			errSnippet: errSnippet,
		})
	}

	for _, testData := range testDatum {
		// Prepare request
		jsonData, _ := json.Marshal(testData.params)
		req := httptest.NewRequest(method, url, bytes.NewReader(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Run handler
		handler := cRes()
		if httpResp := app.Router().Handle(handler, url, w, req); httpResp != nil {
			httpResp.Send(w)
		}
		resp := w.Result()

		// Clear data
		defer resp.Body.Close()

		// Check response
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		assert.Contains(t, w.Body.String(), testData.errSnippet)
	}
}

func CleanupSession() {
	sess, err := session.AppComponent(app)
	if err != nil {
		log.Fatalf("Can not get session app-component: %v", err)
	}
	sess.Provider().Clear()
}

func CleanupUsersTable() {
	db := app.Gorm()
	if err := db.Exec("DELETE FROM users").Error; err != nil {
		log.Fatalf("Failed to clean up users table: %v", err)
	}
	db.Exec("ALTER SEQUENCE users_id_seq RESTART WITH 1")
}

func CleanupCodesTable() {
	db := app.Gorm()
	if err := db.Exec("DELETE FROM codes").Error; err != nil {
		log.Fatalf("Failed to clean up codes table: %v", err)
	}
	db.Exec("ALTER SEQUENCE codes_id_seq RESTART WITH 1")
}

func CleanupTokensTable() {
	db := app.Gorm()
	if err := db.Exec("DELETE FROM tokens").Error; err != nil {
		log.Fatalf("Failed to clean up tokens table: %v", err)
	}
	db.Exec("ALTER SEQUENCE tokens_id_seq RESTART WITH 1")
}

func CleanupAdminsTable() {
	db := app.Gorm()
	if err := db.Exec("DELETE FROM admins").Error; err != nil {
		log.Fatalf("Failed to clean up admins table: %v", err)
	}
	db.Exec("ALTER SEQUENCE admins_id_seq RESTART WITH 1")
}

// CreateAdminUser creates a User, gives them an Admin record, and issues an
// access token for TestAdminClientID (the one authenticateAdmin actually
// checks for admin-gated endpoints - a token from TestClientID, even for
// the same admin User, must NOT work there). Returns the access token
// value to use as a bearer token.
func CreateAdminUser(login, password string) (*models.User, string) {
	user, err := app.UsersRepo().Create(login, password)
	if err != nil {
		log.Fatalf("can not create admin test user: %v", err)
	}
	if _, err := app.AdminsRepo().Create(user.ID, models.ROLE_ADMIN); err != nil {
		log.Fatalf("can not create admin record: %v", err)
	}

	adminClient, err := app.ClientsRepo().FindByID(TestAdminClientID)
	if err != nil {
		log.Fatalf("can not find admin client: %v", err)
	}
	accessToken, err := app.TokensRepo().CreateAccessToken(adminClient, user, models.SCOPE_PROFILE)
	if err != nil {
		log.Fatalf("can not create admin access token: %v", err)
	}

	return user, accessToken.Value
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * PRIVATE
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

func actualizeMigrations(app kernel.IApp) {
	migrator.Init(migrator.Config{
		DB:             app.Connection().DB(),
		MigrationsPath: app.Pathfinder().GetAbsPath("migrations"),
	})
	migrator.Up()
}

func prepareTestClient(app cvn.IApp) {
	// Check test client exists
	_, err := app.ClientsRepo().FindByID(TestClientID)
	if err != nil {
		if !errors.Is(err, repos.ErrClientNotFound) {
			log.Fatalf("error while client searching: %v", err)
		}

		// Create test client
		testClient := &models.Client{
			Model:                gorm.Model{ID: TestClientID},
			Secret:               TestClientSecret,
			AccessTokenLifetime:  models.DefaultAccessTokenLifetime,
			RefreshTokenLifetime: models.DefaultRefreshTokenLifetime,
			RedirectUri:          TestClientRedirectUri,
		}
		db := app.Gorm().Session(&gorm.Session{AllowGlobalUpdate: true})
		repo := app.ClientsRepo()
		repo.SetTx(db)
		_, err := repo.Create(testClient)
		if err != nil {
			log.Fatalf("can not create test client: %v", err)
		}
	}
}

// prepareAdminClient creates the Client that config.yaml's
// Settings.AdminClientID (TestAdminClientID) points at - the one
// authenticateAdmin requires admin-gated endpoints' bearer tokens to have
// been issued through, distinct from the plain TestClientID.
func prepareAdminClient(app cvn.IApp) {
	_, err := app.ClientsRepo().FindByID(TestAdminClientID)
	if err != nil {
		if !errors.Is(err, repos.ErrClientNotFound) {
			log.Fatalf("error while admin client searching: %v", err)
		}

		adminClient := &models.Client{
			Model:                gorm.Model{ID: TestAdminClientID},
			Secret:               TestAdminClientSecret,
			AccessTokenLifetime:  models.DefaultAccessTokenLifetime,
			RefreshTokenLifetime: models.DefaultRefreshTokenLifetime,
			RedirectUri:          "admin_test_redirect",
		}
		db := app.Gorm().Session(&gorm.Session{AllowGlobalUpdate: true})
		repo := app.ClientsRepo()
		repo.SetTx(db)
		if _, err := repo.Create(adminClient); err != nil {
			log.Fatalf("can not create admin client: %v", err)
		}
	}
}
