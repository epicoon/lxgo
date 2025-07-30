// @lx:module lx.Plugin;

// @lx:require Plugin;
// @lx:require common/;

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */
// @lx:<context SERVER:
/**
 * @param config {Object: {
 *     path {String},
 *     params {Dict}
 * }}
 * @param params {Dict}
 */
lx.Box.prototype.setSnippet = function (config, params = null) {
    if (lx.isString(config)) {
        config = {path: config};
        if (params !== null) config.params = params;
    }

    if (!config.path) return;
    if (!config.params) config.params = {};

    let container = this.__buildMode ? this : this._getContainer();

    const d = new lx.Date();
    let key = config.path + '.' + d.format('Y-m-d-h:i:s.l') + '.' + lx.Math.randomInteger(0, 999999)
    key = lx.HashMd5.hex(key);

    this.setAttribute('lx-snippet', key);

    // inner snippet
    container.snippetInfo = {
        hash: key,
        path: config.path,
        params: config.params
    };

    // flag that will be on the client side too
    container.isSnippet = true;
};

/**
 * @param config {String|Object: {
 *     name {String},
 *     [params] {Dict},
 *     [cssScope] {String},
 *     [onLoad] {Function}
 * }}
 */
lx.Box.prototype.setPlugin = function (config) {
    if (lx.isString(config))
        config = {name: config};
    if (!config.name) {
        lx.logError('Attempt to add plugin without name', 'Plugin');
        return;
    }

    const d = new lx.Date();
    let key = config.name + '.' + d.format('Y-m-d-h:i:s.l') + '.' + lx.Math.randomInteger(0, 999999)
    key = lx.HashMd5.hex(key);

    let container = this.__buildMode ? this : this._getContainer();
    container.setAttribute('lx-plugin', key);

    let data = {
        hash: key,
        name: config.name
    };
    if (config.params) data.params = config.params;
    if (config.cssScope) data.cssScope = config.cssScope;
    if (config.onLoad) data.onLoad = config.onLoad;
    $snippet.addPlugin(data);
};
// @lx:context>

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */
// @lx:<context CLIENT:
// @lx:require client/;
lx.Rect.prototype.getRectInPlugin = function () {
    let plugin = lx.app.pluginManager.getPlugin(this);
    return plugin
        ? this.getRelativeRect(plugin.root)
        : this.getGlobalRect();
};

/**
 * @param config {String|Object: {
 *     [info] {Object},
 *     [attributes] {Dict},
 *     [cssScope] {String},
 *     [onLoad] {Function}
 * }}
 */
lx.Box.prototype.setPlugin = function (config) {
    if (this.plugin) lx.app.pluginManager.remove(this.plugin);
    if (config.attributes && !config.attributes.lxEmpty()) {
        if (!config.info.attributes) config.info.attributes = {};
        config.info.attributes.lxMerge(config.attributes, true);
    }
    lx.app.pluginManager.unpack(config.info, this, lx.app.pluginManager.getPlugin(this), config.onLoad);
};
// @lx:context>

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */
// @lx:require components/;
lx.app.registerComponents({
    pluginManager: lx.PluginManager,
    snippetMap: lx.SnippetMap,
});
