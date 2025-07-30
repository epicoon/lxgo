lx.Geom = {
	directionByGeom: function(geom) {
		if (geom == lx.LEFT || geom == lx.CENTER || geom == lx.RIGHT) return lx.HORIZONTAL;
		if (geom == lx.TOP || geom == lx.MIDDLE || geom == lx.BOTTOM) return lx.VERTICAL;
		return false;
	},

	geomConst: function(name) {
		switch (name) {
			case 'left':   return lx.LEFT;
			case 'width': return lx.WIDTH;
			case 'right':  return lx.RIGHT;
			case 'top':    return lx.TOP;
			case 'height': return lx.HEIGHT;
			case 'bottom': return lx.BOTTOM;
		}
	},

	geomName: function(val) {
		switch (val) {
			case lx.LEFT :  return 'left';
			case lx.WIDTH: return 'width';
			case lx.RIGHT:  return 'right';
			case lx.TOP:    return 'top';
			case lx.HEIGHT: return 'height';
			case lx.BOTTOM: return 'bottom';
		}
	},

	alignConst: function(name) {
		switch (name) {
			case 'left':   return lx.LEFT;
			case 'center': return lx.CENTER;
			case 'right':  return lx.RIGHT;
			case 'top':    return lx.TOP;
			case 'middle': return lx.MIDDLE;
			case 'bottom': return lx.BOTTOM;
		}
	},

	alignName: function(val) {
		switch (val) {
			case lx.LEFT :  return 'left';
			case lx.CENTER: return 'center';
			case lx.RIGHT:  return 'right';
			case lx.TOP:    return 'top';
			case lx.MIDDLE: return 'middle';
			case lx.BOTTOM: return 'bottom';
		}
	},

	splitGeomValue: function(val) {
		if (lx.isNumber(val)) return [+val, ''];
		var num = parseFloat(val),
			f = val.split(num)[1];
		return [+num, f];
	},

	calculate: function(op, ...args) {
		let baseSplitted = this.splitGeomValue(args[0]),
			result = baseSplitted[0],
			units = baseSplitted[1];

		for (let i=1, l=args.len; i<l; i++) {
			let splitted = this.splitGeomValue(args[i]);
			if (splitted[1] !== units)
				return NaN;

			result += splitted[0];
		}
		return result + units;
	},

	/**
	 * Подгоняем x и y в шаблон размером xC на yC
	 * */
	scaleBar: function(xC, yC, x, y) {
		var result = [];
		var k = x/y;
		if (k > xC/yC) {
			result.push( xC );
			result.push( xC/k );
		} else {
			result.push( yC*k );
			result.push( yC );
		}
		return result;
	}
};
