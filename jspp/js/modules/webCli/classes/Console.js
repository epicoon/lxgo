#lx:public;

class Console {
	#lx:const
		MODE_INPUT = 'input',
		MODE_SELECT = 'select';

	static init(consoleBox) {
		this.carret = new TextSelection();
		this.commandPressed = false;
		this.useCache = false;
		this.cache = [];
		this.inputMode = null;

		this.consoleBox = consoleBox;
		this._c = consoleBox.getDomElem();
		this.currentRow = 0;
		this.callbackMap = [];

		this.selector = null;
		this.selected = null;
		this._c.setAttribute('contentEditable', 'true');
		this._c.style.whiteSpace = 'pre';
		this._c.style.overflow = 'auto';
		consoleBox.on('mouseup', function(event) {
			Console.checkCarret();
		});
		var t = this;
		consoleBox.on('focus', function(event) {
			if (!t.selector) {
				this.on('keydown', Console.onKeydown);
				this.on('keyup', Console.onKeyup);
			}
		});
		consoleBox.on('blur', function(event) {
			if (!t.selector) {
				this.off('keydown', Console.onKeydown);
				this.off('keyup', Console.onKeyup);
			}
		});

		this.locCss = 'lxWC-loc';
		this.commandCss = 'lxWC-command';
	}

	static input(text, decor=null) {
		var css = decor
			? 'lxWC-msg_' + decor
			: this.locCss;
		this.addText('<span class="'+css+'" lxwc-type="loc" lxwc-row="'+this.currentRow+'">'
			+ text
			+ '</span><span class="'+this.commandCss+'" lxwc-type="command" lxwc-row="'+this.currentRow+'"> </span>'
		);

		this.inputMode = this.MODE_INPUT;
		this.checkCarret();
	}

	static select(config) {
		if (config.hintText) {
			var decor = (config.hintDecor && config.hintDecor.decor) ? config.hintDecor.decor : null;
			this.input(config.hintText, decor);
		}

		this.carret.dropRange();
		this.commandPressed = false;
		this.consoleBox.off('keydown', Console.onKeydown);
		this.consoleBox.off('keyup', Console.onKeyup);

		this.inputMode = this.MODE_SELECT;

		var div = document.createElement('div');
		this._c.appendChild(div);
		this.selector = new ConsoleSelect(
			this,
			this.consoleBox,
			div,
			config.options || [],
			lx.getFirstDefined(config.withQuit, true)
		);
		this.selector.start();
	}

	static out(text, decor = '') {
		var css = 'lxWC-msg_' + decor,
			text = '<span class="'+css+'" lxwc-type="msg" lxwc-row="'+this.currentRow+'">' + text + '</span>';
		if (this.useCache) this.cache.push(text);
		else this.addText(text);
	}

	static outln(text = '', decor = '') {
		this.out(text, decor);
		if (this.useCache) this.cache.push('<br>');
		else this.addText('<br>');
		this.currentRow++;
	}

	static outCache() {
		var text = this.cache.join('');
		this.addText(text);
		this.cache = [];
	}

	static clear() {
		this._c.innerHTML = '';
		this.currentRow = 0;		
	}

	static getCurrentInput() {
		if (this.inputMode == this.MODE_INPUT)
			return new Tag(Console._c.lastChild).text().substring(1);

		if (this.inputMode == this.MODE_SELECT)
			return this.selected;
	}

	static replaceInput(text) {
		this._c.lastChild.innerHTML = ' ' + text;
		var tag = new Tag(this._c.lastChild);
		this.carret.setRange(tag, tag.len());
	}

	/**
	 * @var table - двумерный массив: строки (rows), в каждой строке колонки, значения - строковые (string) данные
	 * @var char - символ, добавлением которого строковые данные будут выравнены
	 * */
	static normalizeTable(table, ch = ' ') {
		var maxes = [],
			columnsCount;
		for (var i in table) {
			columnsCount = table[i].len;
			break;
		}

		for (var key in table) {
			var row = table[key];
			for (var i in row) {
				var text = row[i];
				if (!lx.isString(text)) continue;
				if (i == columnsCount - 1) continue;
				if (maxes[i] === undefined) maxes[i] = 0;
				maxes[i] = Math.max(maxes[i], text.length);
			}
		}

		var result = [];
		for (var key in table) {
			result[key] = [];
			var row = table[key];
			for (var i in row) {
				var text = row[i];
				if (!lx.isString(text)) continue;
				if (i == columnsCount - 1) {
					result[key][i] = text;
					continue;
				}
				var l = maxes[i] - text.length;
				result[key][i] = text + ch.lxRepeat(l);
			}
		}
		return result;
	}

	static checkCarret() {
		if (!this.carret.isActive()) return;

		this.carret.reset();
		if (this.carret.anchor.tag.getAttribute('lxwc-type') == 'loc' || this.carret.anchor.tag.getAttribute('lxwc-row') != this.currentRow) {
			var tag = new Tag(this._c.lastChild);
			if (tag.tag.localName == 'br') {
				this.addText('<span class="'+this.commandCss+'" lxwc-type="command" lxwc-row="'+this.currentRow+'"> </span>');
				this.consoleBox.scrollTo( this._c.scrollHeight );
				tag = new Tag(this._c.lastChild);
			}
			this.carret.setRange(tag, tag.len());
		}
	}

	static setCallback(key, callback) {
		this.callbackMap[key] = callback;
	}
	
	static onEnter(e) {
		lx.preventDefault(e);
		this.command = this.getCurrentInput();
		this.dropInputMode();
		this.addText('<br>');
		this.carret.dropRange();
		this.currentRow++;
	}

	static onSelected(e, index) {
		this.selected = index;
		this.onEnter(e);
		if (this.callbackMap[e.key]) {
			lx.preventDefault(e);
			var callback = this.callbackMap[e.key];
			callback[1].call(callback[0]);
		}
	}

	static onKeydown(e) {
		if (Console.commandPressed) {
			lx.preventDefault(e);
			return;
		}

		Console.carret.reset();

		if (e.key == 'Home') {
			Console.carret.setRange(Console.carret.anchor, 1);
			lx.preventDefault(e);
		}

		if ((e.key == 'Backspace' || e.key == 'ArrowLeft')
			&& Console.carret.anchor.pre().tag.getAttribute('lxwc-type') == 'loc' && Console.carret.anchorOffset == 1
		) {
			lx.preventDefault(e);
		}

		if (e.key == 'Enter')
			Console.onEnter(e);

		if (Console.callbackMap[e.key]) {
			lx.preventDefault(e);
			var callback = Console.callbackMap[e.key];
			callback[1].call(callback[0]);
		}

		if (e.key == 'ArrowLeft' || e.key == 'ArrowRight' || e.key == 'ArrowUp' || e.key == 'ArrowDown' || e.key == 'Enter') {
			Console.commandPressed = true;
		}
	}

	static onKeyup(e) {
		if (e.key == 'ArrowLeft' || e.key == 'ArrowRight' || e.key == 'ArrowUp' || e.key == 'ArrowDown' || e.key == 'Enter') {
			Console.commandPressed = false;
		}
	}

	static addText(text) {
		this._c.innerHTML += text;
		this.actualizeScroll();
	}
	
	static actualizeScroll() {
		this.consoleBox.scrollTo(this._c.scrollHeight);
	}

	static dropInputMode() {
		if (this.inputMode == this.MODE_SELECT) {
			this.selector.finish();
			this.selector = null;
			this.selected = null;

			this.consoleBox.on('keydown', Console.onKeydown);
			this.consoleBox.on('keyup', Console.onKeyup);
		}
		
		this.inputMode = null;
	}
}
