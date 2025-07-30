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
     * @param {String} key - called method key  ключ вызываемого метода
     * @param {Array} params - paramrters to call method
     */
    static ajax(key, params = []) {
        return new lx.ElementRequest(this.getKey(), key, params);
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
