package handlers

import (
	"fmt"
	"net/http"

	cvn "github.com/epicoon/lxgo/auth/internal/conventions"
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
	ResponseType string `dict:"response_type"`
	ClientID     uint   `dict:"client_id"`
	RedirectUri  string `dict:"redirect_uri"`
	State        string `dict:"state"`
}

/** @constructor */
func NewAuthRequest() *AuthRequest {
	req := &AuthRequest{Form: lxHttp.NewForm()}
	req.SetRequired([]string{"response_type", "client_id", "redirect_uri", "state"})
	return req
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
	h := &AuthHandler{Resource: lxHttp.NewResource()}
	h.Receiver = true
	return h
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
		req := NewAuthRequest()
		lxHttp.FormFiller().SetContext(handler.Context()).SetForm(req).Fill()
		if req.HasErrors() {
			http.Error(w, fmt.Sprintf("Invalid request: %s", req.GetFirstError()), http.StatusBadRequest)
			return nil
		}

		params = &AuthParams{
			ResponseType: req.ResponseType,
			ClientID:     req.ClientID,
			RedirectUri:  req.RedirectUri,
			State:        req.State,
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

	//TODO check params.RedirectUri
	// --- client must have field ~ 'domain', for example:
	// --- --- domain == "http://localhost:8081"
	// --- --- params.RedirectUri == "http://localhost:8081/auth-callback"
	// --- --- --- need compare them

	if !coreApp.ClientsRepo().CheckIDExists(params.ClientID) {
		http.Error(w, "Client does not exist", http.StatusBadRequest)
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
