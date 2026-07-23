// @lx:namespace lx;
class ElementRequest extends lx.HttpRequest {
	/**
	 * @param {String} elem
	 * @param {String} path
	 * @param {Object} params
	 */
	constructor(elem, path, params = {}) {
		super('/lx/elem', {elem, path, params});
	}
}
