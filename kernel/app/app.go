package app

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/config"
	"github.com/epicoon/lxgo/kernel/events"
	lxHttp "github.com/epicoon/lxgo/kernel/http"
	"github.com/epicoon/lxgo/kernel/template"
)

/** @interface kernel.IApp */
type App struct {
	port        int
	pathfinder  kernel.IPathfinder
	config      *kernel.Config
	components  map[any]kernel.IAppComponent
	diContainer kernel.IDIContainer
	connection  kernel.IConnection
	router      kernel.IRouter
	tplHolder   kernel.ITemplateHolder
	events      kernel.IEventManager
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

func InitApp(app kernel.IApp, c *kernel.Config) error {
	port, err := config.GetParam[int](c, "Port")
	if err != nil {
		return fmt.Errorf("can not create new application: %s", err)
	}

	app.SetPort(port)
	app.SetConfig(c)

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

func (app *App) SetPort(p int) {
	app.port = p
}

func (app *App) SetConfig(c *kernel.Config) {
	app.config = c
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

func (app *App) Config() *kernel.Config {
	return app.config
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
	if app.HasComponent("Logger") {
		l, ok := app.Component("Logger").(kernel.ILogger)
		if ok {
			l.Log(msg, category)
			return
		}
	}
	log.Println(msg)
}

func (app *App) LogWarning(msg string, category string) {
	if app.HasComponent("Logger") {
		l, ok := app.Component("Logger").(kernel.ILogger)
		if ok {
			l.LogWarning(msg, category)
			return
		}
	}
	log.Println(msg)
}

func (app *App) LogError(msg string, category string) {
	if app.HasComponent("Logger") {
		l, ok := app.Component("Logger").(kernel.ILogger)
		if ok {
			l.LogError(msg, category)
			return
		}
	}
	log.Println(msg)
}

func (app *App) Run() {
	app.Events().Trigger(kernel.EVENT_APP_BEFORE_RUN)

	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Panic: %s\n", r)
			return
		}
	}()

	fmt.Printf("Start a new application on port %d\n", app.port)

	app.router.Start()

	if err := http.ListenAndServe(":"+strconv.Itoa(app.port), nil); err != nil {
		log.Fatalf("Could not start server: %s\n", err.Error())
	}
}

func (app *App) Final() {
	app.connection.Close()
}
