// @lx:namespace lx;
class PluginRequest extends lx.HttpRequest {
	constructor(plugin, respondent, params=[]) {
		super('/lx_plugin', {
			plugin: plugin.name,
			attributes: plugin.attributes,
			respondent,
			data: params
		});
	}
}
