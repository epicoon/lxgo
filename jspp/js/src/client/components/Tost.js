let __tosts = null;

// @lx:namespace lx;
class Tost extends lx.AppComponent {
	// @lx:const TYPE_MESSAGE = 1;
	// @lx:const TYPE_WARNING = 2;
	// @lx:const TYPE_ERROR = 3;

	init() {
		this.lifetime = 3000;
		this.type = lx.self(TYPE_MESSAGE);
		this.widthLimit = '40%';

		lx.tostMessage = msg => this.message(msg);
		lx.tostWarning = msg => this.warning(msg);
		lx.tostError = msg => this.error(msg);
	}

	setType(type) {
		if (type == lx.self(TYPE_MESSAGE) || type == lx.self(TYPE_WARNING) || type == lx.self(TYPE_ERROR))
			this.type = type;
	}

	message(msg) {
		__print(this, msg, lx.Tost.TYPE_MESSAGE);
	}

	warning(msg) {
		__print(this, msg, lx.Tost.TYPE_WARNING);
	}

	error(msg) {
		__print(this, msg, lx.Tost.TYPE_ERROR);
	}

	align(h, v) {
		var config = lx.isObject(h)
			? h
			: {horizontal: h, vertical: v};
		config.subject = 'lx_tost';
		if (config.direction == undefined) config.direction = lx.VERTICAL;
		if (config.indent == undefined) config.indent = '10px';
		__getTosts().align(config);
	}
}

function __print(self, config, typeArg) {
	if (!config) return;

	let message,
		lifetime,
		type;

	if (lx.isString(config) || lx.isArray(config)) {
		message = config;
		lifetime = self.lifetime;
		type = typeArg || self.type;
	} else if (lx.isObject(config)) {
		message = config.message;
		lifetime = lx.getFirstDefined(config.lifetime, self.lifetime);
		type = config.type || typeArg || self.type;
	} else return;

	if (message && lx.isArray(message))
		message = message.join(' ');

	if (!message
		|| !lx.isString(message)
		|| (type != lx.Tost.TYPE_MESSAGE && type != lx.Tost.TYPE_WARNING && type != lx.Tost.TYPE_ERROR)
	) return;

	const el = new lx.Box({
		parent: __getTosts(),
		depthCluster: lx.DepthClusterMap.CLUSTER_URGENT,
		key: 'lx_tost',
		text: message
	});

	//TODO - use css
	var color, borderColor;
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
	el.roundCorners('8px');
	el.border({color: borderColor});
	el.fill(color);
	el.style('color', 'black');

	el.width(self.widthLimit);
	el.width( el.get('text').width('px') + 20 + 'px' );
	el.height( el.get('text').height('px') + 20 + 'px' );
	el.align(lx.CENTER, lx.MIDDLE);
	if (lifetime) setTimeout(function(el) { el.del(); }, lifetime, el);
}

function __getTosts() {
	if (!__tosts) __initTosts();
	return __tosts;
}

function __initTosts() {
	__tosts = lx.Box.rise(lx.app.domSelector.getTostsElement());
	__tosts.key = 'tosts';
	__tosts.align({
		indent: '10px',
		subject: 'lx_tost',
		vertical: lx.TOP,
		horizontal: lx.LEFT,
		direction: lx.VERTICAL
	});
}
