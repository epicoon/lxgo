/**
 * @widget lx.TextBox
 * @content-disallowed
 */
// @lx:namespace lx;
class TextBox extends lx.Rect {
	modifyConfigBeforeApply(config='') {
		if (lx.isString(config)) config = {text: config};

		if (!config.key) config.key = 'text';
		if (config.text) {
			config.html = config.text;
			delete config.text;
		}

		config.width = 'auto';
		config.height = 'auto';

		return config;
	}

	render(config={}) {
		super.render(config);
		this.addClass('lx-TextBox');
	}

	static initCss(css) {
		css.addClass('lx-TextBox', {
			padding: '0px 10px',
			width: 'auto',
			height: 'auto',

			fontFamily: 'inherit',
			fontSize: 'inherit',
			color: 'inherit',

			cursor: 'inherit',
			overflow: 'inherit',
			whiteSpace: 'inherit',
			textOverflow: 'inherit',
		});
	}

	text(val) {
		return this.value(val);
	}

	value(val) {
		if (val === undefined) return this.html();
		this.html(val);
		this.reportSizeHasChanged();
		// @lx:<context CLIENT:
		this.checkResize();
		// @lx:context>
	}

	setFontSize(sz) {
		this.style('fontSize', sz);
		this.reportSizeHasChanged();
	}

	ellipsis() {
		//TODO use CSS
		this.style({
			overflow: 'hidden',
			whiteSpace: 'nowrap',
			textOverflow: 'ellipsis'
		});
	}

	wrap(mode) {
		this.style('white-space', mode);
	}

	// @lx:<context SERVER:
	adapt() {
		this.onPostUnpack('.adapt');
	}
	// @lx:context>

	// @lx:<context CLIENT:
	adapt() {
		if (!this.parent) return this;

		let ctx = this.parent,
			text = this,
			hor = text.domElem.param('clientWidth') > text.domElem.param('clientHeight'),
			horFailed = false;

		if (hor) {
			let cW = ctx.domElem.param('clientWidth'),
				maxW = window.screen.width,
				resW = lx.Math.halfDivisionMethod(0, maxW, function(res) {
					text.domElem.style('fontSize', Math.floor(res) + 'px');
					let tW = text.domElem.param('clientWidth');
					if ( Math.floor(cW) == Math.floor(tW) ) return 0;
					if ( tW > cW ) return 1;
					if ( ( Math.abs(tW - cW) <= 5 ) ) return 0;
					return -1;
				}, 0.1, 50);

			horFailed = text.domElem.param('clientHeight') > ctx.domElem.param('clientHeight');
		}

		if (!hor || horFailed) {
			let cH = ctx.domElem.param('clientHeight'),
				lastH = horFailed ? parseInt(text.domElem.style('fontSize'), 10) : window.screen.height,
				resH = lx.Math.halfDivisionMethod(0, lastH, function(res) {
					text.domElem.style('fontSize', Math.floor(res) + 'px');
					let tH = text.domElem.param('clientHeight');
					if ( Math.floor(cH) == Math.floor(tH) ) return 0;
					if ( tH > cH ) return 1;
					if ( ( Math.abs(tH - cH) <= 1 ) ) return 0;
					return -1;
				}, 0.1, 50);
		}

		ctx.childHasAutoresized(text);
		return this;
	}

	/**
	 * lx.Collection - collection of TextBoxes
	 */
	static adaptTextByMin(c) {
		c = lx.Collection.cast(c);
		c.forEach(child=>child.adapt());
		let min = Infinity;
		c.forEach(a=>{
			let s = parseFloat(a.domElem.style('fontSize'));
			if (min > s) min = s;
		});
		min = min + 'px';
		c.forEach(a=>a.setFontSize(min));
	}
	// @lx:context>
}
