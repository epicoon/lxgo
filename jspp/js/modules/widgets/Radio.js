// @lx:module lx.Radio;

// @lx:use lx.Checkbox;
// @lx:use lx.BasicCssContext;

/**
 * @widget lx.Radio
 * @content-disallowed
 * 
 * CSS classes:
 * - lx-Radio-1
 * - lx-Radio-0
 */
// @lx:namespace lx;
class Radio extends lx.Checkbox {
	cssChecked() {
		return 'lx-Radio-1';
	}

	cssUnchecked() {
		return 'lx-Radio-0';
	}

	static initCss(css) {
		css.useHolder(lx.BasicCssContext);
		css.addClass('lx-Radio-0', {
			border: 'solid #61615e 1px',
			width: '16px',
			height: '16px',
			borderRadius: '50%',
			backgroundColor: 'white',
			cursor: 'pointer'
		}, {
			hover: {
				boxShadow: '0 0 6px ' + css.preset.widgetIconColor,
			},
			active: {
				backgroundColor: '#dedede',
				boxShadow: '0 0 8px ' + css.preset.widgetIconColor,
			}
		});
		css.inheritClass('lx-Radio-1', 'lx-Radio-0', {
			color: 'black',
			'@icon': ['\\25CF', {fontFamily:'main', fontSize:8, paddingBottom:'1px'}],
		});
	}
}
