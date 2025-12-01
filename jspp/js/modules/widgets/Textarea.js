// @lx:module lx.Textarea;

// @lx:use lx.Input;
// @lx:use lx.BasicCssContext;

/**
 * @widget lx.Textarea
 * @content-disallowed
 * 
 * CSS classes:
 * - lx-Textarea
 */
// @lx:namespace lx;
class Textarea extends lx.Input {
	static getStaticTag() {
		return 'textarea';
	}
	
	static initCss(css) {
		css.useExtender(lx.BasicCssContext);
		css.inheritClass('lx-Textarea', 'Input', {
			resize: 'none'
		});
	}

	render(config={}) {
		super.render(config);
		this.addClass('lx-Textarea');
	}
}
