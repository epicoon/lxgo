let _pressedKeys = false,
	_pressedKey = 0,
	_pressedChar = null,
	_pressedCount = 0,
	_keydownHandlers = [],
	_keyupHandlers = [];

// @lx:namespace lx;
class Keyboard extends lx.AppComponent {
	pressedCount() {
		return _pressedCount;
	}

	resetKeys() {
		for (let i=0; i<256; i++) _pressedKeys[i] = false;
		_pressedKey = 0;
		_pressedChar = null;
	}

	onKeydown(key, func, context = {}) {
		if (lx.isObject(key)) {
			for (let k in key) this.onKeydown(k, key[k]);
			return;
		}
		_on(_keydownHandlers, key, func, context);
	}

	offKeydown(key, func, context = {}) {
		_off(_keydownHandlers, key, func, context);
	}

	onKeyup(key, func, context = {}) {
		if (lx.isObject(key)) {
			for (let k in key) this.onKeyup(k, key[k]);
			return;
		}
		_on(_keyupHandlers, key, func, context);
	}

	offKeyup(key, func, context = {}) {
		_off(_keyupHandlers, key, func, context);
	}

	keyPressed(key) {
		if (lx.isString(key)) key = key.charCodeAt(0);
		if (_pressedKeys) return _pressedKeys[key];
		return false;
	}

	shiftPressed() { return this.keyPressed(16); }

	ctrlPressed() { return this.keyPressed(17); }

	altPressed() { return this.keyPressed(18); }

	setWatchForKeypress(bool) {
		if (!bool) {
			if (_pressedKeys === false) return;
			lx.off('keydown', watchForKeydown);
			lx.off('keyup', watchForKeyup);
			_pressedKeys = false;
			return;
		}

		function getPressedChar(event) {
			//TODO check event.type == keypress
			if (event.key) return event.key;

			if (event.which == null) { // IE
				if (event.keyCode < 32) return null; // special character
				return String.fromCharCode(event.keyCode)
			}
			if (event.which != 0 && event.charCode != 0) { // all except IE
				if (event.which < 32) return null; // special character
				return String.fromCharCode(event.which); // rest
			}
			return null; // special character
		}

		function watchForKeydown(event) {
			let e = event || window.event,
				code = (e.charCode) ? e.charCode: e.keyCode;
			if (!_pressedKeys[code]) _pressedCount++;
			_pressedKeys[code] = true;

			//TODO - it won't always be convenient to do this with 13 and 27
			// Disable identification signal when text is entered
			if ((code != 13) && (code != 27) && lx.entryElement) {
				return;
			}

			_pressedKey = +code;
			_pressedChar = getPressedChar(e);

			if (_keydownHandlers['k_' + code])
				for (let i in _keydownHandlers['k_' + code]) {
					let pare = _keydownHandlers['k_' + code][i];
					if (!_checkContext(pare.context)) continue;
					let f = pare.handler;
					if (lx.isFunction(f)) f(e);
					else if (lx.isArray(f)) f[1].call(f[0], e);
				}

			if (_keydownHandlers['c_' + _pressedChar])
				for (let i in _keydownHandlers['c_' + _pressedChar]) {
					let pare = _keydownHandlers['c_' + _pressedChar][i];
					if (!_checkContext(pare.context)) continue;
					let f = pare.handler;
					if (lx.isFunction(f)) f(e);
					else if (lx.isArray(f)) f[1].call(f[0], e);
				}
		}

		function watchForKeyup(event) {
			let e = event || window.event,
				code = (e.charCode) ? e.charCode: e.keyCode;
			_pressedKeys[code] = false;
			_pressedCount--;

			// Disable identification signal when text is entered
			if (lx.entryElement) return;

			if (_keyupHandlers['k_' + code])
				for (let i in _keyupHandlers['k_' + code]) {
					let pare = _keyupHandlers['k_' + code][i];
					if (!_checkContext(pare.context)) continue;
					let f = pare.handler;
					if (lx.isFunction(f)) f(e);
					else if (lx.isArray(f)) f[1].call(f[0], e);
				}

			if (_keyupHandlers['c_' + _pressedChar])
				for (let i in _keyupHandlers['c_' + _pressedChar]) {
					let pare = _keyupHandlers['c_' + _pressedChar][i];
					if (!_checkContext(pare.context)) continue;
					let f = pare.handler;
					if (lx.isFunction(f)) f(e);
					else if (lx.isArray(f)) f[1].call(f[0], e);
				}
		}

		_pressedKeys = [];
		for (let i=0; i<256; i++) _pressedKeys.push(false);
		lx.on('keydown', watchForKeydown);
		lx.on('keyup', watchForKeyup);
	}

	// @lx:<mode DEV:
	status() {
		console.log('Key down handlers:');
		console.log(_keydownHandlers);
		console.log('Key up handlers:');
		console.log(_keyupHandlers);
	}
	// @lx:mode>
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * PRIVATE
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

function _on(handlers, key, func, context) {
	key = lx.isNumber(key) ? 'k_' + key : 'c_' + key;
	if (!handlers[key])
		handlers[key] = [];
	handlers[key].push({handler:func, context});
}

function _off(handlers, key, func, context) {
	if (key === null && func === null) {
		for (let key in handlers) {
			let keyHandlers = handlers[key],
				tempHandlers = [];
			for (let i=0, l=keyHandlers.len; i<l; i++) {
				let keyHandler = keyHandlers[i],
					_context = keyHandler.context;
				if (context.elem && _context.elems) {
					_context.elems.lxRemove(context.elem);
					if (!_context.elems.length) continue;
				}
				tempHandlers.push(keyHandler);
			}
			if (tempHandlers.length) handlers[key] = tempHandlers;
			else delete handlers[key];
		}
		return;
	}

	key = lx.isNumber(key) ? 'k_' + key : 'c_' + key;
	if (!handlers[key]) return;

	let index = -1;
	for (let i=0, l=handlers[key].len; i<l; i++) {
		let handler = handlers[key][i].handler;

		if ((lx.isFunction(handler) && handler === func)
			|| (lx.isArray(handler) && lx.isFunction(handler[1]) && handler[1] === func)
		) {
			index = i;
			break;
		}
	}

	if (index == -1) return;
	delete handlers[key][index].handler;
	delete handlers[key][index].context;
	handlers[key].splice(index, 1);
}

function _checkContext(context) {
	if (context.elems) {
		for (let i in context.elems)
			//TODO remove plugin-dependency
			if (lx.app.pluginManager && context.elems[i] === lx.app.pluginManager.getFocusedPlugin())
				return true;
		return false;
	}

	return true;
}
