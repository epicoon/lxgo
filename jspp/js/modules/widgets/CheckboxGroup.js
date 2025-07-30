// @lx:module lx.CheckboxGroup;

// @lx:use lx.Checkbox;
// @lx:use lx.LabeledGroup;

/**
 * @widget lx.CheckboxGroup
 * @content-disallowed
 */
// @lx:namespace lx;
class CheckboxGroup extends lx.LabeledGroup {
	/**
	 * @widget-init
	 *
	 * @param [config] {Object: {
	 *     #merge(lx.LabeledGroup::render::config),
	 *     [defaultValue] {Number|Array<Number>}
	 * }}
	 */
	render(config) {
		config.widgetSize = '30px';
		config.labelSide = lx.RIGHT;

		super.render(config);
		this.widgets().forEach(w=>{
			let checkbox = w.add(lx.Checkbox, {key: 'checkbox'});
			w.align(lx.CENTER, lx.MIDDLE);
			if (w._field) {
				checkbox._field = w._field;
				delete w._field;
			}
		});
		if (config.defaultValue !== undefined) this.value(config.defaultValue);
	}

	// @lx:<context CLIENT:
	clientRender(config) {
		super.clientRender(config);
		this.checkboxes().forEach(a=>a.on('change', function (e) {
			_handler_onChange.call(this, e);
			this.parent.parent.trigger('change', e);
		}));
		this.labels().forEach(l=>{
			l.style('cursor', 'pointer');
			l.on('mousedown', lx.preventDefault);
			l.on('click', (e)=>{
				let checkbox = this.checkbox(l.index);
				checkbox.todgle();
				_handler_onChange.call(checkbox, e);
			});
		});
	}
	// @lx:context>

	checkboxes() {
		return this.findAll('checkbox');
	}

	checkbox(num) {
		let ch = this.widget(num);
		return lx(ch)>checkbox;
	}

	value(nums) {
		if (nums === undefined) {
			let result = [];
			this.checkboxes().forEach(function(a) {
				if (a.value()) result.push(a.parent.index);
			});

			return result;
		}

		if (!nums) nums = [];

		this.checkboxes().forEach(a=>a.value(false));
		if (!lx.isArray(nums)) nums = [nums];
		nums.forEach(num=>this.checkbox(num).value(true));
	}
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * PRIVATE
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

// @lx:<context CLIENT:
function _handler_onChange(e) {
	let group = this.parent.parent;
	e = e || group.newEvent();
	e.changedIndex = this.parent.index;
	e.currentValue = this.value();
	e.currentValues = this.ancestor({is:lx.CheckboxGroup}).value();
}
// @lx:context>
