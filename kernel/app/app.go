package app

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"strconv"

	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/config"
	"github.com/epicoon/lxgo/kernel/events"
	lxHttp "github.com/epicoon/lxgo/kernel/http"
	"github.com/epicoon/lxgo/kernel/template"
)

/** @interface kernel.IApp */
type App struct {
	port         int
	pathfinder   kernel.IPathfinder
	config       *kernel.Config
	manageSocket *manageSocket
	components   map[any]kernel.IAppComponent
	logger       kernel.ILogger
	diContainer  kernel.IDIContainer
	connection   kernel.IConnection
	router       kernel.IRouter
	tplHolder    kernel.ITemplateHolder
	events       kernel.IEventManager
}

/** @constructor */
func NewApp() *App {
	app := &App{}
	app.pathfinder = NewAppPathfinder(app)
	app.diContainer = NewDIConteiner(app)
	app.tplHolder = template.NewTemplateHolder(app)
	app.events = events.NewEventManager(app)
	return app
}

func Configurate(app kernel.IApp) error {
	path := app.ConfigPath()
	if path == "" {
		return errors.New("unknown configuration file path")
	}

	conf, err := config.Load(app.Pathfinder().GetAbsPath(path))
	if err != nil {
		return fmt.Errorf("can not read application config. Cause: %v", err)
	}

	if err := InitApp(app, conf); err != nil {
		return fmt.Errorf("can not init application: %v", err)
	}

	return nil
}

func InitApp(app kernel.IApp, c *kernel.Config) error {
	port, err := config.GetParam[int](c, "Port")
	if err != nil {
		return fmt.Errorf("can not create new application: %s", err)
	}

	app.SetPort(port)
	app.SetConfig(c)

	if config.HasParam(c, "ManageSocket") {
		a, ok := app.BaseApp().(*App)
		if ok {
			a.manageSocket, err = newManageSocket(app)
			if err != nil {
				return fmt.Errorf("can not create manage socket: %s", err)
			}
		}
	}

	if config.HasParam(c, "Database") {
		dbConf, err := config.GetParam[kernel.Config](c, "Database")
		if err != nil {
			return fmt.Errorf("can not read Database config: %s", err)
		}
		app.SetConnection(NewConnection())
		app.Connection().SetApp(app)
		app.Connection().SetConfig(&dbConf)
	}

	app.SetRouter(lxHttp.NewRouter(app))
	return nil
}

func (app *App) BaseApp() kernel.IApp {
	return app
}

func (app *App) ConfigPath() string {
	// abstract
	return ""
}

func (app *App) SetPort(p int) {
	app.port = p
}

func (app *App) SetConfig(c *kernel.Config) {
	app.config = c
}

func (app *App) SetConfigParam(key string, val any) {
	if app.config == nil {
		return
	}

	conf := *app.config
	oldVal, exists := conf[key]
	if !exists {
		conf[key] = val
		return
	}

	oldType := reflect.TypeOf(oldVal)
	newType := reflect.TypeOf(val)
	if oldType == newType {
		conf[key] = val
		return
	}

	switch oldVal.(type) {
	case int:
		switch v := val.(type) {
		case string:
			if num, err := strconv.Atoi(v); err == nil {
				conf[key] = num
				return
			}
		case int64:
			conf[key] = int(v)
			return
		case float64:
			conf[key] = int(v)
			return
		}
	case float64:
		switch v := val.(type) {
		case string:
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				conf[key] = f
				return
			}
		case int:
			conf[key] = float64(v)
			return
		}
	case bool:
		switch v := val.(type) {
		case string:
			if b, err := strconv.ParseBool(v); err == nil {
				conf[key] = b
				return
			}
		}
	case string:
		conf[key] = fmt.Sprintf("%v", val)
		return
	}

	app.LogWarning(fmt.Sprintf(
		"Config param '%s' type mismatch: old=%T, new=%T — not replaced",
		key, oldVal, val,
	), "Config")
}

func (app *App) Config() *kernel.Config {
	return app.config
}

func (app *App) SetComponent(key any, c kernel.IAppComponent) {
	if app.components == nil {
		app.components = make(map[any]kernel.IAppComponent)
	}
	app.components[key] = c
}

func (app *App) HasComponent(key any) bool {
	_, exists := app.components[key]
	return exists
}

func (app *App) Component(key any) kernel.IAppComponent {
	c, exists := app.components[key]
	if !exists {
		return nil
	}
	return c
}

func (app *App) SetConnection(c kernel.IConnection) {
	app.connection = c
}

func (app *App) SetRouter(r kernel.IRouter) {
	app.router = r
}

func (app *App) Pathfinder() kernel.IPathfinder {
	return app.pathfinder
}

func (app *App) DIContainer() kernel.IDIContainer {
	return app.diContainer
}

func (app *App) Router() kernel.IRouter {
	return app.router
}

func (app *App) TemplateHolder() kernel.ITemplateHolder {
	return app.tplHolder
}

func (app *App) TemplateRenderer() kernel.ITemplateRenderer {
	return app.tplHolder.TemplateRenderer()
}

func (app *App) Events() kernel.IEventManager {
	return app.events
}

func (app *App) Connection() kernel.IConnection {
	return app.connection
}

func (app *App) Log(msg string, category string) {
	if app.logger != nil {
		app.logger.Log(msg, category)
		return
	}
	log.Println("[" + category + "]" + " " + msg)
}

func (app *App) LogWarning(msg string, category string) {
	if app.logger != nil {
		app.logger.LogWarning(msg, category)
		return
	}
	log.Println("[" + category + ": warning]" + " " + msg)
}

func (app *App) LogError(msg string, category string) {
	if app.logger != nil {
		app.logger.LogError(msg, category)
		return
	}
	log.Println("[" + category + ": error]" + " " + msg)
}

func (app *App) Logger() kernel.ILogger {
	return app.logger
}

func (app *App) SetLogger(l kernel.ILogger) {
	app.logger = l
}

func (app *App) Run() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Panic: %s\n", r)
			return
		}
	}()

	fmt.Printf("Start a new application on port %d\n", app.port)

	if app.manageSocket != nil {
		if err := app.manageSocket.Run(); err != nil {
			log.Fatalf("Manage socket failed: %v", err)
		}
	}

	app.router.Start()

	if err := http.ListenAndServe(":"+strconv.Itoa(app.port), nil); err != nil {
		log.Fatalf("Could not start server: %s\n", err.Error())
	}

	for _, c := range app.components {
		if err := c.Run(); err != nil {
			log.Fatalf("Could not start app component '%s': %s\n", c.Name(), err.Error())
		}
	}
}

func (app *App) Final() {
	if app.connection != nil {
		app.connection.Close()
	}

	if app.manageSocket != nil {
		app.manageSocket.Final()
	}

	for _, c := range app.components {
		if err := c.Final(); err != nil {
			app.LogError(fmt.Sprintf("Could not finish app component '%s': %v\n", c.Name(), err.Error()), "App")
		}
	}
}
