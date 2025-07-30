// @lx:namespace lx;
class Behavior {
	behaviorConstructor() {}

	static injectInto(supportedEssence, config=null) {
		var names = Object.getOwnPropertyNames(this.prototype);
		var funcNames = [];
		for (var i=0, l=names.length; i<l; i++) {
			if (names[i] == 'constructor' || names[i] == 'behaviorConstructor') continue;
			if (lx.isFunction(this.prototype[names[i]])) funcNames.push(names[i]);
		}

		if (lx.isFunction(supportedEssence)) {
			var staticNames = Object.getOwnPropertyNames(this);
			var staticFuncNames = [];
			for (var i=0, l=staticNames.length; i<l; i++) {
				if (staticNames[i] == 'inject' || staticNames[i] == 'overridedMethods') continue;
				if (lx.isFunction(this[staticNames[i]])) staticFuncNames.push(staticNames[i]);
			}
			for (var i=0, l=staticFuncNames.length; i<l; i++) {
				if (supportedEssence[staticFuncNames[i]] === undefined || this.overridedMethods().includes(staticFuncNames[i]))
					supportedEssence[staticFuncNames[i]] = this[staticFuncNames[i]];
			}
			for (var i=0, l=funcNames.length; i<l; i++) {
				if (supportedEssence.prototype[funcNames[i]] === undefined || this.overridedMethods().includes(staticFuncNames[i]))
					supportedEssence.prototype[funcNames[i]] = this.prototype[funcNames[i]];
			}
		} else if (supportedEssence instanceof lx.Object) {
			for (var i=0, l=funcNames.length; i<l; i++) {
				if (supportedEssence[funcNames[i]] === undefined || this.overridedMethods().includes(staticFuncNames[i]))
					supportedEssence[funcNames[i]] = this.prototype[funcNames[i]];
			}			
		}

		supportedEssence.behaviorMap.register(this);
	}

	static overridedMethods() {
		return [];
	}
}
