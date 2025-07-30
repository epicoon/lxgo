/**
 * Manager for initializing hotkeys
 * You need to inherit your manager from this class and declare methods with the 'on_' prefix in it - these are keypress handlers
 * The name of the handler methods ends with a code or a keystroke symbol, for example:
 *  - 'on_13' - handler for enter pressed
 *  - 'on_a' - handler for letter 'a' pressed
 * Another way to create a handler is to use the 'key_' prefix, then the method name can end arbitrarily (within the method naming rules),
 * and for this arbitrary name there must be a match specified in the KeypressManager::keys() method - a symbol, code, or an array of symbols and codes, for example:
 *	- 'key_test' - method name with key 'test'
 *	- keys() { return {test: [13, 'a']}; } - the method will be triggered when pressing 'enter' or the 'a' key
 */
// @lx:namespace lx;
class KeypressManager {
	// @lx:const ENTER = 13;

	constructor() {
		this.context = {};
		this.init();
	}

	init() {}

	setContext(context) {
		this.context = context;
	}

	addContext(info) {
		if (info.elem) {
			if (!this.context.elems)
				this.context.elems = [];
			this.context.elems.push(info.elem);
		}
	}

	dropContext(info) {
		if (info.elem) {
			if (!this.context.elems) return;
			this.context.elems.lxRemove(info.elem);
		}
	}

	run() {
		let funcs = this.lxGetAllProperties(),
			upHandlers = {},
			downHandlers = {};
		for (let i=0, l=funcs.len; i<l; i++) {
			let funcName = funcs[i];
			switch (true) {
				case !!funcName.match(/^onUp_/):
					__handleOn(this, funcName, upHandlers);
					break;
				case !!funcName.match(/^onDown_/):
				case !!funcName.match(/^on_/):
					__handleOn(this, funcName, downHandlers);
					break;
				case !!funcName.match(/^keyUp_/):
					__handleKey(this, funcName, upHandlers);
					break;
				case !!funcName.match(/^keyDown_/):
				case !!funcName.match(/^key_/):
					__handleKey(this, funcName, downHandlers);
					break;
			}
		}

		for (let key in upHandlers)
			for (let i=0, l=upHandlers[key].length; i<l; i++)
				lx.app.keyboard.onKeyup(key, upHandlers[key][i], this.context);
		for (let key in downHandlers)
			for (let i=0, l=downHandlers[key].length; i<l; i++)
				lx.app.keyboard.onKeydown(key, downHandlers[key][i], this.context);
	}

	keys() {
		return {};
	}
}

function __handleOn(self, funcName, handlers) {
	let key = funcName.replace(/on(Up|Down)?_/, '');
	__pushHandler(self, key, self[funcName], handlers);
}

function __handleKey(self, funcName, handlers) {
	let key = funcName.replace(/key(Up|Down)?_/, ''),
		hotkeys = self.keys()[key];

	if (!hotkeys) return;
	if (!lx.isArray(hotkeys)) hotkeys = [hotkeys];

	hotkeys.forEach(a=>{
		if (lx.isString(a) && a.match(/\+/)) {
			let arr = a.split('+');
			arr.forEach((item, i)=> {
				if (item != '' && !lx.isNumber(item)) arr[i] = item.toUpperCase().charCodeAt(0);
			});
			let main = arr.pop(),
				f = function(e) {
					if (arr.len == 1 && arr[0] == '') {
						if (lx.app.keyboard.pressedCount() == 1) self[funcName]();
						return;
					}

					for (let i=0, l=arr.len; i<l; i++)
						if (!lx.app.keyboard.keyPressed(arr[i])) return;
					self[funcName](e);
				};
			__pushHandler(self, main, f, handlers);
			return;
		}

		__pushHandler(self, a, self[funcName], handlers);
	});
}

function __pushHandler(self, key, handler, handlers) {
	if (!(key in handlers)) handlers[key] = [];
	handlers[key].push([self, handler]);
}
