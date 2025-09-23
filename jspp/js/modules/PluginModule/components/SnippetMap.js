const map = {};

// @lx:namespace lx;
class SnippetMap extends lx.AppComponent {
	registerSnippet(name, func) {
		map[name] = func;
	}

	getSnippet(name) {
		return map[name] || null;
	}
}
