// For active GET-requests - autosyncronization with URL
class AjaxGet {
	constructor(plugin) {
		this.plugin = plugin;

		this.activeUrl = {};
		this.urlDelimiter = '||';
	}

	registerActiveUrl(key, respondent, handlers, useServer=true) {
		this.activeUrl[key] = {
			state: false,
			useServer,
			respondent,
			handlers
		};

		if (this.plugin.isMainContext()) _checkUrlInAction(this, key);
	}

	request(key, data={}) {
		if (!(key in this.activeUrl)) return;
		_requestProcess(this, key, data);
		if (this.plugin.isMainContext()) _renewLocationHash(this);
	}
}

function _requestProcess(self, key, data={}) {
	var activeUrl = self.activeUrl[key];
	activeUrl.state = data;

	if (activeUrl.useServer) {
		var request = new lx.PluginRequest(self.plugin, activeUrl.respondent, data);
		if (lx.isFunction(activeUrl.handlers))
			request.onLoad(activeUrl.handlers);
		else if (lx.isObject(activeUrl.handlers)) {
			if (activeUrl.handlers.onLoad)
				request.onLoad(activeUrl.handlers.onLoad);
			if (activeUrl.handlers.onError)
				request.onError(activeUrl.handlers.onError);
		}
		request.send();
	} else {
		activeUrl.handlers(data);
	}
}

function _renewLocationHash(self) {
	var arr = [];

	for (let key in self.activeUrl) {
		let activeUrl = self.activeUrl[key];
		if (activeUrl.state === false) continue;
		let params = lx.app.dialog.requestParamsToString(activeUrl.state);
		let fullUrl = params == '' ? key : key + '?' + params;
		arr.push(fullUrl);
	}

	var hash = arr.join(self.urlDelimiter);
	if (hash != '') window.location.hash = hash;
}

function _checkUrlInAction(self, key) {
	var hash = window.location.hash;
	if (hash == '') return;

	hash = hash.substr(1);
	var fullUrls = hash.split(self.urlDelimiter);

	for (var i=0, l=fullUrls.len; i<l; i++) {
		var fullUrl = fullUrls[i],
			urlInfo = fullUrl.split('?'),
			currentUrl = urlInfo[0];

		if (currentUrl != key) continue;

		var data = lx.app.dialog.requestParamsFromString(urlInfo[1]);
		_requestProcess(self, key, data);
	}
}
