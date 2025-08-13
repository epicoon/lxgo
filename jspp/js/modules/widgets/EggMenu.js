// @lx:module lx.EggMenu;

/**
 * @widget lx.EggMenu
 * 
 * CSS classes:
 * - lx-EggMenu
 * - lx-EggMenu-top
 * - lx-EggMenu-bottom
 * - lx-EggMenu-move
 */
// @lx:namespace lx;
class EggMenu extends lx.Box {
	static initCss(css) {
		let shadowSize = css.preset.shadowSize + 2,
			shadowShift = Math.floor(shadowSize * 0.5);
		css.addClass('lx-EggMenu', {
			overflow: 'visible',
			borderRadius: '25px',
			boxShadow: '0 '+shadowShift+'px '+shadowSize+'px rgba(0,0,0,0.5)'
		});
		css.addClass('lx-EggMenu-top', {
			backgroundColor: css.preset.bodyBackgroundColor,
			borderTopLeftRadius: '25px',
			borderTopRightRadius: '25px'
		});
		css.addClass('lx-EggMenu-bottom', {
			backgroundColor: css.preset.checkedSoftColor,
			borderBottomLeftRadius: '25px',
			borderBottomRightRadius: '25px'
		});
		css.addClass('lx-EggMenu-move', {
			marginTop: '-2px',
			boxShadow: '0 '+(Math.round(shadowShift*1.5))+'px '+(Math.round(shadowSize*1.5))+'px rgba(0,0,0,0.5)'
		});
	}

	_getContainer() {
		return lx(this)>menuBox._getContainer();
	}

	modifyConfigBeforeApply(config={}) {
		config.size = ['40px', '50px'];
		return config;
	}

	getDefaultDepthCluster() {
		return lx.DepthClusterMap.CLUSTER_FRONT;
	}

	/**
	 * @widget-init
	 *
	 * @param [config] {Object: {
	 *     #merge(lx.Box::render::config),
	 *     [menuWidget = lx.Box] {lx.Box}
	 *     [menuConfig] {Object: {#schema(lx.Box::render::config)}}
	 *     [menuRenderer] {Function} (: argument - lx.Box :)
	 * }}
	 */
	render(config) {
		super.render(config);

		this.addClass('lx-EggMenu');
		this.setBuildMode(true);
		this.style('positioning', 'fixed');
		this.add(lx.Rect, {
			key: 'top',
			height:'25px',
			css: 'lx-EggMenu-top'
		}).move({parentMove: true});
		this.add(lx.Rect, {
			key: 'switcher',
			top:'25px',
			height:'25px',
			css: 'lx-EggMenu-bottom'
		});
		this.setBuildMode(false);

		let menu = {};
		if (config.menuWidget) menu.widget = config.menuWidget;
		if (config.menuConfig) menu.config = config.menuConfig;
		if (config.menuRenderer) menu.renderer = config.menuRenderer;
		this.buildMenu(menu);
	}

	buildMenu(menuInfo) {
		this.setBuildMode(true);
		let widget = menuInfo.widget || lx.Box,
			config = menuInfo.config || {},
			menuRenderer = menuInfo.renderer;
		config.parent = this;
		config.key = 'menuBox';
		if (!config.geom) config.geom = true;

		let menu = new widget(config);
		menu.setGeomPriority(lx.WIDTH, lx.LEFT);
		menu.setGeomPriority(lx.HEIGHT, lx.TOP);

		//TODO некрасиво, но это частый класс для меню
		if (menu.lxFullClassName() == 'lx.ActiveBox')
	        lx(menu)>resizer.move({
	            parentResize: true,
	            xLimit: false,
	            yLimit: false
	        });

		if (menuRenderer) menuRenderer(menu);
		menu.hide();
		this.setBuildMode(false);
	}

	// @lx:<context CLIENT:
	clientRender(config) {
		super.clientRender(config);

		lx(this)>top.on('moveBegin', ()=>{
			this.addClass('lx-EggMenu-move');
		});
		lx(this)>top.on('moveEnd', ()=>{
			this.removeClass('lx-EggMenu-move');
		});

		this.on('move', ()=>this.holdPultVisibility());
		lx(this)>switcher.click(lx.self(switchOpened));
	}

	open() {
		let menu = lx(this)>menuBox;
		if (!menu) return;
		menu.show();
		menu.left(this.width('px') + 'px');
		this.holdPultVisibility();
	}

	close() {
		let menu = lx(this)>menuBox;
		if (!menu) return;
		menu.hide();
	}

	holdPultVisibility() {
		let menu = lx(this)>menuBox;
		if (!menu) return;
		let out = menu.isOutOfVisibility(this.parent);

		if (out.left) {
			menu.right(null);
			menu.left(this.width('px') + 'px');
		}

		if (out.right) {
			menu.left(null);
			menu.right(this.width('px') + 'px');
		}

		if (out.top) {
			menu.bottom(null);
			menu.top('0px');
		}

		if (out.bottom) {
			menu.top(null);
			menu.bottom('0px');
		}
	}

	static switchOpened() {
		let menu = this.parent,
			menuBox = lx(menu)>menuBox;
		if (!menuBox) return;

		if (menuBox.visibility()) menu.close();
		else menu.open();
	}
	// @lx:context>
}
