// @lx:namespace lx;
class CollectionSelector {
	constructor() {
		this.collection = null;
		this.propertyExistance = new lx.Collection();
		this.propertyValues = {};
		this.filters = new lx.Collection();
		this.strict = true;
	}

	setCollection(collection) {
		this.collection = collection;
		return this;
	}

	reset() {
		this.resetConditions();
		this.resetFilters();
		return this;
	}

	resetConditions() {
		this.propertyExistance.clear();
		this.propertyValues = {};
		return this;
	}

	resetFilters() {
		this.filters.clear();
		return this;
	}

	addFilter(func) {
		this.filters.add(func);
		return this;
	}

	setStrict(strict = true) {
		this.strict = strict;
	}

	ifHasProperty(propertyName) {
		this.propertyExistance.add(propertyName);
		return this;
	}

	ifHasProperties(properties) {
		this.propertyExistance.add(properties);
		return this;
	}

	ifPropertyIs(propertyName, value) {
		this.propertyValues[propertyName] = value;
		return this;
	}

	ifPropertiesAre(map) {
		this.propertyValues.lxMerge(map, true);
		return this;
	}

	getResult() {
		if (!this.collection) return null;

		var result = this.collection.getEmptyInstance();
		this.collection.forEach(elem=>{
			let match = true;

			this.propertyExistance.forEach(function(propName) {
				if (!(propName in elem)) { match = false; this.stop(); }
			});
			if (!match) return;

			for (let propName in this.propertyValues) {
				if (!(propName in elem)) return;
				if (this.strict)
					if (elem[propName] !== this.propertyValues[propName]) return;
				if (elem[propName] != this.propertyValues[propName]) return;
			}

			this.filters.forEach(function(filter) {
				if (!filter(elem)) { match = false; this.stop(); }
			});
			if (!match) return;

			result.add(elem);
		});

		return result;
	}

	getOne() {
		return this.getResult().at(0);
	}
}
