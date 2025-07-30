// @lx:namespace lx;
class ServiceRequest extends lx.HttpRequest {
	constructor(action, params = {}) {
		super('/lx_service', {action, params});
	}
}
