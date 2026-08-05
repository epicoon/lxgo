// Fields for storing bonds
let _binds = {},
	_matrixBinds = [],
	_bindCounter = 0;

const
	BIND_TYPE_FULL = 1,
	BIND_TYPE_WRITE = 2,
	BIND_TYPE_READ = 3;

/**
 * Bond variants:
 * - simple: single model field <-> single widget
 * - simple for form: model fields <-> widget with children
 * - agregated: collection of models <-> widget with children: available to change fields with same values for all models in collection
 * - matrix: collection of models <-> matrix-widget: can generate children for each model from collection
 */
// @lx:namespace lx;
class Binder extends lx.AppComponent {
	// @lx:const BIND_TYPE_FULL = BIND_TYPE_FULL;
	// @lx:const BIND_TYPE_WRITE = BIND_TYPE_WRITE;
	// @lx:const BIND_TYPE_READ = BIND_TYPE_READ;

	/**
	 * Simple
	 */
	bind(obj, widget, type = BIND_TYPE_FULL) {
		return _bind(obj, widget, type);
	}

	unbind(obj, widget = null) {
		return _unbind(obj, widget);
	}

	unbindWidget(widget) {
		_unbindWidget(widget);
	}

	/**
	 * Trigger changes - push model data into bound widgets
	 */
	push(obj, fieldName = null) {
		return _push(obj, fieldName);
	}

	// Pull current values from bound widgets into the model
	pull(obj, fieldName = null) {
		return _pull(obj, fieldName);
	}

	// Push this one widget's own current value out into its bound obj's field
	pushWidget(widget) {
		return _pushWidget(widget);
	}

	// Pull obj's current value for widget's own field into just this one widget
	pullWidget(widget) {
		return _pullWidget(widget);
	}

	makeWidgetMatrix(obj, info) {
		return _makeWidgetMatrix(obj, info);
	}

	setMatrixItemBox(obj, info) {
		return _setMatrixItemBox(obj, info);
	}

	setMatrixItemRender(obj, render) {
		return _setMatrixItemRender(obj, render);
	}

	addMatrixItemRender(obj, render) {
		return _addMatrixItemRender(obj, render);
	}

	unbindMatrix(widget) {
		return _unbindMatrix(widget);
	}

	bindMatrix(c, widget, type=BIND_TYPE_FULL) {
		return _bindMatrix(c, widget, type);
	}

	bindAggregation(c, widget, type=BIND_TYPE_FULL) {
		return _bindAggregation(c, widget, type);
	}

	getBind(id) {
		return _getBind(id);
	}
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * IMPLEMENTATION
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

// Shared core for simple/matrix binding and aggregation binding
// (_bindAggregation's bindFirst) - finds the widget(s) matching each of
// obj's fields among widget's children, registers all of them (regardless
// of _bindType - see _bindProcess/_action/_pullField for where the type
// actually gets applied), pushes obj's current value into them, and wires
// up write-back for write/full ones via the shared _onWidgetChange
// handler. bindTarget is what actually gets the shared lxBindId and holds
// the binding's read/write side (see _binds' own comment, above
// _genBindId) - defaults to obj itself; aggregation binding passes the
// whole collection instead, since values written back have to fan out to
// every item in it, not just obj (see _writeBind/_readBind).
function _attachBinding(obj, widget, type, bindTarget) {
	if (!obj.lxHasMethod('getSetterEvents')) return;
	var setterEvents = obj.getSetterEvents();
	if (!setterEvents) return;
	bindTarget = bindTarget || obj;

	var fields = setterEvents.fields;
	for (let i=0, l=fields.len; i<l; i++) {
		let _field = fields[i],
			c = widget.getChildren
				? widget.getChildren({hasProperties:{_field}, all:true})
				: new lx.Collection();

		if (widget._field == _field) c.add(widget);
		if (c.isEmpty()) continue;

		let matched = new lx.Collection(),
			writeWidgets = new lx.Collection();
		c.forEach(w => {
			if (w._bindType === undefined) w._bindType = type;
			if (w._isMatrix) {
				let val = obj[_field];
				if (val instanceof lx.Collection)
					w.matrix({items: val, type: w._bindType});
				return;
			}

			matched.add(w);
			if (w._bindType == BIND_TYPE_WRITE || w._bindType == BIND_TYPE_FULL) writeWidgets.add(w);
		});
		if (matched.isEmpty()) continue;

		_bindProcess(bindTarget, _field, matched);
		_action(bindTarget, _field, obj[_field]);

		writeWidgets.forEach(w => w.on('change', _onWidgetChange));
	}
}

function _bind(obj, widget, type=BIND_TYPE_FULL) {
	return _attachBinding(obj, widget, type);
}

// Shared 'change' handler for every write/full widget, attached by
// _attachBinding - a single named function instead of a closure allocated
// per bind() call, since it looks up its own binding record via
// this.lxBindId at fire time rather than closing over obj.
function _onWidgetChange(e) {
	const bind = _binds[this.lxBindId];
	if (!bind) return;
	let val = (e.newValue !== undefined)
		? e.newValue
		: this.value();
	_writeBind(bind, this._field, val);
}

function _unbind(obj, widget=null) {
	if (!obj.lxBindId) return;
	const bind = _binds[obj.lxBindId];
	if (!bind) {
		delete obj.lxBindId;
		return;
	}
	const fields = bind.fields;

	for (let name in fields) fields[name].lxForEachRevert((a)=> {
		if (!widget || (a === widget || a.hasAncestor(widget))) {
			delete a.lxBindId;
			_valueToWidgetWithoutBind(a, '');
			fields[name].lxRemove(a);
			if (fields[name].lxEmpty()) delete fields[name];
			// a.off('blur');
			a.off('change');
		}
	});

	if (fields.lxEmpty()) {
		delete _binds[obj.lxBindId];
		delete obj.lxBindId;
	}
}

function _push(obj, fieldName = null) {
	if (fieldName === null) {
		if (!obj.lxHasMethod('getSetterEvents')) return;
		var setterEvents = obj.getSetterEvents();
		if (!setterEvents) return;
		var fields = setterEvents.fields;
		for (let i=0, l=fields.len; i<l; i++) {
			let field = fields[i];
			_action(obj, field, obj[field]);
		}
	} else if (lx.isArray(fieldName)) {
		for (let i=0, l=fieldName.len; i<l; i++) {
			let field = fieldName[i];
			_action(obj, field, obj[field]);
		}
	} else {
		_action(obj, fieldName, obj[fieldName]);
	}
}

// Mirror of _push - reads the current value out of every widget bound to
// a field and writes it into the model (via _writeBind - direct
// obj[field] = val normally, fanned out to a whole collection for
// aggregation binding), instead of the other way round. Same _binds
// registry as _push/_action - no separate registry.
function _pull(obj, fieldName = null) {
	if (fieldName === null) {
		if (!obj.lxHasMethod('getSetterEvents')) return;
		var setterEvents = obj.getSetterEvents();
		if (!setterEvents) return;
		var fields = setterEvents.fields;
		for (let i=0, l=fields.len; i<l; i++) {
			let field = fields[i];
			_pullField(obj, field);
		}
	} else if (lx.isArray(fieldName)) {
		for (let i=0, l=fieldName.len; i<l; i++) {
			let field = fieldName[i];
			_pullField(obj, field);
		}
	} else {
		_pullField(obj, fieldName);
	}
}

// Mirror of _action - reads the current value from every write/full widget
// bound to field `name` of model `obj` and writes it into the model
// (read-only widgets have nothing meaningful to pull, same as _action skips
// write-only widgets on the way out).
function _pullField(obj, name) {
	if (!obj.lxBindId) return;
	const bind = _binds[obj.lxBindId];
	if (!bind) return;
	let arr = bind.fields[name];
	if (!arr || !lx.isArray(arr)) return;
	arr.forEach(a => {
		if (a._bindType == BIND_TYPE_READ) return;
		let val = _valueFromWidget(a);
		if (val !== undefined) _writeBind(bind, name, val);
	});
}

// Push this one widget's own current value out into its bound obj's field -
// direction is relative to the caller: pushBind/pullBind always mean "send
// my own data out"/"pull data toward me", so on a widget (unlike on the
// model) push means widget -> model. A read-only widget has nothing
// meaningful to push (mirrors _action skipping write-only widgets).
function _pushWidget(widget) {
	if (!widget.lxBindId || widget._field === undefined) return;
	if (widget._bindType == BIND_TYPE_READ) return;
	const bind = _binds[widget.lxBindId];
	if (!bind) return;
	let val = _valueFromWidget(widget);
	if (val !== undefined) _writeBind(bind, widget._field, val);
}

// Pull obj's current value for widget's own field into just this one widget.
// A write-only widget is never kept in sync from the model (mirrors
// _pullField skipping read-only widgets).
function _pullWidget(widget) {
	if (!widget.lxBindId || widget._field === undefined) return;
	if (widget._bindType == BIND_TYPE_WRITE) return;
	const bind = _binds[widget.lxBindId];
	if (!bind) return;
	_valueToWidget(widget, _readBind(bind, widget._field));
}

function _makeWidgetMatrix(obj, info) {
	_setMatrixItemBox(obj, info.itemBox);
	if (info.itemRender)
		_setMatrixItemRender(obj, info.itemRender)
	if (info.afterBind) obj.lxcwb_afterBind = info.afterBind;
}

function _setMatrixItemBox(obj, itemBox) {
	if (!itemBox) return;
	let widget, config;
	if (lx.isArray(itemBox)) {
		widget = itemBox[0];
		config = itemBox[1];
	} else widget = itemBox;
	if (widget) obj.lxcwb_widget = widget;
	if (config) obj.lxcwb_config = config;
}

function _setMatrixItemRender(obj, render) {
	obj.lxcwb_itemRender = [render];
}

function _addMatrixItemRender(obj, render) {
	if (!obj.lxcwb_itemRender)
		obj.lxcwb_itemRender = [];
	obj.lxcwb_itemRender.push(render);
}

function _bindMatrix(c, widget, type=BIND_TYPE_FULL) {
	if (!(c instanceof lx.Collection)) return;

	if (c._lxMatrixBindId === undefined) c._lxMatrixBindId = _genBindId();
	if (!(c._lxMatrixBindId in _matrixBinds))
		_matrixBinds[c._lxMatrixBindId] = {collection: c, type, widgets:[widget]};
	else
		_matrixBinds[c._lxMatrixBindId].widgets.push(widget);
	widget._lxMatrixBindId = c._lxMatrixBindId;

	widget.useRenderCache();
	c.forEach(a=>_matrixNewBox(widget, a, type));
	widget.applyRenderCache();

	c.addBehavior(lx.MethodListenerBehavior);
	c.afterMethod('add',       _collectionHandlerOnAdd   );
	c.afterMethod('insert',    _collectionHandlerOnInsert);
	c.beforeMethod('removeAt', _collectionHandlerOnRemove);
	c.beforeMethod('clear',    _collectionHandlerOnClear );
	c.afterMethod('set',       _collectionHandlerOnSet   );
	if (c.lxHasMethod('reset')) {
		c.beforeMethod('reset', ()=>widget.useRenderCache());
		c.afterMethod('reset', ()=>widget.applyRenderCache());
	}
	widget.on('swapped', _matrixHandlerOnSwapped);
}

function _unbindMatrix(widget) {
	if (widget._lxMatrixBindId === undefined) return;

	var bind = _matrixBinds[widget._lxMatrixBindId],
		c = bind.collection;
	c.first();
	let i = 0;
	while (c.current()) {
		_unbind(c.current(), widget.getAll('r').at(i++));
		c.next();
	}

	delete widget._lxMatrixBindId;
	bind.widgets.lxRemove(widget);
	if (bind.widgets.lxEmpty()) {
		delete _matrixBinds[c._lxMatrixBindId];
		delete c._lxMatrixBindId;
	}

	//TODO - remove from Collection all changes by lx.MethodListenerBehavior !!!
}

function _bindAggregation(c, widget, type=BIND_TYPE_FULL) {
	var first = c.first();

	// Blocking different fields in widget
	function disableDifferent() {
		var first = c.first();
		if (!first) return;
		var diff = _collectionDifferent(c);
		var fields = first.getSetterEvents().fields;
		for (var i=0; i<fields.len; i++) {
			var _field = fields[i],
				elem = widget.getChildren({hasProperties:{_field}, all:true}).at(0);
			if (elem) elem.disabled(_field in diff);
		}
	}

	// Bind the first element of the collection to the widget - shares
	// _attachBinding with plain _bind(), only bindTarget differs: the
	// collection itself, not obj, gets the shared lxBindId, so a value
	// coming back from a widget fans out to every item (see
	// _writeBind/_readBind), not just obj. Every current item of the
	// collection is then marked with that same shared id too - a
	// collection item carries lxBindId exactly the same way a widget
	// does (several widgets, or here several items, sharing one id).
	function bindFirst(obj) {
		_attachBinding(obj, widget, type, c);
		c.forEach(a => a.lxBindId = c.lxBindId);
	}

	// Check when adding/changing a collection item
	function checkNewObj(obj) {
		if (c.isEmpty()) bindFirst(obj);
		else if (c.first().constructor !== obj.constructor) return false;
		obj.lxBindId = c.lxBindId;
	}

	function unbindAll() {
		if (c.isEmpty()) return;
		c.first();
		let i = 0;
		while(c.current()) {
			_unbind(c.current(), widget.getAll('r').at(i++));
			c.next();
		}
	};

	// Event handlers
	c.addBehavior(lx.MethodListenerBehavior);
	c.beforeMethod('remove', elem=>{
		// unbindAll() (when this is the last item) needs elem.lxBindId
		// intact to find _binds[id] - clear it only after, not before.
		if (c.len == 1 && c.at(0) === elem) unbindAll();
		delete elem.lxBindId;
	});
	c.afterMethod('remove', elem=>{
		if (c.isEmpty()) return;
		bindFirst(c.first());
		disableDifferent();
	});
	c.beforeMethod('removeAt', i=>{
		if (!c.len) unbindAll();
		delete c.at(i).lxBindId;
	});
	c.afterMethod('removeAt', i=>{
		if (c.isEmpty()) return;
		if (i == 0) bindFirst(c.first());
		disableDifferent();
	});
	c.beforeMethod('add', (obj)=>checkNewObj(obj));
	c.afterMethod('add', disableDifferent);
	c.beforeMethod('set', (i, obj)=>checkNewObj(obj));
	c.afterMethod('set', disableDifferent);
	c.beforeMethod('clear', unbindAll);

	c.lxBindWidget = widget;
	if (first) {
		bindFirst(first);
		disableDifferent();
	}
}

// Returns the bind's {fieldName: [widgets]} map - the shape Binder.getBind()/
// model.getBind() have always publicly returned, even though internally
// _binds[id] now carries more (obj) alongside fields.
function _getBind(id) {
	const bind = _binds[id];
	return bind ? bind.fields : undefined;
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * INNER
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

// Id shared by everything bound together - a bind target (usually a plain
// model, or a whole lx.Collection for aggregation binding) and every
// widget bound to it - key into _binds. _binds[id] = { obj, fields:
// {fieldName: [widgets]} } - obj is the read/write target itself (see
// _writeBind/_readBind for how a Collection there is handled - reading
// uses its first item, writing fans out to every item); fields is the
// same {fieldName: [widgets]} map Binder.getBind()/model.getBind() have
// always returned. A model shares its id with every widget bound to it
// the same way several aggregated collection items now share one id too
// (see _bindAggregation) - not just the one representative "first" item.
function _genBindId() {
	return 'b' + _bindCounter++;
}

function _collectionDifferent(c) {
	if (c.isEmpty()) return {};
	c.cachePosition();
	var first = c.first(),
		fields = first.getSetterEvents().fields,
		boof = {};
	while (obj = c.next()) {
		for (var i=0; i<fields.len; i++) {
			var f = fields[i];
			if ( obj[f] != first[f] ) boof[f] = 1;
		}
	}
	c.loadPosition();
	return boof;
}
// Disables widgets whose field currently differs across an aggregated
// collection c (the same collection stored as bind.obj, see _bindAggregation).
function _collectionAction(c, _field) {
	if (!c.lxBindWidget) return;
	const widgets = c.lxBindWidget.getChildren({hasProperties:{_field}, all:true});
	const diff = _collectionDifferent(c);
	widgets.forEach(w=>w.disabled(w._field in diff));
}

// Method for updating widgets associated with field `name` of model `obj` -
// write-only widgets never receive a pushed display value (mirrors
// _pullField skipping read-only ones for the opposite direction).
function _action(obj, name, newVal) {
	if (!obj.lxBindId) return;
	const bind = _binds[obj.lxBindId];
	if (!bind) {
		delete obj.lxBindId;
		return;
	}
	if (bind.obj instanceof lx.Collection) _collectionAction(bind.obj, name);

	let arr = bind.fields[name];
	if (!arr || !lx.isArray(arr)) return;
	arr.forEach(a => {
		if (a._bindType == BIND_TYPE_WRITE) return;
		_valueToWidget(a, newVal);
	});
}

// Without refresh model
function _valueToWidgetWithoutBind(widget, value) {
	if (widget.lxHasMethod('innerValue'))
		widget.innerValue(value);
	else if (widget.lxHasMethod('value'))
		widget.value(value);
	else if (widget.lxHasMethod('text'))
		widget.text(value);
}

// Method for directly placing a value into a widget
function _valueToWidget(widget, value) {
	if (widget.lxHasMethod('__bindValue'))
		widget.__bindValue(value);
	else if (widget.lxHasMethod('value'))
		widget.value(value);
	else if (widget.lxHasMethod('text'))
		widget.text(value);
}

// Method for reading the current value straight out of a widget
function _valueFromWidget(widget) {
	if (widget.lxHasMethod('value'))
		return widget.value();
	if (widget.lxHasMethod('text'))
		return widget.text();
	return undefined;
}

// Unlink a widget from a model field, if there are no widgets left in this relationship, the relationship is deleted
// The connection ID will be removed from the model when its field is changed, when the connection is not found during an attempt to update
function _unbindWidget(widget) {
	if (!widget.lxBindId) return;
	const bind = _binds[widget.lxBindId];
	if (bind) {
		bind.fields[widget._field].lxRemove(widget);
		if (bind.fields[widget._field].lxEmpty())
			delete bind.fields[widget._field];
		if (bind.fields.lxEmpty())
			delete _binds[widget.lxBindId];
	}
	delete widget.lxBindId;
	// widget.off('blur');
	widget.off('change');
}

// Bind fields `name` of a bind target with widgets - target is usually a
// plain model (see _binds' own comment, above _genBindId) but can be a
// whole lx.Collection for aggregation binding (see _bindAggregation).
//
// Every widget is unbound *before* the target's own _binds record is
// looked up/created, not interleaved widget-by-widget - re-binding the
// same widget to the same target (as happens whenever aggregation binding
// re-runs bindFirst for a new representative item) would otherwise risk
// _unbindWidget deleting _binds[id] out from under this very call, right
// before trying to register into it (if that widget was the id's last
// remaining registration).
function _bindProcess(target, name, widgets) {
	widgets.forEach(a => _unbindWidget(a));

	if (!target.lxBindId)
		target.lxBindId = _genBindId();
	let bind = _binds[target.lxBindId];
	if (!bind)
		bind = _binds[target.lxBindId] = { obj: target, fields: {} };
	if (!(name in bind.fields))
		bind.fields[name] = [];

	widgets.forEach(a => {
		a.lxBindId = target.lxBindId;
		bind.fields[name].push(a);
	});
}

// Writes val into field `name` of a binding's actual target: bind.obj
// itself normally, or - since aggregation binding stores the whole
// collection as bind.obj - every item currently in it. One function
// reused for every binding, not a closure allocated per bind() call.
function _writeBind(bind, name, val) {
	if (bind.obj instanceof lx.Collection)
		bind.obj.forEach(el => el[name] = val);
	else
		bind.obj[name] = val;
}

// Mirror of _writeBind for reading - bind.obj itself normally, or the
// first item of the collection for aggregation binding (the same item
// _attachBinding/_action already treat as the representative one to push
// from).
function _readBind(bind, name) {
	const src = bind.obj instanceof lx.Collection ? bind.obj.first() : bind.obj;
	return src ? src[name] : undefined;
}

function _getMatrixCollection(widget) {
	return _matrixBinds[widget._lxMatrixBindId].collection;
}

function _prepareMatrixNewBoxConfig(w) {
	let rowConfig = w.lxcwb_config ? w.lxcwb_config.lxClone() : {}
	rowConfig.key = 'r';
	rowConfig.parent = w;
	return rowConfig;
}

function _matrixNewBox(w, obj, type, rowConfig = null) {
	rowConfig = rowConfig || _prepareMatrixNewBoxConfig(w);
	let rowClass = w.lxcwb_widget || lx.Box;
	let r = new rowClass(rowConfig);
	r.matrixItems = function() {return _getMatrixCollection(this.parent);};
	r.matrixIndex = function() {return this.index || 0;};
	r.matrixModel = function() {return _getMatrixCollection(this.parent).at(this.index || 0);};
	r.useRenderCache();
	r.begin();
	w.trigger('renderMatrixItem', w.newEvent({box: r, model: obj}))
	if (w.lxcwb_itemRender)
		w.lxcwb_itemRender.forEach(render => render(r, obj));
	r.end();
	r.applyRenderCache();
	_bind(obj, r, type);
	if (w.lxcwb_afterBind) w.lxcwb_afterBind(r, obj);
}

function _matrixInsertNewBox(w, obj, index, type) {
	if (index > w.childrenCount()) index = w.childrenCount();
	if (index == w.childrenCount()) {
		_matrixNewBox(w, obj, type);
		return;
	}

	let rowConfig = _prepareMatrixNewBoxConfig(w);
	rowConfig.before = w.child(index);
	_matrixNewBox(w, obj, type, rowConfig);
}

function _collectionHandlerOnAdd(obj = null) {
	if (this._lxMatrixBindId === undefined || this._lxMatrixBindLocked) return;
	var widgets = _matrixBinds[this._lxMatrixBindId].widgets;
	widgets.forEach(w=>_matrixNewBox(w, this.last(), _matrixBinds[this._lxMatrixBindId].type));
}

function _collectionHandlerOnInsert(i, obj = null) {
	if (this._lxMatrixBindId === undefined) return;
	var widgets = _matrixBinds[this._lxMatrixBindId].widgets;
	widgets.forEach(w=>_matrixInsertNewBox(w, this.at(i), i, _matrixBinds[this._lxMatrixBindId].type));
}

function _collectionHandlerOnRemove(i) {
	if (this._lxMatrixBindId === undefined) return;
	var widgets = _matrixBinds[this._lxMatrixBindId].widgets;
	widgets.lxForEachRevert((w)=>{
		_unbind(this.at(i), w.getAll('r').at(i));
		w.del('r', i);
	});
}

function _collectionHandlerOnClear() {
	if (this._lxMatrixBindId === undefined || this._lxMatrixBindLocked) return;

	var widgets = _matrixBinds[this._lxMatrixBindId].widgets;
	widgets.forEach(w=>{
		this.first();
		let i = 0;
		while (this.current()) {
			_unbind(this.current(), w.getAll('r').at(i++));
			this.next();
		}
		w.del('r');
		if (w.isEmpty()) w.positioning().onClearOwner();
	});
}

function _collectionHandlerOnSet(i, obj) {
	if (this._lxMatrixBindId === undefined) return;
	var widgets = _matrixBinds[this._lxMatrixBindId].widgets,
		type = _matrixBinds[this._lxMatrixBindId].type;
	widgets.lxForEachRevert((w)=>{
		_bind(this.at(i), w.getAll('r').at(i), type);
	});
}

function _matrixHandlerOnSwapped(e) {
	const c = _getMatrixCollection(this);
	c.lockMethodListener();
	c.swap(e.from, e.to);
	c.unlockMethodListener();
}
