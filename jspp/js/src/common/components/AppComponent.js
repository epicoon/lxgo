// @lx:namespace lx;
class AppComponent {
    constructor(app) {
        this.app = app;
        this.init();
    }

    /** @abstract */
    init() {
        // pass
    }

    /** @abstract */
    onReady() {
        // pass
    }
}
