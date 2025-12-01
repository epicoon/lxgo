// @lx:module lx.Input;

// @lx:use lx.BasicCssContext;

/**
 * @widget lx.Input
 * @content-disallowed
 * 
 * CSS classes:
 * - lx-Input
 */
// @lx:namespace lx;
class Input extends lx.Rect {	
	static initCss(css) {
		css.useExtender(lx.BasicCssContext);
		css.inheritClass('lx-Input', 'Input', {
		}, {
			focus: 'border: 1px solid ' + css.preset.checkedMainColor,
			disabled: 'opacity: 0.5'
		});
	}

	static getStaticTag() {
		return 'input';
	}

	/**
	 * @widget-init
	 * 
	 * @param [config] {Object: {
	 *     #merge(lx.Rect::constructor::config),
	 *     [placeholder] {String},
	 *     [value] {String}
	 * }}
	 */
	render(config) {
		super.render(config);

		this.addClass('lx-Input');
		if (config.placeholder) this.setAttribute('placeholder', config.placeholder);
		if (config.value != '') this.value(config.value);
	}

	// @lx:<context CLIENT:
	clientRender(config) {
		super.clientRender(config);
		this.on('focus', lx.self(setEntry) );
		this.on('blur', lx.self(unsetEntry) );
	}

	static setEntry(event) {
		lx.entryElement = this;
		this._oldValue = this.value();
	}

	static unsetEntry(event) {
		lx.entryElement = null;
	}

	valueChanged() {
		return this._oldValue != this.value();
	}

	oldValue() {
		if (this._oldValue === undefined) return null;
		return this._oldValue;
	}
	// @lx:context>

	value(val) {
		// @lx:<context SERVER:
		if (val == undefined) return this.getAttribute('value');
		this.setAttribute('value', val);
		// @lx:context>

		// @lx:<context CLIENT:
		if (val == undefined) return this.domElem.param('value');
		this.domElem.param('value', val);
		// @lx:context>

		return this;
	}

	placeholder(val) {
		if (val === undefined) return this.getAttribute('placeholder');
		this.setAttribute('placeholder');
	}

	focus(func) {
		// @lx:<context CLIENT:
		if (func === undefined) {
			let elem = this.getDomElem();
			if (elem) elem.focus();
			return this;
		}
		// @lx:context>
		this.on('focus', func);
		return this;
	}

	blur(func) {
		// @lx:<context CLIENT:
		if (func === undefined) {
			let elem = this.getDomElem();
			if (elem) elem.blur();
			return this;
		}
		// @lx:context>
		this.on('blur', func);
		return this;
	}
}
