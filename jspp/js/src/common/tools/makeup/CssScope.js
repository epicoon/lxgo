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
        if (this.elems.includes(elemClass.getKey()))
            return;

        let fInitCss = null;
        const preset = this.context.preset,
            injections = preset.injectElementsCss(),
            className = elemClass.lxFullName();
        if (className in injections) fInitCss = injections[className];
        else if (elemClass.initCss && !lx.app.functionHelper.isEmptyFunction(elemClass.initCss))
            fInitCss = elemClass.initCss;
        if (fInitCss === null) return;

        const context = new lx.CssContext();
        if (this.name !== _defaultName)
            context.setPrefix(this.name);
        context.usePreset(preset);
        fInitCss.call(elemClass, context);
        let css = context.toString();

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
        this.elems.push(elemClass.getKey());
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
