// @lx:namespace lx;
class Plugin extends lx.Element {
    constructor(config = {}) {
        super();
        _construct(this, config);
    }

    /**
     * @returns {Function[]} constructor: lx.PluginCssAsset
     */
    static getCssAssetClasses() {
        return [];
    }

    set title(value) {
        _setTitle(this, value);
    }

    get title() {
        return _getTitle(this);
    }

    getCssScope() {
        return this.scope;
    }

    getImagePaths(key = 'default') {
        if (key in this.imagePaths) return this.imagePaths[key] + '/';
        return null;
    }

    getImage(name) {
        let path = lx.app.imageManager.getPath(this, name);
        return path || name;
    }

    /* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
     * CLIENT
     * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */
    // @lx:<context CLIENT:
    init() {
        // pass
    }

    getCoreClass() {
        return null;
    }

    getCore() {
        return this.core;
    }

    getGuiNodeClasses() {
        return {};
    }

    initGuiNodes(map) {
        for (let name in map) {
            let className = map[name];
            if (lx.isString(className) && !lx.classExists(className)) continue;
            let box = this.findOne(name);
            if (box === null) continue;
            this.guiNodes[name] = lx.isString(className)
                ? lx.createObject(className, [this, box])
                : new className(this, box);
        }
    }

    getGuiNode(name) {
        return this.guiNodes[name];
    }

    /** @abstract */
    beforeRender() {
        // pass
    }

    beforeRun() {
        const coreClass = this.getCoreClass();
        if (coreClass) this.core = new coreClass(this);
        this.initGuiNodes(this.getGuiNodeClasses());
    }

    /** @abstract */
    run() {
        // pass
    }

    get eventDispatcher() {
        if (!this._eventDispatcher)
            this._eventDispatcher = new lx.EventDispatcher();
        return this._eventDispatcher;
    }

    on(eventName, callback) {
        this.eventDispatcher.subscribe(eventName, callback);
    }

    onEvent(callback) {
        this.eventCallbacks.push(callback);
    }

    trigger(eventName, data = {}) {
        if (eventName == 'focus') {
            if (this._onFocus) this._onFocus();
            return;
        }
        if (eventName == 'blur') {
            if (this._onUnfocus) this._onUnfocus();
            return;
        }

        var event = new lx.PluginEvent(this, eventName, data);
        this.eventDispatcher.trigger(eventName, event);
        this.eventCallbacks.forEach(c=>c(event));
    }

    setFocusable(val = true) {
        if (this.focusable === val) return;
        this.focusable = val;
        (this.focusable)
            ? this.root.click(_onClick)
            : this.root.off('click', _onClick);
    }

    focus() {
        lx.app.pluginManager.focus(this);
    }

    blur() {
        lx.app.pluginManager.blur(this);
    }

    onFocus(callback) {
        this._onFocus = callback.bind(this);
    }

    onUnfocus(callback) {
        this._onUnfocus = callback.bind(this);
    }

    initKeypressManager(className) {
        let manager = lx.createObject(className);
        manager.addContext({elem: this});
        manager.run();
        return manager;
    }

    onKeydown(key, func) {
        lx.app.keyboard.onKeydown(key, func, {elem:this});
    }

    offKeydown(key, func) {
        lx.app.keyboard.offKeydown(key, func, {elem:this});
    }

    onKeyup(key, func) {
        lx.app.keyboard.onKeyup(key, func, {elem:this});
    }

    offKeyup(key, func) {
        lx.app.keyboard.offKeyup(key, func, {elem:this});
    }

    /**
     * Check if this plugin in main for the page render
     */
    isMainContext() {
        return this.isMain;
    }

    onDestruct(callback) {
        this.destructCallbacks.push(callback);
    }

    getChildPlugins(all=false) {
        var result = [];

        const list = lx.app.pluginManager.getList();
        for (var i in list) {
            let plugin = list[i];
            if (all) {
                if (plugin.hasAncestor(this))
                    result.push(plugin);
            } else {
                if (plugin.parent === this)
                    result.push(plugin);
            }
        }

        return result;
    }

    /**
     * @param name {String}
     * @param [all = false] {Boolean}
     * @return {lx.Plugin|null}
     */
    getChildPlugin(name, all = false) {
        let plugins = this.getChildPlugins(all);
        for (let i in plugins)
            if (plugins[i].name == name) return plugins[i];
        return null;
    }

    hasAncestor(plugin) {
        var parent = this.parent;
        while (parent) {
            if (parent === plugin) return true;
            parent = parent.parent;
        }
        return false;
    }

    useModule(moduleName, callback = null) {
        this.useModules([moduleName], callback);
    }

    useModules(moduleNames, callback = null) {
        var newForPlugin = [];
        if (this.dependencies.modules) {
            for (var i=0, l=moduleNames.len; i<l; i++)
                if (!this.dependencies.modules.includes(moduleNames[i]))
                    newForPlugin.push(moduleNames[i]);
        } else newForPlugin = moduleNames;

        if (!newForPlugin.len) return;

        if (!this.dependencies.modules) this.dependencies.modules = [];

        lx.app.dependencies.promiseModules({
            modules: newForPlugin,
            callback: ()=>{
                newForPlugin.forEach(a=>this.dependencies.modules.push(a));
                if (callback) callback();
            }
        });
    }

    /**
     * AJAX-request in the plugin context
     */
    ajax(path, data=[]) {
        return new lx.PluginRequest(this, path, data);
    }

    /**
     * Register active GET AJAX-request for URL condition actualization
     */
    registerActiveRequest(key, respondent, handlers, useServer=true) {
        if (!this.activeRequestList)
            this.activeRequestList = new AjaxGet(this);
        this.activeRequestList.registerActiveUrl(key, respondent, handlers, useServer);
    }

    /**
     * Call to active GET AJAX-request if it was registered
     */
    activeRequest(key, data={}) {
        if (this.activeRequestList)
            this.activeRequestList.request(key, data);
    }

    get(path) {
        return this.root.get(path);
    }

    find(key) {
        return this.root.find(key);
    }

    findOne(key, all) {
        var c = this.root.find(key, all);
        if (c instanceof lx.Rect) return c;
        if (this.root.key == key) c.add(this.root);
        if (c.empty) return null;
        return c.at(0);
    }

    subscribeNamespacedClass(namespace, className) {
        //TODO
        console.log('Plugin.subscribeNamespacedClass:', namespace, className);
    }
    // @lx:context>

    /* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
     * SERVER
     * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */
    // @lx:<context SERVER:
    set icon(value) {
        this._icon = value;
        this._changes.icon = value;
    }

    get icon() {
        return this._icon;
    }

    /**
     * @param data {Object}
     */
    setData(data) {
        this.data = data;
    }

    /**
     * @param key {String}
     * @param val {any}
     */
    addData(key, val) {
        this.data[key] = val;
    }

    onLoad(f) {
        this._onLoad.push(f);
    }

    getDependencies() {
        return {};
    }

    getResult() {
        let result = {};

        if (this._changes.title) result.title = this._changes.title;
        if (this._changes.icon) result.icon = this._changes.icon;
        if (!this.data.lxEmpty()) result.data = this.data;

        let onLoad = [];
        for (let i in this._onLoad) onLoad.push(
            (lx.isFunction(this._onLoad[i]))
                ? lx.app.functionHelper.functionToString(this._onLoad[i])
                : this._onLoad[i]
        )
        if (onLoad.length) result.onLoad = onLoad;

        return result;
    }
    // @lx:context>
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * PRIVATE
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */
// @lx:<context CLIENT:
function _construct(self, config) {
    self.core = null;
    self.name = config.name;
    self.data =  {};
    self.root = config.root;

    self.scope = config.cssScope || '';
    const stat = self.constructor;
    let css = stat.getCssAssetClasses();
    if (css.length > 0 || !lx.app.functionHelper.isEmptyFunction(stat.initCss))
        lx.app.cssManager.addElement(stat, self.scope);

    self.eventCallbacks = [];
    self.destructCallbacks = [];
    self.namespaces = [];
    self.dependencies = {};

    self._eventDispatcher = null;
    self._onFocus = null;
    self._onUnfocus = null;

    if (self.root) _init(self, config);
    self.guiNodes = {};
    self.init();
}

function _setTitle(self, value) {
    if (this.isMainContext()) document.title = val;
}

function _getTitle(self) {
    return document.title;
}

function _init(self, config) {
    // Gen unique key
    const list = lx.app.pluginManager.getList();
    if (config.key in list) {
        var key;
        function randKey() {
            return '' +
                lx.Math.decChangeNotation(lx.Math.randomInteger(0, 255), 16) +
                lx.Math.decChangeNotation(lx.Math.randomInteger(0, 255), 16) +
                lx.Math.decChangeNotation(lx.Math.randomInteger(0, 255), 16);
        };
        do {
            key = randKey();
        } while (key in list);
        self.key = key;
    } else self.key = config.key;
    lx.app.pluginManager.add(self.key, self);

    if (config.parent) self.parent = config.parent;
    if (config.main || config.isMain) self.isMain = true;
    if (config.data) self.data = config.data;

    if (config.imagePaths) self.imagePaths = config.imagePaths;

    // Dependencies info
    if (config.dep) self.dependencies = config.dep;

    self.focusable = false;
    self.setFocusable();

    self.root.plugin = self;
    self.root.on('beforeDestruct', function() {lx.app.pluginManager.remove(this.plugin)});
}

function _onClick() {
    this.plugin.focus();
}
// @lx:context>

// @lx:<context SERVER:
function _construct(self, config) {
    self.name = config.name;
    self.path = config.path;
    self.imagePaths = config.imagePaths;
    self.scope = config.cssScope || '';

    self._title = config.title;
    self._icon = config.icon;
    self._changes = {
        title: null,
        icon: null
    };

    self.params = config.params ? config.params.lxClone() : {};
    self.data = {};
    self._onLoad = [];
}

function _setTitle(self, value) {
    self._title = value;
    self._changes.title = value;
}

function _getTitle(self) {
    return self._title;
}
// @lx:context>
