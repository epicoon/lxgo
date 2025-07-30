// @lx:namespace lx;
class PluginCore {
    constructor(plugin) {
        this.plugin = plugin;
        this.init();
        this.loadReferences();
        this.initHandlers();
        this.subscribeEvents();
    }

    /** @abstract */
    init() {
        // pass
    }

    /** @abstract */
    loadReferences() {
        // pass
    }

    /** @abstract */
    initHandlers() {
        // pass
    }

    /** @abstract */
    subscribeEvents() {
        // pass
    }
    
    getPlugin() {
        return this.plugin;
    }

    getRootBox() {
        return this.getPlugin().root;
    }
    
    getGuiNode(name) {
        return this.getPlugin().getGuiNode(name);
    }
}
