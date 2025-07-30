// @lx:namespace lx;
class Request {
	constructor() {
		this._success = null;
		this._wait = null;
		this._error = null;
	}

	then(func) {
		this._success = func;
		return this;
	}

	catch(func) {
		this._error = func;
		return this;
	}

	send() {
		// abstract
	}

	onLoad(callback) {
		this._success = callback;
	}

	onWait(callback) {
		this._wait = callback;
	}

	onError(callback) {
		this._error = callback;
	}

	getLoadCallback() {
		return this._success;
	}

	getErrorCallback() {
		return this._error;
	}
}
