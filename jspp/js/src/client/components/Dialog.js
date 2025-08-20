// @lx:namespace lx;
class Dialog extends lx.AppComponent {
	/**
	 * @param config {Object: {
	 * 	   url {String},
	 *     [method == 'get'] {String},
	 *     [data] {Object},
	 *     [headers] {Dict<String>},
	 *     [success] {Function},
	 *     [waiting] {Function},
	 *     [error] {Function},
	 * }}
	 * @param ignoreEvents {Array<String>}
	 */
	request(config, ignoreEvents = []) {
		return _sendRequest(
			config.method,
			config.url,
			config.data,
			config.headers,
			config.success,
			config.waiting,
			config.error,
			ignoreEvents
		);
	}

	get(config) {
		config.method = 'get';
		this.request(config);
	}

	post(config) {
		config.method = 'post';
		this.request(config);
	}

	/**
	 * Redirect to another page on this web-site
	 */
	move(path) {
		window.location.pathname = path;
	}

	requestParamsToString(params) {
		return _requestParamsToString(params);
	}

	requestParamsFromString(params) {
		return _requestParamsFromString(params);
	}
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * PRIVATE
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

function _sendRequest(method, url, data, headers, hSuccess, hWaiting, hError, ignoreEvents = []) {
	url = url || '';
	headers = headers || {};
	let calculatedUrl = url;
	if (method.toLowerCase() === 'get') {
		let argsString = _requestParamsToString(data);
		if (argsString !== '') calculatedUrl += '?' + argsString;
	}

	if (!_isAjax(url)) {
		_sendCorsRequest(method, calculatedUrl, data, headers, hSuccess, hWaiting, hError);
		return;
	}

	const request = _createRequest();
	if (!request) {
		return /*TODO error*/;
	}

	function successHandlerWrapper(request, handler) {
		let response = request.response;

		// Pass control to the custom handler
		let contentType = request.getResponseHeader('Content-Type') || '';

		// @lx:<mode DEV:
		let responseAndDump = _checkAlert(response), dump;
		response = responseAndDump[0];
		dump = responseAndDump[1];
		// @lx:mode>

		let result = contentType.match(/(?:application|text)\/json/)
			? JSON.parse(response)
			: response;
		_callHandler(handler, [result, request]);

		// @lx:<mode DEV:
		if (dump) lx.alert(dump);
		// @lx:mode>
	}

	function errorHandlerWrapper(request, handler) {
		let response = request.response,
			contentType = request.getResponseHeader('Content-Type') || '';

		// @lx:<mode DEV:
		let responseAndDump = _checkAlert(response), dump;
		response = responseAndDump[0];
		dump = responseAndDump[1];
		// @lx:mode>

		let result = contentType.match(/\/json/) ? JSON.parse(response) : response;

		_callHandler(handler, [result, request]);

		// @lx:<mode DEV:
		if (dump) lx.alert(dump);
		// @lx:mode>

		switch (request.status) {
			case 401:
				if (!ignoreEvents.includes(lx.EVENT_AJAX_REQUEST_UNAUTHORIZED)) lx.app.events.trigger(
					lx.EVENT_AJAX_REQUEST_UNAUTHORIZED,
					[result, request, {method, url, data, headers, success:hSuccess, waiting:hWaiting, error:hError}]
				);
				break;
			case 403:
				if (!ignoreEvents.includes(lx.EVENT_AJAX_REQUEST_FORBIDDEN)) lx.app.events.trigger(
					lx.EVENT_AJAX_REQUEST_FORBIDDEN,
					[result, request, {method, url, data, headers, success:hSuccess, waiting:hWaiting, error:hError}]
				);
				break;
		}
	}

	// Assign a custom handler
	request.onreadystatechange = function() {
		// If the data exchange has not yet been completed
		if (request.readyState !== 4) {
			// Notify the user about the download
			_callHandler(hWaiting);
			return;
		}

		if (request.status === 200) {
			successHandlerWrapper(request, hSuccess);
		} else {
			// Notify the user about the error occurred
			errorHandlerWrapper(request, hError);
		}
	};

	// Initialize connection
	request.open(method, calculatedUrl, true);
	if (!ignoreEvents.includes(lx.EVENT_BEFORE_AJAX_REQUEST))
		lx.app.events.trigger(lx.EVENT_BEFORE_AJAX_REQUEST, [request]);
	for (let name in headers) {
		request.setRequestHeader(name, headers[name]);
	}

	switch (method.toLowerCase()) {
		case 'post':
			//TODO - another method?
			// request.setRequestHeader("Content-Type","application/x-www-form-urlencoded; charset=utf-8");

			request.setRequestHeader('Content-Type','application/json; charset=UTF8');
			request.send(lx.Json.encode(data));
			break;
		case 'get':
			request.send(null);
			break;
	}
}

function _sendCorsRequest(method, url, data, headers, success, waiting, error) {
	let options = {method, mode: 'cors'};
	if (method.toLowerCase() !== 'get' && data && !data.lxEmpty()) {
		options.body = lx.Json.encode(data);
	}

	fetch(url, options)
		.then(res=>res.json()).then(res=>_callHandler(success, res))
		.catch(res=>_callHandler(error, res));
}

/**
 * @return {XMLHttpRequest}
 */
function _createRequest() {
	let request = null;

	if (window.XMLHttpRequest) {
		// Safari, Konqueror
		request = new XMLHttpRequest();
	} else if (window.ActiveXObject) {
		// Internet explorer
		try {
			request = new ActiveXObject("Microsoft.XMLHTTP");
		} catch (CatchException) {
			request = new ActiveXObject("Msxml2.XMLHTTP");
		}
	}

	if (!request) {
		console.error("Can not create XMLHttpRequest");
	}

	return request;
}

/**
 * //TODO - "first=value&arr[]=foo+bar&arr[]=baz"
 */
function _requestParamsToString(args) {
	if (!args) return '';
	if (lx.isString(args)) return args;
	if (!lx.isObject(args)) return '';
	let arr = [],
		result = '';
	for (let i in args) arr.push(i + '=' + args[i]);
	if (arr.len) result = arr.join('&');
	return result;
}

/**
 * //TODO - "first=value&arr[]=foo+bar&arr[]=baz"
 */
function _requestParamsFromString(str) {
	if (!str || str === '') return {};

	let arr = str.split('&'),
		result = {};
	for (let i=0, l=arr.len; i<l; i++) {
		let bf = arr[i].split('=');
		result[bf[0]] = bf[1];
	}
	return result;
}

function _callHandler(handler, args) {
	if (!handler) return;
	if (args !== undefined && !lx.isArray(args)) args = [args];
	if (lx.isArray(handler)) {
		args
			? handler[1].apply(handler[0], args)
			: handler[1].call(handler[0]);
		return;
	}
	args
		? handler.apply(null, args)
		: handler();
}

function _isAjax(url) {
	if (url === '' || url === location.origin) return true;

	let reg = new RegExp('^\\w+?:' + '/' + '/');
	if (!url.match(reg)) return true;

	reg = new RegExp('^\\w+?:' + '/' + '/' + '[^\\/]+');
	return url.match(reg) === location.origin;
}

// @lx:<mode DEV:
function _checkAlert(res) {
	let dump = res.match(/<pre class="lx-alert">[\w\W]*<\/pre>$/);
	if (dump) {
		dump = dump[0];
		res = res.replace(dump, '');
		dump = dump.replace(/^<pre class="lx-alert">/, '');
		dump = dump.replace(/<\/pre>$/, '');
	}
	return [res, dump];
}
// @lx:mode>
