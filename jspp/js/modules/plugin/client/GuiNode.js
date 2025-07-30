// @lx:namespace lx;
class GuiNode {
    constructor(plugin, box) {
        this._plugin = plugin;
        this._box = box;
        this._box.guiNode = this;
        this._beforeShow = null;
        this._afterShow = null;
        this._beforeHide = null;
        this._afterHide = null;
        this.init();
        this.initHandlers();
        this.subscribeEvents();
    }

    /** @abstract */
    init() {}
    initHandlers() {}
    subscribeEvents() {}
    beforeShow() {}
    afterShow() {}
    beforeHide() {}
    afterHide() {}

    getPlugin() {
        return this._plugin;
    }
    
    getCore() {
        return this._plugin.core;
    }

    getGuiNode(name) {
        return this.getPlugin().getGuiNode(name);
    }

    getWidget() {
        return this._box;
    }

    getElem(key) {
        return this.getWidget().findOne(key);
    }

    on(name, callback) {
        switch (name) {
            case 'beforeShow': this._beforeShow = callback; break;
            case 'afterShow': this._afterShow = callback; break;
            case 'beforeHide': this._beforeHide = callback; break;
            case 'afterHide': this._afterHide = callback; break;
        }
    }

    show() {
        this.beforeShow();
        if (this._beforeShow)
            this._beforeShow();
        this.getWidget().show();
        this.afterShow();
        if (this._afterShow)
            this._afterShow();
    }

    hide(timer = null) {
        this.beforeHide();
        if (this._beforeHide)
            this._beforeHide();
        this.getWidget().hide(timer);
        this.afterHide();
        if (this._afterHide)
            this._afterHide();
    }

    get(path) {
        return this._box.get(path);
    }

    find(key, all=true) {
        return this._box.find(key, all);
    }

    triggerPluginEvent(eventName, data) {
        this.getPlugin().trigger(eventName, data);
    }
}
