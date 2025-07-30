const _data = {
    /** {Dict<String:Int>} - module name : dependent objects count */
    modules: {},
    css: {},
    scripts: {}
};

/**
 * Map describing elements dependencies
 */
// @lx:namespace lx;
class Dependencies extends lx.AppComponent {
    init() {
        this.cache = true;
    }

    /**
     * @param list {Array<String>}
     */
    noteModules(list) {
        let mm = [], els = [];
        for (let i in list) {
            let m = lx.getClassConstructor(list[i]);
            if (!m) continue;
            mm.push(list[i]);

            let csrs;
            // @lx:<context CLIENT:
            csrs = 'client';
            // @lx:context>
            // @lx:<context SERVER:
            csrs = 'server';
            // @lx:context>
            if (lx.app.getSetting('csrs') === csrs)
                if (m.prototype instanceof lx.Element) els.push(m);
        }
        this.depend({modules: mm});
        lx.app.cssManager.addElements(els);
    }

    // @lx:<context CLIENT:
    /**
     * @param config {Array<string>|Object: {
     *     modules {Array<String>},
     *     [depend: false] {Boolean},
     *     [immediately: true] {Boolean},
     *     [host] {String}
     *     [callback] {Function},
     * }}
     * @returns {lx.ServiceRequest|null}
     */
    promiseModules(config) {
        if (lx.isArray(config)) config = {modules: config};
        let callback = config.callback || null,
            depend = lx.getFirstDefined(config.depend, false),
            immediately = lx.getFirstDefined(config.immediately, true),
            need = _checkNeed(this, config.modules, 'modules', depend, callback);
        if (!need.length) return null;

        let modulesRequest = new lx.ServiceRequest('get-modules', {
            have: lx.app.dependencies.getCurrentModules(),
            need
        });
        if (config.host) modulesRequest.host = config.host;
        let onLoad = function (res) {
            if (!res.success) {
                console.error(res.data);
                return;
            }

            //TODO ??? do we need it? Refactor anywhere
            // let necessaryCss = lx.app.dependencies.defineNecessaryCss(res.data.css);
            // for (let i in necessaryCss) {
            //     let tagRequest = new lx.AssetRequest(
            //         necessaryCss[i],
            //         {name: 'module_asset'},
            //         'head-top'
            //     );
            //     tagRequest.send();
            // }

            lx.app.functionHelper.createAndCallFunction('', res.data.code);
            if (depend) lx.app.dependencies.depend({modules: config.modules});
            if (callback) callback();
        };

        if (immediately) {
            modulesRequest.send().then(onLoad);
            return null;
        }

        modulesRequest.onLoad(onLoad);
        return modulesRequest;
    }

    /**
     * @param config {Array<string>|Object: {
     *     scripts {Array<String>},
     *     [depend: false] {Boolean},
     *     [immediately: true] {Boolean},
     * }}
     * @returns {lx.AssetRequest[]}
     */
    promiseScripts(config) {
        return _promiseAsset(this, config, 'scripts');
    }

    /**
     * @param config {Array<string>|Object: {
     *     css {Array<String>},
     *     [depend: false] {Boolean},
     *     [immediately: true] {Boolean},
     * }}
     * @returns {lx.AssetRequest[]}
     */
    promiseCss(config) {
        return _promiseAsset(this, config, 'css');
    }
    // @lx:context>

    getCurrentModules() {
        let result = [];
        for (let name in _data.modules)
            result.push(name);
        return result;
    }

    /**
     * Subscribe element to resources
     */
    depend(data) {
        for (let i in _data)
            _process(_data[i], data[i] || {}, 1);
    }

    /**
     * If element deleted it unsubscribes from the modules
     * If no one else subscribes to a module and modules are not cached, such a module will be deleted
     */
    independ(data) {
        for (let i in _data)
            _process(_data[i], data[i] || {}, -1);
        _dropZero(this);
    }

    /**
     * @param map {Object: {
     *     [modules] {Array<String>},
     *     [scripts] {Array<String>},
     *     [css] {Array<String>},
     * }}
     * @returns {Object: {
     *     [modules] {Array<String>},
     *     [scripts] {Array<String>},
     *     [css] {Array<String>},
     * }}
     */
    defineNecessary(map) {
        let res = {};
        if (map.modules)
            res.modules = this.defineNecessaryModules(map.modules);
        if (map.scripts)
            res.scripts = this.defineNecessaryScripts(map.scripts);
        if (map.css)
            res.css = this.defineNecessaryCss(map.css);
        return res;
    }

    /**
     * Gets a list of required modules and selects those for which there is no information
     * @param list {Array<String>}
     * @returns {Array<String>}
     */
    defineNecessaryModules(list) {
        if (_data.modules.lxEmpty()) return list;
        let result = [];
        for (let i=0, l=list.len; i<l; i++)
            if (!(list[i] in _data.modules))
                result.push(list[i]);
        return result;
    }

    /**
     * @param list {Array<String>}
     * @returns {Array<String>}
     */
    defineNecessaryCss(list) {
        if (_data.css.lxEmpty()) return list;
        let result = [];
        for (let i=0, l=list.len; i<l; i++)
            if (!(list[i] in _data.css))
                result.push(list[i]);
        return result;
    }

    /**
     * @param list {Array<String>}
     * @returns {Array<String>}
     */
    defineNecessaryScripts(list) {
        if (_data.scripts.lxEmpty()) return list;
        let result = [];
        for (let i=0, l=list.len; i<l; i++)
            if (!(list[i] in _data.scripts))
                result.push(list[i]);
        return result;
    }
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * PRIVATE
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

function _process(data, map, modifier) {
    for (let i=0, l=map.len; i<l; i++) {
        let moduleName = map[i];
        if (!(moduleName in data)) {
            if (modifier == 1) data[moduleName] = 0;
            else continue;
        }

        data[moduleName] += modifier;
    }
}

function _dropZero(self) {
    if (self.cache) return;
    _dropZeroModules();
    _dropZeroCss();
    _dropZeroScripts(_data.scripts);
}

function _dropZeroModules() {
    //TODO
    // Modules are permanent cached in current implementation
}

function _dropZeroCss() {
    for (let name in _data.css) {
        if (_data.css[name] === 0) {
            let asset = lx.app.domSelector.getElementByAttrs({
                href: name,
                name: 'elem_asset'
            });
            asset.parentNode.removeChild(asset);
            delete _data.css[name];
        }
    }
}

function _dropZeroScripts() {
    for (let name in _data.scripts) {
        if (_data.scripts[name] === 0) {
            let asset = lx.app.domSelector.getElementByAttrs({
                src: name,
                name: 'elem_asset'
            });
            asset.parentNode.removeChild(asset);
            delete _data.scripts[name];
        }
    }
}

/**
 * @private
 *
 * @param self {Dependencies}
 * @param list {Array<String>}
 * @param key {String}
 * @param depend {Boolean}
 * @param callback {Function|null}
 * @returns {Array<String>}
 */
function _checkNeed(self, list, key, depend, callback = null) {
    if (!list || !list.length)
        return [];

    let map = {};
    map[key] = list
    let need = self.defineNecessary(map)[key];
    if (need.lxEmpty()) {
        if (depend) self.depend(map);
        if (callback) callback();
        return [];
    }

    return need;
}

/**
 * @private
 *
 * @param self {Dependencies}
 * @param config {Array<string>|Object: {
 *     [scripts] {Array<String>},
 *     [css] {Array<String>},
 *     [depend: false] {Boolean},
 *     [immediately: true] {Boolean},
 * }}
 * @param key {String}
 * @returns {lx.AssetRequest[]}
 */
function _promiseAsset(self, config, key) {
    if (lx.isArray(config)) {
        let obj = {};
        obj[key] = config;
        config = obj;
    }

    let depend = lx.getFirstDefined(config.depend, false),
        immediately = lx.getFirstDefined(config.immediately, true),
        need = _checkNeed(self, config[key], key, depend);
    if (!need.length) return [];

    let res = [];
    for (let i = 0; i < need.length; i++) {
        let src = need[i],
            req = new lx.AssetRequest(src);
        if (immediately) req.send();
        else res.push(req);
    }
    let dep = {};
    dep[key] = config[key];
    self.depend(dep);

    if (!res.length) return [];
    return res;
}
