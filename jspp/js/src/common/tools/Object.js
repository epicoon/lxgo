// @lx:namespace lx;
class Object {
	constructor() {
		lx(STATIC).behaviorMap.forEach(beh=>beh.prototype.behaviorConstructor.call(this));
	}

	get behaviorMap() {
		return new lx.BehaviorMap(this);
	}

	static get behaviorMap() {
		return new lx.BehaviorMap(this);
	}

	addBehavior(behavior, config=null) {
		if (this.behaviorMap.has(behavior) || lx(STATIC).behaviorMap.has(behavior)) return;
		behavior.injectInto(this, config);
	}

	hasBehavior(behavior) {
		return this.behaviorMap.has(behavior) || lx(STATIC).behaviorMap.has(behavior);
	}

	static delegateMethods(map) {
		let ownMethods = lx.globalContext.Object.getOwnPropertyNames(this.prototype);
		ownMethods.lxPushUnique('constructor');
		for (let fieldName in map) {
			let stuff = map[fieldName];
			let methods = lx.globalContext.Object.getOwnPropertyNames(stuff.prototype);
			let delegatedMethods = methods.lxDiff(ownMethods);

			for (let i=0, l=delegatedMethods.length; i<l; i++) {
				let methodName = delegatedMethods[i];
				this.prototype[methodName] = function (...args) {
					return this[fieldName][methodName](args);
				}
			}
		}
	}
	
	static addBehavior(behavior, config=null) {
		if (this.behaviorMap.has(behavior)) return;
		behavior.injectInto(this, config);
	}

	/** @abstract */
	static afterDefinition() {
		// pass
	}

	/**
	 * Magic method will be called after class defenition
	 */
	static __afterDefinition() {
		this.afterDefinition();
		if (this.lxHasMethod('__injectBehaviors')) this.__injectBehaviors();
	}
}
