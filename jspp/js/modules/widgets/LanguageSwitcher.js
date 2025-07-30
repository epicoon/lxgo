// @lx:module lx.LanguageSwitcher;

// @lx:use lx.Dropbox;

/**
 * Language switcher based on lx.Dropbox, keeps language option in Cookies
 *
 * @widget lx.LanguageSwitcher
 * @content-disallowed
 */
// @lx:namespace lx;
class LanguageSwitcher extends lx.Dropbox {
	// @lx:<context CLIENT:
	clientRender(config) {
		super.clientRender(config);
		this.options(lx.app.lang.options());
		this.value(lx.app.lang.current());
		this.on('change', ()=>lx.app.lang.set(this.value()));
	}
	// @lx:context>
}
