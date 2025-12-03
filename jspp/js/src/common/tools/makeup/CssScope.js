const _defaultName = 'lx-common';

// @lx:namespace lx;
class CssScope {
    /**
     * @param {string} name
     * @param {string|Function<lx.CssPreset>|lx.CssPreset|null} preset
     */
    constructor(name = '', preset = null) {
        this.name = (name === '') ? _defaultName : name;
        this.elems = [];
        this.classes = [];
        this.context = new lx.CssContext();
        // @lx:<context SERVER:
        this.css = '';
        // @lx:context>
        preset = (preset instanceof lx.CssPreset)
            ? preset
            : lx.app.cssManager.getPreset(preset);
        this.context.usePreset(preset);
        if (this.name !== _defaultName)
            this.context.setPrefix(this.name);
    }

    // @lx:<context CLIENT:
    /**
     * @param data {Object: {
     *     elems {Array<String>},
     *     classes {Array<String>},
     *     css {String}
     * }}
     */
    unpack(data) {
        this.elems = data.elems;
        this.classes = data.classes;
        const cssTag = new lx.CssTag({id: this.name});
        cssTag.setCss(data.css);
        cssTag.commit();
    }
    // @lx:context>

    /**
     * @returns {String}
     */
    getPrefix() {
        return (this.name === _defaultName)
            ? ''
            : this.name;
    }

    /**
     * @param {Function<lx.Element.constructor>} elemClass
     */
    addElement(elemClass) {
        if (!elemClass || !lx.isFunction(elemClass)) return;

        const className = elemClass.lxFullName();
        if (this.elems.includes(className))
            return;

        const context = _initCss(this, elemClass);
        if (context === null) return;
        const css = context.toString();
        if (css === '') return;

        // @lx:<context CLIENT:
        const cssTag = new lx.CssTag({id: this.name});
        cssTag.addCss(css);
        cssTag.commit();
        // @lx:context>
        // @lx:<context SERVER:
        this.css += css;
        // @lx:context>

        let nn = context.getClassNames();
        for (let i in nn)
            this.classes.lxPushUnique(nn[i]);
        this.context.merge(context);
        this.elems.push(className);
    }

    update() {
        const preset = this.context.preset;
        this.classes = [];
        this.context = new lx.CssContext();
        // @lx:<context CLIENT:
        const cssTag = new lx.CssTag({id: this.name});
        cssTag.setCss('');
        // @lx:context>
        // @lx:<context SERVER:
        this.css = '';
        // @lx:context>
        this.context.usePreset(preset);
        if (this.name !== _defaultName)
            this.context.setPrefix(this.name);

        for (let i in this.elems) {
            const className = this.elems[i],
                elemClass = lx.getClassConstructor(className);
            if (!elemClass) continue;

            const context = _initCss(this, elemClass);
            if (context === null) continue;
            const css = context.toString();
            if (css === '') continue;

            // @lx:<context CLIENT:
            cssTag.addCss(css);
            // @lx:context>
            // @lx:<context SERVER:
            this.css += css;
            // @lx:context>

            let nn = context.getClassNames();
            for (let i in nn)
                this.classes.lxPushUnique(nn[i]);
            this.context.merge(context);
        }

        // @lx:<context CLIENT:
        cssTag.commit();
        // @lx:context>
    }

    /**
     * @param name {String}
     * @returns {Boolean}
     */
    hasClass(name) {
        if (this.classes.includes(name))
            return true;
        let key = '.' + name;
        return this.classes.includes(key);
    }
}

/**
 * @private
 * @param {CssScope} self
 * @param {Function<lx.Element.constructor>} elemClass
 * @param {String} className
 * @returns {lx.CssContext}
 */
function _initCss(self, elemClass) {
    const preset = self.context.preset,
        className = elemClass.lxFullName();

    let fInitCss = null;
    const injections = preset.injectElementsCss();
    if (className in injections) fInitCss = injections[className];
    else if (elemClass.initCss && !lx.app.functionHelper.isEmptyFunction(elemClass.initCss))
        fInitCss = elemClass.initCss;
    if (fInitCss === null) return null;

    const context = new lx.CssContext();
    if (self.name !== _defaultName)
        context.setPrefix(self.name);
    context.usePreset(preset);

    fInitCss.call(elemClass, context);
    return context;
}
