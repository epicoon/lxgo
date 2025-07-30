#lx:public;

class Tag {
	constructor(tag) {
		this._tag = tag;
	}

	get tag() { return this._tag; }

	text() {
		if (this.tag.innerHTML == '<br>') return String.fromCharCode(13);
		return this.tag.innerHTML;
	}
	
	len() {
		// if (this.text() == String.fromCharCode(13)) return 0;
		// if (this.text() == '&lt;') return 1;
		// if (this.text() == '&gt;') return 1;
		// if (this.text() == '&amp;') return 1;
		// if (this.text() == '&amp;&amp;') return 2;
		return this.text().length;
	}
	
	name() {
		return this.tag.getAttribute('name');
	}

	equal(s) {
		if (s === null) return false;
		return (this.tag === s.tag);
	}

	checkCaret(offset) {
		var r = cr.range();
		if (!r.isCaret()) return false;

		// если оффсет не передан, вернет оффсет, если он есть на этом спане, иначе false
		if (offset === undefined) {
			if (this.equal(r.anchor)) return r.anchorOffset;
			var pre = this.pre();
			if (!pre) return false;
			if (pre.equal(r.anchor) && pre.len() == r.anchorOffset) return 0;
		}

		// если оффсет передан, проверит именно эту позицию
		if (!offset) {
			var pre = this.pre();
			if (!pre) return false;
			if (pre.equal(r.anchor) && pre.len() == r.anchorOffset) return true;
		}
		if (!this.equal(r.anchor)) return false;
		if (offset == r.anchorOffset) return true;
		return false;
	}

	next() {
		var next = this.tag.nextElementSibling;
		if (!next) return null;
		return new this.constructor(next);
	}

	pre() {
		var pre = this.tag.previousElementSibling;
		if (!pre) return null;
		return new this.constructor(pre);
	}
}
