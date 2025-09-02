package http

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/epicoon/lxgo/kernel"
)

/** @interface kernel.IHttpResource */
type proxyHandler struct {
	*Resource
}

/** @constructor kernel.CHttpResource */
func newProxyHandler() kernel.IHttpResource {
	return &proxyHandler{Resource: NewResource()}
}

func (h *proxyHandler) Run() kernel.IHttpResponse {
	server := h.Context().Get("Server").(string)

	target, err := url.Parse(server)
	if err != nil {
		h.LogError(fmt.Sprintf("can not make proxy request to '%s': %v", server, err), "Handling")
		return h.JsonResponse(kernel.JsonResponseConfig{
			Code: http.StatusInternalServerError,
			Dict: kernel.Dict{"details": "something went wrong"},
		})
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	if h.Context().Has("Path") {
		path := h.Context().Get("Path").(string)
		proxy.Director = func(req *http.Request) {
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host
			req.URL.Path = path
			req.Host = target.Host
		}
	}

	proxy.ServeHTTP(h.ResponseWriter(), h.Request())

	return nil
}
