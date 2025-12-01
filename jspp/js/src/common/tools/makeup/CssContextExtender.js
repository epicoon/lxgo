// @lx:namespace lx;
class CssContextExtender {
    static context;

    /**
     * @abstract 
     * @param {lx.CssContext} css
     */
    static init(css) {
        // pass
    }

    /**
     * @returns lx.CssContext
     */
    static getContext() {
        if (!this.context) {
            const css = new lx.CssContext;
            this.init(css);
            this.context = css;
        }
        return this.context;
    }

    /**
     * @param {string|Array<string>} valName
     * @param {any|Array<any>} defaultVal
     * @param {Function} [modifier]
     * @returns {lx.PresetFieldtHolder}
     */
    static presetValue(valName, defaultVal, modifier = null) {
        return new lx.PresetFieldtHolder(valName, defaultVal, modifier);
    }
}
