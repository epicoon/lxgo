# Modules

A **module** is the unit of code the preprocessor resolves by name. Every
widget, css preset, and most other reusable pieces of `lx`-code are modules —
see [Elements](https://github.com/epicoon/lxgo/tree/master/jspp/doc/elements.md)
and [Widgets](https://github.com/epicoon/lxgo/tree/master/jspp/doc/widgets.md)
for what you typically build on top of them.


## Declaring a module

A file becomes a module by starting with `@lx:module <Name>;`:

```js
// @lx:module lx.Button;

// @lx:namespace lx;
class Button extends lx.Box {
    // ...
}
```

The directive itself is stripped from the compiled output — it only marks the
file for the module map (see below). A module can also carry metadata via
`@lx:module-data: key = value;`, most commonly a path to an i18n file:

```js
// @lx:module lx.Paginator;
// @lx:module-data: i18n = i18n.yaml;
```
(the path is resolved relative to the module's own file — see `js/modules/widgets/Paginator/Paginator.js`/`i18n.yaml`).

Modules that ship with the package itself live under `js/modules/*` (and are
found automatically, since the map builder scans the whole Go module). For
modules that belong to your application and aren't a Go package, list their
directories in `JSPreprocessor.Modules`:
```yaml
Components:
  JSPreprocessor:
    Modules:
      - /path/to/modules
```


## Using a module

Reference a module from another file by passing its name, **unquoted**, to
`lx.import(...)` (see [Preprocessor features](https://github.com/epicoon/lxgo/tree/master/jspp/doc/pp.md#require)
— an unquoted argument is a module, a quoted one is a file path):
```js
lx.import(lx.Button);

let button = new lx.Button({text: 'Submit'});
```
At compile time this argument is resolved through the module map: the
target module's file is pulled into the build, and its own `lx.import(...)`
module arguments are followed transitively, so you don't need to import
something your dependencies already import themselves.


## The modules map

Resolving a module argument requires knowing which file implements which
module name in advance — that mapping is the **modules map**: a JSON file
built by scanning every `.js` file reachable by the preprocessor (the current
Go module's dependencies, plus `JSPreprocessor.Modules` paths) for the
`@lx:module` directive, and stored under `Components.JSPreprocessor.MapsPath`.

The map has to be (re)built explicitly whenever you add a new module or
change one's declared name/dependencies — it isn't inferred on the fly during
a regular build:
```
go run . jspp:build-modules-map
```
(`jspp:build` rebuilds `core.js` together with both the modules and plugins
maps — see [Start using](https://github.com/epicoon/lxgo/tree/master/jspp/doc/start.md)
for the full list of `jspp:*` commands.) If an imported module isn't in the
map yet (e.g. you just added it and forgot to rebuild), the compiler logs
`Module '<name>' does not exist` and skips it.

To substitute one module name for another when a module argument is resolved
(without touching the files that reference it), use `ModuleInjector` — see
[Start using](https://github.com/epicoon/lxgo/tree/master/jspp/doc/start.md)
for its config syntax.
