// @lx:namespace lx;
class ElementRequest extends lx.HttpRequest {
	constructor(elemName, methodName, params = {}) {
		super('/lx/elem', {elemName, methodName, params});
	}
}
