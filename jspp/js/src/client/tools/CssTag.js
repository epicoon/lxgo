// @lx:namespace lx;
class CssTag {
	constructor(config) {
		this._code = '';
		this.domElem = null;
		if (config.id) {
			let elem = document.getElementById(config.id);
			if (elem) {
				this._code = elem.innerHTML;
			} else {
				elem = document.createElement('style');
				elem.setAttribute('id', config.id);
				let head = document.getElementsByTagName('head')[0];
				let before = null;
				if (config.before) before = head.querySelector(config.before);
				else if (config.after) {
					let after = head.querySelector(config.after);
					if (after) before = after.nextSibling;
				}
				if (before === null) head.appendChild(elem);
				else head.insertBefore(elem, before);
			}
			config.domElem = elem;
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
