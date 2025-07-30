#lx:public;

class TextSelection {
	constructor() {
		this.anchor = null;
		this.focus = null;
		this.anchorOffset = 0;
		this.focusOffset = 0;

		this.reset();
	}
	
	isActive() {
		return (this.anchor !== null);
	}

	/**
	 * Актуализировать данные
	 * */
	reset() {
		var sel = document.getSelection();
		this.selection = sel;
		// проверка для бр-спанов
		var s0 = sel.anchorNode,
			s1 = sel.focusNode;
		if (!s0 || !s1) return;
		// if ( s0.parentElement !== cr.context.canvas ) s0 = s0.parentElement;
		// if ( s1.parentElement !== cr.context.canvas ) s1 = s1.parentElement;

		if (!s0.childNodes.length) s0 = s0.parentElement;
		if (!s1.childNodes.length) s1 = s1.parentElement;

		this.anchor = new Tag(s0);
		this.focus = new Tag(s1);
		this.anchorOffset = sel.anchorOffset;
		this.focusOffset = sel.focusOffset;
	}

	/**
	 * Если не выделена область и активна только каретка
	 * */
	isCaret() {
		return ( this.anchor.tag === this.focus.tag && this.anchorOffset == this.focusOffset );
	}

	/**
	 * Условно "правое" или "левое" выделение области:
	 * - парвое если это каретка, или якорь находится по тексту раньше фокуса
	 * - левое если фокус находится по тексту раньше якоря
	 * */
	rightSequens() {
		if (this.anchor.tag === this.focus.tag) return (this.anchorOffset <= this.focusOffset);
		if ( this.anchor.tag.offsetTop < this.focus.tag.offsetTop
		|| (this.anchor.tag.offsetTop == this.focus.tag.offsetTop && this.anchor.tag.offsetLeft < this.focus.tag.offsetLeft) )
			return true;
		return false;
	}

	/**
	 * Края выделения в виде массива:
	 * [0] - Tag, который идет раньше по тексту
	 * [1] - Tag, который идет позже по тексту
	 * [2] - смещение в первом тэге
	 * [3] - смещение во втором тэге
	 * */
	edges() {
		if ( this.rightSequens() ) return [ this.anchor, this.focus, this.anchorOffset, this.focusOffset ];
		return [ this.focus, this.anchor, this.focusOffset, this.anchorOffset ];
	}

	/**
	 * Массив тэгов, входящих в зону выделения
	 * */
	allTags() {
		if (this.anchor.tag === this.focus.tag) return [this.anchor];
		var edges = this.edges(),
			result = [];
		for (var temp = edges[0]; temp.tag !== edges[1].tag; temp = temp.next()) result.push( temp );
		result.push( edges[1] );
		return result;
	}

	/**
	 * Находится ли каретка в самом начале текста
	 * */
	caretOnStart() {
		return ((this.anchorOffset == 0 && this.anchor.pre() == null)
				|| (this.focusOffset == 0 && this.focus.pre() == null));
	}

	/**
	 * Находится ли каретка в самом конце текста
	 * //todo - нет метода Tag::len()
	 * */
	// caretOnEnd() {
	// 	function end(s, offset) { return (offset == s.len() && (s.next() === null || s.next().next() === null)); };
	// 	return ( end(this.anchor, this.anchorOffset) || end(this.focus, this.focusOffset) );
	// }

	/**
	 * Скинуть позицию
	 * */
	dropRange() {
		this.selection.removeAllRanges();
	}

	/**
	 * Получить данные о позиции
	 * */
	getRange() {
		this.reset();
		return {
			anchor: this.anchor,
			focus: this.focus,
			anchorOffset: this.anchorOffset,
			focusOffset: this.focusOffset
		};
	}

	/**
	 * Установить область выделения
	 * */
	setRange(tag, offset, tag1, offset1) {
		if (offset === undefined) {
			offset = tag.anchorOffset;
			tag1 = tag.focus;
			offset1 = tag.focusOffset;
			tag = tag.anchor;
		}

		if (offset > tag.len()) {
			offset = tag.len();
		}

		if (tag.tag === undefined || tag.tag.childNodes === undefined || tag.tag.childNodes[0] === undefined)
			this.dropRange();

		if (offset === undefined) offset = 0;

		var r = document.createRange();
		r.setStart(tag.tag.childNodes[0], offset);

		if (tag1 === undefined || (tag1 == tag && offset1 == offset)) r.collapse();
		else {
			if (tag1.tag === undefined || tag1.tag.childNodes === undefined || tag1.tag.childNodes[0] === undefined)
				this.dropRange();

			if (offset1 === undefined) offset1 = 0;
			r.setEnd(tag1.tag.childNodes[0], offset1);
		}

		this.selection.removeAllRanges();
		this.selection.addRange(r);
	}
}
