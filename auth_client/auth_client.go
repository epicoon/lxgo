// Package client is the client-side counterpart of the lxgo/auth
// authentication microservice - it wires an lxgo/kernel-based application
// into the OAuth2-like flow that microservice implements, so you don't have
// to write the integration by hand. See the README for the full setup
// (config, wiring the component, registering the ready-made handlers, and
// the one endpoint - "/get-user" - you still write yourself).
package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"

	"github.com/epicoon/lxgo/kernel"
	lxApp "github.com/epicoon/lxgo/kernel/app"
	lxHttp "github.com/epicoon/lxgo/kernel/http"
)

/** @interface kernel.IForm */

// BaseResponse is the common {success, error_code, error_message} shape the
// authorization service's endpoints reply with.
type BaseResponse struct {
	*lxHttp.Form
	Success      bool   `json:"success"`
	ErrorCode    string `json:"error_code,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

var _ kernel.IForm = (*BaseResponse)(nil)

/** @constructor */

// NewBaseResponse returns an empty BaseResponse, ready to be used as a
// lxHttp.RequestBuilder response form.
func NewBaseResponse() *BaseResponse {
	return &BaseResponse{Form: lxHttp.NewForm()}
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * AuthConfig
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

// AuthConfig is AuthClient's config - see "Add the app component to your
// app config file" in the README for the full config.yaml shape this binds to.
type AuthConfig struct {
	lxApp.ComponentConfig
	// ID is this client's ID, as registered with the authorization service.
	ID int
	// Secret is this client's secret, as registered with the authorization service.
	Secret string
	// RedirectUri is where the authorization service sends the user back to
	// after authenticating - must match what's registered for this client.
	RedirectUri string
	// Server is the authorization service's base URL.
	Server string
	// StatePath is the local route that generates the CSRF state - see NewStateHandler.
	StatePath string
	// LogoutPath is the local route that proxies /logout - see NewLogoutHandler.
	LogoutPath string
	// RefreshPath is the local route that proxies /refresh - see NewRefreshHandler.
	RefreshPath string
	// UserDataPath is the local route your own handler proxies /user-data
	// through - see AuthClient.GetUserData.
	UserDataPath string
}

/** kernel.CComponentConfig */

// NewAuthConfig returns an empty AuthConfig - see AuthClient.CConfig, not
// normally called directly.
func NewAuthConfig() kernel.IAppComponentConfig {
	return &AuthConfig{ComponentConfig: *lxApp.NewComponentConfigStruct()}
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * AuthClient
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

/** @interface kernel.IAppComponent */

// AuthClient is the app component that talks to the authorization service's
// API - see SetAppComponent to set one up and AppComponent to get a handle
// to it.
type AuthClient struct {
	*lxApp.AppComponent
}

var _ kernel.IAppComponent = (*AuthClient)(nil)

// APP_COMPONENT_KEY is the key this component registers itself under via
// kernel.IApp.SetComponent - see AppComponent.
const APP_COMPONENT_KEY = "lxgo_auth_client"

// SetAppComponent creates an AuthClient, configures it from configKey (see
// "Add the app component to your app config file" in the README), and
// registers it on app under APP_COMPONENT_KEY - errors if the app already
// has that component.
func SetAppComponent(app kernel.IApp, configKey string) error {
	if app.HasComponent(APP_COMPONENT_KEY) {
		return fmt.Errorf("the application already has component: %s", APP_COMPONENT_KEY)
	}

	authClient := NewAuthClient()
	err := lxApp.InitComponent(authClient, app, configKey)
	if err != nil {
		return fmt.Errorf("can not init session storage component: %s", err)
	}

	app.SetComponent(APP_COMPONENT_KEY, authClient)
	return nil
}

// AppComponent returns app's AuthClient, previously set up via
// SetAppComponent - errors if there isn't one.
func AppComponent(app kernel.IApp) (*AuthClient, error) {
	c := app.Component(APP_COMPONENT_KEY)
	if c == nil {
		return nil, fmt.Errorf("application component '%s' not found", APP_COMPONENT_KEY)
	}

	authClient, ok := c.(*AuthClient)
	if !ok {
		return nil, fmt.Errorf("application component '%s' is not '*AuthClient'", APP_COMPONENT_KEY)
	}

	return authClient, nil
}

/** @constructor */

// NewAuthClient constructs a bare AuthClient - normally reached via
// SetAppComponent instead of calling this directly.
func NewAuthClient() kernel.IAppComponent {
	return &AuthClient{AppComponent: lxApp.NewAppComponent()}
}

// Name returns the component's registration name ("AuthClient").
func (c *AuthClient) Name() string {
	return "AuthClient"
}

// CConfig returns the AuthConfig constructor - see kernel.CAppComponentConfig.
func (c *AuthClient) CConfig() kernel.CAppComponentConfig {
	return NewAuthConfig
}

// Config returns the component's bound AuthConfig.
func (c *AuthClient) Config() *AuthConfig {
	return (c.GetConfig()).(*AuthConfig)
}

// PrepareClientSettings renders the client-side config (client ID,
// redirect/state/logout/refresh/user-data paths) as an inline <script> tag
// that sets window._lxauth_settings - embed it in a page template so
// browser-side auth code can read its own configuration.
func (c *AuthClient) PrepareClientSettings() template.HTML {
	config := c.Config()
	data := struct {
		ID           int    `json:"id"`
		RedirectUri  string `json:"redirect_uri"`
		Server       string `json:"server"`
		StatePath    string `json:"state_path"`
		LogoutPath   string `json:"logout_path"`
		RefreshPath  string `json:"refresh_path"`
		UserDataPath string `json:"user_data_path"`
	}{
		ID:           config.ID,
		RedirectUri:  config.RedirectUri,
		Server:       config.Server,
		StatePath:    config.StatePath,
		LogoutPath:   config.LogoutPath,
		RefreshPath:  config.RefreshPath,
		UserDataPath: config.UserDataPath,
	}

	jsonStr, err := json.Marshal(data)
	if err != nil {
		c.App().LogError(fmt.Sprintf("Can not JSON-encode authentication params: %v", data), "AuthClient")
		return ""
	}

	return template.HTML("<script>window._lxauth_settings='" + string(jsonStr) + "'</script>")
}

// ExchangeCodeForTokens exchanges an authorization code (received via
// NewAuthCallbackHandler) for a token pair.
func (c *AuthClient) ExchangeCodeForTokens(code string) (*Tokens, error) {
	config := c.Config()
	_, tokensResp, err := lxHttp.RequestBuilder().
		SetURL(config.Server + "/tokens").
		SetMethod("POST").
		SetJson().
		SetParams(map[string]any{
			"grant_type":    "authorization_code",
			"code":          code,
			"client_id":     config.ID,
			"client_secret": config.Secret,
		}).
		SetResponseForm(&tokensForm{}).
		Send()
	if err != nil {
		return nil, err
	}
	form := tokensResp.(*tokensForm)
	if !form.Success {
		return nil, errors.New(form.ErrorMessage)
	}
	tokens := new(Tokens)
	tokens.Set(form)
	return tokens, nil
}

// LogOut revokes accessToken on the authorization service.
func (c *AuthClient) LogOut(accessToken string) error {
	config := c.Config()
	_, resp, err := lxHttp.RequestBuilder().
		SetURL(config.Server+"/logout").
		SetMethod("POST").
		AddHeader("Authorization", "Bearer "+accessToken).
		SetJson().
		SetParams(map[string]any{
			"client_id": config.ID,
		}).
		SetResponseForm(NewBaseResponse()).
		Send()
	if err != nil {
		return err
	}

	r := resp.(*BaseResponse)
	if !r.Success {
		return errors.New(r.ErrorMessage)
	}

	return nil
}

// RefreshTokens exchanges a refresh token for a new token pair - narrowing
// the granted scope is possible, broadening it is not (RFC 6749 §6).
func (c *AuthClient) RefreshTokens(refreshToken string) (*Tokens, error) {
	config := c.Config()
	_, resp, err := lxHttp.RequestBuilder().
		SetURL(config.Server + "/refresh").
		SetMethod("POST").
		SetJson().
		SetParams(map[string]any{
			"grant_type":    "refresh_token",
			"client_id":     config.ID,
			"client_secret": config.Secret,
			"refresh_token": refreshToken,
		}).
		SetResponseForm(&tokensForm{}).
		Send()
	if err != nil {
		return nil, err
	}
	tokensResp := resp.(*tokensForm)
	if !tokensResp.Success {
		return nil, errors.New(tokensResp.ErrorMessage)
	}
	tokens := new(Tokens)
	tokens.Set(tokensResp)
	return tokens, nil
}

// GetUserData fetches the data an authenticated user stored on the
// authorization service (see lxgo/auth's /user-data) - requires the
// "profile:data" scope; accessToken must be valid.
func (c *AuthClient) GetUserData(accessToken string) (*UserData, error) {
	config := c.Config()
	type respForm struct {
		Success      bool   `json:"success"`
		ErrorCode    int    `json:"error_code,omitempty"`
		ErrorMessage string `json:"error_message,omitempty"`
		Login        string `json:"login"`
		Data         string `json:"data"`
	}

	resp, form, err := lxHttp.RequestBuilder().
		SetURL(config.Server+"/user-data").
		SetMethod("GET").
		AddHeader("Authorization", "Bearer "+accessToken).
		SetJson().
		SetParams(map[string]any{
			"client_id": config.ID,
		}).
		SetResponseForm(&respForm{}).
		Send()
	if err != nil {
		return nil, err
	}
	_ = resp

	result := form.(*respForm)
	if !result.Success {
		return nil, fmt.Errorf("%d: %s", result.ErrorCode, result.ErrorMessage)
	}

	data := make(map[string]any)
	err = json.Unmarshal([]byte(result.Data), &data)
	if err != nil {
		return nil, err
	}

	userData := &UserData{
		Login: result.Login,
		Data:  data,
	}

	return userData, nil
}
