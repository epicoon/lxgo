// @lx:namespace lx;
class DomSelector extends lx.AppComponentSettable {

	//TODO deprecated
	getTostsElement() {
		return document.querySelector("[lxid^='" + this.settings['tosts'] + "']");
	}

	//TODO deprecated
	getAlertsElement() {
		return document.querySelector("[lxid^='" + this.settings['alerts'] + "']");
	}

	getElementByAttrs(attrs, parent = null) {
		let selector = '';
		for (let name in attrs)
			selector += "[" + name + "^='" + attrs[name] + "']"
		let elem = parent ? parent.getDomElem() : null;
		if (elem) return elem.querySelector(selector);
		return document.querySelector(selector);
	}

	/**
	 * Returns just-in-time generated lx-element for HTML-element
	 */
	getWidgetById(id, type = null) {
		let el = document.getElementById(id);
		if (!el) return null;
		return el.__lx || _getType(type).rise(el);
	}

	/**
	 * Returns just-in-time generated lx-elements for HTML-elements
	 */
	getWidgetsByName(name, type = null) {
		let els = document.getElementsByName(name),
			c = new lx.Collection();
		for (let i = 0, l = els.length; i < l; i++) {
			let el = els[i];
			c.add(el.__lx || _getType(type).rise(el));
		}
		return c;
	}

	/**
	 * Returns just-in-time generated lx-elements for HTML-elements
	 */
	getWidgetsByClass(className, type = null) {
		let els = document.getElementsByClassName(className),
			c = new lx.Collection();
		for (let i = 0, l = els.length; i < l; i++) {
			let el = els[i];
			c.add(el.__lx || _getType(type).rise(el));
		}
		return c;
	}

	/**
	 * Returns just-in-time generated lx-element for HTML-element
	 */
	getWidgetByClass(className, type = null) {
		let el = document.getElementsByClassName(className)[0];
		if (!el) return null;
		if (el.__lx) return el.__lx;
		return _getType(type).rise(el);
	}
}

function _getType(type) {
	if (type === null || !type.rise) return lx.Box;
	return type;
}
