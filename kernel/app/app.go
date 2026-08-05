// Package app provides the default kernel.IApp implementation (App) and its
// supporting pieces - DB connection, pathfinder, DI container. Embed App in
// your own application struct, override ConfigPath, and call Configure to
// load config.yaml and wire everything up.
package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"runtime/debug"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/cast"
	"github.com/epicoon/lxgo/kernel/config"
	"github.com/epicoon/lxgo/kernel/events"
	lxHttp "github.com/epicoon/lxgo/kernel/http"
	"github.com/epicoon/lxgo/kernel/template"
)

// defaultShutdownTimeout is how long Run waits for in-flight requests to
// finish (via http.Server.Shutdown) after SIGINT/SIGTERM, before giving up
// and returning anyway - used unless the app's config sets its own
// "ShutdownTimeout" (whole seconds), see InitApp.
const defaultShutdownTimeout = 5 * time.Second

/** @interface kernel.IApp */

// App is the default kernel.IApp implementation - embed it in your own
// application struct and override at least ConfigPath.
type App struct {
	port            int
	pathfinder      kernel.IPathfinder
	config          kernel.IDict
	manageSocket    *manageSocket
	components      map[any]kernel.IAppComponent
	logger          kernel.ILogger
	diContainer     kernel.IDIContainer
	connection      kernel.IConnection
	router          kernel.IRouter
	tplHolder       kernel.ITemplateHolder
	events          kernel.IEventManager
	shutdownTimeout time.Duration
}

/** @constructor */

// NewApp constructs an App with its pathfinder, DI container, template
// holder and event manager ready to use.
func NewApp() *App {
	app := &App{}
	app.pathfinder = NewAppPathfinder(app)
	app.router = lxHttp.NewRouter(app)
	app.diContainer = NewDIContainer(app)
	app.tplHolder = template.NewTemplateHolder(app)
	app.events = events.NewEventManager(app)
	app.shutdownTimeout = defaultShutdownTimeout
	return app
}

// Configure loads config.yaml from app.ConfigPath() and initializes app
// via InitApp - call this once after constructing your application.
func Configure(app kernel.IApp) error {
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

// InitApp sets up app from an already-loaded config: port, optional manage
// socket, optional DB connection, and router - see Configure for the
// usual entry point that also loads the config file.
func InitApp(app kernel.IApp, c kernel.IDict) error {
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
		dbConf, err := config.GetParam[kernel.Dict](c, "Database")
		if err != nil {
			return fmt.Errorf("can not read Database config: %s", err)
		}
		app.SetConnection(NewConnection())
		app.Connection().SetApp(app)
		app.Connection().SetConfig(dbConf)
	}

	if config.HasParam(c, "ShutdownTimeout") {
		a, ok := app.BaseApp().(*App)
		if ok {
			seconds, err := config.GetParam[int](c, "ShutdownTimeout")
			if err != nil {
				return fmt.Errorf("can not read ShutdownTimeout config: %s", err)
			}
			a.shutdownTimeout = time.Duration(seconds) * time.Second
		}
	}

	return nil
}

// BaseApp returns app itself.
func (app *App) BaseApp() kernel.IApp {
	return app
}

// ConfigPath returns "" - override this in your embedding application struct.
func (app *App) ConfigPath() string {
	// abstract
	return ""
}

// SetPort overrides the port from config.
func (app *App) SetPort(p int) {
	app.port = p
}

// SetConfig replaces the application's config.
func (app *App) SetConfig(c kernel.IDict) {
	app.config = c
}

// SetConfigParam sets a single top-level config key, coercing val to the
// existing value's type when they differ (e.g. a string "42" into an int
// field); logs a warning and leaves the value unchanged if it can't coerce.
func (app *App) SetConfigParam(key string, val any) {
	if app.config == nil {
		return
	}

	if !app.config.Has(key) {
		app.config.Set(key, val)
		return
	}

	oldVal := app.config.Get(key)
	coerced, err := cast.Value(val, reflect.TypeOf(oldVal))
	if err != nil {
		app.LogWarning(fmt.Sprintf(
			"Config param '%s' type mismatch: old=%T, new=%T — not replaced",
			key, oldVal, val,
		), "Config")
		return
	}
	app.config.Set(key, coerced)
}

// ConfigParam returns a config value by dotted path (e.g. "Database.Host"),
// or nil if any segment is missing.
func (app *App) ConfigParam(key string) any {
	var conf kernel.IDict = app.Config()
	path := strings.Split(key, ".")
	for i, step := range path {
		if !config.HasParam(conf, step) {
			return nil
		}
		if i == len(path)-1 {
			val, _ := config.GetParam[any](conf, step)
			return val
		}
		tryConf, err := config.GetParam[kernel.Dict](conf, step)
		if err != nil {
			return nil
		}
		conf = tryConf
	}
	return nil
}

// Config returns the application's config.
func (app *App) Config() kernel.IDict {
	return app.config
}

// SetComponent registers a component under key.
func (app *App) SetComponent(key any, c kernel.IAppComponent) {
	if app.components == nil {
		app.components = make(map[any]kernel.IAppComponent)
	}
	app.components[key] = c
}

// HasComponent reports whether a component is registered under key.
func (app *App) HasComponent(key any) bool {
	_, exists := app.components[key]
	return exists
}

// Component returns the component registered under key, or nil.
func (app *App) Component(key any) kernel.IAppComponent {
	c, exists := app.components[key]
	if !exists {
		return nil
	}
	return c
}

// SetConnection sets the application's DB connection.
func (app *App) SetConnection(c kernel.IConnection) {
	app.connection = c
}

// Pathfinder returns the application's IPathfinder.
func (app *App) Pathfinder() kernel.IPathfinder {
	return app.pathfinder
}

// DIContainer returns the application's dependency-injection container.
func (app *App) DIContainer() kernel.IDIContainer {
	return app.diContainer
}

// Router returns the application's router.
func (app *App) Router() kernel.IRouter {
	return app.router
}

// TemplateHolder returns the application's ITemplateHolder.
func (app *App) TemplateHolder() kernel.ITemplateHolder {
	return app.tplHolder
}

// TemplateRenderer returns a fresh ITemplateRenderer.
func (app *App) TemplateRenderer() kernel.ITemplateRenderer {
	return app.tplHolder.TemplateRenderer()
}

// Events returns the application's event manager.
func (app *App) Events() kernel.IEventManager {
	return app.events
}

// Connection returns the application's DB connection.
func (app *App) Connection() kernel.IConnection {
	return app.connection
}

// Log writes an informational message under category, via the configured
// logger or the standard log package if none is set.
func (app *App) Log(msg string, category string) {
	if app.logger != nil {
		app.logger.Log(msg, category)
		return
	}
	log.Println("[" + category + "]" + " " + msg)
}

// LogWarning writes a warning message under category, via the configured
// logger or the standard log package if none is set.
func (app *App) LogWarning(msg string, category string) {
	if app.logger != nil {
		app.logger.LogWarning(msg, category)
		return
	}
	log.Println("[" + category + ": warning]" + " " + msg)
}

// LogError writes an error message under category, via the configured
// logger or the standard log package if none is set.
func (app *App) LogError(msg string, category string) {
	if app.logger != nil {
		app.logger.LogError(msg, category)
		return
	}
	log.Println("[" + category + ": error]" + " " + msg)
}

// Logger returns the application's ILogger, or nil if none is set.
func (app *App) Logger() kernel.ILogger {
	return app.logger
}

// SetLogger overrides the application's logger.
func (app *App) SetLogger(l kernel.ILogger) {
	app.logger = l
}

// Run starts the manage socket (if configured), the router, the HTTP
// server, and every registered component, then blocks until SIGINT/SIGTERM,
// at which point it gives the server up to shutdownTimeout to finish
// in-flight requests (http.Server.Shutdown) before returning. Run itself
// never calls Final - that stays the caller's job, so there is
// exactly one place that ever calls it, on a graceful shutdown or a
// recovered panic alike.
func (app *App) Run() {
	defer func() {
		if r := recover(); r != nil {
			app.events.Trigger(kernel.EVENT_APP_BEFORE_FAIL)
			fmt.Printf("Panic: %s\n", r)
			debug.PrintStack()
			return
		}
	}()

	// Registered first, before manage socket/router/server/component
	// startup - a signal arriving during that startup work must not fall
	// through to the OS's default disposition (process death without any
	// cleanup at all); it's fine for it to just sit satisfied on ctx until
	// the select below is reached, whenever that ends up being.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	fmt.Printf("Start a new application on port %d\n", app.port)

	if app.manageSocket != nil {
		if err := app.manageSocket.Run(); err != nil {
			fmt.Printf("Manage socket failed: %v\n", err)
			return
		}
	}

	app.router.Start()

	for _, c := range app.components {
		if err := c.Run(); err != nil {
			fmt.Printf("Could not start app component '%s': %s\n", c.Name(), err.Error())
			return
		}
	}

	srv := &http.Server{Addr: ":" + strconv.Itoa(app.port), Handler: nil}

	// Bind synchronously, so a port conflict is reported (and stops
	// startup) right here rather than racing the goroutine
	// below - only Serve, which blocks until Shutdown/Close, moves there.
	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		fmt.Printf("Could not start server: %s\n", err.Error())
		return
	}

	srvErr := make(chan error, 1)
	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			srvErr <- err
			return
		}
		srvErr <- nil
	}()

	select {
	case <-ctx.Done():
		fmt.Println("Shutting down...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), app.shutdownTimeout)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			fmt.Printf("Graceful shutdown failed: %s\n", err.Error())
		}
		<-srvErr // Serve has returned by now (Shutdown/Close unblocks it) - drain so the goroutine isn't left dangling
	case err := <-srvErr:
		if err != nil {
			fmt.Printf("Server error: %s\n", err.Error())
		}
	}
}

// Final fires EVENT_APP_BEFORE_FINAL, closes the DB connection, stops the
// manage socket, and finalizes every registered component.
func (app *App) Final() {
	app.events.Trigger(kernel.EVENT_APP_BEFORE_FINAL)

	if app.connection != nil {
		if err := app.connection.Close(); err != nil {
			app.LogError(fmt.Sprintf("Could not close app connection: %v", err), "App")
		}
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
