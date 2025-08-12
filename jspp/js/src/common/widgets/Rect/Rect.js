// @lx:require Geom;
// @lx:require DepthClusterMap;
// @lx:require -U tools;

/**
 * @widget lx.Rect
 * @content-disallowed
 *
 * @events [
 *     click,
 *     contextmenu,
 *     mousedown,
 *     mouseup,
 *     mousemove,
 *     mouseover,
 *     mouseout,
 *     transitionend,
 *     beforeDestruct,
 *     afterDestruct,
 *     resize,
 *     change,
 *     beforeHide,
 *     displayout,
 *     displayin,
 *     display
 * ]
 */
// @lx:namespace lx;
class Rect extends lx.Element {
    /**
     * @widget-init
     *
     * @param [config = {}] {Object: {
     *     [tag = 'div'] {String},
     *     [key] {String},
     *     [field] {String},
     *     [parent] {lx.Box},
     *     [before] {lx.Rect},
     *     [after] {lx.Rect},
     *     [geom] {Boolean|Tuple: [
     *         (:left position:) {String|Number|null},
     *         (:top position:) {String|Number|null},
     *         (:widget width:) {String|Number|null},
     *         (:widget height:) {String|Number|null},
     *         (:right position:) {String|Number|null|undefined},
     *         (:bottom position:) {String|Number|null|undefined}
     *     ]},
     *     [margin] {String|Number},
     *     [coords] {Tuple: [
     *         (:left position:) {String|Number},
     *         (:top position:) {String|Number}
     *     ]},
     *     [size] {Tuple: [
     *         (:widget width:) {String|Number},
     *         (:widget height:) {String|Number}
     *     ]},
     *     [left] {String|Number},
     *     [right] {String|Number},
     *     [top] {String|Number},
     *     [bottom] {String|Number},
     *     [width] {String|Number},
     *     [height] {String|Number},
     *     [html] {String},
     *     [css] {String|Array<String>},
     *     [cssScope] {String},
     *     [depthCluster] {String|Number&Enum(
     *         lx.DepthClusterMap.CLUSTER_DEEP,
     *         lx.DepthClusterMap.CLUSTER_PRE_MIDDLE,
     *         lx.DepthClusterMap.CLUSTER_MIDDLE,
     *         lx.DepthClusterMap.CLUSTER_PRE_FRONT,
     *         lx.DepthClusterMap.CLUSTER_FRONT,
     *         lx.DepthClusterMap.CLUSTER_PRE_OVER,
     *         lx.DepthClusterMap.CLUSTER_OVER,
     *         lx.DepthClusterMap.CLUSTER_URGENT
     *     )},
     *     [fill] {String},
     *     [opacity] {Number},
     *     [border] {Boolean|Object: {
     *         [width = 1] {Number},
     *         [color = '#000000'] {String},
     *         [style = 'solid'] {String},
     *         [side = 'ltrb'] {String}
     *     }},
     *     [picture] {String},
     *     [style] {Dict<String|Number>},
     *     [click] {Function},
     *     [blur] {Function},
     *     [move] {Boolean|Object: {
     *         [parentMove = false] {Boolean},
     *         [parentResize = false] {Boolean},
     *         [xMove = true] {Boolean},
     *         [yMove = true] {Boolean},
     *         [xLimit = true] {Boolean},
     *         [yLimit = true] {Boolean},
     *         [moveStep = 1] {Number},
     *         [locked = false] {Boolean}
     *     }},
     *     [parentResize] {Boolean},
     *     [parentMove] {Boolean},
     *     [data] {Dict<Any>}
     * }}
     */
    constructor(config = {}) {
        super();
        this.__construct();

        // @lx:<context CLIENT:
        if (config === false) return;
        // @lx:context>

        config = this.modifyConfigBeforeApply(config);

        this.defineDomElement(config);
        this.applyConfig(config);

        this.render(config);
        
        // @lx:<context CLIENT:
        this.clientRender(config);
        // @lx:context>

        if (this.style('z-index') === null)
            this.applyDepthCluster();
        this._type = this.lxClassName();
        let namespace = this.lxNamespace();
        if (namespace != 'lx') this._namespace = namespace;
    }

    // @lx:<context SERVER:
    __construct() {
        this.__self = {
            domElem: null,
            parent: null
        };
    }

    get domElem() { return this.__self.domElem; }
    set domElem(attr) { this.__self.domElem = attr; }

    get parent() { return this.__self.parent; }
    set parent(attr) { this.__self.parent = attr; }

    beforePack() { }
    // @lx:context>

    // @lx:<context CLIENT:
    __construct() {
        this.domElem = null;
        this.parent = null;
        this.__sizeHolder = new SizeHolder();
        this.clientInit();
    }
    // @lx:context>

    static getStaticTag() {
        return 'div';
    }

    modifyConfigBeforeApply(config) {
        return config;
    }

    /**
     * For client and server sides
     * @abstract
     */
    render(config={}) {
        // pass
    }

    /**
     * Directly creating a DOM element, establishing a connection with parent
     */
    defineDomElement(config) {
        if (this.getDomElem()) {
            this.log('already exists');
            return;
        }

        config.tag = config.tag || lx(STATIC).getStaticTag();
        this.domElem = new lx.DomElementDefinition(this, config);

        if (config.key) this.key = config.key;
        else if (config.field) this.key = config.field;

        this.setParent(config);
    }

    /**
     * data
     * field
     * html
     * css
     * cssScope
     * style
     * fill
     * opacity
     * border
     * picture
     * click
     * blur
     * move | parentResize | parentMove
     * depthCluster
     */
    applyConfig(config={}) {
        if (config.data) this.data = config.data;
        if (config.field) this._field = config.field;
        if (config.html) this.html(config.html);

        if (config.cssScope) this._cssScope = config.cssScope;
        if (config.css !== undefined) this.addClass(config.css);

        if (config.style)
            for (let i in config.style)
                this.domElem.style(i, config.style[i]);

        if (config.fill) this.fill(config.fill);
        if (config.opacity) this.opacity(config.opacity);
        if (config.border) this.border(config.border);
        if (config.picture) this.picture(config.picture);

        if (config.click) this.click( config.click );
        if (config.blur) this.blur( config.blur );

        if (config.move) this.move(config.move);
        else if (config.parentResize) this.move({parentResize: true});
        else if (config.parentMove) this.move({parentMove: true});
        
        if (config.depthCluster) this.depthCluster = config.depthCluster;

        return this;
    }

    /**
     * Static method for bulk creation of entities
     */
    static construct(count, config, configurator={}) {
        let c = new lx.Collection();

        // Optimizing Bulk Insert into Parent
        let parent = null;
        if (config.before) {
            parent = config.before.parent;
        } else if (config.after) {
            parent = config.after.parent;
        } else if (config.parent) parent = config.parent;
        if (parent === null && config.parent !== null)
            parent = lx.app.getAutoParent();

        config.parent = parent;
        if (parent) parent.useRenderCache();
        for (let i=0; i<count; i++) {
            let modifConfig = config;
            if (configurator.preBuild) modifConfig = configurator.preBuild.call(null, modifConfig, i);
            let obj = new this(modifConfig);
            c.add(obj);
            if (configurator.postBuild) configurator.postBuild.call(null, obj, i);
        };
        if (parent) parent.applyRenderCache();

        return c;
    }

    /**
     * Method for freeing resources
     */
    destructProcess() {
        // @lx:<context CLIENT:
        this.trigger('beforeDestruct');
        this.unbind();
        // @lx:context>
        this.destruct();
        this.parent = null;
        if (this.domElem) this.domElem.clear();
        this.domElem = null;
        // @lx:<context CLIENT:
        this.trigger('afterDestruct');
        // @lx:context>
    }

    destruct() {}

    // @lx:<context CLIENT:
    /** @abstract */
    clientInit() {
        // pass
    }

    /**
     * Method called in constructor to perform actions when entity is actually built, relationships (to parent) are built
     * The logic is common for creating an element both on the client and for restoring it when received from the server
     */
    clientRender(config={}) {
        this.__sizeHolder.refresh(this.width('px'), this.height('px'));
        this.on('scroll', ()=>this.checkDisplay());

        if (!config) return;
        if (config.disabled !== undefined) this.disabled(config.disabled);
    }

    checkResize(e) {
        let oldW = this.__sizeHolder.width,
            oldH = this.__sizeHolder.height,
            res = this.__sizeHolder.refresh(this.width('px'), this.height('px'));
        if (res) {
            e = e || this.newEvent({
                oldWidth: oldW,
                oldHeight: oldH
            });
            if (this.parent) this.parent.checkContentResize(e);
            this.trigger('resize', e);
        }
        return res;
    }

    /**
     * Startup Recovery
     */
    static rise(elem) {
        let el = new this(false);

        el.domElem = new lx.DomElementDefinition(el);
        el.domElem.setElem(elem);

        let data = {};
        for (let i = 0, n = elem.attributes.length; i < n; i++) {
            let name = elem.attributes[i].nodeName;
            if (name.match(/^data-/)) data[name.replace(/^data-/, '')] = elem.attributes[i].nodeValue
        }
        if (!data.lxEmpty()) el.data = data;

        return el;
    }
    // @lx:context>
    /* 1. Constructor */
    //==================================================================================================================

    
    //==================================================================================================================
    /* 2. Common */
    renderHtml() {
        return this.domElem.content;
    }

    bind(model, type=lx.app.binder.BIND_TYPE_FULL) {
        model.bind(this, type);
    }

    unbind() {
        lx.app.binder.unbindWidget(this);
    }

    get index() {
        if (this._index === undefined) return 0;
        return this._index;
    }

    setImagesMap(map) {
        this.imagesMap = map;
    }

    /**
     * Path for requesting images
     */
    imagePath(name) {
        return lx.app.imageManager.getPath(this, name) || name;
    }

    /**
     * Managing element activity
     */
    disabled(bool) {
        if (bool === undefined) return this.domElem.getAttribute('disabled') !== null;

        if (bool) this.domElem.setAttribute('disabled', '');
        else this.domElem.removeAttribute('disabled');
        return this;
    }

    getDefaultDepthCluster() {
        return lx.DepthClusterMap.CLUSTER_DEEP;
    }

    getDepthCluster() {
        if (this.depthCluster === undefined)
            return this.getDefaultDepthCluster();
        return this.depthCluster;
    }

    setDepthCluster(value) {
        if (value == this.getDefaultDepthCluster())
            delete this.depthCluster;
        else this.depthCluster = value;
        this.applyDepthCluster();
    }

    applyDepthCluster() {
        let zIndex = lx.DepthClusterMap.calculateZIndex(this.getDepthCluster());
        if (zIndex === 0) this.style('z-index', null);
        else this.style('z-index', zIndex);
    }

    emerge() {
        lx.DepthClusterMap.bringToFront(this);
    }
    /* 2. Common */
    //==================================================================================================================


    //==================================================================================================================
    /* 3. Html and Css */
    getDomElem() {
        if (!this.domElem) return null;
        return this.domElem.getElem();
    }

    tag() {
        return this.domElem.getTagName();
    }

    setAttribute(name, val = '') {
        if (val === null) {
            this.domElem.removeAttribute(name);
            return this;
        }

        this.domElem.setAttribute(name, val);
        return this;
    }

    getAttribute(name) {
        return this.domElem.getAttribute(name);
    }

    removeAttribute(name) {
        return this.domElem.removeAttribute(name);
    }

    style(name, val) {
        if (name === undefined) return this.domElem.style();

        if (lx.isObject(name)) {
            for (let i in name) this.style(i, name[i]);
            return this;
        }

        if (val === undefined) return this.domElem.style(name);

        this.domElem.style(name, val);
        return this;
    }

    html(content) {
        if (content == undefined)
            return this.domElem.html();

        this.domElem.html(content);
        return this;
    }

    /**
     * @returns {string}
     */
    getCssScope() {
        if (this._cssScope !== undefined) return this._cssScope;
        let ancestor = this.ancestor({hasProperty: '_cssScope'});
        if (ancestor) return ancestor._cssScope;
        return '';
    }

    /**
     * There are two ways to pass arguments:
     * 1. elem.addClass(class1, class2);
     * 2. elem.addClass([class1, class2]);
     */
    addClass(...args) {
        if (lx.isArray(args[0])) args = args[0];

        args = lx.app.cssManager.defineCssClassNames(this, args);
        args.forEach(name=>{
            if (name === null || name == '') return;
            this.domElem.addClass(name);
        });
        return this;
    }

    /**
     * There are two ways to pass arguments:
     * 1. elem.removeClass(class1, class2);
     * 2. elem.removeClass([class1, class2]);
     */
    removeClass(...args) {
        if (lx.isArray(args[0])) args = args[0];

        args = lx.app.cssManager.defineCssClassNames(this, args);
        args.forEach(name=>{
            if (name == '') return;
            this.domElem.removeClass(name);
        });
        return this;
    }

    /**
     * Check if an element has a CSS class
     */
    hasClass(name) {
        return this.domElem.hasClass(name);
    }

    /**
     * If an element has one of the classes, it will be replaced with the second one
     * If only one class is passed, it will be set if it was not there, or removed if the element had one
     */
    toggleClass(class1, class2 = '') {
        if (this.hasClass(class1)) {
            this.removeClass(class1);
            this.addClass(class2);
        } else {
            this.addClass(class1);
            this.removeClass(class2);
        }
    }

    /**
     * If condition == true - the first class is applied, the second is removed
     * If condition == false - the second class is applied, the first is removed
     */
    toggleClassOnCondition(condition, class1, class2 = null) {
        if (condition) {
            this.addClass(class1);
            if (class2) this.removeClass(class2);
        } else {
            this.removeClass(class1);
            if (class2) this.addClass(class2);
        }
    }

    /**
     * Remove all classes
     */
    clearClasses() {
        this.domElem.clearClasses();
        return this;
    }

    opacity(val) {
        if (val != undefined) {
            this.domElem.style('opacity', val);
            return this;
        }
        return this.domElem.style('opacity');
    }

    fill(color) {
        this.domElem.style('backgroundColor', color);
        return this;
    }

    overflow(val) {
        if (val === undefined)
            return this.style('overflow');
        this.domElem.style('overflow', val);
        return this;
    }

    picture(pic) {
        if (pic === undefined)
            return this.domElem.style.backgroundImage.split('"')[1];

        if (pic === '' || !pic) this.domElem.style('backgroundImage', 'url()');
        else {
            let path = this.imagePath(pic);
            this.domElem.style('backgroundImage', 'url(' + path + ')');
            this.domElem.style('backgroundRepeat', 'no-repeat');
            this.domElem.style('backgroundSize', '100% 100%');
        }
        return this;
    }

    border( info ) {  // info = {width, color, style, side}
        if (info == undefined) info = {};
        let width = ( (info.width != undefined) ? info.width: 1 ) + 'px',
            color = (info.color != undefined) ? info.color: '#000000',
            style = (info.style != undefined) ? info.style: 'solid',
            sides = (info.side != undefined) ? info.side: 'ltrb',
            side = [false, false, false, false],
            sideName = ['Left', 'Top', 'Right', 'Bottom'];
        side[0] = (sides.search('l') != -1);
        side[1] = (sides.search('t') != -1);
        side[2] = (sides.search('r') != -1);
        side[3] = (sides.search('b') != -1);

        if (side[0] && side[1] && side[2] && side[3]) {
            this.domElem.style('borderWidth', width);
            this.domElem.style('borderColor', color);
            this.domElem.style('borderStyle', style);
        } else {
            for (let i=0; i<4; i++) if (side[i]) {
                this.domElem.style('border' + sideName[i] + 'Width', width);
                this.domElem.style('border' + sideName[i] + 'Color', color);
                this.domElem.style('border' + sideName[i] + 'Style', style);
            }
        }
        // @lx:<context CLIENT:
        this.checkResize();
        // @lx:context>
        return this;
    }

    /**
     * Argument options val:
     * 1. Number - rounding on all corners in pixels
     * 2. Object: {
     *     side: string  // specifying corners for rounding in the form 'tlbr'
     *     value: int    // rounding on all corners in pixels
     * }
     */
    roundCorners(val) {
        let arr = [];
        if (lx.isObject(val)) {
            let t = false, b = false, l = false, r = false;
            if ( val.side.indexOf('tl') != -1 ) { t = true; l = true; arr.push('TopLeft'); }
            if ( val.side.indexOf('tr') != -1 ) { t = true; r = true; arr.push('TopRight'); }
            if ( val.side.indexOf('bl') != -1 ) { b = true; l = true; arr.push('BottomLeft'); }
            if ( val.side.indexOf('br') != -1 ) { b = true; r = true; arr.push('BottomRight'); }
            if ( !t && val.side.indexOf('t') != -1 ) { arr.push('TopLeft'); arr.push('TopRight'); }
            if ( !b && val.side.indexOf('b') != -1 ) { arr.push('BottomLeft'); arr.push('BottomRight'); }
            if ( !l && val.side.indexOf('l') != -1 ) { arr.push('TopLeft'); arr.push('BottomLeft'); }
            if ( !r && val.side.indexOf('r') != -1 ) { arr.push('TopRight'); arr.push('BottomRight'); }
            val = val.value;
        }
        if (lx.isNumber(val)) val += 'px';

        if (!arr.length) this.domElem.style('borderRadius', val);

        for (let i=0; i<arr.length; i++)
            this.domElem.style('border' + arr[i] + 'Radius', val);

        return this;
    }

    rotate(angle) {
        this.domElem.style('mozTransform', 'rotate(' + angle + 'deg)');    // Firefox
        this.domElem.style('msTransform', 'rotate(' + angle + 'deg)');     // IE
        this.domElem.style('webkitTransform', 'rotate(' + angle + 'deg)'); // Safari, Chrome, iOS
        this.domElem.style('oTransform', 'rotate(' + angle + 'deg)');      // Opera
        this.domElem.style('transform', 'rotate(' + angle + 'deg)');
        return this;
    }

    visibility(vis) {
        if (vis !== undefined) { vis ? this.show(): this.hide(); return this; }

        if ( !this.domElem.style('visibility') || this.domElem.style('visibility') == 'inherit' ) {
            let p = this.domElem.parent;
            while (p) { if (p.domElem.style('visibility') == 'hidden') return false; p = p.domElem.parent; }
            return true;
        } else return (this.domElem.style('visibility') != 'hidden')
    }

    show() {
        this.domElem.style('visibility', 'inherit');
        // @lx:<context CLIENT:
        this.trigger('beforeShow');
        this.checkDisplay();
        // @lx:context>
        return this;
    }

    hide(timer = null) {
        if (timer === null) this.domElem.style('visibility', 'hidden');
        else setTimeout(()=>this.domElem.style('visibility', 'hidden'), timer);
        // @lx:<context CLIENT:
        this.trigger('beforeHide');
        this.checkDisplay();
        // @lx:context>
        return this;
    }

    toggleVisibility() {
        if (this.visibility()) this.hide();
        else this.show();
    }

    // @lx:<context CLIENT:
    setDomElement(elem) {
        this.del();
        this.domElem.setElem(elem);
        return this;
    }
    // @lx:context>
    /* 3. Html and Css */
    //==================================================================================================================


    //==================================================================================================================
    /* 4. Geometry */
    /**
     * Size without frames, scroll bars, etc.
     */
    getInnerSize(param) {
        if (param === undefined) return [
            this.domElem.param('clientWidth'),
            this.domElem.param('clientHeight')
        ];
        if (param == lx.HEIGHT) return this.domElem.param('clientHeight');
        if (param == lx.WIDTH) return this.domElem.param('clientWidth');
    }
    
    resize(config) {
        if (this.parent) this.parent.tryChildReposition(this, config);
        else {
            let w, h;
            if (this.getDomElem()) {
                w = this.getDomElem().clientWidth;
                h = this.getDomElem().clientHeight;
            }
            for (let paramName in config)
                this.setGeomParam(lx.Geom.geomConst(paramName), config[paramName]);
            // @lx:<context CLIENT:
            if (this.getDomElem() && (w != this.getDomElem().clientWidth || h != this.getDomElem().clientHeight))
                this.checkResize();
            // @lx:context>
        }
        return this;
    }

    /**
     * Setting a value to a geometric parameter according to the parent positioning strategy
     */
    setGeomParam(param, val) {
        /* TODO
         * Here the positioning strategy for descendants is not updated - hence it cannot, because Rect has no descendants.
         * It is necessary to make this method a template - the operator this.parent.tryChildReposition(this, param, val); move it to a separate method,
         * and redefine it in Box to update the strategy there.
         * !BUT it's not a fact that it should be updated here at all
         */
        if (this.parent) this.parent.tryChildReposition(this, param, val);
        else {
            this.setGeomPriority(param);
            this.domElem.style(lx.Geom.geomName(param), val);
        }
        return this;
    }

    /**
     * If the size has changed due to internal processes, this must be reported "up" in the hierarchy
     */
    reportSizeHasChanged() {
        if (this.parent) this.parent.childHasAutoresized(this);
    }

    left(val) {
        if (val === undefined || val == '%' || val == 'px')
            return __getLeft(this, val);
        return this.setGeomParam(lx.LEFT, val);
    }

    right(val) {
        if (val === undefined || val == '%' || val == 'px')
            return __getRight(this, val);
        return this.setGeomParam(lx.RIGHT, val);
    }

    top(val) {
        if (val === undefined || val == '%' || val == 'px')
            return __getTop(this, val);
        return this.setGeomParam(lx.TOP, val);
    }

    bottom(val) {
        if (val === undefined || val == '%' || val == 'px')
            return __getBottom(this, val);
        return this.setGeomParam(lx.BOTTOM, val);
    }

    width(val) {
        if (val === undefined || val == '%' || val == 'px')
            return __getWidth(this, val);
        return this.setGeomParam(lx.WIDTH, val);
    }

    height(val) {
        if (val === undefined || val == '%' || val == 'px')
            return __getHeight(this, val);
        return this.setGeomParam(lx.HEIGHT, val);
    }

    coords(l, t) {
        if (l === undefined) return [ this.left(), this.top() ];
        if (t === undefined) return [ this.left(l), this.top(l) ];
        this.left(l);
        this.top(t);
        return this;
    }

    size(w, h) {
        if (w === undefined) return [ this.width(), this.height() ];
        if (h === undefined) return [ this.width(w), this.height(w) ];
        this.width(w);
        this.height(h);
        return this;
    }

    setGeom(geom) {
        let priorityH = [],
            priorityV = [];
        if (geom[0] !== null && geom[0] !== undefined) priorityH.push(lx.LEFT);
        if (geom[1] !== null && geom[1] !== undefined) priorityV.push(lx.TOP);
        if (geom[2] !== null && geom[2] !== undefined) priorityH.push(lx.WIDTH);
        if (geom[3] !== null && geom[3] !== undefined) priorityV.push(lx.HEIGHT);
        if (geom[4] !== null && geom[4] !== undefined) priorityH.push(lx.RIGHT);
        if (geom[5] !== null && geom[5] !== undefined) priorityV.push(lx.BOTTOM);
        if (priorityH.len < 3) __setGeomPriorityH(this, priorityH[0], priorityH[1]);
        if (priorityV.len < 3) __setGeomPriorityV(this, priorityV[0], priorityV[1]);

        if (geom[0] !== undefined) this.left(geom[0]);
        if (geom[1] !== undefined) this.top(geom[1]);
        if (geom[2] !== undefined) this.width(geom[2]);
        if (geom[3] !== undefined) this.height(geom[3]);
        if (geom[4] !== undefined) this.right(geom[4]);
        if (geom[5] !== undefined) this.bottom(geom[5]);

        // @lx:<context CLIENT:
        this.checkResize();
        // @lx:context>
    }

    getGeomMask(units = undefined) {
        let result = {};
        result.pH = __getGeomPriorityH(this).lxClone();
        result.pV = __getGeomPriorityV(this).lxClone();
        result[result.pH[0]] = this[lx.Geom.geomName(result.pH[0])](units);
        result[result.pH[1]] = this[lx.Geom.geomName(result.pH[1])](units);
        result[result.pV[0]] = this[lx.Geom.geomName(result.pV[0])](units);
        result[result.pV[1]] = this[lx.Geom.geomName(result.pV[1])](units);
        if (this.geom) result.geom = this.geom.lxClone();
        return result;
    }

    /**
     * Copy "as is" - with priorities, without adaptations to old corresponding values
     */
    copyGeom(geomMask, units = undefined) {
        if (geomMask instanceof lx.Rect) geomMask = geomMask.getGeomMask(units);
        let pH = geomMask.pH,
            pV = geomMask.pV;
        this.setGeomParam(pH[1], units ? geomMask[pH[1]] + units : geomMask[pH[1]]);
        this.setGeomParam(pH[0], units ? geomMask[pH[0]] + units : geomMask[pH[0]]);
        this.setGeomParam(pV[1], units ? geomMask[pV[1]] + units : geomMask[pV[1]]);
        this.setGeomParam(pV[0], units ? geomMask[pV[0]] + units : geomMask[pV[0]]);
        if (geomMask.geom) this.geom = geomMask.geom.lxClone();
        // @lx:<context CLIENT:
        this.checkResize();
        // @lx:context>
        return this;
    }

    /**
     * Copy "as is" - with priorities, without adaptations to old corresponding values
     */
    copyGlobalGeom(geomMask, units = undefined) {
        //TODO units not working - __getGlobalGeomMask returns pixels
        if (geomMask instanceof lx.Rect) geomMask = __getGlobalGeomMask(geomMask);

        let pH = geomMask.pH,
            pV = geomMask.pV;
        this.setGeomParam(pH[1], units ? geomMask[pH[1]] + units : geomMask[pH[1]]);
        this.setGeomParam(pH[0], units ? geomMask[pH[0]] + units : geomMask[pH[0]]);
        this.setGeomParam(pV[1], units ? geomMask[pV[1]] + units : geomMask[pV[1]]);
        this.setGeomParam(pV[0], units ? geomMask[pV[0]] + units : geomMask[pV[0]]);
        if (geomMask.geom) this.geom = geomMask.geom.lxClone();
        // @lx:<context CLIENT:
        this.checkResize();
        // @lx:context>
        return this;
    }

    setGeomPriority(param1, param2) {
        let dir1 = lx.Geom.directionByGeom(param1),
            dir2 = lx.Geom.directionByGeom(param2);
        if (dir1 == lx.HORIZONTAL) {
            if (dir2 == lx.HORIZONTAL) {
                __setGeomPriorityH(this, param1, param2);
            } else if (dir2 == lx.VERTICAL) {
                __setGeomPriorityH(this, param1);
                __setGeomPriorityV(this, param2);
            } else {
                __setGeomPriorityH(this, param1);
            }
        } else if (dir1 == lx.VERTICAL) {
            if (dir2 == lx.VERTICAL) {
                __setGeomPriorityV(this, param1, param2);
            } else if (dir2 == lx.HORIZONTAL) {
                __setGeomPriorityV(this, param1);
                __setGeomPriorityH(this, param2);
            } else {
                __setGeomPriorityV(this, param1);
            }
        }
    }

    /**
     * Calculates a percentage or pixel representation of the size passed in any format, with the direction specified - width or height
     * Example:
     * elem.geomPart('50%', 'px', lx.VERTICAL) - returns half the height of the element in pixels
     */
    geomPart(val, unit, direction) {
        if (lx.isNumber(val)) return +val;
        if (!lx.isString(val)) return NaN;

        let num = parseFloat(val),
            baseUnit = val.split(num)[1];

        if (baseUnit == unit) return num;

        if (unit == '%') {
            return (direction == lx.HORIZONTAL)
                ? (num * 100) / this.width('px')
                : (num * 100) / this.height('px');
        }

        if (unit == 'px') {
            return (direction == lx.HORIZONTAL)
                ? num * this.width('px') * 0.01
                : num * this.height('px') * 0.01;
        }

        return NaN;
    }

    rect(format='px') {
        let l = this.left(format),
            t = this.top(format),
            w = this.width(format),
            h = this.height(format);
        return {
            left: l,
            top: t,
            width: w,
            height: h,
            right: l + w,
            bottom: t + h
        }
    }
    
    getRelativeRect(elem) {
        let globalRect = this.getGlobalRect(),
            pGlobalRect = elem.getGlobalRect();
        return {
            top: globalRect.top - pGlobalRect.top,
            left: globalRect.left - pGlobalRect.left,
            width: globalRect.width,
            height: globalRect.height,
            bottom: globalRect.bottom - pGlobalRect.bottom,
            right: globalRect.right - pGlobalRect.right,
        };
    }

    getGlobalRect() {
        let elem = this.getDomElem();
        if (!elem) return {};
        let rect = elem.getBoundingClientRect();

        return {
            top: rect.top,
            left: rect.left,
            width: rect.width,
            height: rect.height,
            bottom: document.documentElement.scrollHeight - rect.bottom,
            right: document.documentElement.scrollWidth - rect.right
        };
    }

    containPoint(x, y) {
        let rect = this.rect();
        return (
            x >= rect.left
            && x <= (rect.left + rect.width)
            && y >= rect.top
            && y <= (rect.top + rect.height)
        );
    }

    containGlobalPoint(x, y) {
        let rect = this.getGlobalRect();
        return (
            x >= rect.left
            && x <= (rect.left + rect.width)
            && y >= rect.top
            && y <= (rect.top + rect.height)
        );
    }

    globalPointToInner(point) {
        let y = point.lxGetFirstDefined(['y', 'clientY'], null),
            x = point.lxGetFirstDefined(['x', 'clientX'], null);
        if (x === null || y === null) return false;

        let rect = this.getGlobalRect();
        return {
            x: x - rect.left,
            y: y - rect.top
        };
    }

    checkCross(elem) {
        let r1, r2;
        if (this.parent === elem.parent) {
            r1 = this.rect();
            r2 = elem.rect();
        } else {
            r1 = this.getGlobalRect();
            r2 = elem.getGlobalRect();
        }
        return (
            r1.top < (r2.top + r2.height)
            && (r1.top + r1.height) > r2.top
            && r1.left < (r2.left + r2.width)
            && (r1.left + r1.width) > r2.left
        );
    }

    /**
     * Checking if an element is out of view up the parent hierarchy
     * The parent to start checking from can be passed explicitly (e.g. if the immediate parent obviously contains the given element outside its geometry)
     */
    isOutOfVisibility(el = null) {
        if (el === null) el = this.domElem.parent;

        let result = {},
            rect = this.getGlobalRect(),
            l = rect.left,
            r = rect.left + rect.width,
            t = rect.top,
            b = rect.top + rect.height,
            p = el;

        while (p) {
            let pRect = p.getGlobalRect(),
                elem = p.getDomElem(),
                pL = pRect.left,
                pR = pRect.left + pRect.width + elem.clientWidth - elem.offsetWidth,
                pT = pRect.top,
                pB = pRect.top + pRect.height + elem.clientHeight - elem.offsetHeight;

            if (l < pL) result.left   = pL - l;
            if (r > pR) result.right  = pR - r;
            if (t < pT) result.top    = pT - t;
            if (b > pB) result.bottom = pB - b;

            if (!result.lxEmpty()) {
                result.element = p;
                return result;
            }

            p = p.domElem.parent;
        }

        return result;
    }

    _parentScreenParams() {
        if (!this.domElem.parent) {
            let left = window.pageXOffset || document.documentElement.scrollLeft,
                width = window.screen.availWidth,
                right = left + width,
                top = window.pageYOffset || document.documentElement.scrollTop,
                height = window.screen.availHeight,
                bottom = top + height;
            return { left, right, width, height, top, bottom };
        }

        let elem = this.domElem.parent.getDomElem(),
            left = elem.scrollLeft,
            width = elem.offsetWidth,
            right = left + width,
            top = elem.scrollTop,
            height = elem.offsetHeight,
            bottom = top + height;

        return { left, right, width, height, top, bottom };
    }

    /**
     * Checking if an element extends beyond its parent
     */
    isOutOfParentScreen() {
        let p = this.domElem.parent,
            rect = this.rect('px'),
            geom = this._parentScreenParams(),
            result = {};

        if (rect.left < geom.left) result.left = geom.left - rect.left;
        if (rect.right > geom.right)  result.right = geom.right - rect.right;
        if (rect.top < geom.top)  result.top = geom.top - rect.top;
        if (rect.bottom > geom.bottom) result.bottom = geom.bottom - rect.bottom;

        if (result.lxEmpty()) return false;
        return result;
    }

    returnToParentScreen() {
        let out = this.isOutOfParentScreen();
        if (out.lxEmpty()) return this;

        if (out.left && out.right) {
        } else {
            if (out.left) this.left( this.left('px') + out.left + 'px' );
            if (out.right) this.left( this.left('px') + out.right + 'px' );
        }
        if (out.top && out.bottom) {
        } else {
            if (out.top) this.top( this.top('px') + out.top + 'px' );
            if (out.bottom) this.top( this.top('px') + out.bottom + 'px' );
        }
        return this;
    }

    checkDisplay(event) {
        __triggerDisplay(this, event);

        if (this.setBuildMode) this.setBuildMode(true);
        if (this.childrenCount) for (let i=0; i<this.childrenCount(); i++) {
            let child = this.child(i);
            if (!child || !child.getDomElem()) continue;
            child.checkDisplay(event);
        }
        if (this.setBuildMode) this.setBuildMode(false);
    }

    /**
     * Whether the element is currently displayed
     */
    isDisplay() {
        if (!this.visibility()) return false;

        let r = this.getGlobalRect(), w, h,
            temp = this.parent,
            box = null;
        while (temp) {
            if (temp.overflow() == 'auto' || temp.overflow() == 'hidden') {
                box = temp;
                break;
            }
            temp = temp.parent;
        }

        if (box) {
            let rect = box.getGlobalRect();
            if (r.top > rect.top + rect.height) return false;
            if (r.top + r.height < rect.top) return false;
            if (r.left > rect.left + rect.width) return false;
            if (r.left + r.width < rect.left) return false;
            return box.isDisplay();
        } else {
            w = document.documentElement.scrollWidth;
            h = document.documentElement.scrollHeight;
            if (r.top > h) return false;
            if (r.bottom > h) return false;
            if (r.left > w) return false;
            if (r.right > w) return false;
        }

        return true;
    }

    /**
     * Examples:
     * 1. small.locateBy(big, lx.RIGHT);
     * 2. small.locateBy(big, lx.RIGHT, 5);
     * 3. small.locateBy(big, [lx.RIGHT, lx.BOTTOM]);
     * 4. small.locateBy(big, {bottom: '20%', right: 20});
     */
    locateBy(elem, align, step) {
        if (lx.isArray(align)) {
            for (let i=0,l=align.len; i<l; i++) this.locateBy(elem, align[i]);
            return this;
        } else if (lx.isObject(align)) {
            for (let i in align) this.locateBy(elem, lx.Geom.alignConst(i), align[i]);
            return this;
        }

        if (!step) step = 0;
        step = elem.geomPart(step, 'px', lx.Geom.directionByGeom(align));
        let rect = elem.getGlobalRect();
        switch (align) {
            case lx.TOP:
                __setGeomPriorityV(this, lx.TOP, lx.MIDDLE);
                this.top( rect.top+step+'px' );
                break;
            case lx.BOTTOM:
                __setGeomPriorityV(this, lx.BOTTOM, lx.MIDDLE);
                this.bottom( rect.bottom+step+'px' );
                break;
            case lx.MIDDLE:
                __setGeomPriorityV(this, lx.TOP, lx.MIDDLE);
                this.top( rect.top + (elem.height('px') - this.height('px')) * 0.5 + step + 'px' );
                break;

            case lx.LEFT:
                __setGeomPriorityH(this, lx.LEFT, lx.CENTER);
                this.left( rect.left+step+'px' );
                break;
            case lx.RIGHT:
                __setGeomPriorityH(this, lx.RIGHT, lx.CENTER);
                this.right( rect.right+step+'px' );
                break;
            case lx.CENTER:
                __setGeomPriorityH(this, lx.LEFT, lx.CENTER);
                this.left( rect.left + (elem.width('px') - this.width('px')) * 0.5 + step + 'px' );
                break;
        };
        return this;
    }

    satelliteTo(elem) {
        this.locateBy(elem, {top: elem.height('px'), center:0});
        if (this.isOutOfParentScreen().bottom)
            this.locateBy(elem, {bottom: elem.height('px'), center:0});
        this.returnToParentScreen();
    }
    /* 4. Geometry */
    //==================================================================================================================


    //==================================================================================================================
    /* 5. Environment managment */
    /**
     * config == Rect | {
     *     parent: Box    // directly parent, if null - parent is not set
     *     index: int     // if there is a group in the parent by the element key, you can set a specific position
     *     before: Rect   // if there is a group in the parent by the element key, it can be positioned before the specified element from the group
     *     after: Rect    // if there is a group in the parent by the element key, it can be positioned after the specified element from the group
     * }
     */
    setParent(config = null) {
        this.dropParent();

        if (config === null) return null;

        let parent = null,
            next = null;
        if (config instanceof lx.Rect) {
            parent = config;
            config = {};
        } else {
            if (config.parent === null) return null;

            if (config.before && config.before.parent) {
                parent = config.before.parent;
                next = config.before;
            } else if (config.after && config.after.parent) {
                parent = config.after.parent;
                next = config.after.nextSibling();
            } else {
                parent = config.parent || lx.app.getAutoParent();
            }
        }
        if (!parent) return null;

        config.nextSibling = next;
        parent.addChild(this, config);
        return parent;
    }

    dropParent() {
        if (this.parent) this.parent.remove(this);
        this.parent = null;
        return this;
    }

    after(el) {
        return this.setParent({ after: el });
    }

    before(el) {
        return this.setParent({ before: el });
    }

    del() {
        //TODO - check the importance of this line. After wrapping a DOM element, it may not exist, but the wrapper does
        // if (!this.getDomElem()) return;
        let p = this.parent;
        if (p) return p.del(this);
        // If there is no parent, it is a root element, we do not delete it
        return 0;
    }

    // @lx:<context CLIENT:
    /**
     * Assign an element a field that it can track
     */
    setField(name, func, type = null) {
        this._field = name;
        this._bindType = type || lx.app.binder.BIND_TYPE_FULL;

        if (func) {
            let valFunc = this.lxHasMethod('value')
                ? this.value
                : function(val) { if (val===undefined) return this._val; this._val = val; };
            this.innerValue = valFunc;

            // Determine whether the passed function can return a value (other than set)
            let str = func.toString(),
                argName = (str[0] == '(')
                    ? str.match(/^\((.*?)(?:,|\))/)[1]
                    : str.match(/(?:^([\w\d_]+?)=>|^function\s*[^\(]*\((.*?)(?:,|\)))/)[1],
                reg = new RegExp('if\\s*\\(\\s*' + argName + '\\s*===\\s*undefined'),
                isCallable = (str.match(reg) !== null);

            /* The method through which communication with the model is carried out:
             * - the model reads the value from here when an event 'change' is triggered on the widget
             * - the model writes the field value here when it changes via the setter
             */
            this.value = function(val) {
                if (val === undefined) {
                    if (isCallable) return func.call(this);
                    return this.innerValue();
                }
                let oldVal = isCallable ? func.call(this) : this.innerValue();
                /* Algorithm:
                 * 1. In user code, the .value(val) method was called, and some value was passed
                 * 2. The value is new - it is assigned to the widget value, the 'change' event is triggered
                 * 3. On 'change' event the model is updated - a new value is written into it via the setter
                 * 4. The model setter also contains the actualization logic - this time for the widget, via the value(val) method
                 * 5. This method will be called repeatedly, so to avoid recursion - when trying to assign a value identical to the current one,
                 * we do nothing - if the value of x "changed" to the value of x, no changes occurred - the 'change' event should not be triggered
                 */
                if (lx.Comparator.deepCompare(val, oldVal)) return;
                this.innerValue(val);
                func.call(this, val, oldVal);
                this.trigger('change', this.newEvent({
                    oldValue: oldVal,
                    newValue: val
                }));
            };
        }

        return this;
    }
    // @lx:context>
    /* 5. Environment managment */
    //==================================================================================================================


    //==================================================================================================================
    /* 6. Environment navigation */
    nextSibling() {
        return this.domElem.nextSibling();
    }

    prevSibling() {
        return this.domElem.prevSibling();
    }

    /**
     * Find the first closest ancestor that satisfies the condition from the passed configuration:
     * 1. is - exact match to the passed constructor or object
     * 2. hasProperty|hasProperties - has a property(s), when values ​​are transferred, their compliance is checked
     * 3. checkMethods - has method(s), their return values ​​are checked
     * 4. instance - instance match (difference from 1 - can be an instance inheritance)
     */
    ancestor(info={}) {
        if (info.hasProperty) info.hasProperties = [info.hasProperty];
        let p = this.parent;
        while (p) {
            if (lx.isFunction(info)) {
                if (info(p)) return p;
            } else {
                if (info.is) {
                    let instances = lx.isArray(info.is) ? info.is : [info.is];
                    for (let i=0, l=instances.len; i<l; i++) {
                        if (p.constructor === instances[i] || p === instances[i])
                            return p;
                    }
                }

                if (info.hasProperties) {
                    let prop = info.hasProperties;
                    if (lx.isObject(prop)) {
                        let match = true;
                        for (let name in prop)
                            if (!(name in p) || prop[name] != p[name]) {
                                match = false;
                                break;
                            }
                        if (match) return p;
                    } else if (lx.isArray(prop)) {
                        let match = true;
                        for (let j=0, l=prop.len; j<l; j++)
                            if (!(prop[j] in p)) {
                                match = false;
                                break;
                            }
                        if (match) return p;
                    }
                }

                if (info.checkMethods) {
                    let match = true;
                    for (let name in info.checkMethods) {
                        if (!(name in p) || !lx.isFunction(p[name]) || p[name]() != info.checkMethods[name]) {
                            match = false;
                            break;
                        }
                    }
                    if (match) return p;
                }

                if (info.instance && p instanceof info.instance) return p;
            }

            p = p.parent;
        }
        return null;
    }

    hasAncestor(box) {
        let temp = this.parent;
        while (temp) {
            if (temp === box) return true;
            temp = temp.parent;
        }
        return false;
    }

    /**
     * The search ascends the parent hierarchy, returning the first closest element found
     */
    neighbor(key) {
        let parent = this.parent;
        while (parent) {
            let el = parent.get(key);
            if (el) return el;
            parent = parent.parent;
        }
        return null;
    }
    /* 6. Environment navigation */
    //==================================================================================================================


    //==================================================================================================================
    /* 7. Events */
    on(eventName, func) {
        if (!func) return this;

        //TODO
        if (eventName == 'mousedown') this.on('touchstart', func);
        else if (eventName == 'mousemove') this.on('touchmove', func);
        else if (eventName == 'mouseup') this.on('touchend', func);
        else if (eventName == 'click') this.on('touchstart', func);

        if (lx.isString(func))
            func = this.unpackFunction(func);

        if (func) this.domElem.addEvent(eventName, func);
        return this;
    }

    off(eventName, func) {
        this.domElem.delEvent(eventName, func);
        return this;
    }

    hasTrigger(type, func) {
        return this.domElem.hasEvent(type, func);
    }

    move(config={}) {
        // @lx:<context CLIENT:
        this.off('mousedown', lx.app.dragNDrop.move);
        // @lx:context>
        this.moveParams = {};

        if (config === false) {
            return;
        }

        if (config.parentMove && config.parentResize) delete config.parentMove;
        this.moveParams = {
            xMove        : lx.getFirstDefined(this.moveParams.xMove, config.xMove, true),
            yMove        : lx.getFirstDefined(this.moveParams.yMove, config.yMove, true),
            parentMove   : lx.getFirstDefined(this.moveParams.parentMove, config.parentMove, false),
            parentResize : lx.getFirstDefined(this.moveParams.parentResize, config.parentResize, false),
            xLimit       : lx.getFirstDefined(this.moveParams.xLimit, config.xLimit, true),
            yLimit       : lx.getFirstDefined(this.moveParams.yLimit, config.yLimit, true),
            moveStep     : lx.getFirstDefined(this.moveParams.moveStep, config.moveStep, 1),
            locked       : false
        };
        // @lx:<context CLIENT:
        if (!this.hasTrigger('mousedown', lx.app.dragNDrop.move))
            this.on('mousedown', lx.app.dragNDrop.move);
        // @lx:context>
        // @lx:<context SERVER:
        this.onLoad('()=>this.on(\'mousedown\', lx.app.dragNDrop.move);');
        // @lx:context>
        return this;
    }

    lockMove() {
        if (!this.moveParams) return;
        this.moveParams.locked = true;
    }

    unlockMove() {
        if (!this.moveParams) return;
        this.moveParams.locked = false;
    }

    click(func) { this.on('click', func); return this; }

    display(func) { this.on('display', func); return this; }

    displayIn(func) { this.on('displayin', func); return this; }

    displayOut(func) { this.on('displayout', func); return this; }

    displayOnce(func) {
        // @lx:<context SERVER:
        this.onLoad('.displayOnce', func);
        // @lx:context>
        // @lx:<context CLIENT:
        if (lx.isString(func)) func = this.unpackFunction(func);
        if (!func) return this;
        let f;
        f = function() {
            func.call(this);
            this.off('displayin', f);
        };
        this.on('displayin', f);
        // @lx:context>
        return this;
    }

    mouseover(handler) {
        this.on('mouseover', e=>{
            if (
                e.target && e.relatedTarget
                && e.target.__lx && e.relatedTarget.__lx
                && (e.target.__lx === this || e.target.__lx.hasAncestor(this))
                && (e.relatedTarget.__lx === this || e.relatedTarget.__lx.hasAncestor(this))
            ) return;
            handler(e);
        });
    }

    mouseout(handler) {
        this.on('mouseout', e=>{
            if (
                e.target && e.relatedTarget
                && e.target.__lx && e.relatedTarget.__lx
                && (e.target.__lx === this || e.target.__lx.hasAncestor(this))
                && (e.relatedTarget.__lx === this || e.relatedTarget.__lx.hasAncestor(this))
            ) return;
            handler(e);
        });
    }

    getEventHandlers(name) {
        let eventList = this.domElem.getEvents();
        if (!eventList) return [];

        console.log(name, eventList);

        if (name in eventList) {
            let result = [];
            for (let i in eventList[name])
                result.push(eventList[name][i]);
            return result;
        }

        return [];
    }

    copyEvents(el) {
        if (!el) return this;

        this.off();

        let eventList = el.domElem.getEvents();
        if (!eventList) return this;

        for (let eventName in eventList) {
            let events = eventList[eventName];
            for (let i in events) {
                this.on(eventName, events[i]);
            }
        }

        return this;
    }

    trigger(eventName, event) {
        // @lx:<context CLIENT:
        function runEventHandlers(context, handlersList, event) {
            event = event || context.newEvent();
            let res = [];
            for (let i in handlersList) {
                let func = handlersList[i];
                res.push(func.call(context, event));
            }
            if (lx.isArray(res) && res.length == 1) res = res[0];
            return res;
        }

        if (lx(STATIC).getCommonEventNames().includes(eventName)) {
            let events = this.domElem.elem ? this.domElem.elem.events : this.domElem.events;
            if (!events || !(eventName in events)) return;
            return runEventHandlers(this, events[eventName], event);
        }

        let elem = this.getDomElem();
        if (!elem) return;
        let disabled = (lx(STATIC).getEnabledEventNames().includes(eventName))
            ? false
            : this.disabled();
        if (disabled || !elem.events || !(eventName in elem.events)) return;

        return runEventHandlers(this, elem.events[eventName], event);
        // @lx:context>
    }

    static getEnabledEventNames() {
        return ['display', 'displayin', 'displayout'];
    }

    static getCommonEventNames() {
        return [];
    }

    // @lx:<context SERVER:
    /**
     * Delegating method execution to the client side
     */
    onLoad(handler, args = null) {
        if (!this.forOnload) this.forOnload = [];
        this.forOnload.push(args ? [handler, args] : handler);
        return this;
    }

    onPostUnpack(handler, flag = lx.POSTUNPACK_TYPE_FIRST_DISPLAY) {
        switch (flag) {
            case lx.POSTUNPACK_TYPE_IMMEDIATLY:
                this.onLoad(handler);
                break;
            case lx.POSTUNPACK_TYPE_FIRST_DISPLAY:
                this.displayOnce(handler);
                break;
        }
    }
    // @lx:context>
    /* 7. Events */
    //==================================================================================================================


    //==================================================================================================================
    /* 8. Load */
    // @lx:<context SERVER:
    /**
     * We pack the function code into a string, converting it to the format:
     * '(arg1, arg2) => ...function code'
     * The reverse method to the client [[unpackFunction(str)]]
     */
    packFunction(func) {
        return lx.app.functionHelper.functionToString(func);
    }
    // @lx:context>

    // @lx:<context CLIENT:
    /**
     * If event handlers redirect to real functions when unpacking
     * Argument options:
     * - ('.funcName')     // will look for function 'funcName' among its methods
     * - ('::funcName')    // will search for function 'funcName' among static methods of its class
     * - ('lx.funcName')   // will look for function 'funcName' in lx function list
     */
    findFunction(handler) {
        // '.funcName'
        if (handler.match(/^\./)) {
            let func = this[handler.split('.')[1]];
            if (!func || !lx.isFunction(func)) return null;
            return func;
        }

        // '::funcName'
        if (handler.match(/^::/)) {
            let func = lx[this.lxClassName()][handler.split('::')[1]];
            if (!func || !lx.isFunction(func)) return null;
            return func;
        }

        // If there is no explicit prefix, we will try to find a handler from the specific to the general
        let f = null;
        f = this.findFunction('.' + handler);
        if (f) return f;
        f = this.findFunction('::' + handler);
        if (f) return f;
        f = this.findFunction('lx.' + handler);
        if (f) return f;

        return null;
    }

    /**
     * The format of a function that is packed into a string on the server side:
     * '(arg1, arg2) => ...function code'
     * The reverse method of the server [[packFunction(func)]]
     */
    unpackFunction(str) {
        let f = (str.match(/^\(.*?\)\s*=>/))
            ? lx.app.functionHelper.stringToFunction(str)
            : this.findFunction(str);
        if (!f) return null;
        return f;
    }

    /**
     * Unpacking geometric parameter priorities
     */
    unpackGeom() {
        let val = this.geom,
            arr = val.split('|');
        this.geom = {};
        if (arr[0] != '') {
            let params = arr[0].split(',');
            __setGeomPriorityH(this, +params[0], +params[1]);
        }
        if (arr[1] != '') {
            let params = arr[1].split(',');
            __setGeomPriorityV(this, +params[0], +params[1]);
        }
    }

    /**
     * Unpacking Extended Properties
     */
    unpackProperties() {
        if (!this.inLoad) return;

        if (this.geom && lx.isString(this.geom)) this.unpackGeom();

        // Positioning strategies
        if (this.__ps) {
            let ps = this.lxExtract('__ps').split(';'),
                psName = ps.shift(),
                con = lx.getClassConstructor(psName);

            this.positioningStrategy = new con(this);
            this.positioningStrategy.unpack(ps);
        }

        // Event handlers
        if (this.handlers) {
            for (let i in this.handlers) for (let j in this.handlers[i]) {
                let f = this.unpackFunction(this.handlers[i][j]);
                if (f) this.on(i, f);
            }

            delete this.handlers;
        }
    }

    /**
     * Method for restoring widget when receiving data from server
     * @abstract
     */
    postUnpack(config={}) {
        // pass
    }

    /**
     * Method called by the loader to restore the element
     */
    postLoad() {
        let config = this.lxExtract('__postBuild') || {};
        this.postUnpack(config);
        this.clientRender(config);
    }

    /** @abstract */
    restoreLinks(loader) {
        // pass
    }
    // @lx:context>
    /* 8. Load */
    //==================================================================================================================
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * PRIVATE
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

/**
 * format = '%' | 'px'  - will return a float value according to the passed format
 * if format is not specified - returns the value as it is written in style
 */
function __getLeft(self, format) {
    if (!format) return self.domElem.style('left');

    let elem = self.getDomElem();
    if (!elem) return null;

    let pw = (self.domElem.parent) ? self.domElem.parent.getDomElem().offsetWidth : elem.offsetWidth;
    return __calcGeomParam(format, elem.style.left, elem.offsetLeft, pw);
}

function __getRight(self, format) {
    if (!format) return self.domElem.style('right');

    let elem = self.getDomElem();
    if (!elem) return null;

    if (!self.domElem.parent) return undefined;
    let pElem = self.domElem.parent.getDomElem();
    if (!pElem) return undefined;
    if (elem.style.right != '') {
        let b = lx.Geom.splitGeomValue(elem.style.right);
        if (format == '%') {
            if ( b[1] != '%' ) b[0] = (b[0] / pElem.offsetWidth) * 100;
            return b[0];
        } else {
            if ( b[1] != 'px' ) b[0] = b[0] * pElem.offsetWidth * 0.01;
            return b[0];
        }
    } else {
        let t = lx.Geom.splitGeomValue(elem.style.left),
            h = lx.Geom.splitGeomValue(elem.style.width),
            pw = pElem.offsetWidth;
        if (format == '%') {
            if ( t[1] != '%' ) t[0] = (t[0] / pw) * 100;
            if ( h[1] != '%' ) h[0] = (h[0] / pw) * 100;
            return 100 - t[0] - h[0];
        } else {
            if ( t[1] != 'px' ) t[0] = elem.offsetLeft;
            if ( h[1] != 'px' ) h[0] = elem.offsetWidth;
            return pw - t[0] - h[0];
        }
    }
}

function __getTop(self, format) {
    if (!format) return self.domElem.style('top');

    let elem = self.getDomElem();
    if (!elem) return null;

    let p = self.domElem.parent,
        ph = (p) ? p.getDomElem().offsetHeight : elem.offsetHeight;
    return __calcGeomParam(format, elem.style.top, elem.offsetTop, ph);
}

function __getBottom(self, format) {
    if (!format) return self.domElem.style('bottom');

    let elem = self.getDomElem();
    if (!elem) return null;

    if (!self.domElem.parent) return undefined;
    pElem = self.domElem.parent.getDomElem();
    if (!pElem) return undefined;
    if (elem.style.bottom != '') {
        let b = lx.Geom.splitGeomValue(elem.style.bottom);
        if (format == '%') {
            if ( b[1] != '%' ) b[0] = (b[0] / pElem.clientHeight) * 100;
            return b[0];
        } else {
            if ( b[1] != 'px' ) b[0] = b[0] * pElem.clientHeight * 0.01;
            return b[0];
        }
    } else {
        let t = lx.Geom.splitGeomValue(elem.style.top),
            h = lx.Geom.splitGeomValue(elem.style.height),
            ph = pElem.clientHeight;
        if (format == '%') {
            if ( t[1] != '%' ) t[0] = (t[0] / ph) * 100;
            if ( h[1] != '%' ) h[0] = (h[0] / ph) * 100;
            return 100 - t[0] - h[0];
        } else {
            if ( t[1] != 'px' ) t[0] = elem.offsetTop;
            if ( h[1] != 'px' ) h[0] = elem.offsetHeight;
            return ph - t[0] - h[0];
        }
    }
}

function __getWidth(self, format) {
    if (!format) return self.domElem.style('width');

    let elem = self.getDomElem();
    if (!elem) return null;

    if (!self.domElem.parent || !self.domElem.parent.getDomElem()) {
        if (format == '%') return 100;
        return elem.offsetWidth;
    }

    return __calcGeomParam(format, elem.style.width,
        elem.offsetWidth, self.domElem.parent.getDomElem().offsetWidth);
}

function __getHeight(self, format) {
    if (!format) return self.domElem.style('height');

    let elem = self.getDomElem();
    if (!elem) return null;

    if (!self.domElem.parent || !self.domElem.parent.getDomElem()) {
        if (format == '%') return 100;
        return elem.offsetHeight;
    }
    return __calcGeomParam(format, elem.style.height,
        elem.offsetHeight, self.domElem.parent.getDomElem().offsetHeight);
}

function __getGlobalGeomMask(self) {
    let result = {};
    result.pH = __getGeomPriorityH(self).lxClone();
    result.pV = __getGeomPriorityV(self).lxClone();
    let rect = self.getGlobalRect();
    result[result.pH[0]] = rect[lx.Geom.geomName(result.pH[0])];
    result[result.pH[1]] = rect[lx.Geom.geomName(result.pH[1])];
    result[result.pV[0]] = rect[lx.Geom.geomName(result.pV[0])];
    result[result.pV[1]] = rect[lx.Geom.geomName(result.pV[1])];
    if (self.geom) result.geom = self.geom.lxClone();
    return result;
}

/**
 * Calculation to return the value in the required format
 * val - how the value is written in the style
 * thisSize - the size of the element itself in pixels
 * parentSize - parent element size in pixels
 */
function __calcGeomParam(format, val, thisSize, parentSize) {
    if (format == 'px') return thisSize;

    if (val == null) return null;

    if ( val.charAt( val.length - 1 ) == '%' ) {
        if (format == '%') return parseFloat(val);
        return thisSize;
    }

    return ( thisSize * 100 ) / parentSize;
}

function __getGeomPriorityH(self) {
    return ((self.geom) ? self.geom.bpg : 0) || [lx.LEFT, lx.CENTER];
}

function __getGeomPriorityV(self) {
    return ((self.geom) ? self.geom.bpv : 0) || [lx.TOP, lx.MIDDLE];
}

function __setGeomPriorityH(self, val, val2) {
    if (val2 !== undefined) {
        if (!self.geom) self.geom = {};
        let dropGeom = __getGeomPriorityH(self).lxDiff([val, val2])[0];
        self.geom.bpg = [val, val2];
        if (dropGeom === undefined) return self;
        self.domElem.style(lx.Geom.geomName(dropGeom), '');
        return self;
    }

    if (!self.geom) self.geom = {};

    if (!self.geom.bpg) self.geom.bpg = (val==lx.RIGHT)
        ? [lx.RIGHT, lx.CENTER]
        : [lx.LEFT, lx.CENTER];

    if (self.geom.bpg[0] == val) return self;

    if (self.geom.bpg[1] != val) switch (self.geom.bpg[1]) {
        case lx.LEFT: self.domElem.style('left', ''); break;
        case lx.CENTER: self.domElem.style('width', ''); break;
        case lx.RIGHT: self.domElem.style('right', ''); break;
    }

    self.geom.bpg[1] = val;
    return self;
}

function __setGeomPriorityV(self, val, val2) {
    if (val2 !== undefined) {
        if (!self.geom) self.geom = {};
        let dropGeom = __getGeomPriorityV(self).lxDiff([val, val2])[0];
        self.geom.bpv = [val, val2];
        if (dropGeom === undefined) return self;
        self.domElem.style(lx.Geom.geomName(dropGeom), '');
        return self;
    }

    if (!self.geom) self.geom = {};
    if (!self.geom.bpv) self.geom.bpv = (val==lx.BOTTOM)
        ? [lx.BOTTOM, lx.MIDDLE]
        : [lx.TOP, lx.MIDDLE];

    if (self.geom.bpv[0] == val) return self;

    if (self.geom.bpv[1] != val) switch (self.geom.bpv[1]) {
        case lx.TOP: self.domElem.style('top', ''); break;
        case lx.MIDDLE: self.domElem.style('height', ''); break;
        case lx.BOTTOM: self.domElem.style('bottom', ''); break;
    }

    self.geom.bpv[1] = val;
    return self;
}

function __triggerDisplay(self, event) {
    if (!self.isDisplay()) {
        if (self.displayNow) {
            self.trigger('displayout', event);
            self.displayNow = false;
        }
    } else {
        if (!self.displayNow) {
            self.displayNow = true;
            self.trigger('displayin', event);
        } else self.trigger('display', event);
    }
}
