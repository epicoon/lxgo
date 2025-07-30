// @lx:module lx.LabeledGroup;

/**
 * @widget lx.LabeledGroup
 * @content-disallowed
 * 
 * CSS classes:
 * - lx-LabeledGroup
 * - lx-LabeledGroup-item
 * - lx-LabeledGroup-label
 */
// @lx:namespace lx;
class LabeledGroup extends lx.Box {
	static initCss(css) {
		css.addClass('lx-LabeledGroup', {
			display: 'grid',
			gridAutoFlow: 'row',
			gridGap: '.8em',
			color: css.preset.textColor
		});
		css.addClass('lx-LabeledGroup-item', {
			position: 'relative',
			gridRow: 'auto'
		});
		css.addClass('lx-LabeledGroup-label', {
		});
	}

	/**
	 * @widget-init
	 *
	 * @param [config] {Object: {
	 *     #merge(lx.Rect::constructor::config),
	 *     [cols = 1] {Number},
	 *     [step] {Number},
	 *     [widget = lx.Box] {lx.Rect},
	 *     [widgetSize = '1fr'] {String},
	 *     [labelSide = lx.LEFT] {Number&Enum(lx.LEFT, lx.RIGHT)},
	 *     [labels] {Array<String>},
	 *     [list] {
	 *         Array<String|Object:{
	 *             #merge(lx.Box::render::config),
	 *             [widget = lx.Box] {lx.Rect},
	 *             [label] {String}
	 *     	   }>
	 *     }
	 *     [fields] {
	 *         Dict<String|Object:{
	 *             #merge(lx.Box::render::config),
	 *             [widget = lx.Box] {lx.Rect},
	 *             [label] {String}
	 *         }>
	 *     }
	 * }}
	 */
	render(config) {
		super.render(config);

		this.addClass('lx-LabeledGroup');
		let labelSide = config.labelSide || lx.LEFT,
			template = '',
			cols = config.cols || 1,
			widgetSize = config.widgetSize || '1fr';
		if (labelSide == lx.LEFT) {
			for (let i=0; i<cols; i++)
				template += '[labels'+i+']auto[controls'+i+']'+widgetSize;
		} else {
			for (let i=0; i<cols; i++)
				template += '[controls'+i+']'+widgetSize+'[labels'+i+']auto';
		}
		this.style('grid-template-columns', template);
		if (config.step !== undefined)
			this.style('grid-gap', config.step);

		let defaultWidget = config.widget || lx.Box;
		let units = [];
		if (config.list) {
			for (let i=0, l=config.list.len; i<l; i++) {
				let unitConfig = {},
					unitWidget,
					labelConfig = {},
					iConfig = config.list[i];
				if (lx.isString(iConfig)) {
					unitWidget = defaultWidget;
					labelConfig.text = iConfig;
				} else {
					unitConfig = iConfig;
					unitWidget = unitConfig.widget || defaultWidget;
					delete unitConfig.widget;
					labelConfig.text = unitConfig.label || '-';
					delete unitConfig.label;
				}
				units.push({widget: unitWidget, config: unitConfig, labelConfig});
			}
		} else if (config.fields) {
			for (let field in config.fields) {
				let unitConfig = {},
					unitWidget,
					labelConfig = {},
					iConfig = config.fields[field];
				if (lx.isString(iConfig)) {
					unitWidget = defaultWidget;
					labelConfig.text = iConfig;
				} else {
					unitConfig = iConfig;
					unitWidget = unitConfig.widget || defaultWidget;
					delete unitConfig.widget;
					labelConfig = {};
					labelConfig.text = unitConfig.label || '-';
					delete unitConfig.label;
				}
				unitConfig.field = field;
				units.push({widget: unitWidget, config: unitConfig, labelConfig});
			}
		} else if (config.labels) {
			for (let i=0, l=config.labels.len; i<l; i++)
				units.push({
					widget: defaultWidget,
					config: {},
					labelConfig: {text: config.labels[i]}
				});
		}

		let counter = 0;
		for (let i=0, l=units.len; i<l; i++) {
			let unit = units[i];
			let widget = unit.widget;
			let widgetConfig = unit.config.lxMerge({
				parent: this,
				key: 'widget',
				css: ['lx-LabeledGroup-item'],
				style: {'grid-column': 'controls' + counter}
			});
			let labelConfig = unit.labelConfig.lxMerge({
				parent: this,
				key: 'label',
				css: ['lx-LabeledGroup-item', 'lx-LabeledGroup-label'],
				style: {'grid-column': 'labels' + counter}
			});
			if (labelSide == lx.LEFT) {
				(new lx.Box(labelConfig)).align(lx.CENTER, lx.MIDDLE);
				new widget(widgetConfig);
			} else {
				new widget(widgetConfig);
				(new lx.Box(labelConfig)).align(lx.CENTER, lx.MIDDLE);
			}
			counter++;
			if (counter >= cols) counter = 0;
		}
	}

	// @lx:<context CLIENT:
	clientRender(config) {
		super.clientRender(config);
		if (!lx(this)>label) return;
		this.getAll('label').forEach(l=>{
			l.on('click', function() {
				this.parent.widget(this.index).trigger('click');
			});
			l.on('mouseup', function() {
				this.parent.widget(this.index).trigger('mouseup');
			});
		});
	}
	// @lx:context>

	align(w, h) {
		this.labels().forEach(a=>a.align(w, h));
	}

	widgets() {
		if (!lx(this)>widget) return new lx.Collection();
		return new lx.Collection(lx(this)>widget);
	}

	labels() {
		if (!lx(this)>label) return new lx.Collection();
		return new lx.Collection(lx(this)>label);
	}

	widget(num) {
		if (!lx(this)>widget) return null;
		return this.getAll('widget').at(num);
	}

	label(num) {
		if (!lx(this)>label) return null;
		return this.getAll('label').at(num);
	}
}
