// @lx:module lx.ConfirmPopup;
// @lx:module-data: i18n = i18n.yaml;

// @lx:use lx.Button;

// @lx:<context CLIENT:
let __instance = null,
	__active = null;
// @lx:context>

/**
 * CSS classes:
 * - lx-ConfirmPopup-back
 */
// @lx:namespace lx;
class ConfirmPopup extends lx.Box {
	// @lx:const COLS_FOR_EXTRA_BUTTONS = 1;

	static initCss(css) {
		css.addClass('lx-ConfirmPopup-back', {
			color: css.preset.textColor,
			backgroundColor: css.preset.bodyBackgroundColor,
			borderRadius: css.preset.borderRadius,
			border: 'solid 1px ' + css.preset.widgetBorderColor,
		});
	}

	modifyConfigBeforeApply(config) {
    	config.key = config.key || 'confirmPopup';
		config.geom = config.geom || ['0%', '0%', '100%', '100%'];
		config.depthCluster = lx.DepthClusterMap.CLUSTER_OVER;

		//TODO ???
		// style: { position: 'fixed' }

        return config;
    }

	/**
	 * @param [config] {Object: {
	 *     #merge(lx.Box::render::config),
	 *     [customButtons = false] {Boolean}
	 * }}
	 */
	render(config) {
		if (config.customButtons !== undefined)
			this.customButtons = config.customButtons;
	}

	// @lx:<context CLIENT:
	clientRender(config) {
		this.holder = _getHolder(this);
		_render(this);
	}

	static open(message, extraButtons = {}, buttonColsCount = 2) {
		if (!__instance)
			__instance = new lx.ConfirmPopup({parent: lx.app.root});
		return __instance.open(message, extraButtons, buttonColsCount);
	}

	static close() {
		if (__instance)
			_close(__instance);
	}

	open(message, extraButtons = {}, buttonColsCount = 2) {
		lx(this)>stream>message.text(message);
		lx(this)>stream>message.height(
			lx(this)>stream>message>text.height('px') + 10 + 'px'
		);

		let top = (this.height('px') - lx(this)>stream.height('px')) * 0.5;
		if (top < 0) top = 0;
		lx(this)>stream.top(top + 'px');

		__active = this;
		this.show();

		lx.app.keyboard.onKeydown(13, _onConfirm);
		lx.app.keyboard.onKeydown(27, _onReject);
		_applyExtraButtons(this.holder, extraButtons, buttonColsCount);
		return this.holder;
	}

	close() {
		_close(this);
	}
	// @lx:context>
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * PRIVATE
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

// @lx:<context CLIENT:
function _getHolder(popup) {
	let holder = {
		_popup: popup,
		_confirmCallback: null,
		_rejectCallback: null,
		_extraButtons: null,
		_extraCallbacks: {}
	};
	holder.confirm = function(callback) {
		this._confirmCallback = callback;
		return this;
	}
	holder.reject = function(callback) {
		this._rejectCallback = callback;
		return this;
	}
	return holder;
}

function _applyExtraButtons(holder, extraButtons, colsCount) {
	if (extraButtons.lxEmpty()) return;

	holder._extraButtons = extraButtons;
	let buttonsWrapper = lx(holder._popup)>>buttons;
	buttonsWrapper.positioning().setCols(colsCount);
	for (let name in extraButtons) {
		let text = extraButtons[name];
		buttonsWrapper.add(lx.Button, {key:'extra', width:1, text, click:()=>_onExtra(holder, name)});
		holder[name] = function(callback) {
			this._extraCallbacks[name] = callback;
			return this;
		};
	}
}

function _clearExtraButtons(holder) {
	if (holder._extraButtons === null) return;

	let buttonsWrapper = lx(holder._popup)>>buttons;
	buttonsWrapper.del('extra');
	buttonsWrapper.positioning().setCols(2);

	for (let name in holder._extraButtons)
		delete holder[name];
	holder._extraButtons = null;
	holder._extraCallbacks = {};
}

function _render(self) {
	self.overflow('auto');
	self.useRenderCache();
	self.begin();
	_renderContent(self);
	self.end();
	self.applyRenderCache();
	self.hide();
}

function _renderContent(self) {
	(new lx.Rect({geom:true})).fill('black').opacity(0.5);

	let confirmPopupStream = new lx.Box({key:'stream', geom:['30%', '40%', '40%', '0%'], css:'lx-ConfirmPopup-back'});
	confirmPopupStream.stream({indent:'10px'});

	confirmPopupStream.begin();
		(new lx.Box({key:'message'})).align(lx.CENTER, lx.MIDDLE);

		let buttons = new lx.Box({key:'buttons', height:'35px'});
		buttons.grid({step:'10px', cols:2});

		if (!self.customButtons) {
			new lx.Button({parent:buttons, key:'confirm', width:1, text:lx(i18n).Yes});
			new lx.Button({parent:buttons, key:'reject', width:1, text:lx(i18n).No});
			lx(confirmPopupStream)>>confirm.click(_onConfirm);
			lx(confirmPopupStream)>>reject.click(_onReject);
		}
	confirmPopupStream.end();
}

function _onConfirm() {
	if (!__active) return;
	let callback = __active.holder._confirmCallback;
	if (callback) {
		if (lx.isFunction(callback)) callback();
		else if (lx.isArray(callback))
			callback[1].call(callback[0]);
	} 
	_close(__active);
}

function _onReject() {
	if (!__active) return;
	let callback = __active.holder._rejectCallback;
	if (callback) {
		if (lx.isFunction(callback)) callback();
		else if (lx.isArray(callback))
			callback[1].call(callback[0]);
	} 
	_close(__active);
}

function _onExtra(holder, name) {
	let callback = holder._extraCallbacks[name];
	if (callback) {
		if (lx.isFunction(callback)) callback();
		else if (lx.isArray(callback))
			callback[1].call(callback[0]);
	} 
	_close(holder._popup);
}

function _close(popup) {
	if (!__active) return;
	popup.hide();
	popup.holder._confirmCallback = null;
	popup.holder._rejectCallback = null;
	_clearExtraButtons(popup.holder);
	lx.app.keyboard.offKeydown(13, _onConfirm);
	lx.app.keyboard.offKeydown(27, _onReject);
	__active = null;
}
// @lx:context>
