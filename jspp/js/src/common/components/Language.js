// @lx:namespace lx;
class Language extends lx.AppComponentSettable {
    /**
     * @returns {string}
     */
    defaultSettingKey() {
        return 'list';
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
        if (!this.settings.list)
            return {'en-EN': 'English'};
        let res = {};
        for (let i in this.settings.list)
            res[i] = this.settings.list[i].name;
        return res;
    }

    /**
     * @returns {map[string]string}
     */
    flags() {
        if (!this.settings.list)
            return {};
        let res = {};
        for (let i in this.settings.list)
            res[i] = this.settings.list[i].flag;
        return res;
    }

    /**
     * @param {string} val 
     */
    set(val) {
        if (this.current() == val) return;

        if (val != 'en-EN') {
            if (!this.settings.list || !(val in this.settings.list)) {
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
