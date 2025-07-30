// @lx:namespace lx;
class ModelSchema {
	constructor(config = null) {
		this.fields = {};
		if (config) this.set(config);
	}

	isEmpty() {
		return this.fields.lxEmpty();
	}

	set(list) {
		if (lx.isArray(list)) {
			let temp = {};
			list.forEach(value=>temp[value] = {});
			list = temp;
		}
		this.fields = list.lxClone();
	}

	getPkName() {
		for (var i in this.fields)
			if (this.fields[i].type == 'pk') return i;
		return null;
	}

	getField(name) {
		return this.fields[name];
	}

	hasField(name) {
		return name in this.fields;
	}

	/**
	 * @param all {Boolean} - return with ref-fields
	 * @return {Array}
	 */
	getFieldNames(all = false) {
		let result = [];
		for (let name in this.fields) {
			if (!all && this.fields[name].ref !== undefined) continue;
			result.push(name);
		}
		return result;
	}

	getFieldTypes() {
		var result = {};
		for (var name in this.fields) {
			var type = lx.isObject(this.fields[name]) 
				? this.fields[name].type
				: this.fields[name];
			result[name] = type;
		}
		return result;
	}

	eachField(func) {
		for (var name in this.fields)
			func(this.fields[name], name);
	}

	getFieldsExportDefinition() {
		var result = [];
		for (var name in this.fields) {
			var def = this.fields[name];
			if (def.ref) result.push(name + ' << ' + def.ref);
			else result.push(name);
		}

		return result;
	}
}
