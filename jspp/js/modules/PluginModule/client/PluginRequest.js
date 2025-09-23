// @lx:namespace lx;
class PluginRequest extends lx.HttpRequest {
	constructor(plugin, path, params={}) {
		super('/lx_plugin', {
			plugin: plugin.name,
			pluginParams: plugin.params || {},
			path,
			data: params
		});
	}
}
