// @lx:namespace lx;
class BitLine {
	// @lx:const BASIS = 32;
	// @lx:const BIT = [1, 2, 4, 8, 16, 32, 64, 128, 256, 512, 1024, 2048, 4096, 8192, 16384, 32768, 65536, 131072, 262144, 524288, 1048576, 2097152, 4194304, 8388608, 16777216, 33554432, 67108864, 134217728, 268435456, 536870912, 1073741824, 2147483648];

	constructor(len) {
		this.len = len || lx(STATIC).BASIS;
		this.innerLen = Math.floor((this.len - 1) / lx(STATIC).BASIS) + 1;
		this.map = new Array(this.innerLen);
		for (let i=0; i<this.innerLen; i++)
			this.map[i]=0
	}

	setLen(len) {
		if (len == this.len) return;

		this.len = len || lx(STATIC).BASIS;
		this.innerLen = Math.floor((this.len - 1) / lx(STATIC).BASIS) + 1;
		if (this.innerLen > this.map.len) {
			var len = this.map.len;
			for (var i=len; i<this.innerLen; i++) this.map.push(0);
			return;
		}
		if (this.innerLen < this.map.len) {
			this.map = this.map.slice(0, this.innerLen);
		}
		for (var i=this.len, l=this.innerLen*lx(STATIC).BASIS; i<l; i++)
			this.unsetBit(i);
	}

	setBit(i, amt = 1) {
		var innerIndex = Math.floor(i / lx(STATIC).BASIS);
		var index = i % lx(STATIC).BASIS;

		for (var k=0; k<amt; k++) {
			this.map[innerIndex] = this.map[innerIndex] | lx(STATIC).BIT[index];
			index++;
			if (index == lx(STATIC).BASIS) {
				innerIndex++;
				if (innerIndex == this.map.len) break;
				index = 0;
			}
		}
	}

	unsetBit(i, amt = 1) {
		var innerIndex = Math.floor(i / lx(STATIC).BASIS);
		var index = i % lx(STATIC).BASIS;

		for (var k=0; k<amt; k++) {
			this.map[innerIndex] = this.map[innerIndex] & ~lx(STATIC).BIT[index];
			index++;
			if (index == lx(STATIC).BASIS) {
				innerIndex++;
				if (innerIndex == this.map.len) break;
				index = 0;
			}
		}
	}

	getBit(i) {
		var innerIndex = Math.floor(i / lx(STATIC).BASIS);
		var index = i % lx(STATIC).BASIS;
		return +!!(this.map[innerIndex] & lx(STATIC).BIT[index]);
	}

	clone() {
		var result = new lx.BitLine(this.len);
		this.map.forEach((a, i)=>result.map[i] = a);
		return result;
	}

	copy(line) {
		this.setLen(line.len);
		this.map.forEach((a, i)=>this.map[i]=line.map[i]);
	}

	project(line) {
		var result = this.clone();
		if (lx.isArray(line)) {
			line.forEach(item=>{
				result.setLen(Math.max(this.len, item.len));
				result.map.forEach((a, i)=>result.map[i]=a|item.map[i]);
			});
		} else {
			result.setLen(Math.max(this.len, line.len));
			result.map.forEach((a, i)=>result.map[i]=a|line.map[i]);
		}
		return result;
	}

	findSpace(size) {
		var start = null;
		var sum = 0;
		for (var i=0, l=this.len; i<l; i++) {
			var val = this.getBit(i);
			if (val) {
				start = null;
				continue;
			}

			if (start === null) {
				start = i;
				sum = 1;
			} else sum++;

			if (sum == size) return start;			
		}

		return false;
	}

	toString() {
		var result = '';
		for (var i=0, l=this.len; i<l; i++) result += this.getBit(i);
		return result;
	}
}
