// @lx:namespace lx;
class HttpRequest extends lx.Request {
	constructor(url = '', params = {}) {
		super();

		this.method = 'POST';
		this.host = null;
		this.url = url;

		this.headers = {};
		this.params = params;
	}

	setMethod(method) {
		this.method = method.toLowerCase();
		return this;
	}

	checkMethod(method) {
		return this.method == method.toLowerCase();
	}

	setHeaders(headers) {
		this.headers = headers;
		return this;
	}

	setHeader(name, value) {
		this.headers[name] = value;
		return this;
	}

	setParams(params) {
		this.params = params;
		return this;
	}

	addParams(params) {
		for (let name in params)
			this.params[name] = params[name];
		return this;
	}

	//TODO same in AssetRequest
	getFullUrl() {
		let host = this.host || lx.app.getProxy(),
			url,
			reg = new RegExp('^\\w+?:' + '/' + '/');
		if (host && lx.isString(this.url) && !this.url.match(reg)) {
			url = host;
			if (lx.isString(this.url)) {
				let slashes = 0;
				if (url[url.length - 1] == '/') slashes++;
				if (this.url[0] == '/') slashes++;
				if (slashes == 0) url += '/';
				else if (slashes == 2) url = url.slice(0, -1);
				url += this.url;
			}
		} else url = this.url;
		return url;
	}

	send() {
		lx.app.dialog.request({
			method: this.method,
			url: this.getFullUrl(),
			headers: this.headers,
			data: this.params,
			success: [this, __onSuccess],
			waiting: [this, __onWait],
			error: [this, __onError]
		});

		return this;
	}
}

function __onSuccess(result, request) {
	if (lx.isFunction(this._success))
		this._success.call(this, result, request);
}

function __onWait(request) {
	if (lx.isFunction(this._wait))
		this._wait.call(this, request);
}

function __onError(request) {
	if (lx.isFunction(this._error))
		this._error.call(this, request);
}
