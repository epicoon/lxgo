------------------------------------------------------------------------------------------------------------------------
Date: 2026.07.28
Version: v0.1.0-alpha.31
Changes:
- fix: `ElemHandler`/`ServiceHandler`/`PluginHandler` called the nil preprocessor's own `LogError` instead of the
  resource's when `"jspp"` was missing from the request context, panicking on exactly the error path meant to
  report that
- fix: `PluginManager.Save()` never populated its in-memory cache from the saved data - `Has()`/`Get()` in the same
  process saw empty entries right after `Save()`, even though the on-disk file was written correctly
- fix: `ServiceHandler`'s `except` list (`get-modules`'s `have` param) came out twice its intended length, padded
  with empty strings ahead of the real names
- fix: `pathfinder.GetAbsPath("")` panicked (indexed into an empty string)
- fix: an LXML block link (`<&Name>`) to a never-defined block (`<*Name>`) silently compiled to a literal, invalid
  `[|Name|]` placeholder instead of raising a compile error; `lxmlParser.AddError` itself could also panic when
  called from the compile phase (out-of-range line lookup) - both fixed
- fix: `checkPluginPath` panicked on a plugin's `client.file` config field of the wrong YAML type - now falls back
  to `Plugin.js`, like its `key` sibling already did
- fix: a widget's repeated `#method()` call in LXML overwrote the previous call's args instead of keeping each
  call's own - `WidgetNode.Methods` is now one entry per call, not one per method name
- fix: `lx.import`ing a bare module name from inside a GuiNode (a file itself pulled in by path) could silently
  drop the import whenever more than one such file was involved - the asset-discovery compile pass overwrote its
  accumulated module list on each nested file instead of merging into it
- fix: a widget with only a `{data}` attribute and no other config lost its data entirely - the compiler's
  early-out for "nothing to configure" didn't check for it
- fix: `lx.alert()` called a DOM-selector method that never existed anywhere in the codebase, throwing on first use
  - rewritten to the same find-or-create pattern already used by `Toast.js`
- remove: dead code (`extractModuleNames`, `nodeStack.Pop()`)
- test: unit and integration tests across the package (directive/macro/import compiler, LXML parser round-trips,
  i18n, executor, HTTP handlers, plugin manager/config)

------------------------------------------------------------------------------------------------------------------------
Date: 2026.07.27
Version: v0.1.0-alpha.30
Changes:
- internal: adapted to `lxgo-kernel`'s `Config`→`IDict`/`Dict` refactor and its `conv`→`cast` package rename
  (`plugins.Config`, `internal/utils.targetBuilder`, `plugins.pluginRenderer`/`snippetRenderer`) - no change to
  `jspp`'s own public API

------------------------------------------------------------------------------------------------------------------------
Date: 2026.07.25
Version: v0.1.0-alpha.29
Changes:
- rename: `IExecutor`/`IExecutorBuilder`/`IExecResult` → `IJSExecutor`/`IJSExecutorBuilder`/`IJSExecResult`;
  `IPreprocessor.ExecutorBuilder()` → `JSExecutorBuilder()` - disambiguates from other "executor" concepts in the package
- rename: `IAsset.Path()` → `IAsset.Src()`
- fix: `NewJSPreprocesor` → `NewJSPreprocessor` (typo in the exported constructor name)
- docs: Go-doc comments for the module's public API (everything outside `internal/`) - root `conventions.go`, plus
  the `cmd`/`component`/`elems`/`plugins` subpackages

------------------------------------------------------------------------------------------------------------------------
Date: 2026.07.24
Version: v0.1.0-alpha.28
Changes:
- add: `lx.import(...)` call syntax replaces the `@lx:require`/`@lx:use` comment directives - one unified call that mixes file/directory requires and module names, with `-R`/`-F`/`-U` flags as arguments
- add: `lx.ml(\`...\`)` call syntax replaces the `@lx:<ml ... ml>` comment-block form for inline LXML templates - supports an escaped backtick (`` \` ``) inside the template and absorbs an immediately preceding `const`/`let`/`var` assignment into the compiled output
- add: `lx.md(...)` directive syntax (was `@lx:md(...)`) for embedding rendered markdown
- add: generic `jspp.IElement`/`elems.Element` - a DI-registered object exposing ajax handlers, dispatched through a new `/lx/elem` endpoint; `IPlugin`/`Plugin` now build on top of it instead of duplicating `Init`/`App`/`Preprocessor`
- add: plugin rendering cache (`off`/`on`/`dev`/`inherit` modes)
- add: `lx.ModelTypeEnum.PK` constant
- rename: `ModelCollectionGrid` widget rewritten and renamed to `ModelsGrid`
- rename: `Tost.js` → `Toast.js` (typo)
- refactor: generated code from `ServiceHandler`'s `get-modules` action and from plugin rendering now uses `lx.import(...)` instead of `@lx:use`/`@lx:require`, matching the new syntax
- docs: `pp.md`/`lxml.md`/`widgets.md`/`plugins.md`/`css.md`/`components.md`/`models.md` rewritten for the new `lx.import`/`lx.ml`/`lx.md` syntax; new `modules.md` and `positioning-strategies.md` guides added

------------------------------------------------------------------------------------------------------------------------
Date: 2026.07.16
Version: v0.1.0-alpha.27
Changes:
- add: `@lx:md('path.md')` directive — renders a markdown file to HTML via a built-in converter (own markdown engine, not a third-party library)
- add: `@lx:macros NAME { ... }` directive and `lx>>>NAME` expansion syntax
- add: JSPreprocessor config `ModuleInjector` — substitute JS-module names when resolving `@lx:use`
- add: dev-mode source markers around compiled JS files/fragments (`Mode: DEV`)
- add: `lx.MdHighlighter` now actually highlights typed markdown code blocks (js/go) instead of being a no-op stub
- fix: markdown code blocks no longer leak unescaped HTML and are no longer corrupted by inline-formatting rules (links/bold/etc.) meant for surrounding text
- fix: a blank line inside a fenced markdown code block no longer prematurely closes the block
- fix: markdown default text color (code blocks, blockquotes) now follows the active CSS preset instead of relying on inheritance
- rename: markdown CSS classes from bare `md-*` to `lx-md-*`, matching the rest of the framework's naming convention
- refactor: removed dead/never-finished compiler extension-hook code and other leftover artifacts

------------------------------------------------------------------------------------------------------------------------
Date: 2026.07.13
Version: v0.1.0-alpha.26
Changes:
- add: JSPreprocessor config `SysPath` — directory for system needs
- fix: failed JS-code execution is now dumped to `{SysPath}/js_fails` and logged instead of writing to a hardcoded dev-machine path and ignoring the write error
- fix: plugin cache (Save/Load) now propagates file/serialization errors instead of silently ignoring them; a corrupted cache file no longer causes a panic
- fix: unsafe type assertions in JS-executor response parsing replaced with checked ones (panic -> error)
- fix: maps builder no longer panics on `go list` failure and no longer continues into a directory it failed to clear; errors are propagated
- fix: format string mismatch in JS-application config compiler error log
- fix: `/lx/service` handler no longer reports `"success": true` when module compilation fails
- fix: target builder (JS-bundle writer) now propagates write errors instead of silently continuing

------------------------------------------------------------------------------------------------------------------------
Date: 2026.07.12
Version: v0.1.0-alpha.25
Changes:
- add: plugin cache
- add widget: lx.Switch
- optimization: js-models binding
- fix: lx.Box.getChildren() now is scoped in a plugin
- fix: lx.Checkbox.click()
- add: lx.Collection.swap(i, j)
- refactor and rename: lx.StreamItemRelocator -> lx.MatrixSwapper
- internal fixes


------------------------------------------------------------------------------------------------------------------------
Date: 2026.03.19
Version: v0.1.0-alpha.24
Changes:
- refactor: snippet render error processing
- rename[js]: lx.ModelTypeEnum.INTEGER -> lx.ModelTypeEnum.NUMBER
- fix[js]: readonly lx.Checkbox
- fix: command preview flag `:build-maps -p`
- add: forwarding an backend-application configuration parameter to frontend-application
- add: component config param `ModulesIgnore []string`
- add: Plugin config param `server.file`
- add: JS-module lx.HashRouter
- fix: i18n with params
- add: lx.ModelCollection.createByData(list, byFirst = true)
- refactor: JS client-components
- add: lxml directive `call`

------------------------------------------------------------------------------------------------------------------------
Date: 2025.12.03
Version: v0.1.0-alpha.23
Changes:
- add: lx.app.cssManager.updatePreset()

------------------------------------------------------------------------------------------------------------------------
Date: 2025.12.02
Version: v0.1.0-alpha.22
Changes:
- fix: lx.CssContext

------------------------------------------------------------------------------------------------------------------------
Date: 2025.12.01
Version: v0.1.0-alpha.21
Changes:
- add: lx.CssContext @media support
- add: lx.Preset.injectElementsCss()
- add: lx.app params from server
- add: JS-application local config
- refactor: plugin require via config now without inline -U flag

------------------------------------------------------------------------------------------------------------------------
Date: 2025.11.12
Version: v0.1.0-alpha.20
Changes:
- refactor: toast messages can be removed by click in any cases
- fix: lx.Rect click event for touchscreen

------------------------------------------------------------------------------------------------------------------------
Date: 2025.11.05
Version: v0.1.0-alpha.19
Changes:
- new positioning strategy gridFit

------------------------------------------------------------------------------------------------------------------------
Date: 2025.11.05
Version: v0.1.0-alpha.18
Changes:
- fix for lxml html content

------------------------------------------------------------------------------------------------------------------------
Date: 2025.10.28
Version: v0.1.0-alpha.17
Changes:
- bugfix plugin path

------------------------------------------------------------------------------------------------------------------------
Date: 2025.09.24
Version: v0.1.0-alpha.16
Changes:
- refactor plugins map build

------------------------------------------------------------------------------------------------------------------------
Date: 2025.09.23
Version: v0.1.0-alpha.15
Changes:
- refactor JS-modules build
- add command ":build"

------------------------------------------------------------------------------------------------------------------------
Date: 2025.09.19
Version: v0.1.0-alpha.13
Changes:
- refactor lx.app.events

------------------------------------------------------------------------------------------------------------------------
Date: 2025.09.18
Version: v0.1.0-alpha.12
Changes:
- fix lx.Toast
- fix AlignPositioningStrategy
- refactor CssTag

------------------------------------------------------------------------------------------------------------------------
Date: 2025.09.18
Version: v0.1.0-alpha.10
Changes:
- fix lx.InputPopup title

------------------------------------------------------------------------------------------------------------------------
Date: 2025.09.09
Version: v0.1.0-alpha.9
Changes:
- added lxml comments

------------------------------------------------------------------------------------------------------------------------
Date: 2025.09.03
Version: v0.1.0-alpha.8
Changes:
- fix lxml parsing

------------------------------------------------------------------------------------------------------------------------
Date: 2025.09.02
Version: v0.1.0-alpha.7
Changes:
- fix lxml parsing with spaces

------------------------------------------------------------------------------------------------------------------------
Date: 2025.08.20
Version: v0.1.0-alpha.6
Changes:
- refactor Dialog.js

------------------------------------------------------------------------------------------------------------------------
Date: 2025.08.15
Version: v0.1.0-alpha.5
Changes:
- add feature plugin ajax-requests routing
- fix lxml html content for empty widget

------------------------------------------------------------------------------------------------------------------------
Date: 2025.08.14
Version: v0.1.0-alpha.4
Changes:
- fix plugin config with map for images
- changed syntax:
    - lx(i18n).key  ->  lx.i18n(key)
    - lx(STATIC).CONST  ->  lx.self(CONST)
- added feature for translations with params, example:
    in tr.yaml:
        key: text with ${param}
    in code:
        lx.i18n(key, {param: value})

------------------------------------------------------------------------------------------------------------------------
Date: 2025.08.13
Version: v0.1.0-alpha.3
Changes:
- fixed main plugin code internationalization

------------------------------------------------------------------------------------------------------------------------
Date: 2025.08.12
Version: v0.1.0-alpha.2
Changes:
- fixed rendering plugin as page without template
- added part of documentation
- fixes in JS-code

------------------------------------------------------------------------------------------------------------------------
Date: 2025.07.24
Version: v0.1.0-alpha.1
