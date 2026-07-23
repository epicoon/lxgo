package handlers

import (
	"errors"
	"fmt"
	"net/http"

	cvn "github.com/epicoon/lxgo/auth/internal/conventions"
	"github.com/epicoon/lxgo/auth/internal/repos"
	"github.com/epicoon/lxgo/kernel"
	lxHttp "github.com/epicoon/lxgo/kernel/http"
)

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * DeleteClientRequest
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */
/** @interface kernel.IForm */
type DeleteClientRequest struct {
	*lxHttp.Form
	ID uint `json:"id"`
}

var _ kernel.IForm = (*DeleteClientRequest)(nil)

func (f *DeleteClientRequest) Config() kernel.FormConfig {
	return kernel.FormConfig{
		"id": kernel.FormFieldConfig{
			Description: "identifier of the client to delete",
			Required:    true,
		},
	}
}

/** @constructor */
func NewDeleteClientRequest() kernel.IForm {
	return lxHttp.PrepareForm(&DeleteClientRequest{Form: lxHttp.NewForm()})
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * DeleteClientHandler
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */
// DeleteClientHandler is the first admin-gated endpoint - requires a token issued by the configured admin
// client (see authenticateAdmin, common.go) for a User who has an Admin record; any admin role is enough
// for this one action.
/** @interface kernel.IHttpResource */
type DeleteClientHandler struct {
	*BaseHandler
}

var _ kernel.IHttpResource = (*DeleteClientHandler)(nil)

/** @constructor kernel.CHttpResource */
func NewDeleteClientHandler() kernel.IHttpResource {
	return &DeleteClientHandler{BaseHandler: NewBaseHandler(kernel.HttpResourceConfig{
		CRequestForm: NewDeleteClientRequest,
	})}
}

func (handler *DeleteClientHandler) Run() kernel.IHttpResponse {
	coreApp, ok := handler.App().(cvn.IApp)
	if !ok {
		panic("app does not implement core.IApp")
	}

	if _, errResp := authenticateAdmin(coreApp, handler); errResp != nil {
		return errResp
	}

	req := handler.RequestForm().(*DeleteClientRequest)

	if err := coreApp.ClientsRepo().Delete(req.ID); err != nil {
		if errors.Is(err, repos.ErrClientNotFound) {
			return errorResponse(handler, http.StatusNotFound, ERR_CLIENT_NOT_FOUND, "Client not found")
		}
		return serverErrorResponse(handler, fmt.Sprintf("can not delete client id=%d: %s", req.ID, err))
	}

	return handler.JsonResponse(kernel.JsonResponseConfig{
		Form: NewSuccessResponse(),
	})
}
