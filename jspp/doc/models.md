# Models and binding

A **model** is a schema-defined data holder (`lx.Model` and its reactive
subclass `lx.BindableModel`). A model can be **bound** to a widget (or a tree
of widgets) so that changing a model field automatically refreshes the widgets
that display it.


## Defining a model

Two ways to get a model class with a schema:

* Subclass and describe the schema statically:
  ```js
  // @lx:namespace app;
  class UserModel extends lx.BindableModel {
      static schema() {
          return {
              id: {type: lx.ModelTypeEnum.PK},
              name: {type: lx.ModelTypeEnum.STRING, default: ''},
              age: {type: lx.ModelTypeEnum.NUMBER, default: 0},
              active: {type: lx.ModelTypeEnum.BOOLEAN, default: true}
          };
      }
  }

  const user = new app.UserModel({name: 'Alex', age: 30});
  ```

* Build an anonymous model class on the fly with `create()`:
  ```js
  const user = lx.Model.create({
      name: {type: lx.ModelTypeEnum.STRING},
      age: {type: lx.ModelTypeEnum.NUMBER}
  });
  ```

Field types come from `lx.ModelTypeEnum`: `NUMBER`, `STRING`, `BOOLEAN`, `PK`. A
field without a `type` is stored as-is (untyped). A field with
`type: lx.ModelTypeEnum.PK` marks the model's primary key (see `getPk()`
below). A field can also be
declared as a reference to another property with `ref`:
```js
{ fullName: {ref: 'name'} } // getField('fullName') reads/writes `this.name`
```

### Model instance API
* `getField(name)` / `setField(name, value)` — read/write a single field
  (respects `ref`).
* `getFields(map = null)` / `setFields(data)` — read/write several fields at
  once; `map`/`data` keys must exist in the schema.
* `resetField(name)` / `resetFields(map = null)` — reset field(s) to their
  schema default (or a built-in default per type).
* `getPk()` — value of the field marked `{type: lx.ModelTypeEnum.PK}`, or `undefined`.
* `getSchema()` — the model's `lx.ModelSchema` instance.


## Reactive models: `lx.BindableModel`

`lx.BindableModel extends lx.Model` and adds:

* **Validation on set** — if an assigned value doesn't match the field's
  declared type, the assignment is rejected (the field keeps its previous
  value) and `onValidateFailed(field, value)` is called. Override it per
  instance:
  ```js
  user.onValidateFailed((field, value) => {
      console.warn(`invalid value for ${field}:`, value);
  });
  ```
* **Auto-refresh on set** — after a field changes, every widget currently
  bound to that field is refreshed automatically (via `bindRefresh`).
* `bind(widgets, type = lx.app.binder.BIND_TYPE_FULL)` — bind the model to one
  widget or an array of widgets.
* `unbind(widget = null)` — unbind everything, or a single widget.
* `getBind()` / `getWidgetsForField(field)` — inspect current bindings.
* `bindRefresh(fieldNames = null)` — manually re-push field values into bound
  widgets.


## Binding a widget to a model

Any `lx.Rect`-based widget (e.g. `lx.Box`) exposes:
```js
box.bind(model);   // model.bind(box, type)
box.unbind();
```
Binding walks the widget's children looking for elements marked with a
`_field` name and keeps their content/value in sync with the matching model
field in both directions.

The actual synchronization logic lives in the `binder` application component
(`lx.app.binder`, key `binder`, client-only) — it is an internal mechanism and
not part of the public API described here.


## An example of building a form from an LXML template

The `_field`/`_isMatrix` markers a widget needs for binding are exactly what
the LXML `[f:name]`/`[m:name]` attributes set (see [LXML](https://github.com/epicoon/lxgo/tree/master/jspp/doc/lxml.md)),
so a form is normally just an LXML tree plus `lx.BindableModel.createForForm()`
— no dedicated form widget is required:

```js
const tpl = lx.ml(`
<lx.Box> @form [10:10:80:80] #border()
  <lx.Box> @grid\
    #grid(cols:4, indent:'10px')
    <lx.Input> [f:name] (width:2)
    <lx.Input> [f:amount] (width:2)
    <lx.Box> [0:1:3:3] #border()
      <lx.Box> [m:features]\
        #stream(indent:'10px')
        <lx.Box>\
          #grid()
          <lx.Input> [f:label] (width:10)
          <lx.Box> @del #fill('red')
`);

let model = lx.BindableModel.createForForm(tpl.form);

tpl.features.addMatrixItemRender((box, feature) => {
    box.find('del').click(() => model.features.remove(feature));
});

model.name = 'Umbrella';
model.amount = 100;
model.features.add({ label: 'Feature 1' });
model.features.add({ label: 'Feature 2' });
```

* `Model.createForForm(form)` walks `form`'s children recursively, collects
  every `[f:name]` into an untyped schema field and every `[m:name]` into a
  `lx.ModelCollection` schema field (its own sub-schema is inferred the same
  way, from the matrix's item template registered via the widget's
  `renderMatrixItem` event — see below), then creates the model and calls
  `model.bind(form)` for you.
* `widget.addMatrixItemRender((box, feature) => {...})` (available on any
  `lx.Box`-based widget bound as a matrix, e.g. `tpl.features` above) registers
  a callback that runs once per item every time the bound collection renders
  one — `box` is the newly created item box (built from the `[m:features]`
  widget's own child template), `feature` is the corresponding model instance.
  Adding to the underlying `lx.ModelCollection` (`model.features.add({...})`)
  is what triggers a new item to be rendered.


## Collections: `lx.ModelCollection`

`lx.ModelCollection` (extends `lx.Collection`) is an ordered list of instances
of the same model class:

```js
const users = lx.ModelCollection.create({
    schema: {
        name: {type: lx.ModelTypeEnum.STRING},
        age: {type: lx.ModelTypeEnum.NUMBER}
    },
    list: [{name: 'Alex', age: 30}, {name: 'Kate', age: 25}]
});

users.add({name: 'Sam', age: 40});   // wraps plain data into a model instance
users.forEach(u => console.log(u.getField('name')));
```

* `add(data)` / `set(i, data)` / `insert(i, data)` — accept either a plain
  object (wrapped into a new model instance) or an existing model instance.
* `load(list)` — `add()` every item of an array.
* `reset(list)` — clear the collection, then optionally `load(list)`.
* `removeByData(data)` / `searchIndexesByData(data)` — find/remove items whose
  fields match a partial data object.
* `unbind()` — unbind every item in the collection.

If you don't want to write a schema by hand, `lx.ModelCollection.createByData(list)`
infers a schema from the value types found in `list[0]` (or, with
`byFirst = false`, scans the whole list).

See `lx.ModelsGrid` (in the [widgets](https://github.com/epicoon/lxgo/tree/master/jspp/doc/widgets.md) set) for a real consumer of a
model collection bound to a table-like widget.
