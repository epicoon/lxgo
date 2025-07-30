// @lx:namespace lx;
class BitMap {
	constructor(x = 0, y = 0) {
		this.x = x;
		this.y = y;
		if (y) this.reset();
		else this.map = [];
	}

	toString() {
		var arr = [];
		this.map.forEach(a=>arr.push(a.toString()));
		return arr.join(lx.EOL);
	}

	static createFromString(str) {
		var arr = str.split(/\s+/);
		var map = new this(arr[0].length, arr.length);
		for (var i=0, l=arr.length; i<l; i++) {
			var line = arr[i];
			for (var j=0, ll=line.length; j<ll; j++)
				if (+line[j]) map.setBit(j, i);
		}
		return map;
	}

	reset() {
		this.map = new Array(this.y);
		for (let i=0; i<this.y; i++)
			this.map[i] = new lx.BitLine(this.x);
	}

	fullReset() {
		this.y = 0;
		this.map = [];
	}

	setX(amt) {
		if (this.x == amt) return;
		this.map.forEach(a=>a.setLen(amt));
		this.x = amt;
	}

	addX() {
		this.setX(this.x + 1);
	}

	dropX() {
		this.setX(this.x - 1);
	}

	setY(amt) {
		if (this.y == amt) return;
		if (this.y > amt) this.map = this.map.slice(0, amt);
		else {
			var len = this.map.len;
			for (var i=len; i<amt; i++) this.map.push(new lx.BitLine(this.x));
		}
		this.y = amt;
	}

	addY() {
		this.setY(this.y + 1);
	}

	dropY() {
		this.setY(this.y - 1);
	}

	setBit(x, y) {
		if (x >= this.x || y >= this.y) return;
		this.map[y].setBit(x);
	}

	unsetBit(x, y) {
		if (x >= this.x || y >= this.y) return;
		this.map[y].unsetBit(x);
	}

	getBit(x, y) {
		if (x >= this.x || y >= this.y) return null;
		return this.map[y].getBit(x);
	}

	getLine(y) {
		if (y >= this.y) return null;
		return this.map[y];
	}

	slice(shift, amt) {
		if (!amt) return null;
		if (shift + amt > this.y) {
			amt = this.y - shift;
		}
		if (amt < 1) return null;
		var result = new lx.BitMap();
		result.x = this.x;
		result.y = amt;
		result.map = this.map.slice(shift, amt);
		return result;
	}

	findSpace(w, h) {
		if (h > this.y || !h) return false;

		var rowIndex = 0;
		while (true) {
			if (rowIndex + h > this.y) return false;
			var row = this.map[rowIndex];
			var slice = this.slice(rowIndex + 1, h - 1);
			var projection = slice ? row.project(slice.map) : row;
			var shift = projection.findSpace(w);
			if (shift !== false) return [shift, rowIndex];
			rowIndex++;
		}
	}

	setSpace(x, y, w, h) {
		if (lx.isArray(x)) {
			this.setSpace(x[0], x[1], x[2], x[3]);
			return;
		}
		if (x + w > this.x) w = this.x - x;
		if (w < 1) return;
		if (y + h > this.y) h = this.y - y;
		if (h < 1) return;

		for (var i=y, l=y+h; i<l; i++)
			this.map[i].setBit(x, w);
	}

	unsetSpace(x, y, w, h) {
		if (lx.isArray(x)) {
			this.unsetSpace(x[0], x[1], x[2], x[3]);
			return;
		}
		if (x + w > this.x) w = this.x - x;
		if (w < 1) return;
		if (y + h > this.y) h = this.y - y;
		if (h < 1) return;

		for (var i=y, l=y+h; i<l; i++)
			this.map[i].unsetBit(x, w);
	}
}
