// @lx:namespace lx;
class Language extends lx.AppComponentSettable {
    /**
     * @returns {string}
     */
    defaultSettingKey() {
        return 'options';
    }

    /**
     * @returns {string}
     */
    current() {
        return lx.app.cookie.get('lxlang') || this.settings.default || 'en-EN';
    }

    /**
     * @returns {map[string]string}
     */
    options() {
        if (this.settings.options)
            return this.settings.options;
        return {'en-EN': 'English'};
    }

    /**
     * @param {string} val 
     */
    set(val) {
        if (this.current() == val) return;

        if (val != 'en-EN') {
            if (!this.settings.options || !(val in this.settings.options)) {
                lx.logError('Unknown language ' + val);
                return;
            }
        }

        // @lx:<context CLIENT:
        lx.app.cookie.set('lxlang', val);
        location.reload();
        // @lx:context>
    }
}
