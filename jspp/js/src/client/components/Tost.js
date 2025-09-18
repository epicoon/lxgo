let _tosts = null;

/**
 * @settings:
 * - message - CSS-class for messages box
 * - warning - CSS-class for warning box
 * - error - CSS-class for error box
 * - lifetime - tost display duration in milliseconds
 * - defaultType - 'message', 'warning' or 'error'
 * - width - tost width
 */
// @lx:namespace lx;
class Tost extends lx.AppComponentSettable {
	// @lx:const TYPE_MESSAGE = 'message';
	// @lx:const TYPE_WARNING = 'warning';
	// @lx:const TYPE_ERROR = 'error';

	init() {
		lx.tostMessage = msg => this.message(msg);
		lx.tostWarning = msg => this.warning(msg);
		lx.tostError = msg => this.error(msg);
	}

	lifetime() {
		if ('lifetime' in this.settings)
			return this.settings.lifetime;
		return 3000;
	}

	defaultType() {
		if ('defaultType' in this.settings)
			return this.settings.defaultType;
		return lx.Tost.TYPE_MESSAGE;
	}

	widthLimit() {
		if ('width' in this.settings)
			return this.settings.width;
		return '40%';
	}

	message(msg) {
		_print(this, msg, lx.Tost.TYPE_MESSAGE);
	}

	warning(msg) {
		_print(this, msg, lx.Tost.TYPE_WARNING);
	}

	error(msg) {
		_print(this, msg, lx.Tost.TYPE_ERROR);
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
		|| (type != lx.Tost.TYPE_MESSAGE && type != lx.Tost.TYPE_WARNING && type != lx.Tost.TYPE_ERROR)
	) return;

	const el = new lx.Box({
		parent: _getTosts(),
		depthCluster: lx.DepthClusterMap.CLUSTER_URGENT,
		key: 'lx_tost',
		text: message
	});
	_decorate(self, el, type)

	if (lifetime) setTimeout(function(el) { el.del(); }, lifetime, el)
	else {
		el.style('cursor', 'pointer');
		el.click(()=>el.del());
	}
}

function _decorate(self, tost, type) {
	if (type in self.settings) {
		tost.addClass(self.settings[type]);
		return;
	}

	let color, borderColor;
	switch (type) {
		case lx.Tost.TYPE_MESSAGE:
			color = 'lightgreen';
			borderColor = 'green';
			break;
		case lx.Tost.TYPE_WARNING:
			color = 'orange';
			borderColor = 'lightcoral';
			break;
		case lx.Tost.TYPE_ERROR:
			color = 'lightcoral';
			borderColor = 'red';
			break;
	}
	tost.roundCorners('8px');
	tost.border({color: borderColor});
	tost.fill(color);
	tost.style('color', 'black');

	tost.width(self.widthLimit());
	tost.width( tost.get('text').width('px') + 20 + 'px' );
	tost.height( tost.get('text').height('px') + 20 + 'px' );
	tost.align(lx.CENTER, lx.MIDDLE);
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
