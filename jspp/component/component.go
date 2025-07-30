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
type JSPreprocessor struct {
	*lxApp.AppComponent

	pf kernel.IPathfinder
	mm cnv.IModulesMap
	pm cnv.IPluginManager
}

/** @interface */
var _ cnv.IPreprocessor = (*JSPreprocessor)(nil)

func SetAppComponent(app kernel.IApp, configKey string) error {
	if app.HasComponent(cnv.APP_COMPONENT_KEY) {
		return fmt.Errorf("the application already has component: %s", cnv.APP_COMPONENT_KEY)
	}

	pp := NewJSPreprocesor()
	err := lxApp.InitComponent(pp, app, configKey)
	if err != nil {
		return fmt.Errorf("can not init js-preprocessor component: %s", err)
	}

	pp.pf = utils.NewPathfinder(pp)
	app.SetComponent(cnv.APP_COMPONENT_KEY, pp)
	return nil
}

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
func NewJSPreprocesor() *JSPreprocessor {
	pp := &JSPreprocessor{AppComponent: lxApp.NewAppComponent()}
	pp.mm = modules.NewMap(pp)
	pp.pm = plugins.NewMap(pp)
	return pp
}

func (pp *JSPreprocessor) Name() string {
	return "JSPreprocessor"
}

func (pp *JSPreprocessor) CConfig() kernel.CAppComponentConfig {
	return base.NewJSPreprocessorConfig
}

func (pp *JSPreprocessor) Config() *base.JSPreprocessorConfig {
	return (pp.GetConfig()).(*base.JSPreprocessorConfig)
}

func (pp *JSPreprocessor) Pathfinder() kernel.IPathfinder {
	return pp.pf
}

func (pp *JSPreprocessor) ModulesMap() cnv.IModulesMap {
	return pp.mm
}

func (pp *JSPreprocessor) PluginManager() cnv.IPluginManager {
	return pp.pm
}

func (pp *JSPreprocessor) CompilerBuilder() cnv.ICompilerBuilder {
	return compiler.Builder().
		SetPreprocessor(pp)
}

func (pp *JSPreprocessor) ExecutorBuilder() cnv.IExecutorBuilder {
	return executor.Builder().
		SetPreprocessor(pp)
}

func (pp *JSPreprocessor) AfterInit() {
	pp.App().Router().RegisterResources(kernel.HttpResourcesList{
		"/lx_service[POST]": handlers.NewServiceHandler,
	})

	pp.App().Router().AddMiddleware(func(ctx kernel.IHandleContext) error {
		if ctx.Route() == "/lx_service" {
			ctx.Set("jspp", pp)
		}
		return nil
	})

	pp.App().Events().Subscribe(kernel.EVENT_APP_BEFORE_SEND_ASSET, func(e kernel.IEvent) {
		filePath := e.Payload().Get("file").(string)

		lang, err := lxHttp.Lang(e.Payload().Get("request").(*http.Request))
		if err != nil {
			pp.LogError("Error while getting 'lxlang' cookie for EVENT_APP_BEFORE_SEND_ASSET: %v", err)
		}

		tb := utils.NewTargetBuilder(pp, filePath, lang)
		tb.Build()
	})
}

func (pp *JSPreprocessor) LogError(msg string, params ...any) {
	if len(params) > 0 {
		pp.App().LogError(fmt.Sprintf(msg, params...), "JSPreprocessor")
	} else {
		pp.App().LogError(msg, "JSPreprocessor")
	}
}
