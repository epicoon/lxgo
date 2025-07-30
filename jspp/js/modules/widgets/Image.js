// @lx:module lx.Image;

/**
 * @widget lx.Image
 * @content-disallowed
 *
 * Events:
 * - load
 * - scale
 */
// @lx:namespace lx;
class Image extends lx.Rect {
	modifyConfigBeforeApply(config) {
		if (lx.isString(config)) config = {filename: config};
		if (!config.key) config.key = 'image';
		return config;
	}

	/**
	 * @widget-init
	 *
	 * @param [config] {Object: {
	 *     #merge(lx.Rect::constructor::config),
	 *     [src] {String} (: path to image file relative to site root :),
	 *     [path] {String} (: path to image according to widget or lx.app.imageManager settings :)
	 * }}
	 */
	render(config) {
		super.render(config);

		let src = config.src || null;
		if (config.path) src = this.imagePath(config.path);

		this.setAttribute('onLoad', 'this.setAttribute(\'loaded\', 1)');
		this.source(src);
	}

	static getStaticTag() {
		return 'img';
	}

	source(url) {
		this.setAttribute('loaded', 0);
		this.domElem.setAttribute('src', url);
		return this;
	}

	picture(url) {
		this.source(this.imagePath(url));
		return this;
	}

	value(src) {
		if (src === undefined) return this.domElem.param('src');
		this.source(src);
	}

	// @lx:<context CLIENT:
	isLoaded() {
		let elem = this.getDomElem();
		if (!elem) return false;
		return !!+this.getAttribute('loaded');
	}

	adapt() {
		let elem = this.getDomElem();
		if (!elem) {
			this.domElem.addAction('adapt');
			return this;
		}

		function scale() {
			let container = this.parent.getContainer().getDomElem(),
				sizes = lx.Geom.scaleBar(
					container.offsetHeight,
					container.offsetWidth,
					elem.naturalHeight,
					elem.naturalWidth
				);
			this.width(sizes[1] + 'px');
			this.height(sizes[0] + 'px');
			this.off('load', scale);
			this.trigger('scale');
		};
		
		if (this.isLoaded()) scale.call(this);
		else this.on('load', scale);
		return this;
	}
	// @lx:context>

	// @lx:<context SERVER:
	adapt() {
		this.onLoad('.adapt');
	}
	// @lx:context>
}
