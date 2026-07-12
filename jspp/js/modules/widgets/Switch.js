// @lx:module lx.Switch;

/**
 * @widget lx.Switch
 * @content-disallowed
 */
// @lx:namespace lx;
class Switch extends lx.Box {
	static initCss(css) {
		css.addClass('lx-switch', {
			position: 'relative',
			width: '48px',
			height: '26px',
			backgroundColor: '#ccc',
			borderRadius: '26px',
			transition: 'background-color 0.3s',
			cursor: 'pointer',
		}, {
			':before': {
				content: '""',
				position: 'absolute',
				left: '3px',
				top: '3px',
				width: '20px',
				height: '20px',
				backgroundColor: 'white',
				borderRadius: '50%',
				transition: 'transform 0.3s',
			},
			'.active'    : { backgroundColor: css.preset.checkedMainColor },
			'.pending': { backgroundColor: css.preset.neutralMainColor },
			'.error'     : { backgroundColor: css.preset.hotMainColor },
			'.active:before'    : { transform: 'translateX(22px)' },
			'.pending:before': { transform: 'translateX(22px)' },
			'.error:before'     : { transform: 'translateX(22px)' },
		});
	}

    static getStaticTag() {
		return 'label';
	}

	/**
	 * @widget-init
	 *
	 * @param [config] {Object: {
	 *     #merge(lx.Rect::constructor::config),
     *     [manual = false] {Boolean}
	 *     [value = false] {Boolean}
	 * }}
	 */
	render(config) {
		super.render(config);
        this.addClass('lx-switch');
        this.state = false;
        if ('value' in config)
            this.value(!!config.value);
	}

	// @lx:<context CLIENT:
	clientRender(config) {
		super.clientRender(config);
        if (!lx.getFirstDefined(config.manual, false))
            this.on('click', e=>this.todgle());
	}
	// @lx:context>

	value(val) {
        if (val === undefined)
            return this.state;

        this.state = !!val;
        this.state
            ? this.addClass('active')
            : _reset(this);

        return this;
	}

	setState(state) {
		_reset(this);
		if (state !== 'inactive') {
			this.addClass(state);
			this.state = true;
		} else this.state = false;
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

function _reset(self) {
	self.removeClass('active');
	self.removeClass('pending');
	self.removeClass('error');
}
