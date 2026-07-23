package handlers

import (
	"fmt"

	cvn "github.com/epicoon/lxgo/auth/internal/conventions"
	"github.com/epicoon/lxgo/auth/internal/models"
	"github.com/epicoon/lxgo/kernel"
	lxHttp "github.com/epicoon/lxgo/kernel/http"
)

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * CreateClientRequest
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */
/** @interface kernel.IForm */
type CreateClientRequest struct {
	*lxHttp.Form
	RedirectUri string `json:"redirect_uri"`
}

var _ kernel.IForm = (*CreateClientRequest)(nil)

func (f *CreateClientRequest) Config() kernel.FormConfig {
	return kernel.FormConfig{
		"redirect_uri": kernel.FormFieldConfig{
			Description: "where to redirect the user after authorization; the exact value this client will send to /auth",
			Required:    true,
		},
	}
}

/** @constructor */
func NewCreateClientRequest() kernel.IForm {
	return lxHttp.PrepareForm(&CreateClientRequest{Form: lxHttp.NewForm()})
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * CreateClientResponse
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */
/** @interface kernel.IForm */
type CreateClientResponse struct {
	*lxHttp.Form
	Success bool   `json:"success"`
	ID      uint   `json:"id"`
	Secret  string `json:"secret"`
}

var _ kernel.IForm = (*CreateClientResponse)(nil)

func (f *CreateClientResponse) Config() kernel.FormConfig {
	return kernel.FormConfig{
		"success": kernel.FormFieldConfig{
			Description: "equals true",
			Required:    true,
		},
		"id": kernel.FormFieldConfig{
			Description: "new client's identifier",
			Required:    true,
		},
		"secret": kernel.FormFieldConfig{
			Description: "new client's secret - shown once here, store it now",
			Required:    true,
		},
	}
}

/** @constructor */
func NewCreateClientResponse() kernel.IForm {
	return lxHttp.PrepareForm(&CreateClientResponse{
		Form:    lxHttp.NewForm(),
		Success: true,
	})
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * CreateClientHandler
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */
// CreateClientHandler is intentionally open (no auth) - self-service OAuth
// client registration, same as e.g. Google/GitHub let any developer register
// an app.
/** @interface kernel.IHttpResource */
type CreateClientHandler struct {
	*BaseHandler
}

var _ kernel.IHttpResource = (*CreateClientHandler)(nil)

/** @constructor kernel.CHttpResource */
func NewCreateClientHandler() kernel.IHttpResource {
	return &CreateClientHandler{BaseHandler: NewBaseHandler(kernel.HttpResourceConfig{
		CRequestForm:  NewCreateClientRequest,
		CResponseForm: NewCreateClientResponse,
	})}
}

func (handler *CreateClientHandler) Run() kernel.IHttpResponse {
	coreApp, ok := handler.App().(cvn.IApp)
	if !ok {
		panic("app does not implement core.IApp")
	}

	req := handler.RequestForm().(*CreateClientRequest)

	client := &models.Client{
		AccessTokenLifetime:  models.DefaultAccessTokenLifetime,
		RefreshTokenLifetime: models.DefaultRefreshTokenLifetime,
		RedirectUri:          req.RedirectUri,
	}

	secret, err := coreApp.ClientsRepo().Create(client)
	if err != nil {
		return serverErrorResponse(handler, fmt.Sprintf("can not create client: %s", err))
	}

	return handler.JsonResponse(kernel.JsonResponseConfig{
		Dict: kernel.Dict{
			"id":     client.ID,
			"secret": secret,
		},
	})
}
