// @lx:module lx.InputPopup;
// @lx:module-data: i18n = i18n.yaml;

// @lx:use lx.Button;
// @lx:use lx.Input;

// @lx:<context CLIENT:
let __instance = null,
	__active = null;
// @lx:context>

/**
 * CSS classes:
 * - lx-InputPopup-back
 */
// @lx:namespace lx;
class InputPopup extends lx.Box {
	static initCss(css) {
		css.addClass('lx-InputPopup-back', {
			color: css.preset.textColor,
			backgroundColor: css.preset.bodyBackgroundColor,
			borderRadius: css.preset.borderRadius,
			border: 'solid 1px ' + css.preset.widgetBorderColor,
		});
	}

	modifyConfigBeforeApply(config) {
    	config.key = config.key || 'inputPopup';
		config.geom = config.geom || ['0%', '0%', '100%', '100%'];
		config.depthCluster = lx.DepthClusterMap.CLUSTER_OVER;

		//TODO ???
		// style: { position: 'fixed' }

        return config;
    }

	// @lx:<context CLIENT:
	clientRender(config) {
		this.holder = _getHolder();
		_render(this);
	}

	static open(title, captions, defaults = []) {
		if (!__instance)
			__instance = new lx.InputPopup({parent: lx.app.root});
		return __instance.open(title, captions, defaults);
	}

	static close() {
		if (__instance)
			_close(__instance);
	}

	open(title, captions, defaults = []) {
		if (!lx.isArray(captions)) captions = [captions];

		let buttons = lx(this)>stream>buttons;

		lx(this)>stream.del('r');
		this.useRenderCache();
		if (title) {
			let row = new lx.Box({
				key: 'title',
				text: title,
				before: buttons
			});
			row.align(lx.CENTER, lx.MIDDLE);
			row.height( lx(row)>text.height('px') + 10 + 'px' );
		}
		captions.forEach((caption, i)=>{
			let row = new lx.Box({
				key: 'r',
				before: buttons
			});
			row.gridProportional({ step: '10px', cols: 2 });

			let textBox = row.add(lx.Box, {
				text : caption,
				width: 1
			});
			textBox.align(lx.CENTER, lx.MIDDLE);
			let input = row.add(lx.Input, {
				key: 'input',
				width: 1
			});
			if (defaults[i] !== undefined) input.value(defaults[i]);

			row.height( lx(textBox)>text.height('px') + 10 + 'px' );
		});
		this.applyRenderCache();

		let top = (this.height('px') - lx(this)>stream.height('px')) * 0.5;
		if (top < 0) top = 0;
		lx(this)>stream.top(top + 'px');

		__active = this;
		this.show();

		lx.app.keyboard.onKeydown(13, _onConfirm);
		lx.app.keyboard.onKeydown(27, _onReject);

		let rows = lx(this)>stream>r;
		if (lx.isArray(rows)) lx(rows[0])>input.focus();
		else lx(rows)>input.focus();

		return this.holder;
	}

	close() {
		_close(this);
	}
	// @lx:context>
}

// @lx:<context CLIENT:
function _getHolder() {
	let holder = {
		_confirmCallback: null,
		_rejectCallback: null,
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

	let inputPopupStream = new lx.Box({key:'stream', geom:['30%', '40%', '40%', '0%'], css:'lx-InputPopup-back'});
	inputPopupStream.stream({indent:'10px'});

	inputPopupStream.begin();
		let buttons = new lx.Box({key:'buttons', height:'35px'});
		buttons.grid({step:'10px',cols:2});
		new lx.Button({parent:buttons, key:'confirm', width:1, text:lx(i18n).OK});
		new lx.Button({parent:buttons, key:'reject', width:1, text:lx(i18n).Close});
	inputPopupStream.end();

	lx(inputPopupStream)>>confirm.click(_onConfirm);
	lx(inputPopupStream)>>reject.click(_onReject);
}

function _onConfirm() {
	if (!__active) return;
	let callback = __active.holder._confirmCallback;
	if (callback) {
		let values = [];
		if (lx(__active)>stream.contains('r')) {
			let rows = lx(__active)>stream>r;
			if (rows) {
				if (!lx.isArray(rows)) rows = [rows];
				rows.forEach(a=>values.push(lx(a)>input.value()));
			}
		}
		if (values.len == 1) values = values[0];
		if (lx.isFunction(callback)) callback(values);
		else if (lx.isArray(callback))
			callback[1].call(callback[0], values);
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

function _close(popup) {
	popup.hide();
	popup.holder._confirmCallback = null;
	popup.holder._rejectCallback = null;
	lx.app.keyboard.offKeydown(13, _onConfirm);
	lx.app.keyboard.offKeydown(27, _onReject);
}
// @lx:context>
