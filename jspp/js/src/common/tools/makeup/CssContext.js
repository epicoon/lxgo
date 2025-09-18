// @lx:namespace lx;
class CssContext {
    constructor() {
        this.reset();
    }

    reset() {
        this.sequens = [];
        this.styles = {};
        this.abstractClasses = {};
        this.classes = {};
        this.mixins = {};
        this.prefix = '';
        this.preset = null;
        this.proxyContexts = [];
        this.presetClasses = [];
        this.presetStyles = [];
    }

    configure(config) {
        this.reset();
        if (config.preset)
            this.usePreset(config.preset);
        if (config.holders)
            for (let i in config.holders)
                this.useHolder(config.holders[i]);
    }

    /**
     * @param {string} prefix
     */
    setPrefix(prefix) {
        this.prefix = prefix;
    }

    /**
     * @param {lx.CssPreset} preset 
     */
    usePreset(preset) {
        this.preset = preset;
    }

    /**
     * @param {class<lx.CssContextHolder>} holder 
     */
    useHolder(holder) {
        this.proxyContexts.lxPushUnique(holder.getContext());
    }

    /**
     * @param {lx.CssContext} context
     */
    merge(context) {
        this.abstractClasses.lxMerge(context.abstractClasses, true);
        this.classes.lxMerge(context.classes, true);
        this.mixins.lxMerge(context.mixins, true);
        this.presetClasses.lxMerge(context.presetClasses);
        this.presetStyles.lxMerge(context.presetStyles);
        this.proxyContexts.lxMerge(context.proxyContexts);
        this.sequens.lxMerge(context.sequens);
        this.styles.lxMerge(context.styles, true);
    }

    /**
     * @param {string|Array<string>} valName
     * @param {any|Array<any>} defaultVal
     * @param {Function} [modifier]
     * @returns {lx.PresetFieldtHolder}
     */
    presetValue(valName, defaultVal, modifier = null) {
        return new lx.PresetFieldtHolder(valName, defaultVal, modifier);
    }

    addStyle(name, content = {}) {
        if (lx.isArray(name)) name = name.join(',');

        this.sequens.push({name, type: 'styles'});

        let constructor = (name[0] == '@') ? CssDirective : CssStyle;
        this.styles[name] = new constructor({
            context: this,
            name,
            content
        });
    }

    hasClass(name) {
        if ('.' + name in this.classes) return true;
        for (let i in this.proxyContexts)
            if (this.proxyContexts[i].hasClass(name))
                return true;
        return false;
    }

    /**
     * @returns {Array<String>}
     */
    getClassNames() {
        let map = {};
        let re = (ctx) => {
            for (let i in ctx.proxyContexts) re(ctx.proxyContexts[i]);
            for (let n in ctx.classes) map[n] = 1;
        };
        re(this);
        return Object.keys(map);
    }

    addClass(name, content = {}, pseudoclasses = {}) {
        if (name[0] != '.') name = '.' + name;

        this.sequens.push({name, type: 'classes'});

        let processed = _processContent(this, content, pseudoclasses);
        this.classes[name] = new CssClass({
            context: this,
            name,
            content: processed.content,
            pseudoclasses: processed.pseudoclasses
        });
    }

    inheritClass(name, parent, content = {}, pseudoclasses = {}) {
        if (name[0] != '.') name = '.' + name;
        if (parent[0] != '.') parent = '.' + parent;

        this.sequens.push({name, type: 'classes'});

        let processed = _processContent(this, content, pseudoclasses);
        this.classes[name] = new CssClass({
            context: this,
            parent,
            name,
            content: processed.content,
            pseudoclasses: processed.pseudoclasses
        });
    }

    addAbstractClass(name, content = {}, pseudoclasses = {}) {
        if (name[0] != '.') name = '.' + name;

        let processed = _processContent(this, content, pseudoclasses);
        this.abstractClasses[name] = new CssClass({
            context: this,
            isAbstract: true,
            name,
            content: processed.content,
            pseudoclasses: processed.pseudoclasses
        });
    }

    inheritAbstractClass(name, parent, content = {}, pseudoclasses = {}) {
        if (name[0] != '.') name = '.' + name;
        if (parent[0] != '.') parent = '.' + parent;

        let processed = _processContent(this, content, pseudoclasses);
        this.abstractClasses[name] = new CssClass({
            context: this,
            isAbstract: true,
            parent,
            name,
            content: processed.content,
            pseudoclasses: processed.pseudoclasses
        });
    }

    addStyleGroup(name, list) {
        for (let nameI in list) {
            let content = list[nameI];
            if (content.lxParent) {
                content = list[content.lxParent]
                    ? list[content.lxParent].lxClone().lxMerge(content, true)
                    : content;
                delete content.lxParent;
            }

            this.addStyle(name + ' ' + nameI, content);
        }
    }

    addClasses(list) {
        for (let name in list) {
            let content = list[name];
            if (lx.isArray(content)) this.addClass(name, content[0], content[1]);
            else this.addClass(name, content);
        }
    }

    inheritClasses(list, parent) {
        for (let name in list) {
            let content = list[name];
            if (lx.isArray(content)) this.inheritClass(name, parent, content[0], content[1]);
            else this.inheritClass(name, parent, content);
        }
    }

    registerMixin(name, callback) {
        this.mixins[name] = callback;
    }
    
    getClass(name) {
        if (name[0] != '.') name = '.' + name;

        if (name in this.abstractClasses)
            return this.abstractClasses[name];
        if (name in this.classes)
            return this.classes[name];
        return null;
    }

    toString() {
        let result = '';
        for (let i=0, l=this.sequens.length; i<l; i++) {
            const rule = this[this.sequens[i].type][this.sequens[i].name];
            result += rule.render();
        }

        if (this.prefix != '') {
            for (let i=0, l=this.presetStyles.length; i<l; i++) {
                let name = this.presetStyles[i];
                let reg = new RegExp('([ :])' + name + '([ ;{}])', 'g');
                result = result.replace(reg, '$1' + this.prefix + '-' + name + '$2');
            }
        }

        return result;
    }
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * CssRule
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

class CssRule {

}

function _defineRulePreset(rule) {
    if (_defineAttrsPreset(rule.content))
        return true;

    if (rule.pseudoclasses) {
        for (let i in rule.pseudoclasses)
            if (_defineAttrsPreset(rule.pseudoclasses[i]))
                return true;
    }

    return false;
}

function _defineAttrsPreset(attrs) {
    for (let i in attrs) {
        let attr = attrs[i];
        if (lx.isInstance(attr, lx.CssValue) || lx.isInstance(attr, lx.PresetFieldtHolder)) return true;
        if (lx.isStrictObject(attr) && _defineAttrsPreset(attr))
            return true;
    }
    return false;
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * CssClass
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

class CssClass {
    constructor(config) {
        this.context = config.context;
        this.basicName = config.name;
        this.isAbstract = lx.getFirstDefined(config.isAbstract, false);
        this.parent = lx.getFirstDefined(config.parent, null);

        this.selfContent = config.content;
        this.selfPseudoclasses = config.pseudoclasses;
        this.content = null;
        this.pseudoclasses = null;
        this._isPreset = false;
        this.refresh();
    }

    isPreset() {
        return this._isPreset;
    }

    refresh() {
        this.content = lx.clone(this.selfContent);
        this.pseudoclasses = lx.clone(this.selfPseudoclasses);
        if (this.parent) _applyClassParent(this);
        this._isPreset = _defineRulePreset(this);
    }

    render() {
        const className = (this.context.prefix)
            ? '.' + this.context.prefix + '-' + this.basicName.replace(/^\./, '')
            : this.basicName;

        //TODO deprecated
        // const className = (this.isPreset() && this.context.preset)
        //     ? this.basicName + '-' + this.context.preset.name
        //     : this.basicName;

        let text = className + '{' + _getContentString(this.context, this.content) + '}';

        for (let pseudoclassName in this.pseudoclasses) {
            let pseudoclass = this.pseudoclasses[pseudoclassName];
            pseudoclassName = (pseudoclassName == 'disabled')
                ? className + '[' + pseudoclassName + ']'
                : className + ':' + pseudoclassName;

            text += pseudoclassName + '{' + _getContentString(this.context, pseudoclass) + '}';
        }

        if (this.isPreset()) {
            let className = this.basicName.substr(1);
            if (!this.context.presetClasses.includes(className))
                this.context.presetClasses.push(className);
        }
        return text;
    }
}

function _applyClassParent(self) {
    self.content = _getClassPropertyWithParent(self, 'content');
    self.pseudoclasses = _getClassPropertyWithParent(self, 'pseudoclasses');
}

function _getClassPropertyWithParent(cssClass, property) {
    if (!cssClass.parent) return cssClass[property];

    let parentClass = null;
    if (lx.isObject(cssClass.parent))
        parentClass = cssClass.parent;
    if (!parentClass) parentClass = _getCssClass(cssClass.context, cssClass.parent);
    if (!parentClass) return cssClass[property];

    let pProperty = _getClassPropertyWithParent(parentClass, property) || {};
    if (lx.isString(pProperty)) pProperty = {__str__:[pProperty]};

    let result = pProperty.lxClone();
    if (!result.__str__) result.__str__ = [];

    if (lx.isObject(cssClass[property]))
        result = cssClass[property].lxMerge(result);
    else if (lx.isString(cssClass[property]))
        result.__str__.push(cssClass[property]);

    if (!result.__str__.len) delete result.__str__;
    if (result.lxEmpty()) return null;
    return result;
}

function _getCssClass(context, name) {
    if (name in context.abstractClasses)
        return context.abstractClasses[name];

    if (name in context.classes)
        return context.classes[name];

    for (let i in context.proxyContexts) {
        let c = _getCssClass(context.proxyContexts[i], name);
        if (c) return c;
    }

    return null;
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * CssStyle
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

class CssStyle {
    constructor(config) {
        this.context = config.context;
        this.selector = config.name;
        this.content = config.content;
    }

    render() {
        let selector = this.selector,
            list = [...selector.matchAll(/\.\b[\w\d_-]+\b/g)];
        for (let i in list) {
            let cssClassName = list[i][0],
                cssClass = this.context.getClass(cssClassName);
            if (!cssClass) continue;

            if (this.context.prefix) {
                let reg = new RegExp(cssClassName + '($|[^\w\d_-])');
                selector = selector.replace(reg, '.' + this.context.prefix + '-' + cssClassName.replace(/^\./, '') + '$1');
            }

            //TODO deprecated
            // if (cssClass.isPreset()) {
            //     let reg = new RegExp(cssClassName + '($|[^\w\d_-])');
            //     selector = selector.replace(reg, cssClassName + '-' + cssClass.context.preset.name + '$1');
            // }
        }

        return selector + '{' + _getContentString(this.context, this.content) + '}';
    }
}


/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * CssDirective
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

class CssDirective extends CssStyle {
    render() {
        if (!/^@keyframes/.test(this.selector)) return super.render();

        if (_defineRulePreset(this))
            this.context.presetStyles.lxPushUnique(this.selector.replace(/^@keyframes\s+/, ''));

        let content = [];
        for (let key in this.content) {
            let attrs = this.content[key];
            let row = _getContentString(this.context, attrs);
            content.push(key + '{' + row + '}');
        }
        return this.selector + '{' + content.join('') + '}';
    }
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * PRIVATE
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

function _processContent(self, content, pseudoclasses) {
    if (lx.isString(content)) {
        return {content, pseudoclasses};
    }

    let processedContent = {};
    for (let name in content) {
        if (name[0] != '@') {
            processedContent[name] = content[name];
            continue;
        }

        let mixin = _getMixin(self, name);
        if (!mixin) continue;

        let args = content[name];
        if (!lx.isArray(args)) args = [args];
        let result = mixin.apply(null, args);

        if (result.content) {
            processedContent.lxMerge(result.content);
            if (result.pseudoclasses) pseudoclasses.lxMerge(result.pseudoclasses);
        } else {
            processedContent.lxMerge(result);
        }
    }

    return {
        content: processedContent,
        pseudoclasses
    };
}

function _getMixin(self, name) {
    let mixinName = (name[0] == '@') ? name.replace(/^@/, '') : name;
    if (mixinName in self.mixins)
        return self.mixins[mixinName];

    for (let i in self.proxyContexts) {
        let mixin = _getMixin(self.proxyContexts[i], name);
        if (mixin) return mixin;
    }

    return null;
}

function _getContentString(context, content) {
    let str = _prepareContentString(context, content);
    return _postProcessContentString(str);
}

function _postProcessContentString(str) {
    let result = str;
    result = result.replace(/(,|:) /g, '$1');
    result = result.replace(/ !important/g, '!important');
    result = result.replace(/([^\d])0(px|%)/g, '$10');

    result = result.replace(/color:white/g, 'color:#fff');
    result = result.replace(/color:black/g, 'color:#000');

    return result;
}

function _prepareContentString(context, content) {
    if (!content) return '';

    if (lx.isString(content)) return content;

    if (lx.isObject(content)) {
        let arr = [];
        for (let prop in content) {
            if (prop == '__str__') {
                if (content.__str__.len) arr.push(content.__str__.join(';'));
                continue;
            }

            let propName = prop.replace(/([A-Z])/g, function(x){return "-" + x.toLowerCase()});

            let propVal = null;
            if (lx.isString(content[prop]) || lx.isNumber(content[prop]))
                propVal = content[prop]
            else if (lx.implementsInterface(content[prop], {methods:['toCssString']}))
                propVal = content[prop].toCssString();
            else if (content[prop] instanceof lx.PresetFieldtHolder)
                propVal = content[prop].getValue(context.preset);

            if (propVal === null) continue;

            arr.push(propName + ':' + propVal);
        }
        return arr.join(';');
    };

    return '';
}
