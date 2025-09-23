// @lx:<context CLIENT:
// @lx:require -U unpacker/;
let _plugin = null;
// @lx:context>

let _list = {};

// @lx:namespace lx;
class PluginManager extends lx.AppComponent {
    getList() {
        return _list;
    }

    get(key) {
        if (!(key in _list)) return null;
        return _list[key];
    }

    add(key, plugin) {
        _list[key] = plugin;
    }

    /**
     * @param ctx {lx.Rect}
     * @returns {lx.Plugin|null}
     */
    getPlugin(ctx) {
        // @lx:<context SERVER:
        return lx.globalContext.$plugin;
        // @lx:context>
        // @lx:<context CLIENT:
        if (!(ctx instanceof lx.Rect)) return null;
        let root = this.getRootSnippet(ctx);
        return root ? root.plugin : null;
        // @lx:context>
    }

    /**
     * @param ctx {lx.Rect}
     * @returns {lx.Box|null}
     */
    getRootSnippet(ctx) {
        // @lx:<context SERVER:
        return lx.globalContext.$snippet;
        // @lx:context>
        // @lx:<context CLIENT:
        if (ctx.plugin) return ctx;
        return ctx.ancestor({hasProperty: 'plugin'});
        // @lx:context>
    }

    /**
     * @param ctx {lx.Rect}
     * @returns {lx.Box|null}
     */
    getSnippet(ctx) {
        // @lx:<context SERVER:
        return lx.globalContext.$snippet;
        // @lx:context>
        // @lx:<context CLIENT:
        return ctx.ancestor({hasProperty: 'isSnippet'});
        // @lx:context>
    }

    // @lx:<context CLIENT:
    /**
     * @param info {Object|string}
     * @param el {lx.Box}
     * @param [parent] {lx.Plugin|null}
     * @param [clientCallback] {Function}
     */
    unpack(info, el, parent = null, clientCallback = null) {
        new lx.Task('unpackPlugin', function() {
            let loadContext = new LoadContext();
            loadContext.run(info, el, parent, ()=>{
                this.setCompleted();
                if (clientCallback) clientCallback();
            });
        });
    }

    /**
     * @param plugin {lx.Plugin|string}
     */
    remove(plugin) {
        let key = lx.isString(plugin) ? plugin : plugin.key;
        plugin = this.get(key);
        if (plugin === null) return;

        this.blur(plugin);
        plugin.setFocusable(false);

        // Delete child plugins
        let childPlugins = plugin.getChildPlugins(true);
        for (let i=0, l=childPlugins.len; i<l; i++)
            this.remove(childPlugins[i]);

        // Delete keyboard handlers
        lx.app.keyboard.offKeydown(null, null, {elem:plugin});
        lx.app.keyboard.offKeyup(null, null, {elem:plugin});

        // Run delete-callbacks
        for (let i=0, l=plugin.destructCallbacks.len; i<l; i++)
            lx.app.functionHelper.callFunction(plugin.destructCallbacks[i]);

        // Custom delete method
        if (plugin.destruct) plugin.destruct();

        // Clear root-widget
        delete plugin.root.plugin;
        plugin.root.clear();

        // Unbound dependencies
        lx.app.dependencies.independ(plugin.dependencies);

        //TODO if such plugin was not along it will delete common sources. Need to notice this namespaces as dependencies
        // Delete plugin-made namespaces
        for (let i=0, l=plugin.namespaces.len; i<l; i++)
            delete window[plugin.namespaces[i]];

        plugin.eventCallbacks = [];

        delete _list[plugin];
    }

    getFocusedPlugin() {
        return _plugin;
    }

    focus(plugin) {
        if (_plugin === plugin) return;
        this.blur();
        _plugin = plugin;
        _plugin.trigger('focus');
    }

    blur(plugin = null) {
        if (_plugin === null) return;
        if (plugin !== null && plugin !== _plugin) return;

        _plugin.trigger('blur');
        _plugin = null;
    }
    // @lx:context>

    // @lx:<mode DEV:
    status() {
        lx.log('Plugins list:');
        lx.log(_list);
        // @lx:<context CLIENT:
        lx.log('Focused plugin:');
        lx.log(_plugin);
        // @lx:context>
    }
    // @lx:mode>
}
