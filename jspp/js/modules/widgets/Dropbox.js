// @lx:module lx.Dropbox;

// @lx:use lx.Input;
// @lx:use lx.Table;
// @lx:use lx.BasicCssContext;

// @lx:<context CLIENT:
let _opened = null;
// @lx:context>

/**
 * @widget lx.Dropbox
 * @content-disallowed
 * 
 * CSS Classes:
 * - lx-Dropbox
 * - lx-Dropbox-input
 * - lx-Dropbox-but
 * - lx-Dropbox-cell
 * 
 * Events:
 * - change
 * - opened
 * - closed
 */
// @lx:namespace lx;
class Dropbox extends lx.Box {
	static initCss(css) {
		css.useHolder(lx.BasicCssContext);
		css.addClass('lx-Dropbox', {
			borderRadius: css.preset.borderRadius,
			cursor: 'pointer',
			overflow: 'hidden'
		}, {
			disabled: 'opacity: 0.5'
		});
		css.addClass('lx-Dropbox-input', {
			position: 'absolute',
			width: 'calc(100% - 30px)',
			height: '100%',
			fontSize: 'inherit',
			borderTopRightRadius: '0 !important',
			borderBottomRightRadius: '0 !important'
		});
		css.addClass('lx-Dropbox-but', {
			position: 'absolute',
			right: 0,
			width: '30px',
			height: '100%',
			borderTop: '1px solid ' + css.preset.widgetBorderColor,
			borderBottom: '1px solid ' + css.preset.widgetBorderColor,
			borderRight: '1px solid ' + css.preset.widgetBorderColor,
			color: css.preset.widgetIconColor,
			background: css.preset.widgetGradient,
			cursor: 'pointer',
			'@icon': ['\\25BC', 15]
		});
		css.addClass('lx-Dropbox-cell', {
		}, {
			hover: {
				color: css.preset.checkedSoftColor,
				backgroundColor: css.preset.checkedDarkColor
			}
		});
	}

	/**
	 * @widget-init
	 *
	 * @param [config] {Object: {
	 *     #merge(lx.Rect::constructor::config),
	 *     [placeholder] {String},
	 *     [options] {Array<String|Number>|Dict<String|Number>},
	 *     [value = null] {Number|String} (: active value key :),
	 *     [button] {Boolean} (: Flag for rendering open-close button :),
	 *     [optionsConfig] {Object: {
	 *         fontSize {String|Number},
	 *         minHeight {String}
	 *     }}
	 * }}
	 */
	render(config) {
		super.render(config);

		this.addClass('lx-Dropbox');

		new lx.Input({
			parent: this,
			key: 'input',
			css: 'lx-Dropbox-input',
		});
		if (config.placeholder) lx(this)>input.placeholder(placeholder);

		let button = (config.button === undefined) ? true : config.button;
		if (button)
			new lx.Rect({
				parent: this,
				key: 'button',
				css: 'lx-Dropbox-but',
			});

		this.optionsConfig = config.optionsConfig || {};
		this._options = config.options || [];
		this._opened = false;
		this._optionsBox = null;
		this.value(config.value !== undefined ? config.value : null);
	}

	// @lx:<context CLIENT:
	clientRender(config) {
		super.clientRender(config);
		this._options = lx.Dict.create(this._options);
		this.click(_handler_open);
		if (this.contains('button'))
			lx(this)>button.click(_handler_toggle);
	}

	postUnpack(config) {
		super.postUnpack(config);
		this._options = lx.Dict.create(this._options);
	}

	close() {
		if (!_opened) return;
		this._optionsBox.del();
		this._optionsBox = null;
		lx.off('click', _handler_checkOutclick);
		_opened = null;
		this._opened = false;
		this.trigger('closed');
	}

	getOption(index) {
		return this._options.nth(index)
	}

	static getOpened() {
		return _opened;
	}
	// @lx:context>

	select(index) {
		this.value(this._options.nthKey(index));
	}

	options(options) {
		if (options === undefined) return this._options;
		// @lx:<context SERVER:
		this._options = options;
		// @lx:context>
		// @lx:<context CLIENT:
		this._options = lx.Dict.create(options);
		// @lx:context>
		return this;
	}

	selectedText() {
		if (this.valueKey === null || this.valueKey === '') return '';

		return this._options[this.valueKey];
	}

	value(key) {
		if (key === undefined) {
			if (this.valueKey === undefined) return null;
			return this.valueKey;
		}

		this.valueKey = key;
		lx(this)>>input.value(this.selectedText());
		return this;
	}
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * PRIVATE
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

// @lx:<context CLIENT:
function _handler_open(event) {
	event.stopPropagation();

	_initOptions(this).show();
	this.trigger('opened', event);

	_opened = this;
	this._opened = true;
	lx.on('click', _handler_checkOutclick);
}

function _handler_toggle(event) {
	event.stopPropagation();

	if (_opened === this.parent) {
		this.parent.close();
		return;
	}

	_handler_open.call(this.parent, event);
}

function _handler_choose(event) {
	let dropbox = _opened,
		oldVal = dropbox.value(),
		num = this.rowIndex();
	dropbox.select(num);
	event = event || dropbox.newEvent();
	event.oldValue = oldVal;
	event.newValue = dropbox.value();
	dropbox.trigger('change', event);
	dropbox.close();
}

function _handler_checkOutclick(event) {
	if (!_opened) return;

	if (_opened.containGlobalPoint(event.clientX, event.clientY)
		|| _opened._optionsBox.containGlobalPoint(event.clientX, event.clientY)
	) return;

	_opened.close();
}

function _initOptions(self) {
	let optionsBox = _createOptionsBox(self);

	optionsBox.style(
		'fontSize',
		self.optionsConfig.fontSize || window.getComputedStyle(self.getDomElem()).fontSize
	);
	optionsBox.width(self.width('px')+'px');
	optionsBox.height('60%');

	let options = [];
	for (let key in self._options)
		options.push(self._options[key]);

	let optionsTable = lx(optionsBox)>dropboxOptionsTable;
	optionsTable.resetContent(options, true);

	optionsTable.cells().forEach(child => {
		child.align(lx.CENTER, lx.MIDDLE);
		child.click(_handler_choose);
		child.addClass('lx-Dropbox-cell');
		if (self.optionsConfig.minHeight)
			child.parent.style('minHeight', self.optionsConfig.minHeight);
	});

	if (optionsTable.height('px') < optionsBox.height('px'))
		optionsBox.height(optionsTable.height('px') + 'px');

	optionsBox.satelliteTo(self);
	self._optionsBox = optionsBox;
	return optionsBox;
}

function _createOptionsBox(self) {
	let conf = {
		parent: lx.app.root,
		key: 'dropboxOptionsWrapper',
		geom: [0, 0, '100%', '60%'],
		depthCluster: lx.DepthClusterMap.CLUSTER_OVER,
	};
	let cssScope = self.getCssScope();
	if (cssScope != '') conf.cssScope = cssScope;
	let wrapper = new lx.Box(conf).hide();
	new lx.Table({
		parent: wrapper,
		key: 'dropboxOptionsTable',
		geom: true,
		height: 'auto',
		cols: 1,
	});
	wrapper.style('overflow-y', 'auto');
	return wrapper;
}
// @lx:context>
