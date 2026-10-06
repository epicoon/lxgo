package http

import (
	"net/http"
	"path/filepath"

	"github.com/epicoon/lxgo/kernel"
)

// assetPathParam is the wildcard path-segment name a RegisterFileAssets
// route captures - see newAssetHandler.
const assetPathParam = "path"

// assetHandler serves one file under dir, at the path captured by the
// route's own "*{path}" segment (see RegisterFileAssets).
/** @interface kernel.IHttpResource */
type assetHandler struct {
	*Resource
	app kernel.IApp
	dir string
}

var _ kernel.IHttpResource = (*assetHandler)(nil)

/** @constructor kernel.CHttpResource */
func newAssetHandler(app kernel.IApp, dir string) kernel.CHttpResource {
	return func() kernel.IHttpResource {
		return &assetHandler{
			Resource: NewResource(),
			app:      app,
			dir:      dir,
		}
	}
}

func (h *assetHandler) Run() kernel.IHttpResponse {
	relPath := h.PathSegments()[assetPathParam]

	var filePath string
	if h.app == nil {
		filePath = filepath.Join(h.dir, relPath)
	} else {
		filePath = filepath.Join(h.app.Pathfinder().GetAbsPath(h.dir), relPath)
		h.app.Events().Trigger(kernel.EVENT_APP_BEFORE_SEND_ASSET, kernel.Dict{
			"request": h.Request(),
			"file":    filePath,
		})
	}

	http.ServeFile(h.ResponseWriter(), h.Request(), filePath)
	return nil
}
