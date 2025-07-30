# The package helps to work with JS

> You can use it if your application is based on [lxgo/kernel](https://github.com/epicoon/lxgo/tree/master/kernel)

## Provides:
* Creating frontend application
* Set of widgets
* CSS organization:
  - Simple CSS-preprocessor on JS
  - Presets
  - Widgets customization
* Models and binding mechanisms
* Internationalization
* Simple AJAX handling
* Way to organize code with plugins
* Markup language `lxml`
* Set of auxiliary tools

TODO design the doc
TODO add full info
  - app configuration
  - app components
  - widgets
  - depthCluster
  - models
  - all about css (context, presets, scopes)
  - i18n
  - plugins
  - ajax
  - tools (lx.Timer, lx.TableManager ect.)
  - lxml

//======================================================================================================================
// JSPreprocessor component configuration
- file `/config.yaml`
```yaml
Components:
  # ...
  JSPreprocessor:
    Mode: DEV
    CorePath: frontend/build/core.js
    CachePath: .sys/cmp/jspp
    AssetLinksPath:
      Inner: frontend/web/assets
      Outer: /web
    AppConfig: frontend/lx-app.yaml
    CssScopeRenderSide: server  # or plugin
    Modules: # optional
      - /path/to/modules
  	Plugins: # optional
	    - /path/to/plugins
    Targets:
      - EntryPoint: frontend/src/App.js
        Output: frontend/build/app.js
	    	Type: app  # optional
```


//======================================================================================================================
// Example of application configuration
  >> You can configure it in file defined by `AppConfig` option
  >> Will be applied automatically for targets with `Type: app` option

* In config file:
  ```yaml
  use:  # modules to use with application
    - lx.CssPresetDark
    - lx.CssPresetWhite

  root:  # attributes of app's root html-tag
    attrs: {lxid: lx-root}

  components:  # JS app components setting
    lang:
      en-EN: English
      ru-RU: Russian
    imageManager:
      default: /img
    cssManager:
      default: lx.CssPresetDark
      white: lx.CssPresetWhite
  ```

* In JS-code:
  ```js
  lx.app.start({
      root: {attrs: {lxid: "lx-root"}},
      components: {
          lang: {
              'en-EN': 'English',
              'ru-RU': 'Russian'
          },
          imageManager: {
              'default': '/img'
          },
          cssManager: {
              'default': lx.CssPresetDark,
              'white': lx.CssPresetWhite
          },
      },
  });
  ```


//======================================================================================================================
// Plug-in component

```go
import (
  jsppComp "github.com/epicoon/lxgo/jspp/component"
  "github.com/epicoon/lxgo/jspp"
)

// Set JS-preprocessor component
err := jsppComp.SetAppComponent(app, "Components.JSPreprocessor");
// ... process err

// Optional: set routes to plugins as pages
pp, _ := jsppComp.AppComponent(app)
pp.PluginManager().SetRoutes(jspp.PluginRoutesList{
  "/pl1": "FirstPlugin",
  "/pl2": "SecondPlugin",
})
```


//======================================================================================================================
// Use console commands
1. Wrapp cmd with pathing app
```go
package cmd

import (
	"fmt"

	"github.com/epicoon/lxgo/cmd"
	jsppCmd "github.com/epicoon/lxgo/jspp/cmd"
)

/** @type cmd.FConstructor */
func NewJSPPCommand(opt ...cmd.TCommandOptions) cmd.ICommand {
  // ... init app

  return jsppCmd.NewCompileCommand(cmd.TCommandOptions{
		"app": app,
	})
}
```

2. Set up console cmd
```go
	cmd.Init(cmd.CommandsList{
    // ...
		"jspp": chatcmd.NewJSPPCommand,
	})
```

3. Using
  * run command to build `core.js`
  `go run . jspp:build-core`
  * run command to build modules and plugins maps
  `go run . jspp:build-maps`
  * run command to build modules map only
  `go run . jspp:build-modules-map`
  * run command to build plugins map only
  `go run . jspp:build-plugins-map`


//======================================================================================================================
// Using `lx` on client side
1. check that assets are available
  ```go
    app.Router().RegisterFileAssets(kernel.AssetsList{
      "/js/":  "frontend/build",
      "/img/": "frontend/img",
    })
  ```

2. plug in `core.js` on pages
  ```html
      <script defer src="/js/core.js"></script>
  ```


//======================================================================================================================
// Make plugin
1. Make plugin directory. For exaple `path/to/project/frontend/plugins/my-plugin`
  >> If you use directory outside of your project specify this path in `JSPreprocessor.Plugins` option

2. Create plugin content
  * Minimal requires for plugin directory content:
    /my-plugin
      ├─ lx-plugin.yaml 
      └─ snippet.js
    >> The file `lx-plugin.yaml` is plugin's configuration and the name can not be changed
    >> THe file `snippet.js` is plugin's view. The name can be configured

    * Minimal plugin config:
      ```yaml
      name: MyPlugin
      server:
        rootSnippet: snippet.js
      ```

  * Example of plugin:
    /example-plugin
      │ ├─ /assets
      │ │    ├─ css
      │ │    │   └─ MainCss.js
      │ │    ├─ i18n
      │ │    │   └─ tr.yaml
      │ │    └─ img
      │ │        └─ image.png
      │ ├─ /client
      │ │    ├─ guiNodes
      │ │    │   ├─ MainBox.js
      │ │    │   └─ Popup.js
      │ │    └─ src
      │ │        └─ Logic.js
      │ └─ /snippets
      │      ├─ _root.js
      │      └─ popup.js
      ├─ lx-plugin.yaml
      ├─ Core.js
      ├─ Plugin.js
      └─ plugin.go

    * Example of full plugin config:
      ```yaml
      name: ExamplePlugin
      images:
        default: assets/img
        specialKey: path/to/img
      i18n: assets/i18n/tr.yaml
      require:
        - assets/css/
      cssAssets:
        - MainCss

      cacheType: 'on'

      server:
        key: namespace.ExamplePlugin
        rootSnippet: snippets/_root.js
        snippets:
          - snippets
        snippetsMap:
          ext1: "@alias/externalSnippet.js"
          ext2:
            path: "@alias/externalSnippet.js"
          ext3:
            plugin: AnotherPlugin
            path: snippets/someSnippet.js
          ext4:
            plugin: AnotherPlugin
            snippet: someSnippetKey

      client:
        file: Plugin.js
        require:
          - Core.js
          - client/guiNodes/
          - -R client/src/
        core: Core
        guiNodes:
          mainBox: MainBox
          somePopup: Popup
      ```

3. You can make `Plugin.js` file
  ```js
  // @lx:namespace myNmsp;
  class Plugin extends lx.Plugin {
      run() {
        // ... code
      }
  }
  ```


//======================================================================================================================
// Plugin as page
```go
pp.PluginManager().SetRoutes(jspp.PluginRoutesList{
  "/pl1": "FirstPlugin",
  "/pl2": "SecondPlugin",
})
```


//======================================================================================================================
// Ajax-loaded Plugin

1. You need ajax handler:
  ```go
  package handlers

  import (
    jsppComp "github.com/epicoon/lxgo/jspp/component"
    "github.com/epicoon/lxgo/kernel"
    "github.com/epicoon/lxgo/kernel/http"
  )

  /* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
  * PluginRequest
  * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */
  /** @interface kernel.IForm */
  type PluginRequest struct {
    *http.Form
    PluginName string `json:"pluginName"`
  }

  var _ kernel.IForm = (*PluginRequest)(nil)

  func (f *PluginRequest) Config() kernel.FormConfig {
    return kernel.FormConfig{
      "pluginName": kernel.FormFieldConfig{
        Description: "requested plugin name",
        Required:    true,
      },
    }
  }

  /** @constructor kernel.CForm */
  func NewPluginRequest() kernel.IForm {
    return http.PrepareForm(&PluginRequest{Form: http.NewForm()})
  }

  /* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
  * PluginHandler
  * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

  /** @interface kernel.IHttpResource */
  type PluginHandler struct {
    *http.Resource
  }

  var _ kernel.IHttpResource = (*PluginHandler)(nil)

  /** @constructor kernel.CHttpResource */
  func NewPluginHandler() kernel.IHttpResource {
    return &PluginHandler{Resource: http.NewResource(kernel.HttpResourceConfig{
      CRequestForm: NewPluginRequest,
    })}
  }

  func (h *PluginHandler) Run() kernel.IHttpResponse {
    request := h.RequestForm().(*PluginRequest)

    pp, err := jsppComp.AppComponent(h.App())
    // ... process JSPreprocess component does not plugged
    plugin := pp.PluginManager().Get(request.Plugin)
    // ... process plugin was not found
    result, err := pp.PluginManager().Render(plugin, h.Lang())
    // ... precess rendering errors

    return h.JsonResponse(kernel.JsonResponseConfig{
      Dict: kernel.Dict{
        "plugin": result,
      },
    })
  }
  ```

2. Register handler:
  ```go
  func initRoutes(app cvn.IApp) {
		"/get-plugin": handlers.NewPluginHandler,
  }
  ```

3. Now you can get a plugin:
  ```js
  (new lx.HttpRequest('/get-plugin')).
    setParams({plugin: 'MyPlugin'}).
    send().then(result => {
        let info = result.plugin;
        // Unpack plugin to some empty widget
        widget.setPlugin({info});
    });
  ```


//======================================================================================================================
// Backend plugin code
1. Make GO plugin code:
  ```go
  package my_plugin

  import (
    "github.com/epicoon/lxgo/jspp"
    "github.com/epicoon/lxgo/jspp/plugins"
  )

  /** @interface jspp.IPlugin */
  type MyPlugin struct {
    *plugins.Plugin
  }

  var _ jspp.IPlugin = (*MyPlugin)(nil)

  /** @constructor jspp.CPlugin */
  func NewMyPlugin() jspp.IPlugin {
    return &MyPlugin{Plugin: plugins.NewPlugin()}
  }

  func (p *MyPlugin) BeforeRender() {
    // ... code
  }

  func (p *MyPlugin) AfterRender() {
    // ... code
  }
  ```

2. Set DI access to plugin:
  ```go
  app.DIContainer().Init(kernel.CAnyList{
		"namespace.MyPlugin": func(args ...any) any {
			return my_plugin.NewMyPlugin()
		},
	})
  ```

3. Set plugin config:
  ```yaml
  server:
    key: namespace.MyPlugin
  ```
