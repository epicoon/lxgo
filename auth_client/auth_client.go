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

type BaseResponse struct {
	*lxHttp.Form
	Success      bool   `json:"success"`
	ErrorCode    string `json:"error_code,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

func NewBaseResponse() *BaseResponse {
	return &BaseResponse{Form: lxHttp.NewForm()}
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * AuthConfig
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

type AuthConfig struct {
	lxApp.ComponentConfig
	ID           int
	Secret       string
	RedirectUri  string
	Server       string
	StatePath    string
	LogoutPath   string
	RefreshPath  string
	UserDataPath string
}

/** kernel.CComponentConfig */
func NewAuthConfig() kernel.IAppComponentConfig {
	return &AuthConfig{ComponentConfig: *lxApp.NewComponentConfigStruct()}
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * AuthClient
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

type AuthClient struct {
	*lxApp.AppComponent
}

const APP_COMPONENT_KEY = "lxgo_auth_client"

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
func NewAuthClient() kernel.IAppComponent {
	return &AuthClient{AppComponent: lxApp.NewAppComponent()}
}

func (c *AuthClient) Name() string {
	return "AuthClient"
}

func (c *AuthClient) CConfig() kernel.CAppComponentConfig {
	return NewAuthConfig
}

func (c *AuthClient) Config() *AuthConfig {
	return (c.GetConfig()).(*AuthConfig)
}

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
	tokens := new(Tokens)
	tokens.Set(tokensResp.(*tokensForm))
	return tokens, nil
}

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
