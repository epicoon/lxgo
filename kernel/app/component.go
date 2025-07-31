package app

import (
	"fmt"
	"strings"

	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/config"
	"github.com/epicoon/lxgo/kernel/conv"
)

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * ComponentConfig
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */
const compTypeStruct = 0
const compTypeMap = 1

/** @interface kernel.IAppComponentConfig */
type ComponentConfig struct {
	tp   int
	data map[string]any
}

/** @constructor */
func NewComponentConfigStruct() *ComponentConfig {
	return &ComponentConfig{tp: compTypeStruct}
}

/** @constructor */
func NewComponentConfigMap() *ComponentConfig {
	return &ComponentConfig{tp: compTypeMap}
}

func (cc *ComponentConfig) IsMap() bool {
	return cc.tp == compTypeMap
}

func (cc *ComponentConfig) Set(key string, val any) {
	if cc.data == nil {
		cc.data = map[string]any{}
	}
	cc.data[key] = val
}

func (cc *ComponentConfig) Has(key string) bool {
	_, ok := cc.data[key]
	return ok
}

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
type AppComponent struct {
	app    kernel.IApp
	config kernel.IAppComponentConfig
}

/** @constructor */
func NewAppComponent() *AppComponent {
	return &AppComponent{}
}

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

func InitComponent(c kernel.IAppComponent, app kernel.IApp, configKey string) error {
	c.SetApp(app)

	path := strings.Split(configKey, ".")
	conf := app.Config()
	for _, step := range path {
		tryConf, err := config.GetParam[kernel.Config](conf, step)
		if err != nil {
			return fmt.Errorf("can not init application component '%s': %s", c.Name(), err)
		}
		conf = &tryConf
	}

	cConf := c.CConfig()
	if cConf != nil {
		compConf := cConf()
		if compConf.IsMap() {
			//TODO
		} else {
			err := conv.DictToStruct((*kernel.Dict)(conf), compConf)
			if err != nil {
				return fmt.Errorf("can not set config for application component '%s': %s", c.Name(), err)
			}
		}
		c.SetConfig(compConf)
	}

	c.AfterInit()
	return nil
}

func (c *AppComponent) SetApp(app kernel.IApp) {
	c.app = app
}

func (c *AppComponent) SetConfig(conf kernel.IAppComponentConfig) {
	c.config = conf
}

func (c *AppComponent) GetConfig() kernel.IAppComponentConfig {
	return c.config
}

/** @abstract */
func (c *AppComponent) Name() string {
	// Pass
	return ""
}

func (c *AppComponent) App() kernel.IApp {
	return c.app
}

/** @abstract */
func (c *AppComponent) CConfig() kernel.CAppComponentConfig {
	// pass
	return nil
}

/** @abstract */
func (c *AppComponent) AfterInit() {
	// Pass
}
