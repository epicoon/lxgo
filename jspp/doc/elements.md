# Elements

`lx.Element` is the shared abstract base behind two fundamentally different
ways of building a piece of `lx` UI:

* a [**widget**](https://github.com/epicoon/lxgo/tree/master/jspp/doc/widgets.md)
  — a reusable tool for controlling and/or visualizing something (a button,
  an input, a table, ...), built on `lx.Rect`/`lx.Box`/`lx.TextBox`;
* a [**plugin**](https://github.com/epicoon/lxgo/tree/master/jspp/doc/plugins.md)
  — a logically self-contained mini-application, plugged into the host app as
  its own isolated piece.

Widgets and plugins otherwise have little in common — a widget wraps a single
DOM node and is meant to be composed with others; a plugin is a whole
separate unit with its own assets and (usually) its own backend counterpart.
What `lx.Element` gives both of them is three shared capabilities:
* **their own CSS** — a static `initCss(css)` hook, see [CSS](https://github.com/epicoon/lxgo/tree/master/jspp/doc/css.md);
* **the ability to emit their own named events** — a widget uses `lx.Rect`'s
  DOM-oriented event system, a plugin keeps its own internal event dispatcher,
  but conceptually both are "a piece of UI that fires events";
* **their own ajax channel to the backend** — for plugins, `plugin.ajax(path, params)`
  (see [Plugins](https://github.com/epicoon/lxgo/tree/master/jspp/doc/plugins.md#ajax));
  for any other element (typically a widget), `this.ajax(path, params)`, see
  below.

Class hierarchy:
* `lx.Element` — the abstract base described above.
* `lx.Rect`/`lx.Box`/`lx.TextBox` — the built-in widgets everything else is built
  on top of, see [Widgets](https://github.com/epicoon/lxgo/tree/master/jspp/doc/widgets.md)
  for their API.
* `lx.Plugin` — extends `lx.Element` directly (not `Rect`/`Box`), see [Plugins](https://github.com/epicoon/lxgo/tree/master/jspp/doc/plugins.md).


## An element's own ajax channel

Any element — in practice, any widget, since `lx.Plugin` has its own separate
mechanism (see [Plugins](https://github.com/epicoon/lxgo/tree/master/jspp/doc/plugins.md#ajax))
— can talk to its own backend endpoints via `this.ajax(path, params)`,
without going through the app's regular routing:
```js
this.ajax('ping').send().then(res => console.log(res));
```
This sends `{elem: this.constructor.getKey(), path, params}` to the built-in
`/lx/elem[POST]` route (registered automatically, same mechanism as a
plugin's `/lx/plugin`). `getKey()` defaults to the class's own namespaced
name (`lx.Button`, `myApp.MyWidget`, ...) — this is also the key the server
side looks the element up by, so the two have to match.

On the Go side you register your own implementation directly in the app's DI
container, under that same key:
```go
package mywidget

import (
    "github.com/epicoon/lxgo/jspp"
    "github.com/epicoon/lxgo/jspp/elems"
    "github.com/epicoon/lxgo/kernel"
)

type MyWidget struct {
    *elems.Element
}

var _ jspp.IElement = (*MyWidget)(nil)

func NewMyWidget() jspp.IElement {
    return &MyWidget{Element: elems.NewElement()}
}

func (w *MyWidget) AjaxHandlers() kernel.HttpResourcesList {
    return kernel.HttpResourcesList{
        "ping": NewPingHandler,
    }
}
```
```go
app.DIContainer().Register(kernel.CAnyList{
    "myApp.MyWidget": func(args ...any) any {
        return mywidget.NewMyWidget()
    },
})
```
(`Register` adds to the DI map without disturbing whatever else is already
registered there — e.g. your plugins' own DI entries; use `Init` instead only
if you mean to set up the whole DI map from scratch in one call.)

A handler registered this way is a plain `kernel.IHttpResource`, dispatched
to exactly like a plugin's own ajax handlers — see [A plugin's own ajax
endpoints](https://github.com/epicoon/lxgo/tree/master/jspp/doc/plugins.md#ajax)
for what a handler's `Run()` sees.
