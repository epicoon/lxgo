// Package kernel provides the core web-server application - components,
// routing and request handling - for lxgo-based applications. See the
// package README for a step-by-step tutorial; public subpackages (app,
// http, config, cast, errors, events, template, utils, cmd) extend this
// core with ready-made implementations and helpers.
package kernel

import (
	"database/sql"
	"net/http"
)

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * Type prefixes:
 * I - Interface
 * F - Function
 * C - Constructor
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * APP
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

// IDict is a generic string-keyed value-bag interface - implemented by Dict.
type IDict interface {
	// Set stores val under key.
	Set(key string, val any)

	// Get returns the value stored under key, or nil.
	Get(key string) any

	// Has reports whether key is set.
	Has(key string) bool

	// ToMap coerces to a plain map[string]any
	ToMap() map[string]any
}

// Dict is a generic string-keyed bag of values, implementing IDict - also
// used as the application's parsed configuration (from config.yaml), see
// IApp.Config. Must be created via make(Dict) or a literal before Set is
// called - a nil Dict panics on assignment, same as a nil map.
type Dict map[string]any

// IApp is the web-server application itself - owns the DB connection,
// router, components, DI container, templates and events, and drives the
// request/response lifecycle. See app.App for the default implementation.
type IApp interface {
	// BaseApp returns the underlying *app.App - useful when IApp is wrapped.
	BaseApp() IApp

	// SetPort overrides the port from config.
	SetPort(p int)

	// ConfigPath returns the path config.yaml was loaded from.
	ConfigPath() string

	// SetConfig replaces the application's config.
	SetConfig(c IDict)

	// SetConfigParam sets a single top-level config key.
	SetConfigParam(key string, val any)

	// ConfigParam returns a config value by dotted path (e.g. "Database.Host"),
	// or nil if any segment is missing.
	ConfigParam(key string) any

	// Config returns the application's config.
	Config() IDict

	// SetComponent registers a component under key.
	SetComponent(key any, c IAppComponent)

	// HasComponent reports whether a component is registered under key.
	HasComponent(key any) bool

	// Component returns the component registered under key, or nil.
	Component(key any) IAppComponent

	// SetConnection sets the application's DB connection.
	SetConnection(c IConnection)

	// SetRouter sets the application's router.
	SetRouter(r IRouter)

	// Pathfinder returns the application's IPathfinder.
	Pathfinder() IPathfinder

	// DIContainer returns the application's dependency-injection container.
	DIContainer() IDIContainer

	// Connection returns the application's DB connection.
	Connection() IConnection

	// Router returns the application's router.
	Router() IRouter

	// TemplateHolder returns the application's ITemplateHolder.
	TemplateHolder() ITemplateHolder

	// TemplateRenderer returns a fresh ITemplateRenderer.
	TemplateRenderer() ITemplateRenderer

	// Events returns the application's event manager.
	Events() IEventManager

	// Log writes an informational message under category.
	Log(msg string, category string)

	// LogWarning writes a warning message under category.
	LogWarning(msg string, category string)

	// LogError writes an error message under category.
	LogError(msg string, category string)

	// Logger returns the application's ILogger.
	Logger() ILogger

	// SetLogger overrides the application's logger.
	SetLogger(ILogger)

	// Run starts the application (blocks serving requests).
	Run()

	// Final runs the application's shutdown/cleanup.
	Final()
}

// CAppComponentConfig constructs an IAppComponentConfig - see IAppComponent.CConfig.
type CAppComponentConfig func() IAppComponentConfig

// IAppComponent is a pluggable piece of application functionality,
// registered on an IApp under a key (e.g. session storage, an auth client) -
// see IApp.SetComponent/Component.
type IAppComponent interface {
	// SetApp binds the component to its owning app.
	SetApp(app IApp)

	// SetConfig sets the component's config.
	SetConfig(conf IAppComponentConfig)

	// GetConfig returns the component's config.
	GetConfig() IAppComponentConfig

	// Name returns the component's name, used in logging.
	Name() string

	// App returns the owning app.
	App() IApp

	// CConfig returns the component's config constructor.
	CConfig() CAppComponentConfig

	// AfterInit runs once the component and its config are fully set up -
	// register routes/middleware/event handlers here.
	AfterInit()

	// LogCategory returns the category the component's log methods write under.
	LogCategory() string

	// Log writes an informational message under LogCategory.
	Log(msg string, params ...any)

	// LogWarning writes a warning message under LogCategory.
	LogWarning(msg string, params ...any)

	// LogError writes an error message under LogCategory.
	LogError(msg string, params ...any)

	// Run starts the component, if it needs its own lifecycle (e.g. a
	// background server).
	Run() error

	// Final runs the component's shutdown/cleanup.
	Final() error
}

// IAppComponentConfig is an IAppComponent's configuration - either a
// section of the app's YAML config (IsMap true) or a hand-built value.
type IAppComponentConfig interface {
	// IsMap reports whether the config was parsed from a YAML mapping.
	IsMap() bool

	// Set sets a single config key.
	Set(key string, val any)

	// Has reports whether key is set.
	Has(key string) bool

	// Get returns the value of key, or nil if it isn't set.
	Get(key string) any
}

// IPathfinder resolves paths relative to the application's root directory.
type IPathfinder interface {
	// GetRoot returns the application's root directory.
	GetRoot() string

	// GetAbsPath resolves path against the application's root directory.
	GetAbsPath(path string) string
}

// IConnection is the application's database connection.
type IConnection interface {
	// SetApp binds the connection to its owning app.
	SetApp(app IApp)

	// SetConfig sets the connection's config.
	SetConfig(cfg IDict)

	// DB returns the underlying *sql.DB.
	DB() *sql.DB

	// Connect opens the connection.
	Connect() error

	// Close closes the connection.
	Close() error
}

// CAnyList maps names to factory functions - see IDIContainer.
type CAnyList map[string]func(...any) any

// IDIContainer is a simple dependency-injection container: register named
// factories, then resolve values by name.
type IDIContainer interface {
	// Init registers list, replacing any previously registered factories.
	Init(list CAnyList)

	// Register adds list's factories to the container.
	Register(list CAnyList) error

	// Get resolves the value registered under key.
	Get(key string) any
}

// ITemplateHolder resolves a namespace's layout template - see ITemplateRenderer.
type ITemplateHolder interface {
	// TemplateRenderer returns a fresh ITemplateRenderer.
	TemplateRenderer() ITemplateRenderer

	// Layout returns the layout template's contents for nmsp (namespace).
	Layout(nmsp string) string

	// LayoutPath returns the layout template's file path for nmsp (namespace).
	LayoutPath(nmsp string) string
}

// ITemplateRenderer renders a template (optionally wrapped in a layout)
// with a set of parameters - configure it via the Set*/AddParam methods
// (each returns the renderer itself for chaining), then call Render.
type ITemplateRenderer interface {
	// SetNamespace sets the template namespace, used to resolve the layout.
	SetNamespace(nmsp string) ITemplateRenderer

	// SetTemplateName sets the template to render by name (resolved within the namespace).
	SetTemplateName(name string) ITemplateRenderer

	// SetLayout sets the layout template's raw source.
	SetLayout(code string) ITemplateRenderer

	// SetTemplate sets the template's raw source directly, bypassing name resolution.
	SetTemplate(code string) ITemplateRenderer

	// SetParams replaces all render parameters.
	SetParams(params any) ITemplateRenderer

	// AddParam sets a single render parameter.
	AddParam(name string, val any) ITemplateRenderer

	// Namespace returns the current template namespace.
	Namespace() string

	// TemplateName returns the current template name.
	TemplateName() string

	// Layout returns the current layout source.
	Layout() string

	// Template returns the current template source.
	Template() string

	// Render renders the template (wrapped in the layout, if set) with the current params.
	Render() (string, error)
}

// ILogger writes application log messages - see IApp.SetLogger.
type ILogger interface {
	// Log writes an informational message under category.
	Log(msg string, category string)

	// LogWarning writes a warning message under category.
	LogWarning(msg string, category string)

	// LogError writes an error message under category.
	LogError(msg string, category string)
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * HTTP ROUTING
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

// CHttpResource constructs an IHttpResource - see IRouter.RegisterResource.
type CHttpResource func() IHttpResource

// HttpResourcesList maps routes to their resource constructors - see IRouter.RegisterResources.
type HttpResourcesList map[string]CHttpResource

// HttpTemplatesList maps template names to their HttpTemplateConfig - see IRouter.RegisterTemplates.
type HttpTemplatesList map[string]HttpTemplateConfig

// AssetsList maps route paths to file paths - see IRouter.RegisterFileAssets.
type AssetsList map[string]string

// FMiddleware runs before a request reaches its resource - returning an
// error aborts the request. See IRouter.AddMiddleware.
type FMiddleware func(IHandleContext) error

// CSerializer constructs an ISerializer.
type CSerializer func() ISerializer

// HttpTemplateConfig registers a named template - see IRouter.RegisterTemplates.
type HttpTemplateConfig struct {
	Template string
	Params   any
}

// HttpProxyConfig configures proxying a set of routes through to another server - see IRouter.RegisterProxy.
type HttpProxyConfig struct {
	// Server is the upstream server's base URL.
	Server string
	// Routes lists the routes to proxy.
	Routes []string
	// Map optionally rewrites route paths before forwarding.
	Map map[string]string
}

// HttpResourceConfig configures an IHttpResource's request/response/fail forms - see IHttpResource.
type HttpResourceConfig struct {
	CRequestForm  CForm
	CResponseForm CForm
	CFailForm     CForm
}

// HtmlResponseConfig configures an HTML response - see IHttpResource.HtmlResponse.
type HtmlResponseConfig struct {
	Code     int
	Headers  map[string]string
	Html     string
	Params   any
	Template string
}

// JsonResponseConfig configures a JSON response - see IHttpResource.JsonResponse/FailResponse.
type JsonResponseConfig struct {
	Code    int
	Headers map[string]string
	Data    any
	Dict    Dict
	Form    IForm
}

// CForm constructs an IForm - see HttpResourceConfig.
type CForm func() IForm

// FormConfig maps field names to their FormFieldConfig - see IForm.Config.
type FormConfig map[string]FormFieldConfig

// FormFieldConfig describes one form field.
type FormFieldConfig struct {
	Description string
	Required    bool
}

// IRouter dispatches incoming requests to registered resources - see
// http.Router for the default implementation.
type IRouter interface {
	// AddMiddleware registers a middleware, run before every request.
	AddMiddleware(FMiddleware)

	// Resources returns all registered resources, keyed by HTTP method then route.
	Resources() map[string]HttpResourcesList

	// RegisterTemplates registers named templates.
	RegisterTemplates(tpls HttpTemplatesList)

	// RegisterResources registers a batch of routes.
	RegisterResources(routes HttpResourcesList)

	// RegisterResource registers a single route/method.
	RegisterResource(route string, method string, cResource CHttpResource)

	// RegisterFileAssets registers static file routes.
	RegisterFileAssets(assets map[string]string)

	// RegisterProxy registers routes proxied through to another server.
	RegisterProxy(conf HttpProxyConfig)

	// GetAssetRoute returns the registered route for a file path, if any.
	GetAssetRoute(path string) string

	// Handle runs res for the given route/writer/request and returns its response.
	Handle(res IHttpResource, route string, w http.ResponseWriter, r *http.Request) IHttpResponse

	// Start begins serving requests.
	Start()
}

// IHandleContext carries one request's state through middleware and its
// resource - see http.HandleContext for the default implementation.
type IHandleContext interface {
	// Init sets up the context for a single request.
	Init(
		app IApp,
		route string,
		method string,
		writer http.ResponseWriter,
		request *http.Request,
	)

	// App returns the owning application.
	App() IApp

	// Route returns the matched route.
	Route() string

	// Method returns the HTTP method.
	Method() string

	// ResponseWriter returns the underlying http.ResponseWriter.
	ResponseWriter() http.ResponseWriter

	// Request returns the underlying *http.Request.
	Request() *http.Request

	// Resource returns the resource handling this request.
	Resource() IHttpResource

	// Has reports whether key is set in the context.
	Has(key any) bool

	// SetParams replaces all of the context's parameters.
	SetParams(params map[string]any)

	// Params returns all of the context's parameters.
	Params() map[string]any

	// Set stores a value under key, for use across middleware/the resource.
	Set(key any, value any)

	// Get returns the value stored under key, or nil.
	Get(key any) any
}

// IHttpResource handles requests for one route - see http.Resource for the
// default implementation.
type IHttpResource interface {
	// Base returns the underlying *http.Resource - useful when IHttpResource is wrapped.
	Base() IHttpResource

	// CRequestForm returns the constructor for the request form, if any.
	CRequestForm() CForm

	// CResponseForm returns the constructor for the response form, if any.
	CResponseForm() CForm

	// CFailForm returns the constructor for the fail-response form, if any.
	CFailForm() CForm

	// Init is a no-op - override it for per-request setup.
	Init()

	// BeforeRun registers a hook to run right before Run.
	BeforeRun(func(IHttpResource))

	// BeforeRunCallbacks returns hooks to run right before Run.
	BeforeRunCallbacks() []func(res IHttpResource)

	// Run handles the request and returns the response - implement this in
	// your resource. Run executes even if the request form failed
	// validation and ProcessRequestErrors wasn't overridden (returned nil)
	// - check RequestForm().HasErrors() yourself here if Run needs to react
	// to that instead of relying on ProcessRequestErrors' short-circuit.
	Run() IHttpResponse

	// ProcessRequestErrors is called when the request form failed
	// validation - override it to return a custom error response and
	// short-circuit Run. Returning nil (the default) does NOT block Run -
	// it still executes, with a request form that may carry errors.
	ProcessRequestErrors() IHttpResponse

	// Lang returns the request's resolved language.
	Lang() string

	// SetContext binds the resource to its IHandleContext.
	SetContext(c IHandleContext)

	// Context returns the resource's IHandleContext.
	Context() IHandleContext

	// App returns the owning application.
	App() IApp

	// Route returns the matched route.
	Route() string

	// Method returns the HTTP method.
	Method() string

	// ResponseWriter returns the underlying http.ResponseWriter.
	ResponseWriter() http.ResponseWriter

	// Request returns the underlying *http.Request.
	Request() *http.Request

	// SetRequestForm sets the parsed request form.
	SetRequestForm(f IForm)

	// RequestForm returns the parsed request form.
	RequestForm() IForm

	// Log writes an informational message under category.
	Log(msg string, category string)

	// LogWarning writes a warning message under category.
	LogWarning(msg string, category string)

	// LogError writes an error message under category.
	LogError(msg string, category string)

	// HtmlResponse builds an HTML response.
	HtmlResponse(conf HtmlResponseConfig) IHttpResponse

	// JsonResponse builds a successful JSON response.
	JsonResponse(conf JsonResponseConfig) IHttpResponse

	// FailResponse builds a failed JSON response.
	FailResponse(conf JsonResponseConfig) IHttpResponse

	// ErrorResponse builds a JSON error response with the given HTTP code and message.
	ErrorResponse(code int, msg string) IHttpResponse

	// Redirect builds an HTTP redirect response.
	Redirect(URL string, code int, params map[string]any) IHttpResponse

	// PostRedirect builds a redirect response for a POST request.
	PostRedirect(url string, params map[string]any) IHttpResponse
}

// IHttpResponse is a response built by an IHttpResource, ready to be sent.
type IHttpResponse interface {
	// SetCode sets the HTTP status code.
	SetCode(code int)

	// AddHeader adds a response header.
	AddHeader(key, val string)

	// SetError sets the response to an error with the given code and message.
	SetError(code int, msg string)

	// SetHtmlData sets the response body to raw HTML.
	SetHtmlData(data string)

	// SetJsonData sets the response body by marshaling data to JSON.
	SetJsonData(data any) error

	// Code returns the HTTP status code.
	Code() int

	// Headers returns the response headers.
	Headers() map[string]string

	// Data returns the raw response body.
	Data() string

	// Send writes the response to w.
	Send(w http.ResponseWriter)
}

// IForm parses and validates request/response data - see http.Form for the
// default implementation.
type IForm interface {
	IErrorsCollector

	// Config describes the form's fields.
	Config() FormConfig

	// SetRequired overrides which fields are required.
	SetRequired(required []string)

	// Required returns the currently required fields.
	Required() []string

	// AfterFill runs once the form's fields have been populated - override
	// it for cross-field logic.
	AfterFill()

	// Validate reports whether the filled form is valid.
	Validate() bool
}

// IFormFiller is a fluent form-filling call, started by FormFiller.
type IFormFiller interface {
	// SetForm sets the form to fill.
	SetForm(f IForm) IFormFiller

	// SetContext sets the request context to fill the form from (an HTTP
	// request's GET query or JSON/urlencoded body) - mutually exclusive with SetDict.
	SetContext(ctx IHandleContext) IFormFiller

	// SetDict sets an already-parsed kernel.Dict to fill the form from -
	// mutually exclusive with SetContext.
	SetDict(d Dict) IFormFiller

	// Fill fills the form from whichever of SetContext/SetDict was called,
	// returning an error (without touching the form) if SetForm wasn't
	// called, or if neither SetContext nor SetDict was.
	Fill() error
}

// ISerializer writes an IHttpResponse's data in a specific format.
type ISerializer interface {
	// Serialize writes r's data in this serializer's format.
	Serialize(r IHttpResponse)
}

// IError is an application error carrying a numeric code alongside its message.
type IError interface {
	// Error returns the error message.
	Error() string

	// Code returns the error's numeric code.
	Code() uint
}

// IErrorsCollector accumulates IErrors, e.g. while validating an IForm.
type IErrorsCollector interface {
	// CollectError adds err to the collection.
	CollectError(IError)

	// CollectErrorf adds a formatted, uncoded error to the collection.
	CollectErrorf(string, ...any)

	// CollectCodifiedErrorf adds a formatted error with the given code to the collection.
	CollectCodifiedErrorf(uint, string, ...any)

	// HasErrors reports whether any errors were collected.
	HasErrors() bool

	// GetFirstError returns the first collected error, or nil.
	GetFirstError() IError
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * EVENTS
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

// EVENT_APP_BEFORE_HANDLE_REQUEST fires before the app handles an incoming request.
const EVENT_APP_BEFORE_HANDLE_REQUEST = "appBeforeHandleRequest"

// EVENT_APP_BEFORE_SEND_RESPONSE fires before the app sends a response.
const EVENT_APP_BEFORE_SEND_RESPONSE = "appBeforeSendResponse"

// EVENT_APP_BEFORE_SEND_ASSET fires before the app sends a static asset.
const EVENT_APP_BEFORE_SEND_ASSET = "appBeforeSendAsset"

// EVENT_APP_BEFORE_FINAL fires before the app runs its shutdown/cleanup.
const EVENT_APP_BEFORE_FINAL = "appBeforeFinal"

// EVENT_APP_BEFORE_FAIL fires before the app sends a failure response.
const EVENT_APP_BEFORE_FAIL = "appBeforeFail"

// EVENT_RENDERER_BEFORE_RENDER fires before a template is rendered.
const EVENT_RENDERER_BEFORE_RENDER = "rendererBeforeRender"

// EVENT_CONFIG_REFRESHED fires after the app's config is reloaded.
const EVENT_CONFIG_REFRESHED = "configRefreshed"

// FEventHandler handles a single IEvent - see IEventManager.Subscribe.
type FEventHandler func(e IEvent)

// IEventManager dispatches named events to subscribed handlers - see IApp.Events.
type IEventManager interface {
	// Subscribe registers a function to run when eventName fires.
	Subscribe(eventName string, handler FEventHandler)

	// Handle registers an IEventHandler to run when eventName fires.
	Handle(eventName string, handler IEventHandler)

	// Trigger fires eventName with the given payload data.
	Trigger(eventName string, d ...IDict)
}

// IEvent is a single firing of a named event, carrying an optional payload.
type IEvent interface {
	// Name returns the event's name.
	Name() string

	// App returns the application the event fired on.
	App() IApp

	// SetPayload sets the event's payload data.
	SetPayload(d IDict)

	// Payload returns the event's payload data.
	Payload() IDict
}

// IEventHandler is a reusable, app-bound handler for one or more events -
// see IEventManager.Handle.
type IEventHandler interface {
	// SetApp binds the handler to its owning app.
	SetApp(app IApp)

	// App returns the owning app.
	App() IApp

	// Run handles the event.
	Run(e IEvent)
}
