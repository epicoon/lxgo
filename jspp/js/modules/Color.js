// @lx:module lx.Color;

// @lx:namespace lx;
class Color {
	constructor(color) {
		let config = __parseConfig(color);
		this.max = lx.getFirstDefined(config.max, 255);
		this._R = lx.getFirstDefined(config.r, 0);
		this._G = lx.getFirstDefined(config.g, 0);
		this._B = lx.getFirstDefined(config.b, 0);
		this.alpha = lx.getFirstDefined(config.a, 1);
		this.HSL = null;
	}

	init(color) {
		let config = __parseConfig(color),
			max = config.max;

		if (config.a !== undefined)
			this.alpha = config.a;

		function setC(self, field, val) {
			if (self.max == max) { self[field] = val; return; }
			self[field] = (val * self.max) / max;
			if (self.max > 1) self[field] = Math.round(self[field]);
			else self[field] = Math.round((self[field] + Number.EPSILON) * 100) / 100;
		}
		if (config.r !== undefined) setC(this, '_R', config.r);
		if (config.g !== undefined) setC(this, '_G', config.g);
		if (config.b !== undefined) setC(this, '_B', config.b);
		this.HSL = null;
	}

	/**
	 * @param color {String|Number|Array|Object|lx.Color}
	 * @return {Number}
	 */
	static castToNumber(color) {
		return (new lx.Color(color)).toNumber();
	}

	/**
	 * @param color {String|Number|Array|Object|lx.Color}
	 * @param max {Number} format - 1, 100, 255
	 * @return {Array} [R, G, B]
	 */
	static castToRgb(color, max= 255) {
		return (new lx.Color(color)).toRGB(max);
	}

	/**
	 * @param color {String|Number|Array|Object|lx.Color}
	 * @return {String}
	 */
	static castToString(color) {
		return (new lx.Color(color)).toString();
	}

	copy(color) {
		this._R = color._R;
		this._G = color._G;
		this._B = color._B;
		this.alpha = color.alpha;
		this.max = color.max;
		if (color.HSL)
			this.HSL = [color.HSL[0], color.HSL[1], color.HSL[2]];
		return this;
	}

	clone() {
		return new lx.Color(this);
	}

	getHSL() {
		if (!this.HSL) this.HSL = __RGBtoHSL(this.toRGB(1));
		return this.HSL;
	}

	getHue() {
		return this.getHSL()[0];
	}

	getSaturation() {
		return this.getHSL()[1];
	}

	getLightness() {
		return this.getHSL()[2];
	}

	getLuma() {
		let RGB = this.toRGB(1),
			r = RGB[0],
			g = RGB[1],
			b = RGB[2];
		r = (r <= 0.03928) ? r / 12.92 : Math.pow(((r + 0.055) / 1.055), 2.4);
		g = (g <= 0.03928) ? g / 12.92 : Math.pow(((g + 0.055) / 1.055), 2.4);
		b = (b <= 0.03928) ? b / 12.92 : Math.pow(((b + 0.055) / 1.055), 2.4);
		return (0.2126 * r) + (0.7152 * g) + (0.0722 * b);
	}

	setHue(value) {
		let HSL = this.getHSL();
		HSL[0] = value;
		this.setHSL(HSL);
	}

	setSaturation(value) {
		let HSL = this.getHSL();
		HSL[1] = value;
		this.setHSL(HSL);
	}

	setLightness(value) {
		let HSL = this.getHSL();
		HSL[2] = value;
		this.setHSL(HSL);
	}

	setHSL(hsl) {
		let RGB = __HSLtoRGB(hsl);
		this.init({r: RGB[0], g: RGB[1], b: RGB[2], max: 1});
		this.HSL = hsl;
		return this;
	}

	get R() {
		return this._R;
	}

	get G() {
		return this._G;
	}

	get B() {
		return this._B;
	}

	set R(value) {
		this._R = value;
		this.HSL = null;
	}

	set G(value) {
		this._G = value;
		this.HSL = null;
	}

	set B(value) {
		this._B = value;
		this.HSL = null;
	}

	darken(persent) {
		let hsl = this.getHSL();
		hsl[2] = __clamp(hsl[2] - persent, 100);
		this.setHSL(hsl);
		return this;
	}

	lighten(persent) {
		let hsl = this.getHSL();
		hsl[2] = __clamp(hsl[2] + persent, 100);
		this.setHSL(hsl);
		return this;
	}

	saturate(persent) {
		let hsl = this.getHSL();
		hsl[1] = __clamp(hsl[2] + persent, 100);
		this.setHSL(hsl);
		return this;
	}

	desaturate(persent) {
		let hsl = this.getHSL();
		hsl[1] = __clamp(hsl[2] - persent, 100);
		this.setHSL(hsl);
		return this;
	}

	spin(angle) {
		let hsl = this.getHSL();
        hsl[0] += angle % 360;
        if (hsl[0] < 0) hsl[0] += 360;
        else if (hsl[0] > 360) hsl[0] -= 360;
		this.setHSL(hsl);
		return this;
	}

	fade(alpha) {
		this.alpha = alpha / 100;
		return this;
	}

	fadeIn(delta) {
		this.alpha = __clamp(this.alpha - delta/100);
		return this;
	}

	fadeOut(delta) {
		this.alpha = __clamp(this.alpha + delta/100);
		return this;
	}

	mix(color, weight = 50) {
		color = new lx.Color(color);
		let w = weight / 50 - 1,
			selfA = this.alpha,
			a = this.alpha - color.alpha,
			w1 = ((w * a == -1 ? w : (w + a)/(1 + w * a)) + 1) / 2.0,
			w2 = 1.0 - w1,
			selfRGB = this.toRGB(255),
			RGB = color.toRGB(255),
			r, g, b;

		r = Math.round(w1 * selfRGB[0] + w2 * RGB[0]);
		g = Math.round(w1 * selfRGB[1] + w2 * RGB[1]);
		b = Math.round(w1 * selfRGB[2] + w2 * RGB[2]);
		this.init({r, g, b, max: 255});

		if (this.alpha != 1 || color.alpha != 1)
			this.alpha = selfA * weight + color.alpha * (weight - 1);
		return this;
	}

	contrast(dark, light, threshold = 0.43) {
		dark = new lx.Color(dark || [0, 0, 0]);
		light = new lx.Color(light || [255, 255, 255]);
		if (dark.getLuma() > light.getLuma()) {
			let temp = light;
			light = dark;
			dark = temp;
		}
		if (this.getLuma() * this.alpha < threshold)
			this.copy(light);
		else
			this.copy(dark);
		return this;
	}

	toRGB(max = 255) {
		if (this.max == max)
			return [this._R, this._G, this._B];

		return [(this._R * max) / this.max, (this._G * max) / this.max, (this._B * max) / this.max];
	}

	[Symbol.toPrimitive](hint) {
		switch (hint) {
			case 'number':
				return this.toNumber();
			case 'string':
			default:
				return this.toString();
		}
	}

	toNumber() {
		let RGB = __toPreStr(this);
		return +('0x' + RGB[0] + RGB[1] + RGB[2]);
	}

	toString() {
		if (this.alpha != 1) {
			let RGB = this.toRGB(1);
			return 'rgba('
				+ Math.round((RGB[0] + Number.EPSILON) * 100) / 100
				+ ', '
				+ Math.round((RGB[1] + Number.EPSILON) * 100) / 100
				+ ', '
				+ Math.round((RGB[2] + Number.EPSILON) * 100) / 100
				+ ', '
				+ this.alpha
				+ ')';
		}

		let RGB = __toPreStr(this);
		return '#' + RGB[0] + RGB[1] + RGB[2];
	}

	toStringWithAlfa() {
		let RGB = this.toRGB(1);
		return 'rgba('
			+ Math.round((RGB[0] + Number.EPSILON) * 100) / 100
			+ ', '
			+ Math.round((RGB[1] + Number.EPSILON) * 100) / 100
			+ ', '
			+ Math.round((RGB[2] + Number.EPSILON) * 100) / 100
			+ ', '
			+ this.alpha
			+ ')';
	}

	toCssString() {
		if (this.alpha != 1) return this.toStringWithAlfa();
		return this.toString();
	}
}


/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * PRIVATE
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

function __parseConfig(color) {
	let r = undefined, g = undefined, b = undefined, a = undefined, max = undefined;

	if (lx.isNumber(color)) {
		let res = __16strToRGB(color.toString(16));
		r = res[0];
		g = res[1];
		b = res[2];
		max = 255;
	} else if (lx.isString(color)) {
		if (color[0] == '#') {
			let res = __16strToRGB(color.replace(/^#/, ''));
			r = res[0];
			g = res[1];
			b = res[2];
			max = 255;
		} else {
			let name = color.toLowerCase();
			if (name in colorsMap) {
				r = colorsMap[name][0];
				g = colorsMap[name][1];
				b = colorsMap[name][2];
				max = 255;
			} else {
				color = color.replace(' ', '');
				if (color.match(/^rgba?\((0\.\d+|\d+) *, *(0\.\d+|\d+) *, *(0\.\d+|\d+)( *, *(0\.\d+|\d+))?\)/)) {
					color = color.replace(/^rgba?\(/, '');
					color = color.replace(')', '');
					color = color.split(/ *, */);
					r = +color[0];
					g = +color[1];
					b = +color[2];
					if (color.len == 4) a = color[3];
					if (r == 0 && g == 0 && b == 0) max = 255;
					else if (r <= 1 && g <= 1 && b <= 1) max = 1;
					else max = 255;
				} else {
					r = 0;
					g = 0;
					b = 0;
					max = 255;
				}
			}
		}
	} else if (lx.isArray(color)) {
		if (color.len == 3) {
			r = color[0];
			g = color[1];
			b = color[2];
		} else if (color.len == 4) {
			r = color[0];
			g = color[1];
			b = color[2];
			a = color[3];
		}
		if (r == 0 && g == 0 && b == 0) max = 255;
		else if (r <= 1 && g <= 1 && b <= 1) max = 1;
		else max = 255;
	} else if (lx.isInstance(color, lx.Color)) {
		r = color._R;
		g = color._G;
		b = color._B;
		a = color.alpha;
		max = color.max;
	} else if (lx.isObject(color)) {
		r = color._R || color.r || color.Red || color.red || 0;
		g = color._G || color.g || color.Green || color.green || 0;
		b = color._B || color.b || color.Blue || color.blue || 0;
		a = color.alpha || color.a;
		if (color.max !== undefined) max = color.max;
		else if (r == 0 && g == 0 && b == 0) max = 255;
		else if (r <= 1 && g <= 1 && b <= 1) max = 1;
		else max = 255;
	}

	return {r, g, b, a, max};
}

function __toPreStr(color) {
	let RGB = color.toRGB(255),
		sR = RGB[0].toString(16),
		sG = RGB[1].toString(16),
		sB = RGB[2].toString(16);
	if (sR.length < 2) sR = '0' + sR;
	if (sG.length < 2) sG = '0' + sG;
	if (sB.length < 2) sB = '0' + sB;
	return [sR, sG, sB];
}

function __16strToRGB(str) {
	let r = +('0x'+str.substr(0, 2)),
		g = +('0x'+str.substr(2, 2)),
		b = +('0x'+str.substr(4, 2));
	return [r, g, b];
}

function __clamp(val, max = 1, min = 0) {
	return Math.min(max, Math.max(min, val));
}

function __RGBtoHSL(rgb) {
	let r = rgb[0],
		g = rgb[1],
		b = rgb[2],
		min = Math.min(r, g, b),
		max = Math.max(r, g, b),
		h, s, l = (min + max) / 2;

	if (min == max) s = h = 0;
	else {
		if (l < 0.5) {
			s = (max - min) / (max + min);
		} else {
			s = (max - min) / (2.0 - max - min);
		}
		if (r == max) h = (g - b) / (max - min);
		else if (g == max) h = 2.0 + (b - r) / (max - min);
		else if (b == max) h = 4.0 + (r - g) / (max - min);
	}

	return [
		(h < 0 ? h + 6 : h) * 60,
		s * 100,
		l * 100,
	];
}

function __HSLtoRGB(hls) {
	function helper(comp, temp1, temp2) {
		if (comp < 0) comp += 1.0;
		else if (comp > 1) comp -= 1.0;

		if (6 * comp < 1) return temp1 + (temp2 - temp1) * 6 * comp;
		if (2 * comp < 1) return temp2;
		if (3 * comp < 2) return temp1 + (temp2 - temp1)*((2/3) - comp) * 6;
		return temp1;
	}

	let h = hls[0] / 360,
		s = hls[1] / 100,
		l = hls[2] / 100,
		r, g, b;
	if (s == 0) r = g = b = l;
	else {
		let temp2 = l < 0.5 ? l * (1.0 + s) : l + s - l * s,
			temp1 = 2.0 * l - temp2;
		r = Math.round(helper(h + 1/3, temp1, temp2));
		g = Math.round(helper(h, temp1, temp2));
		b = Math.round(helper(h - 1/3, temp1, temp2));
	}

	return [r, g, b];
}

const colorsMap = {
	indianred: [205, 92, 92],
	lightcoral: [240, 128, 128],
	salmon: [250, 128, 114],
	darksalmon: [233, 150, 122],
	lightsalmon: [255, 160, 122],
	crimson: [220, 20, 60],
	red: [255, 0, 0],
	firebrick: [178, 34, 34],
	darkred: [139, 0, 0],
	pink: [255, 192, 203],
	lightpink: [255, 182, 193],
	hotpink: [255, 105, 180],
	deeppink: [255, 20, 147],
	mediumvioletred: [199, 21, 133],
	palevioletred: [219, 112, 147],
	lightsalmon: [255, 160, 122],
	coral: [255, 127, 80],
	tomato: [255, 99, 71],
	orangered: [255, 69, 0],
	darkorange: [255, 140, 0],
	orange: [255, 165, 0],
	gold: [255, 215, 0],
	yellow: [255, 255, 0],
	lightyellow: [255, 255, 224],
	lemonchiffon: [255, 250, 205],
	lightgoldenrodyellow: [250, 250, 210],
	papayawhip: [255, 239, 213],
	moccasin: [255, 228, 181],
	peachpuff: [255, 218, 185],
	palegoldenrod: [238, 232, 170],
	khaki: [240, 230, 140],
	darkkhaki: [189, 183, 107],
	lavender: [230, 230, 250],
	thistle: [216, 191, 216],
	plum: [221, 160, 221],
	violet: [238, 130, 238],
	orchid: [218, 112, 214],
	fuchsia: [255, 0, 255],
	magenta: [255, 0, 255],
	mediumorchid: [186, 85, 211],
	mediumpurple: [147, 112, 219],
	blueviolet: [138, 43, 226],
	darkviolet: [148, 0, 211],
	darkorchid: [153, 50, 204],
	darkmagenta: [139, 0, 139],
	purple: [128, 0, 128],
	indigo: [75, 0, 130],
	slateblue: [106, 90, 205],
	darkslateblue: [72, 61, 139],
	cornsilk: [255, 248, 220],
	blanchedalmond: [255, 235, 205],
	bisque: [255, 228, 196],
	navajowhite: [255, 222, 173],
	wheat: [245, 222, 179],
	burlywood: [222, 184, 135],
	tan: [210, 180, 140],
	rosybrown: [188, 143, 143],
	sandybrown: [244, 164, 96],
	goldenrod: [218, 165, 32],
	darkgoldenrod: [184, 134, 11],
	peru: [205, 133, 63],
	chocolate: [210, 105, 30],
	saddlebrown: [139, 69, 19],
	sienna: [160, 82, 45],
	brown: [165, 42, 42],
	maroon: [128, 0, 0],
	black: [0, 0, 0],
	gray: [128, 128, 128],
	silver: [192, 192, 192],
	white: [255, 255, 255],
	fuchsia: [255, 0, 255],
	purple: [128, 0, 128],
	red: [255, 0, 0],
	maroon: [128, 0, 0],
	yellow: [255, 255, 0],
	olive: [128, 128, 0],
	lime: [0, 255, 0],
	green: [0, 128, 0],
	aqua: [0, 255, 255],
	teal: [0, 128, 128],
	blue: [0, 0, 255],
	navy: [0, 0, 128],
	greenyellow: [173, 255, 47],
	chartreuse: [127, 255, 0],
	lawngreen: [124, 252, 0],
	lime: [0, 255, 0],
	limegreen: [50, 205, 50],
	palegreen: [152, 251, 152],
	lightgreen: [144, 238, 144],
	mediumspringgreen: [0, 250, 154],
	springgreen: [0, 255, 127],
	mediumseagreen: [60, 179, 113],
	seagreen: [46, 139, 87],
	forestgreen: [34, 139, 34],
	green: [0, 128, 0],
	darkgreen: [0, 100, 0],
	yellowgreen: [154, 205, 50],
	olivedrab: [107, 142, 35],
	olive: [128, 128, 0],
	darkolivegreen: [85, 107, 47],
	mediumaquamarine: [102, 205, 170],
	darkseagreen: [143, 188, 143],
	lightseagreen: [32, 178, 170],
	darkcyan: [0, 139, 139],
	teal: [0, 128, 128],
	aqua: [0, 255, 255],
	cyan: [0, 255, 255],
	lightcyan: [224, 255, 255],
	paleturquoise: [175, 238, 238],
	aquamarine: [127, 255, 212],
	turquoise: [64, 224, 208],
	mediumturquoise: [72, 209, 204],
	darkturquoise: [0, 206, 209],
	cadetblue: [95, 158, 160],
	steelblue: [70, 130, 180],
	lightsteelblue: [176, 196, 222],
	powderblue: [176, 224, 230],
	lightblue: [173, 216, 230],
	skyblue: [135, 206, 235],
	lightskyblue: [135, 206, 250],
	deepskyblue: [0, 191, 255],
	dodgerblue: [30, 144, 255],
	cornflowerblue: [100, 149, 237],
	mediumslateblue: [123, 104, 238],
	royalblue: [65, 105, 225],
	blue: [0, 0, 255],
	mediumblue: [0, 0, 205],
	darkblue: [0, 0, 139],
	navy: [0, 0, 128],
	midnightblue: [25, 25, 112],
	white: [255, 255, 255],
	snow: [255, 250, 250],
	honeydew: [240, 255, 240],
	mintcream: [245, 255, 250],
	azure: [240, 255, 255],
	aliceblue: [240, 248, 255],
	ghostwhite: [248, 248, 255],
	whitesmoke: [245, 245, 245],
	seashell: [255, 245, 238],
	beige: [245, 245, 220],
	oldlace: [253, 245, 230],
	floralwhite: [255, 250, 240],
	ivory: [255, 255, 240],
	antiquewhite: [250, 235, 215],
	linen: [250, 240, 230],
	lavenderblush: [255, 240, 245],
	mistyrose: [255, 228, 225],
	gainsboro: [220, 220, 220],
	lightgrey: [211, 211, 211],
	lightgray: [211, 211, 211],
	silver: [192, 192, 192],
	darkgray: [169, 169, 169],
	darkgrey: [169, 169, 169],
	gray: [128, 128, 128],
	grey: [128, 128, 128],
	dimgray: [105, 105, 105],
	dimgrey: [105, 105, 105],
	lightslategray: [119, 136, 153],
	lightslategrey: [119, 136, 153],
	slategray: [112, 128, 144],
	slategrey: [112, 128, 144],
	darkslategray: [47, 79, 79],
	darkslategrey: [47, 79, 79],
	black: [0, 0, 0],
};
