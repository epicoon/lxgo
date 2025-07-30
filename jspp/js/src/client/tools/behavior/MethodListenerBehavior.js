// @lx:namespace lx;
class MethodListenerBehavior extends lx.Behavior {
	static injectInto(supportedEssence, config=null) {
		super.injectInto(supportedEssence);

		supportedEssence.behaviorMap.set(behKey, 'methodEvents', {});
	}

	static beforeMethod(funcName, func, duplicate = false) {
		__setMethodEvent.call(this, funcName, func, 'before', duplicate);
	}

	static afterMethod(funcName, func, duplicate = false) {
		__setMethodEvent.call(this, funcName, func, 'after', duplicate);
	}

	beforeMethod(funcName, func, duplicate = false) {
		if (!this.behaviorMap.get(behKey, 'methodEvents'))
			this.behaviorMap.set(behKey, 'methodEvents', {});
		__setMethodEvent.call(this, funcName, func, 'before', duplicate);
	}

	afterMethod(funcName, func, duplicate = false) {
		if (!this.behaviorMap.get(behKey, 'methodEvents'))
			this.behaviorMap.set(behKey, 'methodEvents', {});
		__setMethodEvent.call(this, funcName, func, 'after', duplicate);
	}
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * PRIVATE
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

const behKey = lx.MethodListenerBehavior.lxFullName();

function __setMethodEvent(funcName, func, category, duplicate) {
	var methodEvents = this.behaviorMap.get(behKey, 'methodEvents');
	if (!duplicate
		&& methodEvents[funcName]
		&& methodEvents[funcName][category]
		&& methodEvents[funcName][category].includes(func)
	) return;

	var obj = lx.isFunction(this) ? this.prototype : this,
		temp = Object.getPrototypeOf(obj),
		finded = false;
	while (temp && !finded) {
		var names = Object.getOwnPropertyNames(temp);
		if (names.includes(funcName) && lx.isFunction(obj[funcName])) finded = true;
		temp = Object.getPrototypeOf(temp);
	}
	var names = Object.getOwnPropertyNames(obj);
	var selfFail = (!names.includes(funcName) || !lx.isFunction(obj[funcName]));
	var prototypeFail = true;
	if (obj.prototype) {
		var names = Object.getOwnPropertyNames(obj.prototype);
		prototypeFail = (!names.includes(funcName) || !lx.isFunction(obj[funcName]));
	}
	if (!finded && selfFail && prototypeFail) return;

	__wrapMethod.call(obj, funcName);

	if (!methodEvents[funcName]) {
		methodEvents[funcName] = {
			before: [],
			after: []
		};
	}
	methodEvents[funcName][category].push(func);
}

function __wrapMethod(methodName) {
	var info = lx(STATIC).behaviorMap.get(behKey, 'methodEvents'),
		thisInfo = this.behaviorMap.get(behKey, 'methodEvents');

	if ((info && info[methodName]) || (thisInfo && thisInfo[methodName])) return;

	var funcClosure = this[methodName];

	Object.defineProperty(this, methodName, {
		enumerable: false,
		value: function() {
			var info = lx(STATIC).behaviorMap.get(behKey, 'methodEvents'),
				thisInfo = this.behaviorMap.get(behKey, 'methodEvents');

			if (thisInfo && thisInfo[methodName]) {
				if (thisInfo[methodName].before)
					for (var i=0, l=thisInfo[methodName].before.len; i<l; i++)
						if (thisInfo[methodName].before[i].apply(this, arguments) === false) return;
			}
			if (info && info[methodName]) {
				if (info[methodName].before)
					for (var i=0, l=info[methodName].before.len; i<l; i++)
						if (info[methodName].before[i].apply(this, arguments) === false) return;
			}
			let result = funcClosure.apply(this, arguments);
			if (thisInfo && thisInfo[methodName]) {
				if (thisInfo[methodName].after)
					for (var i=0, l=thisInfo[methodName].after.len; i<l; i++)
						if (thisInfo[methodName].after[i].apply(this, arguments) === false) return;
			}
			if (info && info[methodName]) {
				if (info[methodName].after)
					for (var i=0, l=info[methodName].after.len; i<l; i++)
						if (info[methodName].after[i].apply(this, arguments) === false) return;
			}
			return result;
		}
	});
}
