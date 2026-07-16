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
- refactor: tost messages can be removed by click in any cases
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
- fix lx.Tost
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
