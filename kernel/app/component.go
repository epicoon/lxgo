package app

import (
	"fmt"
	"strings"

	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/cast"
	"github.com/epicoon/lxgo/kernel/config"
)

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * ComponentConfig
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */
const compTypeStruct = 0
const compTypeMap = 1

/** @interface kernel.IAppComponentConfig */

// ComponentConfig is the default kernel.IAppComponentConfig implementation
// - either a typed struct populated from YAML (NewComponentConfigStruct) or
// a plain key/value map (NewComponentConfigMap).
type ComponentConfig struct {
	tp   int
	data map[string]any
}

var _ kernel.IAppComponentConfig = (*ComponentConfig)(nil)

/** @constructor */

// NewComponentConfigStruct constructs a ComponentConfig meant to be
// populated by converting YAML into a typed struct that embeds it - see InitComponent.
func NewComponentConfigStruct() *ComponentConfig {
	return &ComponentConfig{tp: compTypeStruct}
}

/** @constructor */

// NewComponentConfigMap constructs a ComponentConfig backed by a plain key/value map.
func NewComponentConfigMap() *ComponentConfig {
	return &ComponentConfig{tp: compTypeMap}
}

// IsMap reports whether the config is map-backed (NewComponentConfigMap) rather than struct-backed.
func (cc *ComponentConfig) IsMap() bool {
	return cc.tp == compTypeMap
}

// Set sets a single config key.
func (cc *ComponentConfig) Set(key string, val any) {
	if cc.data == nil {
		cc.data = map[string]any{}
	}
	cc.data[key] = val
}

// Has reports whether key is set.
func (cc *ComponentConfig) Has(key string) bool {
	_, ok := cc.data[key]
	return ok
}

// Get returns the value of key, or nil.
func (cc *ComponentConfig) Get(key string) any {
	val, ok := cc.data[key]
	if ok {
		return val
	} else {
		return nil
	}
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * AppComponent
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */
/** @interface kernel.IAppComponent */

// AppComponent is the base kernel.IAppComponent implementation - embed it
// in your own component struct and override at least Name.
type AppComponent struct {
	app    kernel.IApp
	config kernel.IAppComponentConfig

	// self is the component's own outward-facing instance, bound by
	// InitComponent - see selfBinder and logCategory.
	self kernel.IAppComponent
}

// selfBinder lets InitComponent hand a component its own outward-facing
// instance back to itself. Go embedding has no virtual dispatch: a call to
// c.LogCategory() from a method defined on *AppComponent always resolves to
// *AppComponent's own LogCategory(), even when c is embedded in a struct
// that overrides it - the override is a distinct method promoted onto the
// outer type, never reached through the embedded receiver. Routing
// Log/LogWarning/LogError through the bound self (an interface value whose
// dynamic type is the outer struct) restores the override.
type selfBinder interface {
	setSelf(self kernel.IAppComponent)
}

func (c *AppComponent) setSelf(self kernel.IAppComponent) {
	c.self = self
}

// logCategory returns the category to log under: self.LogCategory() if
// InitComponent bound self (the usual case for any component set up via
// RegisterComponent/InitComponent), so an embedding struct's override is
// honored; falling back to c.LogCategory() otherwise.
func (c *AppComponent) logCategory() string {
	if c.self != nil {
		return c.self.LogCategory()
	}
	return c.LogCategory()
}

var _ kernel.IAppComponent = (*AppComponent)(nil)

/** @constructor */

// NewAppComponent constructs an empty AppComponent.
func NewAppComponent() *AppComponent {
	return &AppComponent{}
}

// RegisterComponent registers c on app under componentKey (failing if one's
// already registered there), initializing it from the config section named
// by configKey - see InitComponent.
func RegisterComponent(app kernel.IApp, c kernel.IAppComponent, componentKey, configKey string) error {
	if app.HasComponent(componentKey) {
		return fmt.Errorf("the application already has component: %s", componentKey)
	}

	if err := InitComponent(c, app, configKey); err != nil {
		return fmt.Errorf("can not init '%s': %s", componentKey, err)
	}

	app.SetComponent(componentKey, c)
	return nil
}

// InitComponent binds c to app, populates its config from the section named
// by configKey (dotted path into app's YAML config), and runs c.AfterInit -
// see RegisterComponent for the usual entry point that also registers c on app.
func InitComponent(c kernel.IAppComponent, app kernel.IApp, configKey string) error {
	c.SetApp(app)

	// c is the component's own outward-facing instance (e.g. *session.Storage,
	// not the *AppComponent it embeds) - handing it back to AppComponent lets
	// Log/LogWarning/LogError call LogCategory() through it, so an embedding
	// struct's override actually takes effect. See selfBinder.
	if binder, ok := c.(selfBinder); ok {
		binder.setSelf(c)
	}

	path := strings.Split(configKey, ".")
	var conf kernel.IDict = app.Config()
	for _, step := range path {
		tryConf, err := config.GetParam[kernel.Dict](conf, step)
		if err != nil {
			return fmt.Errorf("can not init application component '%s': %s", c.Name(), err)
		}
		conf = tryConf
	}

	cConf := c.CConfig()
	if cConf != nil {
		compConf := cConf()
		if compConf.IsMap() {
			//TODO
		} else {
			dict, ok := conf.(kernel.Dict)
			if !ok {
				return fmt.Errorf("can not set config for application component '%s': config is not a dict", c.Name())
			}
			if err := cast.DictToStruct(&dict, compConf); err != nil {
				return fmt.Errorf("can not set config for application component '%s': %s", c.Name(), err)
			}
		}
		c.SetConfig(compConf)
	}

	c.AfterInit()
	return nil
}

// SetApp binds the component to its owning app.
func (c *AppComponent) SetApp(app kernel.IApp) {
	c.app = app
}

// SetConfig sets the component's config.
func (c *AppComponent) SetConfig(conf kernel.IAppComponentConfig) {
	c.config = conf
}

// GetConfig returns the component's config.
// The "Get" prefix is used to make specific comp.Config() *CompConfig
// methods for inherited components
func (c *AppComponent) GetConfig() kernel.IAppComponentConfig {
	return c.config
}

// Log writes an informational message under LogCategory, formatting msg
// with params if any are given.
func (c *AppComponent) Log(msg string, params ...any) {
	if len(params) > 0 {
		c.App().Log(fmt.Sprintf(msg, params...), c.logCategory())
	} else {
		c.App().Log(msg, c.logCategory())
	}
}

// LogWarning writes a warning message under LogCategory, formatting msg
// with params if any are given.
func (c *AppComponent) LogWarning(msg string, params ...any) {
	if len(params) > 0 {
		c.App().LogWarning(fmt.Sprintf(msg, params...), c.logCategory())
	} else {
		c.App().LogWarning(msg, c.logCategory())
	}
}

// LogError writes an error message under LogCategory, formatting msg with
// params if any are given.
func (c *AppComponent) LogError(msg string, params ...any) {
	if len(params) > 0 {
		c.App().LogError(fmt.Sprintf(msg, params...), c.logCategory())
	} else {
		c.App().LogError(msg, c.logCategory())
	}
}

/** @abstract */

// LogCategory returns "AppComponent" - override it with a component-specific category.
func (c *AppComponent) LogCategory() string {
	return "AppComponent"
}

/** @abstract */

// Name returns "" - override it with the component's actual name.
func (c *AppComponent) Name() string {
	// Pass
	return ""
}

// App returns the owning app.
func (c *AppComponent) App() kernel.IApp {
	return c.app
}

/** @abstract */

// CConfig returns nil - override it if the component needs a config constructor.
func (c *AppComponent) CConfig() kernel.CAppComponentConfig {
	// pass
	return nil
}

/** @abstract */

// AfterInit is a no-op - override it to run setup once the component and its config are ready.
func (c *AppComponent) AfterInit() {
	// Pass
}

/** @abstract */

// Run is a no-op - override it if the component needs its own lifecycle.
func (c *AppComponent) Run() error {
	// Pass
	return nil
}

/** @abstract */

// Final is a no-op - override it to run shutdown/cleanup.
func (c *AppComponent) Final() error {
	// Pass
	return nil
}
