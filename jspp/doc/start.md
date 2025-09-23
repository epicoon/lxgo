## Steps:
1. [Start use `JSPreprocessor`](#link1)
2. [Set up access to `JSPreprocessor`'s console commands](#link2)
3. [Build `core.js`](#link3)
4. [Create JS application](#link4)


### <a name="link1">1. Start use `JSPreprocessor`</a>

- Get the package with commang `go get https://github.com/epicoon/lxgo/jspp`

- Configurate `JSPreprocessor` component in application configuration file `config.yaml`:
  ```yaml
  Components:
    # ...
    JSPreprocessor:
      # Provides you to get more information in built JS-code
      Mode: DEV
      # Path to directory where will be the core of JS-code
      CorePath: frontend/build/core.js
      # Path to directory where the Preprocessor keeps its maps files
      MapsPath: .sys/cmp/jspp
      # Path to directory where the Preprocessor copies JS-modules files
      ModsPath: .sys/cmp/jspp/modules
      # Preprocessor automatically creates links to asset files
      #  to hide the real location on the server
      AssetLinksPath:
        # Path to directory on the server
        Inner: frontend/web/assets
        # URL path to request asset by client
        Outer: /web
      # Set of automatically rebuilt JS-bundles
      Targets:
          # Path to main JS-file which is has to be built
        - EntryPoint: frontend/src/App.js
          # Path to the built JS-bundle
          Output: frontend/build/app.js
          # Optional: available value - app. Preprocessor build and init an JS-application instance and include it to the bundle
          Type: app
      # Optional: path to JS-application configuration file
      AppConfig: frontend/lx-app.yaml
      # Optional: the side to build css from JS-preprocessor. Available values: client (by default), server
      CssScopeRenderSide: server
      # Optional: directories with JS-modules which are not in the Go modules
      Modules:
        - /path/to/modules
      # Optional: directories with plugins which are not in the Go modules
      Plugins:
        - /path/to/plugins
  ```

- Plug in `JSPreprocessor` component in `go` code:
  ```go
  import (
  	"fmt"

  	jsppComp "github.com/epicoon/lxgo/jspp/component"
  )

  // Set JS-preprocessor component
  if err := jsppComp.SetAppComponent(app, "Components.JSPreprocessor"); err != nil {
  	fmt.Printf("can not init component JSPreprocessor:%v\n", err)
  }
  ```

### <a name="link2">2. Set up access to `JSPreprocessor`'s console commands</a>
You'll need it to build `code.js` and modules and plugins maps

- Make wrapper for `JSPreprocessor`'s console commands:
  ```go
  import (
  	"github.com/epicoon/lxgo/cmd"
  	jsppCmd "github.com/epicoon/lxgo/jspp/cmd"
  )

  func NewJSPPCommand(_ ...cmd.ICommandOptions) cmd.ICommand {
  	// Create your app ...

  	// Pass the application to the jspp command constructor
  	return jsppCmd.NewCompileCommand(jsppCmd.CompileCommandOptions{
  		App: app,
  	})
  }
  ```

- Plug in command constructor:
  ```go
  import (
  	"github.com/epicoon/lxgo/cmd"
  )

  func main() {
  	cmd.Init(cmd.CommandsList{
  		// ...
  		"jspp": NewJSPPCommand,
  	})
  	cmd.Run()
  }
  ```

- Now there commands are available:
  * command to build `core.js`
  `go run . jspp:build-core`
  * command to build modules and plugins maps
  `go run . jspp:build-maps`
  * command to build modules map only
  `go run . jspp:build-modules-map`
  * command to build plugins map only
  `go run . jspp:build-plugins-map`
  * command to build `core.js` and modules and plugins maps
  `go run . jspp:build`


### <a name="link3">3. Build `core.js` and use it on the browser side</a>

- Build `core.js` with command `go run . jspp:build-core`

- Build modules map with command `go run . jspp:build-modules-map`

- Set up `code.js` asset:
  ```go
  import (
  	"github.com/epicoon/lxgo/kernel"
  )

  // "/js/" - URL path to get asset from client side
  // "frontend/build" - path to directory contains [[core.js]] according to [[Components.JSPreprocessor.CorePath]] application configuration parameter
  app.Router().RegisterFileAssets(kernel.AssetsList{
  	"/js/": "frontend/build",
  })
  ```

- Use `code.js` asset on your page:
  ```html
    <script defer src="/js/core.js"></script>
  ```

- Check in browser. Open developer panel, in console write `lx` and press `enter`. You'll see the core lx object.


### <a name="link4">4. Create JS application</a>

- Direct way to create an application is to make file `frontend/src/App.js` according to the `Components.JSPreprocessor.Targets` application configuration parameter:
  ```js
  // Include JS-module
  // @lx:use lx.CssPresetDark;

  // Init and start the application
  lx.app.start({
      root: {attrs: {lxid: "lx-root"}},
      components: {
          lang: {
              'en-EN': 'English',
              // ...
          },
          imageManager: '/img',
          cssManager: lx.CssPresetDark
      },
  });

  // Lets see what the application is
  console.log(lx.app);
  ```
  And use the asset in you template:
  ```html
    <script defer src="/js/app.js"></script>
  ```

- Now you can use widgets. Add this code to your `App.js` file:
  ```js
  // Create css-code on client side using JS!
  const css = new lx.CssContext();
  css.addClass('css-green', {
      cursor: 'pointer',
      backgroundColor: 'lightgreen'
  });
  css.addClass('css-pink', {
      cursor: 'pointer',
      backgroundColor: 'pink'
  });
  const cssTag = new lx.CssTag({id: 'custom-css'});
  cssTag.addCss(css);
  cssTag.commit();

  // Create simple widget
  let box = new lx.Box({
      geom: [30, 30, 40, 40],
      text: 'Hello world!',
      css: 'css-green'
  });
  box.align(lx.CENTER, lx.MIDDLE);
  box.border();
  box.roundCorners('50%');
  box.click(() => {
      // Custom flag for the css class switching
      box._state = !box._state;
      box.toggleClassOnCondition(box._state, 'css-pink', 'css-green');
  });
  ```

- The better way to configurate JS-application is to make configuration file according to `Components.JSPreprocessor.AppConfig` application configuration parameter. It is necessery to move configuration in one place for server rendering. Lets make this file `frontend/lx-app.yaml`:
  ```yaml
  use:  # modules to use with application
    - lx.CssPresetDark
    - lx.CssPresetWhite

  root:  # attributes of app's root html-tag
    attrs: {lxid: lx-root}

  components:  # JS app components setting
    lang:
      en-EN: English
    imageManager:
      default: /img
    cssManager:
      default: lx.CssPresetDark
      white: lx.CssPresetWhite
  ```
  And now you can remove `lx.app.start(...` code from your `App.js` file because this target is marked by `Type: app`

> For more information about application component [see](https://github.com/epicoon/lxgo/tree/master/jspp/doc/components.md)

> For more information about widgets [see](https://github.com/epicoon/lxgo/tree/master/jspp/doc/widgets.md)
