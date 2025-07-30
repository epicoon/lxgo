// @lx:namespace lx;
class EventDispatcher {
	constructor() {
		this.listeners = {};
	}

	subscribe(eventName, callback) {
		if (lx.isObject(eventName)) {
			for (var i in eventName) this.subscribe(i, eventName[i]);
			return;
		}

		if (!(eventName in this.listeners)) this.listeners[eventName] = [];
		this.listeners[eventName].push(callback);
	}

	trigger(eventName, args = []) {
		if (eventName in this.listeners) {
			if (args === null || args === undefined || !lx.isArray(args)) args = [args];
			for (var i=0, l=this.listeners[eventName].len; i<l; i++)
				lx.app.functionHelper.callFunction(this.listeners[eventName][i], args);
		}
	}
}
