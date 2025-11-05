/**
 * @positioningStrategy lx.GridPositioningStrategy
 */
// @lx:namespace lx;
class GridPositioningStrategy extends lx.PositioningStrategy {
	// @lx:const TYPE_SIMPLE = 1;
	// @lx:const TYPE_PROPORTIONAL = 2;
	// @lx:const TYPE_STREAM = 3;
	// @lx:const TYPE_ADAPTIVE = 4;
	// @lx:const TYPE_FIT = 5;
	// @lx:const DEAULT_COLUMNS_AMOUNT = 12;
	// @lx:const COLUMN_MIN_WIDTH = '40px';
	// @lx:const ROW_MIN_HEIGHT = '40px';

	/**
	 * @param [config = {}] {Object: {
	 *     {Number&Enum(
	 *         lx.GridPositioningStrategy.TYPE_SIMPLE,
	 *         lx.GridPositioningStrategy.TYPE_PROPORTIONAL,
	 *         lx.GridPositioningStrategy.TYPE_STREAM,
	 *         lx.GridPositioningStrategy.TYPE_ADAPTIVE,
	 *         lx.GridPositioningStrategy.TYPE_FIT
	 *     )} [type = lx.GridPositioningStrategy.TYPE_SIMPLE],
	 *     {Number} [cols = lx.GridPositioningStrategy.DEAULT_COLUMNS_AMOUNT],
	 *     {String} [minHeight],
	 *     {String} [minWidth],
	 *     {String} [maxHeight],
	 *     {String} [maxWidth],
	 *     {String} [height],
	 *     {String} [width],
	 *     #merge(lx.IndentData::constructor::config)
	 * }}
	 */
	applyConfig(config={}) {
		//TODO direction?
		this.type = config.type || lx.self(TYPE_SIMPLE);
		if (this.type !== lx.self(TYPE_ADAPTIVE) && this.type !== lx.self(TYPE_FIT))
			this.cols = config.cols || lx.self(DEAULT_COLUMNS_AMOUNT);
		if (this.type == lx.self(TYPE_SIMPLE) || this.type == lx.self(TYPE_PROPORTIONAL))
			this.map = new lx.BitMap(this.cols);

		if (config.minHeight !== undefined) this.minHeight = config.minHeight;
		if (config.minWidth !== undefined) this.minWidth = config.minWidth;
		if (config.maxHeight !== undefined) this.maxHeight = config.maxHeight;
		if (config.maxWidth !== undefined) this.maxWidth = config.maxWidth;
		if (config.height !== undefined) {
			this.minHeight = config.height;
			this.maxHeight = config.height;
		}
		if (config.width !== undefined) {
			this.minWidth = config.width;
			this.maxWidth = config.width;
		}

		this.owner.addClass('lxps-grid-v');
		switch (this.type) {
			case lx.self(TYPE_ADAPTIVE):
				this.owner.style(
					'grid-template-columns',
					'repeat(auto-fill,minmax('
						+ lx.getFirstDefined(this.minWidth, lx.self(COLUMN_MIN_WIDTH))
						+ ',1fr))'
				);
				break;
			case lx.self(TYPE_FIT):
				this.owner.style(
					'grid-template-columns',
					'repeat(auto-fit,minmax('
						+ lx.getFirstDefined(this.minWidth, lx.self(COLUMN_MIN_WIDTH))
						+ ',1fr))'
				);
				break;
			default:
				this.owner.style('grid-template-columns', 'repeat(' + this.cols + ',1fr)');
		}

		if (this.type != lx.self(TYPE_PROPORTIONAL)) {
			this.owner.height('auto');
		}
		this.setIndents(config);
	}

	setCols(cols) {
		if (this.type == lx.self(TYPE_ADAPTIVE) || this.type == lx.self(TYPE_FIT) || this.cols === cols)
			return;
		this.cols = cols;
		if (this.map) this.map.setX(cols);
		this.owner.style('grid-template-columns', 'repeat(' + this.cols + ',1fr)');
	}

	getCols() {
		if (this.cols === undefined) return null;
		return this.cols;
	}

	getRows() {
		if (this.type == lx.self(TYPE_SIMPLE) || this.type == lx.self(TYPE_PROPORTIONAL))
			return this.map.y;
		return null;
	}
	
	// @lx:<context SERVER:
	packProcess() {
		var str = ';t:' + this.type;
		if (this.cols !== undefined)
			str += ';c:' + this.cols;
		if (this.minHeight)
			str += ';mh:' + this.minHeight;
		if (this.minWidth)
			str += ';mw:' + this.minWidth;
		if (this.maxHeight)
			str += ';mxh:' + this.maxHeight;
		if (this.maxWidth)
			str += ';mxw:' + this.maxWidth;
		if (this.map !== undefined)
			str += ';m:' + this.map.toString();
		return str;
	}
	// @lx:context>

	// @lx:<context CLIENT:
	unpackProcess(config) {
		this.type = +config.t;
		if (config.c !== undefined) this.cols = +config.c;
		if (config.mh) this.minHeight = config.mh;
		if (config.mw) this.minWidth = config.mw;
		if (config.mxh) this.maxHeight = config.mxh;
		if (config.mxw) this.maxWidth = config.mxw;
		if (config.m !== undefined)
			this.map = config.m == ''
				? new lx.BitMap(this.cols)
				: lx.BitMap.createFromString(config.m);
	}
	// @lx:context>

	onClearOwner() {
		if (this.map) this.map.fullReset();
	}

	/**
	 * To position a new element added to a container
	 */
	allocate(elem, config) {
		elem.style('position', 'relative');
		elem.style('min-height', lx.getFirstDefined(config.minHeight, this.minHeight, lx.self(ROW_MIN_HEIGHT)));
		elem.style('min-width', lx.getFirstDefined(config.minWidth, this.minWidth, lx.self(COLUMN_MIN_WIDTH)));
		var maxHeight = lx.getFirstDefined(config.maxHeight, this.maxHeight),
			maxWidth = lx.getFirstDefined(config.maxWidth, this.maxWidth);
		if (maxHeight) elem.style('max-height', maxHeight);
		if (maxWidth) elem.style('max-width', maxWidth);

		__allocate(this, elem, config);
	}

	setIndents(config) {
		super.setIndents(config);
		var indents = this.getIndents();

		//TODO - same for different strategies
		if (indents.paddingTop) this.owner.style('padding-top', indents.paddingTop);
		if (indents.paddingBottom) this.owner.style('padding-bottom', indents.paddingBottom);
		if (indents.paddingLeft) this.owner.style('padding-left', indents.paddingLeft);
		if (indents.paddingRight) this.owner.style('padding-right', indents.paddingRight);

		if (indents.stepY) this.owner.style('grid-row-gap', indents.stepY);
		if (indents.stepX) this.owner.style('grid-column-gap', indents.stepX);
	}
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * PRIVATE
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

function __allocate(self, elem, config) {
	if (self.type == lx.GridPositioningStrategy.TYPE_SIMPLE || self.type == lx.GridPositioningStrategy.TYPE_PROPORTIONAL) {
		var geom = self.geomFromConfig(config);
		if (!geom.w) geom.w = 1;
		if (!geom.h) geom.h = 1;
		var params = __prepareInGridParams(self, geom),
			rows = self.map.y;

		self.owner.style('grid-template-rows', 'repeat(' + rows + ',1fr)');
		elem.style('grid-area', params.join('/'));
	}
}

function __prepareInGridParams(self, geom) {
	if (geom.l !== undefined && geom.t === undefined) geom.t = 0;
	if (geom.t !== undefined && geom.l === undefined) geom.l = 0;
	if (geom.w > self.map.x) geom.w = self.map.x;
	var needSetBit = geom.l === undefined;
	if (!needSetBit) {
		if (geom.t+geom.h > self.map.y) self.map.setY(geom.t+geom.h);
		return [
			geom.t+1,
			geom.l+1,
			geom.t+geom.h+1,
			geom.l+geom.w+1
		];
	}

	var crds = self.map.findSpace(geom.w, geom.h);;
	while (crds === false) {
		self.map.addY();
		crds = self.map.findSpace(geom.w, geom.h);
	}

	var l = crds[0], t = crds[1];
	self.map.setSpace([l, t, geom.w, geom.h]);
	return [
		t+1,
		l+1,
		t+geom.h+1,
		l+geom.w+1
	];
}
