// @lx:namespace lx;
class CssTag {
	/**
	 * @param [config] {Object: {
	 *     [id] {string},
	 *     [before] {HTMLElement},
	 *     [after] {HTMLElement}
	 * }}
	 */
	constructor(config = {}) {
		this._code = '';
		this.domElem = null;
		if (config.id) {
			let elem = document.getElementById(config.id);
			if (elem) this._code = elem.innerHTML;
			else elem = _gen(config);
			config.domElem = elem;
		} else {
			config.domElem = _gen();
		}
		if (config.domElem) this.domElem = config.domElem;
	}

	static exists(id) {
		return !!document.getElementById(id);
	}

	/**
	 * @param {string} code
	 */
	setCss(code) {
		this._code = code;
	}

	/**
	 * @param {string} code 
	 */
	addCss(code) {
		this._code += code;
	}

	commit() {
		this.domElem.innerHTML = this._code;
	}
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * PRIVATE
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

/**
 * @private
 * @param [config] {Object: {
 *     [id] {string},
 *     [before] {HTMLElement},
 *     [after] {HTMLElement}
 * }}
 * @returns {HTMLStyleElement}
 */
function _gen(config = {}) {
	let elem = document.createElement('style');
	if (config.id) elem.setAttribute('id', config.id);
	let head = document.getElementsByTagName('head')[0];
	let before = null;
	if (config.before) before = head.querySelector(config.before);
	else if (config.after) {
		let after = head.querySelector(config.after);
		if (after) before = after.nextSibling;
	}
	if (before === null) head.appendChild(elem);
	else head.insertBefore(elem, before);
	return elem;
}
