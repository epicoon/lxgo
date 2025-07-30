// @lx:module lx.Button;

// @lx:use lx.BasicCssContext;

/**
 * @widget lx.Button
 * @content-disallowed
 * 
 * CSS classes:
 * - lx-Button
 * - lx-Button-hint
 */
// @lx:namespace lx;
class Button extends lx.Box {
	render(config={}) {
		super.render(config);
		this.addClass('lx-Button');
	}	

	// @lx:<context CLIENT:
	clientRender(config) {
		super.clientRender(config);
		this.align(lx.CENTER, lx.MIDDLE);
		this.on('mousedown', lx.preventDefault);
		this.setEllipsisHint({css: 'lx-Button-hint'});
	}
	// @lx:context>

	/**
	 * @param {lx.CssContext} css
	 */
	static initCss(css) {
		css.useHolder(lx.BasicCssContext);
		css.inheritClass('lx-Button', 'ActiveButton');
		css.inheritClass('lx-Button-hint', 'AbstractBox', {
			padding: '10px'
		});
	}
}
