import AuthManager from './AuthManager';

class App {
    constructor() {
        this.eventHandlers = {};
        this.authManager = new AuthManager(this);
    }

    subscribe(event, handler) {
        if (!(event in this.eventHandlers))
            this.eventHandlers[event] = [];
        this.eventHandlers[event].push(handler)
    }

    trigger(event) {
        if (!(event in this.eventHandlers)) return;
        this.eventHandlers[event].forEach(handler => handler());
    }

    async fetch(url, params = {}) {
        const success = await _prepareParams(this, params);
        if (!success) {
            console.error('Can not send request');
            return null;
        }
        return await fetch(url, params);
    }

    run() {
        this.authManager.checkTokens();
    }
}

const app = new App();
window.lxAuth = {
    TOKENS_FOUND: 'TokensFound',
    TOKENS_NOT_FOUND: 'TokensNotFound',
    TOKENS_REMOVED: 'TokensRemoved',
    app: app,
    run: function() { this.app.run() },
    goToAuth: function() { this.app.authManager.goToAuth() },
    logOut: function() { this.app.authManager.logOut() },
    getUserData: async function() { return await this.app.authManager.getUserData() },
    on: function(event, handler) { this.app.subscribe(event, handler) },
    fetch: async function (url, params = {}) { return await this.app.fetch(url, params) }
};

/**
 * @param {App} app 
 * @param {Object} params 
 * @returns {bool}
 */
async function _prepareParams(app, params) {
    let accessToken = app.authManager.getAccessToken();
    if (!accessToken.isActive()) {
        const refreshToken = app.authManager.getRefreshToken();
        if (!refreshToken.isActive()) {
            return false;
        }

        const result = await app.authManager.refreshTokens();
        if (!result) {
            return false;
        }

        accessToken = app.authManager.getAccessToken();
    }

    if (!('headers' in params)) params.headers = {};
    params.headers['Authorization'] = 'Bearer ' + accessToken.value;
    return true;
}
