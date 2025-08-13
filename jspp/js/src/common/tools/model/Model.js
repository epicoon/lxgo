// @lx:namespace lx;
class Model extends lx.Object {
	constructor(data) {
		super();

		this.__init(data);
		this.init();
	}

	static create(config) {
		let schema, fields = {};
		if (config.schema) {
			fields = config.fields || null;
			schema = config.schema;
		} else schema = config;
		const selfClass = this;
		class _am_ extends selfClass {
			static schema() {
				return schema;
			}
		}
		return new _am_(fields);
	}

	static createForForm(form) {
		let schema = {};
		function runElem(elem, schema) {
			if (elem._field) {
				if (elem._isMatrix) {
					let mSchema = {},
						handlers = elem.getEventHandlers('renderMatrixItem'),
						temp = new lx.Box({parent:null});
					temp.begin();
					handlers.forEach(h => h({box: temp}));
					temp.end();
					temp.getChildren().forEach(child => runElem(child, mSchema));
					schema[elem._field] = {type: lx.ModelCollection, schema: mSchema};
				} else {
					schema[elem._field] = {};
				}
			}
			if (elem.getChildren)
				elem.getChildren().forEach(child => runElem(child, schema));
		}
		runElem(form, schema);
		const model = this.create(schema);
		model.bind(form);
		return model;
	}

	/** @abstract */
	init() {
		// pass
	}

	getPk() {
		let pkName = lx.self(__schema.getPkName());
		if (!pkName) return undefined;
		return this[pkName];
	}

	/**
	 * Init according to schema
	 */
	setFields(data) {
		if (!data || !lx.isObject(data)) return;

		let schema = lx.self(__schema);
		if (!schema) return;

		for (let key in data)
			if (schema.hasField(key))
				this.setField(key, data[key]);
	}

	/**
	 * Returns selected (or all) schema fields
	 */
	getFields(map = null) {
		let schema = lx.self(__schema);
		if (map === null) map = schema.getFieldNames();

		let result = {};

		map.forEach(key=>{
			if (schema.hasField(key)) result[key] = this[key];
		});

		return result;
	}

	setField(name, value) {
		let field = lx.self(__schema.getField(name));
		if (field.ref) {
			var code = 'this.' + field.ref + '=val;',
				f = new Function('val', code);
			f.call(this, value);
		} else {
			this[name] = value;
		}
	}

	getField(name) {
		let field = lx.self(__schema.getField(name));
		if (field.ref) {
			let code = 'return this.' + field.ref + ';',
				f = new Function(code);
			return f.call(this);
		}

		return this[name];
	}

	/**
	 * Reset selected (or all) fields to default values
	 */
	resetFields(map = null) {
		if (!lx.self(__schema)) return;
		if (map === null) map = lx.self(__schema).getFieldNames();
		map.forEach(name=>this.resetField(name));
	}

	/**
	 * Reset field to default value
	 */
	resetField(name) {
		let definition = lx.self(__schema).getField(name);
		if (!definition) return;

		let type = lx.isObject(definition) ? definition.type : definition,
			dflt = lx.isObject(definition) ? definition.default : undefined;

		let val;
		switch (type) {
			case 'int':
				val = lx.getFirstDefined(dflt, lx.self(defaultIntegerFieldValue()));
				break;
			case 'string':
				val = lx.getFirstDefined(dflt, lx.self(defaultStringFieldValue()));
				break;
			case 'bool':
				val = lx.getFirstDefined(dflt, lx.self(defaultBooleanFieldValue()));
				break;
			case lx.ModelCollection:
				val = lx.ModelCollection.create({
					schema: (lx.isObject(definition) ? (definition.schema || {}) : {}),
					list: dflt
				});
				break;
			default:
				val = lx.getFirstDefined(dflt, lx.self(defaultUntypedFieldValue()));
		}
		this.setField(name, val);
	}

	delegateSchema(obj) {
		let model = this;
		this.getSchema().getFieldNames(true).forEach(field => {
			Object.defineProperty(obj, field, {
				set: function (val) {
					model[field] = val;
				},
				get: function () {
					return model[field];
				}
			});
		});
	}

	getSchema() {
		if (!lx.self(__schema)) return null;
		return lx.self(__schema);
	}

	static getSchema() {
		if (!this.__schema) return null;
		return this.__schema;
	}

	static initSchema(config) {
		this.__schema = new lx.ModelSchema(config);
	}

	static getFieldNames(all = false) {
		if (!this.__schema) return [];
		return this.__schema.getFieldNames(all);
	}

	static getFieldTypes() {
		if (!this.__schema) return [];
		return this.__schema.getFieldTypes();
	}

	static defaultIntegerFieldValue() { return 0; }
	static defaultStringFieldValue()  { return ''; }
	static defaultBooleanFieldValue() { return false; }
	static defaultUntypedFieldValue() { return 0; }

	/**
	 * @abstract
	 * @returns {Object}
	 */
	static schema() {
		return {};
	}
	
	/**
	 * Magic method will be called after class defenition
	 */
	static __afterDefinition() {
		this.__schema = null;
		this.initSchema(this.schema());
		super.__afterDefinition();
	}

	__init(data = {}) {
		var schema = lx.self(__schema);
		if (!schema) return;
		schema.eachField((field, name)=>{
			if (field.ref) return;
			if (data[name] !== undefined) this[name] = data[name];
			else this.resetField(name);
		});
	}
}
