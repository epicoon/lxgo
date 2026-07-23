// @lx:namespace lx;
class Element {
    /**
     * @returns {String}
     */
    static getKey() {
        return this.lxFullName();
    }

    // @lx:<context CLIENT:
    /**
     * @abstract
     * @param {lx.CssContext} css
     */
    static initCss(css) {
        // pass
    }

    /**
     * @param {String} path - called path
     * @param {Object} params - parameters to call procedure
     */
    ajax(path, params = {}) {
        return new lx.ElementRequest(this.constructor.getKey(), path, params);
    }

    newEvent(params = {}) {
        return new lx.ElementEvent(this, params);
    }
    // @lx:context>
}
