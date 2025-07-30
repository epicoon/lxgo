// Fields for storing bonds
let __binds = {},
	__matrixBinds = [],
	__bindCounter = 0;

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
		return __bind(obj, widget, type);
	}

	unbind(obj, widget = null) {
		return __unbind(obj, widget);
	}

	unbindWidget(widget) {
		__unbindWidget(widget);
	}

	/**
	 * Trigger changes
	 */
	refresh(obj, fieldName = null) {
		return __refresh(obj, fieldName);
	}

	makeWidgetMatrix(obj, info) {
		return __makeWidgetMatrix(obj, info);
	}

	setMatrixItemBox(obj, info) {
		return __setMatrixItemBox(obj, info);
	}

	setMatrixItemRender(obj, render) {
		return __setMatrixItemRender(obj, render);
	}

	addMatrixItemRender(obj, render) {
		return __addMatrixItemRender(obj, render);
	}

	unbindMatrix(widget) {
		return __unbindMatrix(widget);
	}

	bindMatrix(c, widget, type=BIND_TYPE_FULL) {
		return __bindMatrix(c, widget, type);
	}

	bindAggregation(c, widget, type=BIND_TYPE_FULL) {
		return __bindAggregation(c, widget, type);
	}

	getBind(id) {
		return __getBind(id);
	}
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * IMPLEMENTATION
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

function __bind(obj, widget, type=BIND_TYPE_FULL) {
	if (!obj.lxHasMethod('getSetterEvents')) return;
	var setterEvents = obj.getSetterEvents();
	if (!setterEvents) return;

	var fields = setterEvents.fields;
	for (let i=0, l=fields.len; i<l; i++) {
		let _field = fields[i],
			c = widget.getChildren
				? widget.getChildren({hasProperties:{_field}, all:true})
				: new lx.Collection();

		if (widget._field == _field) c.add(widget);
		if (c.isEmpty()) continue;

		var readWidgets = new lx.Collection(),
			writeWidgets = new lx.Collection();
		c.forEach(widget => {
			if (widget._bindType === undefined) widget._bindType = type;
			if (widget._isMatrix) {
				let val = obj[_field];
				if (val instanceof lx.Collection)
					widget.matrix({items: val, type: widget._bindType})
				return;
			}

			if (widget._bindType == BIND_TYPE_READ || widget._bindType == BIND_TYPE_FULL) readWidgets.add(widget);
			if (widget._bindType == BIND_TYPE_WRITE || widget._bindType == BIND_TYPE_FULL) writeWidgets.add(widget);
		});

		function actualize(a, val) {
			obj[a._field] = val;
		};
		if (!readWidgets.isEmpty()) {
			__bindProcess(obj, _field, readWidgets);
			__action(obj, _field, obj[_field]);
		}

		writeWidgets.forEach(a=>{
			a.on('change', function(e) {
				let val = (e.newValue !== undefined)
					? e.newValue
					: a.value()
				actualize(this, val);
			});
		});
	}
}

function __unbind(obj, widget=null) {
	if (!obj.lxBindId) return;
	var bb = __getBind(obj.lxBindId);

	for (let name in bb) bb[name].lxForEachRevert((a)=> {
		if (!widget || (a === widget || a.hasAncestor(widget))) {
			delete a.lxBindId;
			__valueToWidgetWithoutBind(a, '');
			__binds[obj.lxBindId][name].lxRemove(a);
			if (__binds[obj.lxBindId][name].lxEmpty()) delete __binds[obj.lxBindId][name];
			// a.off('blur');
			a.off('change');
		}
	});

	if (__binds[obj.lxBindId].lxEmpty()) {
		delete __binds[obj.lxBindId];
		delete obj.lxBindId;
	}
}

function __refresh(obj, fieldName = null) {
	if (fieldName === null) {
		if (!obj.lxHasMethod('getSetterEvents')) return;
		var setterEvents = obj.getSetterEvents();
		if (!setterEvents) return;
		var fields = setterEvents.fields;
		for (let i=0, l=fields.len; i<l; i++) {
			let field = fields[i];
			__action(obj, field, obj[field]);
		}
	} else if (lx.isArray(fieldName)) {
		for (let i=0, l=fieldName.len; i<l; i++) {
			let field = fieldName[i];
			__action(obj, field, obj[field]);
		}
	} else {
		__action(obj, fieldName, obj[fieldName]);
	}
}

function __makeWidgetMatrix(obj, info) {
	__setMatrixItemBox(obj, info.itemBox);
	if (info.itemRender)
		__setMatrixItemRender(obj, info.itemRender)
	if (info.afterBind) obj.lxcwb_afterBind = info.afterBind;
}

function __setMatrixItemBox(obj, itemBox) {
	if (!itemBox) return;
	let widget, config;
	if (lx.isArray(itemBox)) {
		widget = itemBox[0];
		config = itemBox[1];
	} else widget = itemBox;
	if (widget) obj.lxcwb_widget = widget;
	if (config) obj.lxcwb_config = config;
}

function __setMatrixItemRender(obj, render) {
	obj.lxcwb_itemRender = [render];
}

function __addMatrixItemRender(obj, render) {
	if (!obj.lxcwb_itemRender)
		obj.lxcwb_itemRender = [];
	obj.lxcwb_itemRender.push(render);
}

function __bindMatrix(c, widget, type=BIND_TYPE_FULL) {
	if (!(c instanceof lx.Collection)) return;

	if (c._lxMatrixBindId === undefined) c._lxMatrixBindId = __genBindId();
	if (!(c._lxMatrixBindId in __matrixBinds))
		__matrixBinds[c._lxMatrixBindId] = {collection: c, type, widgets:[widget]};
	else
		__matrixBinds[c._lxMatrixBindId].widgets.push(widget);
	widget._lxMatrixBindId = c._lxMatrixBindId;

	widget.useRenderCache();
	c.forEach(a=>__matrixNewBox(widget, a, type));
	widget.applyRenderCache();

	c.addBehavior(lx.MethodListenerBehavior);
	c.afterMethod('add',       __matrixHandlerOnAdd   );
	c.afterMethod('insert',    __matrixHandlerOnInsert);
	c.beforeMethod('removeAt', __matrixHandlerOnRemove);
	c.beforeMethod('clear',    __matrixHandlerOnClear );
	c.afterMethod('set',       __matrixHandlerOnSet   );
	if (c.lxHasMethod('reset')) {
		c.beforeMethod('reset', ()=>widget.useRenderCache());
		c.afterMethod('reset', ()=>widget.applyRenderCache());
	}
	if (widget.positioning().lxClassName() == 'StreamPositioningStrategy') {
		widget.on('beforeStreamItemRelocation', __beforeStreamContentRelocation);
		widget.on('afterStreamItemRelocation', __afterStreamContentRelocation);
	}
}

function __unbindMatrix(widget) {
	if (widget._lxMatrixBindId === undefined) return;

	var bind = __matrixBinds[widget._lxMatrixBindId],
		c = bind.collection;
	c.first();
	let i = 0;
	while (c.current()) {
		__unbind(c.current(), widget.getAll('r').at(i++));
		c.next();
	}

	delete widget._lxMatrixBindId;
	bind.widgets.lxRemove(widget);
	if (bind.widgets.lxEmpty()) {
		delete __matrixBinds[c._lxMatrixBindId];
		delete c._lxMatrixBindId;
	}

	//TODO - remove from Collection all changes by lx.MethodListenerBehavior !!!
}

function __bindAggregation(c, widget, type=BIND_TYPE_FULL) {
	var first = c.first();

	c.forEach(a=> a.lxBindC = c);

	// Blocking different fields in widget
	function disableDifferent() {
		var first = c.first();
		if (!first) return;
		var diff = __collectionDifferent(c);
		var fields = first.getSetterEvents().fields;
		for (var i=0; i<fields.len; i++) {
			var _field = fields[i],
				elem = widget.getChildren({hasProperties:{_field}, all:true}).at(0);
			if (elem) elem.disabled(_field in diff);
		}
	}

	// Bind the first element of the collection to the widget
	//TODO close to __bind()
	function bindFirst(obj) {
		if (!obj.lxHasMethod('getSetterEvents')) return;
		var setterEvents = obj.getSetterEvents();
		if (!setterEvents) return;

		var fields = setterEvents.fields;
		for (let i=0, l=fields.len; i<l; i++) {
			let _field = fields[i],
				cw = widget.getChildren
					? widget.getChildren({hasProperties:{_field}, all:true})
					: new lx.Collection();

			if (widget._field == _field) cw.add(widget);
			if (cw.isEmpty()) continue;

			var readWidgets = new lx.Collection(),
				writeWidgets = new lx.Collection();
			cw.forEach(widget=>{
				if (widget._bindType === undefined) widget._bindType = type;
				if (widget._bindType == BIND_TYPE_READ || widget._bindType == BIND_TYPE_FULL) readWidgets.add(widget);
				if (widget._bindType == BIND_TYPE_WRITE || widget._bindType == BIND_TYPE_FULL) writeWidgets.add(widget);
			});
			function actualize(a, val) {
				c.forEach(el=>el[_field] = val);
			}
			if (!readWidgets.isEmpty()) {
				__bindProcess(obj, _field, readWidgets);
				__action(obj, _field, obj[_field]);
			}
			writeWidgets.forEach(a=>{
				a.on('change', function(e) {
					let val = (e.newValue !== undefined)
						? e.newValue
						: a.value()
					actualize(this, val);
				});
			});
		}
	}

	// Check when adding/changing a collection item
	function checkNewObj(obj) {
		if (c.isEmpty()) bindFirst(obj);
		else if (c.first().constructor !== obj.constructor) return false;
		obj.lxBindC = c;
	}

	function unbindAll() {
		if (c.isEmpty()) return;
		c.first();
		let i = 0;
		while(c.current()) {
			__unbind(c.current(), widget.getAll('r').at(i++));
			c.next();
		}
	};

	// Event handlers
	c.addBehavior(lx.MethodListenerBehavior);
	c.beforeMethod('remove', elem=>{
		delete elem.lxBindC;
		if (c.len == 1 && c.at(0) === elem) unbindAll();
	});
	c.afterMethod('remove', elem=>{
		if (c.isEmpty()) return;
		bindFirst(c.first());
		disableDifferent();
	});
	c.beforeMethod('removeAt', i=>{
		delete c.at(i).lxBindC;
		if (!c.len) unbindAll();
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

function __getBind(id) {
	return __binds[id];
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * INNER
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

// Id single for model but can be bound with different widgets
// bond chain: model.id => Binder.binds[id] => bind=fields[] => field=widgets[]
// bond is array: keys - field names, values - arrays of bound widgets
function __genBindId() {
	return 'b' + __bindCounter++;
}

function __collectionDifferent(c) {
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
function __collectionAction(obj, _field) {
	if (!obj.lxBindC.lxBindWidget) return;
	const widgets = obj.lxBindC.lxBindWidget.getChildren({hasProperties:{_field}, all:true});
	const diff = __collectionDifferent(obj.lxBindC);
	widgets.forEach(w=>w.disabled(w._field in diff));
}

// Method for updating widgets associated with field `name` of model `obj`
function __action(obj, name, newVal) {
	if (obj.lxBindC) __collectionAction(obj, name);

	if (!obj.lxBindId) return;
	if (!(obj.lxBindId in __binds)) {
		delete obj.lxBindId;
		return;
	}
	let arr = __getBind(obj.lxBindId)[name];
	if (!arr || !lx.isArray(arr)) return;
	arr.forEach(a=>__valueToWidget(a, newVal));
}

// Without refresh model
function __valueToWidgetWithoutBind(widget, value) {
	if (widget.lxHasMethod('innerValue'))
		widget.innerValue(value);
	else if (widget.lxHasMethod('value'))
		widget.value(value);
	else if (widget.lxHasMethod('text'))
		widget.text(value);
}

// Method for directly placing a value into a widget
function __valueToWidget(widget, value) {
	if (widget.lxHasMethod('value'))
		widget.value(value);
	else if (widget.lxHasMethod('text'))
		widget.text(value);
}

// Unlink a widget from a model field, if there are no widgets left in this relationship, the relationship is deleted
// The connection ID will be removed from the model when its field is changed, when the connection is not found during an attempt to update
function __unbindWidget(widget) {
	if (!widget.lxBindId) return;
	__binds[widget.lxBindId][widget._field].lxRemove(widget);
	if (__binds[widget.lxBindId][widget._field].lxEmpty())
		delete __binds[widget.lxBindId][widget._field];
	if (__binds[widget.lxBindId].lxEmpty())
		delete __binds[widget.lxBindId];
	delete widget.lxBindId;
	// widget.off('blur');
	widget.off('change');
}

// Binds a widget to a specific ID, if there is no connection to that ID, it will be created
function __bindWidget(widget, id) {
	__unbindWidget(widget);
	widget.lxBindId = id;
	if (!(id in __binds))
		__binds[id] = {};
	if (!(widget._field in __binds[id]))
		__binds[id][widget._field] = [];
	__binds[id][widget._field].push(widget);
}

// Bind fields `name` of model `obj` with widgets
function __bindProcess(obj, name, widgets) {
	if (!obj.lxBindId)
		obj.lxBindId = __genBindId();
	if (!(obj.lxBindId in __binds))
		__binds[obj.lxBindId] = {};
	if (!(name in __binds[obj.lxBindId]))
		__binds[obj.lxBindId][name] = [];
	widgets.forEach(a=>__bindWidget(a, obj.lxBindId));
}

function __getMatrixCollection(widget) {
	return __matrixBinds[widget._lxMatrixBindId].collection;
}

function __prepareMatrixNewBoxConfig(w) {
	let rowConfig = w.lxcwb_config ? w.lxcwb_config.lxClone() : {}
	rowConfig.key = 'r';
	rowConfig.parent = w;
	return rowConfig;
}

function __matrixNewBox(w, obj, type, rowConfig = null) {
	rowConfig = rowConfig || __prepareMatrixNewBoxConfig(w);
	let rowClass = w.lxcwb_widget || lx.Box;
	let r = new rowClass(rowConfig);
	r.matrixItems = function() {return __getMatrixCollection(this.parent);};
	r.matrixIndex = function() {return this.index || 0;};
	r.matrixModel = function() {return __getMatrixCollection(this.parent).at(this.index || 0);};
	r.useRenderCache();
	r.begin();
	w.trigger('renderMatrixItem', w.newEvent({box: r, model: obj}))
	if (w.lxcwb_itemRender)
		w.lxcwb_itemRender.forEach(render => render(r, obj));
	r.end();
	r.applyRenderCache();
	__bind(obj, r, type);
	if (w.lxcwb_afterBind) w.lxcwb_afterBind(r, obj);
}

function __matrixInsertNewBox(w, obj, index, type) {
	if (index > w.childrenCount()) index = w.childrenCount();
	if (index == w.childrenCount()) {
		__matrixNewBox(w, obj, type);
		return;
	}

	let rowConfig = __prepareMatrixNewBoxConfig(w);
	rowConfig.before = w.child(index);
	__matrixNewBox(w, obj, type, rowConfig);
}

function __matrixHandlerOnAdd(obj = null) {
	if (this._lxMatrixBindId === undefined || this._lxMatrixBindLocked) return;
	var widgets = __matrixBinds[this._lxMatrixBindId].widgets;
	widgets.forEach(w=>__matrixNewBox(w, this.last(), __matrixBinds[this._lxMatrixBindId].type));
}

function __matrixHandlerOnInsert(i, obj = null) {
	if (this._lxMatrixBindId === undefined) return;
	var widgets = __matrixBinds[this._lxMatrixBindId].widgets;
	widgets.forEach(w=>__matrixInsertNewBox(w, this.at(i), i, __matrixBinds[this._lxMatrixBindId].type));
}

function __matrixHandlerOnRemove(i) {
	if (this._lxMatrixBindId === undefined) return;
	var widgets = __matrixBinds[this._lxMatrixBindId].widgets;
	widgets.lxForEachRevert((w)=>{
		__unbind(this.at(i), w.getAll('r').at(i));
		w.del('r', i);
	});
}

function __matrixHandlerOnClear() {
	if (this._lxMatrixBindId === undefined || this._lxMatrixBindLocked) return;

	var widgets = __matrixBinds[this._lxMatrixBindId].widgets;
	widgets.forEach(w=>{
		this.first();
		let i = 0;
		while (this.current()) {
			__unbind(this.current(), w.getAll('r').at(i++));
			this.next();
		}
		w.del('r');
		if (w.isEmpty()) w.positioning().onClearOwner();
	});
}

function __matrixHandlerOnSet(i, obj) {
	if (this._lxMatrixBindId === undefined) return;
	var widgets = __matrixBinds[this._lxMatrixBindId].widgets,
		type = __matrixBinds[this._lxMatrixBindId].type;
	widgets.lxForEachRevert((w)=>{
		__bind(this.at(i), w.getAll('r').at(i), type);
	});
}

function __beforeStreamContentRelocation() {
	let c = __getMatrixCollection(this);
	c._lxMatrixBindLocked = true;
	this.getChildren(child=>{
		child._lxMatrixModel = child.matrixModel();
	});
}

function __afterStreamContentRelocation() {
	let c = __getMatrixCollection(this);
	c.clear();
	this.getChildren(child=>{
		c.add(child._lxMatrixModel);
		delete child._lxMatrixModel;
	});
	delete c._lxMatrixBindLocked;
}
