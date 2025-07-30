// @lx:namespace lx;
class IndentData
{
	/**
	 * @param [config = {}] {Object: {
	 *     {Number|String} [indent = 0],
	 *     {Number|String} [step = 0],
	 *     {Number|String} [stepX = 0],
	 *     {Number|String} [stepY = 0],
	 *     {Number|String} [padding = 0],
	 *     {Number|String} [paddingX = 0],
	 *     {Number|String} [paddingY = 0],
	 *     {Number|String} [paddingLeft = 0],
	 *     {Number|String} [paddingRight = 0],
	 *     {Number|String} [paddingTop = 0],
	 *     {Number|String} [paddingBottom = 0]
	 * }}
	 */
	constructor(config = {}) {
		this.set(config);
	}

	static createOrNull(config) {
		var result = new this(config);
		if (result.lxEmpty()) return null;
		return result;
	}

	// @lx:<context SERVER:
	pack() {
		var indents = this.get();
		return indents.stepX
			+ ',' + indents.stepY
			+ ',' + indents.paddingLeft
			+ ',' + indents.paddingRight
			+ ',' + indents.paddingTop
			+ ',' + indents.paddingBottom
		;
	}
	// @lx:context>

	// @lx:<context CLIENT:
	static unpackOrNull(info) {
		var config = info.split(',');
		var result = new this(config);
		result.stepX = config[0];
		result.stepY = config[1];
		result.paddingLeft = config[2];
		result.paddingRight = config[3];
		result.paddingTop = config[4];
		result.paddingBottom = config[5];
		return result;
	}
	// @lx:context>

	set(config) {
		if (config.indent       !== undefined) this.indent       = config.indent;
		if (config.step         !== undefined) this.step         = config.step;
		if (config.stepX        !== undefined) this.stepX        = config.stepX;
		if (config.stepY        !== undefined) this.stepY        = config.stepY;
		if (config.padding       !== undefined) this.padding       = config.padding;
		if (config.paddingX      !== undefined) this.paddingX      = config.paddingX;
		if (config.paddingY      !== undefined) this.paddingY      = config.paddingY;
		if (config.paddingLeft   !== undefined) this.paddingLeft   = config.paddingLeft;
		if (config.paddingRight  !== undefined) this.paddingRight  = config.paddingRight;
		if (config.paddingTop    !== undefined) this.paddingTop    = config.paddingTop;
		if (config.paddingBottom !== undefined) this.paddingBottom = config.paddingBottom;
	}

	get(elem = null, format='px') {
		function part(param, dir) {
			if (elem) return elem.geomPart(param, format, dir);
			return param;
		}

		return {
			stepX: part(lx.getFirstDefined(this.stepX, this.step, this.indent, 0), lx.HORIZONTAL),
			stepY: part(lx.getFirstDefined(this.stepY, this.step, this.indent, 0), lx.VERTICAL),
			paddingLeft: part(lx.getFirstDefined(this.paddingLeft,  this.paddingX, this.padding, this.indent, 0), lx.HORIZONTAL),
			paddingRight: part(lx.getFirstDefined(this.paddingRight, this.paddingX, this.padding, this.indent, 0), lx.HORIZONTAL),
			paddingTop: part(lx.getFirstDefined(this.paddingTop,    this.paddingY, this.padding, this.indent, 0), lx.VERTICAL),
			paddingBottom: part(lx.getFirstDefined(this.paddingBottom, this.paddingY, this.padding, this.indent, 0), lx.VERTICAL)
		};
	}

	static getZero() {
		return {
			stepX: 0,
			stepY: 0,
			paddingLeft: 0,
			paddingRight: 0,
			paddingTop: 0,
			paddingBottom: 0
		}
	}
}
