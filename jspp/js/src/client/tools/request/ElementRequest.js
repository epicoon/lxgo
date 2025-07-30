// @lx:namespace lx;
class ElementRequest extends lx.HttpRequest {
	constructor(elemName, methodName, params = {}) {
		super('/lx_elem', {elemName, methodName, params});
	}
}
