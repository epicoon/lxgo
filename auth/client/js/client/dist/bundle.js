class AuthManager {
    constructor(app) {
        this.app = app;
        this.settings = {};
        this.accessToken = null;
        this.refreshToken = null;
        this.active = _loadSettings(this);
        this.redirecting = false;
    }

    
    getAccessToken() {
        return _getAccessToken(this);
    }

    
    getRefreshToken() {
        return _getRefreshToken(this);
    }

    
    checkTokens() {
        if (!this.active) return;

        const accessToken = _getAccessToken(this);
        if (!accessToken.isActive()) {
            const refreshToken = _getRefreshToken(this);
            if (!refreshToken.isActive()) {
                this.app.trigger('TokensNotFound');
                return;    
            }
        }

        this.app.trigger('TokensFound');
    }

    
    async refreshTokens() {
        const refreshToken = _getRefreshToken(this);
        if (!refreshToken.isActive()) {
            console.error("Refresh auth token invalid");
            return false;
        }

        const response = await fetch(this.settings.refresh_path, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                refresh_token: refreshToken.value
            })
        });

        if (!response.ok) {
            console.error("Refresh auth tokens failed");
            return false;
        }

        const data = await response.json();
        const accessToken = _getAccessToken(this);
        accessToken.init(data.access_token, data.access_token_expired);
        accessToken.toStorage('lxAuthAccessToken');
        refreshToken.init(data.refresh_token, data.refresh_token_expired);
        refreshToken.toStorage('lxAuthRefreshToken');

        return true;
    }

    async goToAuth() {
        if (this.redirecting) return;
        this.redirecting = true;

        const state = await _genState(this);
        if (state === null) {
            console.error("Can not redirect");
            return;
        }

        const authData = this.settings;
        _postRedirect(`${authData.server}/auth`, {
            response_type: 'code',
            client_id: authData.id,
            redirect_uri: authData.redirect_uri,
            state
        });
    }

    
    async logOut() {
        const refreshToken = _getRefreshToken(this);
        if (!refreshToken.isActive()) {
            _dropTokens(this);
            this.app.trigger('TokensRemoved');
            return;
        }

        const accessToken = _getAccessToken(this);
        const response = await fetch(this.settings.logout_path, {
            method: 'GET',
            headers: { 'Authorization': 'Bearer ' + accessToken.value }
        });

        if (response.ok) {
            _dropTokens(this);
            this.app.trigger('TokensRemoved');
        }
    }

    
    async getUserData() {
        const response = await this.app.fetch(this.settings.user_data_path);
        if (!response || !response.ok) {
            console.error('Fetching user data failed');
            return {success: false};
        }
        const data = await response.json();
        data.success = true;
        return data;
    }
}if(AuthManager.__afterDefinition)AuthManager.__afterDefinition();




function _loadSettings(self) {
    const as = window._lxauth_settings;
    if (!as || as === '') {
        console.error('Auth settings are not available');
        return false;
    }

    delete window._lxauth_settings;

    let sett;
    try {
        sett = JSON.parse(as);
    } catch (e) {
        console.error('Can not parse auth settings');
        return false;
    }

    self.settings = sett;
    return true;
}


async function _genState(self) {
    const response = await fetch(self.settings.state_path, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ uri: window.location.href })
    });
    if (!response || !response.ok) {
        console.error('Fetching state failed');
        return null;
    }
    const data = await response.json();
    return data.state;
}


function _getAccessToken(self) {
    if (self.accessToken === null) {
        _readToken(self, 'accessToken', 'lxAuthAccessToken');
    }
    return self.accessToken;
}


function _getRefreshToken(self) {
    if (self.refreshToken === null) {
        _readToken(self, 'refreshToken', 'lxAuthRefreshToken');
    }
    return self.refreshToken;
}


function _readToken(self, selfKey, lsKey) {
    if (!self.active) return false;

    self[selfKey] = new Token();

    let tokenData = localStorage.getItem(lsKey);
    if (!tokenData) {
        return;
    }

    try {
        tokenData = JSON.parse(tokenData)
    } catch (e) {
        console.error(e);
        return;
    }

    self[selfKey].init(tokenData[0], tokenData[1]);
}


function _dropTokens(self) {
    localStorage.removeItem('lxAuthAccessToken');
    localStorage.removeItem('lxAuthRefreshToken');
    self.accessToken = null;
    self.refreshToken = null;
}


function _postRedirect(url, params) {
    const form = document.createElement("form");
    form.method = "POST";
    form.action = url;

    for (const key in params) {
        if (params.hasOwnProperty(key)) {
            const input = document.createElement("input");
            input.type = "hidden";
            input.name = key;
            input.value = params[key];
            form.appendChild(input);
        }
    }

    document.body.appendChild(form);
    form.submit();
}

class Token {
    constructor() {
        this.exists = false;
        this.value = null;
        this.expiresAt = null;
    }

    
    init(value, expiresAt) {
        this.value = value;
        this.expiresAt = +expiresAt;
        this.exists = true;
    }

    
    toStorage(key) {
		localStorage.setItem(key, '["'+this.value+'", '+this.expiresAt+']');
    }

    
    isExpired() {
        const currentTime = Math.floor(Date.now() / 1000);
        return this.expiresAt <= currentTime;
    }

    
    isActive() {
        return this.exists && !this.isExpired();
    }
}if(Token.__afterDefinition)Token.__afterDefinition();
;

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
}if(App.__afterDefinition)App.__afterDefinition();

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
