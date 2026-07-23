# Positioning strategies

An `lx.Box` arranges its children through a **positioning strategy** — a
pluggable object (subclass of `lx.PositioningStrategy`) that takes over layout
whenever you call one of `align`/`stream*`/`grid*`/`map`/`slot` on the box.
Only one strategy is active on a box at a time; a box with none set keeps its
children at whatever explicit geometry (`geom`/`coords`/`left`/`top`/...) they
were given.

A strategy can be set either by calling the matching method, or by passing the
same name as a constructor config key:

```js
let box = new lx.Box({
    geom: [0, 0, 300, 200],
    grid: {cols: 3}   // same as calling box.grid({cols: 3}) after construction
});
```

Other useful methods on `lx.Box`, regardless of which strategy is active:
* `box.positioning()` — get the current strategy instance (a no-op base
  `lx.PositioningStrategy` if none was set).
* `box.stopPositioning()` / `box.startPositioning()` — temporarily suspend and
  resume automatic re-layout, useful while adding many children at once.
* `box.dropPositioning()` — remove the strategy entirely.
* `box.setIndents(config)` — update indents on the current strategy in place.

Most strategies accept a common set of indent options (merged into their own
config): `indent`, `step`, `stepX`, `stepY`, `padding`, `paddingX`, `paddingY`,
`paddingLeft`/`paddingRight`/`paddingTop`/`paddingBottom`.


## `align(horizontal, vertical)`

Flex-based alignment of all children inside the box, in one direction:

```js
box.align(lx.CENTER, lx.MIDDLE);
// or, for more options:
box.align({horizontal: lx.LEFT, vertical: lx.TOP, direction: lx.HORIZONTAL});
```

* `horizontal` — `lx.LEFT` / `lx.CENTER` / `lx.RIGHT` / `lx.JUSTIFY`.
* `vertical` — `lx.TOP` / `lx.MIDDLE` / `lx.BOTTOM`.
* `direction` — `lx.HORIZONTAL` (default) or `lx.VERTICAL`: which axis children are laid out along (the other axis is where `horizontal`/`vertical` alignment applies across the whole row/column).


## `stream(config)` / `streamProportional(config)`

Lays children out one after another in a row or column.

```js
box.stream({direction: lx.VERTICAL, height: '60px'});
box.streamProportional({direction: lx.HORIZONTAL});  // children share width via their own `width()`/`height()` as weights
```

* `direction` — `lx.HORIZONTAL` or `lx.VERTICAL` (default depends on context: opposite of the parent's own stream direction, if any).
* Simple mode (`stream()`, `type: TYPE_SIMPLE`) — every child gets the same row height / column width: `height`/`width` (default `40px`), `minHeight`/`minWidth`/`maxHeight`/`maxWidth`.
* Proportional mode (`streamProportional()`, `type: TYPE_PROPORTIONAL`) — implemented as a CSS grid with `fr` units; each child's own `height()`/`width()` value (a plain number) becomes its proportion weight.
* `css` — a class name added to every child placed by the strategy.


## `grid(config)` / `gridProportional` / `gridStream` / `gridAdaptive` / `gridFit`

All five are the same `GridPositioningStrategy`, differing only by `type`:

```js
box.grid({cols: 4});             // TYPE_SIMPLE — fixed number of equal columns
box.gridProportional({cols: 4}); // TYPE_PROPORTIONAL - the box will use all available height, children will share it proportionally
box.gridStream({cols: 4});       // TYPE_STREAM
box.gridAdaptive({minWidth: '120px'}); // TYPE_ADAPTIVE — as many columns as fit, empty space allowed
box.gridFit({minWidth: '120px'});      // TYPE_FIT — columns stretch to fill the row, no empty space
```

* `cols` — number of columns (default 12; ignored by `gridAdaptive`/`gridFit`, which derive column count from `minWidth`/available width instead).
* `minWidth`/`minHeight`/`maxWidth`/`maxHeight`, or `width`/`height` as shorthand for setting both min and max at once.
* `gridAdaptive`/`gridFit` use CSS `repeat(auto-fill, ...)` / `repeat(auto-fit, ...)` respectively under the hood — the difference is whether unfilled trailing space is left empty (`adaptive`) or the existing columns stretch to fill it (`fit`).


## `map(config)`

Free/absolute positioning: children keep whatever explicit geometry you give
them (`geom`/`coords`/`size` config, or `.left()`/`.top()`/... calls) — this is
what a box behaves like with no strategy at all, formalized so you can control
the default unit for geometry values that don't specify one:

```js
box.map('px');                 // or 'map(%)' / {format: '%'}
box.positioning().setFormat(lx.WIDTH, 'px');  // override the format for one param only
```


## `slot(config)`

A fixed number of equal-sized "slots" with a constant width/height ratio,
auto-sized to fill the box; adding more elements recalculates the existing
slots' sizes without resizing the box itself:

```js
box.slot({cols: 3, rows: 2, k: 1.5});  // 3x2 slots, width:height ratio 1.5
```

* `cols`, `rows`, `count` — any two of the three effectively determine the layout (given `cols` and either `rows` or `count`, the other is derived).
* `k` — width/height ratio of a single slot (default `1`).
* `align` — same `lx.LEFT`/`lx.CENTER`/`lx.RIGHT`/`lx.TOP`/`lx.MIDDLE`/`lx.BOTTOM` enum as `align()`, controls how slots are distributed along an axis (centered with even steps, pressed to the edges, or evenly spread).
* `type` — widget class instantiated for each slot (`lx.Box` by default).
