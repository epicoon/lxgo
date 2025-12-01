// @lx:module lx.Checkbox;

// @lx:use lx.BasicCssContext;

/**
 * @widget lx.Checkbox
 * @content-disallowed
 * 
 * CSS classes:
 * - lx-Checkbox-1
 * - lx-Checkbox-0
 */
// @lx:namespace lx;
class Checkbox extends lx.Box {
	static initCss(css) {
		css.useExtender(lx.BasicCssContext);
		css.addClass('lx-Checkbox-0', {
			border: 'solid #61615e 1px',
			width: '16px',
			height: '16px',
			borderRadius: '4px',
			backgroundColor: 'white',
			cursor: 'pointer'
		}, {
			hover: {
				boxShadow: '0 0 6px ' + css.preset.widgetIconColor,
			},
			active: {
				backgroundColor: '#dedede',
				boxShadow: '0 0 8px ' + css.preset.widgetIconColor,
			}
		});
		css.inheritClass('lx-Checkbox-1', 'lx-Checkbox-0', {
			color: 'black',
			'@icon': ['\\2713', {fontFamily:'main', fontSize:8, fontWeight:600, paddingLeft:'1px', paddingBottom:'0px'}],
		});
	}

	cssChecked() {
		return 'lx-Checkbox-1';
	}

	cssUnchecked() {
		return 'lx-Checkbox-0';
	}

	/**
	 * @widget-init
	 *
	 * @param [config] {Object: {
	 *     #merge(lx.Rect::constructor::config),
	 *     [value = false] {Boolean}
	 * }}
	 */
	render(config) {
		super.render(config);
		this.add(lx.Box, {
			key: 'check',
			coords: [0, 0],
			// geom: [0, 0, '24px', '24px']
		});
		this.align(lx.CENTER, lx.MIDDLE);
		this.value(config.value || false);
	}

	// @lx:<context CLIENT:
	clientRender(config) {
		super.clientRender(config);
		this.on('mousedown', lx.preventDefault);
		this.on('mouseup', lx.self(click));
	}

	static click(event) {
		this.value( !this.value() );
		this.trigger('change', event);
	}
	// @lx:context>

	value(val) {
		if (val === undefined) return this.state;

		this.state = !!val;
		lx(this)>check.removeClass(this.cssChecked());
		lx(this)>check.removeClass(this.cssUnchecked());
		if (this.state) lx(this)>check.addClass(this.cssChecked());
		else lx(this)>check.addClass(this.cssUnchecked());

		return this;
	}

	todgle() {
		this.value(!this.value());
		this.trigger('change', this.newEvent({
			oldValue: !this.value(),
			newValue: this.value()
		}));
		return this;
	}
}