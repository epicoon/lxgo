// @lx:namespace lx;
class ModelCollection extends lx.Collection {
	setModelClass(modelClass) {
		this.modelClass = modelClass;
	}

	getModelSchema() {
		if (!this.modelClass) return null;
		return this.modelClass.getSchema();
	}

	getEmptyInstance() {
		let result = new lx.ModelCollection();
		result.modelClass = this.modelClass;
		return result;
	}

	add(data) {
		let obj;
		if (!data) {
			obj = new this.modelClass;
		} else if (lx.isInstance(data, this.modelClass)) {
			obj = data;
		} else {
			obj = new this.modelClass(data);
		}
		super.add(obj);
		return obj;
	}

	set(i, data) {
		if (!data) {
			super.set(i, new this.modelClass);
		} else if (lx.isInstance(data, this.modelClass)) {
			super.set(i, data);
		} else {
			super.set(i, new this.modelClass(data));
		}
	}

	insert(i, data) {
		if (!data) {
			super.insert(i, new this.modelClass);
		} else if (lx.isInstance(data, this.modelClass)) {
			super.insert(i, data);
		} else {
			super.insert(i, new this.modelClass(data));
		}
	}

	load(list) {
		list.forEach(fields=>this.add(fields));
	}

	reset(list) {
		this.clear();
		if (list) this.load(list);
	}

	removeByData(data) {
		var indexes = this.searchIndexesByData(data);
		indexes.lxForEachRevert(index=>{
			this.removeAt(index)
		});
	}

	searchIndexesByData(data) {
		var indexes = [];
		this.forEach((elem, index)=>{
			for (var i in data) {
				if (!(i in elem) || data[i] != elem[i]) return;
			}
			indexes.push(index);
		});
		return indexes;
	}

	unbind() {
		this.forEach(elem=>elem.unbind());
	}

	static create(config) {
		let schema, list = null;
		if (config.schema) {
			list = config.list || null;
			schema = config.schema;
		} else schema = config;
		class _am_ extends lx.BindableModel {
			static schema() {
				return schema;
			}
		}
		let c = new lx.ModelCollection();
		c.setModelClass(_am_);
		if (list) c.load(list);
		return c;
	}

	static createByData(list, byFirst = true) {
		if (!lx.isArray(list)) {
			console.error('Incorrect data for lx.ModelCollection');
			return;
		}
		if (!list.length) {
			console.error('Empty data for lx.ModelCollection');
			return;
		}

		const conf = {
			schema: {},
			list
		};
		if (byFirst) {
			const first = list[0];
			for (let fName in first) {
				let val = first[fName];
				if (val === true || val === false) {
					conf.schema[fName] = {type: lx.ModelTypeEnum.BOOLEAN};
				} else if (lx.isNumber(val)) {
					conf.schema[fName] = {type: lx.ModelTypeEnum.NUMBER};
				} else {
					conf.schema[fName] = {type: lx.ModelTypeEnum.STRING};
				}
			}
			return this.create(conf);
		}

		for (let i = 0; l < list.length; i++) {
			const item = list[i];
			for (let fName in item) {
				if (fName in conf.schema) continue;
				let val = item[fName];
				if (val === true || val === false) {
					conf.schema[fName] = {type: lx.ModelTypeEnum.BOOLEAN};
				} else if (lx.isNumber(val)) {
					conf.schema[fName] = {type: lx.ModelTypeEnum.NUMBER};
				} else {
					conf.schema[fName] = {type: lx.ModelTypeEnum.STRING};
				}
			}
		}
		return this.create(conf);
	}
}
