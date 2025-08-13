// @lx:module lx.ActiveBox;

// @lx:use lx.BasicCssContext;

/**
 * @widget lx.ActiveBox
 * 
 * CSS classes:
 * - lx-ActiveBox
 * - lx-ActiveBox-header
 * - lx-ActiveBox-close
 * - lx-ActiveBox-headerText
 * - lx-ActiveBox-body
 * - lx-ActiveBox-resizer
 * - lx-ActiveBox-move
 */
// @lx:namespace lx;
class ActiveBox extends lx.Box {
	//TODO - only px for now
	// @lx:const HEADER_HEIGHT = 40;
	// @lx:const INDENT = 5;

	_getContainer() {
		return lx(this)>body;
	}

	static initCss(css) {
		css.useHolder(lx.BasicCssContext);
		let shadowSize = css.preset.shadowSize + 2,
			shadowShift = Math.floor(shadowSize * 0.5);
		css.addClass('lx-ActiveBox', {
			overflow: 'hidden',
			borderRadius: css.preset.borderRadius,
			boxShadow: '0 '+shadowShift+'px '+shadowSize+'px rgba(0,0,0,0.5)',
			minWidth: '50px',
			minHeight: '75px',
			backgroundColor: css.preset.bodyBackgroundColor
		});
		css.addClass('lx-ActiveBox-header', {
			overflow: 'hidden',
			whiteSpace: 'nowrap',
			textOverflow: 'ellipsis',
			color: css.preset.textColor,
			cursor: 'move',
			borderRadius: css.preset.borderRadius,
			boxShadow: '0 0px 3px rgba(0,0,0,0.5) inset',
			background: css.preset.widgetGradient
		});
		css.addClass('lx-ActiveBox-close', {
			cursor: 'pointer',
			color: css.preset.widgetIconColor,
			'@icon': ['\\2715', {fontSize:10, paddingBottom:'3px'}]
		});
		css.addClass('lx-ActiveBox-headerText', {
			fontWeight: 'bold',
			color: css.preset.headerTextColor
		});
		css.addClass('lx-ActiveBox-body', {
			overflow: 'auto',
			backgroundColor: css.preset.altBodyBackgroundColor
		});
		css.addClass('lx-ActiveBox-resizer', {
			cursor: 'se-resize',
			borderRadius: css.preset.borderRadius,
			color: css.preset.widgetIconColor,
			backgroundColor: css.preset.bodyBackgroundColor,
			'@icon': ['\\21F2', {fontSize:10, paddingBottom:'0px'}],
			opacity: 0
		}, {
			hover: {
				opacity: 1,
					transition: 'opacity 0.3s linear'
			}
		});
		css.addClass('lx-ActiveBox-move', {
			marginTop: '-2px',
			boxShadow: '0 '+(Math.round(shadowShift*1.5))+'px '+(Math.round(shadowSize*1.5))+'px rgba(0,0,0,0.5)',
		});
	}

	getDefaultDepthCluster() {
		return lx.DepthClusterMap.CLUSTER_MIDDLE;
	}

	/**
	 * @widget-init
	 *
	 * @param [config] {Object: {
	 *     #merge(lx.Box::render::config),
	 * 	   [header] {String}
	 * 	   [headerHeight] {String|Number} 
	 * 	   [headerConfig] {Object: #schema(lx.Box::render::config)}
	 * 	   [closeButton] {Boolean}
	 * 	   [move] {Boolean}
	 * 	   [resize] {Boolean}
	 *	   [adhesive] {Boolean}
	 * }}
	 */
	render(config={}) {
		super.render(config);
		this.addClass('lx-ActiveBox');

		this.setBuildMode(true);

		__setHeader(this, config);
		__setBody(this, config);

		if (lx.getFirstDefined(config.resize, lx.self(DEFAULT_RESIZE))) this.setResizer(config);
		this.adhesive = lx.getFirstDefined(config.adhesive, lx.self(DEFAULT_ADHESIVE));

		this.setBuildMode(false);

		super.render(config);
	}

	// @lx:<context CLIENT:
	clientRender(config) {
		super.clientRender(config);

		if (this.width() === null) this.width(this.width('px')+'px');
		if (this.height() === null) this.height(this.height('px')+'px');

		if (this.adhesive) {
			ActiveBoxAdhesor.makeAdhesion(this);
		}

		this.setBuildMode(true);

		if (this.contains('resizer') || this.contains('header')) {
			this.on('mousedown', ()=>this.emerge());
			this.emerge();
		}

		if (this.contains('header')) {
			let header = lx(this)>header;
			header.on('dblclick', function() {
				if (!this.lxActiveBoxGeom) {
					this.lxActiveBoxGeom = this.parent.getGeomMask();
					this.parent.setGeom([0, 0, '100%', '100%']);
				} else {
					this.parent.copyGeom(this.lxActiveBoxGeom);
					delete this.lxActiveBoxGeom;
				}
			});
			if (header.__move) {
				delete header.__move;
				header.move({parentMove: true});
				header.on('moveBegin', ()=>{
					this.addClass('lx-ActiveBox-move');
				});
				header.on('moveEnd', ()=>{
					this.removeClass('lx-ActiveBox-move');
				});
			}

			let closeButton = this.find('closeButton');
			if (closeButton && closeButton instanceof lx.Rect) {
				closeButton.getActiveBox = function() {
					return this.parent.parent;
				};
				if (!closeButton.hasTrigger('click'))
					closeButton.click(function() {
						this.getActiveBox().hide();
					});
			}
		}
		this.setBuildMode(false);
	}

	show() {
		this.emerge();
		super.show();
	}
	// @lx:context>

	setHeaderText(text) {
		this.find('textWrapper');
	}

	getCloseButton() {
		this.setBuildMode(true);
		let cb = lx(this)>>closeButton;
		this.setBuildMode(false);
		return cb;
	}

	setResizer(config) {
		let resizer = new lx.Rect({
			parent: this,
			key: 'resizer',
			geom: [null, null, lx.self(RESIZER_SIZE), lx.self(RESIZER_SIZE), 0, 0]
		}).move({parentResize: true});
		resizer.addClass('lx-ActiveBox-resizer');

		if (config.adhesive) {
			resizer.moveParams.xMinMove = 5;
			resizer.moveParams.yMinMove = 5;
		}
	}
}

lx.ActiveBox.DEFAULT_MOVE = true;
lx.ActiveBox.DEFAULT_RESIZE = true;
lx.ActiveBox.DEFAULT_ADHESIVE = false;
lx.ActiveBox.ADHESION_DISTANCE = 15;
lx.ActiveBox.RESIZER_SIZE = '25px';

//=============================================================================================================================
function __setHeader(self, config) {
	if (!config.header && !config.headerHeight && !config.headerConfig) return;

	let headerConfig = config.headerConfig || {};
	headerConfig.parent = self;
	headerConfig.key = 'header';

	let text = headerConfig.text || config.header || '';
	delete headerConfig.text;
	headerConfig.height = lx.ActiveBox.HEADER_HEIGHT + 'px';
	headerConfig.style = {
		position: 'relative',
		margin: lx.ActiveBox.INDENT + 'px'
	};

	let header = new lx.Box(headerConfig);
	header.addClass('lx-ActiveBox-header');

	if (text != '') {
		let wrapper = header.add(lx.Box, {size:['100%', '100%'], key:'textWrapper'});
		wrapper.text(text);
		lx(wrapper)>text.addClass('lx-ActiveBox-headerText');
		wrapper.align(lx.CENTER, lx.MIDDLE);
	}

	if (lx.getFirstDefined(config.move, lx.ActiveBox.DEFAULT_MOVE)) header.__move = true;

	if (config.closeButton) {
		let butConfig = lx.isObject(config.closeButton) ? config.closeButton : {};
		let butSize = lx.ActiveBox.HEADER_HEIGHT - lx.ActiveBox.INDENT * 4 + 'px';
		let butIndent = lx.ActiveBox.INDENT * 2 + 'px';
		butConfig.key = 'closeButton';
		if (!butConfig.geom) butConfig.geom = [null, butIndent, butSize, butSize, butIndent];
		if (!butConfig.css) butConfig.css = 'lx-ActiveBox-close';
		butConfig.parent = header;
		let className = butConfig.widget ? butConfig.widget : lx.Box;
		new className(butConfig);
	}
}

function __setBody(self, config) {
	let constructor = config.body || lx.Box,
		bodyConfig = {};
	if (lx.isArray(constructor)) {
		bodyConfig = constructor[1];
		constructor = constructor[0];
	}
	bodyConfig.parent = self;
	bodyConfig.key = 'body';
	let geom = lx.PositioningStrategy.geomFromConfig(config);
	if (geom.h === 'auto') {
		bodyConfig.style = {
			position: 'relative',
			margin: lx.ActiveBox.INDENT + 'px'
		};
	} else {
		bodyConfig.geom = [
			lx.ActiveBox.INDENT + 'px',
			self.contains('header')
				? lx.ActiveBox.INDENT * 2 + lx.ActiveBox.HEADER_HEIGHT + 'px'
				: lx.ActiveBox.INDENT + 'px',
			null,
			null,
			lx.ActiveBox.INDENT + 'px',
			lx.ActiveBox.INDENT + 'px'
		];
	}

	let body = new constructor(bodyConfig);
	body.addClass('lx-ActiveBox-body');
}

// @lx:<context CLIENT:
class ActiveBoxAdhesor {
	static makeAdhesion(el) {
		el.adhesiveBonds = this.getAdhesiveBondsBlank();
		this.check(el);

		el.on('move', this.checkOnMove);
		el.on('resize', this.checkOnResize);
	}

	static checkOnMove() {
		ActiveBoxAdhesor.check(this, 'move');
	}

	static checkOnResize() {
		ActiveBoxAdhesor.check(this, 'resize');
	}

	static check(ctx, action) {
		let env = ctx.parent.getChildren();
		env.forEach(a=>{
			if (a === ctx) return;
			let lims = ctx.rect('px'),
				aLims = a.rect('px'),
				lDist = lims.left - aLims.right,
				rDist = lims.right - aLims.left,
				tDist = lims.top - aLims.bottom,
				bDist = lims.bottom - aLims.top,
				valid = this.getValid(lDist, rDist, tDist, bDist, lims, aLims);

			if (!valid.ok() && ctx.adhesiveBonds && ctx.adhesiveBonds.contains(a)) {
				ctx.adhesiveBonds.remove(a);
				if (a.adhesiveBonds) {
					a.adhesiveBonds.remove(ctx);
					this.actualizeSizeShare(a);
				}
			}

			if (valid.ok()) {
				// Сразу нацепим прилипание во время движения и ресайза
				if (action == 'resize') {
					if (valid.x()) {
						let xDist = Math.abs(lDist) < Math.abs(rDist) ? lDist : rDist;
						ctx.width(lims.width - xDist + 'px');
					}
					if (valid.y()) {
						let yDist = Math.abs(tDist) < Math.abs(bDist) ? tDist : bDist;
						ctx.height(lims.height - yDist + 'px');
					}
				} else if (action == 'move') {
					if (valid.l) ctx.left(aLims.right + 'px');
					if (valid.r) ctx.left(aLims.left - lims.width + 'px');
					if (valid.t) ctx.top(aLims.bottom + 'px');
					if (valid.b) ctx.top(aLims.top - lims.height + 'px');
				}

				// Запишем кто с кем связался
				ctx.adhesiveBonds[valid.side()].set(a, a);

				// Если соседи оба адгезивные - они умеют делить размер
				if (a.adhesiveBonds) {
					a.adhesiveBonds[valid.contrSide()].set(ctx, ctx);
					this.actualizeSizeShare(a);
				}
			}

		});
		this.actualizeSizeShare(ctx);
	}

	static getValid(lDist, rDist, tDist, bDist, lims0, lims1) {
		return {
			l: Math.abs(lDist) < lx.ActiveBox.ADHESION_DISTANCE && lims0.top < lims1.bottom,
			r: Math.abs(rDist) < lx.ActiveBox.ADHESION_DISTANCE && lims0.top < lims1.bottom,
			t: Math.abs(tDist) < lx.ActiveBox.ADHESION_DISTANCE && lims0.left < lims1.right,
			b: Math.abs(bDist) < lx.ActiveBox.ADHESION_DISTANCE && lims0.left < lims1.right,
			x: function () {
				return this.l || this.r
			},
			y: function () {
				return this.t || this.b
			},
			ok: function () {
				return this.x() || this.y()
			},
			side: function () {
				if (this.l) return 'l';
				if (this.r) return 'r';
				if (this.t) return 't';
				if (this.b) return 'b';
			},
			contrSide: function () {
				if (this.l) return 'r';
				if (this.r) return 'l';
				if (this.t) return 'b';
				if (this.b) return 't';
			}
		};
	}

	static getAdhesiveBondsBlank() {
		return {
			l:new Map(), r:new Map(), t:new Map(), b:new Map(),
			contains: function (el) {
				if (this.l.has(el)) return true;
				if (this.r.has(el)) return true;
				if (this.t.has(el)) return true;
				if (this.b.has(el)) return true;
				return false;
			},
			remove: function (el) {
				if (this.l.has(el)) this.l.delete(el);
				else if (this.r.has(el)) this.r.delete(el);
				else if (this.t.has(el)) this.t.delete(el);
				else if (this.b.has(el)) this.b.delete(el);
			}
		};
	}

	static actualizeSizeShare(el) {
		let size = Math.round(lx.ActiveBox.ADHESION_DISTANCE * 0.75),
			seams = [];

		el.setBuildMode(true);
		if (!el.adhesiveBonds.r.size) {
			el.del('r_size_share');
		} else if (!el.contains('r_size_share')) {
			seams.push(el.add(lx.Rect, {
				key: 'r_size_share',
				width: size + 'px',
				right: 0,
				style: {cursor: 'ew-resize'}
			}).move({parentResize: true, yMove: false}));
		}

		if (!el.adhesiveBonds.b.size) {
			el.del('b_size_share');
		} else if (!el.contains('b_size_share')) {
			seams.push(el.add(lx.Rect, {
				key: 'b_size_share',
				ignoreHeaderHeight: true,
				bottom: 0,
				height: size + 'px',
				style: {cursor: 'ns-resize'}
			}).move({parentResize: true, xMove: false}));
		}
		el.setBuildMode(false);

		seams.forEach(a=>{
			// a.on('moveBegin', function() { el.off('resize', ActiveBoxAdhesor.checkOnResize); });
			// a.on('moveEnd', function() { el.on('resize', ActiveBoxAdhesor.checkOnResize); });
			a.on('move', function () {
				ActiveBoxAdhesor[this.key[0] + 'SeamMove'](this);
			});
		});
	}

	static rSeamMove(seam) {
		let ab = seam.parent,
			bonds = ab.adhesiveBonds,
			els = bonds.r,
			r = ab.left('px') + ab.width('px'),
			delta = null;
		for (let el of els.values()) {
			if (delta === null) delta = el.left('px') - r;
			el.width(el.width('px') + delta + 'px');
			el.left(r + 'px');
			el.checkResize();
		}
	}

	static bSeamMove(seam) {
		let ab = seam.parent,
			bonds = ab.adhesiveBonds,
			els = bonds.b,
			b = ab.top('px') + ab.height('px'),
			delta = null;
		for (let el of els.values()) {
			if (delta === null) delta = el.top('px') - b;
			el.height(el.height('px') + delta + 'px');
			el.top(b + 'px');
			el.checkResize();
		}
	}
}
// @lx:context>
