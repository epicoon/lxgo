# The package will help you create web-server

You can create your own web-server - an application with components, routing and requests handling.


## There are steps from this tutorial to make your app:
1. [Create an app directory](#link1)
2. [Write main code](#link2)
3. [Create the app configuration](#link3)
4. [Create the app instance](#link4)
5. [Configure the first routing](#link5)
6. [Use templates with layout](#link6)
7. [Make a request handler](#link7)
8. [Use forms](#link8)
9. [Generate API documentation](#link9)


## Useful features:

* [Templates](#tpl)
* [Components](#components)
* [Events](#events)


## See also:
* [Package for console commands creating](https://github.com/epicoon/lxgo/tree/master/cmd)
* [Package for manage migrations](https://github.com/epicoon/lxgo/tree/master/migrator)
* [Package for HTTP sessions maintenance](https://github.com/epicoon/lxgo/tree/master/session)
* [Package JS preprocessor](https://github.com/epicoon/lxgo/tree/master/jspp)


## Create a simple app step by step

### <a name="link1">1. Create directory and file for your app:</a>
- `mkdir my_app && cd my_app`
- `touch main.go`


### <a name="link2">2. Write main code:</a>
- main.go
    ```go
    package main

    func main() {

    }
    ```
- run `go mod init {{your module name}}` to create `go.mod` file


### <a name="link3">3. Create app configuration file `config.yaml` in the root directory of the app:</a>
```yaml
# Port your app will use
Port: 8081
```


### <a name="link4">4. Lets create the app instance and run it. Change your `main.go` file:</a>
- use this code for yout `main.go` file:
    ```go
    package main

    import (
        "fmt"

        "github.com/epicoon/lxgo/kernel"
        "github.com/epicoon/lxgo/kernel/config"
        lxApp "github.com/epicoon/lxgo/kernel/app"
    )

    type App struct {
        *lxApp.App
    }

    var _ (kernel.IApp) = (*App)(nil)

    func NewApp() kernel.IApp {
        return &App{App: lxApp.NewApp()}
    }

    func main() {
        // Create app instance
        app := NewApp()

        // Apply app config
        conf, err := config.Load(app.Pathfinder().GetAbsPath("config.yaml"))
        if err != nil {
            fmt.Printf("can not read application config. Cause: %v\n", err)
            return
        }
        if err := lxApp.InitApp(app, conf); err != nil {
            fmt.Printf("can not init application: %sv\n", err)
            return
        }

        // Run app
        app.Run()
        app.Final()
    }
    ```
- run `go mod tidy` command to get the `lxgo/kernel` package
- you can check your `go.mod` file:
    ```
    module {{ your module name }}

    go {{ go version }}

    require github.com/epicoon/lxgo/kernel v{{ version }}
    ```
    > You can check actual `lxgo/kernel` version [here](https://github.com/epicoon/lxgo/tree/master/kernel/CHANGE_LOG.md)
- Now you can run your app with command `go run .`. But now the execution stops because the request routes for handling are not configured.


### <a name="link5">5. Lets configure request route and create the first template</a>
- Create template:
    * make directory with command `mkdir templates && cd templates`
    * make template file `touch home.html`
    ```html
    {{ define "home" }}
    <!DOCTYPE html>
    <html lang="en">
    <head>
        <meta charset="UTF-8">
        <title>{{ .Title }}</title>
    </head>
    <body>
        <p>Hello, world!</p>
    </body>
    </html>
    {{ end }}
    ```
- Change `config.yaml`:
```yaml
# Templates map
Templates:
  # Templates directory
  - Dir: templates
```
- Change `main.go` file:
```go
	// ...

	InitRoutes(app.Router())

	app.Run()
	app.Final()
}

func InitRoutes(router kernel.IRouter) {
	router.RegisterTemplates(kernel.HttpTemplatesList{
		"/": kernel.HttpTemplateOptions{
			Template: "home",
			Params:   struct{ Title string }{Title: "Home page"},
		},
	})
}
```
- restart your application. Now you can open your page in browser using URL `http://localhost:8081`


### <a name="link6">6. Of course layout is the useful approach so lets reorganize our templates:</a>
- change `home.html` file:
```html
{{ define "title" }}
    {{ .Title }}
{{ end }}

{{ define "content" }}
    <p>Hello, world!</p>
{{ end }}
```
- add `layout.html` file next to `home.html`:
```html
{{ define "layout" }}
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>{{ block "title" . }}{{ end }}</title>
</head>
<body>
    {{ block "content" . }}{{ end }}
</body>
</html>
{{ end }}
```
- change `config.yaml` file:
```yaml
Templates:
  - Dir: templates
    Layout: layout
```
- restart your application. Nothing has changed but now you can make a lot of templates with the same layout!


### <a name="link7">7. Now we can make a handler to process request with code</a>
- create handlers file `tpl_handler.go`:
```go
package main

import (
	"github.com/epicoon/lxgo/kernel"
	lxHttp "github.com/epicoon/lxgo/kernel/http"
)

type TplHandler struct {
	*lxHttp.Resource
}

var _ kernel.IHttpResource = (*TplHandler)(nil)

func NewTplHandler() kernel.IHttpResource {
	return &TplHandler{Resource: lxHttp.NewResource()}
}

func (handler *TplHandler) Run() kernel.IHttpResponse {
	queryParams := handler.Context().Request().URL.Query()
	val, exists := queryParams["name"]
	var tpl string
	if exists {
		tpl = val[0]
	} else {
		tpl = "home"
	}

	return handler.HtmlResponse(kernel.HtmlResponseConfig{
		Template: tpl,
	})
}
```
- change `main.go` file:
```go
func InitRoutes(router kernel.IRouter) {
    router.RegisterResources(kernel.HttpResourcesList{
        "tpl": NewTplHandler,
    })

    // ...
```
- create pare of templates:
    * `tpl1.html`
    ```htlm
    {{ define "title" }}
        Template 1
    {{ end }}

    {{ define "content" }}
        <p>Hello, from Template 1!</p>
    {{ end }}
    ```
    * `tpl2.html`
    ```htlm
    {{ define "title" }}
        Template 2
    {{ end }}

    {{ define "content" }}
        <p>Hello, from Template 2!</p>
    {{ end }}
    ```
- restart your application. Now you can get the `home` page by `http://localhost:8081/tpl` URL.  
    And `tpl1.html` and `tpl2.html` by `http://localhost:8081/tpl?name=tpl1` and `http://localhost:8081/tpl?name=tpl2` respectively.


### <a name="link8">8. What if we want to make parameter `name` obligate? We can use forms:</a>
- change `tpl_handler.go` file:
```go
package main

import (
	"fmt"

	"github.com/epicoon/lxgo/kernel"
	lxHttp "github.com/epicoon/lxgo/kernel/http"
)

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * TplRequest
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */
type TplRequest struct {
	*lxHttp.Form
	Name string `dict:"name"`
}

var _ kernel.IForm = (*TplRequest)(nil)

func (f *TplRequest) Config() kernel.FormConfig {
	return kernel.FormConfig{
		"name": kernel.FormFieldConfig{
			Description: "requested template name",
			Required:    true,
		},
	}
}

func NewTplRequest() kernel.IForm {
	return lxHttp.PrepareForm(&TplRequest{Form: lxHttp.NewForm()})
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * TplHandler
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

type TplHandler struct {
	*lxHttp.Resource
}

var _ kernel.IHttpResource = (*TplHandler)(nil)

func NewTplHandler() kernel.IHttpResource {
	return &TplHandler{Resource: lxHttp.NewResource(kernel.HttpResourceConfig{
		CRequestForm:  NewTplRequest,
	})}
}

func (handler *TplHandler) Run() kernel.IHttpResponse {
	req := handler.RequestForm().(*TplRequest)
	if req.HasErrors() {
		return handler.FailResponse(kernel.JsonResponseConfig{
			Code: http.StatusBadRequest,
			Dict: kernel.Dict{
				"error": fmt.Sprintf("Invalid request: %v", req.GetFirstError()),
			},
		})
	}

	return handler.HtmlResponse(kernel.HtmlResponseConfig{
		Template: req.Name,
	})
}
```
- restart your application. Now you get an error if try to get `http://localhost:8081/tpl` without `name` parameter. Request params processing is cleaner and more obvious! Moreover using forms you can generate API documentation, see below for details.


## <a name="link9">9. Generate API documentation</a>
- plug in console command, create command wrapper in file `apidoc.go`:
    > How to set up console commands in your application you can find [here](https://github.com/epicoon/lxgo/tree/master/cmd)
```go
package main

import (
	"github.com/epicoon/lxgo/cmd"
	apidoc "github.com/epicoon/lxgo/kernel/cmd"
	lxHttp "github.com/epicoon/lxgo/kernel/http"
)

func NewApiDocCommand(_ ...cmd.ICommandOptions) cmd.ICommand {
	r := lxHttp.NewRouter(nil)
	InitRoutes(r)
	return apidoc.NewApiDocCommand(apidoc.ApiDocCommandOptions{
		Router: r,
		Output: "ApiDoc.md",
	})
}
```
- set command call in your `main.go` file:
```go
package main

import (
	"github.com/epicoon/lxgo/cmd"
)

func main() {
	cmd.Init(cmd.CommandsList{
		"":       NewMainCommand,
		"apidoc": NewApiDocCommand,
	})
	cmd.Run()
}
```
- call console command `go run . apidoc` and you'll find a new file `ApiDoc.md`!


## Features

### <a name="tpl">Templates</a>
You can organize templates in different directories using `namespace`. An example of the application configuration:
```yaml
Templates:
  # The first templates directory without namespace
  - Dir: templates/common
    Layout: layout
  # The second templates directory with namespace
  - Dir: templates/alt
    Namespace: "alt"
    Layout: layout
```
Every directory has to have `layout.html` - layout file. For example:
```html
{{ define "layout" }}
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>{{ block "title" . }}{{ end }}</title>
</head>
<body>
    {{ block "content" . }}{{ end }}
</body>
</html>
{{ end }}
```
So your directory tree can be like this:
```
/templates
  ├─ /common
  │    ├─ layout.html
  │    └─ home.html
  └─ /alt
       ├─ layout.html
       └─ home.html
```
An example of `home.html` file:
```html
{{ define "title" }}
    {{ .Title }}
{{ end }}
{{ define "content" }}
    <p>Home page</p>
{{ end }}
```
And you can use namespaced template:
```go
html, _ := handler.App().TemplateRenderer().
	SetTemplateName("alt:home").
	SetParams(struct {
		Title string
	}{
		Title: "Home page",
	}).Render()
```


### <a name="components">Components</a>

You can make application components. These are units of functionality registered in the application core. They represent services used by different parts of the code: for example a database component, logging or authorization. Components are initialized once and are available globally which simplifies the architecture and code reuse.

1. Make new package directory for the component: `mkdir subpkg && cd subpkg`

2. Create new component in file `my_component.go`:
```go
package subpkg

import (
	"fmt"

	"github.com/epicoon/lxgo/kernel"
	lxApp "github.com/epicoon/lxgo/kernel/app"
)

const MY_COMPONENT_KEY = "my_component"

// Component config
type MyComponentConfig struct {
	*lxApp.ComponentConfig
	Name string
}

var _ kernel.IAppComponentConfig = (*MyComponentConfig)(nil)

// Config constructor
func NewMyComponentConfig() kernel.IAppComponentConfig {
	return &MyComponentConfig{ComponentConfig: lxApp.NewComponentConfigStruct()}
}

// Component
type MyComponent struct {
	*lxApp.AppComponent
}

var _ kernel.IAppComponent = (*MyComponent)(nil)

// Component initializator
func SetAppComponent(app kernel.IApp, configKey string) error {
	comp := &MyComponent{AppComponent: lxApp.NewAppComponent()}
	return lxApp.RegisterComponent(app, comp, MY_COMPONENT_KEY, configKey)
}

// Method to get the component
func AppComponent(app kernel.IApp) (*MyComponent, error) {
	c := app.Component(MY_COMPONENT_KEY)
	if c == nil {
		return nil, fmt.Errorf("application component '%s' does not exist", MY_COMPONENT_KEY)
	}

	comp, ok := c.(*MyComponent)
	if !ok {
		return nil, fmt.Errorf("application component '%s' is not '*MyComponent'", MY_COMPONENT_KEY)
	}

	return comp, nil
}

// According to interface kernel.IAppComponent
func (comp *MyComponent) Name() string {
	return "MyComponent"
}

// According to interface kernel.IAppComponent
// Config constructor
func (comp *MyComponent) CConfig() kernel.CAppComponentConfig {
	return NewMyComponentConfig
}

// Useful method - config getter
func (comp *MyComponent) Config() *MyComponentConfig {
	return (comp.GetConfig()).(*MyComponentConfig)
}

// Custom component method
func (comp *MyComponent) DoSomething() {
	fmt.Println("Hi "+comp.Config().Name+"!")
}
```

3. Prepare component configuration in file `config.yaml`:
```yaml
Components:
  MyComponent:
    Name: Al
```

4. Plug in component before `app.Run()`:
```go
// ...

// Use "Components.MyComponent" as key according to structure in `config.yaml`
if err := subpkg.SetAppComponent(app, "Components.MyComponent"); err != nil {
	fmt.Printf("can not init component 'MyComponent': %v\n", err)
	return
}

app.Run()
```

5. Use component:
```go
// ...

comp, err := subpkg.AppComponent(app)
if err != nil {
	fmt.Printf("can not get 'MyComponent': %v\n", err)
	return
}

comp.DoSomething()
```


### <a name="events">Events</a>

There are several events of application lifecycle:

* `kernel.EVENT_APP_BEFORE_SEND_ASSET`
    - **trigger**: before send asset like css, js files etc.
    - **payload**:
        | key     | type          |
        | ------- | ------------- |
        | request | *http.Request |
        | file    | string        |

* `kernel.EVENT_APP_BEFORE_HANDLE_REQUEST`
    - **trigger**: before call method `kernel.IHttpResource.Run()`
    - **payload**:
        | key     | type                  |
        | ------- | --------------------- |
        | context | kernel.IHandleContext |

* `kernel.EVENT_APP_BEFORE_SEND_RESPONSE`
    - **trigger**: before sending responce after the method `kernel.IHttpResource.Run()`
    - **payload**:
        | key      | type                  |
        | -------- | --------------------- |
        | context  | kernel.IHandleContext |
        | response | kernel.IHttpResponse  |
* `kernel.EVENT_RENDERER_BEFORE_RENDER`
    - **trigger**: before `app.TemplateRenderer()` render a template
    - **payload**:
        | key       | type                     |
        | --------- | ------------------------ |
        | renderer  | kernel.ITemplateRenderer |

Example of events using:
```go
app.Events().Subscribe(kernel.EVENT_APP_BEFORE_SEND_ASSET, func(e kernel.IEvent) {
	filePath := e.Payload().Get("file").(string)
	request := e.Payload().Get("request").(*http.Request)

	// do something ...
})
```
