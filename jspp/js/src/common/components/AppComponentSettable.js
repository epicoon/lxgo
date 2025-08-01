// @lx:namespace lx;
class AppComponentSettable extends lx.AppComponent {
    /**
     * @param {lx.Application} app
     */
    constructor(app) {
        super(app);
        this.settings = {};
    }

    /**
     * @param {Object} list
     */
    addSettings(list) {
        let key = this.defaultSettingKey();
        if (!(key in list)) {
            let temp = {};
            temp[key] = list;
            list = temp;
        }
        for (let key in list)
            this.settings[key] = list[key];
        this.processSettings();
    }

    /**
     * @abstract
     * @returns {string}
     */
    defaultSettingKey() {
        return '';
    }

    /**
     * @abstract
     */
    processSettings() {
        // pass
    }
}
