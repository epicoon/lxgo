// @lx:module lx.Form;

// @lx:use lx.Button;

/**
 * @widget lx.Form
 */
// @lx:namespace lx;
class Form extends lx.Box {
	/**
	 * list - hash table, entries - {name, info} - are converted to arguments for the method .field(className, fieldName, config)
	 *	- name - the name that will become the key value of the new element and the value of the field field
	 *	- info - info - either a class for creating a widget, or an array of a class and a config
	 */
	fields(list) {
		for (let key in list) {
			let item = list[key],
				config = {},
				widget;
			if (lx.isArray(item)) {
				widget = item[0];
				config = item[1];
			} else widget = item;
			this.field(key, widget, config);
		}
	}

	/**
	 * className - widget class
	 * fieldName - name that will become the key value of the new element and the value of the model field
	 * config - widget creating config
	 */
	field(fieldName, className, config = {}) {
		if (config.after && config.after.parent !== this) delete config.after;
		if (config.before && config.before.parent !== this) delete config.before;
		config.parent = this;

		config.key = fieldName;
		config.field = fieldName;
		let elem = new className(config);
	}

	/**
	 * Model values from elements that have a field
	 */
	content(map=null) {
		let obj = {},
			list = this.getChildren({ hasProperty: '_field', all: true });
		list.forEach(a=>{
			if (map !== null && lx.isArray(map) && !map.includes(a._field)) return;
			obj[a._field] = a.lxHasMethod('value')
				? a.value()
				: a.text();
		});
		return obj;
	}

	getFields(types = null) {
		if (types === null)
			return this.getChildren({ hasProperty: '_field', all: true });

		types = lx.isArray(types) ? types : [types];
		return this.getChildren(child=>{
			if (!('_field' in child)) return false;
			let match = false;
			types.forEach(type => {
				if (lx.isInstance(child, type)) match = true;
			});
			return match;
		}, true);
	}

	addButton(text='', config={}, onClick=null) {
		if (lx.isFunction(config)) {
			config = {
				click: config
			};
		} else if (lx.isFunction(onClick)) {
			config.click = onClick;
		}
		config.text = text;
		return this.add(lx.Button, config);
	}
}
