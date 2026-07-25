// Package component provides the default jspp.IPreprocessor implementation
// (JSPreprocessor) - register it on an application via SetAppComponent,
// then access it through AppComponent (or the "jspp" context key middleware
// sets on every /lx/... request).
package component

import (
	"fmt"
	"net/http"

	cnv "github.com/epicoon/lxgo/jspp"
	"github.com/epicoon/lxgo/jspp/internal/base"
	"github.com/epicoon/lxgo/jspp/internal/compiler"
	"github.com/epicoon/lxgo/jspp/internal/executor"
	"github.com/epicoon/lxgo/jspp/internal/handlers"
	"github.com/epicoon/lxgo/jspp/internal/modules"
	"github.com/epicoon/lxgo/jspp/internal/utils"
	"github.com/epicoon/lxgo/jspp/plugins"
	"github.com/epicoon/lxgo/kernel"
	lxApp "github.com/epicoon/lxgo/kernel/app"
	lxHttp "github.com/epicoon/lxgo/kernel/http"
)

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * JSPreprocessor
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

/** @interface kernel.IAppComponent */
/** @interface cnv.IPreprocessor */

// JSPreprocessor is the default jspp.IPreprocessor implementation - see SetAppComponent to register it on an app.
type JSPreprocessor struct {
	*lxApp.AppComponent

	pf kernel.IPathfinder
	mm cnv.IModulesMap
	pm cnv.IPluginManager
}

var _ cnv.IPreprocessor = (*JSPreprocessor)(nil)

// SetAppComponent registers a new JSPreprocessor on app under
// jspp.APP_COMPONENT_KEY, configured from the config section named by configKey.
func SetAppComponent(app kernel.IApp, configKey string) error {
	if app.HasComponent(cnv.APP_COMPONENT_KEY) {
		return fmt.Errorf("the application already has component: %s", cnv.APP_COMPONENT_KEY)
	}

	pp := NewJSPreprocessor()
	if err := lxApp.InitComponent(pp, app, configKey); err != nil {
		return fmt.Errorf("can not init js-preprocessor component: %s", err)
	}

	pp.pf = utils.NewPathfinder(pp)
	app.SetComponent(cnv.APP_COMPONENT_KEY, pp)
	return nil
}

// AppComponent returns the JSPreprocessor registered on app under jspp.APP_COMPONENT_KEY.
func AppComponent(app kernel.IApp) (*JSPreprocessor, error) {
	c := app.Component(cnv.APP_COMPONENT_KEY)
	if c == nil {
		return nil, fmt.Errorf("application component '%s' not found", cnv.APP_COMPONENT_KEY)
	}

	pp, ok := c.(*JSPreprocessor)
	if !ok {
		return nil, fmt.Errorf("application component '%s' is not '*JSPreprocessor'", cnv.APP_COMPONENT_KEY)
	}

	return pp, nil
}

/** @constructor */

// NewJSPreprocessor constructs a JSPreprocessor with its modules/plugin maps ready to use.
func NewJSPreprocessor() *JSPreprocessor {
	pp := &JSPreprocessor{AppComponent: lxApp.NewAppComponent()}
	pp.mm = modules.NewMap(pp)
	pp.pm = plugins.NewMap(pp)
	return pp
}

// Name returns the component's name - see kernel.IAppComponent.
func (c *JSPreprocessor) Name() string {
	return "JSPreprocessor"
}

// LogCategory returns the category the component's log methods write under.
func (pp *JSPreprocessor) LogCategory() string {
	return "JSPreprocessor"
}

// CConfig returns the preprocessor config's constructor - see kernel.IAppComponent.
func (pp *JSPreprocessor) CConfig() kernel.CAppComponentConfig {
	return base.NewJSPreprocessorConfig
}

// Config returns the component's config.
func (pp *JSPreprocessor) Config() *base.JSPreprocessorConfig {
	return (pp.GetConfig()).(*base.JSPreprocessorConfig)
}

// Pathfinder returns the preprocessor's own IPathfinder.
func (pp *JSPreprocessor) Pathfinder() kernel.IPathfinder {
	return pp.pf
}

// ModulesMap returns the preprocessor's modules map.
func (pp *JSPreprocessor) ModulesMap() cnv.IModulesMap {
	return pp.mm
}

// PluginManager returns the preprocessor's plugin manager.
func (pp *JSPreprocessor) PluginManager() cnv.IPluginManager {
	return pp.pm
}

// CompilerBuilder returns a fresh ICompilerBuilder, pre-bound to this preprocessor.
func (pp *JSPreprocessor) CompilerBuilder() cnv.ICompilerBuilder {
	return compiler.Builder().
		SetPreprocessor(pp)
}

// JSExecutorBuilder returns a fresh IJSExecutorBuilder, pre-bound to this preprocessor.
func (pp *JSPreprocessor) JSExecutorBuilder() cnv.IJSExecutorBuilder {
	return executor.Builder().
		SetPreprocessor(pp)
}

// AfterInit registers the /lx/service, /lx/elem and /lx/plugin routes and
// the asset-build hook - see kernel.IAppComponent.
func (pp *JSPreprocessor) AfterInit() {
	pp.App().Router().RegisterResources(kernel.HttpResourcesList{
		"/lx/service[POST]": handlers.NewServiceHandler,
		"/lx/elem[POST]":    handlers.NewElemHandler,
		"/lx/plugin[POST]":  handlers.NewPluginHandler,
	})
	pp.App().Router().AddMiddleware(func(ctx kernel.IHandleContext) error {
		list := []string{
			"/lx/service",
			"/lx/elem",
			"/lx/plugin",
		}
		for _, route := range list {
			if ctx.Route() == route {
				ctx.Set("jspp", pp)
			}
		}
		return nil
	})

	pp.App().Events().Subscribe(kernel.EVENT_APP_BEFORE_SEND_ASSET, func(e kernel.IEvent) {
		filePath := e.Payload().Get("file").(string)

		lang := lxHttp.Lang(pp.App(), e.Payload().Get("request").(*http.Request))

		tb := utils.NewTargetBuilder(pp, filePath, lang)
		if err := tb.Build(); err != nil {
			pp.LogError("can not build asset '%s': %v", filePath, err)
		}
	})
}
