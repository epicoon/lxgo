// @lx:module lx.LanguageSwitcher;

/**
 * Flag-icon language switcher, driven by lx.app.lang (lx.Language). The
 * language list and full names come from lx.app.lang.options(); flag images
 * are not built in - the caller supplies them via the "flags" config option
 * (locale key -> image path, e.g. {"en-EN": "lang/en.png"}). A locale with
 * no matching entry in "flags" just shows its code with no icon.
 *
 * @widget lx.LanguageSwitcher
 * @content-disallowed
 *
 * Events:
 * - change
 */
// @lx:namespace lx;
class LanguageSwitcher extends lx.Box {
	static initCss(css) {
		css.addClass('lx-LanguageSwitcher', {
			cursor: 'pointer',
			borderRadius: '5px',
			overflow: 'hidden'
		});
		css.addClass('lx-LanguageSwitcher-list', {
			borderRadius: '5px',
			overflow: 'hidden',
			background: '#fff',
			border: '1px solid #ddd'
		});
		css.addClass('lx-LanguageSwitcher-row', {
			cursor: 'pointer'
		}, {
			hover: { background: '#eee' }
		});
	}

	/**
	 * @widget-init
	 *
	 * @param [config] {Object: {
	 *     #merge(lx.Rect::constructor::config),
	 *     [flags] {Dict<String>} (: locale key (as in lx.app.lang.options())
	 *         -> flag image path; a locale missing here renders with no icon :)
	 * }}
	 */
	render(config = {}) {
		super.render(config);
		this.addClass('lx-LanguageSwitcher');
		this.flags = config.flags || {};

		new lx.Box({parent: this, key: 'flag', geom: [6, 18, 26, 64]});
		new lx.Box({parent: this, key: 'code', geom: [34, 0, 60, 100]})
			.align(lx.CENTER, lx.MIDDLE);
	}

	// @lx:<context CLIENT:
	clientRender(config) {
		super.clientRender(config);
		this._list = null;
		this.click(()=>_toggle(this));
		_setCurrent(this, lx.app.lang.current());
	}
	// @lx:context>
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * PRIVATE
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

// @lx:<context CLIENT:
lx.LanguageSwitcher.opened = [];

function _shortCode(key) {
	return key.split('-')[0].toUpperCase();
}

function _setCurrent(self, key) {
	if (self.flags[key]) lx(self)>>flag.picture(self.flags[key]);
	lx(self)>>code.text(_shortCode(key));
}

function _toggle(self) {
	if (self._list) _close(self);
	else _open(self);
}

function _open(self) {
	const options = lx.app.lang.options(),
		rowHeight = 30,
		rowCount = Object.keys(options).length;

	const list = new lx.Box({
		parent: lx.app.root,
		css: 'lx-LanguageSwitcher-list',
		geom: [0, 0, self.width('px') + 'px', (rowCount * rowHeight) + 'px'],
		depthCluster: lx.DepthClusterMap.CLUSTER_OVER
	});

	let i = 0;
	for (let key in options) {
		const row = new lx.Box({
			parent: list,
			css: 'lx-LanguageSwitcher-row',
			geom: [0, (i++ * rowHeight) + 'px', '100%', rowHeight + 'px']
		});
		if (self.flags[key])
			new lx.Box({
				parent: row,
				geom: [6, 15, 22, 70],
				picture: self.flags[key]
			});
		new lx.Box({parent: row, geom: [32, 0, 62, '100%'], text: options[key]})
			.align(lx.LEFT, lx.MIDDLE);
		row.click(()=>{
			_close(self);
			const old = lx.app.lang.current();
			lx.app.lang.set(key);
			self.trigger('change', {oldValue: old, newValue: key});
		});
	}

	list.satelliteTo(self);
	self._list = list;
	lx.LanguageSwitcher.opened.push(self);
	setTimeout(()=>lx.on('click', _handler_outclick));
}

function _close(self) {
	if (!self._list) return;
	self._list.del();
	self._list = null;
	lx.LanguageSwitcher.opened.lxRemove(self);
}

function _handler_outclick() {
	lx.off('click', _handler_outclick);
	lx.LanguageSwitcher.opened.slice().forEach(self => _close(self));
}
// @lx:context>
