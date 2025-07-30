// @lx:module lx.Html;

// @lx:namespace lx;
class Html extends lx.Rect {
    static getStaticTag() {
        return 'slug';
    }

    modifyConfigBeforeApply(config) {
        if (lx.isString(config))
            config = {html: config};
        return config;
    }
}
