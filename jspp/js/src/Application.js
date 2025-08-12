// @lx:require common/components/;
// @lx:<context CLIENT:
// @lx:require client/components/;
// @lx:context>

let _settings = {};
let _root = null;
let _autoParent = [];
let _ready = false;

/**
 * @private
 * @returns {Dict<lx.AppComponent>}
 */
function _componentsMap() {
    return {
        cssManager: lx.CssManager,
        imageManager: lx.ImageManager,
        functionHelper: lx.FunctionHelper,
        dependencies: lx.Dependencies,
        lang: lx.Language,
        // @lx:<context CLIENT:
        queues: lx.Queues,
        dialog: lx.Dialog,
        domSelector: lx.DomSelector,
        domEvents: lx.DomEvents,
        events: lx.Events,
        cookie: lx.Cookie,
        storage: lx.Storage,
        binder: lx.Binder,
        alert: lx.Alert,
        tost: lx.Tost,
        mouse: lx.Mouse,
        keyboard: lx.Keyboard,
        dragNDrop: lx.DragNDrop,
        animation: lx.Animation,
        // @lx:context>
    };
}

// @lx:namespace lx;
class Application {
    constructor() {
        this._components = {};
        this.registerComponents(_componentsMap());
    }

    /**
     * @param map {Dict<lx.AppComponent>}
     */
    registerComponents(map) {
        for (let key in map) {
            this._components[key] = new map[key](this);
            Object.defineProperty(this, key, {
                get: function() { return this._components[key]; }
            });
        }
    }

    isReady() {
        return _ready;
    }

    getProxy() {
        return _settings.proxy || null;
    }

    get root() {
        if (_root === null)
            throw "Application root widget is not defined";
        return _root;
    }

    getAutoParent() {
        if (_autoParent.length)
            return _autoParent.lxLast();
        return _root;
    }

    setAutoParent(box) {
        if (box === this.root || box === null)
            this.resetAutoParent();
        else _autoParent.push(box);
    }

    dropAutoParent(box) {
        if (box !== this.root && this.getAutoParent() === box)
            _autoParent.pop();
    }

    resetAutoParent() {
        _autoParent = [];
    }

    getSetting(name) {
        return _settings[name];
    }

    /**
     * @param {Object} list
     */
    setupComponents(list) {
        for (let name in list) {
            if (!(name in this._components)) {
                lx.logError('App does not have component ' + name);
                continue;
            }
            let comp = this._components[name];
            if (!(comp instanceof lx.AppComponentSettable)) {
                lx.logError('App component ' + name + ' is not settable');
                continue;
            }
            comp.addSettings(list[name]);
        }
    }

    // @lx:<context CLIENT:
    /**
     * @param [config] {Object:{
     *     [root] {lx.Box|Object|bool},
     *     [settings] {Dict},
     *     [components] {Object},
     *     [modulesCode] {String},
     * }}
     */
    start(config = {}) {
        if (!lx.CssTag.exists('lxbody')) {
            let css = new lx.CssTag({id:'lxbody'});
            css.addCss('.lxbody{position:absolute;left:0;top:0;width:100%;height:100%;overflow:hidden}.lx-abspos{position:absolute}.lxps-grid-v{display:grid;grid-auto-flow:row;grid-template-columns:1fr;grid-auto-rows:auto}.lxps-grid-h{display:grid;grid-auto-flow:column;grid-template-rows:1fr;grid-auto-columns:auto}input{overflow:hidden;visibility:inherit;box-sizing:border-box}div{overflow:visible;visibility:inherit;box-sizing:border-box;color:inherit}');
            css.commit();
        }

        if (config.settings)
            _settings = config.settings;

        this.domEvents.add(window, 'resize', e=>lx.app.root.checkResize(e));
        this.keyboard.setWatchForKeypress(true);
        this.dragNDrop.useElementMoving();
        this.animation.useTimers(true);
        this.animation.useAnimation();

        // Js-modules
        if (config.modulesCode && config.modulesCode != '')
            lx.app.functionHelper.createAndCallFunction('', config.modulesCode);

        if (config.components)
            this.setupComponents(config.components);

        if (config.root)
            _defineRoot(this, config.root);

        _setReady(this);

        // @lx:<mode DEV:
        let elems = document.getElementsByClassName('lx-alert');
        if (!elems.length) return;
        let elem = elems[0],
            text = elem.innerHTML;
        elem.offsetParent.removeChild(elem);
        lx.alert(text);
        // @lx:mode>
    }
    // @lx:context>

    // @lx:<context SERVER:
    start(config = {}) {
        if (config.settings) _settings = config.settings;
        if (config.components)
            this.setupComponents(config.components);
        this.i18nArray = {};
        if (config.root)
            _defineRoot(this, config.root);
        _setReady(this);
    }

    useI18n(config) {
        this.i18nArray['_' + lx.HashMd5.hex(config)] = config;
    }

    getDependencies() {
        return {
            i18n: this.i18nArray
        };
    }

    getResult() {
        return {};
    }
    // @lx:context>
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * PRIVATE
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

/**
 * @param app {lx.Application}
 * @param root {lx.Box|Object|boolean}
 */
function _defineRoot(app, root) {
    if (root instanceof lx.Box) {
        _root = root;
        return;
    }
    // @lx:<context CLIENT:
    if (root.attrs) {
        const el = app.domSelector.getElementByAttrs(root.attrs);
        if (el) _root = lx.Box.rise(el);
    }
    if (_root === null) {
        let rootConf = (root === true) ? {} : root;
        _root = _createBody(rootConf);
    }
    // @lx:context>
}

/**
 * @param conf {Object: {
 *     tag {string},
 *     attrs {Dict}
 * }}
 * @returns {lx.Box}
 */
function _createBody(conf) {
    const tag = document.createElement(conf.tag || 'div');
    if (conf.attrs)
        for (let key in conf.attrs)
            tag.setAttribute(key, conf.attrs[key]);
    tag.classList.add('lxbody')
    const body = document.getElementsByTagName('body')[0];
    if (body.children.length)
        body.insertBefore(tag, body.children[0]);
    else body.appendChild(tag);
    return lx.Box.rise(tag);
}

/**
 * @param app {lx.Application}
 */
function _setReady(app) {
    if (_ready) return;
    for (let name in app._components) {
        const component = app._components[name];
        component.onReady();
    }
    _ready = true;
}
