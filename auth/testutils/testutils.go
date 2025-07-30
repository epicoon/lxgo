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

	// Start tests
	code := m.Run()

	//TODO Clear resources

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
		app.Router().Handle(handler, url, w, req)
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

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * PRIVATE
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

func actualizeMigrations(app kernel.IApp) {
	migrator.Init(app.Connection().DB(), app.Pathfinder().GetAbsPath("migrations"))
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
			RoleID:               4,
			AccessTokenLifetime:  900,
			RefreshTokenLifetime: 604800,
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
