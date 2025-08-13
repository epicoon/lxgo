/**
 * @positioningStrategy lx.MapPositioningStrategy
 */
// @lx:namespace lx;
class MapPositioningStrategy extends lx.PositioningStrategy {
	// @lx:const FORMAT_PERCENT = 1;
	// @lx:const FORMAT_PX = 2;
	// @lx:const FORMAT_FREE = 3;

	/**
	 * @param [config = {}] {
	 *     String&Enum('%', 'px')
	 *     |Object: {
	 *         format {String&Enum('%', 'px')}
	 *     }
	 *     |Object: {
	 *         format {Number&Enum(
	 *             lx.MapPositioningStrategy.FORMAT_PERCENT,
	 *             lx.MapPositioningStrategy.FORMAT_PX,
	 *             lx.MapPositioningStrategy.FORMAT_FREE
	 *         )}
	 *     }
	 * }
	 */
	applyConfig(config = {}) {
		if (!lx.isObject(config))
			config = {format: config};
		if (config.format == '%') config.format = lx.self(FORMAT_PERCENT);
		else if (config.format == 'px') config.format = lx.self(FORMAT_PX);

		this.format = config.format || lx.self(FORMAT_FREE);
		this.defaultFormat = config.format || lx.self(FORMAT_PERCENT);
		this.formats = {};

		this.owner.getChildren(c=>c.addClass('lx-abspos'));
	}

	// @lx:<context SERVER:
	packProcess() {
		return ';df:' + this.defaultFormat
			+ ';cf:' + this.format;
	}
	// @lx:context>

	// @lx:<context CLIENT:
	unpackProcess(config) {
		this.format = +config.cf || lx.self(FORMAT_FREE);
		this.defaultFormat = +config.df || lx.self(FORMAT_PERCENT);
	}
	// @lx:context>

	/**
	 * You can set a custom format for the geometric parameter for a specific strategy
	 */
	setFormat(param, format=null) {
		if (format === null) {
			if (this.formats) delete this.formats[param];
			return;
		}
		this.formats[param] = format;
	}

	/**
	 * You can get a format for any geometric parameter
	 */
	getFormat(param) {
		if (param in this.formats) return this.formats[param];
		return this.format;
	}

	allocate(elem, config) {
		var geom = this.geomFromConfig(config);

		if (geom.lxEmpty()) geom = {l:0, t:0, r:0, b:0};
		elem.addClass('lx-abspos');

		for (var i in geom) {
			if (geom[i] && lx.isString(geom[i]) && geom[i].includes('/')) {
				var parts = geom[i].split('/');
				geom[i] = Math.round(100 * parts[0] / parts[1]) + '%';
			}
		}

		if (geom.w === undefined) {
			geom.l = geom.l || 0;
			geom.r = geom.r || 0;
		}

		if (geom.h === undefined) {
			geom.t = geom.t || 0;
			geom.b = geom.b || 0;
		}

		if ( geom.r !== undefined ) this.setParam(elem, lx.RIGHT, geom.r);
		if ( geom.w !== undefined ) this.setParam(elem, lx.WIDTH, geom.w);
		if ( geom.l !== undefined ) this.setParam(elem, lx.LEFT, geom.l);

		if ( geom.b !== undefined ) this.setParam(elem, lx.BOTTOM, geom.b);
		if ( geom.h !== undefined ) this.setParam(elem, lx.HEIGHT, geom.h);
		if ( geom.t !== undefined ) this.setParam(elem, lx.TOP, geom.t);
		// elem.trigger('resize');
	}

	setParam(elem, param, val) {
		var splittedVal = __splitParam(this, val);

		var finalFormat = this.getFormat(param);
		if (finalFormat == lx.self(FORMAT_FREE))
			finalFormat = splittedVal[1];
		if (finalFormat == splittedVal[1]) {
			elem.domElem.style([lx.Geom.geomName(param)], splittedVal[0] + __getFormatText(this, splittedVal[1]));
			return;
		}

		let container = elem.parent.getContainer();
		if (container.getDomElem() && elem.getDomElem()) {
			__setParam(this, container, elem, param, val, finalFormat);
			return;
		}

		let self = this;
		elem.displayOnce(function() {
			__setParam(self, elem.parent.getContainer(), this, param, val, finalFormat);
		});
	}
}

function __setParam(self, container, elem, param, val, format) {
	let dir = lx.Geom.directionByGeom(param);
	let formatText = __getFormatText(self, format);
	let calcVal = Math.round(container.geomPart(val, formatText, dir) * 100) * 0.01;
	elem.domElem.style([lx.Geom.geomName(param)], calcVal + formatText);
}

/**
 * Type casting PositioningStrategy.FORMAT_PERCENT => '%'
 */
function __getFormatText(self, format) {
	if (format == lx.MapPositioningStrategy.FORMAT_FREE)
		format = (self.format == lx.MapPositioningStrategy.FORMAT_FREE)
			? self.defaultFormat
			: self.format;
	return __formatToText(format);
}

function __formatToText(format) {
	if (format == lx.MapPositioningStrategy.FORMAT_PERCENT) return '%';
	if (format == lx.MapPositioningStrategy.FORMAT_PX) return 'px';
	return '';
}

/**
 * Split params '50%' as [50, PositioningStrategy.FORMAT_PERCENT]
 */
function __splitParam(self, val) {
	if (lx.isNumber(val)) return [val, self.defaultFormat];
	var num = parseFloat(val),
		f = val.split(num)[1];
	return [num, f=='%' ? lx.MapPositioningStrategy.FORMAT_PERCENT : lx.MapPositioningStrategy.FORMAT_PX];
}
