/**
 * @positioningStrategy lx.PositioningStrategy
 */
// @lx:namespace lx;
class PositioningStrategy {
	constructor(owner) {
		this.owner = owner;
		this.autoActualize = true;
	}

	/**
	 * @param {Object} [config = {}]
	 */
	init(config = {}) {
		this.applyConfig(config);
		this.actualizeOnInit();
	}

	/** @abstract */
	applyConfig(config) {
		// pass
	}

	/** @abstract */
	actualizeOnInit() {
		// pass
	}

	// @lx:<context SERVER:
	pack() {
		var str = this.lxFullClassName();
		if (this.needJsActualize) str += ';na:1';
		var indents = this.packIndents();
		if (indents) str += ';i:' + indents;
		return str + this.packProcess();
	}

	packIndents() {
		if (!this.indents) return false;
		return this.indents.pack();
	}

	packProcess() {
		return '';
	}
	// @lx:context>

	// @lx:<context CLIENT:
	unpack(info) {
		var config = {};
		for (var i = 0, l = info.length; i < l; i++) {
			var temp = info[i].split(':');
			config[temp[0]] = temp[1];
		}
		this.unpackProcess(config);
		if (config.i) this.unpackIndents(config.i);
		if (config.na) this.owner.__na = true;
	}

	unpackIndents(info) {
		var indents = lx.IndentData.unpackOrNull(info);
		if (indents) this.indents = indents;
	}

	unpackProcess(config) {}
	// @lx:context>

	actualize(info) {
		if (this.autoActualize) this.actualizeProcess(info);
	}

	/**
	 * Reset strategy data when clearing container
	 */
	clear() {}

	/**
	 * When removing an element from a container
	 */
	onElemDel() {}

	/**
	 * When cleaning the container
	 */
	onClearOwner() {}

	/**
	 * To position a new element added to a container
	 */
	allocate(elem, config) {
		var geom = this.geomFromConfig(config);

		if (geom.lxEmpty()) {
			// elem.trigger('resize');
			return;
		}

		var abs = false;
		if (geom.l !== undefined || geom.t !== undefined || geom.r !== undefined || geom.b !== undefined) {			
			elem.addClass('lx-abspos');
			abs = true;
		}

		for (var i in geom) {
			if (geom[i] && lx.isString(geom[i]) && geom[i].includes('/')) {
				var parts = geom[i].split('/');
				geom[i] = Math.round(100 * parts[0] / parts[1]);
			}
		}

		if (geom.w === undefined && abs) {
			geom.l = geom.l || 0;
			geom.r = geom.r || 0;
		}

		if (geom.h === undefined && abs) {
			geom.t = geom.t || 0;
			geom.b = geom.b || 0;
		}

		if (geom.lxEmpty()) return;
		if ( geom.l !== undefined ) this.setParam(elem, lx.LEFT, geom.l);
		if ( geom.r !== undefined ) this.setParam(elem, lx.RIGHT, geom.r);
		if ( geom.w !== undefined ) this.setParam(elem, lx.WIDTH, geom.w);

		if ( geom.t !== undefined ) this.setParam(elem, lx.TOP, geom.t);
		if ( geom.b !== undefined ) this.setParam(elem, lx.BOTTOM, geom.b);
		if ( geom.h !== undefined ) this.setParam(elem, lx.HEIGHT, geom.h);
		// elem.trigger('resize');
	}

	/**
	 * Updating the positions of elements in a container
	 */
	actualizeProcess(info) {}

	/**
	 * Request to change the positional parameter for a specific element
	 * Should return a Boolean value - true and change the parameter, or false and not change the parameter
	 */
	tryReposition(elem, param, val) {
		this.setParam(elem, param, val);
		return true;		
	}

	/**
	 * If the child element has automatic sizing, such as Text, this describes how to react to changing dimensions
	 */
	reactForAutoresize(elem) {}

	/**
	 * Setting a geometric parameter to an element
	 */
	setParam(elem, param, val) {
		if (val === null) {
			elem.domElem.style(lx.Geom.geomName(param), null);
			return;
		}

		if (lx.isNumber(val)) val += '%';
		elem.setGeomPriority(param);
		elem.domElem.style(lx.Geom.geomName(param), val);
	}

	/**
	 * You can set settings for indents
	 */
	setIndents(config={}) {
		var indents = lx.IndentData.createOrNull(config);
		if (indents) this.indents = indents;
		else delete this.indents;
		return this;
	}

	/**
	 * If no padding settings are specified, a full settings object filled with zeros will be returned
	 */
	getIndents(format = null) {
		if (!this.indents) {
			if (this.owner.getIndents) return this.owner.getIndents();
			return lx.IndentData.getZero();
		}
		if (format === null) return this.indents.get();
		return this.indents.get(this.owner, format);
	}

	geomFromConfig(config) {
		return lx(STATIC).geomFromConfig(config);
	}

	/**
	 * Extracts positional parameters from the configuration
	 */
	static geomFromConfig(config) {
		if (lx.isArray(config)) return this.geomFromConfig({
			left: config[0],
			right: config[1],
			width: config[2],
			height: config[3],
			top: config[4],
			bottom: config[5]
		});

		var geom = {};

		if (config.geom === true) config.geom = [
			0, 0, undefined, undefined, 0, 0
		];

		if ( config.margin ) config.geom = [
			config.margin,
			config.margin,
			undefined,
			undefined,
			config.margin,
			config.margin
		];
		if ( config.geom ) {
			geom.l = config.geom[0];
			geom.t = config.geom[1];
			geom.w = config.geom[2];
			geom.h = config.geom[3];
			geom.r = config.geom[4];
			geom.b = config.geom[5];
		}
		if (config.coords) {
			geom.l = config.coords[0];
			geom.t = config.coords[1];
			geom.r = config.coords[2];
			geom.b = config.coords[3];
		}
		if (config.size) {
			geom.w = config.size[0];
			geom.h = config.size[1];
		}
		if ( config.right  !== undefined ) geom.r = config.right;
		if ( config.bottom !== undefined ) geom.b = config.bottom;

		if ( config.width  !== undefined ) geom.w = config.width;
		if ( config.height !== undefined ) geom.h = config.height;

		if ( config.left   !== undefined ) geom.l = config.left;
		if ( config.top    !== undefined ) geom.t = config.top;

		for (var i in geom) if (geom[i] === null) delete geom[i];
		return geom;
	}
}
