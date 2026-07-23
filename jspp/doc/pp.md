# Preprocessor features

`JSPreprocessor` recognizes a set of special markers in your JS source files
— mostly `@lx:something`, plus a couple of standalone shorthands (`lx.self(...)`,
`lx>>>NAME`, `lx(...)>...`) — and rewrites them at build time into plain JS,
before the code ever reaches the browser (or, for server rendering, before
it's run at all). None of this syntax exists at runtime; it's purely a
build-time transformation.

A directive can be commented out (`// @lx:namespace app;`) and still works —
the very first thing the preprocessor does is strip a leading `// ` in front
of any `@lx:` directive, so editors/linters that don't understand this syntax
can still be kept happy without disabling the directive. `lx.import(...)` (see
[Including other files](#require)/[Modules](#modules) below) doesn't need
this at all — it's valid JS syntax on its own, so it's written bare, without a
comment wrapper.

A file mixing several directives together, to show the general idea:
```js
lx.import(lx.Box);

// @lx:namespace app;
class Greeter extends lx.Box {
    @lx:const DEFAULT_NAME = 'World';

    render() {
        this.text(lx.i18n('greeting', {name: lx.self(DEFAULT_NAME)}));
    }
}
```

## Contents
* [Namespaces and classes](#classes)
* [Including other files](#require)
* [Modules](#modules)
* [Conditional compilation](#conditional)
* [Extra assets](#assets)
* [Embedding data](#data)
* [Macros](#macros)
* [Markdown](#md)
* [Markup language (LXML)](#lxml)
* [Internationalization](#i18n)
* [Forwarding a backend config parameter](#config)


## <a name="classes">Namespaces and classes</a>

`@lx:namespace Name;`, written right before (or, with a blank/comment line
between, right above) a `class` declaration, registers that class globally
under `Name.ClassName` instead of leaving it a bare top-level identifier:
```js
// @lx:namespace app;
class UserModel { /* ... */ }
// compiles to roughly:
// (()=>{
//   lx.createNamespace('app');
//   if ('UserModel' in lx.globalContext.app) return;  // already defined - skip
//   class UserModel { ... }
//   UserModel.__namespace='app'; lx.globalContext.app.UserModel=UserModel;
// })();
```
This is what makes `new app.UserModel()` (or `lx.Box`, `lx.CssPresetDark`,
etc. — practically every class in this package) resolve as a namespaced
global rather than requiring an explicit import. A class with no
`@lx:namespace` in front of it is left as a plain top-level declaration.

Inside a class body, `@lx:const NAME = value;` declares a read-only class
constant:
```js
class ModelTypeEnum {
    @lx:const NUMBER = 'number';
    @lx:const STRING = 'string';
}
// compiles to:
// class ModelTypeEnum {
//   static get NUMBER(){return 'number';}
//   static get STRING(){return 'string';}
// }
```

Two more shorthands, usable anywhere (not just inside a class):
* `lx.self(NAME)` → `this.constructor.NAME` — read a static member from an
  instance method without spelling out the class name.
* `lx(expr)>key`/`lx(expr)>>key` — a shorthand for the tree lookups described
  in [Widgets](https://github.com/epicoon/lxgo/tree/master/jspp/doc/widgets.md#keys-and-lookup):
  a single `>` is `.get('key')`, a double `>>` is `.find('key')`, and they
  chain:
  ```js
  lx(box)>>item          // box.find('item')
  lx(box)>child>>deep    // box.get('child').find('deep')
  ```


## <a name="require">Including other files</a>

`lx.import('path')` inlines another file's (already-compiled) code in place,
skipping a file that's already been included elsewhere in the same build:
```js
lx.import('Core.js');
lx.import('-R client/src/');
```
`path` (a quoted string argument — see [Modules](#modules) below for what an
*un*quoted argument means) can be:
* a plain file — `.js` is appended automatically if missing;
* a directory (trailing `/`) — every `.js` file directly inside it; add a
  `-R` flag to also recurse into subdirectories.

A flag is glued to its path with a space, on either side —
`'-R client/src/'` and `'client/src/ -R'` are equivalent. Other flags: `-F`
(force) includes the file even if it was already compiled elsewhere in this
build; `-U` (unwrapped) skips IIFE-wrapping the included file's code — used
when importing code that isn't itself a self-contained module. Combine
letters in one flag (`-RF`) or repeat them on either side of the path;
whichever is more readable.

`lx.import(...)` takes any number of arguments, mixing paths and modules (see
below) freely in one call:
```js
lx.import('client/guiNodes/', lx.Box, 'Core.js', lx.Button);
```
A comment (`// lx.import(...)`) disables the call, same as any other JS
statement — this isn't special-cased, it falls out of the normal
comment-stripping that happens before this directive is resolved, and works
per-line if several calls are on separate lines.

This is the same mechanism (and the same path syntax) behind a plugin's
`require`/`client.require`/`server.require` config keys — see
[Plugins](https://github.com/epicoon/lxgo/tree/master/jspp/doc/plugins.md#config).


## <a name="modules">Modules</a>

`@lx:module Name;` and `@lx:module-data: key = value;` mark a file as
implementing a named module; referencing that name as an **unquoted**
argument to `lx.import(...)` — `lx.import(Name)` — is the more structured
alternative to a path argument, resolved through the modules map rather than
a literal file path. See
[Modules](https://github.com/epicoon/lxgo/tree/master/jspp/doc/modules.md)
for how they work; not repeated here.


## <a name="conditional">Conditional compilation</a>

Two block directives strip code depending on how the current build is
configured — the check happens once per build, so unmatched blocks simply
don't exist in the output, rather than being wrapped in a runtime `if`:

* `@lx:<context SIDE: ... @lx:context>` keeps the block only when compiling
  for that side (`CLIENT` or `SERVER`) — used constantly to give a widget or
  component different code on each side:
  ```js
  render(config) {
      // runs on both sides
  }

  // @lx:<context CLIENT:
  clientRender(config) {
      // browser-only: DOM events, etc.
  }
  // @lx:context>
  ```
* `@lx:<mode NAME: ... @lx:mode>` keeps the block only when
  `Components.JSPreprocessor.Mode` (see [Start
  using](https://github.com/epicoon/lxgo/tree/master/jspp/doc/start.md)) is
  set to that exact string — in practice, almost always used with `DEV` to
  strip debug-only code from a production build:
  ```js
  // @lx:<mode DEV:
  console.log('extra diagnostics, dropped outside DEV mode');
  // @lx:mode>
  ```


## <a name="assets">Extra assets</a>

`@lx:js path;` / `@lx:css path;` register a plain (non-`lx`) JS or CSS file
as an asset the current build depends on, and strip the directive itself
from the output — a way for a module to say "load this alongside me" without
that file going through the JS-module compilation pipeline at all:
```js
@lx:js vendor/chart.min.js;
@lx:css vendor/chart.min.css;
```


## <a name="data">Embedding data</a>

`lx.json('path')` / `lx.yaml('path')` inlines a JSON or YAML file's content
as a JS literal at build time:
```js
const config = lx.json('data/defaults.json');
```
The path is resolved the same way as elsewhere in the preprocessor (relative
to the app's own path configuration). If the file can't be read or parsed,
the directive is replaced with `null` and a build-time error is logged.


## <a name="macros">Macros</a>

`@lx:macros NAME { ... }` declares a named block of code without emitting it
in place; every occurrence of `lx>>>NAME` anywhere else in the same file is
then replaced with the macro's body, and the declaration itself is stripped
from the compiled output:
```js
@lx:macros LOG_PREFIX {
    console.log('[MyWidget]', 
};

lx>>>LOG_PREFIX 'initialized');
lx>>>LOG_PREFIX 'destroyed');
```
Macro bodies may contain nested `{}` (matched as balanced pairs), so a macro
can hold a multi-statement block, not just a single expression.


## <a name="md">Markdown</a>

`lx.md('path/to/file')` is replaced at build time with a JSON-encoded string
containing the file rendered to HTML (using the preprocessor's own markdown
engine). The path is resolved relative to the directory of the file the
directive is written in, and `.md` is appended automatically if the path
doesn't already end with it:
```js
// Renders "./doc/widget-help.md" to HTML and inlines it as a JS string
const helpHtml = lx.md('doc/widget-help');
```
If the target file doesn't exist, the directive is replaced with an empty
string `""` instead of failing the build.


## <a name="lxml">Markup language (LXML)</a>

`` lx.ml(`...`) `` compiles a block of [LXML](https://github.com/epicoon/lxgo/tree/master/jspp/doc/lxml.md)
— a compact markup language for declaring a tree of widgets — into the
equivalent `new lx.Widget({...})` calls. See [LXML](https://github.com/epicoon/lxgo/tree/master/jspp/doc/lxml.md)
for the full syntax; not repeated here.


## <a name="i18n">Internationalization</a>

`lx.i18n('key')` is replaced with the translated string for the key in the
language the current build/request is for, looked up from the translation
data the app or a module registered (a module's own `i18n.yaml`, see
[Modules](https://github.com/epicoon/lxgo/tree/master/jspp/doc/modules.md)).
If no translation is found for that key, the directive is simply replaced
with the key itself as a plain string — so an untranslated string degrades
to its key rather than breaking the build.

A second argument passes named placeholders to substitute into the
translated string (which should contain `` `${name}` `` -style template
placeholders):
```js
lx.i18n('greeting', {name: user.name});
// if the translation for 'greeting' is `Hello, ${name}!`, compiles to
// something equivalent to `Hello, ${user.name}!`
```
Two translation lookups can be layered on top of each other for the same
build: module-level translations (keys are automatically namespaced as
`module-<ModuleName>-<key>` so different modules' short keys don't collide)
and the application's own top-level translations.


## <a name="config">Forwarding a backend config parameter</a>

`@config(Params.Param)`, used inside the *JS-application configuration
file* (`Components.JSPreprocessor.AppConfig`, not inside `.js` module code —
see [Start using](https://github.com/epicoon/lxgo/tree/master/jspp/doc/start.md)),
forwards a value from the backend `config.yaml` to the frontend. If you have:
```yaml
Params:
  Param: 1
```
you can reference it from the JS-app config:
```yaml
params:
  paramFromBackendOnFrontend: '@config(Params.Param)'
```
