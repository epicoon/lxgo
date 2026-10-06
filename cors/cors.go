package cors

import (
	"fmt"
	"strings"

	"github.com/epicoon/lxgo/kernel"
	lxApp "github.com/epicoon/lxgo/kernel/app"
)

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * Config
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

/** @interface kernel.IAppComponentConfig */

// Config is Cors's app-component configuration.
type Config struct {
	*lxApp.ComponentConfig
	// Origins maps an allowed Origin header value to its own settings - an
	// Origin not listed here never gets CORS headers, regardless of
	// EnableFor/EnableForAll.
	Origins map[string]OriginConfig
}

/** @constructor kernel.CAppComponentConfig */

// NewConfig constructs a Config.
func NewConfig() kernel.IAppComponentConfig {
	return &Config{ComponentConfig: lxApp.NewComponentConfigStruct()}
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * Cors
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

/** @interface kernel.IAppComponent */
/** @interface ICors */

// Cors is the default ICors implementation - see SetAppComponent to
// register it on an app.
type Cors struct {
	*lxApp.AppComponent

	enabledForAll  bool
	enabledRoutes  []string
	disabledRoutes []string
}

var _ ICors = (*Cors)(nil)

// SetAppComponent registers a new Cors on app under APP_COMPONENT_KEY,
// configured from the config section named by configKey.
func SetAppComponent(app kernel.IApp, configKey string) error {
	if app.HasComponent(APP_COMPONENT_KEY) {
		return fmt.Errorf("the application already has component: %s", APP_COMPONENT_KEY)
	}

	c := NewCors()
	if err := lxApp.InitComponent(c, app, configKey); err != nil {
		return fmt.Errorf("can not init cors component: %s", err)
	}

	app.SetComponent(APP_COMPONENT_KEY, c)
	return nil
}

// AppComponent returns the Cors registered on app under APP_COMPONENT_KEY.
func AppComponent(app kernel.IApp) (ICors, error) {
	c := app.Component(APP_COMPONENT_KEY)
	if c == nil {
		return nil, fmt.Errorf("application component '%s' not found", APP_COMPONENT_KEY)
	}

	cors, ok := c.(ICors)
	if !ok {
		return nil, fmt.Errorf("application component '%s' is not 'cors.ICors'", APP_COMPONENT_KEY)
	}

	return cors, nil
}

/** @constructor */

// NewCors constructs a Cors.
func NewCors() ICors {
	return &Cors{AppComponent: lxApp.NewAppComponent()}
}

// Name returns the component's name - see kernel.IAppComponent.
func (c *Cors) Name() string {
	return "Cors"
}

// LogCategory returns the category the component's log methods write under.
func (c *Cors) LogCategory() string {
	return "Cors"
}

// CConfig returns Config's constructor - see kernel.IAppComponent.
func (c *Cors) CConfig() kernel.CAppComponentConfig {
	return NewConfig
}

// Config returns the component's Config.
func (c *Cors) Config() *Config {
	return (c.GetConfig()).(*Config)
}

// AfterInit registers the header-adding middleware - see kernel.IAppComponent.
func (c *Cors) AfterInit() {
	c.App().Router().AddMiddleware(func(ctx kernel.IHandleContext) error {
		c.apply(ctx)
		return nil
	})
}

// EnableFor opts routes into CORS handling - see ICors.
func (c *Cors) EnableFor(routes []string) {
	c.enabledRoutes = append(c.enabledRoutes, routes...)
}

// EnableForAll opts every route into CORS handling - see ICors.
func (c *Cors) EnableForAll() {
	c.enabledForAll = true
}

// DisableFor excludes routes from CORS handling - see ICors.
func (c *Cors) DisableFor(routes []string) {
	c.disabledRoutes = append(c.disabledRoutes, routes...)
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * PRIVATE
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

// apply adds this request's CORS headers, if the route is opted in (and
// not excluded) and the request's Origin is in Config.Origins.
//
// Only simple-response headers are added (Access-Control-Allow-Origin,
// Access-Control-Expose-Headers) - this runs for every request, including
// an actual CORS preflight (OPTIONS with Access-Control-Request-Method),
// but doesn't recognize it as one yet.
//
// TODO preflight support: detect an OPTIONS request carrying
// Access-Control-Request-Method/-Headers, short-circuit it with a 204 and
// Access-Control-Allow-Methods/-Allow-Headers/-Max-Age instead of letting
// it reach the real resource.
func (c *Cors) apply(ctx kernel.IHandleContext) {
	route := ctx.Route()

	disabled := matchesAny(route, c.disabledRoutes)
	if disabled {
		return
	}

	enabled := c.enabledForAll || matchesAny(route, c.enabledRoutes)
	if !enabled {
		return
	}

	origin := ctx.Request().Header.Get("Origin")
	if origin == "" {
		return
	}

	originCfg, ok := c.Config().Origins[origin]
	if !ok {
		return
	}

	header := ctx.ResponseWriter().Header()
	header.Set("Access-Control-Allow-Origin", origin)
	if originCfg.ExposeHeaders != "" {
		header.Set("Access-Control-Expose-Headers", originCfg.ExposeHeaders)
	}
}

// matchesAny reports whether route equals, or starts with, any of prefixes.
func matchesAny(route string, prefixes []string) bool {
	for _, p := range prefixes {
		if route == p || strings.HasPrefix(route, p) {
			return true
		}
	}
	return false
}
