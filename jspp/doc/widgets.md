# Widgets

A **widget** is one of the two things you can build on top of an [element](https://github.com/epicoon/lxgo/tree/master/jspp/doc/elements.md) (the other being a [plugin](https://github.com/epicoon/lxgo/tree/master/jspp/doc/plugins.md)) — a ready-to-use UI control: a button, an input, a table, a popup, etc. `lx.Rect`/`lx.Box`/`lx.TextBox` are the built-in widgets everything else extends; the rest ship as separate [modules](https://github.com/epicoon/lxgo/tree/master/jspp/doc/modules.md) under `js/modules/widgets/*`.

## Using a widget

Built-in widgets:
* `lx.Rect` — the base concrete element: a positioned/sized DOM node with css, border, events, drag-move, etc.
* `lx.Box` — a `Rect` that can contain other elements (children, layout/positioning).
* `lx.TextBox` — a `Rect` specialized for a single text node (`width`/`height` auto by default).

The rest of widgets are [modules](https://github.com/epicoon/lxgo/tree/master/jspp/doc/modules.md) and require explicit including via `lx.import(...)`, see [Preprocessor features](https://github.com/epicoon/lxgo/tree/master/jspp/doc/pp.md#require)

```js
let box = new lx.Box({
    geom: [30, 30, 40, 40],  // [left, top, width, height]
    text: 'Hello world!',
    css: 'css-green'
});

lx.import(lx.Button);
let button = new lx.Button({
    geom: [30, 50, 10, 10],
    text: 'Push',
    click: ()=>{ alert('Hi!') }
});
```

Most widgets from `js/modules/widgets` extend `lx.Box` or `lx.Rect`, so everything below is also available on them.

Common constructor config keys (all optional):
* `tag` — HTML tag to render (`div` by default)
* `key` — element key, used to look it up from its parent (`parent.get(key)`)
* `parent` — parent `lx.Box` to attach to right away
* `before` / `after` — sibling `lx.Rect` to insert next to
* `geom` — `[left, top, width, height, right, bottom]` tuple (any of them can be `null`)
* `coords` — `[left, top]`, `size` — `[width, height]`
* `left` / `right` / `top` / `bottom` / `width` / `height` — individual geometry values
* `margin` — shorthand margin
* `html` — inner HTML (for `lx.Rect`), `text` (for `lx.TextBox`/`lx.Box`, escaped as text)
* `css` — one class name or an array of class names to apply
* `cssScope` — CSS-scope name (see [CSS](https://github.com/epicoon/lxgo/tree/master/jspp/doc/css.md))
* `border` — `false` or `{width, color, style, side}`
* `fill` — background color, `opacity` — number, `picture` — background image
* `style` — a dict of raw CSS properties
* `click` / `blur` — event handlers
* `move` — `false` or `{parentMove, parentResize, xMove, yMove, xLimit, yLimit, moveStep, locked}` to make the element draggable
* `data` — an arbitrary dict of user data


## Geometry

```js
box.coords(10, 20);      // move to (10, 20)
box.size(100, 50);       // resize to 100x50
box.left(10).top(20).width(100).height(50);
box.setGeom([10, 20, 100, 50]);

let r = box.rect();           // {left, top, width, right, bottom, height} in px
let g = box.getGlobalRect();  // rect relative to the document
box.resize({width: 200});
```

`containPoint(x, y)`, `checkCross(otherElement)` and `isOutOfVisibility()` help with hit-testing and viewport checks.


## CSS and style

```js
box.addClass('css-pink');
box.removeClass('css-green');
box.toggleClass('css-a', 'css-b');
box.toggleClassOnCondition(box._state, 'css-pink', 'css-green');

box.style('cursor', 'pointer');
box.style({cursor: 'pointer', opacity: 0.9});

box.border({width: 1, color: '#000', style: 'solid', side: 'ltrb'});
box.roundCorners('50%');                       // all corners
box.roundCorners({side: 'tl br', value: 8});   // specific corners, px

box.opacity(0.5);
box.fill('#eee');
box.show();
box.hide();
box.disabled(true);
```


## Events

```js
box.click(() => { /* ... */ });
box.on('mouseover', handler);
box.off('mouseover', handler);

box.display(() => { /* fires once the element becomes visible */ });
box.displayIn(() => {});
box.displayOut(() => {});
```

The full list of built-in events an element can fire: `click`, `contextmenu`, `mousedown`, `mouseup`, `mousemove`, `mouseover`, `mouseout`, `transitionend`, `beforeHide`, `resize`, `change`, `moveBegin`, `beforeMove`, `move`, `moveEnd`, `resizeBegin`, `resizeEnd`, `display`, `displayin`, `displayout`, `beforeDestruct`, `afterDestruct`.


## `lx.Box`: containers

`lx.Box` is an element that can hold children:

```js
let box = new lx.Box({geom: [0, 0, 300, 200]});
let child = box.add(lx.Rect, {css: 'css-green'});

box.eachChild(child => { /* ... */ });
box.clear();              // remove all children
box.remove(child);        // remove a specific child
```

### Keys and lookup

A child's `key` (from its constructor config) doesn't have to be unique. Several
children can share the same `key` — each of them then gets an auto-computed
`index` (0, 1, 2, ...) among its same-keyed siblings, and lookups by that key
return all of them together:

```js
box.get('item');       // one child ('item' unique), an array of children ('item' repeated), or null
box.getAll('item');    // always an lx.Collection (empty if none, one item, or many)
box.getOne('item');    // always a single child or null (the first one, if several share the key)

box.find('item');      // like get(), but searches the whole subtree, not just direct children
box.findAll('item');   // like getAll(), searched recursively
box.findOne('item');   // like getOne(), searched recursively
```

`get`/`find` return whatever is "natural" for what's found (`null`, a single
widget, or a plain array/`lx.Collection` if several children share the key) —
use `getAll`/`findAll` or `getOne`/`findOne` when you need a guaranteed,
uniform return type regardless of how many children actually match.

```js
box.remove('item');   // removes every direct child keyed 'item'
```
`remove(key)` (and its "also destroy" counterpart `del(key)`) removes **all**
same-keyed children at once when called with just a key; pass `(key, index)`
or `(key, index, count)` to remove only a slice of them.

### Layout

Children can be laid out with a positioning strategy — a pluggable object that
controls how the box arranges its children (flex alignment, streams, grids,
free positioning, equal-sized slots):

```js
box.align(lx.CENTER, lx.MIDDLE);   // align all children inside the box
box.stream({direction: lx.HORIZONTAL});  // lay children out in a row/column
box.grid({cols: 3});               // arrange children in a grid, 12 columns by default
```

See [Positioning strategies](https://github.com/epicoon/lxgo/tree/master/jspp/doc/positioning-strategies.md) for the full set (`align`/`stream`/`grid`/`map`/`slot`) and their options.

## Anatomy of a widget module

A typical widget [module](https://github.com/epicoon/lxgo/tree/master/jspp/doc/modules.md) file (for example `js/modules/widgets/Checkbox.js`) follows this shape:
```js
// @lx:module lx.Checkbox;

lx.import(lx.BasicCssContext);

/**
 * @widget lx.Checkbox
 * @content-disallowed
 *
 * CSS classes:
 * - lx-Checkbox-1
 * - lx-Checkbox-0
 *
 * Events:
 * - change
 */
// @lx:namespace lx;
class Checkbox extends lx.Box {
    static initCss(css) {
        // define the widget's own CSS classes, usually via lx.BasicCssContext
    }

    /**
     * @widget-init
     *
     * @param [config] {Object: {
     *     #merge(lx.Rect::constructor::config),
     *     [value = false] {Boolean}
     * }}
     */
    render(config) {
        // build the widget's internal structure (server + client)
    }

    // @lx:<context CLIENT:
    clientRender(config) {
        // browser-only wiring: DOM events, interactivity
    }
    // @lx:context>

    value(val) {
        // getter/setter-style public API
    }
}
```

Conventions worth knowing when reading or writing a widget:
- `@widget lx.Name` in the header jsdoc block documents the widget for tooling, lists the CSS classes the widget defines and, where relevant, the custom events it triggers via `.trigger('eventName', ...)`.
- `@content-disallowed` means the widget manages its own children and does not accept arbitrary nested content the way a plain `lx.Box` does.
- `static initCss(css)` is where the widget's CSS classes are declared (see [CSS](https://github.com/epicoon/lxgo/tree/master/jspp/doc/css.md)); most widgets extend `lx.BasicCssContext` to reuse the active preset's colors.
- `render(config)` runs on both server and client and builds the widget's structure; a `@lx:<context CLIENT: ... @lx:context>` block (often a `clientRender(config)` override) adds browser-only behavior such as DOM event listeners — see [Preprocessor features](https://github.com/epicoon/lxgo/tree/master/jspp/doc/pp.md) for other preprocessor directives.
- Public API is plain getter/setter-style methods (`value()`, `disabled()`, ...) plus DOM-style events (`widget.on('change', handler)`); a widget signals a state change to its consumer by calling `this.trigger('eventName', this.newEvent({...}))`.
- Many widgets are composed from other widgets (e.g. `lx.RadioGroup` is a `lx.LabeledGroup` of `lx.Radio`s, `lx.Textarea` extends `lx.Input`, `lx.Radio` extends `lx.Checkbox`) — reading a widget's `lx.import(...)` calls shows what it is built from.

## Catalog

Each widget is documented in its own source file (`js/modules/widgets/<Name>.js` or `js/modules/widgets/<Name>/<Name>.js`) — the table below is a map to find the right file, not a substitute for reading it.

| Widget | Extends | Notable events | Purpose |
|---|---|---|---|
| `lx.Button` | `lx.Box` | — | Clickable button with a hover/press hint. |
| `lx.Checkbox` | `lx.Box` | `change` | Single checkbox, boolean `value()`. |
| `lx.CheckboxGroup` | `lx.LabeledGroup` | — | Group of labeled checkboxes. |
| `lx.Radio` | `lx.Checkbox` | — | Single radio button (checkbox-like, grouped by `lx.RadioGroup`). |
| `lx.RadioGroup` | `lx.LabeledGroup` | `change` | Group of `lx.Radio` widgets with a single selected value. |
| `lx.LabeledGroup` | `lx.Box` | — | Base for widgets composed of `{control, label}` pairs (used by `RadioGroup`/`CheckboxGroup`). |
| `lx.Input` | `lx.Rect` | — | Text `<input>` wrapper: `value()`, `placeholder()`, `focus()`/`blur()`. |
| `lx.Textarea` | `lx.Input` | — | Multiline text input. |
| `lx.Switch` | `lx.Box` | — | On/off toggle switch. |
| `lx.Slider` | `lx.Box` | `started`, `moved`, `stopped`, `change` | Draggable-handle numeric slider. |
| `lx.Scroll` | `lx.Box` | — | Custom scrollbar widget. |
| `lx.Dropbox` | `lx.Box` | `change`, `opened`, `closed` | Dropdown selector built on `lx.Input` + `lx.Table`. |
| `lx.LanguageSwitcher` | `lx.Dropbox` | — | Dropbox pre-wired to switch `lx.Language` and persist the choice in a cookie. |
| `lx.Table` | `lx.Box` | — | Tabular grid of rows/cells. |
| `lx.TreeBox` | `lx.Box` | `leafOpening`, `leafOpened`, `leafClosed`, `beforeAddLeaf`, `afterAddLeaf`, `beforeDropLeaf`, `afterDropLeaf` | Expandable tree view. |
| `lx.Paginator` | `lx.Box` | `change` | Page navigation control (first/prev/next/last). |
| `lx.ModelsGrid` | `lx.Box` | `applyFilters` | Grid bound to a [model](https://github.com/epicoon/lxgo/tree/master/jspp/doc/models.md) collection — sortable/resizable/reorderable columns, per-column filters, built-in `lx.Paginator`. |
| `lx.Form` | `lx.Box` | — | Helper for building a set of labeled fields from a `{name: widgetClassOrConfig}` map. |
| `lx.Calculator` | `lx.Box` | — | Numeric keypad + `lx.Input`/`lx.Button` calculator. |
| `lx.Calendar` | `lx.Input` | — | Date-picking input with a popup calendar table. |
| `lx.ColorPicker` | `lx.Box` | — | Color picker (HSV wheel + `lx.Input`/`lx.Button`/`lx.Scroll`). |
| `lx.ConfirmPopup` | `lx.Box` | — | Modal confirm ("yes/no") dialog, opened via a static call rather than instantiated inline. |
| `lx.InputPopup` | `lx.Box` | — | Modal single-value input dialog, same static-call pattern as `lx.ConfirmPopup`. |
| `lx.ActiveBox` | `lx.Box` | — | Draggable/resizable window-like box with a header and a close button. |
| `lx.EggMenu` | `lx.Box` | — | Circular popout menu ("egg" of buttons expanding from a point). |
| `lx.Marks` | `lx.Box` | — | Row of closable "mark"/tag chips, used by `lx.MultiBox`. |
| `lx.MultiBox` | `lx.Box` | `selected`, `unselected`, `sheetOpened`, `sheetClosed`, `selectionChange`, `markAppended`, `beforeDropMark`, `markDropped` | Multi-sheet box with a `lx.Marks` tab strip and drag-and-drop between sheets. |
| `lx.JointMover` | `lx.Rect` | — | Thin draggable divider/handle (used to build resizable layouts, e.g. by `lx.MultiBox`). |
| `lx.MatrixSwapper` | `lx.Box` | `swapped` | Grid of cells whose contents (`lx.Box` items) can be swapped by drag-and-drop. |
| `lx.BoxSlider` | `lx.Box` | — | Base slideshow container (left/right buttons switching child boxes), extended by `lx.ImageSlider`. |
| `lx.ImageSlider` | `lx.BoxSlider` | — | Image slideshow. |
| `lx.Image` | `lx.Rect` | `load`, `scale` | `<img>` wrapper. |
| `lx.Html` | `lx.Rect` | — | Raw HTML content wrapper (takes a plain string as config). |
