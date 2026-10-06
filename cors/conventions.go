// Package cors provides an optional, per-route CORS component for
// lxgo/kernel applications. Register it as an app component, configure
// which origins are allowed (Config.Origins), then opt specific routes
// into CORS handling via EnableFor/EnableForAll - nothing is CORS-enabled
// just because the component exists.
package cors

import "github.com/epicoon/lxgo/kernel"

// APP_COMPONENT_KEY is the key Cors registers itself under - see
// SetAppComponent/AppComponent.
const APP_COMPONENT_KEY = "lxgo_cors"

// ICors is the app component that adds CORS response headers to opted-in
// routes - see the package doc comment.
type ICors interface {
	kernel.IAppComponent

	// EnableFor opts routes (each an exact route string or a prefix, e.g.
	// "/img/" matching every path under it) into CORS handling.
	EnableFor(routes []string)

	// EnableForAll opts every route into CORS handling, without having to
	// list them individually.
	EnableForAll()

	// DisableFor excludes routes (each exact or a prefix, same matching as
	// EnableFor) from CORS handling, regardless of EnableForAll/EnableFor -
	// when a route matches both an enabled and a disabled entry, the
	// exclusion wins (the safer choice: when in doubt, don't hand out the
	// resource).
	DisableFor(routes []string)
}

// OriginConfig is one allowed origin's own settings.
type OriginConfig struct {
	// ExposeHeaders becomes the response's "Access-Control-Expose-Headers"
	// header, if set - which of the response's own headers client-side JS
	// is allowed to read, for a cross-origin request from this origin.
	ExposeHeaders string
}
