/* Collection *//**********************************************
    len
    isEmpty()
    clear()
    getEmptyInstance()
    at(k)
    to(k)
    set(i, val)
    contains(obj)
    first()
    last()
    current(value)
    next()
    prev()
    toCopy()
    add(*arguments*)
    addCopy(*arguments*)
    addList(list)
    flat(deep)
    construct(*arguments*)
    indexOf(el)
    remove(el)
    removeAt(k)
    pop()
    sub(k, amt)
    forEach(func)
    forEachRevert(func)
    stop()
    toArray()
***************************************************************/

// @lx:namespace lx;
class Collection extends lx.Object {
	constructor(...args) {
		super();

		this.actPart = null;
		this.actI = null;
		this.actPartI = null;

		this.isCopy = false;
		this.elements = [];
		this.map = [];

		this.reversIteration = false;
		this.stopFlag = false;
		this.repeat = true;

		if (args.length) this.add.apply(this, args);
	}

	static cast(obj) {
		if (obj === undefined || obj === null) return new this();
		if (lx.isInstance(obj, this)) return obj;
		return new this(obj);
	}

	isEmpty() {
		if (this.isCopy) return this.elements.length == 0;
		for (var i=0, l=this.map.length; i<l; i++)
			if (this.map[i].length)
				return false;
		return true;
	}

	clear() {
		this.actPart = null;
		this.actI = null;
		this.actPartI = null;

		this.isCopy = false;
		this.elements = [];
		this.map = [];
	}

	getEmptyInstance() {
		return new lx.Collection();
	}

	at(k) {
		if (this.reversIteration) k = this.len - 1 - k;
		if (!this.to(k)) return null;
		return this.current();
	}

	/**
	 * k - might to be index (number) or element (instead of number)
	 * k - reset iterator position if null
	 */
	to(k) {
		if (k === null) {
			this.actPart = null;
			return this;
		}

		if (lx.isNumber(k)) {
			if (k >= this.len) return false;

			if (this.isCopy) {
				this.actPart = this.elements;
				this.actPartI = k;
			} else {
				for (var i=0, l=this.map.length; i<l; i++) {
					if (this.map[i].length > k) {
						this.actPart = this.map[i];
						this.actI = i;
						this.actPartI = k;
						break;
					} else k -= this.map[i].length;
				}
			}
		} else {
			var match = false;
			this.first();
			while (!match && this.current()) {
				if (this.current() === k) match = true;
				else this.next();
			}
			if (!match) return false;
		}

		return this;
	}

	set(i, val) {
		this.to(i);
		this.current(val);
		return this;
	}

	contains(obj) {
		this.cachePosition();
		var match = false;
		var curr = this.first();
		while (curr && !match)
			if (curr === obj) match = true;
			else curr = this.next();
		this.loadPosition();
		return match;
	}

	current(value) {
		if (this.actPart === null)
			return null;
		if (value === undefined)
			return this.actPart[this.actPartI];
		this.actPart[this.actPartI] = value;
		return this;
	}

	first() {
		if (this.reversIteration) return _last(this);
		return _first(this);
	}

	last() {
		if (this.reversIteration) return _first(this);
		return _last(this);
	}

	next() {
		if (this.reversIteration) return _prev(this);
		return _next(this);
	}

	prev() {
		if (this.reversIteration) return _next(this);
		return _prev(this);
	}

	toCopy() {
		if (this.isCopy) return this;
		var iter = 0;
		for (var i=0, l=this.map.len; i<l; i++) {
			if (this.actPart && i < this.actI) iter += this.map[i].len;
			else if (this.actPart && i == this.actI) iter += this.actPartI;
			for (var j=0, ll=this.map[i].len; j<ll; j++) {
				this.elements.push(this.map[i][j]);
			}
		}
		this.map = [];
		this.isCopy = true;
		if (this.actPart) {
			this.actPart = this.elements;
			this.actPartI = iter;
		}
		return this;
	}

	add() {
		if ( arguments == undefined ) return this;
		if (this.isCopy) return this.addCopy.apply(this, arguments);

		for (var i=0, l=arguments.length; i<l; i++) {
			var arg = arguments[i];

			if (arg === null) continue;

			if (lx.isArray(arg)) {
				if (!arg.length) continue;
				this.map.push(arg);
			} else if ( arg.lxClassName() == 'Collection' ) {
				if (arg.isCopy) this.add(arg.elements);
				else for (var j=0, ll=arg.map.length; j<ll; j++)
					this.add( arg.map[j] );
			} else {
				if ( this.map.len && this.map.lxLast().singles ) {
					this.map.lxLast().push(arg);
				} else {
					var arr = [arg];
					Object.defineProperty(arr, "singles", { get: function() { return true; } });
					this.map.push(arr);
				}
			}
		}

		return this;
	}

	addCopy() {
		this.toCopy();
		if ( arguments == undefined ) return this;

		for (var i=0, l=arguments.length; i<l; i++) {
			var arg = arguments[i];
			if (arg === null) continue;

			if (lx.isArray(arg)) {
				for (var j=0, ll=arg.length; j<ll; j++)
					this.elements.push( arg[j] );
			} else if ( arg.lxClassName() == 'Collection' ) {
				arg.first();
				while (arg.current()) {
					this.elements.push( arg.current() );
					arg.next();
				}
			} else this.elements.push( arg );
		}

		return this;
	}

	addList(list, func) {
		for (var i in list) {
			if (func) func(list[i], i);
			this.add(list[i]);
		}
		return this;
	}

	flat(deep) {
		// To change inner structure is available only for copy mode
		this.toCopy();
		var arr = [];

		function rec(tempArr, counter) {
			for (var i=0,l=tempArr.length; i<l; i++) {
				if ((deep && (counter+1 > deep)) || !lx.isArray(tempArr[i])) {
					arr.push( tempArr[i] );
				}
				else rec(tempArr[i], counter + 1);
			}
		}
		rec(this.elements, 0);

		this.elements = arr;
		return this;
	}

	construct(/*arguments*/) {
		this.add(lx.Collection.construct.apply(null, arguments));
		return this;
	}

	indexOf(el) {
		this.toCopy();
		return this.elements.indexOf(el);
	}

	insert(index, value) {
		// To change inner structure is available only for copy mode
		this.toCopy();
		this.elements.splice(index, 0, value);
	}

	remove(el) {
		// To change inner structure is available only for copy mode
		this.toCopy();
		var index = this.elements.indexOf(el);
		if (index == -1) return false;
		return this.removeAt(index);
	}

	removeAt(k) {
		// To change inner structure is available only for copy mode
		this.toCopy();
		this.to(k);
		this.elements.splice(k, 1);
		if (this.actPartI >= this.elements.length)
			this.actPartI = this.elements.length - 1;
		return true;
	}

	pop() {
		var elem = this.last();
		if (!elem) return null;

		this.removeAt(this.len - 1);
		return elem;
	}

	sub(k, amt) {
		var c = new lx.Collection();
		if (k === undefined) return c;
		amt = amt || 1;

		this.to(k);
		for (var i=0; i<amt; i++) {
			if (!this.current()) return c;
			c.add(this.current()); 
			this.next();
		}

		return c;
	}

	forEach(func) {
		this.stopFlag = false;
		this.cachePosition();
		let i = 0,
			el = this.first();
		while (el && !this.stopFlag) {
			func.call( this, el, i++ );
			el = this.next();
		}
		this.loadPosition();
		return this;
	}

	forEachRevert(func) {
		this.stopFlag = false;
		this.cachePosition();
		var i = this.len - 1,
			el = this.last();
		while (el && !this.stopFlag) {
			func.call( this, el, i-- );
			el = this.prev();
		}
		this.loadPosition();
		return this;
	}

	/**
	 * To interact each element of the collection with each element
	 */
	eachToEach(func) {
		this.stopFlag = false;
		this.cachePosition();

		var i = 0,
			el0 = this.first();
		while (el0 && !this.stopFlag) {
			this.cachePosition();
			var j = i+1,
				el1 = this.next();

			while (el1 && !this.stopFlag) {
				func.call( this, el0, el1, i, j++ );
				el1 = this.next();
			}

			this.loadPosition();
			el0 = this.next();
			i++;
		}

		this.loadPosition();
		return this;
	}

	stop() {
		this.stopFlag = true;
	}

	cachePosition() {
		if (!this.cachepos) this.cachepos = [];
		this.cachepos.push([this.actPart, this.actI, this.actPartI]);
	}

	loadPosition() {
		if (!this.cachepos) return false;
		var cache = this.cachepos.pop();
		this.actPart = cache[0];
		this.actI = cache[1];
		this.actPartI = cache[2];
		if (!this.cachepos.len) delete this.cachepos;
		return true;
	}

	toArray() {
		let list = [];
		this.forEach(elem => list.push(elem));
		return list;
	}
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * PRIVATE
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

function _first(self, value) {
	if (self.isCopy) {
		if (!self.elements.len) return null;
		self.actPart = self.elements;
		self.actPartI = 0;
	} else {
		if (!self.map.len || !self.map[0].len) return null;
		self.actPart = self.map[0];
		self.actI = 0;
		self.actPartI = 0;
	}
	if (value === undefined)
		return self.actPart[self.actPartI];
	self.actPart[self.actPartI] = value;
	return self;
}

function _last(self, value) {
	if (self.isCopy) {
		if (!self.elements.len) return null;
		self.actPart = self.elements;
		self.actPartI = self.elements.len - 1;
	} else {
		if (!self.map.len || !self.map[0].len) return null;
		self.actI = self.map.len - 1;
		self.actPart = self.map[self.actI];
		self.actPartI = self.actPart.len - 1;
	}
	if (value === undefined)
		return self.actPart[self.actPartI];
	self.actPart[self.actPartI] = value;
	return self;
}

function _next(self) {
	if (self.actPart === null) return self.first();

	self.actPartI++;
	if (self.actPart.len == self.actPartI) {
		if (self.isCopy) {
			self.actPart = null;
			return null;
		} else {
			self.actI++;
			if (self.map.len == self.actI) {
				self.actPart = null;
				return null;
			} else {
				self.actPartI = 0;
				self.actPart = self.map[self.actI];
			}
		}
	}
	return self.actPart[self.actPartI];
}

function _prev(self) {
	if (self.actPart === null) return self.last();

	self.actPartI--;
	if (self.actPartI == -1) {
		if (self.isCopy) {
			self.actPart = null;
			return null;
		} else {
			self.actI--;
			if (self.actI == -1) {
				self.actPart = null;
				return null;
			} else {
				self.actPart = self.map[self.actI];
				self.actPartI = self.actPart.len - 1;
			}
		}
	}
	return self.actPart[self.actPartI];
}

/*
 * Args structure: (constructor, count, {configurator,} arguments...)
 * - consctructor - object constructor we are creating
 * - count - count of objects to create
 * - configurator - if passed has to contain at leats one of:
 *   {
 * 		preBuild: function(i, args) {},  // for object to create - args is an array for object constructor
 *  	                                 // you can modify args and return
 * 	                                     // you can ignore passed args and return another set of args
 * 		postBuild: function(i) {}  // the context is object had been created
 *   }
 * - arguments - array for object constructor
 * Example:
 * lx.Collection().construct(
 * 	lx.Widget, 3, {
 * 		preBuild: function(i, args) {
 * 			args[0].key = 'obj' + i;
 * 			return args;
 * 		},
 * 		postBuild : function(i) {
 * 			this.text(i);
 * 		}
 * 	},
 * 	{height: 10}
 * );
 */
Object.defineProperty(lx.Collection, "construct", {
	value: function(/*arguments*/) {
		var result = this(), // lx.Collection(),
			constructor = arguments[0],
			count = arguments[1],
			configurator = {},
			pos = 2,
			args;

		if (arguments[2].preBuild || arguments[2].postBuild) {
			configurator = arguments[2];
			pos++;
		}

		if (arguments.length > pos) {
			args = new Array(arguments.length - pos);
			for (var i=0, l=args.length; i<l; i++)
				args[i] = arguments[i + pos];
		}

		for (var i=0; i<count; i++) {
			var modifArgs = args;
			if (configurator.preBuild) modifArgs = configurator.preBuild.call(null, args, i);
			var obj = constructor.apply(null, modifArgs);
			if (configurator.postBuild) configurator.postBuild.call(null, obj, i); 
			result.add(obj);
		}

		return result;
	}
});

Object.defineProperty(lx.Collection.prototype, 'len', {
	get: function() {
		if (this.isCopy) return this.elements.length;
		var len = 0;
		for (var i=0, l=this.map.length; i<l; i++)
			len += this.map[i].length;
		return len;
	}
});

Object.defineProperty(lx.Collection.prototype, 'currentIndex', {
	get: function() {
		if (this.actPart === null) return -1;

		if (this.isCopy) return this.actPartI;

		var res = 0;
		for (var i=0, l=this.actI; i<l; i++)
			res += this.map[i].length;
		return res + this.actPartI;
	}
});
