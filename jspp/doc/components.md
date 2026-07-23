## JS-application components

A **component** is a functional unit that extends the capabilities of an application at build time. It can provide additional logic, configuration, services, or reusable resources to the application or its elements. Unlike UI [elements](https://github.com/epicoon/lxgo/tree/master/jspp/doc/elements.md), components are not necessarily visual but serve as modular building blocks that can be plugged into the application’s structure to add or modify behavior.

* You can get access to **component** with code:
  ```js
  // Where "componentKey" is the key of a component
  lx.app.componentKey
  ```

* Component initialization. The best practice is initialization by configuration file. You can set the path to the file in the application configuration using key `Components.JSPreprocessor.AppConfig`. Alternative way is calling application method `start` in code and passing configuration object as parameter.
  - Example initialization by configuration file (see [Start using](https://github.com/epicoon/lxgo/tree/master/jspp/doc/start.md), step 4, for the full file):
    ```yaml
    components:
      lang:
        en-EN: English
      imageManager:
        default: /img
      cssManager:
        default: lx.CssPresetDark
        white: lx.CssPresetWhite
    ```
  - Example initialization in code:
    ```js
    lx.app.start({
        components: {
            lang: {'en-EN': 'English'},
            imageManager: '/img',
            cssManager: lx.CssPresetDark
        }
    });
    // or, once the app is already running:
    lx.app.setupComponents({
        lang: {'en-EN': 'English'}
    });
    ```

  Only components extending `lx.AppComponentSettable` (rather than the plain
  `lx.AppComponent` base) actually accept settings this way — the rest (most
  of the ones listed below have no options at all) are simply always
  available as `lx.app.<key>`, and passing a config for one of them logs an
  error instead of doing anything. Each component's own section below says
  which is the case.


## Built-in components

* [lx.CssManager](#comp1)
* [lx.ImageManager](#comp2)
* [lx.Language](#comp3)
* [lx.Toast](#comp4)
* [lx.Storage](#comp5)
* [lx.Cookie](#comp6)
* [lx.Events](#comp7)
* [lx.Dependencies](#comp8)
* [lx.Mouse](#comp9)
* [lx.Keyboard](#comp10)
* [lx.Animation](#comp11)
* [lx.Alert](#comp12)
* [lx.DomSelector](#comp13)
* [lx.Queues](#comp14)
* [lx.FunctionHelper](#comp15)
* [lx.DomEvents](#comp16)
* [lx.DragNDrop](#comp17)
* [lx.Binder](#comp18)
* [lx.Dialog](#comp19)
* [lx.PluginManager](#comp20)
* [lx.SnippetMap](#comp21)

Almost all above are always present, registered unconditionally when the app
starts. The next two are different: they only exist once the `lx.Plugin`
module is in use (`lx.import(lx.Plugin);` somewhere in your app, typically its
entry file) — that module registers them itself via `lx.app.registerComponents(...)`,
the same mechanism described in [Add custom component](#custom) below.


### <a name="comp1">lx.CssManager</a>
- key: `cssManager`
- available: `client` `server`
#### Description
Manages CSS generation for the whole app: named **scopes** (see [CSS](https://github.com/epicoon/lxgo/tree/master/jspp/doc/css.md#manager)) that pair a preset with an optional class-name prefix, and lazily generate/inject each widget's own CSS (`initCss(css)`) into the right scope the first time that widget class is used in it.
#### Initialization
- Short form — one default (unprefixed) scope, plus optional named ones:
  ```yaml
  components:
    cssManager:
      default: lx.CssPresetDark
      white: lx.CssPresetWhite
  ```
  ```js
  lx.app.setupComponents({
      cssManager: {
          default: lx.CssPresetDark,
          white: lx.CssPresetWhite
      }
  });
  ```
- Full form — needed when the default scope itself should also have a
  name/prefix:
  ```yaml
  components:
    cssManager:
      defaultPreset: dark
      scopes:
        dark: lx.CssPresetDark
        white: lx.CssPresetWhite
  ```
- Directly in code, one scope at a time: `lx.app.cssManager.createPresetScope(lx.CssPresetDark)` (default, unprefixed), `lx.app.cssManager.createPresetScope(lx.CssPresetWhite, 'white')` (named, prefixed).

See [CSS](https://github.com/epicoon/lxgo/tree/master/jspp/doc/css.md) for what a preset/scope actually is and how a widget's CSS ends up prefixed by scope.
#### Methods
* `createPresetScope(preset, prefix = '')` — register a scope (also registers the preset under that key, if it isn't already).
* `getScope(name = '')` / `getScopeNames()` — look up a registered scope / list every scope name.
* `getPreset(name = null)` / `getPresetName()` — look up a registered preset (the default preset if no name is given).
* `updatePreset(name, params)` — patch a preset's token values at runtime and refresh every scope currently using it.
* `addElement(elem, scopeName = null)` / `addElements(elems, scopeName = null)` — force-generate a widget class's CSS for a scope (this normally happens automatically the first time the widget is used); `scopeName = null` applies it to every scope.
* `pack()` *(server only)* — serialize every scope's built CSS (prefix, preset name, involved element classes, class names, css text), used for `CssScopeRenderSide: server` builds (see [CSS](https://github.com/epicoon/lxgo/tree/master/jspp/doc/css.md#csrs)).


### <a name="comp2">lx.ImageManager</a>
- key: `imageManager`
- available: `client` `server`
#### Description
Resolves a short image name into an actual path/URL, given a per-prefix
mapping — used by widgets/elements that accept an image name instead of a
full path.
#### Initialization
```yaml
components:
  imageManager:
    default: /img
    icons: /assets/icons
```
```js
lx.app.setupComponents({
    imageManager: {default: '/img', icons: '/assets/icons'}
});
```
The config is a map from prefix to base path. A bare name (`icon.png`)
resolves against `default`. A name like `@icons/star.png` resolves against the
`icons` entry, joined with the rest of the path (`@icons/star.png` →
`/assets/icons/star.png`) and, for `@`-prefixed names only, prefixed with
`lx.app.getProxy()` if a proxy is configured. A name that already starts with
`/` is returned unchanged.
#### Methods
* `getPath(ctx, name)` — resolve `name` to a path. `ctx` is the element asking
  for it: its own `ctx.imagePaths` (if set) is tried first, then the image
  paths of the plugin it belongs to (if any), then this component's own
  config — the first mapping that resolves `name` wins.


### <a name="comp3">lx.Language</a>
- key: `lang`
- available: `client` `server`
#### Description
Tracks which language the running app is currently showing, and the set of
languages it offers. This is distinct from the file-based i18n lookups used
for module/plugin translations (`lx.i18n(...)`, see [Preprocessor
features](https://github.com/epicoon/lxgo/tree/master/jspp/doc/pp.md)) — this
component is about the *current selection*, not about resolving translated
strings.
#### Initialization
```yaml
components:
  lang:
    default: en-EN
    options:
      en-EN: English
      ru-RU: Russian
```
```js
lx.app.setupComponents({
    lang: {default: 'en-EN', options: {'en-EN': 'English', 'ru-RU': 'Russian'}}
});
```
#### Methods
* `current()` — the active language: the `lxlang` cookie if set, else the
  configured `default`, else `en-EN`.
* `options()` — the configured `options` map (or `{'en-EN': 'English'}` if
  none was given).
* `set(val)` — switches the language: stores `val` in the `lxlang` cookie and
  reloads the page. `val` must be `en-EN` or a key of `options`; anything else
  logs an error and is ignored.

`current()`/`set()` read/write `lx.app.cookie`, a client-only component (see
[lx.Cookie](#comp6)) — even though `lx.Language` itself is registered on both
sides, these two methods only work in the browser.


### <a name="comp4">lx.Toast</a>
- key: `toast`
- available: `client`
#### Description
Shows short-lived, styled "toast" notifications (message/warning/error) stacked in a corner of the page. Installs global shortcuts `lx.toastMessage(msg)`, `lx.toastWarning(msg)`, `lx.toastError(msg)`.
#### Initialization
- Via yaml (see [Start using](https://github.com/epicoon/lxgo/tree/master/jspp/doc/start.md), step 4):
  ```yaml
  components:
    toast:
      message: my-toast-message-css-class   # custom CSS class instead of the built-in green style
      warning: my-toast-warning-css-class
      error: my-toast-error-css-class
      lifetime: 4000                       # ms before a toast auto-removes itself (default 3000, falsy = stays until clicked)
      defaultType: message                 # 'message' | 'warning' | 'error', used when a call doesn't specify a type
      width: 320px                         # max width (default '40%')
  ```
- Or in code: `lx.app.setupComponents({toast: {lifetime: 4000, ...}})`.

  All keys are optional — with no config at all, `lx.Toast` still works using its built-in colors/timing.
#### Methods
* `message(msg)` / `warning(msg)` / `error(msg)` — show a toast of that type; `msg` is a string (or array of strings, joined with spaces).
* `align(horizontal, vertical)` (or `align({horizontal, vertical, direction, indent})`) — reposition the toast stack (default: top-left, stacked vertically, `10px` gaps).

  A toast auto-removes after `lifetime()` ms (0/falsy = stays until clicked) and can always be dismissed early by clicking it.


### <a name="comp5">lx.Storage</a>
- key: `storage`
- available: `client`
#### Description
Wrapper around `localStorage`/`sessionStorage`, with values transparently JSON-encoded/decoded and an optional in-memory read cache.
#### Initialization
No configuration — always available as `lx.app.storage`.
#### Methods
* `get(key)` / `set(key, value)` / `remove(key)` / `clear()` — backed by `localStorage`, values JSON-encoded.
* `sessionGet(key)` / `sessionSet(key, value)` / `sessionRemove(key)` / `sessionClear()` — same, backed by `sessionStorage`.
* `useCache(bool = true)` — turn an in-memory read cache on/off (shared by `get`/`sessionGet`); write methods keep the cache in sync whenever it's on.


### <a name="comp6">lx.Cookie</a>
- key: `cookie`
- available: `client`
#### Description
Thin wrapper around `document.cookie`.
#### Initialization
No configuration — always available as `lx.app.cookie`.
#### Methods
* `get(name)` / `set(name, value, options = {})` / `remove(name)` — `options` passed to `set` become cookie attributes: `expires` is a number of seconds from now (or a `Date`); any other key (e.g. `path`/`domain`) is written as `key=value`, or bare if its value is `true`.
* `getNames()` — names of every cookie currently set.
* `removeAll()` — removes every cookie returned by `getNames()`.


### <a name="comp7">lx.Events</a>
- key: `events`
- available: `client`
#### Description
A small named-event pub/sub bus for the application itself — distinct from a widget's own DOM-style events (`widget.on('click', ...)`, see [Elements](https://github.com/epicoon/lxgo/tree/master/jspp/doc/elements.md)) and from the internal `domEvents` component that powers those (see [lx.DomEvents](#comp16)). Currently used by the framework itself for the ajax request lifecycle (`lx.EVENT_BEFORE_AJAX_REQUEST` = `'beforeAjax'`, `lx.EVENT_AJAX_REQUEST_UNAUTHORIZED` = `'ajaxUnauthorized'`, `lx.EVENT_AJAX_REQUEST_FORBIDDEN` = `'ajaxForbidden'`), but you can register and fire your own event names through it the same way.
#### Initialization
No configuration — always available as `lx.app.events`.
#### Methods
* `on(eventName, callback)` — subscribe.
* `off(eventName, callback)` — unsubscribe.
* `trigger(eventName, data = {})` — call every subscriber with a new `lx.Event(eventName, data)` (`event.getData()` returns `data`).


### <a name="comp8">lx.Dependencies</a>
- key: `dependencies`
- available: `client` `server`
#### Description
Reference-counts which modules/scripts/css assets are currently "in use" by
the page, so a shared resource isn't loaded twice and can be dropped once
nothing needs it anymore. On the client it also drives on-demand loading —
lazily fetching a module that wasn't part of the initial `core.js`/app bundle,
instead of listing it in `lx.import(...)` ahead of time.
#### Initialization
No configuration — always available as `lx.app.dependencies`.
#### Methods
* `depend(map)` / `independ(map)` — increment/decrement the reference count
  for a `{modules, scripts, css}` map of names; a resource is only actually
  dropped once its count reaches 0 (and only if caching is off, see below).
* `defineNecessary(map)` / `defineNecessaryModules(list)` / `defineNecessaryCss(list)` / `defineNecessaryScripts(list)` — filter a list down to the names that aren't already tracked (i.e. actually need loading).
* `cache` *(property, default `true`)* — when `false`, a css/script asset whose
  reference count drops to 0 is removed from the page (its `<link>`/`<script>`
  tag is deleted); modules are always cached regardless of this flag.
* *(client only)* `promiseModules(config)` — fetch and evaluate one or more
  modules that aren't loaded yet, via a `get-modules` service request (see
  [Preprocessor features](https://github.com/epicoon/lxgo/tree/master/jspp/doc/pp.md)),
  then run `config.callback` once ready:
  ```js
  lx.app.dependencies.promiseModules({
      modules: ['lx.Calendar'],
      depend: true,          // also call depend() for them once loaded
      callback: () => { /* lx.Calendar is usable now */ }
  });
  ```
* *(client only)* `promiseScripts(config)` / `promiseCss(config)` — same idea
  for plain `<script>`/`<link>` assets (by URL, via `lx.AssetRequest`), not
  `lx`-modules.


### <a name="comp9">lx.Mouse</a>
- key: `mouse`
- available: `client`
#### Description
Global pointer tracker: last known mouse/touch position plus subscriptions
for move/down/up, so any code can react to pointer activity without
attaching its own DOM listeners.
#### Initialization
No configuration — extends the plain `lx.AppComponent` (no settings support).
Always available as `lx.app.mouse` once the app has started.
#### Methods
- `x` / `y` — last known pointer position (page coordinates, getters).
- `getPosition(context = null)` — `{x, y}`; without `context`, same as
  `{x, y}`; with an element `context`, position relative to that element's
  top-left corner.
- `onDown(f)` / `onMove(f)` / `onUp(f)` — subscribe `f(event)` to
  `mousedown`/`touchstart`, `mousemove`/`touchmove`, `mouseup`/`touchend`
  respectively.
- `offDown(f)` / `offMove(f)` / `offUp(f)` — unsubscribe.


### <a name="comp10">lx.Keyboard</a>
- key: `keyboard`
- available: `client`
#### Description
Global keyboard state and key-specific event subscriptions (by key code or
character) — used e.g. by widgets that react to Enter/Escape while focused.
#### Initialization
No configuration — extends the plain `lx.AppComponent`. Available as
`lx.app.keyboard`; global key tracking is switched on automatically by
`lx.app.start()` (`setWatchForKeypress(true)`), no manual setup needed.
#### Methods
- `onKeydown(key, func, context)` / `onKeyup(key, func, context)` — subscribe
  `func` to a specific key (number = key code, string = character), or pass
  an object of `{key: func}` pairs at once.
- `offKeydown(key, func, context)` / `offKeyup(key, func, context)` — unsubscribe.
- `keyPressed(key)` — whether a key (code or character) is currently held down.
- `shiftPressed()` / `ctrlPressed()` / `altPressed()` — shortcuts for the
  corresponding modifier key codes.
- `pressedCount()` — how many keys are currently held.
- `resetKeys()` — clear the pressed-key state.
- `setWatchForKeypress(bool)` — turn global keydown/keyup tracking on/off
  (already on by default via `lx.app.start()`).


### <a name="comp11">lx.Animation</a>
- key: `animation`
- available: `client`
#### Description
A single shared `requestAnimationFrame` loop the whole app runs on —
per-frame callbacks and `lx.Timer` instances (`js/src/client/tools/Timer.js`)
hook into it instead of each starting their own loop.
#### Initialization
No configuration — extends the plain `lx.AppComponent`. Available as
`lx.app.animation`; `useTimers(true)` and `useAnimation()` are called
automatically by `lx.app.start()`.
#### Methods
- `addAction(f, ctx)` / `resetActions()` — register a plain per-frame
  callback (`f.call(ctx)` every frame), or clear all registered actions.
- `addTimer(timer)` / `removeTimer(timer)` — register/unregister an
  `lx.Timer` instance (what `Timer.start()`/`stop()` call under the hood).
- `useTimers(bool)` / `useAnimation()` — toggle the timer registry / start
  the `requestAnimationFrame` loop (called automatically by `lx.app.start()`
  — not something you'd normally call yourself).


### <a name="comp12">lx.Alert</a>
- key: `alert`
- available: `client`
#### Description
Shows a blocking-ish modal message box (built on the `lx.ActiveBox` widget) — for quick "dump this to the screen" debugging/notices rather than styled user-facing alerts (compare with [lx.Toast](#comp4) for the latter). Installs a global shortcut function `lx.alert(msg)`.
#### Initialization
No configuration.
#### Methods
* `print(msg)` — same as calling the global `lx.alert(msg)`: opens an `lx.ActiveBox` titled "Alert" with `msg` inside a `<pre>` (so multi-line/object-dump-style strings keep their formatting), closable via its header button.


### <a name="comp13">lx.DomSelector</a>
- key: `domSelector`
- available: `client`
#### Description
A small set of DOM-lookup helpers for finding `lx` elements/widgets from plain HTML: by a partial set of attributes (`getElementByAttrs`), by `id` (`getWidgetById`), or by `name`/CSS class (`getWidgetsByName`/`getWidgetsByClass`/`getWidgetByClass`). Useful when you're handed a raw DOM node (e.g. from a browser event outside of `lx`'s own widget tree) and need the `lx`-side wrapper for it.
#### Initialization
No configuration — the component has no settings, it's always available as `lx.app.domSelector`.
#### Methods
* `getElementByAttrs(attrs, parent = null)` — `attrs` is an object like `{attr: value}`, plain DOM `querySelector` by a dict of `[attr^='value']` conditions, optionally scoped to a parent element.
* `getWidgetById(id, type = null)` / `getWidgetByClass(className, type = null)` — find a single HTML element by `id`/class and return its `lx` widget wrapper (existed instance if it's already an `lx` element, otherwise creates a wrapping widget according to `type` — `lx.Box` by default).
* `getWidgetsByName(name, type = null)` / `getWidgetsByClass(className, type = null)` — same, but return an `lx.Collection` of every match.


### <a name="comp14">lx.Queues</a>
- key: `queues`
- available: `client`
#### Description
A simple named-queue task runner: tasks added under the same queue name run
one at a time, in the order they were added, driven by the app's animation
frame loop rather than `setTimeout`. It's used internally by `lx.Timer`'s
`syncStart()` (`js/src/client/tools/Timer.js`) to serialize a timer's actions
through a queue, and is available directly for the same "run these one after
another" pattern.
#### Initialization
No configuration — queues are created on demand by name.
#### Methods
* `add(queueName, task)` — adds an `lx.Task` to the named queue, creating the
  queue if it doesn't exist yet.
* `remove(queue)` — removes a queue, given its name or an `lx.Queue` instance.

A queue is usually not created directly: construct `new lx.Task(queueName,
callback)` and it registers itself via `lx.app.queues.add(queueName, task)`.
The queue runs the first task in line (`task.run()`), and the task must call
`task.setCompleted()` when done so the queue advances to the next one. A queue
keeps itself alive while it has tasks and removes itself once empty (unless
created with `type: lx.Queue.TYPE_CONSTANT` instead of the default
`TYPE_TEMPORARY`).


### <a name="comp15">lx.FunctionHelper</a>
- key: `functionHelper`
- available: `client` `server`
#### Description
A small toolbox for building and calling functions dynamically from data —
used internally wherever a widget/LXML config value can be either a plain
function or serialized as source text (e.g. `#method(args)` calls, or a
config option documented as "function or `[thisArg, function]`"), and
available as a component if you need the same tricks yourself.
#### Initialization
No configuration — just call its methods.
#### Methods
* `callFunction(data, args=[])` — calls `data` if it's a function, or
  `data[1].apply(data[0], args)` if `data` is a `[thisArg, function]` pair.
* `createFunction(args, code)` — builds a `Function` from a parameter list and
  a code body (thin wrapper over the `Function` constructor).
* `createAndCallFunction(argNames, code, context=null, args=[])` — builds such a
  function and calls it immediately with `context`/`args`.
* `createAndCallFunctionWithArguments(namedArgs, code, context=null)` — same,
  but `namedArgs` is a `{name: value}` map instead of a positional arg list.
* `stringToFunction(str)` / `functionToString(func)` — convert between a
  function and its source-text form (`"(args)=>body"` or
  `"function(args){body}"`).
* `isEmptyFunction(func)` — true if the function's body is blank.


### <a name="comp16">lx.DomEvents</a>
- key: `domEvents`
- available: `client`
#### Description
A low-level, cross-browser DOM event listener utility (`addEventListener` vs
old-IE `attachEvent`, event object normalization, etc.) — this is the
internal plumbing behind a widget's own `.on()`/`.off()`/`.trigger()` (see
[Elements](https://github.com/epicoon/lxgo/tree/master/jspp/doc/elements.md)),
not something application code normally calls directly.
#### Initialization
No configuration — always available as `lx.app.domEvents`.
#### Methods
* `add(el, type, handler)` / `remove(el, type, handler = undefined)` — attach/detach a raw DOM event listener on a plain DOM node (`remove(el, type)` without a handler removes every handler for that type; `remove(el)` without a type removes everything).
* `has(el, type, handler)` — check whether a listener is currently attached.


### <a name="comp17">lx.DragNDrop</a>
- key: `dragNDrop`
- available: `client`
#### Description
The shared engine behind an element's `move` config (see
[Elements](https://github.com/epicoon/lxgo/tree/master/jspp/doc/elements.md)):
tracks the currently dragged element and updates its position on pointer
move, respecting `xLimit`/`yLimit`/`moveStep` and parent-move/parent-resize
options.
#### Initialization
No configuration — extends the plain `lx.AppComponent`. Available as
`lx.app.dragNDrop`; `useElementMoving()` is called automatically by
`lx.app.start()`.
#### Methods
In practice you rarely call these directly — just set an element's `move`
config and `lx.DragNDrop` handles the rest:
- `move(event)` — starts a drag; called internally by an element's own
  mousedown/touchstart handler when it has a `move` config.
- `resetDelta(elem)` — recompute the pointer-to-element offset without
  starting a new drag.
- `useElementMoving(bool = true)` — attach/detach the global
  mousemove/mouseup listeners that drive dragging
  (already on by default via `lx.app.start()`).


### <a name="comp18">lx.Binder</a>
- key: `binder`
- available: `client`
#### Description
`lx.Binder` is the internal engine behind model ↔ widget data binding — it's what actually runs when you call `model.bind(widget)`/`widget.bind(model)` (see [Models and binding](https://github.com/epicoon/lxgo/tree/master/jspp/doc/models.md)). It supports four bond kinds: a single model field ↔ a single widget, a model's fields ↔ a form-like widget with matching `_field` children, a collection of models ↔ one widget (aggregated — shared fields disabled when values differ across the collection), and a collection of models ↔ a matrix-widget (a child box is created/removed per collection item).
#### Initialization
No configuration — it's a stateless dispatch service, nothing to set up beyond using it.
#### Methods
In practice you don't call these directly — `model.bind()`/`widget.bind()` and `widget.addMatrixItemRender()` (see [Models and binding](https://github.com/epicoon/lxgo/tree/master/jspp/doc/models.md)) cover normal usage. For reference, the component itself exposes:
* `bind(obj, widget, type = lx.Binder.BIND_TYPE_FULL)` / `unbind(obj, widget = null)` / `unbindWidget(widget)` — simple field ↔ widget binding.
* `refresh(obj, fieldName = null)` — re-push current field value(s) into bound widgets.
* `bindMatrix(collection, widget, type)` / `unbindMatrix(widget)`, `setMatrixItemBox(obj, [widgetClass, config])`, `setMatrixItemRender`/`addMatrixItemRender(obj, render)`, `makeWidgetMatrix(obj, info)` — matrix (collection ↔ generated children) binding.
* `bindAggregation(collection, widget, type)` — one widget shared by every item of a collection.
* `getBind(id)` — look up the raw bond record by its internal id.
* Bind type constants: `lx.Binder.BIND_TYPE_FULL`, `lx.Binder.BIND_TYPE_WRITE`, `lx.Binder.BIND_TYPE_READ`.


### <a name="comp19">lx.Dialog</a>
- key: `dialog`
- available: `client`
#### Description
The actual HTTP transport underneath `lx.HttpRequest`/`lx.Request` (see [Also
worth knowing](https://github.com/epicoon/lxgo/tree/master/jspp/README.md) in
the package README) and, through it, `lx.ServiceRequest`/`lx.ElementRequest`
(see [Elements](https://github.com/epicoon/lxgo/tree/master/jspp/doc/elements.md))
— you normally reach it indirectly by using one of those, not by calling this
component directly.
#### Initialization
No configuration — always available as `lx.app.dialog`.
#### Methods
* `request(config, ignoreEvents = [])` — send a request; `config` is
  `{url, method = 'get', data, headers, success, waiting, error}`. Same-origin
  requests go through `XMLHttpRequest`; cross-origin ones fall back to
  `fetch(..., {mode: 'cors'})`.
* `get(config)` / `post(config)` — shortcuts that set `config.method`.
* `move(path)` — `window.location.pathname = path` (same-site redirect).
* `requestParamsToString(params)` / `requestParamsFromString(str)` — encode/decode a plain object to/from a `key=value&...` query string.

A same-origin request that comes back `401`/`403` fires
`lx.EVENT_AJAX_REQUEST_UNAUTHORIZED`/`lx.EVENT_AJAX_REQUEST_FORBIDDEN` on
[`lx.Events`](#comp7) (unless the event name is listed in `ignoreEvents`) —
`lx.app.events.on(lx.EVENT_AJAX_REQUEST_UNAUTHORIZED, ...)` is a convenient
place to redirect to a login page, for example.


### <a name="comp20">lx.PluginManager</a>
- key: `pluginManager`
- available: `client` `server` (most methods below are client-only — noted where relevant)
#### Description
Tracks every currently-loaded [plugin](https://github.com/epicoon/lxgo/tree/master/jspp/doc/plugins.md)
instance and drives turning a plugin's server-rendered info into a live
widget tree in the browser. Only registered once the `lx.Plugin` module is in
use (see the note above the component list) — not one of the always-present
components.

Not to be confused with the Go-side `plugins.PluginManager` (a different
type, in a different language, reached via `pp.PluginManager()` — see
[Plugins](https://github.com/epicoon/lxgo/tree/master/jspp/doc/plugins.md)).
This is the client-side JS component of (mostly) the same name.
#### Initialization
No configuration — always available as `lx.app.pluginManager` once `lx.Plugin`
is used.
#### Methods
* `get(key)` / `getList()` — look up a loaded plugin by its instance key / get every loaded plugin.
* *(client only)* `unpack(info, el, parent = null, clientCallback = null)` — what `widget.setPlugin({info})` calls under the hood, see [Plugins](https://github.com/epicoon/lxgo/tree/master/jspp/doc/plugins.md#ondemand).
* *(client only)* `remove(plugin)` — tear down a plugin (by instance or key): runs its destroy callbacks/`destruct()`, unbinds its dependencies, removes its child plugins.
* *(client only)* `focus(plugin)` / `blur(plugin = null)` / `getFocusedPlugin()` — track which single plugin is currently "focused" app-wide (see `plugin.focus()` in [Plugins](https://github.com/epicoon/lxgo/tree/master/jspp/doc/plugins.md)).
* `getPlugin(ctx)` / `getSnippet(ctx)` / `getRootSnippet(ctx)` — given an element inside a plugin's tree, find the owning plugin instance / the nearest snippet / that plugin's root snippet. (These are what a snippet's own generated code uses internally as `$plugin`/`$snippet`.)


### <a name="comp21">lx.SnippetMap</a>
- key: `snippetMap`
- available: `client` `server`
#### Description
A tiny name → function registry for [snippets](https://github.com/epicoon/lxgo/tree/master/jspp/doc/plugins.md#frontend).
Registered alongside `pluginManager` by the `lx.Plugin` module; internal
plumbing, not something you typically call yourself.
#### Initialization
No configuration.
#### Methods
* `registerSnippet(name, func)` / `getSnippet(name)` — register/look up a snippet-rendering function by name.


## <a name="custom">Add custom component</a>

Any class extending `lx.AppComponent` (or `lx.AppComponentSettable`, if it
should accept configuration the way the built-ins above do) can be added as
an app component with `lx.app.registerComponents({key: YourClass, ...})` —
after that it's available as `lx.app.key`, exactly like a built-in one. This
is a different operation from configuring an existing component (see the
`components:`/`setupComponents()` examples at the top of this page) — it adds
a brand new one.

This is how the plugin system itself registers its own components, at module
load time (top-level code in the module file, not inside a class):
```js
lx.import('components/');
lx.app.registerComponents({
    pluginManager: lx.PluginManager,
});
```
