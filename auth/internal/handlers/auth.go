package handlers

import (
	"fmt"
	"net/http"

	cvn "github.com/epicoon/lxgo/auth/internal/conventions"
	"github.com/epicoon/lxgo/auth/internal/models"
	"github.com/epicoon/lxgo/kernel"
	lxHttp "github.com/epicoon/lxgo/kernel/http"
	"github.com/epicoon/lxgo/session"
)

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * AuthRequest
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */
/** @interface kernel.IForm */
type AuthRequest struct {
	*lxHttp.Form
	ResponseType string `json:"response_type"`
	ClientID     uint   `json:"client_id"`
	RedirectUri  string `json:"redirect_uri"`
	State        string `json:"state"`
	Scope        string `json:"scope"`
}

func (f *AuthRequest) Config() kernel.FormConfig {
	return kernel.FormConfig{
		"response_type": kernel.FormFieldConfig{
			Description: "authentication type, available values: 'code'",
			Required:    true,
		},
		"client_id": kernel.FormFieldConfig{
			Description: "client application identifier",
			Required:    true,
		},
		"redirect_uri": kernel.FormFieldConfig{
			Description: "where to redirect the user after authorization; must exactly match the URI registered for this client, otherwise the request is rejected with 400 Bad Request",
			Required:    true,
		},
		"state": kernel.FormFieldConfig{
			Description: "unique string for protection against CSRF - generated and saved (in the session) by the state-generation endpoint, checked here when returning from the authorization form",
			Required:    true,
		},
		"scope": kernel.FormFieldConfig{
			Description: "requested access level: 'profile' or 'profile:data' (see README for the current fixed set); optional, defaults to the narrowest one ('profile') if omitted",
			Required:    false,
		},
	}
}

/** @constructor */
func NewAuthRequest() kernel.IForm {
	return lxHttp.PrepareForm(&AuthRequest{Form: lxHttp.NewForm()})
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * AuthHandler
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */
/** @interface kernel.IHttpResource */
type AuthHandler struct {
	*lxHttp.Resource
	Receiver bool
}

/** @constructor kernel.CHttpResource */
func NewGetAuthHandler() kernel.IHttpResource {
	h := &AuthHandler{Resource: lxHttp.NewResource()}
	h.Receiver = false
	return h
}

/** @constructor kernel.CHttpResource */
func NewPostAuthHandler() kernel.IHttpResource {
	h := &AuthHandler{Resource: lxHttp.NewResource(kernel.HttpResourceConfig{
		CRequestForm: NewAuthRequest,
	})}
	h.Receiver = true
	return h
}

// ProcessRequestErrors only ever fires for the POST/Receiver path - GET
// doesn't have CRequestForm set, so the router never calls this for it.
func (handler *AuthHandler) ProcessRequestErrors() kernel.IHttpResponse {
	return handler.ErrorResponse(
	    http.StatusBadRequest,
	    fmt.Sprintf("Invalid request: %s", handler.RequestForm().GetFirstError()),
    )
}

func (handler *AuthHandler) Run() kernel.IHttpResponse {
	coreApp, ok := handler.App().(cvn.IApp)
	if !ok {
		panic("app does not implement core.IApp")
	}

	ctx := handler.Context()
	w := handler.ResponseWriter()

	var params *AuthParams
	if handler.Receiver {
		req := handler.RequestForm().(*AuthRequest)

		scope := req.Scope
		if scope == "" {
			scope = models.DefaultScope
		} else if !models.ValidateScope(scope) {
			http.Error(w, "Invalid scope", http.StatusBadRequest)
			return nil
		}

		params = &AuthParams{
			ResponseType: req.ResponseType,
			ClientID:     req.ClientID,
			RedirectUri:  req.RedirectUri,
			State:        req.State,
			Scope:        scope,
		}

		sess, err := session.ExtractSession(ctx)
		if err != nil {
			handler.LogError("Server configuration is wrong: sessions are required", "App")
			http.Error(w, "Something went wrong", http.StatusInternalServerError)
			return nil
		}

		sess.SetForce("lxgo_auth_params", params)
	} else {
		sess, err := session.ExtractSession(ctx)
		if err != nil {
			handler.LogError("Server configuration is wrong: sessions are required", "App")
			http.Error(w, "Something went wrong", http.StatusInternalServerError)
			return nil
		}

		p := sess.Get("lxgo_auth_params")
		ap, ok := p.(*AuthParams)
		if !ok {
			http.Error(w, "Don't have auth data", http.StatusBadRequest)
			return nil
		}
		params = ap
	}

	client, err := coreApp.ClientsRepo().FindByID(params.ClientID)
	if err != nil {
		http.Error(w, "Client does not exist", http.StatusBadRequest)
		return nil
	}

	// Exact match against the client's registered redirect_uri - the safer
	// of the two per RFC 6749, and avoids bypasses via path traversal/extra
	// segments on an allowed domain.
	if params.RedirectUri != client.RedirectUri {
		http.Error(w, "Invalid redirect_uri", http.StatusBadRequest)
		return nil
	}

	return handler.HtmlResponse(kernel.HtmlResponseConfig{
		Template: "form",
		Params: struct {
			Title string
		}{
			Title: "Auth",
		},
	})
}
