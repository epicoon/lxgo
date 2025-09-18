/**
 * @positioningStrategy lx.AlignPositioningStrategy
 */
// @lx:namespace lx;
class AlignPositioningStrategy extends lx.PositioningStrategy {
	/**
	 * @param [config = {}] {Object: {
	 *     {Number&Enum(
	 *         lx.HORIZONTAL,
	 *         lx.VERTICAL
	 *     )} [direction = lx.HORIZONTAL],
	 *     {Number&Enum(
	 *         lx.LEFT,
	 *         lx.CENTER,
	 *         lx.RIGHT
	 *     )} [horizontal = lx.CENTER],
	 *     {Number&Enum(
	 *         lx.TOP,
	 *         lx.MIDDLE,
	 *         lx.BOTTOM
	 *     )} [vertical = lx.MIDDLE],
	 *     #merge(lx.IndentData::constructor::config)
	 * }}
	 */
	applyConfig(config = {}) {
		this.direction = config.direction || lx.HORIZONTAL;
		this.horizontal = config.horizontal || lx.CENTER;
		this.vertical = config.vertical || lx.MIDDLE;
		var indents = this.setIndents(config);

		this.owner.style('display', 'flex');
		this.owner.style('flex-direction', this.direction == lx.HORIZONTAL ? 'row' : 'column');
		var vAlign, hAlign;
		switch (this.horizontal) {
			case lx.LEFT:
				this.owner.style('text-align', 'left');
				hAlign = 'flex-start';
				break;
			case lx.CENTER:
				this.owner.style('text-align', 'center');
				hAlign = 'center';
				break;
			case lx.JUSTIFY:
				this.owner.style('text-align', 'justify');
				hAlign = 'center';
				break;
			case lx.RIGHT:
				this.owner.style('text-align', 'right');
				hAlign = 'flex-end';
				break;
		}
		switch (this.vertical) {
			case lx.TOP: vAlign = 'flex-start'; break;
			case lx.MIDDLE: vAlign = 'center'; break;
			case lx.BOTTOM: vAlign = 'flex-end'; break;
		}
		this.owner.style('align-items', this.direction == lx.HORIZONTAL ? vAlign : hAlign);
		this.owner.style('justify-content', this.direction == lx.HORIZONTAL ? hAlign : vAlign);

		this.owner.getChildren().forEach((el)=>{
			el.removeClass('lx-abspos');
		});
		this.actualizeIndents(indents);
	}

	// @lx:<context SERVER:
	packProcess() {
		return ';d:' + this.direction
			+ ';h:' + this.horizontal
			+ ';v:' + this.vertical;
	}
	// @lx:context>

	// @lx:<context CLIENT:
	unpackProcess(config) {
		this.direction = +config.d;
		this.horizontal = +config.h;
		this.vertical = +config.v;
	}
	// @lx:context>

	allocate(elem, config) {
		var geom = this.geomFromConfig(config);
		if (geom.w !== undefined) elem.style('width', geom.w);
		if (geom.h !== undefined) elem.style('height', geom.h);
		this.actualizeIndents();
	}

	setIndents(config) {
		super.setIndents(config);
		var indents = this.getIndents();

		//TODO - same for grid
		if (indents.paddingTop) this.owner.style('padding-top', indents.paddingTop);
		if (indents.paddingBottom) this.owner.style('padding-bottom', indents.paddingBottom);
		if (indents.paddingLeft) this.owner.style('padding-left', indents.paddingLeft);
		if (indents.paddingRight) this.owner.style('padding-right', indents.paddingRight);

		return indents;
	}

	actualizeIndents(indents) {
		if (!indents) indents = this.getIndents();

		if (this.direction == lx.HORIZONTAL && indents.stepX) {
			var lastIndex = this.owner.childrenCount() - 1;
			this.owner.getChildren().forEach((el, i)=>{
				el.style('margin-bottom', null);
				if (i == lastIndex) el.style('margin-right', null);
				else el.style('margin-right', indents.stepX);
			});
		} else if (this.direction == lx.VERTICAL && indents.stepY) {
			var lastIndex = this.owner.childrenCount() - 1;
			this.owner.getChildren().forEach((el, i)=>{
				el.style('margin-right', null);
				if (i == lastIndex) el.style('margin-bottom', null);
				else el.style('margin-bottom', indents.stepX);
			});
		}
	}
}
