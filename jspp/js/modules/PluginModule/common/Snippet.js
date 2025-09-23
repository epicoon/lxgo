// @lx:namespace lx;
class Snippet {
    // @lx:<context SERVER:
    /**
     * @param [config = {}] {Object: {
     *     [filePath] {String},
     *     [params] {Dict}
     * }}
     */
    constructor(config = {}) {
        this.filePath = config.filePath || '';
        this.params = _extractParams(config.params);
        this.data = {};
        this.metaData = {};

        // Snippet Box
        this._widget = new lx.Box({parent: null});
        this._widget.snippet = this;

        this.selfData = {};     // Box data
        this.htmlContent = '';  // Snippet html
        this.lx = [];           // Info for widgets
        this.clientJs = '';     // Client-side js-code

        this.plugins = [];  // Plugin dependencies
    }
    // @lx:context>
    // @lx:<context CLIENT:
    constructor(widget, info = {}) {
        this._widget = widget;
        widget.snippet = this;
        this.data = info.data || {};
        if (info.meta) _setMeta(this, info.meta);
    }
    // @lx:context>

    /**
     * @param snippetPath {String|Object: {
     *     plugin {String},
     *     snippet {String}
     * }}
     * @param [config] {Object: {
     *     [widget = lx.Box] {Function},
     *     [attributes] {Dict},
     *     [backLock = false] {Boolean},
     *     [backLockClickable = true] {Boolean},
     *     [hidden = false] {Boolean},
     *     [config] {Object}
     * }}
     */
    addSnippet(snippetPath, config = {}) {
        let widgetClass = config.widget || lx.Box,
            attributes = config.lxExtract('attributes') || {},
            backLock = lx.getFirstDefined(config.backLock, config.backLockClickable, false),
            backLockClickable = lx.getFirstDefined(config.backLockClickable, true),
            hidden = lx.getFirstDefined(config.hidden, false);
        config = (config.config) ? config.config : config;
        if (!config.key) {
            // Name uses path so change slashes
            config.key = lx.isString(snippetPath)
                ? snippetPath.replace('/', '_')
                : snippetPath.snippet.replace('/', '_');
        }

        let widget, head;
        if (backLock) {
            const wrapper = new lx.Box({ key: config.key + 'Wrapper', geom: true });
            const back = wrapper.add(lx.Box, {key:'backLock', geom:true});
            back.fill('black');
            back.opacity(0.5);
            widget = wrapper.add(widgetClass, config);
            wrapper.backLockClickable = backLockClickable;
            wrapper.onLoad(function() {
                this.child(1).show = () => this.show();
                this.child(1).hide = () => this.hide();
                if (this.backLockClickable) {
                    this.child(0).click(() => this.hide());
                    delete this.backLockClickable;
                }
            });
            head = wrapper;
        } else {
            widget = new widgetClass(config);
            head = widget;
        }

        widget.setSnippet({
            path: snippetPath,
            attributes
        });
        if (hidden) head.hide();

        return widget.snippet;
    }

    addSnippets(list, commonPath = '') {
        let result = [];
        for (let key in list) {
            let snippetConfig = list[key],
                path = '';

            if (lx.isNumber(key)) {
                if (lx.isObject(snippetConfig)) {
                    if (!snippetConfig.path) continue;
                    path = snippetConfig.path;
                } else if (lx.isString(snippetConfig)) {
                    path = snippetConfig;
                    snippetConfig = {};
                } else continue;
            } else if (lx.isString(key)) {
                path = key;
                if (!lx.isObject(snippetConfig)) snippetConfig = {};
            }

            if (snippetConfig.config) snippetConfig.config.key = path;
            else snippetConfig.key = path;

            if (commonPath != '' && commonPath[commonPath.length - 1] != '/') {
                commonPath += '/';
            }
            let snippetPath = lx.isString(path)
                ? commonPath + path
                : path;
            result.push(this.addSnippet(snippetPath, snippetConfig));
        }

        return result;
    }

    get widget() {
        return this._widget;
    }

    get(key) {
        return this.widget.get(key);
    }

    find(key) {
        return this.widget.find(key);
    }

    onLoad(code) {
        // @lx:<context SERVER:
        let js = lx.app.functionHelper.functionToString(code);
        js = js.replace(/^\([^)]*?\)=>/, '');
        this.clientJs = js;
        // @lx:context>
        // @lx:<context CLIENT:
        this.onLoadCallback = callback;
        // @lx:context>
    }

    setScreenModes(map) {
        // @lx:<context SERVER:
        this.metaData.sm = map;
        for (let i in this.metaData.sm)
            if (this.metaData.sm[i] == Infinity)
                this.metaData.sm[i] = 'INF';
        // @lx:context>
        // @lx:<context CLIENT:
        _setScreenModes(this, map);
        // @lx:context>
    }

    /* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
     * SERVER
     * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */
    // @lx:<context SERVER:
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

    /**
     * @param conf {Object: {
     *     name {String},
     *     anchor {String},
     *     [cssScope] {String},
     *     [params] {Dict},
     *     [onLoad] {Function}
     * }}
     */
    addPlugin(conf) {
        let re = function(obj) {
            if (lx.isFunction(obj)) return lx.app.functionHelper.functionToString(obj);
            if (!lx.isObject(obj)) return obj;
            for (let name in obj) obj[name] = re(obj[name]);
            return obj;
        };
        conf = re(conf);
        this.plugins.push(conf);
    }

    getSubPlugins() {
        return this.plugins;
    }

    getResult() {
        _prepareSelfData(this);
        _renderContent(this);
        return {
            data: this.data,
            selfData: this.selfData,
            html: this.htmlContent,
            lx: this.lx,
            js: this.clientJs,
            meta: this.metaData
        };
    }
    // @lx:context>

    /* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
     * CLIENT
     * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */
    // @lx:<context CLIENT:
    run() {
        if (this.screenWatcher) this.widget.trigger('resize');
        delete this.run;
    }

    setLoaded() {
        let code = this.onLoadCallback.toString();
        delete this.onLoadCallback;
        lx.app.functionHelper.createAndCallFunctionWithArguments({
            Plugin: this.widget.getPlugin(),
            Snippet: this
        }, code);
    }
    // @lx:context>
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * PRIVATE
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

// @lx:<context SERVER:
class PackData {
    constructor(widget) {
        this.data = {};
        this.widget = widget;
    }

    getResult() {
        this.packProperties();
        this.packHandlers();
        this.packOnLoad();
        return this.data;
    }

    packProperties() {
        for (let name in this.widget) {
            if (name == '__self' || name == 'lxid') continue;

            let value = this.widget[name];
            if (name == 'geom') {
                if (!value.lxEmpty()) {
                    let temp = [];
                    temp.push(value.bpg ? value.bpg[0]+','+value.bpg[1] : '');
                    temp.push(value.bpv ? value.bpv[0]+','+value.bpv[1] : '');
                    this.data.geom = temp.join('|');
                }
                continue;
            }

            this.data[name] = value;
        }
    }

    packHandlers() {
        if (this.widget.domElem.events.lxEmpty()) return;

        this.data.handlers = {};
        for (let name in this.widget.domElem.events) {
            let handlers = this.widget.domElem.events[name];
            this.data.handlers[name] = [];
            for (let i=0; i<handlers.len; i++) {
                let funcText = lx.app.functionHelper.functionToString(handlers[i]);
                if (funcText) this.data.handlers[name].push(funcText);
            }
        }
    }

    packOnLoad() {
        if (!this.widget.forOnload) return;

        this.data.forOnload = [];
        for (let i=0, l=this.widget.forOnload.len; i<l; i++) {
            let item = this.widget.forOnload[i], strItem;
            if (lx.isArray(item)) {
                strItem = lx.app.functionHelper.functionToString(item[0]);
                if (strItem) strItem = [strItem, item[1]];
            } else strItem = lx.app.functionHelper.functionToString(item);
        }

        if (strItem) this.data.forOnload.push(strItem);
    }
}

function _extractParams(params) {
    if (!params) return {};
    let result = {};
    if (lx.isArray(params) || lx.isObject(params))
        for (let i in params) result[i] = params[i];
    return result;
}

function _prepareSelfData(self) {
    let attrs = self.widget.domElem.attributes;
    if (!attrs.lxEmpty()) self.selfData.attrs = attrs;
    
    let classes = self.widget.domElem.classList;
    if (!classes.lxEmpty()) self.selfData.classes = classes;
    
    let style = self.widget.domElem.styleList;
    if (!style.lxEmpty()) self.selfData.style = style;

    self.widget.beforePack();
    let props = {};
    for (let name in self.widget) {
        if (name == '__self' || name == 'snippet') continue;
        props[name] = self.widget[name];
    }

    if (!props.lxEmpty()) self.selfData.props = props;
}

function _renderContent(self) {
    self.renderIndexCounter = 0;
    self.widget.children.forEach(a=>_setRenderIndex(self, a));

    let html = '';
    self.widget.children.forEach(a=>html+=_renderWidget(self, a));
    self.htmlContent = html;
}

function _setRenderIndex(self, widget) {
    widget.renderIndex = self.renderIndexCounter++;
    if (widget.children === undefined) return;
    widget.children.forEach(a=>_setRenderIndex(self, a));
}

function _renderWidget(self, widget) {
    _getWidgetData(self, widget);

    if (widget.children === undefined) return widget.domElem.getHtmlString();

    let result = widget.domElem.getHtmlStringBegin() + widget.domElem.content;
    widget.children.forEach(a=>result += _renderWidget(self, a));
    result += widget.domElem.getHtmlStringEnd();
    return result;
}

function _getWidgetData(self, widget) {
    widget.setAttribute('lx');
    widget.beforePack();
    let pack = new PackData(widget);
    self.lx.push(pack.getResult());
}

// @lx:context>

// @lx:<context CLIENT:
function _setMeta(self, meta) {
    if (meta.sm) _setScreenModes(self, meta.sm);
}

function _setScreenModes(self, map) {
    self.screenWatcher = new ScreenWatcher(self, map);
    if (!self.widget.hasTrigger('resize', _onResize))
        self.widget.on('resize', _onResize);
}

function _onResize(event) {
    let snippet = this.snippet;
    if (!snippet.screenWatcher.checkModeChange()) return;
    function rec(el) {
        el.trigger('changeScreenMode', event, snippet.screenWatcher.mode);
        if (!el.childrenCount) return;
        for (let i=0; i<el.childrenCount(); i++) {
            let child = el.child(i);
            if (!child || !child.getDomElem()) continue;
            rec(child);
        }
    }
    rec(this);
}

class ScreenWatcher {
    constructor(snippet, map) {
        this.snippet = snippet;
        let modes = [];
        for (let i in map) {
            let item = {
                name: i,
                lim: map[i]
            };
            if (item.lim == 'INF') item.lim = Infinity;
            modes.push(item);
        }
        modes.sort(function(a, b) {
            if (a.lim > b.lim) return 1;
            if (a.lim < b.lim) return -1;
            return 0;
        });
        this.map = modes;
        // this.identifyScreenMode();
    }

    checkModeChange() {
        let currentMode = this.mode;
        this.identifyScreenMode();
        return (currentMode != this.mode);
    }

    identifyScreenMode() {
        let w = this.snippet.widget.width('px'),
            mode;
        for (let i=0, l=this.map.length; i<l; i++) {
            if (w <= this.map[i].lim) {
                mode = this.map[i].name;
                break;
            }
        }
        this.mode = mode;
    };
}
// @lx:context>
