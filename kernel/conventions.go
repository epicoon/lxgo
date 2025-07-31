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

type Dict map[string]any
type Config Dict

type CAppComponentConfig func() IAppComponentConfig

type IApp interface {
	SetPort(p int)
	SetConfig(c *Config)
	SetComponent(key any, c IAppComponent)
	HasComponent(key any) bool
	Component(key any) IAppComponent
	SetConnection(c IConnection)
	SetRouter(r IRouter)
	Config() *Config
	Pathfinder() IPathfinder
	DIContainer() IDIContainer
	Connection() IConnection
	Router() IRouter
	TemplateHolder() ITemplateHolder
	TemplateRenderer() ITemplateRenderer
	Events() IEventManager
	Log(msg string, category string)
	LogWarning(msg string, category string)
	LogError(msg string, category string)
	Run()
	Final()
}

type IAppComponent interface {
	SetApp(app IApp)
	SetConfig(conf IAppComponentConfig)
	GetConfig() IAppComponentConfig
	Name() string
	App() IApp
	CConfig() CAppComponentConfig
	AfterInit()
}

type IAppComponentConfig interface {
	IsMap() bool
	Set(key string, val any)
	Has(key string) bool
	Get(key string) any
}

type IPathfinder interface {
	GetRoot() string
	GetAbsPath(path string) string
}

type IConnection interface {
	SetApp(app IApp)
	SetConfig(cfg *Config)
	DB() *sql.DB
	Connect() error
	Close() error
}

type CAnyList map[string]func(...any) any

type IDIContainer interface {
	Init(list CAnyList)
	Get(key string) any
}

type ITemplateHolder interface {
	TemplateRenderer() ITemplateRenderer
	Layout(nmsp string) string
	LayoutPath(nmsp string) string
}

type ITemplateRenderer interface {
	SetNamespace(nmsp string) ITemplateRenderer
	SetTemplateName(name string) ITemplateRenderer
	SetLayout(code string) ITemplateRenderer
	SetTemplate(code string) ITemplateRenderer
	SetParams(params any) ITemplateRenderer
	AddParam(name string, val any) ITemplateRenderer
	Namespace() string
	TemplateName() string
	Layout() string
	Template() string
	Render() (string, error)
}

type ILogger interface {
	Log(msg string, category string)
	LogWarning(msg string, category string)
	LogError(msg string, category string)
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * HTTP ROUTING
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

type CHttpResource func() IHttpResource
type HttpResourcesList map[string]CHttpResource
type HttpTemplatesList map[string]HttpTemplateOptions
type AssetsList map[string]string
type FMiddleware func(IHandleContext) error
type CSerializer func() ISerializer

type HttpTemplateOptions struct {
	Template string
	Params   any
}

type HttpResourceConfig struct {
	CRequestForm  CForm
	CResponseForm CForm
	CFailForm     CForm
}

type HtmlResponseConfig struct {
	Code     int
	Headers  map[string]string
	Html     string
	Params   any
	Template string
}

type JsonResponseConfig struct {
	Code    int
	Headers map[string]string
	Data    any
	Dict    Dict
	Form    IForm
}

type CForm func() IForm
type FormConfig map[string]FormFieldConfig
type FormFieldConfig struct {
	Description string
	Required    bool
}

type IRouter interface {
	AddMiddleware(FMiddleware)
	Resources() map[string]HttpResourcesList
	RegisterTemplates(tpls HttpTemplatesList)
	RegisterResources(routes HttpResourcesList)
	RegisterResource(route string, method string, cResource CHttpResource)
	RegisterFileAssets(assets map[string]string)
	GetAssetRoute(path string) string
	Handle(res IHttpResource, route string, w http.ResponseWriter, r *http.Request)
	Start()
}

type IHandleContext interface {
	App() IApp
	Route() string
	Method() string
	ResponseWriter() http.ResponseWriter
	Request() *http.Request
	Resource() IHttpResource
	Set(key any, value any)
	Get(key any) any
}

type IHttpResource interface {
	CRequestForm() CForm
	CResponseForm() CForm
	CFailForm() CForm
	Init()
	BeforeRun(func(IHttpResource))
	PreRun()
	Run() IHttpResponse
	ProcessRequestErrors() IHttpResponse
	Lang() string
	SetContext(c IHandleContext)
	Context() IHandleContext
	App() IApp
	Route() string
	Method() string
	ResponseWriter() http.ResponseWriter
	Request() *http.Request
	SetRequestForm(f IForm)
	RequestForm() IForm
	Log(msg string, category string)
	LogWarning(msg string, category string)
	LogError(msg string, category string)
	HtmlResponse(conf HtmlResponseConfig) IHttpResponse
	JsonResponse(conf JsonResponseConfig) IHttpResponse
	FailResponse(conf JsonResponseConfig) IHttpResponse
	ErrorResponse(code int, msg string) IHttpResponse
	PostRedirect(url string, params map[string]any) IHttpResponse
}

type IHttpResponse interface {
	SetCode(code int)
	Code() int
	AddHeader(key, val string)
	SetError(code int, msg string)
	SetHtmlData(data string)
	SetJsonData(data any) error
	Send(w http.ResponseWriter)
}

type IForm interface {
	IErrorsCollector
	Config() FormConfig
	SetRequired(required []string)
	Required() []string
	Fill(d *Dict) error
	AfterFill()
	Validate() bool
}

type ISerializer interface {
	Serialize(IHttpResponse)
}

type IError interface {
	Error() string
	Code() uint
}

type IErrorsCollector interface {
	CollectError(IError)
	CollectErrorf(string, ...any)
	CollectCodifiedErrorf(uint, string, ...any)
	HasErrors() bool
	GetFirstError() IError
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * EVENTS
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

const EVENT_APP_BEFORE_HANDLE_REQUEST = "appBeforeHandleRequest"
const EVENT_APP_BEFORE_SEND_RESPONSE = "appBeforeSendResponse"
const EVENT_APP_BEFORE_SEND_ASSET = "appBeforeSendAsset"
const EVENT_RENDERER_BEFORE_RENDER = "rendererBeforeRender"

type FEventHandler func(e IEvent)

type IEventManager interface {
	Subscribe(eventName string, handler FEventHandler)
	Handle(eventName string, handler IEventHandler)
	Trigger(eventName string, d ...IData)
}

type IEvent interface {
	Name() string
	App() IApp
	SetPayload(d IData)
	Payload() IData
}

type IEventHandler interface {
	SetApp(app IApp)
	App() IApp
	Run(e IEvent)
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * COMMON
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

type IData interface {
	Set(key string, val any)
	Get(key string) any
	Has(key string) bool
}
