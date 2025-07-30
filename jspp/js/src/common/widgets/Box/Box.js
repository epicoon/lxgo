// @lx:require -R positioningStrategies/;
// @lx:require -U tools;

/* * 1. Constructor
 * render(config)
 * clientRender(config)
 * postUnpack(config)
 * positioning()
 * static onresize()
 * isAutoParent()
 * begin()
 * end()
 *
 * * 2. Content managment
 * imagePath(name)
 * renderHtml()
 * addChild(elem, config = {})
 * modifyNewChildConfig(config)
 * insert(c, next, config={})
 * add(type, count=1, config={}, configurator={})
 * clear()
 * del(el, index, count)
 * text(text)
 * setEditable(value = true)
 * isEditable()
 * isEditing()
 * edit()
 * blur(e)
 * showOnlyChild(key)
 * scrollTo(adr)
 * getScrollPos()
 * getScrollSize()
 * checkResizeChild(callback)
 *
 * * 3. Content navigation
 * get(path)
 * getAll(path)
 * find(key)
 * findAll(key)
 * findOne(key)
 * contains(key)
 * childrenCount(key)
 * child(num)
 * lastChild()
 * divideChildren(info)
 * getChildren(info=false, all=false)
 * eachChild(func, all=false)
 *
 * * 4. PositioningStrategies
 * align(hor, vert, els)
 * stream(config)
 * streamProportional(config={})
 * getStreamDirection()
 * grid(config)
 * gridProportional(config={})
 * gridStream(config={})
 * gridAdaptive(config={})
 * slot(config)
 * setIndents(config)
 * tryChildReposition(elem, param, val)
 * childHasAutoresized(elem)
 *
 * * 5. Js-features
 * bind(model)
 * matrix(config)
 * agregator(c, toWidget=true, fromWidget=true)
 * 
 * Special events:
 * - beforeAddChild(child)
 * - afterAddChild(child)
 */

/**
 * @widget lx.Box
 *
 * @events [
 *     contentResize,
 *     beforeAddChild,
 *     afterAddChild,
 *     xScrollBarOn,
 *     xScrollBarOff,
 *     xScrollBarChange,
 *     yScrollBarOn,
 *     yScrollBarOff,
 *     yScrollBarChange,
 *     scrollBarChange,
 *     scroll,
 *     blur,
 *     beforeDestruct,
 *     afterDestruct
 * ]
 */
// @lx:namespace lx;
class Box extends lx.Rect {
    //==================================================================================================================
    /* 1. Constructor */
    modifyConfigBeforeApply(config) {
        if (config.matrix) {
            this._isMatrix = true;
            config.field = config.matrix;
        }
        return config;
    }

    /**
     * @widget-init
     *
     * @param [config] {Object: {
     *     #merge(lx.Rect::constructor::config),
     *     [matrix] {String},
     *     [text] {String},
     *     [align]              {Object: #schema(lx.AlignPositioningStrategy::applyConfig::config)},
     *     [stream]             {Object: #schema(lx.StreamPositioningStrategy::applyConfig::config)},
     *     [streamProportional] {Object: #schema(lx.StreamPositioningStrategy::applyConfig::config)},
     *     [grid]             {Object: #schema(lx.GridPositioningStrategy::applyConfig::config)},
     *     [gridProportional] {Object: #schema(lx.GridPositioningStrategy::applyConfig::config)},
     *     [gridStream]       {Object: #schema(lx.GridPositioningStrategy::applyConfig::config)},
     *     [gridAdaptive]     {Object: #schema(lx.GridPositioningStrategy::applyConfig::config)},
     *     [slot]             {Object: #schema(lx.SlotPositioningStrategy::applyConfig::config)}
     * }}
     */
    render(config) {
        super.render(config);

        if ( config.text ) this.text( config.text );

        if (config.align)
            lx.isArray(config.align)
                ? this.align.call(this, ...config.align)
                : this.align(lx.isObject(config.align) ? config.align : {});
        if (config.stream)
            this.stream(lx.isObject(config.stream) ? config.stream : {});
        else if (config.streamProportional)
            this.streamProportional(lx.isObject(config.streamProportional) ? config.streamProportional : {});
        else if (config.grid)
            this.grid(lx.isObject(config.grid) ? config.grid : {});
        else if (config.gridProportional)
            this.gridProportional(lx.isObject(config.gridProportional) ? config.gridProportional : {});
        else if (config.gridStream)
            this.gridStream(lx.isObject(config.gridStream) ? config.gridStream : {});
        else if (config.gridAdaptive)
            this.gridAdaptive(lx.isObject(config.gridAdaptive) ? config.gridAdaptive : {});
        else if (config.slot)
            this.slot(lx.isObject(config.slot) ? config.slot : {});
    }

    // @lx:<context CLIENT:
    __construct() {
        super.__construct();
        this.children = new BoxChildren();
        this.childrenByKeys = {};
    }

    clientRender(config) {
        super.clientRender(config);
        let sizes = this.getScrollSize();
        this.__sizeHolder.refreshContent(sizes.width, sizes.height);
        this.on('resize', lx(STATIC).onresize);
        this.on('scrollBarChange', lx(STATIC).onresize);
    }

    postUnpack(config) {
        super.postUnpack(config);
        if (this.lxExtract('__na'))
            this.positioningStrategy.actualize();
    }

    restoreLinks(loader) {
        if (this._container)
            this._container = loader.getWidget(this._container);
    }

    destruct() {
        this.trigger('beforeDestruct');
        this.setBuildMode(true);
        this.eachChild((child)=>{
            if (child.destructProcess) child.destructProcess();
        });
        this.setBuildMode(false);

        super.destruct();
        this.trigger('afterDestruct');
    }

    checkResize(e) {
        let resized = super.checkResize(e);
        if (resized) this.children.forEach(c=>c.checkResize(e));
        return resized;
    }

    checkContentResize(e) {
        if (e && e.boxInitiator === this) return;
        e = e || this.newEvent({
            boxInitiator: this
        });
        let sizes = this.getScrollSize(),
            res = this.__sizeHolder.refreshContent(sizes.width, sizes.height);
        if (res) {
            this.trigger('contentResize');
            const container = __getContainer(this);
            if (container !== this)
                container.trigger('contentResize');
            if (!this.checkResize(e))
                this.children.forEach(c=>c.checkResize(e));
        }
        return res;
    }
    // @lx:context>

    // @lx:<context SERVER:
    __construct() {
        super.__construct();
        this.__self.children = new BoxChildren();
        this.__self.childrenByKeys = {};
        this.__self.positioningStrategy = null;
    }

    get children() { return this.__self.children; }
    set children(attr) { this.__self.children = attr; }

    get childrenByKeys() { return this.__self.childrenByKeys; }
    set childrenByKeys(attr) { this.__self.childrenByKeys = attr; }

    get positioningStrategy() { return this.__self.positioningStrategy; }
    set positioningStrategy(attr) { this.__self.positioningStrategy = attr; }

    beforePack() {
        if (this.positioningStrategy !== null) {
            this.__ps = this.positioningStrategy.pack();
        }
        if (this._container) this._container = this._container.renderIndex;
    }
    // @lx:context>

    static getCommonEventNames() {
        return ['beforeAddChild', 'afterAddChild'];
    }

    static onresize() {
        this.positioning().actualize();
    }

    begin() {
        let c = this.getContainer();
        lx.app.setAutoParent(c);
        return true;
    }

    isAutoParent() {
        return (lx.app.getAutoParent() === this);
    }

    end() {
        let c = this.getContainer();
        lx.app.dropAutoParent(c);
        return true;
    }

    overflow(val) {
        if (val === undefined)
            return this.style('overflow');
        _eachContainer(this, c=>super.overflow.call(c, val));
    }
    /* 1. Constructor */
    //==================================================================================================================


    //==================================================================================================================
    /* 2. Content management */
    imagePath(name) {
        const c = __getContainer(this);
        let path = lx.app.imageManager.getPath(c, name);
        if (path) return path;
        return super.imagePath(name);
    }

    renderHtml() {
        const renderContent = function(node) {
            if (!node.children || node.children.isEmpty()) return node.domElem.content;
            let arr = [];
            let mode = node.__buildMode;
            node.setBuildMode(true);
            node.eachChild(child=>{
                if (child.domElem.rendered()) arr.push(child.domElem.outerHtml());
                else arr.push(render(child));
            });
            node.setBuildMode(mode);
            return arr.join('');
        }

        const render = function(node) {
            return node.domElem.getHtmlStringBegin() + renderContent(node) + node.domElem.getHtmlStringEnd();
        }

        return renderContent(this);
    }

    useRenderCache() {
        // @lx:<context CLIENT:
        if (this.renderCacheStatus !== undefined) return;
        this.stopPositioning();
        this.renderCacheStatus = true;
        this.renderCache = 0;
        let container = __getContainer(this);
        if (container !== this) {
            container.renderCacheStatus = true;
            container.renderCache = 0;
        }
        // @lx:context>
    }

    applyRenderCache() {
        // @lx:<context CLIENT:
        // If the element does not exist, there is nowhere to apply it. Most likely, this element itself is in the cache
        // and need to apply the cache to a higher level
        if (!this.getDomElem()) return;

        // If the cache collection has not started on the element
        if (!this.renderCacheStatus) return;

        delete this.renderCacheStatus;

        // If nothing was added
        if (this.renderCache === 0) {
            let container = __getContainer(this);
            // The contents may have changed
            if (container === this) {
                this.getChildren(c=>{if (c.applyRenderCache) c.applyRenderCache()});
                this.startPositioning();
            // The element is a wrapper over the container
            } else container.applyRenderCache();
            return;
        }

        let text = this.renderHtml();
        this.domElem.html(text);
        _refreshAfterRender(this);
        this.startPositioning();
        this.checkContentResize();
        // @lx:context>
    }

    /**
     * Enabling build mode sets the widget's own element as the main container
     */
    setBuildMode(bool) {
        if (bool) this.__buildMode = true;
        else delete this.__buildMode;
    }

    /**
     * Can be overridden in descendants to define a child element that will be responsible
     * for interaction with descendants added after the widget is created
     */
    _getContainer() {
        if (this._container) return this._container;
        return this;
    }

    getContainer() {
        let temp = this,
            container = temp._getContainer();
        while (container !== temp) {
            temp = container;
            container = temp._getContainer();
        }
        return temp;
    }
    
    setContainer(box) {
        this._container = box;
    }

    /**
     * The method used by a new widget to register with its parent
     */
    addChild(widget, config = {}) {
        // @lx:<context CLIENT:
        this.trigger('beforeAddChild', this.newEvent({child: widget}));
        // @lx:context>

        widget.parent = this;
        config = this.modifyNewChildConfig(config);
        let container = __getContainer(this);
        widget.domElem.setParent(container, config.nextSibling);

        // @lx:<context CLIENT:
        _checkParentRenderCache(this);
        if (this.renderCacheStatus) _addToRenderCache(container, widget);
        else widget.domElem.applyParent();
        // @lx:context>

        let tElem, clientHeight0, clientWidth0;
        if (container.getDomElem() && widget.getDomElem()) {
            tElem = container.getDomElem();
            clientHeight0 = tElem.clientHeight;
            clientWidth0 = tElem.clientWidth;
        }

        container.registerChild(widget, config.nextSibling);
        this.positioning().allocate(widget, config);

        // @lx:<context CLIENT:
        if (container.getDomElem() && widget.getDomElem()) {
            let clientHeight1 = tElem.clientHeight;
            let trigged = false;
            if (clientHeight0 > clientHeight1) {
                container.trigger('xScrollBarOn');
                container.trigger('xScrollBarChange');
                container.trigger('scrollBarChange');
                trigged = true;
            } else if (clientHeight0 < clientHeight1) {
                container.trigger('xScrollBarOff');
                container.trigger('xScrollBarChange');
                container.trigger('scrollBarChange');
                trigged = true;
            }

            let clientWidth1 = tElem.clientWidth;
            if (clientWidth0 > clientWidth1) {
                container.trigger('yScrollBarOn');
                container.trigger('yScrollBarChange');
                if (!trigged) container.trigger('scrollBarChange');
            } else if (clientWidth0 < clientWidth1) {
                container.trigger('yScrollBarOff');
                container.trigger('yScrollBarChange');
                if (!trigged) container.trigger('scrollBarChange');
            }

            this.checkContentResize();
            widget.trigger('displayin');
        }
        this.trigger('afterAddChild', this.newEvent({child: widget}));
        // @lx:context>
    }

    /**
     * Registering a new widget in the parent's (current widget's) structures
     * direct registration - without a container intermediary
     */
    registerChild(child, next) {
        if (next) this.children.insertBefore(child, next);
        else this.children.push(child);

        if (!child.key) return;

        if (child.key in this.childrenByKeys) {
            if (!lx.isArray(this.childrenByKeys[child.key])) {
                this.childrenByKeys[child.key]._index = 0;
                this.childrenByKeys[child.key] = [this.childrenByKeys[child.key]];
            }
            if (next && child.key == next.key) {
                child._index = next._index;
                this.childrenByKeys[child.key].splice(child._index, 0, child);
                for (let i=child._index+1,l=this.childrenByKeys[child.key].length; i<l; i++) {
                    this.childrenByKeys[child.key][i]._index = i;
                }
            } else {
                child._index = this.childrenByKeys[child.key].length;
                this.childrenByKeys[child.key].push(child);
            }
        } else this.childrenByKeys[child.key] = child;
    }

    /**
     * Preprocessing the config of the added element
     */
    modifyNewChildConfig(config) {
        return config;
    }

    /**
     * Options:
     * 1. el.add(lx.Box, config);
     * 2. el.add(lx.Box, 5, config, configurator);
     * 3. el.add([
     *        [lx.Box, config1],
     *        [lx.Box, 5, config2, configurator]
     *    ]);
     */
    add(type, count=1, config={}, configurator={}) {
        let conf = (lx.isObject(count)) ? count : config;
        if (conf.buildMode)
            return this.addStructure(type, count, config, configurator);
        return this.addContent(type, count, config, configurator);
    }

    addContainer() {
        if (!this.isEmpty())
            throw 'You can create container only in empty box';

        this.setContainer(this.add(lx.Box, {geom: true}));
    }

    addContent(type, count=1, config={}, configurator={}) {
        if (lx.isArray(type)) {
            let result = [];
            for (let i=0, l=type.len; i<l; i++)
                result.push( this.add.apply(this, type[i]) );
            return result;
        }

        if (lx.isObject(count)) {
            config = count;
            count = 1;
        }
        config.parent = this;
        delete config.buildMode;

        let result = (count == 1)
            ? new type(config)
            : type.construct(count, config, configurator);

        return result;
    }

    addStructure(type, count=1, config={}, configurator={}) {
        this.setBuildMode(true);
        this.addContent(type, count, config, configurator);
        this.setBuildMode(false);
    }

    /**
     * Removes all descendants
     */
    clear() {
        let container = __getContainer(this);
        // @lx:<context CLIENT:
        if (container.domElem.html() == '') return;
        // @lx:context>

        // First, all descendants must release the resources they are using
        container.eachChild((child)=>{
            if (child.destructProcess) child.destructProcess();
        });

        // After which we can reset the contents at once
        // @lx:<context CLIENT:
        container.domElem.html('');
        lx.DepthClusterMap.checkFrontMap();
        // @lx:context>

        container.children.reset();
        container.childrenByKeys = {};
        container.positioning().onClearOwner();
        // @lx:<context CLIENT:
        this.checkContentResize();
        // @lx:context>
    }

    /**
     * Options:
     * 1. Without arguments - removes the element on which the method is called
     * 2. Argument el - element - if there is one in the element on which the method is called, it will be removed
     * 3. Argument el - key (the only argument) - the element by key is deleted, if the key is an array,
     *    then all elements from this array are deleted
     * 4. Arguments el (key) + index - makes sense if the key is an array,
     *    the element with index in the array is removed from the array
     * 5. Arguments el (key) + index + count - like 4, but removes count elements starting at index
     */
    del(el, index, count) {
        if (el === undefined) return super.del();
        let c = this.remove(el, index, count);
        c.forEach(a=>a.destructProcess());
    }

    /**
     * Options:
     * 1. Argument el - element - if there is one in the element on which the method is called, it will be removed
     * 2. Argument el - key (the only argument) - the element by key is deleted, if the key is an array,
     *    then all elements from this array are deleted
     * 3. Arguments el (key) + index - makes sense if the key is an array,
     *    the element with index in the array is removed from the array
     * 4. Arguments el (key) + index + count - like 3, but removes count elements starting at index
     */
    remove(el, index, count) {
        const container = __getContainer(this);

        // el - object
        if (!lx.isString(el)) {
            // Do not delete someone else's element
            if (el.parent !== this) return false;

            // If the element has a key, we will delete it by key
            if (el.key && el.key in container.childrenByKeys) return this.remove(el.key, el._index, 1);

            // If there is no key
            let result = new lx.Collection(),
                pre = el.prevSibling();
            container.domElem.removeChild(el.domElem);
            container.children.remove(el);
            // @lx:<context CLIENT:
            lx.DepthClusterMap.checkFrontMap();
            // @lx:context>
            container.positioning().actualize({from: pre, deleted: [el]});
            container.positioning().onElemDel();
            result.add(el);
            // @lx:<context CLIENT:
            this.checkContentResize();
            // @lx:context>
            return result;
        }

        // el - key
        let key = el,
            result = new lx.Collection();
        if (!(key in container.childrenByKeys)) return result;

        // childrenByKeys[key] - not array
        if (!lx.isArray(container.childrenByKeys[key])) {
            let elem = container.childrenByKeys[key],
                pre = elem.prevSibling();
            container.domElem.removeChild(elem.domElem);
            container.children.remove(elem);
            // @lx:<context CLIENT:
            lx.DepthClusterMap.checkFrontMap();
            // @lx:context>
            delete container.childrenByKeys[key];
            container.positioning().actualize({from: pre, deleted: [elem]});
            container.positioning().onElemDel();
            result.add(elem);
            // @lx:<context CLIENT:
            this.checkContentResize();
            // @lx:context>
            return result;
        }

        // childrenByKeys[key] - array
        if (count === undefined) count = 1;
        if (index === undefined) {
            index = 0;
            count = container.childrenByKeys[key].length;
        } else if (index >= container.childrenByKeys[key].length) return result;
        if (index + count > container.childrenByKeys[key].length)
            count = container.childrenByKeys[key].length - index;

        let deleted = [],
            pre = container.childrenByKeys[key][index].prevSibling();
        for (let i=index,l=index+count; i<l; i++) {
            let elem = container.childrenByKeys[key][i];

            deleted.push(elem);
            container.domElem.removeChild(elem.domElem);
            container.children.remove(elem);
        }
        // @lx:<context CLIENT:
        lx.DepthClusterMap.checkFrontMap();
        // @lx:context>

        container.childrenByKeys[key].splice(index, count);
        for (let i=index,l=container.childrenByKeys[key].length; i<l; i++)
            container.childrenByKeys[key][i]._index = i;
        if (!container.childrenByKeys[key].length) {
            delete container.childrenByKeys[key];
        } else if (container.childrenByKeys[key].length == 1) {
            container.childrenByKeys[key] = container.childrenByKeys[key][0];
            delete container.childrenByKeys[key]._index;
        }
        container.positioning().actualize({from: pre, deleted});
        container.positioning().onElemDel();
        result.add(deleted);
        // @lx:<context CLIENT:
        this.checkContentResize();
        // @lx:context>
        return result;
    }

    text(text) {
        let container = __getContainer(this);

        if (text === undefined) {
            if ( !container.contains('text') ) return '';
            return container.get('text').value();
        }

        if (!container.contains('text')) new lx.TextBox({parent: container});

        container.get('text').value(text);
        return container;
    }

    setEditable(value = true) {
        if (value && !this.isEditable()) {
            let container = __getContainer(this);
            if (!container.contains('text'))
                this.add(lx.TextBox);
            container.get('text').setAttribute('contenteditable','true');
            container.on('click', _handlerEdit);
            container.get('text').on('blur', _handlerBlur);
            return;
        }
        if (!value && this.isEditable()) {
            let container = __getContainer(this);
            container.get('text').removeAttribute('contenteditable');
            container.off('click', _handlerEdit);
            container.get('text').off('blur', _handlerBlur);
        }
    }

    isEditable() {
        let container = __getContainer(this);
        if (!container.contains('text')) return false;
        return !!container.get('text').getAttribute('contenteditable');
    }
    
    isEditing() {
        return this.__isEditing;
    }

    edit() {
        if (!this.isEditable() || this.__isEditing) return;
        this.__isEditing = true;

        const textElem = this.get('text').getDomElem();
        textElem.focus();
        if (textElem.innerText.length) {
            let sel = window.getSelection();
            if (textElem.lastChild.innerText && /(\r\n|\r|\n)/.test(textElem.lastChild.innerText)) {
                sel.collapse(textElem.lastChild, 0);
                return;
            }
            sel.collapse(
                textElem.lastChild.innerText ? textElem.lastChild.firstChild : textElem.lastChild,
                textElem.lastChild.innerText ? textElem.lastChild.innerText.length : textElem.lastChild.length
            );
        }
    }

    showOnlyChild(key) {
        this.eachChild(c=>c.visibility(c.key == key));
    }

    scrollTo(adr) {
        if (!lx.isObject(adr)) adr = {y:adr};
        const c = __getContainer(this);

        if (adr.x !== undefined) c.domElem.param('scrollLeft', +adr.x);
        if (adr.y !== undefined) c.domElem.param('scrollTop', +adr.y);

        if (adr.xShift !== undefined) {
            let size = c.getScrollSize();
            let shift = Math.round((size.width - c.width('px')) * adr.xShift);
            c.domElem.param('scrollLeft', shift);
        }
        if (adr.yShift !== undefined) {
            let size = c.getScrollSize();
            let shift = Math.round((size.height - c.height('px')) * adr.yShift);
            c.domElem.param('scrollTop', shift);
        }

        this.trigger('scroll');
        return this;
    }

    getScrollPos() {
        const c = __getContainer(this);
        return {
            x: c.domElem.param('scrollLeft'),
            y: c.domElem.param('scrollTop')
        };
    }

    getScrollSize() {
        let c = __getContainer(this);
        if (!c.getDomElem())
            return {width: null, height: null};

        return {
            width: c.getDomElem().scrollWidth,
            height: c.getDomElem().scrollHeight
        };
    }

    // @lx:<context CLIENT:
    /**
     * @param [config] {Object: {
     *     condition {Fuction},
     *     hint {String},
     *     css {String}
     * }}
     */
    setEllipsisHint(config = {}) {
        config.condition = config.condition || function(box) {
            if (lx.isString(this.hint) || box.hintText) return true;
            const elem = box.get('text');
            if (!elem) return false;
            if (elem.getDomElem().offsetWidth == elem.getDomElem().scrollWidth) return false;
            return true;
        };
        config.hint = config.hint || function(box) {
            return box.hintText || box.get('text').html();
        }
        this.mouseover(()=>{
            if (!config.condition(this)) return;
            this.__hint = new lx.Box({
                geom: [0, 0, 'auto', 'auto'],
                css: config.css,
                depthCluster: lx.DepthClusterMap.CLUSTER_FRONT
            });
            let hint = '';
            if (lx.isString(config.hint)) hint = config.hint;
            else if (lx.isFunction(config.hint)) hint = config.hint(this);
            this.__hint.html(hint);
            this.__hint.satelliteTo(this);
        });
        this.mouseout(()=>{
            if (this.__hint) this.__hint.del();
        });
    }

    hasOverflow(direction = null) {
        if (!this.getDomElem()) return false;

        let c = __getContainer(this),
            scrollSize = c.getScrollSize();
        if (direction == lx.VERTICAL)
            return scrollSize.height - 1 > this.getDomElem().clientHeight;

        if (direction == lx.HORIZONTAL)
            return scrollSize.width - 1 > this.getDomElem().clientWidth;

        return scrollSize.height - 1 > this.getDomElem().clientHeight
            || scrollSize.width - 1 > this.getDomElem().clientWidth;
    }
    // @lx:context>
    /* 2. Content managment */
    //==================================================================================================================


    //==================================================================================================================
    /* 3. Content navigation */

    isEmpty() {
        let container = __getContainer(this);
        return container.children.isEmpty();
    }

    get(key) {
        let result = this.childrenByKeys[key];
        if (result) return result;

        let container = __getContainer(this);
        if (container !== this)
            result = container.childrenByKeys[key];
        if (result) return result;

        return null;
    }

    getAll(key) {
        return new lx.Collection(this.get(key));
    }

    getOne(key) {
        let c = lx.Collection.cast(this.get(key));
        if (c.isEmpty()) return null;
        return c.at(0);
    }

    find(key) {
        let c = this.getChildren({hasProperties:{key}, all:true});
        if (c.len == 1) return c.at(0);
        return c;
    }

    findAll(key) {
        let c = this.getChildren({hasProperties:{key}, all:true});
        return c;
    }

    findOne(key) {
        let c = lx.Collection.cast(this.find(key));
        if (c.isEmpty()) return null;
        return c.at(0);
    }

    getChildIndex(child) {
        let container = __getContainer(this);
        return container.children.indexOf(child);
    }

    contains(key) {
        let container = __getContainer(this);

        if (key instanceof lx.Rect) {
            if (key.key) {
                if (!(key.key in container.childrenByKeys)) return false;
                if (lx.isArray(container.childrenByKeys[key.key])) {
                    if (key._index === undefined) return false;
                    return container.childrenByKeys[key.key][key._index] === key;
                }
                return container.childrenByKeys[key.key] === key;
            } else {
                return container.children.contains(key);
            }
        }

        return (key in container.childrenByKeys);
    }

    childrenCount(key) {
        let container = __getContainer(this);

        if (key === undefined) return container.children.count();

        if (!container.childrenByKeys[key]) return 0;
        if (!lx.isArray(container.childrenByKeys[key])) return 1;
        return container.childrenByKeys[key].len;
    }

    child(num) {
        let container = __getContainer(this);
        if (lx.isNumber(num)) return container.children.get(num);
        if (lx.isFunction(num)) return container.children.getByCondition(num);
        return null;
    }

    lastChild() {
        let container = __getContainer(this);
        return container.children.last();
    }

    divideChildren(info) {
        const all = (info.all !== undefined) ? info.all : false;
        if (info.hasProperty) info.hasProperties = [info.hasProperty];
        const match = (info.notMatch === true) ? null : new lx.Collection();
        const notMatch = info.match === true ? null : new lx.Collection();
        function rec(el) {
            if (el === null || !el.childrenCount) return;
            for (let i=0; i<el.childrenCount(); i++) {
                const child = el.child(i);
                if (!child) continue;

                let matched = true;
                if (info.callback) matched = info.callback(child);

                if (matched && info.hasProperties) {
                    let prop = info.hasProperties;
                    if (lx.isObject(prop)) {
                        for (let j in prop) {
                            if (!(j in child)) { matched = false; break; }
                            let val = prop[j];
                            if (lx.isArray(val)) {
                                if (!val.includes(child[j])) { matched = false; break; }
                            } else {
                                if (child[j] != val) { matched = false; break; }
                            }
                        }
                    } else if (lx.isArray(prop)) {
                        for (let j=0, l=prop.len; j<l; j++)
                            if (!(prop[j] in child)) { matched = false; break; }
                    }
                }

                if (matched) {
                    if (match) match.add(child);
                } else {
                    if (notMatch) notMatch.add(child);
                }
                if (all) rec(child);
            }
        }
        rec(__getContainer(this));
        return {match, notMatch};
    }

    /**
     * Getting a collection of descendants given the passed conditions
     * Options:
     * 1. getChildren()  - will return his immediate descendants
     * 2. getChildren(true)  - will return all descendants, all nesting levels
     * 3. getChildren((a)=>{...})  - from its immediate descendants will return those for whom the callback returns true
     * 4. getChildren((a)=>{...}, true)  - from all its descendants will return those for whom the callback returns true
     * 5. getChildren({hasProperty:''}) | getChildren({hasProperties:[]})
     * 6. getChildren({hasProperties:[], all:true})
     * 7. getChildren({callback:(a)=>{...}})  - as 3
     * 8. getChildren({callback:(a)=>{...}, all:true})  - as 4
     */
    getChildren(info={}, all=false) {
        if (info === true) info = {all:true};
        if (lx.isFunction(info)) info = {callback: info, all};
        info.match = true;
        return this.divideChildren(info).match;
    }

    /**
     * Traversing all descendants without constructing intermediate structures is the most efficient method for this purpose
     */
    eachChild(func, all=false) {
        function re(elem) {
            if (!elem.child) return;
            let num = 0,
                child = elem.child(num);
            while (child) {
                func(child);
                if (all) re(child);
                child = elem.child(++num);
            }
        }
        re(__getContainer(this));
    }
    /* 3. Content navigation */
    //==================================================================================================================


    //==================================================================================================================
    /* 4. PositioningStrategies */
    positioning() {
        let container = __getContainer(this);
        if (container.positioningStrategy) return container.positioningStrategy;
        return new lx.PositioningStrategy(container);
    }

    stopPositioning() {
        let container = __getContainer(this);
        if (container.positioningStrategy) container.positioningStrategy.autoActualize = false;
    }

    startPositioning() {
        const container = __getContainer(this);
        if (container.positioningStrategy) {
            container.positioningStrategy.autoActualize = true;
            container.positioningStrategy.actualize();
        }
    }

    /**
     * @positioning lx.AlignPositioningStrategy
     * @param horizontal {Number&Enum(lx.LEFT, lx.CENTER, lx.RIGHT)}
     * @param vertical {Number&Enum(lx.TOP, lx.MIDDLE, lx.BOTTOM)}
     */
    align(horizontal, vertical) {
        let config = (vertical === undefined && lx.isObject(horizontal))
            ? horizontal
            : {horizontal, vertical};
        _preparePositioningStrategy(this, lx.AlignPositioningStrategy, config);
        return this;
    }

    /**
     * @positioning lx.MapPositioningStrategy
     * @param [config] {Object: #schema(lx.MapPositioningStrategy::init::config)}
     */
    map(config) {
        _preparePositioningStrategy(this, lx.MapPositioningStrategy, config);
        return this;
    }

    /**
     * @positioning lx.StreamPositioningStrategy[.TYPE_SIMPLE]
     * @param [config] {Object: #schema(lx.StreamPositioningStrategy::init::config)}
     */
    stream(config) {
        _preparePositioningStrategy(this, lx.StreamPositioningStrategy, config);
        return this;
    }

    /**
     * @positioning lx.StreamPositioningStrategy[.TYPE_PROPORTIONAL]
     * @param [config] {Object: #schema(lx.StreamPositioningStrategy::init::config)}
     */
    streamProportional(config={}) {
        config.type = lx.StreamPositioningStrategy.TYPE_PROPORTIONAL;
        return this.stream(config);
    }

    getStreamDirection() {
        if (!this.positioningStrategy || this.positioningStrategy.lxClassName() != 'StreamPositioningStrategy')
            return null;
        return this.positioningStrategy.direction;
    }

    /**
     * @positioning lx.GridPositioningStrategy[.TYPE_SIMPLE]
     * @param [config] {Object: #schema(lx.GridPositioningStrategy::init::config)}
     */
    grid(config) {
        _preparePositioningStrategy(this, lx.GridPositioningStrategy, config);
        return this;
    }

    /**
     * @positioning lx.GridPositioningStrategy[.TYPE_PROPORTIONAL]
     * @param [config] {Object: #schema(lx.GridPositioningStrategy::init::config)}
     */
    gridProportional(config={}) {
        config.type = lx.GridPositioningStrategy.TYPE_PROPORTIONAL;
        return this.grid(config);
    }

    /**
     * @positioning lx.GridPositioningStrategy[.TYPE_STREAM]
     * @param [config] {Object: #schema(lx.GridPositioningStrategy::init::config)}
     */
    gridStream(config={}) {
        config.type = lx.GridPositioningStrategy.TYPE_STREAM;
        return this.grid(config);
    }

    /**
     * @positioning lx.GridPositioningStrategy[.TYPE_ADAPTIVE]
     * @param [config] {Object: #schema(lx.GridPositioningStrategy::init::config)}
     */
    gridAdaptive(config={}) {
        config.type = lx.GridPositioningStrategy.TYPE_ADAPTIVE;
        return this.grid(config);        
    }

    /**
     * @positioning lx.SlotPositioningStrategy
     * @param [config] {Object: #schema(lx.SlotPositioningStrategy::init::config)}
     */
    slot(config) {
        _preparePositioningStrategy(this, lx.SlotPositioningStrategy, config);
        return this;
    }

    setIndents(config) {
        let container = __getContainer(this);
        if (!container.positioningStrategy) return this;
        container.positioningStrategy.setIndents(config);
        container.positioningStrategy.actualize();
        return this;
    }

    dropPositioning() {
        let container = __getContainer(this);
        if (container.positioningStrategy) {
            container.positioningStrategy.clear();
            container.positioningStrategy = null;
        }
    }

    tryChildReposition(elem, param, val) {
        let el = this.getDomElem(),
            container = __getContainer(this);

        if (lx.isObject(param)) {
            let config = param;
            if (!el) {
                for (let paramName in config)
                    container.positioning().tryReposition(elem, lx.Geom.geomConst(paramName), config[paramName]);
                return;
            }

            for (let paramName in config)
                container.positioning().tryReposition(elem, lx.Geom.geomConst(paramName), config[paramName]);
            // @lx:<context CLIENT:
            elem.checkResize();
            // @lx:context>
            return;
        }

        if (!el) {
            this.positioning().tryReposition(elem, param, val);
            return;
        }

        this.positioning().tryReposition(elem, param, val);
        // @lx:<context CLIENT:
        this.checkContentResize();
        // @lx:context>
    }

    childHasAutoresized(elem) {
        this.positioning().reactForAutoresize(elem);
    }
    /* 4. PositioningStrategies */
    //==================================================================================================================


    //==================================================================================================================
    /* 5. Client-features */
    // @lx:<context CLIENT:
    /**
     * If one argument
     * - or a collection of elements
     * - or full config transfer:
     * {
     * 	items: lx.Collection,
     * 	itemBox: [Widget, Config],
     * 	itemRender: function(itemBox, model) {}
     *  afterBind: function(itemBox, model) {}
     * 	type: bool
     * }
     * If three (two) arguments - short transfer of collection and callbacks:
     * - lx.Collection
     * - Function  - itemRender
     * - Function  - afterBind
     */
    matrix(...args) {
        let config;
        if (args.len == 1) {
            if (args[0] instanceof lx.Collection) config = {items: args[0]};
            else if (lx.isObject(args[0])) config = args[0];
        } else { config = {
            items: args[0],
            itemRender: args[1],
            afterBind: args[2]
        }; }

        lx.app.binder.makeWidgetMatrix(this, config);
        lx.app.binder.bindMatrix(
            config.items,
            this,
            lx.getFirstDefined(config.type, lx.app.binder.BIND_TYPE_FULL)
        );
    }

    /**
     * @param info {lx.Box.constructor|Tuple: [
     *     (: Widget constructor :) {lx.Box.constructor},
     *     (: Widget config :) {Object}
     * ]}
     */
    setMatrixItemBox(info) {
        lx.app.binder.setMatrixItemBox(this, info);
    }

    /**
     * @param render {Function} func(itemBox, model)
     */
    setMatrixItemRender(render) {
        lx.app.binder.setMatrixItemRender(this, render);
    }

    /**
     * @param render {Function} func(itemBox, model)
     */
    addMatrixItemRender(render) {
        lx.app.binder.addMatrixItemRender(this, render);
    }

    dropMatrix() {
        lx.app.binder.unbindMatrix(this);
    }

    agregator(c, type=lx.app.binder.BIND_TYPE_FULL) {
        lx.app.binder.bindAggregation(c, this, type);
    }
    // @lx:context>
    /* 6. Client-features */
    //==================================================================================================================
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * PRIVATE
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

function _preparePositioningStrategy(self, strategy, config) {
    let container = __getContainer(self);
    if (container.positioningStrategy) {
        container.positioningStrategy.clear();
        if (container.positioningStrategy.lxFullClassName() == strategy.lxFullName()) {
            container.positioningStrategy.init(config);
            return container.positioningStrategy;
        }
    }
    container.positioningStrategy = (strategy === lx.PositioningStrategy)
        ? null
        : new strategy(container);
    if (container.positioningStrategy)
        container.positioningStrategy.init(config);
    return container.positioningStrategy;
}

function __getContainer(self) {
    if (self.__buildMode) return self;
    return self.getContainer();
}

function _eachContainer(self, func) {
    func(self);
    let temp = self;
    let contaner = temp._getContainer();
    if (!contaner) return;
    while (contaner !== temp) {
        func(contaner);
        temp = contaner;
        contaner = temp._getContainer();
    }
}

// @lx:<context CLIENT:
function _checkParentRenderCache(self) {
    if (!self.renderCacheStatus && self.domElem.parent && self.domElem.parent.renderCacheStatus) {
        self.useRenderCache();
    }
}

function _addToRenderCache(self, widget) {
    if (self.renderCache === undefined)
        self.renderCache = 0;
    self.renderCache++;
}

function _refreshAfterRender(self) {
    if ( ! self.children) return;

    let mode = self.__buildMode;
    self.setBuildMode(true);
    let childNum = 0,
        elemNum,
        child = self.child(childNum),
        elemsList = self.getDomElem().children;
    while (child) {
        elemNum = childNum;
        let elem = elemsList[elemNum];
        child.domElem.refreshElem(elem);
        child.trigger('displayin');
        _refreshAfterRender(child);

        child = self.child(++childNum);
    }
    self.setBuildMode(mode);

    self.startPositioning();
    delete self.renderCacheStatus;
    delete self.renderCache;
}

function _handlerEdit(e) {
    this.overflow('auto');
    if (this.getDomElem() !== e.target) return;
    this.edit();
}

function _handlerBlur(e) {
    _triggerBlur(this.parent, e);
}

function _triggerBlur(self, e) {
    if (!self.__isEditing) return;

    let container = __getContainer(self);
    // container.overflow('hidden');

    let text = container.get('text').html();
    text = text.replaceAll('<div>', '\r\n');
    text = text.replaceAll('</div>', '');
    container.get('text').html(text);

    delete self.__isEditing;
    self.trigger('blur', self.newEvent({text}));
}
// @lx:context>
