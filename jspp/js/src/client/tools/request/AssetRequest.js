// @lx:namespace lx;
class AssetRequest extends lx.Request {
	constructor(url = '', attributes = {}, location = 'head') {
		super();

		this.url = url;
		this.attributes = attributes;
		this.location = location;
	}

	static createByConfig(config) {
		const request = new this();

		request.url = config.path;
		request.attributes = config.attributes || {};
		request.location = config.location || 'head';
		if (config.onLoad) {
			if (lx.isString(config.onLoad))
				config.onLoad = lx.app.functionHelper.createFunction(config.onLoad);
			this.onLoad(config.onLoad);
		}
		if (config.onError) {
			if (lx.isString(config.onError))
				config.onError = lx.app.functionHelper.createFunction(config.onError);
			this.onError(config.onError);
		}

		return request;
	}

	//TODO same in HttpRequest
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
		let tag = null,
			url = this.getFullUrl();
		if (url.match(/\.js$/)) {
			tag = document.createElement('script');
			tag.src = url;
		} else {
			tag  = document.createElement('link');
			tag.rel  = 'stylesheet';
			tag.type = 'text/css';
			tag.href = url;
		}

		for (let name in this.attributes)
			tag.setAttribute(name, this.attributes[name]);

		tag.onload = __onLoad.bind(this);
		tag.onerror = __onError.bind(this);

		switch (this.location) {
			case 'head-top':
				var head = document.getElementsByTagName('head')[0];
				if (head.children.length)
					head.insertBefore(tag, head.children[0]);
				else head.appendChild(tag);
				break;
			case 'head-bottom':
			case 'head':
				var head = document.getElementsByTagName('head')[0];
				head.appendChild(tag);
				break;
			case 'body-top':
				var body = document.getElementsByTagName('body')[0];
				if (body.children.length)
					body.insertBefore(tag, body.children[0]);
				else body.appendChild(tag);
				break;
			case 'body-bottom':
				var body = document.getElementsByTagName('body')[0];
				body.appendChild(tag);
				break;
		}

		return this;
	}
}

function __onLoad() {
	if (lx.isFunction(this._success))
		this._success.call(this);
}

function __onError() {
	if (lx.isFunction(this._error))
		this._error.call(this);
}
