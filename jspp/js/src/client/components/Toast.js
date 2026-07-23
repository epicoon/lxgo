let _tosts = null;

/**
 * @settings:
 * - message - CSS-class for messages box
 * - warning - CSS-class for warning box
 * - error - CSS-class for error box
 * - lifetime - toast display duration in milliseconds
 * - defaultType - 'message', 'warning' or 'error'
 * - width - toast width
 */
// @lx:namespace lx;
class Toast extends lx.AppComponentSettable {
	// @lx:const TYPE_MESSAGE = 'message';
	// @lx:const TYPE_WARNING = 'warning';
	// @lx:const TYPE_ERROR = 'error';

	init() {
		lx.toastMessage = msg => this.message(msg);
		lx.toastWarning = msg => this.warning(msg);
		lx.toastError = msg => this.error(msg);
	}

	lifetime() {
		if ('lifetime' in this.settings)
			return this.settings.lifetime;
		return 3000;
	}

	defaultType() {
		if ('defaultType' in this.settings)
			return this.settings.defaultType;
		return lx.Toast.TYPE_MESSAGE;
	}

	widthLimit() {
		if ('width' in this.settings)
			return this.settings.width;
		return '40%';
	}

	message(msg) {
		_print(this, msg, lx.Toast.TYPE_MESSAGE);
	}

	warning(msg) {
		_print(this, msg, lx.Toast.TYPE_WARNING);
	}

	error(msg) {
		_print(this, msg, lx.Toast.TYPE_ERROR);
	}

	align(h, v) {
		let config = lx.isObject(h)
			? h
			: {horizontal: h, vertical: v};
		config.subject = 'lx_tost';
		if (config.direction == undefined) config.direction = lx.VERTICAL;
		if (config.indent == undefined) config.indent = '10px';
		_getTosts().align(config);
	}
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * PRIVATE
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

function _print(self, config, typeArg) {
	if (!config) return;

	let message,
		lifetime,
		type;

	if (lx.isString(config) || lx.isArray(config)) {
		message = config;
		lifetime = self.lifetime();
		type = typeArg || self.defaultType();
	} else if (lx.isObject(config)) {
		message = config.message;
		lifetime = lx.getFirstDefined(config.lifetime, self.lifetime());
		type = config.type || typeArg || self.defaultType();
	} else return;

	if (message && lx.isArray(message))
		message = message.join(' ');

	if (!message
		|| !lx.isString(message)
		|| (type != lx.Toast.TYPE_MESSAGE && type != lx.Toast.TYPE_WARNING && type != lx.Toast.TYPE_ERROR)
	) return;

	const el = new lx.Box({
		parent: _getTosts(),
		depthCluster: lx.DepthClusterMap.CLUSTER_URGENT,
		key: 'lx_tost',
		text: message
	});
	_decorate(self, el, type)

	if (lifetime)
		el._timeoutID = setTimeout(function(el) { el.del(); }, lifetime, el);
	el.style('cursor', 'pointer');
	el.click(()=>{
		if (el._timeoutID) clearTimeout(el._timeoutID);
		el.del();
	});
}

function _decorate(self, toast, type) {
	if (type in self.settings) {
		toast.addClass(self.settings[type]);
		return;
	}

	let color, borderColor;
	switch (type) {
		case lx.Toast.TYPE_MESSAGE:
			color = 'lightgreen';
			borderColor = 'green';
			break;
		case lx.Toast.TYPE_WARNING:
			color = 'orange';
			borderColor = 'lightcoral';
			break;
		case lx.Toast.TYPE_ERROR:
			color = 'lightcoral';
			borderColor = 'red';
			break;
	}
	toast.roundCorners('8px');
	toast.border({color: borderColor});
	toast.fill(color);
	toast.style('color', 'black');

	toast.width(self.widthLimit());
	toast.width( toast.get('text').width('px') + 20 + 'px' );
	toast.height( toast.get('text').height('px') + 20 + 'px' );
	toast.align(lx.CENTER, lx.MIDDLE);
}

function _getTosts() {
	if (!_tosts) _initTosts();
	return _tosts;
}

function _initTosts() {
	let wrapper = lx.app.domSelector.getElementByAttrs({lxid: 'lx-tosts'});
	if (!wrapper) {
		wrapper = document.createElement('div');
		wrapper.setAttribute('lxid', 'lx-tosts');
		Object.assign(wrapper.style, {
			position: 'absolute',
			top: '0',
			left: '0',
			width: '100%',
			height: '100%',
		});
		const body = document.body;
		if (body.firstChild) body.insertBefore(wrapper, body.firstChild);
		else body.appendChild(wrapper);
	}

	_tosts = lx.Box.rise(wrapper);
	_tosts.key = 'tosts';
	_tosts.align({
		indent: '10px',
		subject: 'lx_tost',
		vertical: lx.TOP,
		horizontal: lx.LEFT,
		direction: lx.VERTICAL
	});
}
