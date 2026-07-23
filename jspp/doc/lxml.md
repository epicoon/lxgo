# LXML

LXML is a compact, indentation-based markup language for describing a tree of
`lx` widgets declaratively, instead of writing `new lx.Widget({...})` calls by
hand. It is parsed and compiled by `JSPreprocessor` at build time into plain
JS — at runtime there is no LXML, only the same widget-construction code you
would otherwise write yourself (see
[Elements](https://github.com/epicoon/lxgo/tree/master/jspp/doc/elements.md)
and
[Widgets](https://github.com/epicoon/lxgo/tree/master/jspp/doc/widgets.md)
for what the constructed objects are).

There is a [VSCode extension](https://github.com/epicoon/lxgo-jspp-vscode-ext)
that syntax-highlights LXML blocks.


## Embedding LXML in a JS file

An LXML block is written as `` lx.ml(`...`) `` — a JS template literal (so it
is naturally multi-line, like any other backtick string):

```js
lx.ml(`
<lx.Box> @root [0:0:300:200]
  <lx.Button> @btn (text:'Click me')\
    #click(()=>{ alert('Clicked!'); })
`);
```

The preprocessor replaces the whole `lx.ml(...)` call with the compiled JS
before the file is otherwise processed. A literal backtick can appear inside
the block (for example in a raw `(...)` attribute holding real JS, like an
inline arrow function that itself uses a template literal) if escaped as
`` \` ``.

If you need to keep a reference to the created widgets after the block,
assign the call to a variable — the compiled code then keeps whichever
keyword (`const`/`let`/`var`) and name you used, building an object from
every top-level `@key`/`[f:field]` in the tree, so you can do `root.form`,
`root.btn`, etc. afterwards:

```js
const root = lx.ml(`
<lx.Box> @form
  <lx.Button> @btn (text:'Click me')
`);

root.btn.click(() => alert('Clicked!'));
```


## Syntax

Every line is one of: a **widget**, a **control-flow** statement, a
**block** definition/call, or (when nested under a widget) raw **HTML**.
Nesting is expressed purely by indentation (a consistent number of spaces or
tabs per level, the first indent used in the file sets the unit) — there are
no closing tags.

A logical line can be split across several source lines by ending each
physical line except the last with `\` — the parser joins them (with a
single space) before parsing, so a widget's attributes/method calls can be
spread out for readability:

```
<lx.Button> @btn.primary (text:'Click me')\
  #click(()=>{ alert('Clicked!'); })
```

### Widgets

A widget line starts with `<` followed by the widget's full name — one of
the built-ins `lx.Rect`/`lx.Box`, or any module tagged with a `@widget
lx.Name` doc comment (see
[Widgets](https://github.com/epicoon/lxgo/tree/master/jspp/doc/widgets.md)).
The closing `>` and the space before the first attribute are the recommended
style:

```
<lx.Box> @key (e:1)
```

(a compact form without the `>`/space, `<lx.Box@key(e:1)`, is also accepted
by the parser, but the spaced form above is preferred.)

Attributes can appear in any order, each recognized by its leading character:

| Syntax | Meaning |
|---|---|
| `@key` | registers the widget under `key` (usable from the block's output object, see above) |
| `.cssClass` | adds a CSS class; repeatable |
| `'text'` / `"text"` | quoted static text content (can span multiple physical lines) |
| `[f:name]` / `[field:name]` | binds the widget to model field `name` |
| `[m:name]` / `[matrix:name]` | binds to matrix field `name`; the widget must then have exactly one nested widget line, used as the item template for every matrix entry |
| `[_]` | marks the widget as volumetric (auto-sized from content) |
| `[l:t:w:h]` | sets geometry — see [Geometry](#geometry) below |
| `(...)` | raw JS object entries, merged as-is into the widget's config object |
| `{...}` | raw JS object entries used as the widget's `data` |
| `#method` / `#method(args)` | calls `.method(args)` on the created widget after construction; repeatable. If `args` looks like `key:value` pairs, it is wrapped in `{}` automatically |

Example:

```
<lx.Box> @root [0:0:300:200]
  <lx.Button> @btn.primary (text:'Click me')\
    #click(()=>{ alert('Clicked!'); })
```

compiles to (simplified):

```js
var _w0=new lx.Box({key:"root",geom:[0,0,300,200]});
_w0.begin();
var _w1=new lx.Button({key:"btn",css:["primary"],text:'Click me'});
_w1.click(()=>{ alert('Clicked!'); });
_w0.end();
```

Lines nested under a widget that aren't recognized as a widget, control-flow
or block line are treated as raw HTML and appended to the widget's inner
markup verbatim — a widget can have nested widgets *or* raw HTML, not both.

**JS variables from the surrounding code** can be used almost anywhere: as
values inside `(...)` and `{...}` (they're copied through as plain JS, no
special handling needed), and inside quoted text via `${...}` interpolation —
`"hello ${name}!"` compiles to `"hello "+name+"!"`. The one exception is
geometry (`[...]`, see below), which only accepts number/string/`null`
literals; to set a geometry value from a variable, call the corresponding
method after construction instead — `#left(v)`, `#width(v)`, etc.

The parser extracts `'...'`/`"..."` chunks before splitting the source into
lines, so a text attribute can freely span several physical lines. Outside a
`<pre>...</pre>` wrapper, any run of whitespace in the text (including line
breaks) collapses to a single space when compiled — the same rule browsers
apply to whitespace in HTML text:

```
<lx.Button> 'Has
  several
  rows'
```

compiles to `text:"Has several rows"`. Wrapping (part of) the text in
`<pre>...</pre>` preserves its formatting instead — line breaks and relative
indentation inside `<pre>` are kept as-is, with only the indentation that
precedes the `<pre>` tag itself stripped from every line (so the block reads
naturally at whatever depth it's nested in the surrounding LXML):

```
<lx.Box> "
  <pre>
  A
    B
  C
  </pre>
"
```

compiles to `` text:"A\n  B\nC" ``. Several `<pre>` spans in the same text
attribute are each dedented independently (`<pre>` doesn't nest).

A quote character used as the text's own delimiter needs escaping (`\'`
inside a single-quoted text, `\"` inside a double-quoted one) — the other
quote type doesn't, so picking whichever delimiter avoids escaping in a given
string is usually the more readable choice.

### <a name="geometry">Geometry</a>

```
[left:top:width:height]
```

Any field can be left empty (`null`): `[10::100:100]` skips `top`. Two more
fields can be appended, in this fixed order — **right**, then **bottom**:
`[10:10:100:100:5]` (right only), `[10:10:100:100:5:8]` (right and bottom), or
`[10:10:100:100::8]` (bottom only, right skipped) — same empty-placeholder rule
as the base four.

As with the widget's own geometry API (see
[Widgets](https://github.com/epicoon/lxgo/tree/master/jspp/doc/widgets.md)),
only two of `{left, width, right}` and two of `{top, height, bottom}` end up
actually driving layout — specifying all three of a pair is redundant.

A bare number has no implicit unit — it's passed straight through to the
widget. A unit suffix can be attached to one field (`[0:0:100:100px]` — only
`height` gets `px`) or once after the closing bracket, applying to every field
that doesn't have its own suffix (`[0:0:100:100]px` — all four get `px`; a
field with its own suffix, e.g. `[0:10%:100:100]px`, keeps its own).

Geometry (and other positioning, like `#stream(...)`/`#grid(...)`) is also
where [positioning strategies](https://github.com/epicoon/lxgo/tree/master/jspp/doc/positioning-strategies.md)
come in — a strategy is just configured as a regular `#method(args)` call:

```
<lx.Box> @list [0:0:100%:100%]\
  #stream(direction:lx.VERTICAL, indent:'10px')
  <lx.Box> "item 1"
  <lx.Box> "item 2"
```

### Control flow

```
if a > 1:
  call: doA();
elseif a < 0:
  call: doB();
else:
  call: doC();
```

`a` above is a plain JS expression — typically a variable from the
surrounding code, but any expression works (`if a.length > 1:`, etc.).

`call: <expr>;` inserts `<expr>;` as a plain statement (the trailing `;` is
added automatically if omitted).

`for` accepts several shorthand forms besides the full C-style one; the
limit/count in every form is a plain JS expression too, so a variable from
the surrounding code works the same as a literal number:

```
for i = 1; i < count; i++:   // kept as-is
for i < someVar:             // implicit iterator, from 0
for i <= someVar:
for i = 1 to someVar:        // from 1 through someVar inclusive
for someVar:                 // implicit iterator "_iter", from 0 through someVar
```

### Reusable blocks

```
<*Greet(name)>
  call: console.log(name);

<&Greet('hi')>
```

`<*Name(args)>` defines a reusable block (must be at the top indentation
level); its nested lines are the block body. `<&Name(args)>` calls a
previously defined block from anywhere in the tree.

### Comments

`// ...` lines are ignored.


## Notes

- Widget names used in a tree are added to the compiled output's
  `lx.import(...)` calls automatically (except the built-in
  `lx.Rect`/`lx.Box`), so you don't need to import them yourself for widgets
  you only reference from LXML.
- LXML has no expression language of its own for anything beyond what's
  listed above — arbitrary logic belongs in `(...)`/`{...}`/`#method(...)`
  payloads or `call:` statements, which are plain JS.
