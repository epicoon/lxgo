// @lx:namespace lx;
class PluginEvent {
	constructor(plugin, name, data = {}) {
		this.plugin = plugin;
		this.name = name;
		this.data = data;
	}

	getData() {
		return this.data;
	}
}
