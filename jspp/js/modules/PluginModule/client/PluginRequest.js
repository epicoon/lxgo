// @lx:namespace lx;
class PluginRequest extends lx.HttpRequest {
	constructor(plugin, path, params={}) {
		super('/lx/plugin', {
			plugin: plugin.name,
			pluginParams: plugin.params || {},
			path,
			params
		});
	}
}
