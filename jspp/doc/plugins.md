# Plugins

A **plugin** is a logically self-contained mini-application: its own assets,
its own (optional) Go-side counterpart, and its own client-side entry point,
plugged into the host app as one isolated piece rather than composed from
individual widgets by hand. Like a [widget](https://github.com/epicoon/lxgo/tree/master/jspp/doc/widgets.md),
a plugin is built on [`lx.Element`](https://github.com/epicoon/lxgo/tree/master/jspp/doc/elements.md)
(so it can have its own CSS and its own ajax channel), and it's shipped as a
[module](https://github.com/epicoon/lxgo/tree/master/jspp/doc/modules.md) —
but unlike a widget, a plugin isn't instantiated with `new`; it's rendered
server-side into a chunk of markup (a set of **snippets**) plus metadata,
which the client then turns back into a live widget tree.

A plugin can be shown two ways: as a whole page at its own URL, or embedded
into any existing widget on demand (loaded via ajax and unpacked with
`widget.setPlugin(...)`).

## Contents
* [Anatomy of a plugin](#anatomy)
* [The plugin config (`lx-plugin.yaml`)](#config)
* [Backend implementation](#backend)
* [Frontend implementation](#frontend)
* [Rendering a plugin as a page](#page)
* [Loading a plugin on demand](#ondemand)
* [A plugin's own ajax endpoints](#ajax)


## <a name="anatomy">Anatomy of a plugin</a>

A plugin is a directory. Plugins that belong to the current Go module are
found automatically; for plugins that live elsewhere, list their parent
directories in `Components.JSPreprocessor.Plugins` (see [Start
using](https://github.com/epicoon/lxgo/tree/master/jspp/doc/start.md)).

The only file every plugin needs is `lx-plugin.yaml` — its name is fixed and
can't be changed. A genuinely minimal plugin is just that file plus a root
snippet at the default location:
```
/my-plugin
  ├─ lx-plugin.yaml
  └─ snippets/_root.js
```
```yaml
# lx-plugin.yaml
name: MyPlugin
```
(`server.rootSnippet` defaults to `snippets/_root.js`, so it doesn't need to
be spelled out for a minimal plugin — see [config](#config) below for the
full list of defaults.)

A plugin structure typically looks like this:
```
/example_plugin
  ├─ /assets
  │   ├─ /css
  │   │   └─ MainCss.js
  │   ├─ /i18n
  │   │   └─ tr.yaml
  │   └─ /img
  │        └─ image.png
  ├─ /client
  │   ├─ /guiNodes
  │   │   ├─ MainBox.js
  │   │   └─ Popup.js
  │   └─ /src
  │       └─ Logic.js
  ├─ /server
  │   └─ example_plugin.go
  ├─ /snippets
  │   ├─ _root.js
  │   └─ popup.js
  ├─ lx-plugin.yaml
  ├─ Core.js
  └─ Plugin.js
```
* `snippets/` — server-rendered markup fragments, see [Frontend implementation](#frontend).
* `Plugin.js` — the client-side entry point, a subclass of `lx.Plugin`.
* `Core.js` / `client/guiNodes/*` — optional structuring helpers for a plugin's client logic, see [Frontend implementation](#frontend).
* `server/*.go` — the optional Go-side counterpart, see [Backend implementation](#backend).
* `assets/` — images, i18n file, extra CSS modules — anything referenced from the config below.


## <a name="config">The plugin config (`lx-plugin.yaml`)</a>

```yaml
name: ExamplePlugin

# Base path(s) for image names used by this plugin (see doc/components.md,
# lx.ImageManager). Short form (one default path):
images: assets/img
# ...or the full form, one base path per prefix:
# images:
#   default: assets/img
#   icons: path/to/icons

i18n: assets/i18n/tr.yaml

# Extra files/directories to include whenever the plugin is used (see the
# path syntax below) — merged with client.require into one list.
require:
  - assets/css/

# lx.PluginCssAsset subclasses to plug in (a plugin-level equivalent of a
# widget's initCss), declared via Plugin.getCssAssetClasses()
cssAssets:
  - MainCss

cacheType: inherit   # default; see plugins/plugin_cache.go for the other values

server:
  # Go DI key for a custom jspp.IPlugin implementation (optional — with no
  # key, or if the DI lookup fails, the plain base plugins.Plugin is used)
  key: namespace.ExamplePlugin
  file: Plugin.js                # default: Plugin.js
  rootSnippet: snippets/_root.js # default: snippets/_root.js
  snippets:
    - snippets                   # default: [snippets]
  # Alternative ways to reference a snippet by a short name from elsewhere,
  # see the path syntax below for what {plugin:...}/{snippet:...} mean
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
  require:
    - some/server-only/file.js

client:
  file: Plugin.js   # default: Plugin.js
  require:
    - Core.js
    - client/guiNodes/
    - -R client/src/
  core: Core   # class extending lx.PluginCore, see Frontend implementation
  guiNodes:    # classes extending lx.GuiNode, see Frontend implementation
    mainBox: MainBox
    somePopup: Popup

# Uses for rendering the plugin as a page
page:
  title: Title                    # default: "lx"
  icon: '@app/path/to/icon.png'   # default: none; '@...' resolved via the app's own Pathfinder, same as elsewhere in the framework
  template:
    namespace: ''
    block: 'content'
```

### Path syntax

A path in `require`/`client.require`/`server.require` (and in the identical
quoted-path argument to `lx.import(...)` used in ordinary modules, see
[Preprocessor features](https://github.com/epicoon/lxgo/tree/master/jspp/doc/pp.md#require))
can be:
* a plain file — `.js` is appended automatically if missing;
* a directory (trailing `/`) — every file directly inside it; add a `-R`
  flag to also recurse into subdirectories;
* combined with a `-F` (force) or `-U` (unwrapped) flag to change how the
  included code gets wrapped.

Two more path forms are specific to cross-plugin references (used by
`snippetsMap` above): `{plugin:Name}/rest/of/path` resolves relative to
another plugin's own root directory, and `{snippet:PluginName.Key}` refers to
a snippet that plugin registered under `Key` in its own `snippetsMap`.


## <a name="backend">Backend implementation</a>

A plugin's Go counterpart is optional — skip `server.key` entirely and the
plain base plugin (`plugins.Plugin`) is used, with no-op render hooks and no
ajax endpoints. To add custom logic, implement `jspp.IPlugin` by embedding
`*plugins.Plugin`:

```go
package example_plugin

import (
    "github.com/epicoon/lxgo/jspp"
    "github.com/epicoon/lxgo/jspp/plugins"
)

/** @interface jspp.IPlugin */
type ExamplePlugin struct {
    *plugins.Plugin
}

var _ jspp.IPlugin = (*ExamplePlugin)(nil)

/** @constructor jspp.CPlugin */
func NewExamplePlugin() jspp.IPlugin {
    return &ExamplePlugin{Plugin: plugins.NewPlugin()}
}

func (p *ExamplePlugin) BeforeRender() {
    // runs before the plugin's snippets are rendered - e.g. load data the templates need
}

func (p *ExamplePlugin) AfterRender(info *jspp.PluginRenderInfo) {
    // runs after rendering; info is {Html, Root, Lx, Assets{Modules,Scripts,Css}}
}
```

Register it in the app's DI container under the key referenced by the
plugin's own `server.key`:
```go
app.DIContainer().Init(kernel.CAnyList{
    "namespace.ExamplePlugin": func(args ...any) any {
        return example_plugin.NewExamplePlugin()
    },
})
```

For a plugin's own ajax endpoints, see [A plugin's own ajax endpoints](#ajax)
below.


## <a name="frontend">Frontend implementation</a>

Using plugins at all requires including the `lx.Plugin` module somewhere in
your app — once, typically in its entry file:
```js
lx.import(lx.Plugin);
```
This is what makes `lx.app.pluginManager` (and the smaller
`lx.app.snippetMap`) available — see [Application
components](https://github.com/epicoon/lxgo/tree/master/jspp/doc/components.md#comp20)
for what that component tracks and its full API; day-to-day you mostly reach
it indirectly, through `widget.setPlugin(...)` and the `lx.Plugin` API below,
rather than calling it directly.

### `Plugin.js` and the client lifecycle

`Plugin.js` (or whatever `client.file` points to) declares a subclass of
`lx.Plugin`:
```js
// @lx:namespace myNmsp;
class Plugin extends lx.Plugin {
    run() {
        // widgets, this.core and this.guiNodes are all ready here
    }
}
```
Once a plugin's server-rendered info reaches the browser — whether it was on
the page from the start, or fetched later and unpacked with
`widget.setPlugin({info})` (see [Loading a plugin on demand](#ondemand)) — it
goes through a fixed sequence:

1. The plugin instance is constructed from server-generated JS.
2. `plugin.beforeRender()` — a hook before its snippets become real widgets.
3. Snippets are unpacked into an actual `lx` widget tree (see below).
4. `plugin.beforeRun()` — the base implementation instantiates `this.core`
   (if `core` is configured or `getCoreClass()` returns one) and populates
   `this.guiNodes` (if `guiNodes` are configured or `getGuiNodeClasses()`
   returns any) — see "Core and GUI nodes" below.
5. `plugin.run()` — the plugin's own entry point, from the example above.

### Snippets

A **snippet** is a chunk of server-rendered markup for one region of the
plugin's page: real HTML plus the `lx` widget metadata needed to reconstitute
that HTML back into actual `lx.Rect`/`lx.Box` instances in the browser. On the
client, `SnippetLoader` walks a snippet's DOM subtree, turns every tagged
element back into its widget class and restores its properties/handlers, and
recurses into any nested snippets or child plugins it finds along the way.

### Core and GUI nodes

Two optional, related conventions for structuring a non-trivial plugin's
client code, both wired up automatically in `beforeRun()`:

* **Core** (`client.core: Core` in the config, a class extending
  `lx.PluginCore`) — a single composition root for the plugin's own logic,
  separate from `run()`. Override `getCoreClass()` to return your `Core`
  class; the instance ends up at `plugin.getCore()`/`this.core`. Its
  constructor runs `init()` → `loadReferences()` → `initHandlers()` →
  `subscribeEvents()` (all overridable no-ops) — a fixed place to look up
  widgets, wire up handlers, and subscribe to events, instead of doing it all
  inline in `run()`.
* **GUI nodes** (`client.guiNodes: {name: WidgetClassRef}` in the config,
  classes extending `lx.GuiNode`) — named wrappers around a specific widget
  found by key. Override `getGuiNodeClasses()` to map a name to a class;
  `initGuiNodes()` looks up `this.findOne(name)` for each and wraps it. A
  `GuiNode` gives that widget its own `show()`/`hide()` with
  `beforeShow`/`afterShow`/`beforeHide`/`afterHide` hooks, plus
  `getCore()`/`getPlugin()`/`getGuiNode(otherName)` for cross-references.
  Its constructor runs `init()` → `initHandlers()` → `subscribeEvents()`
  (all overridable no-ops) — a fixed place to look up widgets, wire up handlers,
  and subscribe to events, instead of doing it all inline in `run()`.
  Access one via `plugin.getGuiNode('mainBox')`.

### Other `lx.Plugin` API worth knowing

* `plugin.ajax(path, params)` — see [A plugin's own ajax endpoints](#ajax).
* `plugin.on(eventName, cb)` / `plugin.trigger(eventName, data)` — the
  plugin's own event dispatcher, independent of any widget's DOM-style
  events. `focus`/`blur` are handled specially, via
  `onFocus(cb)`/`onUnfocus(cb)`.
* `plugin.focus()` / `plugin.blur()` — mark this plugin as the "focused" one
  app-wide (`lx.app.pluginManager.getFocusedPlugin()`); enabled via
  `plugin.setFocusable(true)`. Useful for parallel keyboard processing.
* `plugin.getChildPlugins(all=false)` / `plugin.getChildPlugin(name, all=false)`
  — nested plugins (a snippet containing a child plugin gets unpacked
  automatically, with `parent` set to the outer one).
* `plugin.useModule(name, cb)` / `plugin.useModules(names, cb)` — lazily load
  a JS module the plugin didn't declare upfront (a thin wrapper over
  `lx.app.dependencies.promiseModules`, see [Application
  components](https://github.com/epicoon/lxgo/tree/master/jspp/doc/components.md)).
* `plugin.onKeydown(key, func)` / `onKeyup(key, func)` (and `off...`) —
  keyboard handlers scoped to this plugin instance.
* `plugin.onDestruct(callback)` — cleanup hook, run (along with an optional
  `destruct()` method) when the plugin is torn down, e.g. right before
  `setPlugin()` replaces it with a different one.


## <a name="page">Rendering a plugin as a page</a>

Register one or more plugins as full pages at given URLs:
```go
import (
    jsppComp "github.com/epicoon/lxgo/jspp/component"
    "github.com/epicoon/lxgo/jspp"
)

pp, _ := jsppComp.AppComponent(app)
pp.PluginManager().SetRoutes(jspp.PluginRoutesList{
    "/pl1": "FirstPlugin",
    "/pl2": "SecondPlugin",
})
```
This registers a handler on every listed URL that renders the whole HTML page
for that plugin — the plugin itself is unpacked via an inline `<script>`
calling `lx.app.root.setPlugin({info: {root, lx}})` on page load, and the
surrounding page shell comes from one of two places:

* by default, a minimal built-in HTML layout (just enough `<head>`/`<body>`
  to load `core.js` and run that script);
* or, if `page.template` is set, the host app's own [kernel
  Templates](https://github.com/epicoon/lxgo/tree/master/kernel/README.md#tpl)
  system. `page.template.namespace` picks one of the app's configured
  `Templates:` entries (see the kernel README's Templates section) and its
  `layout.html` is used as the page shell; `page.template.block` is the name
  of the `{{block "name" .}}` placeholder inside that layout where the
  plugin's rendered HTML and startup script get inserted — exactly the
  `content` block in the kernel README's own layout example. `title`/`icon`
  from the plugin's own `page` config are spliced into the layout's
  `<title>`/icon `<link>` either way. If `namespace` doesn't resolve to a
  configured layout, the plugin falls back to the built-in default instead.


## <a name="ondemand">Loading a plugin on demand</a>

To load a plugin into an already-running app (rather than as a whole page),
fetch its render info and unpack it into a target widget. This is a different
thing from [a plugin's own ajax endpoints](#ajax) below — that's for a
*loaded* plugin talking to its own backend; this is about obtaining a plugin
in the first place. Loading a brand-new plugin instance on demand isn't wired
up as a ready-made route; write a small handler using the same building block
the built-in page rendering above uses, `PluginManager.Render(plugin, lang)`:

```go
package handlers

import (
    jsppComp "github.com/epicoon/lxgo/jspp/component"
    "github.com/epicoon/lxgo/kernel"
    "github.com/epicoon/lxgo/kernel/http"
)

type PluginHandler struct {
    *http.Resource
}

func NewPluginHandler() kernel.IHttpResource {
    return &PluginHandler{Resource: http.NewResource()}
}

func (h *PluginHandler) Run() kernel.IHttpResponse {
    pp, err := jsppComp.AppComponent(h.App())
    if err != nil {
        return h.ErrorResponse(500, "jspp is not plugged")
    }

    // Plugin you want to receive
    myPluginName := "MyPlugin"

    plugin := pp.PluginManager().Get(myPluginName)
    if plugin == nil {
        return h.ErrorResponse(400, "plugin not found")
    }
    info, err := pp.PluginManager().Render(plugin, h.Lang())
    if err != nil {
        return h.ErrorResponse(500, err.Error())
    }

    return h.JsonResponse(kernel.JsonResponseConfig{
        Dict: kernel.Dict{"plugin": info},
    })
}
```
Register it like any other resource (see [`lxgo-kernel`'s
README](https://github.com/epicoon/lxgo/tree/master/kernel/README.md#link7)
for how request handlers get registered), then load it from the client:
```js
(new lx.HttpRequest('/get-my-plugin')).
    send().then(result => {
        // result.plugin is a full PluginRenderInfo (html/root/lx/assets) -
        // setPlugin() injects the html and loads the assets itself
        widget.setPlugin({info: result.plugin});
    });
```
`widget.setPlugin({info})` (defined on `lx.Box`) tears down any plugin
currently occupying `widget`, then hands `info` to
`lx.app.pluginManager.unpack(info, widget, ...)` — which injects `info.html`
into the widget, waits for `info.assets` (modules/scripts/css) to finish
loading, and runs the client lifecycle described in [Frontend
implementation](#frontend).


## <a name="ajax">A plugin's own ajax endpoints</a>

Once a plugin is loaded (by either method above), it can talk to its own
backend endpoints without any extra setup — routing for this is already
built into the framework (`/lx/plugin[POST]`, registered automatically, see
`component/component.go`). All a plugin needs to do is implement
`AjaxHandlers()`:
```go
func (p *ExamplePlugin) AjaxHandlers() kernel.HttpResourcesList {
    return kernel.HttpResourcesList{
        "path": NewPathHandler,
    }
}
```
From the client, `plugin.ajax(path, params)` sends `{plugin: name, path,
params, pluginParams}` to `/lx/plugin`; the built-in handler looks the plugin
up by name, finds the matching entry in its `AjaxHandlers()` map, and
delegates the request to that handler as if it had been called directly —
your handler's `Run()` sees `params` as its request body, nothing extra
required on your end.

> Plugin ajax handlers are `kernel.IHttpResource`, the exact same interface as
> the app's own request handlers — see [`lxgo-kernel`'s
> README](https://github.com/epicoon/lxgo/tree/master/kernel/README.md#link7).


## See also
* [Elements](https://github.com/epicoon/lxgo/tree/master/jspp/doc/elements.md) — the base every plugin is built on.
* [Modules](https://github.com/epicoon/lxgo/tree/master/jspp/doc/modules.md) — how a plugin's own JS is found/included as `@lx:module`.
* [Application components](https://github.com/epicoon/lxgo/tree/master/jspp/doc/components.md) — `lx.Dependencies`, used to lazily load a plugin's assets.
