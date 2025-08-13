/**
 * Works with `new`
 * For subclasses use method `init()` as constructor
 */
// @lx:namespace lx;
class Singleton {
	constructor(...args) {
		if (lx.self(getInstance) === undefined || lx.self(getInstance().constructor) !== this.constructor) {
			initInstance(this.constructor);
			lx.self(setInstance(this));
			this.init.apply(this, args);
		} else return lx.self(getInstance());
	}

	/** @abstract */
	init() {
		// pass
	}
}

function initInstance(ctx) {
	// Instance is in closure
	(function() {
		let instance = null;
		return function() {
			this.getInstance = function() { return instance; };
			this.setInstance = function(val) { instance = val; delete this.setInstance; };
		};		
	})().call(ctx);
}
