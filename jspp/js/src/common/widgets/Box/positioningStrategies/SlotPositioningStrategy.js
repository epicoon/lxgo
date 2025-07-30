/**
 * The number of elements is determined by the specified number of rows and columns, or by the total number of elements
 * The dimensions of all elements are the same
 * There is a constant ratio of the width and height of the elements
 * Element alignment options are supported (for each axis):
 *     - centered (steps between elements work, margins from edges are calculated)
 *     - pressed to the edges (margins from the edges work, steps between elements are calculated)
 *     - evenly distributed (margins from edges and steps between elements are calculated to be the same)
 * Adding new elements recalculates the sizes of existing ones without affecting the size of the box itself
 */

/**
 * @positioningStrategy lx.SlotPositioningStrategy
 */
// @lx:namespace lx;
class SlotPositioningStrategy extends lx.PositioningStrategy {
	constructor(owner, config) {
		super(owner);
		if (config) this.init(config);
	}

	/**
	 * @param [config = {}] {Object: {
	 *     {Number} [k = 1],
	 *     {Number} [cols = 1],
	 *     {Number} [rows],
	 *     {Number} [count],
	 *     {Number&Enum(
	 *         lx.LEFT,
	 *         lx.CENTER,
	 *         lx.RIGHT,
	 *         lx.TOP,
	 *         lx.MIDDLE,
	 *         lx.BOTTOM
	 *     )} [align],
	 *     {Class(lx.Rect)} [type = lx.Box],
	 *     #merge(lx.IndentData::constructor::config)
	 * }}
	 */
	applyConfig(config = {}) {
		this.k = config.k || 1;
		this.cols = config.cols || 1;
		if (config.align !== undefined) this.align = config.align;

		this.setIndents(config);

		var count;
		if (config.count !== undefined) {
			count = config.count;
		} else if (config.rows) {
			count = this.cols * config.rows;
		} else return;

		var type = config.type || lx.Box;
		type.construct(count, { key:'s', parent:this.owner });

		// @lx:<context SERVER:
		this.needJsActualize = 1;
		// @lx:context>
		// @lx:<context CLIENT:
		this.actualize();
		// @lx:context>
	}

	setK(k) {
		this.k = k;
		// @lx:<context CLIENT:
		this.actualizeProcess();
		// @lx:context>
	}
	
	// @lx:<context SERVER:
	packProcess() {
		var str = ';k:' + this.k + ';c:' + this.cols;
		if (this.align) str += ';a:' + this.align;
		return str;
	}
	// @lx:context>

	// @lx:<context CLIENT:
	unpackProcess(config) {
		this.k = +config.k;
		this.cols = +config.c;
		if (config.a) this.align = +config.a;
	}
	// @lx:context>

	/**
	 * To position a new element added to a container
	 */
	allocate(elem, config) {
		elem.addClass('lx-abspos');
		// @lx:<context SERVER:
		super.allocate(elem, config);
		// @lx:context>
		// @lx:<context CLIENT:
		this.actualize();
		// @lx:context>
	}

	/**
	 * Request to change the positional parameter for a specific element
	 * Should return a Boolean value - true and change the parameter, or false and not change the parameter
	 */
	tryReposition(elem, param, val) {
		return false;
	}

	// @lx:<context CLIENT:
	/**
	 * Updating the positions of elements in a container
	 */
	actualizeProcess() {
		var sz = this.owner.size('px'),
			rows = this.rows(),
			amt = [this.cols, rows],
			k = this.k,
			r = this.getIndents('px'),
			align = this.align || null,
			step = [r.stepX, r.stepY],
			marg = [[r.paddingLeft, r.paddingRight], [r.paddingTop, r.paddingBottom]],
			axe = (k*this.cols/rows > sz[0]/sz[1]) ? 0 : 1,
			axe2 = +!axe,
			cellSz = [0, 0];

		cellSz[axe] = (sz[axe] - marg[axe][0] - marg[axe][1] - step[axe] * (amt[axe] - 1)) / amt[axe];
		if (axe == 1) cellSz[axe2] = k * cellSz[axe];
		else cellSz[axe2] = cellSz[axe] / k;

		switch (align) {
			case null:
				step[axe2] = (sz[axe2] - cellSz[axe2] * amt[axe2]) / (amt[axe2] + 1);
				marg[axe2][0] = step[axe2];
				break;
			case lx.CENTER:
			case lx.MIDDLE:
				marg[axe2][0] = (sz[axe2] - cellSz[axe2] * amt[axe2] - step[axe2] * (amt[axe2] - 1)) * 0.5;
				break;
			case lx.JUSTIFY:
				step[axe2] = (sz[axe2] - cellSz[axe2] * amt[axe2] - marg[axe2][0] - marg[axe2][1]) / (amt[axe2] - 1);
				break;
			case lx.LEFT:
			case lx.TOP:
				marg[axe2][0] = marg[+!axe2][0];
				break;
			case lx.RIGHT:
			case lx.BOTTOM:
				marg[axe2][0] = sz[axe2] - (cellSz[axe2] + step[axe2]) * amt[axe2];
				break;
		}

		this.relocate(marg[0][0], marg[1][0], cellSz, step);
	}

	relocate(x0, y0, sz, step) {
		var slots = this.owner.getChildren(),
			x = x0,
			y = y0;

		for (var i=0, rows=this.rows(); i<rows; i++) {
			for (var j=0; j<this.cols; j++) {
				var slot = slots.next();
				if (!slot) return;
				this.setParam(slot, lx.LEFT, x + 'px');
				this.setParam(slot, lx.TOP, y + 'px');
				this.setParam(slot, lx.WIDTH, sz[0] + 'px');
				this.setParam(slot, lx.HEIGHT, sz[1] + 'px');
				slot.checkResize();
				x += sz[0] + step[0];
			}
			x = x0;
			y += sz[1] + step[1];
		}
	}

	rows() {
		return Math.floor((this.owner.getChildren().len + this.cols - 1) / this.cols);
	}
	// @lx:context>
}
