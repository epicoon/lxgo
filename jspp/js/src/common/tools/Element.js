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
     * @param {String} method - called method name
     * @param {Array} params - parameters to call method
     */
    static ajax(method, params = []) {
        return new lx.ElementRequest(this.getKey(), method, params);
    }

    /**
     * @abstract
     * @param {lx.CssContext} css
     */
    static initCss(css) {
        // pass
    }

    newEvent(params = {}) {
        return new lx.ElementEvent(this, params);
    }
    // @lx:context>
}
