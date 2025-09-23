// @lx:namespace lx;
class PluginEvent extends lx.Event {
	constructor(plugin, name, data = {}) {
		super(name, data);
		this.plugin = plugin;
	}
}
