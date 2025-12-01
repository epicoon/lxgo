// @lx:module lx.Slider;

// @lx:use lx.BasicCssContext;

/**
 * @widget lx.Slider
 * @content-disallowed
 * 
 * CSS classes:
 * - lx-slider-track
 * - lx-slider-handle
 * 
 * Events:
 * - started
 * - moved
 * - stopped
 * - change
 */
// @lx:namespace lx;
class Slider extends lx.Box {
	static initCss(css) {
		css.useExtender(lx.BasicCssContext);
		css.inheritClass('lx-slider-track', 'Button');
		css.inheritClass('lx-slider-handle', 'ActiveButton');
	}

	/**
	 * @widget-init
	 *
	 * @param [config] {Object: {
	 *     #merge(lx.Rect::constructor::config),
	 *     [min = 0] {Number},
	 *     [max = 100] {Number},
	 *     [step = 1] {Number},
	 *     [value = 0] {Number}
	 * }}
	 */
	render(config) {
		super.render(config);

		this.min = config.min || 0;
		this.max = config.max || 100;
		this.step = config.step || 1;

		let value = config.value || 0;
		if (value < this.min) value = this.min;
		if (value > this.max) value = this.max;
		this._value = value;

		this.style('overflow', 'visible');

		let track = new lx.Rect({
			parent: this,
			geom: true,
			key: 'track',
			css: 'lx-slider-track'
		});
		let handle = new lx.Rect({
			parent: this,
			geom: true,
			key: 'handle',
			css: 'lx-slider-handle'
		});
	}

	// @lx:<context CLIENT:
	clientRender(config) {
		super.clientRender(config);

		let h = this.height('px'),
			w = this.width('px'),
			handleSize = Math.min(h, w);
		this.orientation = (w > h)
			? lx.HORIZONTAL
			: lx.VERTICAL;

		let handle = this.handle();
		handle.size(handleSize+'px', handleSize+'px');
		this.locateHandle();

		if (this.orientation == lx.HORIZONTAL)
			lx(this)>track.setGeom(['0%', '17%', '100%', '66%']);
		else
			lx(this)>track.setGeom(['17%', '0%', '66%', '100%']);

		handle.move()
			.on('moveBegin', lx.self(start))
			.on('move', lx.self(move))
			.on('moveEnd', lx.self(stop));

		lx(this)>track.click(lx.self(trackClick));
	}
	// @lx:context>

	started(f) {
		this.on('started', f);
		return this;
	}

	moved(f) {
		this.on('moved', f);
		return this;
	}

	stopped(f) {
		this.on('stopped', f);
		return this;
	}

	change(f) {
		this.on('change', f);
		return this;
	}

	handle() {
		return lx(this)>handle;
	}

	value(val, event) {
		if (val === undefined) {
			return this._value;
		}

		if (val > this.max) val = this.max;
		if (val < this.min) val = this.min;
		this._value = val;

		// @lx:<context CLIENT:
		this.locateHandle();
		if (event) this.trigger('moved', event);
		// @lx:context>

		return this;
	}

	// @lx:<context CLIENT:
	static start(event) {
		this.parent._oldValue = this.parent._value;
		this.parent.trigger('started', event);
	}

	static move(event) {
		this.parent.setValueByHandle(this, event);
	}

	static stop(event) {
		let oldVal = this.parent._oldValue;
		if (this.parent._value == oldVal) return;
		event = event || this.newEvent();
		event.oldValue = oldVal;
		event.newValue = this.parent.value();
		this.parent.trigger('change', event);
		this.parent.trigger('stopped', event);
	}

	static trackClick(event) {
		let slider = this.parent,
			handle = lx(slider)>handle,
			point = slider.globalPointToInner(event),
			crd  , param;
		if (slider.orientation == lx.HORIZONTAL) {
			crd = point.x - handle.width('px') * 0.5;
			param = 'left';
		} else {
			crd = point.y - handle.height('px') * 0.5;
			param = 'top';
		}
		handle[param](crd + 'px');
		handle.returnToParentScreen();
		let oldVal = slider.value();
		slider.setValueByHandle(handle);
		if (slider.value() != oldVal) {
			event = event || this.newEvent();
			event.oldValue = oldVal;
			event.newValue = slider.value();
			slider.trigger('change', event);
		}
	}

	setValueByHandle(handle, event) {
		let val, rangeW,
			min = this.min,
			max = this.max,
			step = this.step,
			range = max - min;

		if (this.orientation == lx.HORIZONTAL) {
			val = handle.left('px');
			rangeW = this.width('px') - handle.width('px');
		} else {
			val = handle.top('px');
			rangeW = this.height('px') - handle.height('px');
		}

		let locval = (val * range / rangeW) + min;
		locval = Math.floor(locval / step) * step;

		if (this._value != locval) this.value(locval, event);
	}

	locateHandle() {
		let handle = this.handle(),
			range = this.max - this.min,
			rangeW, pos;
		if ( this.orientation == lx.HORIZONTAL ) {
			rangeW = this.width('px') - handle.width('px');
			pos = (this._value - this.min) * rangeW / range;
			handle.left( pos + 'px' );
		} else {
			rangeW = this.height('px') - handle.height('px');
			pos = (this._value - this.min) * rangeW / range;
			handle.top( pos + 'px' );
		}
	}
	// @lx:context>
}
