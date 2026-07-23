# CSS

`lxgo-jspp` lets you write CSS in JavaScript instead of separate `.css` files. There are two layers:

* **`lx.CssContext`** — a low-level, standalone builder for a set of CSS classes/styles, rendered to a plain CSS string.
* **`lx.CssManager`** (component `cssManager`, see [Application components](https://github.com/epicoon/lxgo/tree/master/jspp/doc/components.md)) — a higher-level system built on top of `CssContext` that groups styles into named **scopes**, ties them to a **preset** (a set of theme tokens like colors), and knows how to inject the resulting CSS into the page (or build it on the server, see [CssScopeRenderSide](#csrs)).

Widgets use the second layer automatically (each widget class can define its own `initCss(css)`, see [Widgets](https://github.com/epicoon/lxgo/tree/master/jspp/doc/widgets.md)); `CssContext`/`CssTag` are what you reach for directly when you just want to generate some CSS from JS.


## <a name="context">Low-level: `lx.CssContext` and `lx.CssTag`</a>

```js
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
cssTag.addCss(css.toString());
cssTag.commit();
```

`CssContext` methods:
* `addClass(name, content, specification)` — define a `.name { ... }` rule. `content` is a plain object of CSS properties in camelCase (`backgroundColor`, `fontSize`, ...); it's converted to kebab-case automatically.
* `inheritClass(name, parent, content, specification)` — like `addClass`, but merges in another class's `content`/`specification` first.
* `addAbstractClass(name, ...)` / `inheritAbstractClass(name, parent, ...)` — same, but the class is not meant to be rendered on its own, only used as a `parent` for `inheritClass`/`inheritAbstractClass`.
* `specification` (third argument to the `*Class` methods above) describes extra selectors derived from the class, e.g. pseudo-classes (`hover`, `disabled`) or `@media` blocks:
  ```js
  css.addClass('css-btn', {backgroundColor: 'white'}, {
      hover: {backgroundColor: 'lightgray'},
      '@media': {'(max-width: 600px)': {display: 'none'}}
  });
  ```
* `addStyle(name, content)` — a raw CSS rule/selector that isn't a single class (e.g. `.a .b`, or an `@keyframes ...` directive when `name` starts with `@`).
* `addStyleGroup(name, list)` — shortcut to call `addStyle` for a group of related selectors sharing a prefix (`list` keys are appended to `name`).
* `registerMixin(name, callback)` — register a helper usable inside `content` as `'@name': args`; the callback returns a partial `content`/`specification` object to merge in (see [presets](#preset) for a built-in example: `@img`, `@ellipsis`, `@icon`).
* `usePreset(preset)` / `setPrefix(prefix)` — attach an `lx.CssPreset` instance (see below) and/or a class-name prefix (used by scopes to avoid collisions between widgets).
* `useExtender(extenderClass)` — pull in another context's classes/mixins/styles, see [`CssContextExtender`](#extender).
* `merge(context)` — merge another `CssContext`'s content into this one.
* `hasClass(name)` / `getClass(name)` / `getClassNames()` — inspect what's defined (including through `useExtender`-linked contexts).
* `toString()` — render everything added so far into a single CSS string, in the order it was added.

`CssTag` wraps a `<style>` element in the page:
* `new lx.CssTag({id, before, after})` — if an element with `id` already exists, reuses it (and keeps its current content in `_code`); `before`/`after` are CSS selectors used to position a newly created `<style>` tag inside `<head>`.
* `setCss(code)` / `addCss(code)` — replace or append to the pending CSS text.
* `commit()` — write the pending text into the `<style>` element's `innerHTML`.
* `CssTag.exists(id)` — static check whether a `<style id="...">` is already on the page.


## <a name="preset">Presets</a>

A preset is a named set of theme tokens (colors, sizes, etc.) that CSS definitions can reference instead of hardcoding values, so switching a preset re-themes everything that uses it.

```js
class CssPresetWhite extends lx.CssPreset {
    static getName() { return 'white'; }

    getParams() {
        return {
            mainBackgroundColor: 'white',
            textColor: 'black',
            widgetBorderColor: '#D9D9D9',
            // ...
        };
    }
}
```

Two built-in presets ship with the package: `lx.CssPresetWhite` and `lx.CssPresetDark` (`js/modules/cssPresets/`), each defining the same set of tokens (background/text/widget colors, a "checked/cold/hot/neutral" color scale, shadow size, border radius, etc.) for a light and a dark theme respectively.

A preset only takes effect once it's attached to a `CssContext` that's actually
in use — for a whole application, that means assigning it to a
[`CssManager` scope](#manager), either through the app's config
(`cssManager: {default: lx.CssPresetDark}`, see [Start using](https://github.com/epicoon/lxgo/tree/master/jspp/doc/start.md))
or in code (`lx.app.cssManager.createPresetScope(lx.CssPresetDark)`). Every
widget's own CSS is then generated against that scope's preset automatically
— you don't attach a preset per widget yourself.

Every token declared in `getParams()` becomes a property on the preset instance, and reading it (`css.preset.textColor`) returns an `lx.CssValue` wrapper rather than a raw string. Inside an element (see [Elements](https://github.com/epicoon/lxgo/tree/master/jspp/doc/elements.md)), in particular a widget's `initCss(css)` (see [Widgets](https://github.com/epicoon/lxgo/tree/master/jspp/doc/widgets.md)), reference tokens directly through `css.preset.*`:

```js
static initCss(css) {
    css.addClass('my-widget', {
        backgroundColor: css.preset.widgetBackgroundColor,
        color: css.preset.textColor
    });
}
```

`CssContext.presetValue(name, defaultVal, modifier)` (or the static `CssContextExtender.presetValue(...)`, see below) builds a value that falls back to `defaultVal` when no preset (or no such token) is set, optionally passing the resolved value through `modifier`.

### <a name="extender">`lx.CssContextExtender`</a>

A reusable, cacheable bundle of mixins/classes that isn't tied to one particular widget — a static class overriding `init(css)`:

```js
class BasicCssContext extends lx.CssContextExtender {
    static init(css) {
        css.registerMixin('ellipsis', () => ({
            overflow: 'hidden',
            whiteSpace: 'nowrap',
            textOverflow: 'ellipsis'
        }));
    }
}
```

`CssContextExtender.getContext()` builds the underlying `CssContext` once (lazily, cached as `this.context`) and hands it to `CssContext.useExtender(...)`. The built-in `BasicCssContext` registers a handful of general-purpose mixins this way: `@img` (background image, cover/no-repeat), `@ellipsis` (text overflow ellipsis), `@icon` (centered icon font glyph).


## <a name="manager">`lx.CssManager` component and scopes</a>

`lx.CssManager` is an application component (key `cssManager`, available `client`/`server`) — see [Application components](https://github.com/epicoon/lxgo/tree/master/jspp/doc/components.md) for how to configure it and its full method reference.

What matters for this page: `CssManager` manages one or more named **scopes** — each scope is an `lx.CssScope`, pairing a `CssContext` with a preset and (for named scopes) a class-name prefix, so several themed variants can be active on the same page without their class names colliding (a non-default scope's classes get rendered with `<name>-` prepended, e.g. `.my-widget` becomes `.white-my-widget`). Widgets don't call `CssManager` directly — a widget's own CSS is generated lazily, once per scope, the first time an instance of that widget class is added to that scope, by calling the widget's `static initCss(css)`.


## <a name="csrs">Client vs server CSS rendering (`CssScopeRenderSide`)</a>

By default (`Components.JSPreprocessor.CssScopeRenderSide: client`, see [Start using](https://github.com/epicoon/lxgo/tree/master/jspp/doc/start.md)) all of the above runs in the browser: as widgets are created, their CSS is generated and written into `<style>` tags on the fly.

Setting `CssScopeRenderSide: server` moves this work to build time instead: for an `app`-type build target, the preprocessor additionally runs a small server-side script (through its own JS executor) that builds the application's modules and calls `lx.app.cssManager.pack()`, then prepends the resulting CSS (as plain data — per scope: prefix, preset name, involved element classes, class names, css text) to the built JS bundle. On page load, the client-side `CssManager.onReady()` detects this (`lx.app.getSetting('csrs') === 'server'`) and calls `CssScope.unpack(...)` to recreate the scopes and commit the already-built CSS text directly, instead of rebuilding it from scratch in the browser.
