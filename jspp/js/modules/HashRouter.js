// @lx:module lx.HashRouter;

// @lx:namespace lx;
class HashRouter {
    constructor() {
        this.routes = [];
        this._onChange = [];
        this._onRoute = {};        
    }

    /**
     * @param conf {Object: {
     *     routes {Array<String>}
     * }}
     */
    init(conf) {
        this.routes = conf.routes
    }

    start() {
        _define(this);
    }

    /**
     * @param {Function} callback 
     */
    onChange(callback) {
        this._onChange.push(callback);
    }

    /**
     * @param {string} route 
     * @param {Function} callback 
     */
    onRoute(route, callback) {
        if (!(route in this._onRoute))
            this._onRoute[route] = [];
        this._onRoute[route].push(callback);
    }

    /**
     * @param {string} route
     */
    change(route) {
        let current = location.hash;
        if (current === route) return;

        if (!this.routes.includes(route)) {
            console.error('Route is unknown!');
            return;
        }

        let hash = route;
        if (hash !== '') {
            hash = hash.replace(/^#/, '');
            hash = '#' + hash;
        }

        history.replaceState(null, '', location.pathname + hash);
        _define(this);
    }
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * PRIVATE
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

/**
 * @private
 * @param {Router} self 
 */
function _define(self) {
    let hash = location.hash;

    if (self.routes.includes(hash)) {
        _trigger(self, hash);
        return;
    }

    if (hash == '') {
        console.error('Hash is empty!');
        return;
    }

    hash = hash.replace(/^#/, '');
    if (self.routes.includes(hash)) {
        _trigger(self, hash);
        return;
    }

    console.error('Hash is unknown!');
}

function _trigger(self, route) {
    self._onChange.forEach(f=>f(route));
    if (route in self._onRoute)
        self._onRoute[route].forEach(f=>f());
}
